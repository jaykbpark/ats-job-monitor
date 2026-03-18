# ATS Job Monitor

Small project for tracking ATS-backed job boards without browser scraping.

## Goal

Given a careers URL or ATS company slug, poll the backing jobs API, filter for roles that match my criteria, and notify me only when new matching roles appear.

## Initial Scope

- Detect common ATS providers like Lever, Greenhouse, and Ashby
- Fetch structured job listings from public ATS endpoints
- Filter jobs by keywords, location, and level
- Store previously seen job IDs
- Send a simple notification when a new matching role appears

## Chosen Stack

- Frontend: React + TypeScript + Vite
- Backend: Go
- Database: SQLite
- Worker: Go background process

## Why This Approach

- Less brittle than Playwright or DOM scraping
- Lower cost than browser-based polling
- Easier to maintain because job data is already structured

## Next Steps

1. Define a simple config format for target companies and filters.
2. Implement fetchers for Lever, Greenhouse, and Ashby.
3. Add a diff step against stored seen jobs.
4. Send alerts through email, Slack, or SMS.

## Current Spike

The current Go spike focuses on normalizing ATS inputs into a stable identifier:

- Lever: `provider=lever`, `identifierKind=site`
- Greenhouse: `provider=greenhouse`, `identifierKind=board_token`
- Ashby: `provider=ashby`, `identifierKind=job_board_name`

This is the foundation for a hybrid UX:

- Let users pick from a curated company list when we already know the ATS identifier
- Let users paste a direct ATS job board URL when the company is not in our catalog yet

## Usage

The repo now contains:

- a Go ATS identifier parser
- an embedded seed catalog
- a runnable Go API skeleton
- Greenhouse, Lever, and Ashby sync paths from API trigger to stored jobs
- a typed hard-filter schema for watch target preferences
- persisted derived job signals for matching
- a live signal-audit CLI for provider-backed spot checks
- a developer CLI for testing ATS resolution
- a SQLite store package with versioned migrations

Read the system design in [ARCHITECTURE.md](./ARCHITECTURE.md).

## Usage

```bash
go test ./...
```

Detect an ATS board from a direct provider URL:

```bash
go run ./cmd/atsctl detect https://job-boards.greenhouse.io/greenhouse
```

List the seeded catalog entries:

```bash
go run ./cmd/atsctl companies
```

Apply SQLite migrations to a local database file:

```bash
go run ./cmd/atsctl migrate ./ats-job-monitor.db
```

Run the live signal audit against the embedded board manifest:

```bash
go run ./cmd/atsctl audit-signals
```

The audit uses content-enabled Greenhouse fetches so it can verify that description/evidence text is present and that explicit years-of-experience in the job content match the derived signals.

Filter the live audit to one provider or switch to JSON output:

```bash
go run ./cmd/atsctl audit-signals --provider ashby --sample 2
go run ./cmd/atsctl audit-signals --format json --limit 5
```

Run the API server:

```bash
go run ./cmd/api
```

The API currently serves:

- `GET /health`
- `GET /api/companies`
- `GET /api/watch-targets`
- `POST /api/watch-targets`
- `POST /api/watch-targets/{id}/sync`
- `GET /api/watch-targets/{id}/jobs`
- `GET /api/watch-targets/{id}/sync-runs`
- `GET /api/watch-targets/{id}/notifications`

`POST /api/watch-targets` accepts an optional `notificationEmail` field. When present, newly matched jobs for that target are queued for `email` delivery instead of the default `inbox` channel.

Deliver pending notifications from the worker CLI:

```bash
go run ./cmd/atsctl deliver-notifications ./ats-job-monitor.db
```

For email delivery, configure SMTP with environment variables:

```bash
export ATS_JOB_MONITOR_SMTP_HOST="smtp.example.com"
export ATS_JOB_MONITOR_SMTP_PORT="587"
export ATS_JOB_MONITOR_SMTP_USERNAME="smtp-user"
export ATS_JOB_MONITOR_SMTP_PASSWORD="smtp-password"
export ATS_JOB_MONITOR_SMTP_FROM="alerts@example.com"
```

The SMTP sink currently targets standard SMTP/STARTTLS style endpoints such as port `587`.
