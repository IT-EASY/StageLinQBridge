# StageLinQBridge – Setup & Deployment

## Build

### Debug-Build (mit Konsolenfenster)

```
go build -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```

### Release-Build (kein Konsolenfenster, startet direkt die Web-UI)

```
go build -ldflags="-H windowsgui" -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```

---

## Windows-Firewall

Die App benötigt eingehende **und** ausgehende Freigabe für **TCP und UDP** auf allen Ports.
Einmalig als Administrator ausführen:

```
netsh advfirewall firewall add rule name="StageLinQBridge" dir=in action=allow program="%CD%\StageLinQBridge.exe" enable=yes protocol=any
netsh advfirewall firewall add rule name="StageLinQBridge" dir=out action=allow program="%CD%\StageLinQBridge.exe" enable=yes protocol=any
```

> **Hinweis:** `%CD%` muss durch den tatsächlichen Pfad zur EXE ersetzt werden, z. B.  
> `C:\DJTools\StageLinQBridge\StageLinQBridge.exe`

Alternativ lässt sich die App auch per GUI unter  
**Windows-Sicherheit → Firewall → App durch Firewall zulassen** freigeben.

---

## Konfiguration (`config.json`)

Die `config.json` liegt im selben Verzeichnis wie die EXE.  
Beim ersten Start wird eine Vorlage automatisch angelegt.

### Wichtigste Einstellung: `lan_ip`

```json
{
  "network": {
    "lan_ip": "192.168.1.100"
  }
}
```

Hier muss die **IPv4-Adresse des Netzwerk-Adapters** eingetragen werden,  
der sich im selben Netzwerk wie der Denon PRIME 4 befindet.

Die aktuell verfügbaren IP-Adressen lassen sich ermitteln mit:

```
ipconfig
```

Den **LAN-Adapter** heraussuchen, über den der PRIME 4 erreichbar ist,  
und dessen IPv4-Adresse in `lan_ip` eintragen.

**Bleibt `lan_ip` leer**, sendet die App im Auto-Detect-Modus nur auf physischen  
Ethernet-Adaptern (Typ 6) — auf Systemen mit mehreren Adaptern (z. B. LAN + VPN)  
empfiehlt sich dennoch die explizite Konfiguration.

---

## Deployment auf dem DJ-Laptop

1. `StageLinQBridge.exe` **und `config.json`** in ein gemeinsames Verzeichnis kopieren.
2. `config.json` öffnen und `lan_ip` auf die IP des Adapters setzen, der den PRIME 4 sieht.
3. Firewall-Regeln wie oben anlegen (Pfad zur EXE anpassen).
4. `StageLinQBridge.exe` starten — die Web-UI öffnet sich automatisch.

> ⚠️ **Wichtig: `config.json` immer mitdeployen!**  
> Die Datei enthält ein `token`-Feld (zufällig generiert beim ersten Start), das die App  
> gegenüber dem PRIME 4 identifiziert. Beim **allerersten Kontakt** mit einem neuen Token  
> erscheint auf dem PRIME 4 Touchscreen eine Bestätigungsabfrage — wird diese nicht  
> bestätigt (oder übersehen), verweigert der PRIME 4 dauerhaft die Verbindung für dieses Token.  
>  
> **Lösung:** Immer dieselbe `config.json` verwenden und nur `lan_ip` anpassen.  
> Bei einem wirklichen Neustart mit neuem Token: Bestätigungsdialog am PRIME 4 beachten.
