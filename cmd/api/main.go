package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jaykbpark/ats-job-monitor/internal/api"
	"github.com/jaykbpark/ats-job-monitor/internal/monitor"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

func main() {
	dbPath := envOrDefault("ATS_JOB_MONITOR_DB", "./ats-job-monitor.db")
	addr := envOrDefault("ATS_JOB_MONITOR_ADDR", ":8080")

	dbStore, err := store.Open(dbPath)
	if err != nil {
		fatalf("open store: %v", err)
	}
	defer func() {
		_ = dbStore.Close()
	}()

	if err := dbStore.Migrate(context.Background()); err != nil {
		fatalf("apply migrations: %v", err)
	}

	server := api.NewServer(dbStore, monitor.NewService(dbStore, nil, nil, nil))
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		fmt.Printf("API listening on %s using %s\n", addr, dbPath)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatalf("listen and serve: %v", err)
		}
	}()

	waitForShutdown(httpServer)
}

func waitForShutdown(httpServer *http.Server) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := api.Shutdown(ctx, httpServer); err != nil {
		fatalf("shutdown API: %v", err)
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
