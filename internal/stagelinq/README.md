# internal/stagelinq

> **[Deutsche Version](README.de.md)**

StageLinQ protocol implementation — discovery, main connection, StateMap, and BeatInfo.

## Protocol Overview

StageLinQ is a proprietary discovery and data protocol used by Denon DJ / inMusic hardware running Engine OS. It operates over both UDP (discovery) and TCP (data services).

```
UDP :26999 multicast   ←→  Discovery announcements
TCP <dynamic port>     ←→  Main connection (service negotiation)
TCP <dynamic port>     ←→  StateMap  (key-value state subscriptions)
TCP <dynamic port>     ←→  BeatInfo  (high-frequency beat stream)
```

## Sub-packages

### `announce/`

Broadcasts a StageLinQ presence announcement on the local network so that Denon devices can discover and connect to this bridge.

- Sends UDP multicast to `224.0.0.1:26999`
- Re-announces periodically
- Reports its own token, software name/version, and main connection port

### `discovery/`

Listens for announcements from other StageLinQ devices on the network.

- Useful for diagnostics (seeing which devices are present)
- The bridge uses this passively — devices connect inbound to `mainconn`

### `mainconn/`

TCP server that accepts incoming connections from Denon devices. Handles service capability exchange and directs the device to connect to the StateMap and BeatInfo servers.

### `statemap/`

TCP server that handles StateMap protocol connections. Accepts a list of subscription paths and delivers value updates as `StateUpdate` events.

Key paths used:

```
/Engine/Deck{1-4}/Play
/Engine/Deck{1-4}/CurrentBPM
/Engine/Deck{1-4}/Track/ArtistName
/Engine/Deck{1-4}/Track/SongName
/Engine/Deck{1-4}/DeckIsMaster
/Engine/Deck{1-4}/ExternalMixerVolume
/Mixer/CH{1-4}faderPosition
/Mixer/CrossfaderPosition
/Mixer/ChannelAssignment{1-4}
/Engine/Master/MasterTempo
```

Values arrive as JSON strings: `{"state":true}`, `{"value":128.5}`, `{"string":"Artist"}`.

### `beatinfo/`

TCP server that decodes the high-frequency BeatInfo stream. Delivers `BeatEvent` structs at ≈33 Hz containing per-player beat position as a floating-point number. The integer part is the beat number; the fractional part is the phase within the current beat.

### `encoding/`

Low-level wire protocol helpers: network string encoding (length-prefixed UTF-16), binary reader/writer wrappers. Used internally by all other StageLinQ sub-packages.

### `token/`

16-byte token used to identify this client to the Denon device. Generated randomly on first run and persisted to `config.json`.
