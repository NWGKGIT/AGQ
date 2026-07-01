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

func f64(v float64) *float64 { return &v }

func day(y int, m time.Month, d, hour int) time.Time {
	return time.Date(y, m, d, hour, 0, 0, 0, time.UTC)
}

func doGet(t *testing.T, srv *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestAnalyticsTimeseriesGroupsByDayAndProvider(t *testing.T) {
	store := &fakeStore{fractionSamples: []domain.FractionSample{
		{CapturedAt: day(2026, 6, 29, 8), Label: "Gemini 2.5 Pro", RemainingFraction: f64(0.4)},
		{CapturedAt: day(2026, 6, 29, 9), Label: "Gemini 2.5 Pro", RemainingFraction: f64(0.6)},
		{CapturedAt: day(2026, 6, 29, 10), Label: "Claude Sonnet", RemainingFraction: f64(0.2)},
		{CapturedAt: day(2026, 6, 30, 10), Label: "Claude Opus", RemainingFraction: f64(0.8)},
		{CapturedAt: day(2026, 6, 30, 11), Label: "Mystery Model", RemainingFraction: f64(0.9)},
	}}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/analytics/timeseries?range=7d&agg=avg")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Range string `json:"range"`
		Agg   string `json:"agg"`
		Days  []struct {
			Date      string              `json:"date"`
			Providers map[string]*float64 `json:"providers"`
		} `json:"days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.Range != "7d" || body.Agg != "avg" {
		t.Fatalf("range/agg = %q/%q", body.Range, body.Agg)
	}
	if len(body.Days) != 2 {
		t.Fatalf("len(days) = %d, want 2", len(body.Days))
	}
	if body.Days[0].Date != "2026-06-29" || body.Days[1].Date != "2026-06-30" {
		t.Fatalf("days ordering = %s then %s", body.Days[0].Date, body.Days[1].Date)
	}

	first := body.Days[0].Providers
	if first[ProviderGemini] == nil || *first[ProviderGemini] != 0.5 {
		t.Fatalf("day0 Gemini = %v, want averaged 0.5", first[ProviderGemini])
	}
	if first[ProviderAnthropic] == nil || *first[ProviderAnthropic] != 0.2 {
		t.Fatalf("day0 Anthropic = %v, want 0.2", first[ProviderAnthropic])
	}
	if _, ok := first[ProviderOpenAI]; !ok {
		t.Fatal("day0 missing OpenAI key; providers must be consistently shaped")
	}
	if first[ProviderOpenAI] != nil {
		t.Fatalf("day0 OpenAI = %v, want null", *first[ProviderOpenAI])
	}

	// Day two has only an Anthropic sample plus an unclassifiable one; the
	// unknown model is dropped but the day still appears via Anthropic.
	second := body.Days[1].Providers
	if second[ProviderGemini] != nil {
		t.Fatalf("day1 Gemini = %v, want null", *second[ProviderGemini])
	}
	if second[ProviderAnthropic] == nil || *second[ProviderAnthropic] != 0.8 {
		t.Fatalf("day1 Anthropic = %v, want 0.8", second[ProviderAnthropic])
	}
}

func TestAnalyticsTimeseriesMinAggregation(t *testing.T) {
	store := &fakeStore{fractionSamples: []domain.FractionSample{
		{CapturedAt: day(2026, 6, 29, 8), Label: "Gemini 2.5 Pro", RemainingFraction: f64(0.4)},
		{CapturedAt: day(2026, 6, 29, 9), Label: "Gemini 2.5 Pro", RemainingFraction: f64(0.6)},
	}}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/analytics/timeseries?agg=min")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Days []struct {
			Providers map[string]*float64 `json:"providers"`
		} `json:"days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Days) != 1 {
		t.Fatalf("len(days) = %d, want 1", len(body.Days))
	}
	if g := body.Days[0].Providers[ProviderGemini]; g == nil || *g != 0.4 {
		t.Fatalf("Gemini min = %v, want 0.4", g)
	}
}

