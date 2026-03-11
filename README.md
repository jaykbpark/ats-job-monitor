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

## Why This Approach

- Less brittle than Playwright or DOM scraping
- Lower cost than browser-based polling
- Easier to maintain because job data is already structured

## Next Steps

1. Define a simple config format for target companies and filters.
2. Implement fetchers for Lever, Greenhouse, and Ashby.
3. Add a diff step against stored seen jobs.
4. Send alerts through email, Slack, or SMS.
