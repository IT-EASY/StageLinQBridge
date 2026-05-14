# web

> **[Deutsche Version](README.de.md)**

Embedded Web UI for StageLinQBridge, served at `http://localhost:8080`.

## Embedding

`web.go` uses `//go:embed index.html` to bake the UI directly into the binary. No separate web server or file copy step is needed.

```go
import webui "github.com/it-easy/StageLinQBridge/web"
mux.Handle("/", http.FileServer(http.FS(webui.Files)))
```

## Features

- **4-deck strip** — artist, title, BPM, beat-phase block display
- **LIVE button** per deck — three states:
  - Green: active (fader open + manual on)
  - Dimmed: auto-muted (fader closed by hardware)
  - Red: force-muted (manually disabled via button click)
- **Crossfader** — A↔B position indicator with ±3 % visual center snap
- **Time signature** — 3/4 · 4/4 selector
- **Slow/Retard divisor** — ÷16 · ÷32 · ÷64 · ÷128
- **Protocol toggles** — sACN · Art-Net · OSC, synced from `/state` on load

## SSE events consumed

| Event | Payload | Description |
|-------|---------|-------------|
| `state` | `StateEvent` JSON | Deck state, BPM, fader, routing, crossfader |
| `beat` | `{"deck":0}` | Quarter-note beat on deck N |
| `downbeat` | `{"deck":0}` | Bar beat 1 on deck N |
| `slow` | `{"deck":0}` | Slow/Retard pulse on deck N |

## Window sizing

On load, `window.resizeTo(1200, 230)` is called. The application is launched in Edge `--app` mode (`--window-size=1200,230 --window-position=10,10`) for a compact, tab-free standalone window.
