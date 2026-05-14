# internal/output

> **[English version](README.md)**

Output-Manager und Protokoll-Sender. Verteilt Beat-Events aus der Bridge an ein oder mehrere Licht-/Automatisierungsprotokolle gleichzeitig.

## Architektur

```
Tracker.OutputFn  →  Manager.Dispatch(event, deck)
                         ├── sacn.Sender   (wenn aktiviert)
                         ├── artnet.Sender (wenn aktiviert)
                         └── osc.Sender    (wenn aktiviert)
```

Jeder Sender führt seinen Impuls in einer eigenen Goroutine aus (`go flash(ch)`), sodass ein langsamer Netzwerk-Write den Beat-Stream nie blockiert.

## Manager

```go
m := output.New(cfg, logger)
defer m.Close()

m.Dispatch("beat", 0)        // Beat-Event für Deck 0 auslösen
m.SetEnabled("sacn", true)   // Protokoll zur Laufzeit umschalten
m.Enabled()                  // map[string]bool{"sacn":…,"artnet":…,"osc":…}
```

Sender, die nicht initialisiert werden können, werden als Warnung geloggt und übersprungen — die Anwendung läuft mit den verbleibenden Sendern weiter.

---

## sACN / E1.31 (`sacn/`)

UDP-Multicast. Die Multicast-Gruppe ergibt sich aus der Universums-Nummer:

```
239.255.<universe_hi>.<universe_lo>:5568
```

- 638-Byte-E1.31-Paket, vollständiger 512-Kanal-DMX-Frame
- Quellname: `StageLinQBridge`
- Zufällige UUID-CID beim Start generiert (RFC 4122 §4.4)
- Sequenznummer wird bei jedem Paket erhöht

```go
s, err := sacn.New(universe, beatCh, downbeatCh, slowCh, pulseMS)
```

---

## Art-Net ArtDmx (`artnet/`)

UDP-**Unicast** an eine konfigurierte Ziel-IP. **Broadcast wird bewusst nicht unterstützt.**

> Beat-Trigger feuern kontinuierlich mit ≈33 Events/s. Als Broadcast würden diese das gesamte Netzwerksegment fluten — in einer Live-Produktionsumgebung inakzeptabel.

- 530-Byte-ArtDmx-Paket (OpCode 0x5000), vollständiger 512-Kanal-DMX-Frame
- Standard-Port: 6454
- `target` akzeptiert `"ip"` oder `"ip:port"`
- Gibt beim Erstellen einen Fehler zurück, wenn `target` leer ist

```go
a, err := artnet.New(target, universe, beatCh, downbeatCh, slowCh, pulseMS)
```

---

## OSC (`osc/`)

UDP-Unicast. Implementiert einen minimalen OSC-1.0-Message-Encoder — keine externe Bibliothek erforderlich.

- Nachrichtenformat: aufgefüllter Adressstring + Type-Tag `,f` + Big-Endian-float32
- Sendet `1.0` beim Trigger, `0.0` nach `pulse_ms` Millisekunden
- Adressen werden beim Erstellen vorberechnet (4-Byte-Grenze)

```go
o, err := osc.New(target, beatAddr, downbeatAddr, slowAddr, pulseMS)
```

---

## Impuls-Konzept

Alle drei Sender verwenden dasselbe Impulsmodell:

```
Event feuert
  → DMX-Kanal auf 255 setzen (oder OSC 1.0 senden)
  → pulse_ms warten
  → DMX-Kanal auf 0 setzen (oder OSC 0.0 senden)
```

`pulse_ms` ist pro Protokoll unabhängig, sodass sACN und OSC unterschiedliche Impulslängen haben können.
