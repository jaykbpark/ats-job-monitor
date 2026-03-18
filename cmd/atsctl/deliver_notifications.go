package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/internal/notify"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

func runDeliverNotifications(args []string) {
	limit, dbPath, err := parseDeliverNotificationsArgs(args)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
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

	service := notify.NewService(dbStore, notify.NewConsoleSink(os.Stdout))
	result, err := service.DeliverPending(context.Background(), limit)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to deliver notifications: %v\n", err)
		os.Exit(1)
	}

	writeJSON(result)
}

func parseDeliverNotificationsArgs(args []string) (int, string, error) {
	limit := 0
	dbPath := ""

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "--limit", "-limit":
			if i+1 >= len(args) {
				return 0, "", fmt.Errorf("missing value for --limit")
			}
			value, err := strconv.Atoi(strings.TrimSpace(args[i+1]))
			if err != nil || value < 0 {
				return 0, "", fmt.Errorf("--limit must be a non-negative integer")
			}
			limit = value
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return 0, "", fmt.Errorf("unsupported flag %q", arg)
			}
			if dbPath != "" {
				return 0, "", fmt.Errorf("only one database path may be provided")
			}
			dbPath = arg
		}
	}

	if dbPath == "" {
		return 0, "", fmt.Errorf("database path is required")
	}

	return limit, dbPath, nil
}
