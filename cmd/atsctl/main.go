package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/internal/ats"
	"github.com/jaykbpark/ats-job-monitor/internal/catalog"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(1)
	}

	switch os.Args[1] {
	case "detect":
		runDetect(os.Args[2:])
	case "companies":
		runCompanies(os.Args[2:])
	case "migrate":
		runMigrate(os.Args[2:])
	default:
		printUsage(1)
	}
}

func runDetect(args []string) {
	input := strings.TrimSpace(strings.Join(args, " "))
	if input == "" {
		printUsage(1)
	}

	match, ok := ats.ParseIdentifier(input)
	if !ok {
		_, _ = fmt.Fprintln(os.Stderr, "Could not match a supported ATS URL.")
		os.Exit(1)
	}

	writeJSON(match)
}

func runCompanies(args []string) {
	query := strings.TrimSpace(strings.Join(args, " "))
	writeJSON(catalog.SearchCompanies(query))
}

func runMigrate(args []string) {
	if len(args) != 1 {
		printUsage(1)
	}

	dbPath := strings.TrimSpace(args[0])
	if dbPath == "" {
		printUsage(1)
	}

	dbStore, err := store.Open(dbPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = dbStore.Close()
	}()

	if err := dbStore.Migrate(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to apply migrations: %v\n", err)
		os.Exit(1)
	}

	records, err := dbStore.AppliedMigrations(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read applied migrations: %v\n", err)
		os.Exit(1)
	}

	writeJSON(records)
}

func writeJSON(value any) {
	output, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func printUsage(exitCode int) {
	_, _ = fmt.Fprintln(os.Stderr, "Usage:")
	_, _ = fmt.Fprintln(os.Stderr, "  go run ./cmd/atsctl detect <ats-url>")
	_, _ = fmt.Fprintln(os.Stderr, "  go run ./cmd/atsctl companies [query]")
	_, _ = fmt.Fprintln(os.Stderr, "  go run ./cmd/atsctl migrate <db-path>")
	os.Exit(exitCode)
}
