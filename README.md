# wa-bot-notif

WhatsApp notification service — Go + [whatsmeow](https://github.com/tulir/whatsmeow).

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/send` | Bearer | Send a WhatsApp message |
| `GET` | `/contacts` | Bearer | List synced contacts |
| `GET` | `/messages` | Bearer | Recent message cache (runtime memory) |
| `GET` | `/healthz` | — | Always `200 OK` |
| `GET` | `/readyz` | — | `200` when WA connected, `503` otherwise |

### POST /send

```json
{ "message": "hello", "userId": "628xxx", "groupId": "120363...@g.us" }
```

Target priority: `userId` → `groupId` → `GROUP_JID` env fallback.

Response: `{ "success": true, "sent_to": "<jid>", "timestamp": "<RFC3339>" }`

## Configuration

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `AUTH_TOKEN` | — | ✅ | Bearer token for all authenticated endpoints |
| `PORT` | `5000` | — | HTTP listen port (1–65535) |
| `GROUP_JID` | — | — | Default send target when request omits userId/groupId |
| `AUTH_DB_DSN` | `file:auth.db?_foreign_keys=on` | — | WhatsApp session SQLite DSN |
| `LOGS_DB_DSN` | `file:logs.db?_foreign_keys=on` | — | Audit log SQLite DSN |
| `LOG_LEVEL` | `info` | — | Zerolog level: trace, debug, info, warn, error |

Copy `.env.example` to `.env` and fill in `AUTH_TOKEN` and `GROUP_JID`.

## Run Locally

Prerequisites: Go 1.25+, CGO toolchain, SQLite dev headers.

- macOS: Xcode Command Line Tools
- Debian/Ubuntu: `build-essential libsqlite3-dev`

```bash
GOTOOLCHAIN=auto go run ./cmd/api
```

Development checks:

```bash
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go vet ./...
GOTOOLCHAIN=auto gofmt -l .
```

## Docker

```bash
cp .env.example .env
# fill in AUTH_TOKEN

docker compose -f deploy/docker-compose.yml up --build -d
docker compose -f deploy/docker-compose.yml logs -f api
```

SQLite files are persisted in Docker volume `wa_bot_notif_data`.

For shared-network deployments, the service is reachable as `http://wa-bot-notif-api:5000` on the `homelab_integration` network:

```bash
docker network create homelab_integration
```

Set `INTEGRATION_NETWORK=homelab_integration` in `.env`.

## Project Structure

```
.
├── cmd/api/          — entry point
├── internal/
│   ├── config/       — env config loading + validation
│   ├── httpapi/      — HTTP handlers
│   ├── storage/      — SQLite audit log store
│   └── wa/           — WhatsApp connection manager
├── deploy/           — Docker Compose
├── docs/ai/          — AI agent guidance and planning docs
├── Dockerfile
└── go.mod
```

## AI Documentation

- `AGENTS.md` — agent entrypoint
- `docs/ai/README.md` — full AI docs index

## Deployment Guide

- `docs/deploy.md` — full deployment runbook (local, Docker, homelab, WireGuard)
