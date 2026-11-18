# statusboard

Internal operational health monitoring dashboard for local branch services and network infrastructure. Built for IT Support staff to track live device uptime, trigger manual re-checks, and review incident downtime history.

![Statusboard Overview Placeholder](docs/screenshots/dashboard.png)

## Stack

- **Backend:** Go 1.26
- **Frontend:** HTML + HTMX (polling & dynamic partial updates)
- **Database:** SQLite (`/var/lib/statusboard/statusboard.db`)

## Local Development Setup

### Prerequisites
- Go 1.26 or later
- CGO enabled (required for `mattn/go-sqlite3` drivers)

### Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/internal-it/statusboard.git
   cd statusboard
   ```

2. Copy the sample database environment configuration if needed, or run direct:
   ```bash
   go run ./cmd/statusboard
   ```

3. Open your browser at `http://localhost:8080` to view the board.

## Production Deployment

The app runs as a systemd service on our local branch servers running Debian 12.

1. Build the static binary:
   ```bash
   CGO_ENABLED=1 GOOS=linux go build -ldflags="-extldflags '-static'" -o statusboard ./cmd/statusboard
   ```

2. Copy binary and configure `/etc/systemd/system/statusboard.service`:
   ```ini
   [Unit]
   Description=Statusboard Operational Health Tracker
   After=network.target

   [Service]
   Type=simple
   User=it-ops
   WorkingDirectory=/var/lib/statusboard
   ExecStart=/usr/local/bin/statusboard -db /var/lib/statusboard/statusboard.db
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

3. Ensure WAL mode is active on target SQLite database:
   ```bash
   sqlite3 /var/lib/statusboard/statusboard.db "PRAGMA journal_mode=WAL;"
   ```

4. Enable and start service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now statusboard
   ```
