package catalog

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/internal/ats"
)

//go:embed companies.json
var companiesJSON []byte

type Company struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	BoardKey       string `json:"boardKey"`
	IdentifierKind string `json:"identifierKind"`
	BoardURL       string `json:"boardUrl"`
	APIURL         string `json:"apiUrl"`
}

type companyRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BoardKey string `json:"boardKey"`
}

var seededCompanies = mustLoadCompanies()

func ListCompanies() []Company {
	companies := make([]Company, len(seededCompanies))
	copy(companies, seededCompanies)
	return companies
}

func SearchCompanies(query string) []Company {
	normalized := strings.TrimSpace(strings.ToLower(query))
	if normalized == "" {
		return ListCompanies()
	}

	results := make([]Company, 0, len(seededCompanies))
	for _, company := range seededCompanies {
		if strings.Contains(strings.ToLower(company.Name), normalized) ||
			strings.Contains(strings.ToLower(company.Provider), normalized) ||
			strings.Contains(strings.ToLower(company.BoardKey), normalized) {
			results = append(results, company)
		}
	}

	return results
}

func FindCompanyByID(id string) (Company, bool) {
	for _, company := range seededCompanies {
		if company.ID == id {
			return company, true
		}
	}

	return Company{}, false
}

func mustLoadCompanies() []Company {
	var records []companyRecord
	if err := json.Unmarshal(companiesJSON, &records); err != nil {
		panic(err)
	}

	companies := make([]Company, 0, len(records))
	for _, record := range records {
		companies = append(companies, Company{
			ID:             record.ID,
			Name:           record.Name,
			Provider:       record.Provider,
			BoardKey:       record.BoardKey,
			IdentifierKind: ats.GetIdentifierKind(record.Provider),
			BoardURL:       ats.BuildBoardURL(record.Provider, record.BoardKey),
			APIURL:         ats.BuildAPIURL(record.Provider, record.BoardKey),
		})
	}

	return companies
}
