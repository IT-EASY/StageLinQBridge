# internal/stagelinq

> **[English version](README.md)**

StageLinQ-Protokollimplementierung — Discovery, Hauptverbindung, StateMap und BeatInfo.

## Protokoll-Übersicht

StageLinQ ist ein proprietäres Discovery- und Datenprotokoll, das von Denon DJ / inMusic-Hardware mit Engine OS verwendet wird. Es arbeitet sowohl über UDP (Discovery) als auch TCP (Datendienste).

```
UDP :26999 Multicast   ←→  Discovery-Ankündigungen
TCP <dynamischer Port> ←→  Hauptverbindung (Service-Aushandlung)
TCP <dynamischer Port> ←→  StateMap  (Key-Value-Status-Subscriptions)
TCP <dynamischer Port> ←→  BeatInfo  (hochfrequenter Beat-Stream)
```

## Sub-Pakete

### `announce/`

Sendet eine StageLinQ-Präsenzankündigung im lokalen Netz, damit Denon-Geräte diese Bridge erkennen und sich verbinden können.

- Sendet UDP-Multicast an `224.0.0.1:26999`
- Kündigt sich regelmäßig neu an
- Überträgt eigenes Token, Software-Name/-Version und Hauptverbindungs-Port

### `discovery/`

Lauscht auf Ankündigungen anderer StageLinQ-Geräte im Netz.

- Nützlich für Diagnose (welche Geräte sind vorhanden?)
- Die Bridge nutzt dies passiv — Geräte verbinden sich eingehend zu `mainconn`

### `mainconn/`

TCP-Server, der eingehende Verbindungen von Denon-Geräten akzeptiert. Behandelt den Service-Fähigkeitsaustausch und weist das Gerät an, sich mit den StateMap- und BeatInfo-Servern zu verbinden.

### `statemap/`

TCP-Server für StateMap-Protokollverbindungen. Akzeptiert eine Liste von Subscription-Pfaden und liefert Wertaktualisierungen als `StateUpdate`-Events.

Verwendete Hauptpfade:

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

Werte kommen als JSON-Strings: `{"state":true}`, `{"value":128.5}`, `{"string":"Artist"}`.

### `beatinfo/`

TCP-Server, der den hochfrequenten BeatInfo-Stream dekodiert. Liefert `BeatEvent`-Structs mit ≈33 Hz, die die Beat-Position pro Player als Gleitkommazahl enthalten. Der ganzzahlige Anteil ist die Beat-Nummer; der Bruchteil ist die Phase innerhalb des aktuellen Beats.

### `encoding/`

Low-Level-Wire-Protokoll-Hilfsfunktionen: Network-String-Encoding (längen-präfixiertes UTF-16), Binary-Reader/-Writer-Wrapper. Wird intern von allen anderen StageLinQ-Sub-Paketen verwendet.

### `token/`

16-Byte-Token zur Identifikation dieses Clients beim Denon-Gerät. Wird beim ersten Start zufällig generiert und in `config.json` gespeichert.