func TestAnalyticsTimeseriesEmptyAndInvalid(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/analytics/timeseries")
	if rec.Code != http.StatusOK {
		t.Fatalf("empty status = %d", rec.Code)
	}
	var body struct {
		Days []json.RawMessage `json:"days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Days) != 0 {
		t.Fatalf("len(days) = %d, want 0 for no data", len(body.Days))
	}

	for _, path := range []string{
		"/api/analytics/timeseries?range=90d",
		"/api/analytics/timeseries?agg=median",
	} {
		if rec := doGet(t, srv, path); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", path, rec.Code)
		}
	}
}

func TestAnalyticsBreakdownSortsByConsumedDesc(t *testing.T) {
	reset := day(2026, 7, 2, 0)
	store := &fakeStore{breakdown: []domain.BreakdownRow{
		{Email: "a@e", Label: "Low", ModelID: "low", CurrentFraction: f64(0.7),
			StartingFraction: f64(0.8), Consumed: f64(0.1), ResetTime: &reset},
		{Email: "a@e", Label: "High", ModelID: "high", CurrentFraction: f64(0.2),
			StartingFraction: f64(0.7), Consumed: f64(0.5), ResetTime: &reset},
		{Email: "a@e", Label: "Unknown", ModelID: "unk", CurrentFraction: f64(0.9)},
	}}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/analytics/breakdown")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Rows []struct {
			Label            string   `json:"label"`
			CurrentFraction  *float64 `json:"current_fraction"`
			StartingFraction *float64 `json:"starting_fraction"`
			Consumed         *float64 `json:"consumed"`
			ResetTime        *string  `json:"reset_time"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(body.Rows))
	}
	if body.Rows[0].Label != "High" || body.Rows[1].Label != "Low" {
		t.Fatalf("order = %s, %s; want High, Low", body.Rows[0].Label, body.Rows[1].Label)
	}
	// Row without a computed consumption sorts last and omits starting/consumed.
	last := body.Rows[2]
	if last.Label != "Unknown" {
		t.Fatalf("last row = %s, want Unknown", last.Label)
	}
	if last.StartingFraction != nil || last.Consumed != nil {
		t.Fatalf("insufficient-data row leaked starting/consumed: %+v", last)
	}
	if last.CurrentFraction == nil || *last.CurrentFraction != 0.9 {
		t.Fatalf("current fraction = %v, want 0.9", last.CurrentFraction)
	}
}

func TestAnalyticsBreakdownEmpty(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{})
	rec := doGet(t, srv, "/api/analytics/breakdown")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Rows []json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Rows) != 0 {
		t.Fatalf("len(rows) = %d, want 0", len(body.Rows))
	}
}

func TestAnalyticsStatsReturnsFigures(t *testing.T) {
	store := &fakeStore{stats: domain.AnalyticsStats{
		TotalPollsThisWeek:   1440,
		MostDepletedModel:    &domain.DepletedModel{Email: "a@e", Label: "Claude Opus", RemainingFraction: 0.05},
		AccountMostRemaining: &domain.AccountRemaining{Email: "b@e", RemainingFraction: 0.9},
		NextReset:            &domain.NextReset{Email: "a@e", Label: "Gemini", ResetTime: "2026-07-02T00:00:00Z"},
	}}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/analytics/stats")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		TotalPolls int64 `json:"total_polls_this_week"`
		Depleted   *struct {
			Label             string  `json:"label"`
			RemainingFraction float64 `json:"remaining_fraction"`
		} `json:"most_depleted_model"`
		MostRemaining *struct {
			Email string `json:"email"`
		} `json:"account_most_remaining"`
		NextReset *struct {
			ResetTime string `json:"reset_time"`
		} `json:"next_reset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.TotalPolls != 1440 {
		t.Fatalf("total polls = %d, want 1440", body.TotalPolls)
	}
	if body.Depleted == nil || body.Depleted.Label != "Claude Opus" {
		t.Fatalf("most depleted = %+v", body.Depleted)
	}
	if body.MostRemaining == nil || body.MostRemaining.Email != "b@e" {
		t.Fatalf("account most remaining = %+v", body.MostRemaining)
	}
	if body.NextReset == nil || body.NextReset.ResetTime != "2026-07-02T00:00:00Z" {
		t.Fatalf("next reset = %+v", body.NextReset)
	}
}

func TestAnalyticsStatsEmptyNulls(t *testing.T) {
	store := &fakeStore{stats: domain.AnalyticsStats{TotalPollsThisWeek: 0}}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/analytics/stats")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	for _, key := range []string{"most_depleted_model", "account_most_remaining", "next_reset"} {
		if got := string(body[key]); got != "null" {
			t.Fatalf("%s = %s, want null", key, got)
		}
	}
}

func TestAccountSparklinesGroupsPointsAscending(t *testing.T) {
	store := &fakeStore{accountSamples: map[string][]domain.FractionSample{
		"u@e": {
			{CapturedAt: day(2026, 6, 30, 8), Label: "Claude Sonnet", ModelID: "claude-sonnet-4", RemainingFraction: f64(0.5)},
			{CapturedAt: day(2026, 6, 30, 9), Label: "Claude Sonnet", ModelID: "claude-sonnet-4", RemainingFraction: nil},
			{CapturedAt: day(2026, 6, 30, 10), Label: "Claude Sonnet", ModelID: "claude-sonnet-4", RemainingFraction: f64(0.3)},
			{CapturedAt: day(2026, 6, 30, 8), Label: "Gemini", ModelID: "gemini-2.5", RemainingFraction: f64(0.9)},
		},
	}}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/accounts/u@e/sparklines")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Email  string `json:"email"`
		Models []struct {
			Label  string `json:"label"`
			Points []struct {
				CapturedAt        time.Time `json:"captured_at"`
				RemainingFraction *float64  `json:"remaining_fraction"`
			} `json:"points"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if body.Email != "u@e" {
		t.Fatalf("email = %q", body.Email)
	}
	if len(body.Models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(body.Models))
	}
	// Models sorted by label: Claude Sonnet before Gemini.
	sonnet := body.Models[0]
	if sonnet.Label != "Claude Sonnet" || len(sonnet.Points) != 3 {
		t.Fatalf("first model = %s with %d points", sonnet.Label, len(sonnet.Points))
	}
	if sonnet.Points[1].RemainingFraction != nil {
		t.Fatal("middle point should preserve null fraction for gap rendering")
	}
	if !sonnet.Points[0].CapturedAt.Before(sonnet.Points[2].CapturedAt) {
		t.Fatal("points must be ascending by time")
	}
}

