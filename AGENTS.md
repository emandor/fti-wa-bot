# AGENTS.md

## AI Entrypoint

This root file is the AI entrypoint. Canonical AI documentation is organized under:

- `docs/ai/README.md`
- `docs/ai/agents/agent-strategy.md`
- `docs/ai/skills/skill-matrix.md`
- `docs/ai/knowledge/project-knowledge.md`
- `docs/ai/knowledge/go-rewrite-context.md`
- `docs/ai/plans/go-rewrite-branch-cicd-homelab-plan.md`

## Quick Agent Policy

- Use `explore` for internal code and behavior mapping.
- Use `librarian` for external references (`whatsmeow`, Docker, WireGuard, Cloudflare).
- Use `oracle` for architecture and high-risk decisions.
- Use `momus` to review plans before major implementation.
- Use `metis` when requirements are ambiguous.

## Default Execution Order

1. `explore` -> map current behavior/parity
2. `librarian` -> collect implementation references
3. `oracle` -> validate architecture/risk decisions
4. implement
5. `momus` -> plan/risk completeness check

## Project Objective

- Rewrite WhatsApp notification service from TypeScript/Bun to Go/whatsmeow
- Preserve API behavior parity
- Improve reliability and deploy to Debian homelab via Docker
