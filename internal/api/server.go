package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jaykbpark/ats-job-monitor/internal/catalog"
	monitorpkg "github.com/jaykbpark/ats-job-monitor/internal/monitor"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

type Server struct {
	store   *store.Store
	monitor *monitorpkg.Service
	mux     *http.ServeMux
}

type createWatchTargetRequest struct {
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	BoardKey   string `json:"boardKey"`
	SourceURL  string `json:"sourceUrl"`
	Filters    any    `json:"filters"`
	FiltersRaw string `json:"filtersJson"`
	Status     string `json:"status"`
}

func NewServer(store *store.Store, syncService *monitorpkg.Service) *Server {
	if syncService == nil {
		syncService = monitorpkg.NewService(store, nil, nil, nil)
	}

	server := &Server{
		store:   store,
		monitor: syncService,
		mux:     http.NewServeMux(),
	}

	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return withLogging(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/companies", s.handleListCompanies)
	s.mux.HandleFunc("GET /api/watch-targets", s.handleListWatchTargets)
	s.mux.HandleFunc("POST /api/watch-targets", s.handleCreateWatchTarget)
	s.mux.HandleFunc("POST /api/watch-targets/{id}/sync", s.handleSyncWatchTarget)
	s.mux.HandleFunc("GET /api/watch-targets/{id}/jobs", s.handleListWatchTargetJobs)
	s.mux.HandleFunc("GET /api/watch-targets/{id}/sync-runs", s.handleListWatchTargetSyncRuns)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, catalog.SearchCompanies(query))
}

func (s *Server) handleListWatchTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := s.store.ListWatchTargets(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("list watch targets: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, targets)
}

func (s *Server) handleCreateWatchTarget(w http.ResponseWriter, r *http.Request) {
	var req createWatchTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	filtersJSON, err := normalizeFilters(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	target, err := s.store.CreateWatchTarget(r.Context(), store.CreateWatchTargetParams{
		Name:        req.Name,
		Provider:    req.Provider,
		BoardKey:    req.BoardKey,
		SourceURL:   req.SourceURL,
		FiltersJSON: filtersJSON,
		Status:      req.Status,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if isValidationError(err) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, target)
}

func (s *Server) handleSyncWatchTarget(w http.ResponseWriter, r *http.Request) {
	watchTargetID, ok := parseWatchTargetID(w, r)
	if !ok {
		return
	}

	run, err := s.monitor.SyncWatchTarget(r.Context(), watchTargetID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not supported") {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleListWatchTargetJobs(w http.ResponseWriter, r *http.Request) {
	watchTargetID, ok := parseWatchTargetID(w, r)
	if !ok {
		return
	}

	jobs, err := s.store.ListJobsByWatchTarget(r.Context(), watchTargetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("list watch target jobs: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleListWatchTargetSyncRuns(w http.ResponseWriter, r *http.Request) {
	watchTargetID, ok := parseWatchTargetID(w, r)
	if !ok {
		return
	}

	runs, err := s.store.ListSyncRunsByWatchTarget(r.Context(), watchTargetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("list sync runs: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, runs)
}

func normalizeFilters(req createWatchTargetRequest) (string, error) {
	if strings.TrimSpace(req.FiltersRaw) != "" && req.Filters != nil {
		return "", errors.New("provide either filters or filtersJson, not both")
	}

	if strings.TrimSpace(req.FiltersRaw) != "" {
		if !json.Valid([]byte(req.FiltersRaw)) {
			return "", errors.New("filtersJson must be valid JSON")
		}
		return req.FiltersRaw, nil
	}

	if req.Filters == nil {
		return "{}", nil
	}

	encoded, err := json.Marshal(req.Filters)
	if err != nil {
		return "", errors.New("filters must be JSON-serializable")
	}

	return string(encoded), nil
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{
		"error": message,
	})
}

func isValidationError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "required")
}

func parseWatchTargetID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	rawID := strings.TrimSpace(r.PathValue("id"))
	if rawID == "" {
		writeError(w, http.StatusBadRequest, "watch target id is required")
		return 0, false
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "watch target id must be a positive integer")
		return 0, false
	}

	return id, true
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		fmt.Printf("%s %s %d %s\n", r.Method, r.URL.Path, recorder.statusCode, time.Since(start).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func Shutdown(ctx context.Context, server *http.Server) error {
	if server == nil {
		return nil
	}

	return server.Shutdown(ctx)
}
