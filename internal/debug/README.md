# internal/debug

> **[Deutsche Version](README.de.md)**

Minimal leveled console logger used throughout StageLinQBridge.

## Levels

```
Error  →  only critical failures
Warn   →  non-fatal issues, degraded behaviour
Info   →  startup messages, connection events
Debug  →  detailed operation flow
Trace  →  high-frequency events (beat ticks, packet details)
```

In normal operation the application is initialised at `Error` level — the console stays silent. Pass `-debug` on the command line to switch to `Trace`.

## Usage

```go
logger := debug.New(debug.Error)   // production
logger := debug.New(debug.Trace)   // debug mode

logger.Info("server listening", "port", 8080)
logger.Warn("config not found, using defaults")
logger.Error("fatal", "error", err)
```

Key-value pairs after the message are formatted as `key=value` on the same line.

## Output format

```
2026-05-14 21:04:01 [INFO] server listening port=8080
```
