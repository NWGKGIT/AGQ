package api

import (
	"time"

	"agq-daemon/internal/domain"
)

// applyAssumedRefill rewrites served model quotas whose reset time has already
// passed. The provider refills the pool server-side at the reset instant, so a
// stored fraction from before the reset no longer describes reality — showing
// it would surface a stale, depleted number indefinitely while no fresh poll
// is possible (account logged out, monitor down). Stored rows are never
// mutated; only the response is.
func applyAssumedRefill(models []domain.ModelQuota, now time.Time) []domain.ModelQuota {
	out := make([]domain.ModelQuota, len(models))
	copy(out, models)
	for i := range out {
		reset := out[i].PoolResetTime
		if reset == nil {
			reset = out[i].ResetTime
		}
		if reset == nil || !reset.Before(now) {
			continue
		}
		full := 1.0
		fullPct := 100.0
		out[i].RemainingFraction = &full
		out[i].RemainingPct = &fullPct
		out[i].IsExhausted = false
		out[i].AssumedRefilled = true
	}
	return out
}

// applyAssumedRefillAggregates is applyAssumedRefill for the aggregate rows
// served by /api/models/latest, where reset times are RFC3339 strings.
func applyAssumedRefillAggregates(models []domain.ModelQuotaAggregate, now time.Time) []domain.ModelQuotaAggregate {
	out := make([]domain.ModelQuotaAggregate, len(models))
	copy(out, models)
	for i := range out {
		resetStr := out[i].PoolResetTime
		if resetStr == nil {
			resetStr = out[i].ResetTime
		}
		if resetStr == nil {
			continue
		}
		reset, err := time.Parse(time.RFC3339, *resetStr)
		if err != nil || !reset.Before(now) {
			continue
		}
		full := 1.0
		fullPct := 100.0
		out[i].RemainingFraction = &full
		out[i].RemainingPct = &fullPct
		out[i].IsExhausted = false
		out[i].AssumedRefilled = true
	}
	return out
}
