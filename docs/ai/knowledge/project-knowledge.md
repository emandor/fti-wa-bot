# Project Knowledge

## Product Summary

The service provides an authenticated HTTP endpoint to send WhatsApp text messages.

Current external contract:
- Endpoint: `POST /send`
- Request: `message` (required), `groupId` (optional)
- Auth: Bearer token (`AUTH_TOKEN`)
- Side effects: send WA message and store logs

## Key Risks Already Identified

- stale socket/client reference after reconnect
- recursive reconnect loop without bounded backoff
- empty token startup risk
- missing guarded send failure handling
- duplicated event listener registration
- auth/key persistence timing gaps

## Migration Invariants

- Preserve API contract during rewrite
- Preserve session persistence across restarts
- Preserve logging side effects (schema can improve)
- Sending must remain functional after reconnect events

## Target Go Architecture

- `ConnectionManager`: single active whatsmeow client, safe reconnect, state exposure
- `SendService`: input validation, timeout-bounded send, stable error mapping
- `AuthStore`: durable credentials/keys persistence
- `LogStore`: send + security logs
- API endpoints: `/send`, `/healthz`, `/readyz`

## Deployment Topology (Planned)

`Cloudflare DNS -> DO VPS edge proxy -> WireGuard tunnel -> Debian homelab (Docker)`

## Baseline Environment Variables

- `PORT`
- `AUTH_TOKEN` (required, non-empty)
- `GROUP_JID` (required if no per-request override is intended)

## CI/CD Baseline

- `go fmt`
- `go vet`
- `go test ./...`
- `go test -race` (critical packages)
- Docker build + image vulnerability scan

## Rollback Model

- Keep TypeScript implementation available during soak period
- Route switch at proxy/DNS layer for fast rollback
- rollback triggers: send success drop, reconnect storm, persistent readiness failure

## Canonical References

- `docs/ai/knowledge/go-rewrite-context.md`
- `docs/ai/plans/go-rewrite-branch-cicd-homelab-plan.md`
- `docs/ai/agents/agent-strategy.md`
- `docs/ai/skills/skill-matrix.md`
