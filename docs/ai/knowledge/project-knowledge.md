# Project Knowledge

## Status

Go rewrite is **complete**. The service is a pure Go project (whatsmeow), deployed via Docker Compose on Debian homelab. TypeScript/Bun implementation has been removed.

## Product Summary

Authenticated HTTP service that sends WhatsApp text messages via a connected WhatsApp account.

External contract:
- `POST /send` — send a message, auth: Bearer token
- `GET /contacts` — list synced WA contacts
- `GET /messages?limit=100` — recent runtime message cache
- `GET /healthz` / `GET /readyz` — health and readiness probes

## Architecture

```
cmd/api/main.go         — entry point, server bootstrap
internal/config/        — env config loading and validation
internal/httpapi/       — HTTP handlers (net/http ServeMux)
internal/wa/            — WhatsApp connection manager (whatsmeow)
internal/storage/       — SQLite audit log store
deploy/                 — Docker Compose
```

## Key Design Decisions

- Single `Manager` owns the whatsmeow client; thread-safe via `sync.RWMutex`
- Exponential backoff with jitter on reconnect (1s base, 60s cap)
- `MessageBuffer` is a bounded (500-entry) FIFO for incoming message cache
- `LogStore.StartRetention` purges logs older than 30 days (daily goroutine)
- `AUTH_TOKEN` is validated non-empty at startup (fail-fast)
- zerolog for structured logging; `LOG_LEVEL` env controls verbosity

## Baseline Environment Variables

| Variable | Required | Default |
|----------|----------|---------|
| `AUTH_TOKEN` | ✅ | — |
| `PORT` | — | `5000` |
| `GROUP_JID` | — | — |
| `AUTH_DB_DSN` | — | `file:auth.db?_foreign_keys=on` |
| `LOGS_DB_DSN` | — | `file:logs.db?_foreign_keys=on` |
| `LOG_LEVEL` | — | `info` |

## Deployment Topology

`Cloudflare DNS → DO VPS edge proxy → WireGuard tunnel → Debian homelab (Docker Compose)`

See `docs/deploy.md` for the full deployment guide.

## CI

GitHub Actions (`.github/workflows/ci.yml`):
- `go fmt` check
- `go vet`
- `go test ./...`
- `go test -race`
- `go build`
- Docker image build

## Canonical References

- `docs/deploy.md` — deployment runbook
- `docs/ai/knowledge/go-rewrite-context.md` — original rewrite context (historical)
- `docs/ai/plans/go-rewrite-branch-cicd-homelab-plan.md` — original infra plan (historical)
- `docs/ai/agents/agent-strategy.md` — agent workflow
- `docs/ai/skills/skill-matrix.md` — skill matrix
