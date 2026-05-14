# internal/output

> **[Deutsche Version](README.de.md)**

Output manager and protocol senders. Dispatches beat events from the bridge to one or more lighting/automation protocols simultaneously.

## Architecture

```
Tracker.OutputFn  →  Manager.Dispatch(event, deck)
                         ├── sacn.Sender   (if enabled)
                         ├── artnet.Sender (if enabled)
                         └── osc.Sender    (if enabled)
```

Each sender runs its pulse in a separate goroutine (`go flash(ch)`), so a slow network write never blocks the beat stream.

## Manager

```go
m := output.New(cfg, logger)
defer m.Close()

m.Dispatch("beat", 0)        // fire beat event for deck 0
m.SetEnabled("sacn", true)   // toggle protocol at runtime
m.Enabled()                  // map[string]bool{"sacn":…,"artnet":…,"osc":…}
```

Senders that fail to initialise are logged as warnings and skipped — the application continues with the remaining senders.

---

## sACN / E1.31 (`sacn/`)

UDP multicast. The multicast group is derived from the universe number:

```
239.255.<universe_hi>.<universe_lo>:5568
```

- 638-byte E1.31 packet, full 512-channel DMX frame
- Source name: `StageLinQBridge`
- Random UUID CID generated at startup (RFC 4122 §4.4)
- Sequence number incremented on every packet

```go
s, err := sacn.New(universe, beatCh, downbeatCh, slowCh, pulseMS)
```

---

## Art-Net ArtDmx (`artnet/`)

UDP **unicast** to a configured target IP. **Broadcast is explicitly not supported.**

> Beat triggers fire continuously at ≈33 events/s. Sending those as broadcast would flood the entire network segment — unacceptable in a live production environment.

- 530-byte ArtDmx packet (OpCode 0x5000), full 512-channel DMX frame
- Default port: 6454
- Target accepts `"ip"` or `"ip:port"` formats
- Returns an error at construction time if `target` is empty

```go
a, err := artnet.New(target, universe, beatCh, downbeatCh, slowCh, pulseMS)
```

---

## OSC (`osc/`)

UDP unicast. Implements a minimal OSC 1.0 message encoder — no external library needed.

- Message format: padded address string + type tag `,f` + big-endian float32
- Sends `1.0` on trigger, `0.0` after `pulse_ms` milliseconds
- Addresses are pre-padded at construction time (4-byte boundary)

```go
o, err := osc.New(target, beatAddr, downbeatAddr, slowAddr, pulseMS)
```

---

## Pulse concept

All three senders share the same pulse model:

```
event fires
  → set DMX channel to 255 (or send OSC 1.0)
  → sleep pulse_ms
  → set DMX channel to 0 (or send OSC 0.0)
```

`pulse_ms` is independent per protocol so sACN and OSC can have different durations.
