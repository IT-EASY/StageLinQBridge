# cmd/stagelinqbridge-discover

> **[Deutsche Version](README.de.md)**

Main entry point of the StageLinQBridge application.

## Responsibilities

- Parses CLI flags (`-debug`)
- Resolves and loads `config.json`
- Initialises all subsystems in order: config → token → bridge → output → StageLinQ stack → HTTP server
- Hosts the Web UI at `http://localhost:8080`
- Launches the Edge app-mode window automatically
- Handles graceful shutdown on `SIGINT` / `SIGTERM`

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-debug` | `false` | Enable verbose (Trace-level) console logging. Without this flag only errors are printed. |

## Config file resolution

The executable looks for `config.json` in this order:

1. `<exe directory>/config.json` — production layout
2. `configs/config.json` — development fallback (repo working directory)
3. If neither exists, a default `config.json` is written to the exe directory

## HTTP endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Embedded Web UI (static files) |
| `GET` | `/events` | SSE stream — `state`, `beat`, `downbeat`, `slow` events |
| `GET` | `/state` | Current routing, retardDiv, timeSig, protocol enable states (JSON) |
| `POST` | `/config` | Set `retardDiv` and/or `timeSig` at runtime |
| `POST` | `/route` | Enable/disable beat output for a specific deck (`{"deck":0,"active":false}`) |
| `POST` | `/protocol` | Toggle protocol senders (`{"sacn":true,"artnet":false,"osc":true}`) |

## Build

```sh
go build -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```
