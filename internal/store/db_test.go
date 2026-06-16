package store

import (
	"path/filepath"
	"testing"
	"time"

	"agq-daemon/internal/domain"
)

func TestSaveAndReadSnapshot(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	reset := time.Date(2026, 7, 1, 13, 0, 0, 0, time.UTC)
	frac := 0.5
	pct := 50.0
	until := int64(time.Hour / time.Millisecond)
	snapshot := domain.QuotaSnapshot{
		Email:                  "user@example.com",
		PlanName:               "Pro",
		CapturedAt:             time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		PromptCreditsAvailable: 90,
		PromptCreditsMonthly:   100,
		FlowCreditsAvailable:   40,
		FlowCreditsMonthly:     50,
		RawJSON:                `{"ok":true}`,
		Models: []domain.ModelQuota{{
			Label:             "Model A",
			ModelID:           "model-a",
			RemainingFraction: &frac,
			RemainingPct:      &pct,
			IsExhausted:       false,
			ResetTime:         &reset,
			PoolResetTime:     &reset,
			TimeUntilResetMs:  &until,
		}},
	}

	if err := db.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("SaveSnapshot returned error: %v", err)
	}

	accounts, err := db.GetAllAccounts()
	if err != nil {
		t.Fatalf("GetAllAccounts returned error: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("len(accounts) = %d, want 1", len(accounts))
	}
	if accounts[0].Email != snapshot.Email || accounts[0].PlanName != snapshot.PlanName {
		t.Fatalf("account = %#v", accounts[0])
	}

	latest, err := db.GetLatestSnapshot(snapshot.Email)
	if err != nil {
		t.Fatalf("GetLatestSnapshot returned error: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatestSnapshot returned nil snapshot")
	}
	if latest.Email != snapshot.Email || latest.PlanName != snapshot.PlanName {
		t.Fatalf("latest identity = %#v", latest)
	}
	if !latest.CapturedAt.Equal(snapshot.CapturedAt) {
		t.Fatalf("CapturedAt = %s, want %s", latest.CapturedAt, snapshot.CapturedAt)
	}
	if len(latest.Models) != 1 {
		t.Fatalf("len(latest.Models) = %d, want 1", len(latest.Models))
	}
	if latest.Models[0].PoolResetTime == nil || !latest.Models[0].PoolResetTime.Equal(reset) {
		t.Fatalf("PoolResetTime = %v, want %s", latest.Models[0].PoolResetTime, reset)
	}

	aggregates, err := db.GetLatestModelQuotas()
	if err != nil {
		t.Fatalf("GetLatestModelQuotas returned error: %v", err)
	}
	if len(aggregates) != 1 {
		t.Fatalf("len(aggregates) = %d, want 1", len(aggregates))
	}
	if aggregates[0].Email != snapshot.Email || aggregates[0].Label != "Model A" {
		t.Fatalf("aggregate = %#v", aggregates[0])
	}
}

