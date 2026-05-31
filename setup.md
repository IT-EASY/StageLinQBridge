# StageLinQBridge – Setup & Deployment

## Build

### Debug-Build (mit Konsolenfenster)

```
go build -o StageLinQBridge-debug.exe ./cmd/stagelinqbridge-discover/
```

### Release-Build (kein Konsolenfenster, startet direkt die Web-UI)

```
go build -ldflags="-H windowsgui" -o StageLinQBridge.exe ./cmd/stagelinqbridge-discover/
```

---

## Windows SmartAppControl (Windows 11)

Windows 11 blockiert unsignierte EXEs mit **SmartAppControl (SAC)** — ohne
Ausweichmöglichkeit (kein „Trotzdem ausführen"-Button).

### Sofortlösung: SAC deaktivieren (empfohlen für DJ-Laptops)

> **Windows-Sicherheit → App- & Browsersteuerung → Smart App Control → Aus**

⚠️ Das ist ein **einmaliger, dauerhafter Schalter** — lässt sich ohne
Windows-Neuinstallation nicht rückgängig machen. Für einen dedizierten
DJ-Laptop, auf dem nur bekannte Software läuft, ist das die pragmatische Wahl.

### Langfristige Lösung: Code-Signing-Zertifikat

Da StageLinQBridge Open Source ist, stehen kostenlose Optionen zur Verfügung:

| Anbieter | Kosten | Link |
|---|---|---|
| **Certum Open Source** | kostenlos | https://www.certum.eu/en/open-source/ |
| **SignPath Foundation** | kostenlos | https://signpath.io/product/open-source |
| Certum Standard | ~50 €/Jahr | günstigstes kommerzielles OV-Zertifikat |

Mit einem gültigen Zertifikat müssen sowohl **Installer** als auch **EXE**
signiert werden — SAC prüft jede EXE beim Start unabhängig.

---

## Windows-Firewall

Die App benötigt eingehende **und** ausgehende Freigabe für **TCP und UDP**.
Einmalig als Administrator ausführen:

```
netsh advfirewall firewall add rule name="StageLinQBridge" dir=in action=allow program="%CD%\StageLinQBridge.exe" enable=yes protocol=any
netsh advfirewall firewall add rule name="StageLinQBridge" dir=out action=allow program="%CD%\StageLinQBridge.exe" enable=yes protocol=any
```

> **Hinweis:** `%CD%` durch den tatsächlichen Pfad zur EXE ersetzen, z. B.
> `C:\StageLinQ_Bridge\StageLinQBridge.exe`

Alternativ per GUI: **Windows-Sicherheit → Firewall → App durch Firewall zulassen**

---

## Konfiguration (`config.json`)

Die `config.json` liegt im selben Verzeichnis wie die EXE.
Beim ersten Start öffnet sich automatisch ein **Setup-Dialog** zur Auswahl
des LAN-Adapters — die IP wird dann automatisch in `config.json` gespeichert.

### Manuelle Konfiguration

```json
{
  "network": {
    "lan_ip": "192.168.1.100"
  }
}
```

Die IP-Adresse des Adapters ermitteln mit `ipconfig` — den LAN-Adapter
heraussuchen, über den der PRIME 4 erreichbar ist.

### Token

Der Token wird beim ersten Start automatisch generiert und in `config.json`
gespeichert. Ein **Token-Rotation Watchdog** stellt sicher, dass die Verbindung
zum PRIME 4 auch dann zustande kommt, wenn der erste Token-Versuch fehlschlägt.

> ⚠️ **`config.json` nie löschen** — sie enthält den Token, über den der PRIME 4
> die App identifiziert. Bei einem neuen Token muss die Verbindung neu aufgebaut
> werden, was mehrere Sekunden dauern kann.

---

## Deployment auf dem DJ-Laptop

1. `StageLinQBridge.exe` in ein Verzeichnis kopieren (z. B. `C:\StageLinQ_Bridge\`)
2. SmartAppControl deaktivieren (siehe oben)
3. Firewall-Regeln anlegen (Pfad zur EXE anpassen)
4. `StageLinQBridge.exe` starten — Setup-Dialog erscheint beim ersten Start
5. LAN-Adapter auswählen → App verbindet sich automatisch mit dem PRIME 4
