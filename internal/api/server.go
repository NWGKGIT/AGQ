package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"agq-daemon/internal/domain"
)

var localhostOrigin = regexp.MustCompile(`^https?://(localhost|127\.0\.0\.1)(:\d+)?$`)

// currentModelsLookbackDays bounds how far back /models/current searches for a
// model's newest non-null fraction. Older values are too stale to present as
// "current", even with assumed-refill applied.
const currentModelsLookbackDays = 14

// Store is the persistence surface used by API handlers.
type Store interface {
	GetAllAccounts() ([]domain.AccountSummary, error)
	GetAccount(email string) (*domain.AccountSummary, error)
	GetLatestSnapshot(email string) (*domain.QuotaSnapshot, error)
	GetSnapshotHistory(email string, limit int, before time.Time) ([]domain.QuotaSnapshot, error)
	GetLatestModelQuotas() ([]domain.ModelQuotaAggregate, error)
	GetCurrentModelQuotas(email string, since time.Time) ([]domain.ModelQuotaAggregate, error)
	GetFractionSamplesSince(since time.Time) ([]domain.FractionSample, error)
	GetAccountFractionSamplesSince(email string, since time.Time) ([]domain.FractionSample, error)
	GetBreakdown() ([]domain.BreakdownRow, error)
	GetAnalyticsStats(now time.Time) (domain.AnalyticsStats, error)
	GetSnapshotRefsSince(email string, since time.Time) ([]domain.SnapshotRef, error)
	GetSnapshotModels(snapshotID int64) ([]domain.ModelQuota, error)
}

// StatusProvider returns the current daemon status.
type StatusProvider interface {
	Snapshot() domain.DaemonStatus
}

// Server owns the API handler dependencies.
type Server struct {
	store  Store
	status StatusProvider
	logger *slog.Logger
	now    func() time.Time
}

// Option customizes a Server.
type Option func(*Server)

// WithLogger sets the logger used for server lifecycle and encoding warnings.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithClock sets the clock used to compute response staleness.
func WithClock(now func() time.Time) Option {
	return func(s *Server) {
		s.now = now
	}
}

// New creates an API server.
func New(store Store, status StatusProvider, opts ...Option) *Server {
	s := &Server{
		store:  store,
		status: status,
		logger: slog.Default(),
		now:    time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

// Handler returns the complete HTTP handler tree.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.healthHandler)
	mux.HandleFunc("GET /api/status", s.statusHandler)
	mux.HandleFunc("GET /api/accounts", s.accountsHandler)
	mux.HandleFunc("GET /api/account/current", s.currentAccountHandler)
	mux.HandleFunc("GET /api/accounts/{email}/latest", s.accountLatestHandler)
	mux.HandleFunc("GET /api/accounts/{email}/snapshots", s.accountSnapshotsHandler)
	mux.HandleFunc("GET /api/accounts/{email}/sparklines", s.accountSparklinesHandler)
	mux.HandleFunc("GET /api/accounts/{email}/timeline", s.accountTimelineHandler)
	mux.HandleFunc("GET /api/accounts/{email}/models/current", s.accountModelsCurrentHandler)
	mux.HandleFunc("GET /api/models/latest", s.modelsLatestHandler)
	mux.HandleFunc("GET /api/analytics/timeseries", s.analyticsTimeseriesHandler)
	mux.HandleFunc("GET /api/analytics/breakdown", s.analyticsBreakdownHandler)
	mux.HandleFunc("GET /api/analytics/stats", s.analyticsStatsHandler)
	return corsMiddleware(mux)
}

// Run starts the HTTP API on addr and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ctx, listener)
}

// Serve runs the API on an already-open listener. Embedded hosts can reserve
// their port synchronously and surface startup failures before showing UI.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	httpSrv := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpSrv.Shutdown(shutCtx); err != nil {
				s.logger.Warn("api: shutdown error", "err", err)
			}
		case <-done:
		}
	}()

	s.logger.Info("api: listening", "addr", listener.Addr().String())
	err := httpSrv.Serve(listener)
	close(done)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type snapshotResponse struct {
	Email                  string              `json:"email"`
	PlanName               string              `json:"plan_name"`
	CapturedAt             time.Time           `json:"captured_at"`
	StalenessSeconds       int64               `json:"staleness_seconds"`
	PromptCreditsAvailable int64               `json:"prompt_credits_available"`
	PromptCreditsMonthly   int64               `json:"prompt_credits_monthly"`
	FlowCreditsAvailable   int64               `json:"flow_credits_available"`
	FlowCreditsMonthly     int64               `json:"flow_credits_monthly"`
	Models                 []domain.ModelQuota `json:"models"`
}

