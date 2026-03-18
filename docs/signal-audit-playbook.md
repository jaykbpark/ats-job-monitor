# Signal Audit Playbook

Use this when validating derivation and matching changes against live ATS data.

This file is the repeatable skill/playbook for signal-audit work in this repo.

## Goal

Keep the loop tight:

1. Run a live audit.
2. Inspect the sampled jobs manually.
3. Make one atomic fix.
4. Rerun the audit.
5. Promote stable examples into fixtures only after the signal is behaving.

## Commands

Run the live audit:

```bash
go run ./cmd/atsctl audit-signals --sample 6 --format json
```

Inspect the JSON summary and failures:

```bash
go run ./cmd/atsctl audit-signals --sample 6 --format json > /tmp/signal-audit.json
jq '.summary' /tmp/signal-audit.json
jq '.targets[].jobs[] | select(.checks[]?.status == "fail")' /tmp/signal-audit.json
```

Run focused tests after each fix:

```bash
go test ./internal/signals
go test ./internal/providers
go test ./internal/audit
go test ./...
```

Mine suspicious cases even when the automatic checks are green:

```bash
go run ./cmd/atsctl audit-signals --format json > /tmp/signal-audit.json
python3 - <<'PY'
import json, re
from pathlib import Path
obj = json.loads(Path('/tmp/signal-audit.json').read_text())
for target in obj["targets"]:
    for job in target["jobs"]:
        evidence = (job.get("evidenceText") or "").lower()
        signals = job.get("signals") or {}
        if signals.get("normalizedEmploymentType") == "unknown" and re.search(r"\b(full[ -]?time|part[ -]?time|contract|intern(ship)?)\b", evidence):
            print("employment candidate", target["name"], job["title"])
        if not signals.get("isRemote") and re.search(r"\b(remote|distributed|work from home|anywhere)\b", evidence):
            print("remote candidate", target["name"], job["title"])
        if signals.get("experienceConfidence") == "unknown" and re.search(r"\b(?:at least|minimum of)?\s*\d{1,2}(?:\+|\s*(?:-|to)\s*\d{1,2})?\s*(?:years?|yrs?)\b", evidence):
            print("experience candidate", target["name"], job["title"])
PY
```

## Manual Review Heuristics

Treat these as hard rules first.

- Seniority should be title-first.
- Ignore body text that mentions other people or teams, such as `senior engineers`, `junior engineers`, or `chief of staff`.
- `lead` in a title should map to `senior` unless a later product decision says otherwise.
- Experience should be numeric-only unless the posting explicitly says `3+ years`, `at least 5 years`, or similar.
- Prefer provider fields over prose when employment type or remote status is available.
- For Greenhouse, prefer structured metadata and content blocks before free-text inference.
- Treat `unknown` as a failure for hard filters unless `allowUnknownExperience` is set.

## What To Fix First

When a live audit shows failures, classify each one before coding:

- Derivation bug: a field is present but parsed incorrectly.
- Missing data path: the provider exposes useful content, but we are not reading it.
- Sampler bug: the audit sample includes roles outside the intended engineering cohort.
- Fixture gap: the logic is correct, but we do not have a regression case yet.

Do not mix those in one commit.

## Fixture Promotion Rules

Promote a live sample into a fixture only when all of this is true:

- The input is stable enough to reproduce.
- The expected output is explicit, not inferred.
- The case failed or was manually verified at least once in live data.
- The fixture represents one specific rule, not a grab bag of behaviors.

Fixture format should include:

- provider
- board key
- job title
- the raw evidence snippet
- the expected derived signals

If a case depends on broad text that may drift frequently, keep it as a live audit note instead of a fixture.

## Recommended Loop

1. Run the live audit.
2. Review the sampled jobs and list only the concrete failures or misses.
3. Choose one fix scope and keep it atomic.
4. Add or update unit tests for the exact rule.
5. Rerun the live audit.
6. If the case now passes consistently, add a fixture.
7. Repeat until the remaining failures are intentional or accepted tradeoffs.

## Source Of Truth

- Current findings: [signal-review-2026-03-18.md](signal-review-2026-03-18.md)
- This playbook: the repeatable workflow for future passes
