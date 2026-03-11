import test from "node:test";
import assert from "node:assert/strict";

import { parseAtsIdentifier } from "../src/ats.js";

test("parses Lever board URLs", () => {
  const match = parseAtsIdentifier("https://jobs.lever.co/acme/jobs/123");

  assert.deepEqual(
    {
      provider: match.provider,
      identifierKind: match.identifierKind,
      boardKey: match.boardKey,
      sourceType: match.sourceType
    },
    {
      provider: "lever",
      identifierKind: "site",
      boardKey: "acme",
      sourceType: "board-url"
    }
  );
});

test("parses Lever API URLs", () => {
  const match = parseAtsIdentifier("https://api.lever.co/v0/postings/acme?mode=json");

  assert.equal(match.provider, "lever");
  assert.equal(match.boardKey, "acme");
  assert.equal(match.sourceType, "api-url");
});

test("parses Greenhouse board URLs", () => {
  const match = parseAtsIdentifier("https://job-boards.greenhouse.io/acme/jobs/77");

  assert.equal(match.provider, "greenhouse");
  assert.equal(match.identifierKind, "board_token");
  assert.equal(match.boardKey, "acme");
});

test("parses Greenhouse embed URLs", () => {
  const match = parseAtsIdentifier("https://boards.greenhouse.io/embed/job_board?for=acme");

  assert.equal(match.provider, "greenhouse");
  assert.equal(match.boardKey, "acme");
  assert.equal(match.sourceType, "embed-url");
});

test("parses Ashby board URLs", () => {
  const match = parseAtsIdentifier("https://jobs.ashbyhq.com/Acme/jobs/123");

  assert.equal(match.provider, "ashby");
  assert.equal(match.identifierKind, "job_board_name");
  assert.equal(match.boardKey, "Acme");
});

test("parses Ashby API URLs", () => {
  const match = parseAtsIdentifier("https://api.ashbyhq.com/posting-api/job-board/Acme");

  assert.equal(match.provider, "ashby");
  assert.equal(match.boardKey, "Acme");
  assert.equal(match.sourceType, "api-url");
});

test("returns null for unsupported URLs", () => {
  assert.equal(parseAtsIdentifier("https://careers.example.com/jobs"), null);
});
