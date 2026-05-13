# StageLinQBridge

StageLinQBridge is a Go-based bridge and integration layer for Denon StageLinQ devices such as the PRIME 4.

The project is focused on reliable real-time extraction of:

- Beat information
- BPM and tempo data
- Deck/player state
- Track metadata
- StateMap updates

and forwarding this information to lighting and automation systems using protocols such as:

- sACN / E1.31
- OSC
- WebSocket / WebUI APIs

---

## Project Goals

The primary goal of StageLinQBridge is stable and low-latency synchronization between Denon DJ hardware and external lighting or automation systems.

Typical use cases include:

- Beat-synchronized lighting control
- Trigger generation for Avolites Titan cue lists
- BPM-driven effects
- External visualization and monitoring
- Embedded StageLinQ integrations using TinyGo / ESP32 (experimental)

---

## Current Status

This project is currently under active redesign and restructuring.

The implementation is based on research, protocol captures, and reference implementations from the StageLinQ community.

Current focus areas:

- Stable Discovery implementation
- Reliable TCP handshake handling
- StateMap subscriptions
- BeatInfo service support
- Configurable network handling
- Modular bridge architecture

---

## Credits / Acknowledgements

This project is heavily inspired by and partially based on the excellent work by Carl Kittelberger:

- https://github.com/icedream/go-stagelinq

The original repository helped validating protocol behavior against Denon PRIME devices running Engine OS 4.3.4.

Additional protocol observations and interoperability tests were performed using community implementations and network captures.

---

## Planned Features

- StageLinQ Discovery
- Automatic service negotiation
- BeatInfo decoding
- StateMap subscriptions
- sACN output engine
- OSC output
- JSON-based configuration
- Lightweight WebUI
- Multi-interface support
- Embedded/TinyGo experiments

---

## Design Principles

- No hardcoded IP addresses
- Modular architecture
- Minimal external dependencies
- Protocol-first implementation

Clean separation between:

- StageLinQ core
- bridge/output logic
- configuration
- UI

---

## Repository Structure (planned)

```text
/cmd
/internal/stagelinq
/internal/bridge
/internal/config
/internal/webui
/docs
/configs
```

---

## Development Environment

Recommended environment:

- Windows 11
- Go 1.26+
- VS Code
- GitHub Desktop

Hardware used during development:

- Denon PRIME 4
- Engine OS 4.3.4

---

## Disclaimer

StageLinQ is a proprietary protocol by Denon DJ / inMusic.

This project is an independent interoperability and integration effort and is not affiliated with or endorsed by Denon DJ or inMusic.
