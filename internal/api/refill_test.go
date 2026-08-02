package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agq-daemon/internal/domain"
)

func TestApplyAssumedRefillPastReset(t *testing.T) {
	now := fixedTime(0)
	past := fixedTime(-time.Hour)
	future := fixedTime(time.Hour)
	depleted := 0.2
	pct := 20.0

	models := []domain.ModelQuota{
		{Label: "Gemini 3 Pro", RemainingFraction: &depleted, RemainingPct: &pct,
			IsExhausted: true, ResetTime: &past, PoolResetTime: &past},
		{Label: "Claude Sonnet", RemainingFraction: &depleted, RemainingPct: &pct,
			ResetTime: &future, PoolResetTime: &future},
		{Label: "GPT no reset", RemainingFraction: &depleted, RemainingPct: &pct},
	}

	out := applyAssumedRefill(models, now)

	if !out[0].AssumedRefilled {
		t.Fatal("past-reset model should be assumed refilled")
	}
	if *out[0].RemainingFraction != 1.0 || *out[0].RemainingPct != 100.0 || out[0].IsExhausted {
		t.Fatalf("past-reset model = %+v, want full and not exhausted", out[0])
	}
	if out[1].AssumedRefilled || *out[1].RemainingFraction != depleted {
		t.Fatalf("future-reset model should pass through unchanged, got %+v", out[1])
	}
	if out[2].AssumedRefilled || *out[2].RemainingFraction != depleted {
		t.Fatalf("no-reset model should pass through unchanged, got %+v", out[2])
	}
	// The input slice must not be mutated: stored data stays as observed.
	if *models[0].RemainingFraction != depleted || models[0].AssumedRefilled {
		t.Fatal("applyAssumedRefill mutated its input")
	}
}

func TestLatestSnapshotAssumesRefillHistoryDoesNot(t *testing.T) {
	past := fixedTime(-time.Hour)
	depleted := 0.1
	snap := &domain.QuotaSnapshot{
		Email:      "user@example.com",
		CapturedAt: fixedTime(-2 * time.Hour),
		Models: []domain.ModelQuota{
			{Label: "Gemini 3 Pro", RemainingFraction: &depleted, ResetTime: &past},
		},
	}
	store := &fakeStore{
		latest:  map[string]*domain.QuotaSnapshot{"user@example.com": snap},
		history: []domain.QuotaSnapshot{*snap},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	// /latest serves the present: assumed refilled.
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/accounts/user@example.com/latest", nil))
	var latest struct {
		Models []struct {
			RemainingFraction float64 `json:"remaining_fraction"`
			AssumedRefilled   bool    `json:"assumed_refilled"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &latest); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(latest.Models) != 1 || !latest.Models[0].AssumedRefilled ||
		latest.Models[0].RemainingFraction != 1.0 {
		t.Fatalf("latest models = %+v, want assumed refilled at 1.0", latest.Models)
	}

	// /snapshots serves history: observed values, no reinterpretation.
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/accounts/user@example.com/snapshots", nil))
	var hist struct {
		Snapshots []struct {
			Models []struct {
				RemainingFraction float64 `json:"remaining_fraction"`
				AssumedRefilled   bool    `json:"assumed_refilled"`
			} `json:"models"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &hist); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(hist.Snapshots) != 1 || len(hist.Snapshots[0].Models) != 1 {
		t.Fatalf("history = %+v, want one snapshot with one model", hist.Snapshots)
	}
	if hist.Snapshots[0].Models[0].AssumedRefilled ||
		hist.Snapshots[0].Models[0].RemainingFraction != depleted {
		t.Fatalf("history model = %+v, want observed value untouched", hist.Snapshots[0].Models[0])
	}
}

func TestModelsLatestAssumesRefill(t *testing.T) {
	pastStr := fixedTime(-time.Hour).UTC().Format(time.RFC3339)
	depleted := 0.05
	store := &fakeStore{
		models: []domain.ModelQuotaAggregate{
			{Label: "Gemini 3 Pro", Email: "user@example.com",
				RemainingFraction: &depleted, IsExhausted: true, ResetTime: &pastStr},
		},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/models/latest", nil))

	var body struct {
		Models []struct {
			RemainingFraction float64 `json:"remaining_fraction"`
			IsExhausted       bool    `json:"is_exhausted"`
			AssumedRefilled   bool    `json:"assumed_refilled"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(body.Models))
	}
	m := body.Models[0]
	if !m.AssumedRefilled || m.RemainingFraction != 1.0 || m.IsExhausted {
		t.Fatalf("model = %+v, want assumed refilled", m)
	}
}

func TestAccountModelsCurrentEndpoint(t *testing.T) {
	frac := 0.42
	futureStr := fixedTime(time.Hour).UTC().Format(time.RFC3339)
	store := &fakeStore{
		currentModels: map[string][]domain.ModelQuotaAggregate{
			"user@example.com": {
				{Label: "Claude Sonnet", ModelID: "claude-sonnet", Email: "user@example.com",
					RemainingFraction: &frac, ResetTime: &futureStr},
			},
		},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/accounts/user@example.com/models/current", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Email  string `json:"email"`
		Models []struct {
			Label             string  `json:"label"`
			RemainingFraction float64 `json:"remaining_fraction"`
			AssumedRefilled   bool    `json:"assumed_refilled"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.Email != "user@example.com" {
		t.Fatalf("email = %q, want user@example.com", body.Email)
	}
	if len(body.Models) != 1 || body.Models[0].RemainingFraction != frac ||
		body.Models[0].AssumedRefilled {
		t.Fatalf("models = %+v, want one unmodified row", body.Models)
	}
}

func TestAccountModelsCurrentEmpty(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/accounts/none@example.com/models/current", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Models []json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.Models == nil || len(body.Models) != 0 {
		t.Fatalf("models = %v, want empty non-null array", body.Models)
	}
}

func TestBreakdownAssumesRefillForElapsedCycle(t *testing.T) {
	past := fixedTime(-time.Hour)
	future := fixedTime(time.Hour)
	cur := 0.1
	start := 0.9
	consumed := 0.8
	store := &fakeStore{
		breakdown: []domain.BreakdownRow{
			{Email: "a@example.com", Label: "Gemini 3 Pro", ModelID: "g3p",
				CurrentFraction: &cur, StartingFraction: &start, Consumed: &consumed,
				ResetTime: &past},
			{Email: "a@example.com", Label: "Claude Sonnet", ModelID: "cs",
				CurrentFraction: &cur, StartingFraction: &start, Consumed: &consumed,
				ResetTime: &future},
		},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/analytics/breakdown", nil))

	var body struct {
		Rows []struct {
			Label           string   `json:"label"`
			CurrentFraction float64  `json:"current_fraction"`
			Consumed        *float64 `json:"consumed"`
			AssumedRefilled bool     `json:"assumed_refilled"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(body.Rows))
	}
	for _, row := range body.Rows {
		switch row.Label {
		case "Gemini 3 Pro":
			if !row.AssumedRefilled || row.CurrentFraction != 1.0 || row.Consumed != nil {
				t.Fatalf("elapsed-cycle row = %+v, want assumed full without consumption", row)
			}
		case "Claude Sonnet":
			if row.AssumedRefilled || row.CurrentFraction != cur || row.Consumed == nil {
				t.Fatalf("active-cycle row = %+v, want untouched", row)
			}
		}
	}
}
