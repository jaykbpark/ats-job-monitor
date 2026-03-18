# Signal Review 2026-03-18

## Scope

- Source: live `atsctl audit-signals --sample 6 --format json`
- Providers covered: Greenhouse, Lever, Ashby
- Targets audited: 18
- Sampled jobs reviewed: 98
- Automatic audit result: 5 failures, 433 automatic checks

This note captures the follow-up fixes that should be implemented before trusting the hard-filter engine too much.

## Summary

The current system is good enough to fetch and normalize jobs across providers, but it is still too eager in a few important areas:

- seniority is overfitting to description text
- Greenhouse employment type is frequently recoverable but currently missed
- the engineering-job audit sampler is still too broad and mixes in non-software roles
- the experience parser still misses some explicit numeric cases

The good news is that remote detection looked materially better in this pass. I did not find a strong backlog of remote misses in the 98 sampled jobs.

## Status After Fixes

The high-confidence fixes in this note have now been implemented:

- Greenhouse employment type recovery from structured metadata and anchored content
- tighter engineering-job audit sampling
- title-first seniority derivation with explicit `lead` handling
- more robust experience parsing from provider payload text

Post-fix validation on March 18, 2026:

- full live audit: 18 successful targets, 52 sampled jobs, 237 automatic checks, 0 automatic failures
- sample-6 live audit: 18 successful targets, 97 sampled jobs, 450 automatic checks, 0 automatic failures

Manual spot checks after the fixes covered:

- Greenhouse: Figma `Manager, Software Engineering - Billing`
- Lever: Level AI `Lead Software Engineer`
- Lever: Tutor Intelligence `Robot Perception Engineer`
- Ashby: OpenAI `Senior Software Engineer, Identity Platform`
- Ashby: Ramp `Mobile Engineer, Android`

Current interpretation:

- the derivation layer is now stable enough to start promoting reviewed live cases into checked-in fixtures
- remaining gaps are mostly acceptable `unknown` outcomes where the provider does not expose a strong structured signal

## High-Confidence Fixes

### 1. Make seniority title-first

Current behavior:
- body text can override the title and produce clearly wrong levels

Observed examples:
- Figma `Manager, Software Engineering - Billing` -> derived `staff`
- Vercel `Engineering Manager, CDN` -> derived `principal`
- Datadog `Director, Engineering Operations` -> derived `staff`
- Instrumentl `Senior Full Stack Software Engineer` -> derived `director`

Root cause:
- `deriveSeniority` searches the combined `SearchText`, so body phrases outrank or contaminate title signals

Required fix:
- derive seniority from title first
- only fall back to body text when the title is truly neutral
- title matches should be final unless we explicitly decide otherwise

### 2. Add phrase-level guards for seniority false positives

Current false-positive triggers found in live jobs:
- `senior engineers`
- `junior engineers`
- `staff work`
- `board of director approval`
- `senior or staff level`
- `hiring at multiple levels from mid level to principal`

Observed examples:
- Roblox early-career software role -> derived `senior` because the body mentions mentorship from senior engineers
- Level AI `Lead Software Engineer` -> derived `junior` because the body says mentor junior engineers
- Reveal Tech `AI Engineer (Agentic/LLMs)` -> derived `staff` because the body says manual staff work
- Scale AI infra roles -> derived `director` because compensation text mentions board of director approval

Required fix:
- strip or ignore known noise phrases
- do not treat mentions of other people or compensation/legal language as the candidate's level

### 3. Add explicit handling for `lead`

Current behavior:
- `Lead Software Engineer` is not recognized as a positive title-level signal
- then body text can drag the result to something nonsensical like `junior`

Observed example:
- Level AI `Lead Software Engineer` -> derived `junior`

Required fix:
- decide whether `lead` becomes its own normalized level or maps to `senior`
- whichever option we choose, title should win before body fallback

### 4. Fix the explicit experience parser miss

Current behavior:
- we still missed at least one obvious numeric experience requirement