func TestAccountSparklinesUnknownEmail(t *testing.T) {
	srv := New(&fakeStore{}, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))
	rec := doGet(t, srv, "/api/accounts/nobody@e/sparklines")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Models []json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(body.Models) != 0 {
		t.Fatalf("len(models) = %d, want 0", len(body.Models))
	}
}

func TestAccountTimelineInfersSessionsFromGaps(t *testing.T) {
	base := day(2026, 6, 30, 8)
	store := &fakeStore{
		snapshotRefs: map[string][]domain.SnapshotRef{
			"u@e": {
				{ID: 1, CapturedAt: base},
				{ID: 2, CapturedAt: base.Add(time.Minute)},
				{ID: 3, CapturedAt: base.Add(30 * time.Minute)},
			},
		},
		snapshotModels: map[int64][]domain.ModelQuota{
			1: {{Label: "Claude Sonnet", RemainingFraction: f64(0.5)}},
			2: {{Label: "Claude Sonnet", RemainingFraction: f64(0.4)}},
			3: {{Label: "Gemini", RemainingFraction: f64(0.9)}},
		},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/accounts/u@e/timeline")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []struct {
			Type  string              `json:"type"`
			At    time.Time           `json:"at"`
			Quota map[string]*float64 `json:"quota"`
		} `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	// login (id1), logout (id2, before the 29-minute gap), login (id3).
	if len(body.Events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(body.Events))
	}
	wantTypes := []string{"login", "logout", "login"}
	for i, want := range wantTypes {
		if body.Events[i].Type != want {
			t.Fatalf("event[%d] type = %s, want %s", i, body.Events[i].Type, want)
		}
	}
	if !body.Events[0].At.Before(body.Events[2].At) {
		t.Fatal("events must be ascending by time")
	}
	// Boundary quota grouped by provider, complete across all providers.
	logout := body.Events[1].Quota
	if logout[ProviderAnthropic] == nil || *logout[ProviderAnthropic] != 0.4 {
		t.Fatalf("logout Anthropic = %v, want 0.4", logout[ProviderAnthropic])
	}
	if _, ok := logout[ProviderOpenAI]; !ok {
		t.Fatal("quota summary must include every provider key")
	}
	if body.Events[2].Quota[ProviderGemini] == nil || *body.Events[2].Quota[ProviderGemini] != 0.9 {
		t.Fatalf("second login Gemini = %v, want 0.9", body.Events[2].Quota[ProviderGemini])
	}
}

func TestAccountTimelineSingleSnapshotAndEmpty(t *testing.T) {
	store := &fakeStore{
		snapshotRefs: map[string][]domain.SnapshotRef{
			"one@e": {{ID: 1, CapturedAt: day(2026, 6, 30, 8)}},
		},
		snapshotModels: map[int64][]domain.ModelQuota{
			1: {{Label: "Claude Sonnet", RemainingFraction: f64(0.5)}},
		},
	}
	srv := New(store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))

	rec := doGet(t, srv, "/api/accounts/one@e/timeline")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Events []struct {
			Type string `json:"type"`
		} `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	// A lone snapshot is a login with no inferred logout (session may be live).
	if len(body.Events) != 1 || body.Events[0].Type != "login" {
		t.Fatalf("events = %+v, want single login", body.Events)
	}

	// Unknown email yields an empty, non-null events slice.
	rec = doGet(t, srv, "/api/accounts/nobody@e/timeline")
	var empty struct {
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(empty.Events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(empty.Events))
	}
}

func TestAnalyticsEndpointsPropagateStoreErrors(t *testing.T) {
	cases := []struct {
		name  string
		store *fakeStore
		path  string
	}{
		{"timeseries", &fakeStore{samplesErr: errors.New("boom")}, "/api/analytics/timeseries"},
		{"breakdown", &fakeStore{breakdownErr: errors.New("boom")}, "/api/analytics/breakdown"},
		{"stats", &fakeStore{statsErr: errors.New("boom")}, "/api/analytics/stats"},
		{"sparklines", &fakeStore{samplesErr: errors.New("boom")}, "/api/accounts/u@e/sparklines"},
		{"timeline-refs", &fakeStore{refsErr: errors.New("boom")}, "/api/accounts/u@e/timeline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := New(tc.store, fakeStatus{}, WithClock(func() time.Time { return fixedTime(0) }))
			if rec := doGet(t, srv, tc.path); rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
		})
	}
}
