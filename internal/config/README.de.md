# internal/config

> **[English version](README.md)**

JSON-basierte Konfiguration für StageLinQBridge. Verwaltet Laden, Speichern und sichere Standardwerte.

## Funktionen

```go
func Default() *Config              // gibt Config mit allen deaktivierten Outputs zurück
func Load(path string) (*Config, error)
func Save(path string, cfg *Config) error
```

`Load` startet von `Default()` und dekodiert das JSON darüber — unbekannte Felder werden ignoriert, fehlende behalten ihren Standardwert.

## Config-Struct

```go
type Config struct {
    Network NetworkConfig
    SACN    SACNConfig
    ArtNet  ArtNetConfig
    OSC     OSCConfig
}
```

### NetworkConfig

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `lan_ip` | `string` | LAN-Interface-IP für StageLinQ-Ankündigungen. Leer = alle Interfaces. |
| `token` | `string` | Hex-kodiertes Client-Token. Wird automatisch generiert und gespeichert. |

### SACNConfig / ArtNetConfig — gemeinsame Felder

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `enabled` | `bool` | Ob dieser Sender beim Start aktiv ist |
| `universe` | `uint16` | DMX-Universum (sACN: 1–63999 · Art-Net: 0–32767) |
| `channels.beat` | `uint16` | DMX-Kanalnummer für Beat-Impuls (1-basiert) |
| `channels.downbeat` | `uint16` | DMX-Kanal für Taktschlag 1 |
| `channels.slow` | `uint16` | DMX-Kanal für Slow/Retard-Impuls |
| `pulse_ms` | `int` | Wie lange der Kanal auf 255 bleibt (ms) |

### ArtNetConfig — zusätzliches Feld

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `target` | `string` | Ziel-IP oder `"ip:port"`. **Pflichtfeld** — Broadcast wird nicht unterstützt. Leer = Sender nicht initialisiert. |

### OSCConfig

| Feld | Typ | Beschreibung |
|------|-----|--------------|
| `enabled` | `bool` | Ob der Sender beim Start aktiv ist |
| `target` | `string` | `"ip:port"` des OSC-Empfängers |
| `addresses.beat` | `string` | OSC-Adresse für Beat-Events |
| `addresses.downbeat` | `string` | OSC-Adresse für Downbeat-Events |
| `addresses.slow` | `string` | OSC-Adresse für Slow/Retard-Events |
| `pulse_ms` | `int` | Trigger-Dauer (ms) |

## Standardwerte

```
sACN/Art-Net-Kanäle: beat=1, downbeat=2, slow=11
pulse_ms: 50
Art-Net target: "" (deaktiviert)
OSC target: "" (deaktiviert)
OSC-Adressen: /stagelinq/beat · /stagelinq/downbeat · /stagelinq/slow
```