Observed example:
- Tutor Intelligence `Robot Perception Engineer`
- evidence clearly contains `3 years of experience`
- derived experience stayed `unknown`

Required fix:
- inspect `deriveExperience` against the real Tutor payload
- add a regression test from that exact example
- verify whether the miss is regex shape, escaping, or raw-source preprocessing

### 5. Recover employment type from Greenhouse metadata/content

Current behavior:
- many Greenhouse jobs are still `normalizedEmploymentType=unknown`
- but the provider content or metadata often contains `full time`

Observed examples:
- Figma jobs
- Datadog jobs
- Roblox jobs
- Scale AI jobs

Required fix:
- extract employment type from high-confidence Greenhouse sources first:
  - metadata fields like `Time Type`
  - repeated structured field blocks embedded in the payload
  - clearly anchored role text like `this is a full time role`

### 6. Avoid employment-type false positives from disclaimers

Current behavior:
- naive content scanning would misclassify some Stripe roles as `internship`

Observed examples:
- Stripe `Backend/API Engineer, Money as a Service`
- Stripe `Backend / API Engineer, Payouts`

Why:
- the posting says things like `if you are an intern ... apply elsewhere`
- that is not the role's employment type

Required fix:
- prefer structured metadata over free-text scanning
- if we scan prose, require anchored phrases about the role itself, not a generic disclaimer

### 7. Tighten the engineering audit sampler

Current behavior:
- the audit sample still includes technical-adjacent roles that are not really the software-engineering cohort we care about

Questionable sampled titles:
- Figma `IT Engineer (London, United Kingdom)`
- Vercel `Developer Success Engineer`
- Datadog `Enablement Systems Engineer`
- Scale AI `AI Applications Ops Lead, GPS`
- OpenAI `Technical Program Manager, Compute Infrastructure`
- Zapier `Pre-Sales Solutions Architect`
- Snowflake `Solution Engineer`
- Snowflake `Senior Solution Engineer`

Required fix:
- narrow the sampler toward product/platform/data/security/backend/frontend/mobile/infrastructure engineering
- explicitly exclude:
  - solutions architecture
  - program management
  - sales engineering / solution engineering
  - enablement
  - IT / internal ops
- keep a separate bucket later if we want to audit broader technical roles

## Medium-Confidence Fixes

### 8. Revisit whether `Systems Engineer` should count as a target role

Observed example:
- Substack `Systems Engineer`

This one is ambiguous. It may be a legitimate infrastructure role or it may be closer to IT/ops depending on the org.

Required decision:
- decide whether the product should treat `systems engineer` as in-scope for engineering monitoring or as a separate category

### 9. Consider extracting remote eligibility from Greenhouse office data later

This pass did not show a strong backlog of missed remote cases.

That said, Greenhouse often exposes remote office names inside office arrays, and we should probably use that later when we build stricter matching.

Current recommendation:
- do not prioritize this ahead of seniority, employment type, sampler quality, and experience parsing

## Proposed Implementation Order

1. Fix seniority derivation:
title-first, phrase guards, and `lead` handling

2. Fix experience parsing:
Tutor regression plus any regex/input normalization issues

3. Add Greenhouse employment-type extraction:
structured fields first, anchored prose second

4. Tighten the audit sampler:
exclude non-software technical roles from the default engineering review set

5. Re-run the large live audit and update this note with the new findings

## Regression Cases To Save As Fixtures

These should become checked-in fixtures after the fixes land:

- Figma `Manager, Software Engineering - Billing`
- Vercel `Engineering Manager, CDN`
- Datadog `Director, Engineering Operations`
- Instrumentl `Senior Full Stack Software Engineer`
- Tutor Intelligence `Robot Perception Engineer`
- Level AI `Lead Software Engineer`
- Stripe `Backend/API Engineer, Money as a Service`
- Scale AI `AI Infrastructure Engineer, Core Infrastructure`
- Reveal Tech `AI Engineer (Agentic/LLMs)`
