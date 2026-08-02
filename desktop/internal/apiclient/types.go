package apiclient

// Response types mirror the daemon's JSON API exactly. Timestamps stay as
// RFC3339 strings so Wails generates clean TypeScript models; the frontend
// parses them where needed.

// Health is returned by GET /api/health.
type Health struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// DaemonStatus is returned by GET /api/status.
type DaemonStatus struct {
	State      string   `json:"state"`
	Emails     []string `json:"emails,omitempty"`
	Uptime     string   `json:"uptime"`
	StartedAt  string   `json:"started_at"`
	LastPollAt *string  `json:"last_poll_at,omitempty"`
	NextPollAt *string  `json:"next_poll_at,omitempty"`
}

// ModelQuota is one model entry inside a snapshot. AssumedRefilled marks
// values the daemon inferred after a reset passed without a fresh poll.
type ModelQuota struct {
	Label             string   `json:"label"`
	ModelID           string   `json:"model_id"`
	RemainingFraction *float64 `json:"remaining_fraction"`
	RemainingPct      *float64 `json:"remaining_pct"`
	IsExhausted       bool     `json:"is_exhausted"`
	ResetTime         *string  `json:"reset_time,omitempty"`
	PoolResetTime     *string  `json:"pool_reset_time,omitempty"`
	TimeUntilResetMs  *int64   `json:"time_until_reset_ms,omitempty"`
	AssumedRefilled   bool     `json:"assumed_refilled,omitempty"`
}

// Snapshot is one persisted quota snapshot.
type Snapshot struct {
	Email                  string       `json:"email"`
	PlanName               string       `json:"plan_name"`
	CapturedAt             string       `json:"captured_at"`
	StalenessSeconds       int64        `json:"staleness_seconds"`
	PromptCreditsAvailable int64        `json:"prompt_credits_available"`
	PromptCreditsMonthly   int64        `json:"prompt_credits_monthly"`
	FlowCreditsAvailable   int64        `json:"flow_credits_available"`
	FlowCreditsMonthly     int64        `json:"flow_credits_monthly"`
	Models                 []ModelQuota `json:"models"`
}

// Account is one account row, optionally joined with its newest snapshot.
type Account struct {
	ID             int64     `json:"id"`
	Email          string    `json:"email"`
	PlanName       string    `json:"plan_name"`
	FirstSeen      string    `json:"first_seen"`
	LastSeen       string    `json:"last_seen"`
	LatestSnapshot *Snapshot `json:"latest_snapshot,omitempty"`
}

// AccountsResponse is returned by GET /api/accounts.
type AccountsResponse struct {
	Accounts []Account `json:"accounts"`
}

// CurrentAccount is returned by GET /api/account/current.
type CurrentAccount struct {
	State       string    `json:"state"`
	IsLive      bool      `json:"is_live"`
	Email       string    `json:"email,omitempty"`
	Account     *Account  `json:"account,omitempty"`
	Accounts    []Account `json:"accounts"`
	LastPollAt  *string   `json:"last_poll_at,omitempty"`
	NextPollAt  *string   `json:"next_poll_at,omitempty"`
	LastAccount *Account  `json:"last_account,omitempty"`
	AsOf        string    `json:"as_of"`
}

// SnapshotsResponse is returned by GET /api/accounts/{email}/snapshots.
type SnapshotsResponse struct {
	Email     string     `json:"email"`
	Snapshots []Snapshot `json:"snapshots"`
}

// ModelAggregate is one row of GET /api/models/latest and
// GET /api/accounts/{email}/models/current.
type ModelAggregate struct {
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
	AssumedRefilled   bool     `json:"assumed_refilled,omitempty"`
}

// ModelsLatestResponse is returned by GET /api/models/latest.
type ModelsLatestResponse struct {
	Models []ModelAggregate `json:"models"`
}

// AccountModelsResponse is returned by GET /api/accounts/{email}/models/current.
type AccountModelsResponse struct {
	Email  string           `json:"email"`
	Models []ModelAggregate `json:"models"`
}

// TimeseriesDay carries the aggregated remaining fraction per provider for
// one calendar day. A provider key is always present; nil means no data.
type TimeseriesDay struct {
	Date      string              `json:"date"`
	Providers map[string]*float64 `json:"providers"`
}

// TimeseriesResponse is returned by GET /api/analytics/timeseries.
type TimeseriesResponse struct {
	Range string          `json:"range"`
	Agg   string          `json:"agg"`
	Days  []TimeseriesDay `json:"days"`
}

// BreakdownRow is one account x model row of GET /api/analytics/breakdown.
type BreakdownRow struct {
	Email            string   `json:"email"`
	Label            string   `json:"label"`
	ModelID          string   `json:"model_id"`
	CurrentFraction  *float64 `json:"current_fraction"`
	StartingFraction *float64 `json:"starting_fraction,omitempty"`
	Consumed         *float64 `json:"consumed,omitempty"`
	ResetTime        *string  `json:"reset_time,omitempty"`
	AssumedRefilled  bool     `json:"assumed_refilled,omitempty"`
}

// BreakdownResponse is returned by GET /api/analytics/breakdown.
type BreakdownResponse struct {
	Rows []BreakdownRow `json:"rows"`
}

// DepletedModel names the model with the lowest remaining fraction.
type DepletedModel struct {
	Email             string  `json:"email"`
	Label             string  `json:"label"`
	RemainingFraction float64 `json:"remaining_fraction"`
}

// AccountRemaining names the account with the most remaining quota.
type AccountRemaining struct {
	Email             string  `json:"email"`
	RemainingFraction float64 `json:"remaining_fraction"`
}

// NextReset names the soonest upcoming quota reset.
type NextReset struct {
	Email     string `json:"email"`
	Label     string `json:"label"`
	ResetTime string `json:"reset_time"`
}

// Stats is returned by GET /api/analytics/stats.
type Stats struct {
	TotalPollsThisWeek   int64             `json:"total_polls_this_week"`
	MostDepletedModel    *DepletedModel    `json:"most_depleted_model"`
	AccountMostRemaining *AccountRemaining `json:"account_most_remaining"`
	NextReset            *NextReset        `json:"next_reset"`
}

// SparklinePoint is one snapshot's remaining fraction for a model.
type SparklinePoint struct {
	CapturedAt        string   `json:"captured_at"`
	RemainingFraction *float64 `json:"remaining_fraction"`
}

// SparklineModel is a per-model 7-day series.
type SparklineModel struct {
	Label   string           `json:"label"`
	ModelID string           `json:"model_id"`
	Points  []SparklinePoint `json:"points"`
}

// SparklinesResponse is returned by GET /api/accounts/{email}/sparklines.
type SparklinesResponse struct {
	Email  string           `json:"email"`
	Models []SparklineModel `json:"models"`
}

// TimelineEvent is one inferred login/logout boundary.
type TimelineEvent struct {
	Type  string              `json:"type"`
	At    string              `json:"at"`
	Quota map[string]*float64 `json:"quota"`
}

// TimelineResponse is returned by GET /api/accounts/{email}/timeline.
type TimelineResponse struct {
	Email  string          `json:"email"`
	Events []TimelineEvent `json:"events"`
}
