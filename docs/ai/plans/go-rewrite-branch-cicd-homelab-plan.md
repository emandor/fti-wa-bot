# Go Rewrite Plan: Branch, CI/CD, and Homelab Deployment

## Goal

Build Go + `whatsmeow` rewrite safely, then deploy on Debian homelab with Docker, exposed through DO VPS + WireGuard + Cloudflare DNS.

## Branch Strategy

Use a dedicated long-lived branch:
- `rewrite/go-whatsmeow`

Reason:
- keep `main` stable
- run side-by-side parity validation
- reduce rollback risk

Milestone PR sequence:
1. bootstrap Go service + health endpoints
2. whatsmeow connection manager + auth persistence
3. `/send` parity + strict auth behavior
4. Docker + CI pipeline
5. homelab deployment runbook + cutover plan

## Suggested Repository Layout

```text
go-service/
  cmd/api/main.go
  internal/
    config/
    http/
    wa/
    storage/
    observability/
  migrations/
  Dockerfile
  go.mod
deploy/
  docker-compose.yml
  .env.example
  scripts/
docs/runbooks/
```

## Parity and Reliability Targets

Must preserve:
- `POST /send` contract
- bearer token auth
- send + unauthorized logs
- WA session persistence

Must improve:
- no stale client after reconnect
- bounded reconnect/backoff behavior
- fail-fast config validation
- consistent error semantics

## CI Plan (GitHub Actions)

On PR/push to rewrite branch:
1. `go fmt`
2. `go vet`
3. `go test ./...`
4. `go test -race` (critical packages)
5. Docker image build
6. image vulnerability scan (Trivy)

On release/tag:
1. build + push image (GHCR/Docker Hub)
2. tag scheme: `vX.Y.Z` and `sha-<short>`

## CD Plan (Debian Homelab)

Preferred model: pull-based deployment on homelab host.

- homelab runs Docker + Compose
- deploy script pulls approved image tag and restarts service

Options:
- A: GitHub Action deploy over SSH through WireGuard
- B: self-hosted runner in homelab (recommended long-term)

## Network Topology

1. Cloudflare DNS -> VPS public IP
2. VPS reverse proxy (Caddy/Traefik/Nginx)
3. proxy forwards via WireGuard tunnel to homelab private IP
4. homelab Docker serves API internally

Flow:
`Client -> Cloudflare -> VPS edge -> WireGuard -> Homelab container`

## Security Baseline

Mandatory:
- `AUTH_TOKEN` non-empty, strong random value
- secrets not stored in git
- homelab firewall restricted to WireGuard ingress path
- TLS at edge (Cloudflare Full Strict)
- request rate limiting at app and optionally proxy layer

Recommended:
- Cloudflare WAF
- IP allowlist for trusted callers
- structured audit logs and request IDs
- daily backup of auth/session data

## Rollout and Cutover

Phase A: staging parity verification
- compare Go behavior against TS contract

Phase B: shadow validation
- controlled sends and reconnect observations for at least 72h

Phase C: cutover
- switch proxy route to Go service
- keep TS path available for immediate rollback

Phase D: rollback
- rollback on send success drop, reconnect storm, or persistent readiness failures

## Acceptance Criteria

- `/send` parity confirmed
- reconnect does not break sending
- readiness accurately reflects WA connectivity
- CI pipeline fully green
- image deploys to Debian with persistent storage
- Cloudflare -> VPS -> WG -> homelab path validated
- rollback procedure documented and tested
