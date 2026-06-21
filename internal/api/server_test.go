package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agq-daemon/internal/domain"
)

func TestAccountsIncludesLatestSnapshot(t *testing.T) {
	store := &fakeStore{
		accounts: []domain.AccountSummary{{
			ID:        1,
			Email:     "user@example.com",
			PlanName:  "Pro",
			FirstSeen: fixedTime(-2 * time.Hour),
			LastSeen:  fixedTime(-time.Hour),
		}},
		latest: map[string]*domain.QuotaSnapshot{
			"user@example.com": {
				Email:                  "user@example.com",
				PlanName:               "Pro",
				CapturedAt:             fixedTime(-30 * time.Second),
				PromptCreditsAvailable: 10,
				PromptCreditsMonthly:   100,
				Models:                 []domain.ModelQuota{},
			},
		},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Accounts []struct {
			Email          string `json:"email"`
			LatestSnapshot struct {
				StalenessSeconds int64 `json:"staleness_seconds"`
			} `json:"latest_snapshot"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Accounts) != 1 {
		t.Fatalf("len(accounts) = %d, want 1", len(body.Accounts))
	}
	if body.Accounts[0].Email != "user@example.com" {
		t.Fatalf("email = %q, want user@example.com", body.Accounts[0].Email)
	}
	if body.Accounts[0].LatestSnapshot.StalenessSeconds != 30 {
		t.Fatalf("staleness = %d, want 30", body.Accounts[0].LatestSnapshot.StalenessSeconds)
	}
}

func TestCurrentAccountReturnsActiveAccount(t *testing.T) {
	store := &fakeStore{
		accounts: []domain.AccountSummary{{
			ID:        7,
			Email:     "active@example.com",
			PlanName:  "Pro",
			FirstSeen: fixedTime(-2 * time.Hour),
			LastSeen:  fixedTime(-time.Minute),
		}},
		latest: map[string]*domain.QuotaSnapshot{
			"active@example.com": {
				Email:                  "active@example.com",
				PlanName:               "Pro",
				CapturedAt:             fixedTime(-15 * time.Second),
				PromptCreditsAvailable: 42,
				Models:                 []domain.ModelQuota{},
			},
		},
	}
	status := fakeStatus{state: domain.StateActive, emails: []string{"active@example.com"}}
	srv := New(store, status, WithClock(func() time.Time { return fixedTime(0) }))

	req := httptest.NewRequest(http.MethodGet, "/api/account/current", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		State   string `json:"state"`
		Email   string `json:"email"`
		Account struct {
			ID             int64 `json:"id"`
			LatestSnapshot *struct {
				StalenessSeconds int64 `json:"staleness_seconds"`
			} `json:"latest_snapshot"`
		} `json:"account"`
		Accounts []struct {
			Email string `json:"email"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.State != string(domain.StateActive) {
		t.Fatalf("state = %q, want ACTIVE", body.State)
	}
	if body.Email != "active@example.com" {
		t.Fatalf("email = %q, want active@example.com", body.Email)
	}
	if body.Account.ID != 7 {
		t.Fatalf("account.id = %d, want 7", body.Account.ID)
	}
	if body.Account.LatestSnapshot == nil || body.Account.LatestSnapshot.StalenessSeconds != 15 {
		t.Fatalf("account latest snapshot staleness = %+v, want 15", body.Account.LatestSnapshot)
	}
	if len(body.Accounts) != 1 {
		t.Fatalf("len(accounts) = %d, want 1", len(body.Accounts))
	}
}

func TestCurrentAccountIdle(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{})

	req := httptest.NewRequest(http.MethodGet, "/api/account/current", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		State    string            `json:"state"`
		Email    string            `json:"email"`
		Accounts []json.RawMessage `json:"accounts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.State != string(domain.StateIdle) {
		t.Fatalf("state = %q, want IDLE", body.State)
	}
	if body.Email != "" {
		t.Fatalf("email = %q, want empty", body.Email)
	}
	if len(body.Accounts) != 0 {
		t.Fatalf("len(accounts) = %d, want 0", len(body.Accounts))
	}
}

func TestSnapshotsRejectsInvalidQuery(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{})

	for _, path := range []string{
		"/api/accounts/user@example.com/snapshots?limit=nope",
		"/api/accounts/user@example.com/snapshots?before=nope",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", path, rec.Code)
		}
	}
}

func TestCORSAllowsLocalhostOrigins(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{})
	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestAccountsPropagatesLatestSnapshotErrors(t *testing.T) {
	store := &fakeStore{
		accounts:  []domain.AccountSummary{{Email: "user@example.com"}},
		latestErr: errors.New("db failed"),
	}
	srv := New(store, fakeStatus{})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

type fakeStore struct {
	accounts        []domain.AccountSummary
	latest          map[string]*domain.QuotaSnapshot
	history         []domain.QuotaSnapshot
	models          []domain.ModelQuotaAggregate
	fractionSamples []domain.FractionSample
	accountSamples  map[string][]domain.FractionSample
	breakdown       []domain.BreakdownRow
	stats           domain.AnalyticsStats
	snapshotRefs    map[string][]domain.SnapshotRef
	snapshotModels  map[int64][]domain.ModelQuota
	accountsErr     error
	latestErr       error
	historyErr      error
	modelsErr       error
	samplesErr      error
	breakdownErr    error
	statsErr        error
	refsErr         error
	snapModelsErr   error
}

func (f *fakeStore) GetAllAccounts() ([]domain.AccountSummary, error) {
	return f.accounts, f.accountsErr
}

func (f *fakeStore) GetAccount(email string) (*domain.AccountSummary, error) {
	if f.accountsErr != nil {
		return nil, f.accountsErr
	}
	for i := range f.accounts {
		if f.accounts[i].Email == email {
			return &f.accounts[i], nil
		}
	}
	return nil, nil
}

func (f *fakeStore) GetLatestSnapshot(email string) (*domain.QuotaSnapshot, error) {
	if f.latestErr != nil {
		return nil, f.latestErr
	}
	return f.latest[email], nil
}

func (f *fakeStore) GetSnapshotHistory(email string, limit int, before time.Time) ([]domain.QuotaSnapshot, error) {
	return f.history, f.historyErr
}

func (f *fakeStore) GetLatestModelQuotas() ([]domain.ModelQuotaAggregate, error) {
	return f.models, f.modelsErr
}

func (f *fakeStore) GetFractionSamplesSince(since time.Time) ([]domain.FractionSample, error) {
	return f.fractionSamples, f.samplesErr
}

func (f *fakeStore) GetAccountFractionSamplesSince(email string, since time.Time) ([]domain.FractionSample, error) {
	if f.samplesErr != nil {
		return nil, f.samplesErr
	}
	return f.accountSamples[email], nil
}

func (f *fakeStore) GetBreakdown() ([]domain.BreakdownRow, error) {
	return f.breakdown, f.breakdownErr
}

func (f *fakeStore) GetAnalyticsStats(now time.Time) (domain.AnalyticsStats, error) {
	return f.stats, f.statsErr
}

func (f *fakeStore) GetSnapshotRefsSince(email string, since time.Time) ([]domain.SnapshotRef, error) {
	if f.refsErr != nil {
		return nil, f.refsErr
	}
	return f.snapshotRefs[email], nil
}

func (f *fakeStore) GetSnapshotModels(snapshotID int64) ([]domain.ModelQuota, error) {
	if f.snapModelsErr != nil {
		return nil, f.snapModelsErr
	}
	return f.snapshotModels[snapshotID], nil
}

type fakeStatus struct {
	state      domain.DaemonState
	emails     []string
	lastPollAt *time.Time
}

func (f fakeStatus) Snapshot() domain.DaemonStatus {
	state := f.state
	if state == "" {
		state = domain.StateIdle
	}
	return domain.DaemonStatus{
		State:      state,
		Emails:     f.emails,
		Uptime:     "1s",
		StartedAt:  fixedTime(-time.Second),
		LastPollAt: f.lastPollAt,
	}
}

func fixedTime(offset time.Duration) time.Time {
	return time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).Add(offset)
}
