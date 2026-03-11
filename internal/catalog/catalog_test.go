package catalog

import "testing"

func TestListCompaniesIncludesDerivedURLs(t *testing.T) {
	companies := ListCompanies()
	var greenhouse Company
	found := false

	for _, company := range companies {
		if company.ID == "greenhouse" {
			greenhouse = company
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected greenhouse company in catalog")
	}

	if greenhouse.BoardURL != "https://job-boards.greenhouse.io/greenhouse" {
		t.Fatalf("unexpected board URL: %q", greenhouse.BoardURL)
	}

	if greenhouse.APIURL != "https://boards-api.greenhouse.io/v1/boards/greenhouse/jobs" {
		t.Fatalf("unexpected API URL: %q", greenhouse.APIURL)
	}
}

func TestSearchCompaniesMatchesNameAndProvider(t *testing.T) {
	if matches := SearchCompanies("ash"); len(matches) != 1 {
		t.Fatalf("expected 1 match for ash, got %d", len(matches))
	}

	matches := SearchCompanies("lever")
	if len(matches) == 0 || matches[0].Provider != "lever" {
		t.Fatalf("unexpected matches for lever: %#v", matches)
	}
}

func TestFindCompanyByID(t *testing.T) {
	company, ok := FindCompanyByID("ashby")
	if !ok {
		t.Fatal("expected ashby company to exist")
	}

	if company.Name != "Ashby" {
		t.Fatalf("expected company name Ashby, got %q", company.Name)
	}

	if company.IdentifierKind != "job_board_name" {
		t.Fatalf("expected job_board_name, got %q", company.IdentifierKind)
	}
}
