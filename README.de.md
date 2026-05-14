# StageLinQBridge

> **[English version](README.md)**

Eine Go-Anwendung, die **Denon PRIME DJ-Hardware** (PRIME 4, PRIME 2, SC6000 und kompatible Geräte) über **sACN/E1.31**, **Art-Net** und **OSC** an Licht- und Automationssysteme anbindet.

StageLinQBridge empfängt Live-Beat-Events über das StageLinQ-Protokoll, wertet Kanalfader- und Crossfader-Stellungen aus und sendet DMX- oder OSC-Impulse — beatgenau, pro Deck, mit automatischem Routing.

---

## Funktionen

| Funktion | Beschreibung |
|----------|--------------|
| **Beat** | Feuert auf jeden Viertelschlag des aktiven Decks |
| **Downbeat** | Feuert auf Zählzeit 1 jedes Taktes (3/4 oder 4/4, konfigurierbar) |
| **Slow / Retard** | Feuert jeden N-ten Beat (Standard ÷64 ≈ 32 s bei 120 BPM), Divisor einstellbar |
| **Fader-Routing** | Nur das Deck, dessen Kanalfader geöffnet ist, produziert Output |
| **Crossfader** | A/B/Through-Kanalzuweisung wird automatisch ausgewertet |
| **Manuelles Muting** | LIVE-Button pro Deck im Web-UI überschreibt das Auto-Routing |
| **sACN / E1.31** | UDP-Multicast — keine Ziel-IP erforderlich |
| **Art-Net ArtDmx** | UDP **Unicast only** — Broadcast ist bewusst nicht unterstützt |
| **OSC** | UDP-Unicast, float32 `1.0` Trigger / `0.0` Release |
| **Laufzeit-Toggle** | Ausgabeprotokolle im UI ohne Neustart aktivieren/deaktivieren |
| **Web-UI** | Öffnet beim Start automatisch als eigenständiges Edge-App-Fenster |
| **Kein Installer** | Nur `.exe` + `config.json` im selben Ordner — fertig |

---

## Schnellstart

### Bauen

```sh
go build -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```

### Starten

```sh
# Normalbetrieb — still (nur Fehler auf der Konsole)
StageLinQBridge.exe

# Ausführliches Logging für die Diagnose
StageLinQBridge.exe -debug
```

Beim Start:

1. Liest `config.json` aus dem Verzeichnis der ausführbaren Datei (legt eine Standard-Config an, falls nicht vorhanden)
2. Kündigt sich per StageLinQ-UDP-Multicast im Netz an
3. Wartet auf ein Denon PRIME-Gerät, das sich verbindet und Beat- sowie Statusdaten liefert
4. Öffnet `http://localhost:8080` automatisch im Edge-App-Modus

---

## Konfiguration

`config.json` wird beim ersten Start automatisch mit sicheren Standardwerten erstellt (alle Ausgaben deaktiviert). Die Datei liegt im selben Verzeichnis wie die ausführbare Datei.

```jsonc
{
  "network": {
    "lan_ip": "192.168.1.x",    // IP des LAN-Interfaces; leer = alle Interfaces
    "token": ""                   // wird beim ersten Start generiert, nicht manuell ändern
  },
  "sacn": {
    "enabled": false,
    "universe": 1,                // DMX-Universum 1–63999
    "channels": {
      "beat":     1,              // DMX-Kanalnummer für Beat-Impuls (1-basiert)
      "downbeat": 2,              // DMX-Kanal für Taktschlag 1
      "slow":    11               // DMX-Kanal für Slow/Retard-Impuls
    },
    "pulse_ms": 50                // wie lange der Kanal auf 255 bleibt (ms)
  },
  "artnet": {
    "enabled": false,
    "target":   "192.168.1.50",  // Ziel-IP (erforderlich — kein Broadcast)
    "universe": 0,                // Art-Net Port-Adresse 0–32767
    "channels": { "beat": 1, "downbeat": 2, "slow": 11 },
    "pulse_ms": 50
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

### Pfad der Config-Datei

| Szenario | Pfad |
|----------|------|
| Produktiv | `<exe-Verzeichnis>/config.json` |
| Entwicklung (Repo) | `configs/config.json` (Fallback) |

Existiert keine, wird eine Standard-`config.json` neben der exe angelegt.

---

## Ausgabeprotokolle

### sACN / E1.31

UDP-Multicast an `239.255.{universe_hi}.{universe_lo}:5568`. Keine Ziel-IP nötig — die Multicast-Adresse ergibt sich aus der Universums-Nummer. Vollständiges 638-Byte-E1.31-Paket mit 512-Kanal-DMX-Frame, Quellname `StageLinQBridge`.

### Art-Net ArtDmx

UDP-**Unicast** an die konfigurierte `target`-IP (Standard-Port 6454). **Broadcast ist bewusst nicht unterstützt** — Beat-Trigger feuern kontinuierlich (≈33 Events/s aus dem BeatInfo-Stream) und würden das gesamte Netzwerksegment fluten. Ist `target` leer, wird der Sender nicht initialisiert.

### OSC

UDP-Unicast an `"ip:port"`. Sendet float32 `1.0` auf der konfigurierten Adresse, wenn ein Event auslöst, gefolgt von float32 `0.0` nach `pulse_ms` Millisekunden.

---

## Web-UI

Erreichbar unter `http://localhost:8080`. Öffnet beim Start automatisch im Edge-App-Modus.

