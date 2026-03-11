package ats

import (
	"net/url"
	"strings"
)

const (
	ProviderLever      = "lever"
	ProviderGreenhouse = "greenhouse"
	ProviderAshby      = "ashby"
)

type Match struct {
	Provider       string `json:"provider"`
	IdentifierKind string `json:"identifierKind"`
	BoardKey       string `json:"boardKey"`
	SourceType     string `json:"sourceType"`
	NormalizedURL  string `json:"normalizedUrl"`
	BoardURL       string `json:"boardUrl"`
	APIURL         string `json:"apiUrl"`
}

func GetIdentifierKind(provider string) string {
	switch provider {
	case ProviderLever:
		return "site"
	case ProviderGreenhouse:
		return "board_token"
	case ProviderAshby:
		return "job_board_name"
	default:
		return "board_key"
	}
}

func BuildBoardURL(provider string, boardKey string) string {
	switch provider {
	case ProviderLever:
		return "https://jobs.lever.co/" + boardKey
	case ProviderGreenhouse:
		return "https://job-boards.greenhouse.io/" + boardKey
	case ProviderAshby:
		return "https://jobs.ashbyhq.com/" + boardKey
	default:
		return ""
	}
}

func BuildAPIURL(provider string, boardKey string) string {
	switch provider {
	case ProviderLever:
		return "https://api.lever.co/v0/postings/" + boardKey + "?mode=json"
	case ProviderGreenhouse:
		return "https://boards-api.greenhouse.io/v1/boards/" + boardKey + "/jobs"
	case ProviderAshby:
		return "https://api.ashbyhq.com/posting-api/job-board/" + boardKey
	default:
		return ""
	}
}

func ParseIdentifier(input string) (*Match, bool) {
	u, ok := coerceURL(input)
	if !ok {
		return nil, false
	}

	if match, ok := parseLever(u); ok {
		return match, true
	}

	if match, ok := parseGreenhouse(u); ok {
		return match, true
	}

	if match, ok := parseAshby(u); ok {
		return match, true
	}

	return nil, false
}

func coerceURL(input string) (*url.URL, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, false
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return nil, false
	}

	return u, true
}

func parseLever(u *url.URL) (*Match, bool) {
	host := strings.ToLower(u.Hostname())
	pathParts := trimPath(u.EscapedPath())

	if host == "api.lever.co" && len(pathParts) >= 3 && pathParts[0] == "v0" && pathParts[1] == "postings" {
		return createMatch(ProviderLever, pathParts[2], u, "api-url"), true
	}

	if (host == "jobs.lever.co" || host == "jobs.eu.lever.co") && len(pathParts) >= 1 {
		return createMatch(ProviderLever, pathParts[0], u, "board-url"), true
	}

	return nil, false
}

func parseGreenhouse(u *url.URL) (*Match, bool) {
	host := strings.ToLower(u.Hostname())
	pathParts := trimPath(u.EscapedPath())

	if host == "boards-api.greenhouse.io" && len(pathParts) >= 3 && pathParts[0] == "v1" && pathParts[1] == "boards" {
		return createMatch(ProviderGreenhouse, pathParts[2], u, "api-url"), true
	}

	if host == "boards.greenhouse.io" && len(pathParts) >= 2 && pathParts[0] == "embed" && pathParts[1] == "job_board" {
		boardToken := strings.TrimSpace(u.Query().Get("for"))
		if boardToken != "" {
			return createMatch(ProviderGreenhouse, boardToken, u, "embed-url"), true
		}
	}

	if (host == "boards.greenhouse.io" || host == "job-boards.greenhouse.io") && len(pathParts) >= 1 {
		return createMatch(ProviderGreenhouse, pathParts[0], u, "board-url"), true
	}

	return nil, false
}

func parseAshby(u *url.URL) (*Match, bool) {
	host := strings.ToLower(u.Hostname())
	pathParts := trimPath(u.EscapedPath())

	if host == "api.ashbyhq.com" && len(pathParts) >= 3 && pathParts[0] == "posting-api" && pathParts[1] == "job-board" {
		return createMatch(ProviderAshby, pathParts[2], u, "api-url"), true
	}

	if host == "jobs.ashbyhq.com" && len(pathParts) >= 1 {
		return createMatch(ProviderAshby, pathParts[0], u, "board-url"), true
	}

	return nil, false
}

func createMatch(provider string, boardKey string, u *url.URL, sourceType string) *Match {
	normalizedKey, err := url.PathUnescape(boardKey)
	if err != nil {
		normalizedKey = boardKey
	}

	normalizedKey = strings.TrimSpace(normalizedKey)

	return &Match{
		Provider:       provider,
		IdentifierKind: GetIdentifierKind(provider),
		BoardKey:       normalizedKey,
		SourceType:     sourceType,
		NormalizedURL:  u.String(),
		BoardURL:       BuildBoardURL(provider, normalizedKey),
		APIURL:         BuildAPIURL(provider, normalizedKey),
	}
}

func trimPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return filtered
}
