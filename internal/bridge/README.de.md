# internal/bridge

> **[English version](README.md)**

SSE-Event-Hub und Deck-Statustracker. Die beiden Typen in diesem Paket bilden den Echtzeit-Kern von StageLinQBridge.

## Hub

`Hub` ist ein Fan-out-Broadcaster für Server-Sent Events. Beliebig viele Browser-Clients können sich anmelden; der Hub liefert typisierte JSON-Events gleichzeitig an alle.

```go
hub := bridge.NewHub()
hub.Broadcast("beat", BeatEvent{Deck: 0})
```

Der Hub wird direkt als `http.Handler` auf der Route `/events` registriert.

## Tracker

`Tracker` verwaltet den Zustand aller vier Decks und entscheidet, wann Output-Events ausgelöst werden.

### Beat-Event-Fluss

1. `beatinfo.BeatEvent` trifft ein (≈33-mal/s pro Deck)
2. Der ganzzahlige Anteil von `Beat` wird mit dem zuletzt gespeicherten Wert verglichen
3. Bei einem Übergang: `beatCount` wird erhöht; SSE + `OutputFn` werden für aktive Events aufgerufen

### Output-Events

| Event | Bedingung |
|-------|-----------|
| `beat` | Jeder Beat-Übergang |
| `downbeat` | `beatCount % timeSig == 0` |
| `slow` | `beatCount % retardDiv == 0` |

### Routing

`effectiveRoute(deck)` kombiniert zwei Ebenen:

| Ebene | Quelle | Priorität |
|-------|--------|-----------|
| Manuelles Muting | `routeActive[deck]` (gesetzt über `/route`) | Immer vorrangig |
| Auto-Routing | Fader + Crossfader aus `/Mixer/…` StateMap-Pfaden | Nur wenn `hasFaderData == true` |

`hasFaderData` wird erst auf `true` gesetzt, wenn ein Kanalfader-Wert über 0,05 empfangen wird — das verhindert ein irrtümliches Muting durch den Null-Burst mancher Geräte beim Start.

## Wichtige Typen

```go
type DeckState struct {
    Playing   bool
    BPM       float64
    Artist    string
    Title     string
    Beat      float64  // Bruchteil-Phase innerhalb des Beats (0.0–1.0)
    BeatInBar int      // 0-basierte Position innerhalb des Takts
    Route     bool     // Effektives Routing (Auto + Manuell kombiniert)
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
