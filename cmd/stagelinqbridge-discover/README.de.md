# cmd/stagelinqbridge-discover

> **[English version](README.md)**

Einstiegspunkt der StageLinQBridge-Anwendung.

## Aufgaben

- Wertet CLI-Flags aus (`-debug`)
- Ermittelt und lädt `config.json`
- Initialisiert alle Subsysteme in der richtigen Reihenfolge: Config → Token → Bridge → Output → StageLinQ-Stack → HTTP-Server
- Stellt das Web-UI unter `http://localhost:8080` bereit
- Öffnet beim Start automatisch ein Edge-App-Modus-Fenster
- Behandelt sauberes Herunterfahren bei `SIGINT` / `SIGTERM`

## Flags

| Flag | Standard | Beschreibung |
|------|----------|--------------|
| `-debug` | `false` | Aktiviert ausführliches Logging (Trace-Level). Ohne dieses Flag werden nur Fehler ausgegeben. |

## Auflösung des Config-Pfads

Die ausführbare Datei sucht `config.json` in dieser Reihenfolge:

1. `<exe-Verzeichnis>/config.json` — Produktiv-Layout
2. `configs/config.json` — Entwicklungs-Fallback (Arbeitsverzeichnis des Repos)
3. Existiert keine, wird eine Standard-`config.json` im exe-Verzeichnis angelegt

## HTTP-Endpunkte

| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| `GET` | `/` | Eingebettetes Web-UI (statische Dateien) |
| `GET` | `/events` | SSE-Stream — Events `state`, `beat`, `downbeat`, `slow` |
| `GET` | `/state` | Aktuelles Routing, retardDiv, timeSig, Protokoll-Aktivierungsstatus (JSON) |
| `POST` | `/config` | Setzt `retardDiv` und/oder `timeSig` zur Laufzeit |
| `POST` | `/route` | Aktiviert/deaktiviert den Beat-Output für ein bestimmtes Deck (`{"deck":0,"active":false}`) |
| `POST` | `/protocol` | Schaltet Protokoll-Sender um (`{"sacn":true,"artnet":false,"osc":true}`) |

## Bauen

```sh
go build -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```
