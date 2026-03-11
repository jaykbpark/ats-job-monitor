import companies from "../data/companies.json" with { type: "json" };
import { buildApiUrl, buildBoardUrl, getIdentifierKind } from "./ats.js";

export function listCompanies() {
  return companies.map(enrichCompany);
}

export function searchCompanies(query) {
  const normalizedQuery = typeof query === "string" ? query.trim().toLowerCase() : "";

  if (!normalizedQuery) {
    return listCompanies();
  }

  return listCompanies().filter((company) => {
    return (
      company.name.toLowerCase().includes(normalizedQuery) ||
      company.provider.toLowerCase().includes(normalizedQuery) ||
      company.boardKey.toLowerCase().includes(normalizedQuery)
    );
  });
}

export function findCompanyById(id) {
  const company = companies.find((entry) => entry.id === id);
  return company ? enrichCompany(company) : null;
}

function enrichCompany(company) {
  return {
    ...company,
    identifierKind: getIdentifierKind(company.provider),
    boardUrl: buildBoardUrl(company.provider, company.boardKey),
    apiUrl: buildApiUrl(company.provider, company.boardKey)
  };
}