type accountResponse struct {
	ID             int64             `json:"id"`
	Email          string            `json:"email"`
	PlanName       string            `json:"plan_name"`
	FirstSeen      time.Time         `json:"first_seen"`
	LastSeen       time.Time         `json:"last_seen"`
	LatestSnapshot *snapshotResponse `json:"latest_snapshot,omitempty"`
}

// currentAccountResponse describes the account(s) the daemon is currently
// polling, i.e. the account(s) presently logged in to Antigravity. It reflects
// live daemon state joined with the newest persisted snapshot for each account.
//
// When state is IDLE, active account fields are empty. last_account carries the
// most recently seen account from the database so frontends can show "last known
// account" rather than a blank state. is_live is false in that case, signalling
// that the data is historical, not a confirmed live session.
type currentAccountResponse struct {
	State       domain.DaemonState `json:"state"`
	IsLive      bool               `json:"is_live"`
	Email       string             `json:"email,omitempty"`
	Account     *accountResponse   `json:"account,omitempty"`
	Accounts    []accountResponse  `json:"accounts"`
	LastPollAt  *time.Time         `json:"last_poll_at,omitempty"`
	NextPollAt  *time.Time         `json:"next_poll_at,omitempty"`
	LastAccount *accountResponse   `json:"last_account,omitempty"`
	AsOf        time.Time          `json:"as_of"`
}

func (s *Server) currentAccountHandler(w http.ResponseWriter, r *http.Request) {
	status := s.status.Snapshot()

	resp := currentAccountResponse{
		State:      status.State,
		IsLive:     status.State == domain.StateActive,
		Accounts:   []accountResponse{},
		LastPollAt: status.LastPollAt,
		NextPollAt: status.NextPollAt,
		AsOf:       s.now().UTC(),
	}

	for _, email := range status.Emails {
		ar := accountResponse{Email: email}

		account, err := s.store.GetAccount(email)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query account", err)
			return
		}
		if account != nil {
			ar.ID = account.ID
			ar.PlanName = account.PlanName
			ar.FirstSeen = account.FirstSeen
			ar.LastSeen = account.LastSeen
		}

		snap, err := s.store.GetLatestSnapshot(email)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query latest snapshot", err)
			return
		}
		if snap != nil {
			if ar.PlanName == "" {
				ar.PlanName = snap.PlanName
			}
			sr := s.toLatestSnapshotResponse(snap)
			ar.LatestSnapshot = &sr
		}

		resp.Accounts = append(resp.Accounts, ar)
	}

	// Antigravity is logged in to one account at a time; surface the first
	// active account directly for convenience while still returning the full
	// list for the rare multi-server case.
	if len(resp.Accounts) > 0 {
		resp.Email = resp.Accounts[0].Email
		resp.Account = &resp.Accounts[0]
	}

	// When idle, surface the most recently seen account from the DB so the
	// frontend can show "last known account" instead of a blank state. The
	// is_live=false flag makes clear this is historical, not a live session.
	if status.State == domain.StateIdle {
		all, err := s.store.GetAllAccounts()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query accounts", err)
			return
		}
		if len(all) > 0 {
			last := all[0]
			ar := accountResponse{
				ID:        last.ID,
				Email:     last.Email,
				PlanName:  last.PlanName,
				FirstSeen: last.FirstSeen,
				LastSeen:  last.LastSeen,
			}
			snap, err := s.store.GetLatestSnapshot(last.Email)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to query last snapshot", err)
				return
			}
			if snap != nil {
				sr := s.toLatestSnapshotResponse(snap)
				ar.LatestSnapshot = &sr
			}
			resp.LastAccount = &ar
		}
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	st := s.status.Snapshot()
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"uptime": st.Uptime,
	})
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.status.Snapshot())
}

