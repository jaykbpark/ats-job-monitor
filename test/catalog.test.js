import test from "node:test";
import assert from "node:assert/strict";

import { findCompanyById, listCompanies, searchCompanies } from "../src/catalog.js";

test("lists seeded companies with derived URLs", () => {
  const companies = listCompanies();
  const greenhouse = companies.find((company) => company.id === "greenhouse");

  assert.ok(greenhouse);
  assert.equal(greenhouse.boardUrl, "https://job-boards.greenhouse.io/greenhouse");
  assert.equal(greenhouse.apiUrl, "https://boards-api.greenhouse.io/v1/boards/greenhouse/jobs");
});

test("searches seeded companies by name and provider", () => {
  assert.equal(searchCompanies("ash").length, 1);
  assert.equal(searchCompanies("lever")[0].provider, "lever");
});

test("finds a seeded company by id", () => {
  const company = findCompanyById("ashby");

  assert.equal(company.name, "Ashby");
  assert.equal(company.identifierKind, "job_board_name");
});
