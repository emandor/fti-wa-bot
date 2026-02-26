# wa-bot-notif

WhatsApp notification service with authenticated `POST /send` API.

The repository now runs as a Go service with `whatsmeow` under `go-service/**`.

## Features

### API behavior
- `POST /send`
  - body: `message` (required), `userId` (optional), `groupId` (optional)
  - target priority: `userId` -> `groupId` -> `GROUP_JID` fallback
  - auth: `Authorization: Bearer <AUTH_TOKEN>`
  - response shape: `{ success, sent_to, timestamp }`
- `GET /contacts`
  - auth: `Authorization: Bearer <AUTH_TOKEN>`
  - returns cached contacts from WhatsApp store after sync/connect
- `GET /messages?limit=100`
  - auth: `Authorization: Bearer <AUTH_TOKEN>`
  - returns recent incoming message cache from runtime memory

### Implemented runtime behavior
- unauthorized access is rejected and audited
- send attempts are logged to SQLite
- WhatsApp auth/session data is persisted to SQLite
- Go service exposes:
  - `GET /healthz`
  - `GET /readyz`

### Current implementation status
- connection manager with readiness state and reconnect backoff
- `/send` parity handler with timeout-bounded send and error mapping
- SQLite log store for send + unauthorized logs
- strict startup validation for required env (`AUTH_TOKEN`)

## Configuration

### Shared env
- `PORT` (default: `5000`)
- `AUTH_TOKEN` (required)
- `GROUP_JID` (optional fallback for requests without `userId` and `groupId`)

### Go service env
- `AUTH_DB_DSN` (default: `file:auth.db?_foreign_keys=on`)
- `LOGS_DB_DSN` (default: `file:logs.db?_foreign_keys=on`)

## Run Locally

### Go runtime (`go-service`)

Prerequisites:
- Go `1.25+` (module target in `go-service/go.mod`)
- CGO-capable toolchain for `github.com/mattn/go-sqlite3`
  - macOS: Xcode Command Line Tools
  - Debian/Ubuntu: `build-essential` (or at least `gcc`) and sqlite dev headers (`libsqlite3-dev`)

Run:

```bash
cd go-service
GOTOOLCHAIN=auto go mod tidy
GOTOOLCHAIN=auto go run ./cmd/api
```

Env loading behavior:
- `go-service` automatically loads dotenv files from `go-service/.env` and `../.env` (repository root) when present.
- OS environment variables still take precedence over values from dotenv files.

Useful checks:

```bash
cd go-service
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go vet ./...
GOTOOLCHAIN=auto go build ./...
```

## Deployment

### Local Docker deployment

1. Create env file:

```bash
cp .env.example .env
```

2. Make sure old local containers are stopped first:

```bash
docker compose -f deploy/docker-compose.yml down --remove-orphans
```

3. Build and run:

```bash
docker compose -f deploy/docker-compose.yml up --build -d
```

Shared-network setup for other Docker projects on the same host:

```bash
docker network create homelab_integration
```

Set in `.env`:

```env
INTEGRATION_NETWORK=homelab_integration
```

On the shared network, this API is reachable as `http://wa-bot-notif-api:${PORT}/send` (default `5000`).

4. Check logs:

```bash
docker compose -f deploy/docker-compose.yml logs -f api
```

5. Stop local Docker stack:

```bash
docker compose -f deploy/docker-compose.yml down
```

Notes:
- SQLite files are persisted in Docker volume `wa_bot_notif_data`.
- The API exposes `PORT` from `.env` (defaults to `5000`).
- Go service handles graceful shutdown on container stop (`SIGTERM` + `stop_grace_period`).

### Homelab topology reference

Planned production route:
- Cloudflare DNS -> VPS edge proxy -> WireGuard tunnel -> Debian homelab
- see `docs/ai/plans/go-rewrite-branch-cicd-homelab-plan.md`

## Environment template

- `.env.example` contains the baseline variables for local and Docker runs.

## AI Documentation

AI-related planning and migration docs:
- `docs/ai/README.md`

Agent entrypoint:
- `AGENTS.md`
