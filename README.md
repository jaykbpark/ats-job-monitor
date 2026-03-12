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

Run the API server:

```bash
go run ./cmd/api
```

The API currently serves:

- `GET /health`
- `GET /api/companies`
- `GET /api/watch-targets`
- `POST /api/watch-targets`