func (s *Server) accountsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.GetAllAccounts()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query accounts", err)
		return
	}

	out := make([]accountResponse, 0, len(accounts))
	for _, a := range accounts {
		ar := accountResponse{
			ID:        a.ID,
			Email:     a.Email,
			PlanName:  a.PlanName,
			FirstSeen: a.FirstSeen,
			LastSeen:  a.LastSeen,
		}
		snap, err := s.store.GetLatestSnapshot(a.Email)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query latest snapshot", err)
			return
		}
		if snap != nil {
			sr := s.toLatestSnapshotResponse(snap)
			ar.LatestSnapshot = &sr
		}
		out = append(out, ar)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"accounts": out,
	})
}

func (s *Server) accountLatestHandler(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	snap, err := s.store.GetLatestSnapshot(email)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query snapshot", err)
		return
	}
	if snap == nil {
		s.writeError(w, http.StatusNotFound, "no snapshots found for account", nil)
		return
	}
	s.writeJSON(w, http.StatusOK, s.toLatestSnapshotResponse(snap))
}

func (s *Server) accountSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")

	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		n, err := strconv.Atoi(rawLimit)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid limit", err)
			return
		}
		limit = n
	}

	var before time.Time
	if rawBefore := r.URL.Query().Get("before"); rawBefore != "" {
		t, err := time.Parse(time.RFC3339, rawBefore)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid before timestamp", err)
			return
		}
		before = t
	}

	snaps, err := s.store.GetSnapshotHistory(email, limit, before)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query snapshots", err)
		return
	}

	out := make([]snapshotResponse, 0, len(snaps))
	for i := range snaps {
		out = append(out, s.toSnapshotResponse(&snaps[i]))
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"email":     email,
		"snapshots": out,
	})
}

func (s *Server) modelsLatestHandler(w http.ResponseWriter, r *http.Request) {
	models, err := s.store.GetLatestModelQuotas()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query model quotas", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"models": applyAssumedRefillAggregates(nilSafeSlice(models), s.now()),
	})
}

// accountModelsCurrentHandler serves each model's newest known quota value for
// one account. Unlike the latest snapshot — which can carry null fractions —
// every row here has a value: the newest non-null capture per model within the
// lookback window, with assumed-refill applied for elapsed resets.
func (s *Server) accountModelsCurrentHandler(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	since := s.now().UTC().AddDate(0, 0, -currentModelsLookbackDays)
	models, err := s.store.GetCurrentModelQuotas(email, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query current model quotas", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"email":  email,
		"models": applyAssumedRefillAggregates(nilSafeSlice(models), s.now()),
	})
}

// toSnapshotResponse serves a snapshot as observed, without reinterpretation.
// Used for history, where rows describe what was true at capture time.
func (s *Server) toSnapshotResponse(snapshot *domain.QuotaSnapshot) snapshotResponse {
	return snapshotResponse{
		Email:                  snapshot.Email,
		PlanName:               snapshot.PlanName,
		CapturedAt:             snapshot.CapturedAt,
		StalenessSeconds:       s.now().Unix() - snapshot.CapturedAt.Unix(),
		PromptCreditsAvailable: snapshot.PromptCreditsAvailable,
		PromptCreditsMonthly:   snapshot.PromptCreditsMonthly,
		FlowCreditsAvailable:   snapshot.FlowCreditsAvailable,
		FlowCreditsMonthly:     snapshot.FlowCreditsMonthly,
		Models:                 nilSafeSlice(snapshot.Models),
	}
}

// toLatestSnapshotResponse serves a snapshot as a description of the present:
// models whose reset has since passed are assumed refilled. Used wherever the
// snapshot stands in for "current state" (latest-snapshot fields).
func (s *Server) toLatestSnapshotResponse(snapshot *domain.QuotaSnapshot) snapshotResponse {
	resp := s.toSnapshotResponse(snapshot)
	resp.Models = applyAssumedRefill(resp.Models, s.now())
	return resp
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && localhostOrigin.MatchString(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		s.logger.Warn("api: encode response", "err", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg string, err error) {
	if err != nil {
		s.logger.Warn("api request failed", "status", status, "error", msg, "err", err)
	}
	s.writeJSON(w, status, map[string]string{
		"error": msg,
	})
}

func nilSafeSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
