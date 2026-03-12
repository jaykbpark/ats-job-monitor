package filters

import (
	"encoding/json"
	"fmt"
	"strings"
)

type HardFilters struct {
	LocationsAny           []string `json:"locationsAny,omitempty"`
	RemoteOnly             bool     `json:"remoteOnly,omitempty"`
	EmploymentTypes        []string `json:"employmentTypes,omitempty"`
	IncludeKeywordsAny     []string `json:"includeKeywordsAny,omitempty"`
	ExcludeKeywords        []string `json:"excludeKeywords,omitempty"`
	SeniorityAny           []string `json:"seniorityAny,omitempty"`
	MinYearsExperience     *int     `json:"minYearsExperience,omitempty"`
	MaxYearsExperience     *int     `json:"maxYearsExperience,omitempty"`
	AllowUnknownExperience bool     `json:"allowUnknownExperience,omitempty"`
}

func ParseHardFilters(raw string) (HardFilters, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return HardFilters{}, nil
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()

	var filters HardFilters
	if err := decoder.Decode(&filters); err != nil {
		return HardFilters{}, fmt.Errorf("parse hard filters: %w", err)
	}

	if err := validateHardFilters(filters); err != nil {
		return HardFilters{}, err
	}

	return normalizeHardFilters(filters), nil
}

func NormalizeHardFiltersJSON(raw string) (string, HardFilters, error) {
	filters, err := ParseHardFilters(raw)
	if err != nil {
		return "", HardFilters{}, err
	}

	encoded, err := json.Marshal(filters)
	if err != nil {
		return "", HardFilters{}, fmt.Errorf("encode hard filters: %w", err)
	}

	return string(encoded), filters, nil
}

func validateHardFilters(filters HardFilters) error {
	if filters.MinYearsExperience != nil && *filters.MinYearsExperience < 0 {
		return fmt.Errorf("minYearsExperience must be >= 0")
	}

	if filters.MaxYearsExperience != nil && *filters.MaxYearsExperience < 0 {
		return fmt.Errorf("maxYearsExperience must be >= 0")
	}

	if filters.MinYearsExperience != nil && filters.MaxYearsExperience != nil && *filters.MinYearsExperience > *filters.MaxYearsExperience {
		return fmt.Errorf("minYearsExperience must be <= maxYearsExperience")
	}

	return nil
}

func normalizeHardFilters(filters HardFilters) HardFilters {
	filters.LocationsAny = normalizeStringList(filters.LocationsAny)
	filters.EmploymentTypes = normalizeStringList(filters.EmploymentTypes)
	filters.IncludeKeywordsAny = normalizeStringList(filters.IncludeKeywordsAny)
	filters.ExcludeKeywords = normalizeStringList(filters.ExcludeKeywords)
	filters.SeniorityAny = normalizeStringList(filters.SeniorityAny)
	return filters
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
