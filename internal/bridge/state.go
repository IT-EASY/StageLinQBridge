package bridge

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/it-easy/StageLinQBridge/internal/stagelinq/beatinfo"
	"github.com/it-easy/StageLinQBridge/internal/stagelinq/statemap"
)

// DeckState holds the current known state of one deck.
type DeckState struct {
	Playing   bool    `json:"playing"`
	BPM       float64 `json:"bpm"`
	Artist    string  `json:"artist"`
	Title     string  `json:"title"`
	Beat      float64 `json:"beat"`      // fractional position within current beat, 0.0–1.0
	BeatInBar int     `json:"beatInBar"` // 0-based position within bar (0 = bar beat 1)
	Route     bool    `json:"route"`     // effective routing (auto + manual combined)
	Fader     float64 `json:"fader"`     // channel fader 0.0–1.0
	IsMaster  bool    `json:"isMaster"`  // true = this deck is the sync master
}

// StateEvent is broadcast on every meaningful change.
type StateEvent struct {
	Decks      [4]DeckState `json:"decks"`
	Crossfader float64      `json:"crossfader"` // /Mixer/CrossfaderPosition (0=A, 1=B)
}

// BeatEvent is broadcast when a deck crosses a beat boundary (beat / onset / retard).
type BeatEvent struct {
	Deck int `json:"deck"` // 0-based
}

// OutputFunc is called for every beat output event that passes routing.
// event is "beat", "downbeat", or "slow"; deck is 0-based.
type OutputFunc func(event string, deck int)

// Tracker maintains deck state and feeds the Hub.
type Tracker struct {
	hub      *Hub
	OutputFn OutputFunc // optional — wired up by main to drive sACN/Art-Net/OSC

	decks     [4]DeckState
	lastFloor [4]float64
	beatCount [4]int64

	// Mixer state (received from /Mixer/… paths).
	xfaderPos         float64   // CrossfaderPosition 0.0–1.0
	chanAssign        [4]string // ChannelAssignment{N}: "A", "B", "THROUGH", ""
	hasFaderData      bool      // true once we receive a fader value > 0 (confirms path works)
	faderUpdateCount  int       // counts received fader updates for diagnostics

	// Atomic fields — safe to read/write from any goroutine.
	retardDiv   int32    // beat divisor for retard channel
	timeSig     int32    // beats per bar: 3 or 4 (default 4)
	routeActive [4]int32 // manual kill: 1=allowed (default), 0=force-muted

	// Output mode — controls which beat types reach the DMX/OSC output.
	// mode:     0 = "3ch" (all three types simultaneously, default)
	//           1 = "1ch" (only the selected beatType)
	// beatType: 0 = "beat", 1 = "onset", 2 = "retard"
	mode     int32
	beatType int32
}

func NewTracker(hub *Hub) *Tracker {
	t := &Tracker{hub: hub}
	atomic.StoreInt32(&t.retardDiv, 64) // default ÷64 ≈ 32 s at 120 BPM
	atomic.StoreInt32(&t.timeSig, 4)   // default 4/4
	for i := range t.routeActive {
		atomic.StoreInt32(&t.routeActive[i], 1) // all decks allowed by default
		t.chanAssign[i] = "THROUGH"             // assume through until we know better
		t.decks[i].Route = true
	}
	return t
}

// SetRetardDiv sets the retard beat divisor (>= 1).
func (t *Tracker) SetRetardDiv(n int) {
	if n < 1 {
		n = 1
	}
	atomic.StoreInt32(&t.retardDiv, int32(n))
}

// SetTimeSig sets beats per bar (2, 3, or 4). Controls onset timing and block display.
func (t *Tracker) SetTimeSig(n int) {
	if n < 2 {
		n = 2
	}
	if n > 4 {
		n = 4
	}
	atomic.StoreInt32(&t.timeSig, int32(n))
	// Reset bar counters so the new time sig aligns immediately.
	for i := range t.beatCount {
		t.beatCount[i] = 0
	}
}

// TimeSig returns the current beats-per-bar setting.
func (t *Tracker) TimeSig() int {
	return int(atomic.LoadInt32(&t.timeSig))
}

// SetRoute manually enables/disables beat output for a deck (0-based).
// When disabled, the deck is silenced regardless of fader state.
func (t *Tracker) SetRoute(deck int, active bool) {
	if deck < 0 || deck >= 4 {
		return
	}
	v := int32(0)
	if active {
		v = 1
	}
	atomic.StoreInt32(&t.routeActive[deck], v)
	t.decks[deck].Route = t.effectiveRoute(deck)
	t.hub.Broadcast("state", StateEvent{Decks: t.decks, Crossfader: t.xfaderPos})
}

// RouteActive returns the current manual routing flags for all 4 decks.
func (t *Tracker) RouteActive() [4]bool {
	var r [4]bool
	for i := range r {
		r[i] = atomic.LoadInt32(&t.routeActive[i]) == 1
	}
	return r
}