func TestGetSnapshotHistoryDefaultsAndOrdersNewestFirst(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	old := domain.QuotaSnapshot{
		Email:      "user@example.com",
		PlanName:   "Pro",
		CapturedAt: base,
		RawJSON:    `{"old":true}`,
	}
	newer := old
	newer.CapturedAt = old.CapturedAt.Add(time.Minute)
	newer.RawJSON = `{"new":true}`

	if err := db.SaveSnapshot(old); err != nil {
		t.Fatalf("SaveSnapshot old returned error: %v", err)
	}
	if err := db.SaveSnapshot(newer); err != nil {
		t.Fatalf("SaveSnapshot newer returned error: %v", err)
	}

	history, err := db.GetSnapshotHistory("user@example.com", 0, time.Time{})
	if err != nil {
		t.Fatalf("GetSnapshotHistory returned error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if !history[0].CapturedAt.Equal(newer.CapturedAt) || !history[1].CapturedAt.Equal(old.CapturedAt) {
		t.Fatalf("history order = %s then %s", history[0].CapturedAt, history[1].CapturedAt)
	}

	beforeNewer := newer.CapturedAt
	history, err = db.GetSnapshotHistory("user@example.com", 50, beforeNewer)
	if err != nil {
		t.Fatalf("GetSnapshotHistory before returned error: %v", err)
	}
	if len(history) != 1 || !history[0].CapturedAt.Equal(old.CapturedAt) {
		t.Fatalf("filtered history = %#v", history)
	}
}

func TestBreakdownComputesConsumedWithinCycle(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	reset := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)

	// Two snapshots in the same cycle (same reset time) plus a model that never
	// carries a reset time, so it must not get a starting/consumed value.
	saveModels(t, db, "a@example.com", base, []domain.ModelQuota{
		modelWithReset("Claude Sonnet", "claude", 0.8, reset),
		modelNoReset("Credits", "credits", 0.5),
	})
	saveModels(t, db, "a@example.com", base.Add(time.Minute), []domain.ModelQuota{
		modelWithReset("Claude Sonnet", "claude", 0.3, reset),
		modelNoReset("Credits", "credits", 0.4),
	})

	rows, err := db.GetBreakdown()
	if err != nil {
		t.Fatalf("GetBreakdown returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}

	byModel := map[string]domain.BreakdownRow{}
	for _, r := range rows {
		byModel[r.ModelID] = r
	}

	claude := byModel["claude"]
	if claude.CurrentFraction == nil || *claude.CurrentFraction != 0.3 {
		t.Fatalf("claude current = %v, want 0.3", claude.CurrentFraction)
	}
	if claude.StartingFraction == nil || *claude.StartingFraction != 0.8 {
		t.Fatalf("claude starting = %v, want 0.8", claude.StartingFraction)
	}
	if claude.Consumed == nil || !floatNear(*claude.Consumed, 0.5) {
		t.Fatalf("claude consumed = %v, want 0.5", claude.Consumed)
	}

	credits := byModel["credits"]
	if credits.StartingFraction != nil || credits.Consumed != nil {
		t.Fatalf("model without reset time must omit starting/consumed: %+v", credits)
	}
	if credits.CurrentFraction == nil || *credits.CurrentFraction != 0.4 {
		t.Fatalf("credits current = %v, want 0.4", credits.CurrentFraction)
	}
}

func TestBreakdownSingleSnapshotOmitsStarting(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	reset := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	saveModels(t, db, "a@example.com", time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
		[]domain.ModelQuota{modelWithReset("Claude Sonnet", "claude", 0.6, reset)})

	rows, err := db.GetBreakdown()
	if err != nil {
		t.Fatalf("GetBreakdown returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	// The only snapshot is both first and current, so there is nothing to
	// diff against: report current only, no guessed consumption.
	if rows[0].StartingFraction != nil || rows[0].Consumed != nil {
		t.Fatalf("single snapshot leaked starting/consumed: %+v", rows[0])
	}
}

func TestAnalyticsStatsAndSamples(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	reset := now.Add(time.Hour)

	saveModels(t, db, "a@example.com", now.Add(-2*time.Minute), []domain.ModelQuota{
		modelWithReset("Claude Opus", "opus", 0.05, reset),
		{Label: "Exhausted", ModelID: "exhausted"}, // nil fraction
	})
	saveModels(t, db, "b@example.com", now.Add(-time.Minute), []domain.ModelQuota{
		modelNoReset("Gemini", "gemini", 0.9),
	})

	stats, err := db.GetAnalyticsStats(now)
	if err != nil {
		t.Fatalf("GetAnalyticsStats returned error: %v", err)
	}
	if stats.TotalPollsThisWeek != 2 {
		t.Fatalf("total polls = %d, want 2", stats.TotalPollsThisWeek)
	}
	if stats.MostDepletedModel == nil || stats.MostDepletedModel.Label != "Claude Opus" {
		t.Fatalf("most depleted = %+v, want Claude Opus", stats.MostDepletedModel)
	}
	if stats.AccountMostRemaining == nil || stats.AccountMostRemaining.Email != "b@example.com" {
		t.Fatalf("account most remaining = %+v, want b@example.com", stats.AccountMostRemaining)
	}
	if stats.NextReset == nil || stats.NextReset.Label != "Claude Opus" {
		t.Fatalf("next reset = %+v, want Claude Opus reset", stats.NextReset)
	}

	// Time-series samples exclude null fractions across all accounts.
	samples, err := db.GetFractionSamplesSince(now.Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("GetFractionSamplesSince returned error: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("len(samples) = %d, want 2 (null fraction excluded)", len(samples))
	}
}

func TestAccountFractionSamplesIncludeNulls(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	saveModels(t, db, "a@example.com", base,
		[]domain.ModelQuota{modelNoReset("Claude Sonnet", "claude", 0.5)})
	saveModels(t, db, "a@example.com", base.Add(time.Minute),
		[]domain.ModelQuota{{Label: "Claude Sonnet", ModelID: "claude"}}) // nil fraction

	samples, err := db.GetAccountFractionSamplesSince("a@example.com", base.Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetAccountFractionSamplesSince returned error: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("len(samples) = %d, want 2", len(samples))
	}
	// Ordered by model then ascending time; the null-fraction point is retained.
	if !samples[0].CapturedAt.Before(samples[1].CapturedAt) {
		t.Fatal("samples must be ascending by time")
	}
	if samples[1].RemainingFraction != nil {
		t.Fatal("null fraction must be preserved for gap rendering")
	}
}

func TestSnapshotRefsAndModels(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	saveModels(t, db, "a@example.com", base,
		[]domain.ModelQuota{modelNoReset("Claude Sonnet", "claude", 0.5)})
	saveModels(t, db, "a@example.com", base.Add(time.Minute),
		[]domain.ModelQuota{modelNoReset("Gemini", "gemini", 0.9)})

	refs, err := db.GetSnapshotRefsSince("a@example.com", base.Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetSnapshotRefsSince returned error: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("len(refs) = %d, want 2", len(refs))
	}
	if !refs[0].CapturedAt.Before(refs[1].CapturedAt) {
		t.Fatal("refs must be ascending by time")
	}

	models, err := db.GetSnapshotModels(refs[1].ID)
	if err != nil {
		t.Fatalf("GetSnapshotModels returned error: %v", err)
	}
	if len(models) != 1 || models[0].ModelID != "gemini" {
		t.Fatalf("models = %+v, want single gemini", models)
	}
}

func saveModels(t *testing.T, db *DB, email string, capturedAt time.Time, models []domain.ModelQuota) {
	t.Helper()
	if err := db.SaveSnapshot(domain.QuotaSnapshot{
		Email:      email,
		PlanName:   "Pro",
		CapturedAt: capturedAt,
		RawJSON:    `{}`,
		Models:     models,
	}); err != nil {
		t.Fatalf("SaveSnapshot returned error: %v", err)
	}
}

func modelWithReset(label, id string, frac float64, reset time.Time) domain.ModelQuota {
	m := modelNoReset(label, id, frac)
	m.ResetTime = &reset
	m.PoolResetTime = &reset
	return m
}

func modelNoReset(label, id string, frac float64) domain.ModelQuota {
	f := frac
	pct := frac * 100
	return domain.ModelQuota{
		Label:             label,
		ModelID:           id,
		RemainingFraction: &f,
		RemainingPct:      &pct,
		IsExhausted:       frac == 0,
	}
}

func floatNear(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-9
}

func openTestDB(t *testing.T) *DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "agq.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	return db
}
