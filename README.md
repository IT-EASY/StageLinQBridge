# StageLinQBridge

> **[Deutsche Version](README.de.md)**

A Go application that bridges **Denon PRIME DJ hardware** (PRIME 4, PRIME 2, SC6000 and compatible) to lighting and automation systems via **sACN/E1.31** and **OSC**.

StageLinQBridge listens to live beat events from the StageLinQ protocol, evaluates channel fader and crossfader positions in real time, and fires DMX or OSC pulses — beat-synchronized, per deck, with automatic routing.

---

## Features

| Feature | Description |
|---------|-------------|
| **Beat** | Fires on every quarter note of the active deck |
| **Downbeat** | Fires on beat 1 of each bar (3/4 or 4/4, configurable) |
| **Slow / Retard** | Fires every N beats (default ÷64 ≈ 32 s at 120 BPM), divisor configurable |
| **Fader routing** | Only the deck whose channel fader is open produces output |
| **Crossfader** | A/B/Through channel assignment is evaluated automatically |
| **Manual mute** | Per-deck LIVE button in the Web UI overrides auto-routing |
| **sACN / E1.31** | UDP multicast — no target IP configuration needed |
| **OSC** | UDP unicast, float32 `1.0` trigger / `0.0` release |
| **Runtime toggle** | Enable/disable each output protocol from the UI without restart |
| **Web UI** | Auto-opens as a standalone Edge app window on startup |
| **Zero installer** | Single `.exe` + `config.json` in the same folder — that's it |

---

## Quick Start

### Build

```sh
go build -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```

### Run

```sh
# Normal operation — silent (errors only to console)
StageLinQBridge.exe

# Verbose logging for diagnostics
StageLinQBridge.exe -debug
```

On startup the application:

1. Reads `config.json` from the same directory as the executable (creates a default if absent)
2. Announces itself on the local network via StageLinQ UDP multicast
3. Waits for a Denon PRIME device to connect and subscribe to beat + state data
4. Opens `http://localhost:8080` automatically in Edge app mode

---

## Configuration

`config.json` is created automatically on first run with safe defaults (all outputs disabled). Place it in the same directory as the executable.

```jsonc
{
  "network": {
    "lan_ip": "192.168.1.x",    // LAN interface IP; empty = use all interfaces
    "token": ""                   // auto-generated on first run, do not edit
  },
  "sacn": {
    "enabled": false,
    "universe": 1,                // DMX universe 1–63999
    "channels": {
      "beat":     1,              // DMX channel number for beat pulse (1-based)
      "downbeat": 2,              // DMX channel number for bar beat 1
      "slow":    11               // DMX channel number for slow/retard pulse
    },
    "pulse_ms": 50                // how long the channel stays at 255 (ms)
  },
  "osc": {
    "enabled": false,
    "target": "192.168.1.100:9000",
    "addresses": {
      "beat":     "/stagelinq/beat",
      "downbeat": "/stagelinq/downbeat",
      "slow":     "/stagelinq/slow"
    },
    "pulse_ms": 50
  }
}
```

### Config file location

| Scenario | Path |
|----------|------|
| Production | `<exe directory>/config.json` |
| Development (repo) | `configs/config.json` (fallback) |

If neither exists a default `config.json` is written next to the executable.

---

## Output Protocols

### sACN / E1.31

UDP multicast to `239.255.{universe_hi}.{universe_lo}:5568`. No target IP needed — the multicast address is derived from the universe number. Full 638-byte E1.31 packet with 512-channel DMX frame, source name `StageLinQBridge`.

### OSC

UDP unicast to `"ip:port"`. Sends float32 `1.0` on the configured address when an event fires, followed by float32 `0.0` after `pulse_ms` milliseconds.

---

## Web UI

Available at `http://localhost:8080`. Opens automatically in Edge app mode on startup.

| Element | Description |
|---------|-------------|
| 4-deck strip | Artist, title, BPM, beat-phase blocks |
| **LIVE** button | Green = active · Dimmed = auto-muted (fader closed) · Red = force-muted |
| Crossfader | A↔B position display |
| **3/4 · 4/4** | Time signature selector |
| **÷16 · ÷32 · ÷64 · ÷128** | Slow/Retard divisor |
| **sACN · OSC** | Runtime protocol enable/disable |

---

## Beat Channels Explained

| Channel | When | Typical use |
|---------|------|-------------|
| **beat** | Every quarter note | Running cue, chaser step |
| **downbeat** | Beat 1 of each bar | Bar-aligned effect trigger |
| **slow** | Every ÷N beats | Atmospheric effect, fog machine |

*"Slow/Retard"* — "Retard" is the established term from DJ/lighting practice (e.g. Avolites Titan). The channel is called `slow` in code and config to avoid ambiguity; both terms appear in the UI.

---

## Repository Structure

```
cmd/
  stagelinqbridge-discover/   Main entry point, HTTP server, Web UI host
internal/
  bridge/                     SSE hub + deck state tracker
  config/                     JSON configuration (load, save, defaults)
  debug/                      Leveled console logger
  network/                    Interface validation, broadcast helpers
  output/                     Output manager
    osc/                      OSC UDP sender
    sacn/                     sACN E1.31 sender (multicast)
  stagelinq/
    announce/                 StageLinQ UDP announcer
    beatinfo/                 BeatInfo TCP server + packet decoder
    discovery/                Device discovery listener
    encoding/                 Wire protocol encoding/decoding
    mainconn/                 Main connection TCP server
    statemap/                 StateMap subscription server
    token/                    Token generation and hex parsing
web/                          Embedded web UI (go:embed)
configs/                      Development config.json
```

---

## Development Environment

- **Go** 1.22+
- **OS** Windows 10/11 (primary target; Linux builds work but Edge app-mode window won't open)
- **Hardware** Denon PRIME 4 running Engine OS 4.x on the same LAN segment

---

## Credits

Protocol research and reference implementation by Carl Kittelberger:
[github.com/icedream/go-stagelinq](https://github.com/icedream/go-stagelinq)

---

## Disclaimer

StageLinQ is a proprietary protocol by Denon DJ / inMusic. This project is an independent interoperability effort and is not affiliated with or endorsed by Denon DJ or inMusic.
