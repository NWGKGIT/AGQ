package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"agq-daemon/internal/domain"
)

var localhostOrigin = regexp.MustCompile(`^https?://(localhost|127\.0\.0\.1)(:\d+)?$`)

type Store interface {
	GetAllAccounts() ([]domain.AccountSummary, error)
	GetAccount(email string) (*domain.AccountSummary, error)
	GetLatestSnapshot(email string) (*domain.QuotaSnapshot, error)
	GetSnapshotHistory(email string, limit int, before time.Time) ([]domain.QuotaSnapshot, error)
	GetLatestModelQuotas() ([]domain.ModelQuotaAggregate, error)
}

type StatusProvider interface {
	Snapshot() domain.DaemonStatus
}

type Server struct {
	store  Store
	status StatusProvider
	logger *slog.Logger
	now    func() time.Time
}

type Option func(*Server)

func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) { s.logger = logger }
}

func WithClock(now func() time.Time) Option {
	return func(s *Server) { s.now = now }
}

func New(store Store, status StatusProvider, opts ...Option) *Server {
	s := &Server{store: store, status: status, logger: slog.Default(), now: time.Now}
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

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.healthHandler)
	mux.HandleFunc("GET /api/status", s.statusHandler)
	mux.HandleFunc("GET /api/accounts", s.accountsHandler)
	mux.HandleFunc("GET /api/account/current", s.currentAccountHandler)
	mux.HandleFunc("GET /api/accounts/{email}/latest", s.accountLatestHandler)
	mux.HandleFunc("GET /api/accounts/{email}/snapshots", s.accountSnapshotsHandler)
	mux.HandleFunc("GET /api/models/latest", s.modelsLatestHandler)
	return corsMiddleware(mux)
}

func (s *Server) Run(ctx context.Context, addr string) error {
	httpSrv := &http.Server{Addr: addr, Handler: s.Handler()}
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
	s.logger.Info("api: listening", "addr", addr)
	err := httpSrv.ListenAndServe()
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

type currentAccountResponse struct {
	State      domain.DaemonState `json:"state"`
	Email      string             `json:"email,omitempty"`
	Account    *accountResponse   `json:"account,omitempty"`
	Accounts   []accountResponse  `json:"accounts"`
	LastPollAt *time.Time         `json:"last_poll_at,omitempty"`
	AsOf       time.Time          `json:"as_of"`
}

func (s *Server) currentAccountHandler(w http.ResponseWriter, r *http.Request) {
	status := s.status.Snapshot()
	resp := currentAccountResponse{
		State:      status.State,
		Accounts:   []accountResponse{},
		LastPollAt: status.LastPollAt,
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
			sr := s.toSnapshotResponse(snap)
			ar.LatestSnapshot = &sr
		}
		resp.Accounts = append(resp.Accounts, ar)
	}
	if len(resp.Accounts) > 0 {
		resp.Email = resp.Accounts[0].Email
		resp.Account = &resp.Accounts[0]
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	st := s.status.Snapshot()
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "uptime": st.Uptime})
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
		ar := accountResponse{ID: a.ID, Email: a.Email, PlanName: a.PlanName, FirstSeen: a.FirstSeen, LastSeen: a.LastSeen}
		snap, err := s.store.GetLatestSnapshot(a.Email)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query latest snapshot", err)
			return
		}
		if snap != nil {
			sr := s.toSnapshotResponse(snap)
			ar.LatestSnapshot = &sr
		}
		out = append(out, ar)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
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
	s.writeJSON(w, http.StatusOK, s.toSnapshotResponse(snap))
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
	s.writeJSON(w, http.StatusOK, map[string]any{"email": email, "snapshots": out})
}

func (s *Server) modelsLatestHandler(w http.ResponseWriter, r *http.Request) {
	models, err := s.store.GetLatestModelQuotas()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query model quotas", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"models": nilSafeSlice(models)})
}

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
	detail := ""
	if err != nil {
		detail = strings.ReplaceAll(err.Error(), "/home/", "~/")
	}
	s.writeJSON(w, status, map[string]string{"error": msg, "detail": detail})
}

func nilSafeSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
