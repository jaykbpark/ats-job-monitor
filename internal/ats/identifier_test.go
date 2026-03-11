package ats

import "testing"

func TestParseIdentifierLeverBoardURL(t *testing.T) {
	match, ok := ParseIdentifier("https://jobs.lever.co/acme/jobs/123")
	if !ok {
		t.Fatal("expected Lever match")
	}

	if match.Provider != ProviderLever {
		t.Fatalf("expected provider %q, got %q", ProviderLever, match.Provider)
	}

	if match.IdentifierKind != "site" {
		t.Fatalf("expected identifier kind site, got %q", match.IdentifierKind)
	}

	if match.BoardKey != "acme" {
		t.Fatalf("expected board key acme, got %q", match.BoardKey)
	}

	if match.SourceType != "board-url" {
		t.Fatalf("expected source type board-url, got %q", match.SourceType)
	}
}

func TestParseIdentifierLeverAPIURL(t *testing.T) {
	match, ok := ParseIdentifier("https://api.lever.co/v0/postings/acme?mode=json")
	if !ok {
		t.Fatal("expected Lever API match")
	}

	if match.Provider != ProviderLever || match.BoardKey != "acme" || match.SourceType != "api-url" {
		t.Fatalf("unexpected Lever API match: %#v", match)
	}
}

func TestParseIdentifierGreenhouseBoardURL(t *testing.T) {
	match, ok := ParseIdentifier("https://job-boards.greenhouse.io/acme/jobs/77")
	if !ok {
		t.Fatal("expected Greenhouse match")
	}

	if match.Provider != ProviderGreenhouse {
		t.Fatalf("expected provider %q, got %q", ProviderGreenhouse, match.Provider)
	}

	if match.IdentifierKind != "board_token" {
		t.Fatalf("expected identifier kind board_token, got %q", match.IdentifierKind)
	}

	if match.BoardKey != "acme" {
		t.Fatalf("expected board key acme, got %q", match.BoardKey)
	}
}

func TestParseIdentifierGreenhouseEmbedURL(t *testing.T) {
	match, ok := ParseIdentifier("https://boards.greenhouse.io/embed/job_board?for=acme")
	if !ok {
		t.Fatal("expected Greenhouse embed match")
	}

	if match.Provider != ProviderGreenhouse || match.BoardKey != "acme" || match.SourceType != "embed-url" {
		t.Fatalf("unexpected Greenhouse embed match: %#v", match)
	}
}

func TestParseIdentifierAshbyBoardURL(t *testing.T) {
	match, ok := ParseIdentifier("https://jobs.ashbyhq.com/Acme/jobs/123")
	if !ok {
		t.Fatal("expected Ashby match")
	}

	if match.Provider != ProviderAshby {
		t.Fatalf("expected provider %q, got %q", ProviderAshby, match.Provider)
	}

	if match.IdentifierKind != "job_board_name" {
		t.Fatalf("expected identifier kind job_board_name, got %q", match.IdentifierKind)
	}

	if match.BoardKey != "Acme" {
		t.Fatalf("expected board key Acme, got %q", match.BoardKey)
	}
}

func TestParseIdentifierAshbyAPIURL(t *testing.T) {
	match, ok := ParseIdentifier("https://api.ashbyhq.com/posting-api/job-board/Acme")
	if !ok {
		t.Fatal("expected Ashby API match")
	}

	if match.Provider != ProviderAshby || match.BoardKey != "Acme" || match.SourceType != "api-url" {
		t.Fatalf("unexpected Ashby API match: %#v", match)
	}
}

func TestParseIdentifierWithoutScheme(t *testing.T) {
	match, ok := ParseIdentifier("job-boards.greenhouse.io/acme")
	if !ok {
		t.Fatal("expected Greenhouse match without scheme")
	}

	if match.Provider != ProviderGreenhouse || match.BoardKey != "acme" {
		t.Fatalf("unexpected match without scheme: %#v", match)
	}
}

func TestParseIdentifierUnsupportedURL(t *testing.T) {
	if match, ok := ParseIdentifier("https://careers.example.com/jobs"); ok || match != nil {
		t.Fatalf("expected unsupported URL to return no match, got %#v", match)
	}
}
