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
	"github.com/jaykbpark/ats-job-monitor/internal/filters"
	monitorpkg "github.com/jaykbpark/ats-job-monitor/internal/monitor"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

type Server struct {
	store   *store.Store
	monitor *monitorpkg.Service
	mux     *http.ServeMux
}

type createWatchTargetRequest struct {
	Name              string `json:"name"`
	Provider          string `json:"provider"`
	BoardKey          string `json:"boardKey"`
	SourceURL         string `json:"sourceUrl"`
	NotificationEmail string `json:"notificationEmail"`
	Filters           any    `json:"filters"`
	FiltersRaw        string `json:"filtersJson"`
	Status            string `json:"status"`
}

type updateWatchTargetRequest struct {
	Name              *string         `json:"name"`
	SourceURL         *string         `json:"sourceUrl"`
	NotificationEmail *string         `json:"notificationEmail"`
	Filters           json.RawMessage `json:"filters"`
	FiltersRaw        *string         `json:"filtersJson"`
	Status            *string         `json:"status"`
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
	s.mux.HandleFunc("PATCH /api/watch-targets/{id}", s.handleUpdateWatchTarget)
	s.mux.HandleFunc("POST /api/watch-targets/{id}/sync", s.handleSyncWatchTarget)
	s.mux.HandleFunc("GET /api/watch-targets/{id}/jobs", s.handleListWatchTargetJobs)
	s.mux.HandleFunc("GET /api/watch-targets/{id}/sync-runs", s.handleListWatchTargetSyncRuns)
	s.mux.HandleFunc("GET /api/watch-targets/{id}/notifications", s.handleListWatchTargetNotifications)
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
		Name:              req.Name,
		Provider:          req.Provider,
		BoardKey:          req.BoardKey,
		SourceURL:         req.SourceURL,
		NotificationEmail: req.NotificationEmail,
		FiltersJSON:       filtersJSON,
		Status:            req.Status,
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

func (s *Server) handleUpdateWatchTarget(w http.ResponseWriter, r *http.Request) {
	watchTargetID, ok := parseWatchTargetID(w, r)
	if !ok {
		return
	}

	var req updateWatchTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	filtersJSON, err := normalizeUpdateFilters(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	target, err := s.store.UpdateWatchTarget(r.Context(), store.UpdateWatchTargetParams{
		ID:                watchTargetID,
		Name:              req.Name,
		SourceURL:         req.SourceURL,
		NotificationEmail: req.NotificationEmail,
		FiltersJSON:       filtersJSON,
		Status:            req.Status,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if isValidationError(err) {
			status = http.StatusBadRequest
		} else if strings.Contains(err.Error(), "no rows") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, target)
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

	matchedFilter, err := parseOptionalBoolQuery(r, "matched")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	jobs, err := s.store.ListJobsByWatchTarget(r.Context(), store.ListJobsParams{
		WatchTargetID: watchTargetID,
		Matched:       matchedFilter,
	})
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

func (s *Server) handleListWatchTargetNotifications(w http.ResponseWriter, r *http.Request) {
	watchTargetID, ok := parseWatchTargetID(w, r)
	if !ok {
		return
	}

	notifications, err := s.store.ListNotificationsByWatchTarget(r.Context(), watchTargetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("list notifications: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, notifications)
}

func normalizeFilters(req createWatchTargetRequest) (string, error) {
	if strings.TrimSpace(req.FiltersRaw) != "" && req.Filters != nil {
		return "", errors.New("provide either filters or filtersJson, not both")
	}

	if strings.TrimSpace(req.FiltersRaw) != "" {
		normalizedJSON, _, err := filters.NormalizeHardFiltersJSON(req.FiltersRaw)
		if err != nil {
			return "", fmt.Errorf("filtersJson is invalid: %w", err)
		}
		return normalizedJSON, nil
	}

	if req.Filters == nil {
		normalizedJSON, _, err := filters.NormalizeHardFiltersJSON(`{}`)
		return normalizedJSON, err
	}

	encoded, err := json.Marshal(req.Filters)
	if err != nil {
		return "", errors.New("filters must be JSON-serializable")
	}

	normalizedJSON, _, err := filters.NormalizeHardFiltersJSON(string(encoded))
	if err != nil {
		return "", fmt.Errorf("filters are invalid: %w", err)
	}

	return normalizedJSON, nil
}

func normalizeUpdateFilters(req updateWatchTargetRequest) (*string, error) {
	if req.FiltersRaw != nil && req.Filters != nil {
		return nil, errors.New("provide either filters or filtersJson, not both")
	}

	if req.FiltersRaw != nil {
		normalizedJSON, _, err := filters.NormalizeHardFiltersJSON(strings.TrimSpace(*req.FiltersRaw))
		if err != nil {
			return nil, fmt.Errorf("filtersJson is invalid: %w", err)
		}
		return &normalizedJSON, nil
	}

	if req.Filters == nil {
		return nil, nil
	}

	if string(req.Filters) == "null" {
		normalizedJSON, _, err := filters.NormalizeHardFiltersJSON(`{}`)
		return &normalizedJSON, err
	}

	normalizedJSON, _, err := filters.NormalizeHardFiltersJSON(string(req.Filters))
	if err != nil {
		return nil, fmt.Errorf("filters are invalid: %w", err)
	}

	return &normalizedJSON, nil
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
	return strings.Contains(message, "required") || strings.Contains(message, "invalid")
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

func parseOptionalBoolQuery(r *http.Request, key string) (*bool, error) {
	rawValue := strings.TrimSpace(r.URL.Query().Get(key))
	if rawValue == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(rawValue)
	if err != nil {
		return nil, fmt.Errorf("%s must be true or false", key)
	}

	return &parsed, nil
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
