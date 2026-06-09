package domain

import "time"

// DaemonState represents whether the daemon is actively polling or idle.
type DaemonState string

const (
	// StateIdle means no authenticated Antigravity language server is present.
	StateIdle DaemonState = "IDLE"

	// StateActive means at least one authenticated language server is being polled.
	StateActive DaemonState = "ACTIVE"
)

// ProcessInfo holds connection details extracted from a detected language server
// process. Fields sourced from cmdline flags are retained for diagnostics.
type ProcessInfo struct {
	Pid                int
	CsrfToken          string
	ActivePort         int
	Scheme             string
	ExtensionPort      int
	ExtensionCsrfToken string
	HttpsServerPort    int
	LspPort            int
	WorkspaceID        string
	CloudEndpoint      string
}

// ModelQuota is a parsed model entry stored per snapshot.
//
// PoolResetTime mirrors ResetTime because models sharing the same reset value
// belong to the same quota pool and can be grouped by the dashboard.
type ModelQuota struct {
	Label             string     `json:"label"`
	ModelID           string     `json:"model_id"`
	RemainingFraction *float64   `json:"remaining_fraction"`
	RemainingPct      *float64   `json:"remaining_pct"`
	IsExhausted       bool       `json:"is_exhausted"`
	ResetTime         *time.Time `json:"reset_time,omitempty"`
	PoolResetTime     *time.Time `json:"pool_reset_time,omitempty"`
	TimeUntilResetMs  *int64     `json:"time_until_reset_ms,omitempty"`
}

// QuotaSnapshot is the fully parsed result of one successful poll, ready for
// persistence. RawJSON preserves the original response for future use.
type QuotaSnapshot struct {
	Email                  string
	PlanName               string
	CapturedAt             time.Time
	PromptCreditsAvailable int64
	PromptCreditsMonthly   int64
	FlowCreditsAvailable   int64
	FlowCreditsMonthly     int64
	RawJSON                string
	Models                 []ModelQuota
}

// AccountSummary describes one account known to the local store.
type AccountSummary struct {
	ID             int64          `json:"id"`
	Email          string         `json:"email"`
	PlanName       string         `json:"plan_name"`
	FirstSeen      time.Time      `json:"first_seen"`
	LastSeen       time.Time      `json:"last_seen"`
	LatestSnapshot *QuotaSnapshot `json:"latest_snapshot,omitempty"`
}

// DaemonStatus is returned by GET /api/status.
type DaemonStatus struct {
	State      DaemonState `json:"state"`
	Emails     []string    `json:"emails,omitempty"`
	Uptime     string      `json:"uptime"`
	StartedAt  time.Time   `json:"started_at"`
	LastPollAt *time.Time  `json:"last_poll_at,omitempty"`
	NextPollAt *time.Time  `json:"next_poll_at,omitempty"`
}

// FractionSample is one model's remaining fraction at a single capture time.
// It is the raw input for the analytics time-series and sparkline endpoints.
// RemainingFraction is nil when the model reported no quota value at that
// capture, which the sparkline endpoint surfaces as a gap.
type FractionSample struct {
	CapturedAt        time.Time
	Label             string
	ModelID           string
	RemainingFraction *float64
}

// BreakdownRow describes one account/model pair for the analytics breakdown
// table. StartingFraction and Consumed are nil when the current quota cycle
// does not have enough data to compute them (no reset time, or only a single
// snapshot since the last reset).
type BreakdownRow struct {
	Email            string
	Label            string
	ModelID          string
	CurrentFraction  *float64
	StartingFraction *float64
	Consumed         *float64
	ResetTime        *time.Time
}

// SnapshotRef is a lightweight snapshot identifier used to infer login/logout
// session boundaries without loading full snapshot bodies.
type SnapshotRef struct {
	ID         int64
	CapturedAt time.Time
}

// AnalyticsStats holds the four headline figures for GET /api/analytics/stats.
// The nested pointers are nil when no data supports the figure, so the frontend
// can render each card directly without further computation.
type AnalyticsStats struct {
	TotalPollsThisWeek   int64             `json:"total_polls_this_week"`
	MostDepletedModel    *DepletedModel    `json:"most_depleted_model"`
	AccountMostRemaining *AccountRemaining `json:"account_most_remaining"`
	NextReset            *NextReset        `json:"next_reset"`
}

// DepletedModel names the model with the lowest remaining fraction across the
// latest snapshot of each account.
type DepletedModel struct {
	Email             string  `json:"email"`
	Label             string  `json:"label"`
	RemainingFraction float64 `json:"remaining_fraction"`
}

// AccountRemaining names the account whose latest snapshot has the highest
// average remaining fraction across its models.
type AccountRemaining struct {
	Email             string  `json:"email"`
	RemainingFraction float64 `json:"remaining_fraction"`
}

// NextReset names the soonest upcoming quota reset across all accounts.
type NextReset struct {
	Email     string `json:"email"`
	Label     string `json:"label"`
	ResetTime string `json:"reset_time"`
}

// ModelQuotaAggregate is returned by GET /api/models/latest.
type ModelQuotaAggregate struct {
	Label             string   `json:"label"`
	ModelID           string   `json:"model_id"`
	RemainingFraction *float64 `json:"remaining_fraction"`
	RemainingPct      *float64 `json:"remaining_pct"`
	IsExhausted       bool     `json:"is_exhausted"`
	ResetTime         *string  `json:"reset_time,omitempty"`
	PoolResetTime     *string  `json:"pool_reset_time,omitempty"`
	Email             string   `json:"email"`
	CapturedAt        string   `json:"captured_at"`
	StalenessSeconds  int64    `json:"staleness_seconds"`
}
