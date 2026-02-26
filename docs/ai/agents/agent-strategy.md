# Agent Strategy

## Core Agent Set

### `explore`
- Use for codebase mapping, behavior parity tracing, and change impact discovery.
- Expected output: file-path evidence and concrete findings.

### `librarian`
- Use for external references and implementation guidance.
- Primary topics: `whatsmeow`, Docker hardening, WireGuard, Cloudflare.

### `oracle`
- Use for high-risk architecture decisions.
- Typical topics: reconnect model, persistence strategy, rollback rules, incident failure modes.

### `momus`
- Use for plan quality checks before substantial implementation.
- Expected output: missing risks, unclear assumptions, acceptance criteria gaps.

### `metis` (as needed)
- Use when requirements are ambiguous or conflicting.

## Decision Matrix

- "Where is this behavior implemented?" -> `explore`
- "How do we implement this with library X?" -> `librarian`
- "Which architecture option should we choose?" -> `oracle`
- "Is this plan complete enough to execute?" -> `momus`
- "Requirement unclear" -> `metis`

## Recommended Workflows

### Rewrite Feature Workflow
1. `explore` current TS behavior
2. `librarian` gather Go/whatsmeow references
3. `oracle` validate architecture choice
4. implement
5. `momus` review completeness

### CI/CD Workflow
1. `librarian` gather best-practice pipelines
2. `explore` align with repo constraints
3. `oracle` verify risk + rollback design
4. implement and verify

### Networking Workflow (WireGuard + Cloudflare)
1. `librarian` source-backed topology references
2. `oracle` security and failure-path validation
3. document runbook, then implement

## Prompt Contract for Subagents

Always include:
1. TASK
2. EXPECTED OUTCOME
3. REQUIRED TOOLS
4. MUST DO
5. MUST NOT DO
6. CONTEXT

## Quality Gates

Agent output is accepted only when:
- evidence is repository/source-backed
- assumptions and risks are explicit
- recommendations are actionable in this project
