# AI Documentation Index

This directory contains AI-facing guidance and planning artifacts.

## Structure

```text
docs/
├── deploy.md                          ← deployment runbook (start here for ops)
└── ai/
    ├── README.md                      ← this file
    ├── agents/agent-strategy.md
    ├── skills/skill-matrix.md
    ├── knowledge/
    │   ├── project-knowledge.md       ← canonical project state (start here for context)
    │   └── go-rewrite-context.md      ← rewrite rationale (historical reference)
    └── plans/
        └── go-rewrite-branch-cicd-homelab-plan.md  ← infra plan (historical reference)
```

## What to Read First

1. `docs/ai/knowledge/project-knowledge.md` — current architecture and design decisions
2. `docs/deploy.md` — deployment topology and runbook
3. `docs/ai/plans/go-rewrite-branch-cicd-homelab-plan.md` — homelab infra context

## Operational Rule

The root `AGENTS.md` remains the entrypoint for agent bootstrapping. Files in `docs/ai/` are the canonical detailed references.