// RetardDiv returns the current retard divisor.
func (t *Tracker) RetardDiv() int {
	return int(atomic.LoadInt32(&t.retardDiv))
}

// SetMode switches between "3ch" (all types) and "1ch" (selected type only).
func (t *Tracker) SetMode(m string) {
	v := int32(0)
	if m == "1ch" {
		v = 1
	}
	atomic.StoreInt32(&t.mode, v)
}

// Mode returns the current output mode ("3ch" or "1ch").
func (t *Tracker) Mode() string {
	if atomic.LoadInt32(&t.mode) == 1 {
		return "1ch"
	}
	return "3ch"
}

// SetBeatType sets which beat type is forwarded to output in 1ch mode.
// Valid values: "beat", "onset", "retard".
func (t *Tracker) SetBeatType(bt string) {
	var v int32
	switch bt {
	case "onset":
		v = 1
	case "retard":
		v = 2
	default:
		v = 0 // "beat"
	}
	atomic.StoreInt32(&t.beatType, v)
}

// BeatType returns the current beat type selection ("beat", "onset", or "retard").
func (t *Tracker) BeatType() string {
	switch atomic.LoadInt32(&t.beatType) {
	case 1:
		return "onset"
	case 2:
		return "retard"
	default:
		return "beat"
	}
}

// effectiveRoute returns whether a deck should currently emit beat events.
//
//   - Manual kill (routeActive == 0) always blocks, regardless of fader.
//   - If fader data is available (hasFaderData), auto-routing applies:
//     fader must be open AND crossfader must face this channel's side.
//   - If no fader data has arrived yet, manual-on means active.
func (t *Tracker) effectiveRoute(deck int) bool {
	if atomic.LoadInt32(&t.routeActive[deck]) == 0 {
		return false // manually killed
	}
	if !t.hasFaderData {
		return true // no fader data yet → trust manual setting
	}
	// Fader must be open (> 5 % threshold).
	if t.decks[deck].Fader < 0.05 {
		return false
	}
	// Crossfader must face this channel's assigned side.
	switch strings.ToUpper(t.chanAssign[deck]) {
	case "A":
		return t.xfaderPos < 0.95 // not fully on B side
	case "B":
		return t.xfaderPos > 0.05 // not fully on A side
	default: // "THROUGH" or unknown — crossfader irrelevant
		return true
	}
}

// recomputeAllRoutes recalculates Route for all decks and broadcasts state if anything changed.
func (t *Tracker) recomputeAllRoutes() {
	changed := false
	for i := range t.decks {
		r := t.effectiveRoute(i)
		if r != t.decks[i].Route {
			t.decks[i].Route = r
			changed = true
		}
	}
	if changed {
		t.hub.Broadcast("state", StateEvent{Decks: t.decks, Crossfader: t.xfaderPos})
	}
}

// OnStateUpdate processes a StateMap update.
func (t *Tracker) OnStateUpdate(u statemap.StateUpdate) {
	switch {
	case strings.HasPrefix(u.Name, "/Engine/Deck"):
		t.handleDeckState(u)
	case strings.HasPrefix(u.Name, "/Mixer/"):
		t.handleMixerState(u)
	}
}

func (t *Tracker) handleDeckState(u statemap.StateUpdate) {
	parts := strings.SplitN(u.Name, "/", 4)
	if len(parts) < 3 {
		return
	}
	deckStr := parts[2] // e.g. "Deck1"
	if !strings.HasPrefix(deckStr, "Deck") {
		return
	}
	idx, err := strconv.Atoi(deckStr[4:])
	if err != nil || idx < 1 || idx > 4 {
		return
	}
	i := idx - 1

	tail := strings.TrimPrefix(u.Name, "/Engine/"+deckStr+"/")

	switch tail {
	case "Play", "PlayState":
		t.decks[i].Playing = parseBool(u.Value)
	case "CurrentBPM":
		t.decks[i].BPM = parseFloat(u.Value)
	case "Track/ArtistName":
		t.decks[i].Artist = parseString(u.Value)
	case "Track/SongName":
		newTitle := parseString(u.Value)
		if newTitle != t.decks[i].Title && newTitle != "" {
			t.beatCount[i] = 0 // realign onset/retard counters on new track
		}
		t.decks[i].Title = newTitle
	case "DeckIsMaster":
		t.decks[i].IsMaster = parseBool(u.Value)
	case "ExternalMixerVolume":
		// Used when an external mixer is connected. Store as fader fallback;
		// /Mixer/CH{N}faderPosition takes precedence when available.
		if !t.hasFaderData {
			t.decks[i].Fader = parseFloat(u.Value)
			t.recomputeAllRoutes()
			return // recomputeAllRoutes already broadcasts
		}
	default:
		return // unhandled path — don't broadcast
	}

	t.decks[i].Route = t.effectiveRoute(i)
	t.hub.Broadcast("state", StateEvent{Decks: t.decks, Crossfader: t.xfaderPos})
}

