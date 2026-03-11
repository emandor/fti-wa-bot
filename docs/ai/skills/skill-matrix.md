# Skill Matrix

## Existing High-Value Skills

- `go-engineering`
  - Build service architecture, interfaces, context-driven handlers, safe error boundaries.

- `unit-test-engineering`
  - Build parity and regression tests for reconnect/send/auth flows.

- `javascript-engineering`
  - Extract legacy TS behavior contracts for migration parity.

- `git-master`
  - Atomic branch workflow, history checks, and clean merge sequencing.

- `writing`
  - Runbooks, deployment playbooks, incident docs.

## Recommended Custom Skills

- `whatsmeow-engineering`
  - Connection lifecycle, event handlers, auth/key persistence, reconnect backoff.

- `github-actions-cicd`
  - Go CI, Docker image release, deployment pipelines.

- `docker-debian-ops`
  - Dockerfile hardening, compose runtime, persistent volume strategy.

- `wireguard-homelab-networking`
  - VPS hub setup, routing/firewall model, troubleshooting MTU/DNS issues.

- `cloudflare-edge-routing`
  - DNS/proxy mode, TLS mode, origin cert and edge routing patterns.

- `service-security-baseline`
  - Token policy, secret handling, auth semantics, audit trails.

- `go-observability`
  - Structured logs, readiness/liveness endpoints, service telemetry.

## Skill Profiles by Workstream

### Rewrite implementation
`["go-engineering", "unit-test-engineering", "javascript-engineering"]`

### CI/CD setup
`["go-engineering", "github-actions-cicd", "docker-debian-ops"]`

### Network + edge rollout
`["wireguard-homelab-networking", "cloudflare-edge-routing", "service-security-baseline"]`

### Documentation and operations
`["writing", "service-security-baseline"]`

## Completion Gate

Before merging:
- parity tests pass
- reconnect behavior validated by tests
- deployment reproducible from docs
- rollback path documented and tested
