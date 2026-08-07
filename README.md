# go-news

Ein Go-Service, der Artikel von [Hacker News](https://news.ycombinator.com/) fetcht, im Speicher vorhält und über eine HTTP-API sowie einen MCP-Server bereitstellt.

## Funktionsweise

- Beim Start wird die aktuellste Artikel-ID (Watermark) von Hacker News ermittelt.
- Anschließend werden ältere Artikel (default `2000`) nachgeladen.
- Ein Ticker pollt sekündlich nach neuen Artikeln und fügt diese hinzu.
- Alle `HALVEMEMORYDURATION` Minuten (default `30`) wird die In-Memory-Datenstruktur halbiert, um den Speicherverbrauch zu begrenzen (mindestens `MINARTICLE` Artikel bleiben erhalten).
- Die Artikel werden über Worker (`WORKERCOUNT`, default `10`) parallel gefetcht.

## Konfiguration (Umgebungsvariablen)

| Variable          | Default       | Beschreibung                                           |
|-------------------|---------------|--------------------------------------------------------|
| `SERVER`          | `localhost:7777` | Adresse des HTTP-Debug-Servers                       |
| `WORKERCOUNT`     | `10`          | Anzahl paralleler Fetch-Worker                          |
| `HALFTIME`        | `1800` (Sekunden → 30 Min) | Intervall (Minuten) zum Halbieren des Speichers |
| `PRELOADITEMS`    | `2000`        | Anzahl der initial zu ladenden älteren Artikel          |

## HTTP-Endpunkte

| Route             | Methode | Beschreibung                                              |
|-------------------|---------|-----------------------------------------------------------|
| `/`               | GET     | Debug-View (`index.html`)                                 |
| `/api/items`      | GET     | Alle Artikel als JSON                                      |
| `/api/watermark`  | GET     | Aktuelle Watermark-ID als JSON (für Polling)              |

![](images/hnews.png)
## MCP-Server

Neben der HTTP-API startet der Service einen MCP-Server, über den die geladenen Hacker-News-Artikel ebenfalls abgefragt werden können. Der MCP-Server läuft über HTTP auf Port `13333`.

## Start

```bash
go run main.go
```

Optional mit eigenen Werten:

```bash
SERVER=localhost:8080 WORKERCOUNT=20 go run main.go
```

## systemd-Service

Binary bauen und z.B. nach `/usr/local/bin` legen:

```bash
go build -o go-news main.go
sudo install -m 0755 go-news /usr/local/bin/go-news
```

Unit-Datei `/etc/systemd/system/go-news.service`:

```ini
[Unit]
Description=go-news Hacker News Fetcher & MCP-Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=marco
Group=marco
WorkingDirectory=/home/marco/go/src/gittea.kittel.dev/marco/go-news
ExecStart=/usr/local/bin/go-news
Restart=on-failure
RestartSec=5

# Umgebungsvariablen (optional, Defaults siehe oben)
Environment=SERVER=localhost:7777
Environment=WORKERCOUNT=10
Environment=HALFTIME=1800
Environment=PRELOADITEMS=2000

# Ressourcen / Sicherheit
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/log

# Logging nach journald
StandardOutput=journal
StandardError=journal
SyslogIdentifier=go-news

[Install]
WantedBy=multi-user.target
```

Service aktivieren und starten:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now go-news
```

Status und Logs:

```bash
systemctl status go-news
journalctl -u go-news -f
```