func (t *Tracker) handleMixerState(u statemap.StateUpdate) {
	tail := strings.TrimPrefix(u.Name, "/Mixer/")

	switch {
	case tail == "CrossfaderPosition":
		t.xfaderPos = parseFloat(u.Value)
		t.recomputeAllRoutes()

	case strings.HasPrefix(tail, "CH") && strings.HasSuffix(tail, "faderPosition"):
		// /Mixer/CH1faderPosition … CH4faderPosition
		numStr := strings.TrimPrefix(tail, "CH")
		numStr = strings.TrimSuffix(numStr, "faderPosition")
		n, err := strconv.Atoi(numStr)
		if err != nil || n < 1 || n > 4 {
			return
		}
		v := parseFloat(u.Value)
		t.decks[n-1].Fader = v
		t.faderUpdateCount++
		// Only switch to auto-routing once we see at least one fader clearly open.
		// This prevents false-muting if the device sends all-zeros on startup.
		if !t.hasFaderData && v > 0.05 {
			t.hasFaderData = true
		}
		t.recomputeAllRoutes()

	case strings.HasPrefix(tail, "ChannelAssignment"):
		// /Mixer/ChannelAssignment1 … ChannelAssignment4 → "A", "B", "THROUGH"
		numStr := strings.TrimPrefix(tail, "ChannelAssignment")
		n, err := strconv.Atoi(numStr)
		if err != nil || n < 1 || n > 4 {
			return
		}
		assign := parseString(u.Value)
		if assign == "" {
			// Some devices send a numeric enum instead of a string.
			// 0 = THROUGH, 1 = A, 2 = B (assumption — adjust if needed).
			switch int(parseFloat(u.Value)) {
			case 1:
				assign = "A"
			case 2:
				assign = "B"
			default:
				assign = "THROUGH"
			}
		}
		t.chanAssign[n-1] = strings.ToUpper(assign)
		t.recomputeAllRoutes()
	}
}

// OnBeatEvent processes a BeatInfo event, detects beat / onset / retard boundaries.
// Only emits events for decks whose effective routing is active.
func (t *Tracker) OnBeatEvent(e beatinfo.BeatEvent) {
	changed := false
	div := int64(atomic.LoadInt32(&t.retardDiv))
	ts := int64(atomic.LoadInt32(&t.timeSig))

	for i, p := range e.Players {
		if i >= 4 {
			break
		}
		floor := math.Floor(p.Beat)
		frac := p.Beat - floor
		t.decks[i].Beat = frac

		if floor != t.lastFloor[i] && t.lastFloor[i] != 0 {
			t.beatCount[i]++
			cnt := t.beatCount[i]

			// Update bar position for block display (0-based, regardless of routing).
			t.decks[i].BeatInBar = int((cnt - 1) % ts)

			if t.effectiveRoute(i) {
				// In 1ch mode only one beat type reaches the DMX/OSC output.
				// SSE events always fire so the deck LEDs in the UI stay live.
				is1ch := atomic.LoadInt32(&t.mode) == 1
				bt    := atomic.LoadInt32(&t.beatType) // 0=beat, 1=onset, 2=retard

				t.hub.Broadcast("beat", BeatEvent{Deck: i})
				if t.OutputFn != nil && (!is1ch || bt == 0) {
					t.OutputFn("beat", i)
				}

				// Downbeat = beat 1 of a new bar.
				if cnt%ts == 0 {
					t.hub.Broadcast("downbeat", BeatEvent{Deck: i})
					if t.OutputFn != nil && (!is1ch || bt == 1) {
						t.OutputFn("downbeat", i)
					}
				}
				// Slow = every N beats (formerly "retard").
				if div > 0 && cnt%div == 0 {
					t.hub.Broadcast("slow", BeatEvent{Deck: i})
					if t.OutputFn != nil && (!is1ch || bt == 2) {
						t.OutputFn("slow", i)
					}
				}
			}
		}
		t.lastFloor[i] = floor
		changed = true
	}
	if changed {
		t.hub.Broadcast("state", StateEvent{Decks: t.decks, Crossfader: t.xfaderPos})
	}
}

// --- JSON value helpers ---

type rawVal struct {
	State  *bool    `json:"state"`
	Value  *float64 `json:"value"`
	String *string  `json:"string"`
}

func parseRaw(v string) rawVal {
	var r rawVal
	_ = json.Unmarshal([]byte(v), &r)
	return r
}

func parseBool(v string) bool {
	r := parseRaw(v)
	if r.State != nil {
		return *r.State
	}
	return false
}

func parseFloat(v string) float64 {
	r := parseRaw(v)
	if r.Value != nil {
		return *r.Value
	}
	return 0
}

func parseString(v string) string {
	r := parseRaw(v)
	if r.String != nil {
		return *r.String
	}
	return ""
}
