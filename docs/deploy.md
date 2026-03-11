# Deployment Guide

This guide covers everything needed to run `wa-bot-notif` in any environment.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Setup](#environment-setup)
3. [WhatsApp Pairing](#whatsapp-pairing)
4. [Local Development](#local-development)
5. [Docker Compose (local)](#docker-compose-local)
6. [Homelab Debian Deployment](#homelab-debian-deployment)
7. [Network Topology (Homelab + VPS)](#network-topology)
8. [Maintenance](#maintenance)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Local / CI

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.25+ | `GOTOOLCHAIN=auto` downloads if needed |
| CGO toolchain | — | Required for `mattn/go-sqlite3` |
| macOS | Xcode Command Line Tools (`xcode-select --install`) |
| Debian/Ubuntu | `sudo apt-get install build-essential libsqlite3-dev` |

### Docker deployment

| Tool | Notes |
|------|-------|
| Docker Engine | 24+ |
| Docker Compose plugin | `docker compose` (v2) |

---

## Environment Setup

Copy the example env file and fill in required values:

```bash
cp .env.example .env
```

Edit `.env`:

```env
# Required
AUTH_TOKEN=change-me-to-a-strong-random-token

# Optional — default send target when request omits userId and groupId
GROUP_JID=120363420921445931@g.us

# Port (default: 5000)
PORT=5000

# SQLite paths — use defaults for local dev; Docker overrides these automatically
AUTH_DB_DSN=file:auth.db?_foreign_keys=on
LOGS_DB_DSN=file:logs.db?_foreign_keys=on

# Log verbosity: trace | debug | info | warn | error (default: info)
LOG_LEVEL=info
```

Generate a strong `AUTH_TOKEN`:

```bash
openssl rand -hex 32
```

---

## WhatsApp Pairing

The service uses the WhatsApp multi-device protocol. On first run with no existing session, it prints a QR code to stdout. Scan it from the WhatsApp app on your phone:

**WhatsApp → Linked Devices → Link a Device**

Once paired, the session is persisted to `auth.db` (or the path in `AUTH_DB_DSN`). Subsequent restarts use the saved session without re-pairing.

> **Important:** Keep `auth.db` backed up. Losing it requires re-pairing.

---

## Local Development

```bash
# Install deps (first time or after go.mod changes)
GOTOOLCHAIN=auto go mod tidy

# Run (loads .env automatically)
GOTOOLCHAIN=auto go run ./cmd/api

# Tests
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go test -race ./...

# Lint
GOTOOLCHAIN=auto go vet ./...
gofmt -l .
```

The server listens on `http://localhost:5000` by default.

Quick smoke test (after pairing):

```bash
curl -s -X POST http://localhost:5000/send \
  -H "Authorization: Bearer <AUTH_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"message":"hello from wa-bot-notif"}' | jq
```

---

## Docker Compose (local)

### First-time setup

```bash
cp .env.example .env
# Edit .env — set AUTH_TOKEN and GROUP_JID

docker compose -f deploy/docker-compose.yml build
docker compose -f deploy/docker-compose.yml up
```

The first run shows the QR code in the container logs. Scan it, then restart in detached mode:

```bash
# Ctrl+C to stop, then:
docker compose -f deploy/docker-compose.yml up -d
```

### Day-to-day commands

```bash
# Start
docker compose -f deploy/docker-compose.yml up -d

# View logs
docker compose -f deploy/docker-compose.yml logs -f api

# Rebuild after code changes
docker compose -f deploy/docker-compose.yml up --build -d

# Stop
docker compose -f deploy/docker-compose.yml down

# Stop and remove volumes (DESTROYS session data — requires re-pairing)
docker compose -f deploy/docker-compose.yml down -v
```

### SQLite data persistence

All data lives in the named Docker volume `wa_bot_notif_data`, mounted at `/data` inside the container:

| File | Purpose |
|------|---------|
| `/data/auth.db` | WhatsApp session keys |
| `/data/logs.db` | Send and unauthorized audit logs |

---

## Homelab Debian Deployment

### 1. Install Docker on Debian

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo tee /etc/apt/keyrings/docker.asc > /dev/null
sudo chmod a+r /etc/apt/keyrings/docker.asc

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo usermod -aG docker $USER
```

### 2. Clone the repository

```bash
git clone https://github.com/<your-org>/wa-bot-notif.git
cd wa-bot-notif
```

### 3. Configure environment

```bash
cp .env.example .env
nano .env  # set AUTH_TOKEN, GROUP_JID, PORT
```

### 4. Create shared network (if integrating with other containers)

```bash
docker network create homelab_integration
```

Set in `.env`:
```env
INTEGRATION_NETWORK=homelab_integration
```

Other Docker projects on the same host can reach this API at `http://wa-bot-notif-api:5000`.

### 5. First run (pairing)

```bash
docker compose -f deploy/docker-compose.yml up
# Scan QR in WhatsApp → Linked Devices → Link a Device
# Ctrl+C once paired
```

### 6. Run as a service

```bash
# Enable restart policy in deploy/docker-compose.yml if desired:
# restart: unless-stopped  (change from "no")

docker compose -f deploy/docker-compose.yml up -d
docker compose -f deploy/docker-compose.yml ps
```

### 7. Auto-start on boot (systemd)

Create `/etc/systemd/system/wa-bot-notif.service`:

```ini
[Unit]
Description=wa-bot-notif Docker Compose
Requires=docker.service
After=docker.service network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/wa-bot-notif
ExecStart=/usr/bin/docker compose -f deploy/docker-compose.yml up -d
ExecStop=/usr/bin/docker compose -f deploy/docker-compose.yml down
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo mv /opt/wa-bot-notif  # or adjust path
sudo systemctl daemon-reload
sudo systemctl enable wa-bot-notif
sudo systemctl start wa-bot-notif
sudo systemctl status wa-bot-notif
```

---

## Network Topology

Recommended production setup for exposing the API externally:

```
Cloudflare DNS
      ↓ (HTTPS, Full Strict TLS)
DO VPS / edge proxy (Caddy or Nginx)
      ↓ (WireGuard tunnel)
Debian homelab (Docker Compose)
      → wa-bot-notif-api:5000
```

### VPS edge proxy (Caddy example)

```
api.yourdomain.com {
    reverse_proxy <homelab-wireguard-ip>:5000
}
```

### WireGuard tunnel (brief)

On VPS (`/etc/wireguard/wg0.conf`):
```ini
[Interface]
Address = 10.10.0.1/24
PrivateKey = <vps-private-key>
ListenPort = 51820

[Peer]
PublicKey = <homelab-public-key>
AllowedIPs = 10.10.0.2/32
```

On homelab:
```ini
[Interface]
Address = 10.10.0.2/24
PrivateKey = <homelab-private-key>

[Peer]
PublicKey = <vps-public-key>
Endpoint = <vps-public-ip>:51820
AllowedIPs = 10.10.0.1/32
PersistentKeepalive = 25
```

### Security baseline

- `AUTH_TOKEN` — strong random value (at least 32 hex chars)
- Secrets never committed to git (`.env` is gitignored)
- Homelab firewall restricts port 5000 to WireGuard ingress only
- TLS at Cloudflare edge (Full Strict mode)
- Consider IP allowlist for trusted callers at the proxy layer

---

## Maintenance

### Backup session data

The WhatsApp session (`auth.db`) is the only stateful file that can't be recovered without re-pairing:

```bash
# From the homelab host
docker run --rm -v wa_bot_notif_data:/data \
  -v $(pwd)/backups:/backup \
  debian:bookworm-slim \
  cp /data/auth.db /backup/auth-$(date +%Y%m%d).db
```

Automate with a daily cron:

```bash
crontab -e
# Add:
0 3 * * * docker run --rm -v wa_bot_notif_data:/data -v /opt/backups/wa-bot:/backup debian:bookworm-slim cp /data/auth.db /backup/auth-$(date +\%Y\%m\%d).db
```

### View audit logs

```bash
# Connect to the SQLite log database
docker run --rm -it -v wa_bot_notif_data:/data \
  keinos/sqlite3 sqlite3 /data/logs.db

# Useful queries:
SELECT * FROM logs ORDER BY timestamp DESC LIMIT 20;
SELECT ip, COUNT(*) as hits FROM unauthorized_logs GROUP BY ip ORDER BY hits DESC;
SELECT DATE(timestamp) as day, COUNT(*) as sends FROM logs GROUP BY day;
```

### Update the service

```bash
cd /opt/wa-bot-notif
git pull
docker compose -f deploy/docker-compose.yml up --build -d
```

### Log retention

Audit logs older than 30 days are automatically deleted by the `StartRetention` goroutine (runs daily at startup). No manual intervention needed.

---

## Troubleshooting

### QR code not appearing

- The container needs to output to a TTY on first run — use `docker compose up` (without `-d`) for pairing.
- If session already exists but is invalid, remove the volume: `docker compose down -v` then re-pair.

### Service shows `not_ready` on `/readyz`

The WhatsApp connection hasn't established yet. Check logs:

```bash
docker compose -f deploy/docker-compose.yml logs -f api
```

Look for `[WA] connected` — if not present, the client is still connecting or backing off. Wait up to 60s on retry. If the session is invalid, re-pair.

### `401 Unauthorized` on `/send`

Verify the `Authorization` header is exactly `Bearer <AUTH_TOKEN>` (no extra spaces, correct token value from `.env`).

### Send returns `503 Service Unavailable`

WhatsApp client is not ready. Check `/readyz`. If the service just started, wait for connection. If persistent, check logs for reconnect failures.

### Send returns `502 Bad Gateway`

The send succeeded at the API layer but WhatsApp rejected or dropped the message. Check `sent_to` in the response to confirm the JID is valid.

### Container exits immediately

Almost always a missing `AUTH_TOKEN`. Check:

```bash
docker compose -f deploy/docker-compose.yml logs api | head -20
```

You should see: `failed to load config: AUTH_TOKEN must be non-empty`
