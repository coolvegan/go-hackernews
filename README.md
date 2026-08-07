# go-news

A Go service that fetches articles from [Hacker News](https://news.ycombinator.com/), keeps them in memory, and exposes them via an HTTP API and an MCP server.

## How it works

- On startup, the most recent article ID (watermark) is fetched from Hacker News.
- Older articles are then loaded (default `2000`).
- A ticker polls for new articles every second and adds them.
- Every `HALVEMEMORYDURATION` minutes (default `30`), the in-memory data structure is halved to limit memory usage (at least `MINARTICLE` articles are kept).
- Articles are fetched in parallel by workers (`WORKERCOUNT`, default `10`).

## Configuration (environment variables)

| Variable          | Default       | Description                                              |
|-------------------|---------------|----------------------------------------------------------|
| `SERVER`          | `localhost:7777` | Address of the HTTP debug server                       |
| `WORKERCOUNT`     | `10`          | Number of parallel fetch workers                          |
| `HALFTIME`        | `1800` (seconds → 30 min) | Interval (minutes) for halving memory             |
| `PRELOADITEMS`    | `2000`        | Number of older articles to load initially                |

## HTTP endpoints

| Route             | Method | Description                                               |
|-------------------|--------|-----------------------------------------------------------|
| `/`               | GET    | Debug view (`index.html`)                                 |
| `/api/items`      | GET    | All articles as JSON                                       |
| `/api/watermark`  | GET    | Current watermark ID as JSON (for polling)                |

![](images/hnews.png)
## MCP server

In addition to the HTTP API, the service starts an MCP server through which the loaded Hacker News articles can also be queried. The MCP server runs over HTTP on port `13333`.

## Start

```bash
go run main.go
```

Optionally with custom values:

```bash
SERVER=localhost:8080 MCPSERVER=localhost:13333 WORKERCOUNT=20 go run main.go
```

## systemd service

Build the binary and place it e.g. in `/usr/local/bin`:

```bash
go build -o go-news main.go
sudo install -m 0755 go-news /usr/local/bin/go-news
```

Unit file `/etc/systemd/system/go-news.service`:

```ini
[Unit]
Description=go-news Hacker News Fetcher & MCP-Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=marco
Group=marco
WorkingDirectory=/home/marco/go/src/github.com/coolvegan/go-hackernews
ExecStart=/usr/local/bin/go-news
Restart=on-failure
RestartSec=5

# Environment variables (optional, defaults see above)
Environment=SERVER=localhost:7777
Environment=MCPSERVER=localhost:13333
Environment=WORKERCOUNT=10
Environment=HALFTIME=1800
Environment=PRELOADITEMS=2000

# Resources / security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/log

# Logging to journald
StandardOutput=journal
StandardError=journal
SyslogIdentifier=go-news

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now go-news
```

Status and logs:

```bash
systemctl status go-news
journalctl -u go-news -f
```
