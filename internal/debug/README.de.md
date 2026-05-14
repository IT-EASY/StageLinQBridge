# internal/debug

> **[English version](README.md)**

Minimaler Level-basierter Konsolen-Logger für StageLinQBridge.

## Level

```
Error  →  nur kritische Fehler
Warn   →  nicht-fatale Probleme, eingeschränkter Betrieb
Info   →  Startmeldungen, Verbindungsereignisse
Debug  →  detaillierter Programmablauf
Trace  →  hochfrequente Events (Beat-Ticks, Paketdetails)
```

Im Normalbetrieb wird die Anwendung mit Level `Error` initialisiert — die Konsole bleibt still. Mit `-debug` beim Programmstart wird auf `Trace` umgeschaltet.

## Verwendung

```go
logger := debug.New(debug.Error)   // Produktion
logger := debug.New(debug.Trace)   // Debug-Modus

logger.Info("Server lauscht", "port", 8080)
logger.Warn("Config nicht gefunden, verwende Standardwerte")
logger.Error("Fatal", "error", err)
```

Schlüssel-Wert-Paare nach der Nachricht werden als `key=value` in derselben Zeile formatiert.

## Ausgabeformat

```
2026-05-14 21:04:01 [INFO] Server lauscht port=8080
```
