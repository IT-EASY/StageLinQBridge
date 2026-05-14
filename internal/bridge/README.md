# internal/bridge

> **[Deutsche Version](README.de.md)**

SSE event hub and deck state tracker. The two types in this package form the real-time core of StageLinQBridge.

## Hub

`Hub` is a fan-out broadcaster for Server-Sent Events. Any number of browser clients can subscribe; the hub delivers typed JSON events to all of them concurrently.

```go
hub := bridge.NewHub()
hub.Broadcast("beat", BeatEvent{Deck: 0})
```

The hub is registered directly as an `http.Handler` on the `/events` route.

## Tracker

`Tracker` maintains the state of all four decks and decides when to fire output events.

### Beat event flow

1. `beatinfo.BeatEvent` arrives (≈33 times/s per deck)
2. The integer part of `Beat` is compared to the last known value
3. On crossing: `beatCount` is incremented; SSE + `OutputFn` are called for active events

### Output events

| Event | Condition |
|-------|-----------|
| `beat` | Every beat crossing |
| `downbeat` | `beatCount % timeSig == 0` |
| `slow` | `beatCount % retardDiv == 0` |

### Routing

`effectiveRoute(deck)` combines two layers:

| Layer | Source | Overrides |
|-------|--------|-----------|
| Manual kill | `routeActive[deck]` (set via `/route`) | Always wins |
| Auto-routing | Fader + crossfader from `/Mixer/…` StateMap paths | Only when `hasFaderData == true` |

`hasFaderData` is only set to `true` once a channel fader value above 0.05 is received — this prevents false-muting from the all-zero startup burst some devices send.

## Key types

```go
type DeckState struct {
    Playing   bool
    BPM       float64
    Artist    string
    Title     string
    Beat      float64  // fractional phase within beat (0.0–1.0)
    BeatInBar int      // 0-based position within bar
    Route     bool     // effective routing (auto + manual combined)
    Fader     float64
    IsMaster  bool
}

type StateEvent struct {
    Decks      [4]DeckState
    Crossfader float64
}

type BeatEvent struct{ Deck int }

type OutputFunc func(event string, deck int)
```
