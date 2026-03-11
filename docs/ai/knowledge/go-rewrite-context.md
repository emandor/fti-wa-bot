# Go Rewrite Context: WhatsApp Notification Bot

## Purpose

Rewrite the service from TypeScript/Bun to Go while preserving external behavior and improving runtime reliability.

Current business intent:
- accept trusted `POST /send`
- send text to WhatsApp group/target
- persist send and unauthorized audit logs
- persist WA auth/session state across restarts

## Behavior Contract to Preserve

### API behavior
- endpoint: `POST /send`
- payload:
  - `message` (required)
  - `groupId` (optional; fallback to `GROUP_JID`)
- auth: Bearer token (`AUTH_TOKEN`)
- side effects:
  - send WhatsApp text message
  - insert logs for send or unauthorized attempt

### Runtime behavior
- initialize WhatsApp before API starts serving
- print pairing QR when no session exists
- maintain send capability when connected
- persist auth keys/creds in SQLite

## Known Defects to Eliminate

1. stale client reference after reconnect
2. recursive reconnect without bounded backoff
3. startup auth bypass risk from empty token
4. unguarded send error path
5. duplicate event handlers
6. auth/key persistence race on crash
7. high-noise logging on hot event paths

## Recommended Go Stack

Primary WA library:
- `go.mau.fi/whatsmeow`

Legacy library to avoid for new builds:
- `Rhymen/go-whatsapp` (maintenance/version drift risk)

Supporting stack:
- HTTP: `chi`/`gin`/`net/http`
- config: `caarlos0/env` or `envconfig`
- logging: `zerolog` or `zap`
- SQLite: `modernc.org/sqlite` or `mattn/go-sqlite3`
- retry/backoff: `cenkalti/backoff/v4`

## Go Runtime Design Requirements

1. single connection manager
- exactly one active client handle
- thread-safe client access for send path
- atomic client replacement on reconnect

2. controlled reconnect
- exponential backoff with cap + jitter
- bounded retry policy and unhealthy state handling
- one event registration lifecycle per client instance

3. strict startup validation
- fail fast if `AUTH_TOKEN` is empty
- fail fast if required defaults are missing

4. safe send path
- timeout-bounded send context
- stable error mapping (`401`, `502/503`)
- no process crash on send failure

5. durable auth persistence
- persist key/cred updates safely and promptly
- transactional writes where possible

6. observability
- structured logs
- `/healthz` and `/readyz`

## Behavior Mapping: TS -> Go

- `initWhatsApp()` -> `ConnectionManager.Start(ctx)`
- event callbacks -> typed whatsmeow event dispatch
- `createSendModule(sock)` -> handler using `ConnectionManager.CurrentClient()`
- sqlite services -> repository interfaces (`AuthStore`, `LogStore`)

## Rewrite Acceptance Checklist

- sending still works after disconnect/reconnect
- startup fails when token is missing/empty
- unauthorized returns `401`
- send failures are controlled and non-fatal
- no duplicate event side effects after repeated reconnects
- session survives normal restart without re-pairing
- readiness reflects actual WA connectivity
