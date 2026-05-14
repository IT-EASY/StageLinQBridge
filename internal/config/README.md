# internal/config

> **[Deutsche Version](README.de.md)**

JSON-based configuration for StageLinQBridge. Handles loading, saving, and providing safe defaults.

## Functions

```go
func Default() *Config              // returns a Config with all outputs disabled
func Load(path string) (*Config, error)
func Save(path string, cfg *Config) error
```

`Load` starts from `Default()` and decodes the JSON on top — unknown fields are ignored and missing fields keep their default values.

## Config struct

```go
type Config struct {
    Network NetworkConfig
    SACN    SACNConfig
    ArtNet  ArtNetConfig
    OSC     OSCConfig
}
```

### NetworkConfig

| Field | Type | Description |
|-------|------|-------------|
| `lan_ip` | `string` | LAN interface IP for StageLinQ announcements. Empty = all interfaces. |
| `token` | `string` | Hex-encoded client token. Auto-generated and saved on first run. |

### SACNConfig / ArtNetConfig — shared fields

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | `bool` | Whether this sender is active at startup |
| `universe` | `uint16` | DMX universe (sACN: 1–63999 · Art-Net: 0–32767) |
| `channels.beat` | `uint16` | DMX channel number for beat pulse (1-based) |
| `channels.downbeat` | `uint16` | DMX channel number for bar beat 1 |
| `channels.slow` | `uint16` | DMX channel number for slow/retard pulse |
| `pulse_ms` | `int` | How long the channel stays at 255 (ms) |

### ArtNetConfig — additional field

| Field | Type | Description |
|-------|------|-------------|
| `target` | `string` | Target IP or `"ip:port"`. **Required** — broadcast is not supported. If empty, the sender is not initialised. |

### OSCConfig

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | `bool` | Whether the sender is active at startup |
| `target` | `string` | `"ip:port"` of the OSC receiver |
| `addresses.beat` | `string` | OSC address for beat events |
| `addresses.downbeat` | `string` | OSC address for downbeat events |
| `addresses.slow` | `string` | OSC address for slow/retard events |
| `pulse_ms` | `int` | Trigger duration (ms) |

## Defaults

```
sACN/Art-Net channels: beat=1, downbeat=2, slow=11
pulse_ms: 50
Art-Net target: "" (disabled)
OSC target: "" (disabled)
OSC addresses: /stagelinq/beat · /stagelinq/downbeat · /stagelinq/slow
```