| Element | Beschreibung |
|---------|--------------|
| 4-Deck-Leiste | Interpret, Titel, BPM, Beat-Phase-Blöcke |
| **LIVE**-Button | Grün = aktiv · Gedimmt = auto-gemutet (Fader zu) · Rot = manuell gemutet |
| Crossfader | A↔B-Positionsanzeige |
| **3/4 · 4/4** | Taktart-Auswahl |
| **÷16 · ÷32 · ÷64 · ÷128** | Slow/Retard-Divisor |
| **sACN · Art-Net · OSC** | Protokoll-Aktivierung zur Laufzeit |

---

## Beat-Kanäle erklärt

| Kanal | Wann | Typische Verwendung |
|-------|------|---------------------|
| **beat** | Jeder Viertelschlag | Laufender Cue, Chaser-Schritt |
| **downbeat** | Zählzeit 1 jedes Taktes | Takt-synchroner Effekt-Trigger |
| **slow** | Alle ÷N Beats | Atmosphärischer Effekt, Nebelmaschine |

*„Slow/Retard"* — „Retard" ist der etablierte Begriff aus der DJ-/Lichttechnik (z. B. Avolites Titan). Im Code und in der Config heißt der Kanal `slow`, um Missverständnisse zu vermeiden; beide Begriffe erscheinen im UI.

---

## Repository-Struktur

```
cmd/
  stagelinqbridge-discover/   Einstiegspunkt, HTTP-Server, Web-UI-Host
internal/
  bridge/                     SSE-Hub + Deck-Statustracker
  config/                     JSON-Konfiguration (Laden, Speichern, Defaults)
  debug/                      Level-basierter Konsolen-Logger
  network/                    Interface-Validierung, Broadcast-Hilfsfunktionen
  output/                     Output-Manager
    artnet/                   Art-Net ArtDmx-Sender (Unicast)
    osc/                      OSC-UDP-Sender
    sacn/                     sACN-E1.31-Sender (Multicast)
  stagelinq/
    announce/                 StageLinQ-UDP-Announcer
    beatinfo/                 BeatInfo-TCP-Server + Paket-Decoder
    discovery/                Device-Discovery-Listener
    encoding/                 Wire-Protokoll-Encoding/-Decoding
    mainconn/                 Haupt-Verbindungs-TCP-Server
    statemap/                 StateMap-Subscription-Server
    token/                    Token-Generierung und Hex-Parsing
web/                          Eingebettetes Web-UI (go:embed)
configs/                      Entwicklungs-config.json
```

---

## Entwicklungsumgebung

- **Go** 1.22+
- **OS** Windows 10/11 (primäres Ziel; Linux-Builds funktionieren, Edge-App-Fenster öffnet sich jedoch nicht)
- **Hardware** Denon PRIME 4 mit Engine OS 4.x im selben LAN-Segment

---

## Danksagung

Protokoll-Recherche und Referenzimplementierung von Carl Kittelberger:
[github.com/icedream/go-stagelinq](https://github.com/icedream/go-stagelinq)

---

## Haftungsausschluss

StageLinQ ist ein proprietäres Protokoll von Denon DJ / inMusic. Dieses Projekt ist ein unabhängiges Interoperabilitätsprojekt und steht in keiner Verbindung zu Denon DJ oder inMusic.
