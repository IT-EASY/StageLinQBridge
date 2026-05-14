# web

> **[English version](README.md)**

Eingebettetes Web-UI für StageLinQBridge, bereitgestellt unter `http://localhost:8080`.

## Einbettung

`web.go` verwendet `//go:embed index.html`, um das UI direkt in das Binary zu integrieren. Kein separater Webserver oder Datei-Copy-Schritt erforderlich.

```go
import webui "github.com/it-easy/StageLinQBridge/web"
mux.Handle("/", http.FileServer(http.FS(webui.Files)))
```

## Funktionen

- **4-Deck-Leiste** — Interpret, Titel, BPM, Beat-Phase-Blöcke
- **LIVE-Button** pro Deck — drei Zustände:
  - Grün: aktiv (Fader offen + manuell aktiviert)
  - Gedimmt: auto-gemutet (Fader durch Hardware geschlossen)
  - Rot: manuell gemutet (durch Klick auf den Button deaktiviert)
- **Crossfader** — A↔B-Positionsanzeige mit ±3 % visuellem Mittenpunkt-Snap
- **Taktart** — Auswahl 3/4 · 4/4
- **Slow/Retard-Divisor** — ÷16 · ÷32 · ÷64 · ÷128
- **Protokoll-Toggles** — sACN · Art-Net · OSC, beim Laden aus `/state` synchronisiert

## Empfangene SSE-Events

| Event | Payload | Beschreibung |
|-------|---------|--------------|
| `state` | `StateEvent` JSON | Deck-Status, BPM, Fader, Routing, Crossfader |
| `beat` | `{"deck":0}` | Viertelschlag auf Deck N |
| `downbeat` | `{"deck":0}` | Taktschlag 1 auf Deck N |
| `slow` | `{"deck":0}` | Slow/Retard-Impuls auf Deck N |

## Fenstergröße

Beim Laden wird `window.resizeTo(1200, 230)` aufgerufen. Die Anwendung wird im Edge-`--app`-Modus gestartet (`--window-size=1200,230 --window-position=10,10`) für ein kompaktes, verschiebliches Standalone-Fenster ohne Tabs und Adressleiste.
