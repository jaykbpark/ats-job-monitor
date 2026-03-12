# Architecture

## Stack

- Frontend: React + TypeScript + Vite
- Backend API: Go
- Database: SQLite
- Background jobs: Go worker process

## Why This Shape

The application has two very different concerns:

- a user-facing dashboard for configuring watches and reviewing results
- backend polling and diffing work against ATS APIs

React is the better fit for the dashboard experience. Go is a good fit for the backend because the work is mostly IO-bound API polling, normalization, scheduling, and persistence without requiring a heavy framework.

## Components

### Frontend

The frontend will manage:

- company selection or ATS URL input
- watch filters
- run history
- notification preferences

It should talk to the backend only through JSON APIs.

### API

The Go API owns:

- creating and updating watch targets
- exposing run status and matched jobs
- triggering manual syncs
- serving the curated company catalog

Current skeleton endpoints:

- `GET /health`
- `GET /api/companies`
- `GET /api/watch-targets`
- `POST /api/watch-targets`
- `POST /api/watch-targets/{id}/sync`
- `GET /api/watch-targets/{id}/jobs`
- `GET /api/watch-targets/{id}/sync-runs`

### Worker

The worker is a separate Go process that:

- reads active watch targets from SQLite
- fetches jobs from ATS providers on a schedule
- normalizes jobs into a shared shape
- diffs jobs against stored state
- records notifications to send

### Provider Adapters

Each ATS gets its own adapter:

- Lever
- Greenhouse
- Ashby

Every adapter should return the same internal job shape so filtering and diffing stay provider-agnostic.

Greenhouse and Lever are wired end to end.

## Data Model

The first versioned schema lives in [db/migrations/0001_init.sql](./db/migrations/0001_init.sql).

Core tables:

- `watch_targets`: what to monitor
- `jobs`: normalized jobs per target
- `sync_runs`: execution history and failures
- `notifications`: delivered or pending alerts

For the first version, filters should be stored as JSON on `watch_targets`. That keeps the schema simple until we learn which filters deserve first-class columns.

## Runtime Flow

### Add Watch

1. User selects a known company or pastes a direct ATS board URL.
2. Backend resolves the ATS provider and canonical board key.
3. Backend stores a `watch_target` row.

### Sync Watch

1. Worker reads active targets due for polling.
2. Worker fetches jobs from the provider API.
3. Worker normalizes jobs into a shared model.
4. Worker applies filters.
5. Worker upserts current jobs and marks unseen jobs inactive.
6. Worker records a `sync_run`.
7. Worker writes `notifications` for newly matching jobs.

## Repo Direction

The repo will move toward this layout:

- `cmd/`: Go executables such as API server, worker, and developer CLI
- `internal/api/`: HTTP server and handlers
- `internal/ats/`: ATS URL resolution and provider-specific logic
- `internal/catalog/`: curated company catalog
- `internal/store/`: database access and migration runner
- `db/migrations/`: versioned SQLite migrations
- `frontend/`: React application

The current repo includes the first runnable API skeleton on top of the SQLite store.
