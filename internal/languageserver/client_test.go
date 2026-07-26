package languageserver

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseSnapshot(t *testing.T) {
	capturedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	body := []byte(`{
		"userStatus": {
			"email": "user@example.com",
			"userTier": {"name": "Pro"},
			"planStatus": {
				"availablePromptCredits": "123",
				"availableFlowCredits": 45,
				"planInfo": {
					"monthlyPromptCredits": 1000,
					"monthlyFlowCredits": "500"
				}
			},
			"cascadeModelConfigData": {
				"clientModelConfigs": [
					{
						"label": "Claude Sonnet",
						"modelOrAlias": {"model": "claude-sonnet-4"},
						"quotaInfo": {
							"remainingFraction": 0.25,
							"resetTime": "2026-07-01T13:00:00Z"
						}
					},
					{
						"label": "No quota",
						"modelOrAlias": {"model": "no-quota"},
						"quotaInfo": {}
					}
				]
			}
		}
	}`)

	snapshot, err := ParseSnapshot(body, capturedAt)
	if err != nil {
		t.Fatalf("ParseSnapshot returned error: %v", err)
	}

	if snapshot.Email != "user@example.com" {
		t.Fatalf("Email = %q, want user@example.com", snapshot.Email)
	}
	if snapshot.PlanName != "Pro" {
		t.Fatalf("PlanName = %q, want Pro", snapshot.PlanName)
	}
	if snapshot.PromptCreditsAvailable != 123 || snapshot.PromptCreditsMonthly != 1000 {
		t.Fatalf("prompt credits = %d/%d, want 123/1000", snapshot.PromptCreditsAvailable, snapshot.PromptCreditsMonthly)
	}
	if snapshot.FlowCreditsAvailable != 45 || snapshot.FlowCreditsMonthly != 500 {
		t.Fatalf("flow credits = %d/%d, want 45/500", snapshot.FlowCreditsAvailable, snapshot.FlowCreditsMonthly)
	}
	if len(snapshot.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(snapshot.Models))
	}

	first := snapshot.Models[0]
	if first.Label != "Claude Sonnet" || first.ModelID != "claude-sonnet-4" {
		t.Fatalf("first model = %#v", first)
	}
	if first.RemainingFraction == nil || *first.RemainingFraction != 0.25 {
		t.Fatalf("RemainingFraction = %v, want 0.25", first.RemainingFraction)
	}
	if first.RemainingPct == nil || *first.RemainingPct != 25 {
		t.Fatalf("RemainingPct = %v, want 25", first.RemainingPct)
	}
	if first.IsExhausted {
		t.Fatal("IsExhausted = true, want false")
	}
	if first.TimeUntilResetMs == nil || *first.TimeUntilResetMs != int64(time.Hour/time.Millisecond) {
		t.Fatalf("TimeUntilResetMs = %v, want one hour", first.TimeUntilResetMs)
	}

	second := snapshot.Models[1]
	if second.IsExhausted {
		t.Fatal("missing remainingFraction should be unknown, not exhausted")
	}
	if second.RemainingFraction != nil || second.RemainingPct != nil {
		t.Fatalf("missing quota produced percentage data: %#v", second)
	}

	if !json.Valid([]byte(snapshot.RawJSON)) {
		t.Fatal("RawJSON is not valid JSON")
	}
	if snapshot.RawJSON != "{}" {
		t.Fatalf("RawJSON = %q, want redacted object", snapshot.RawJSON)
	}
}

func TestParseSnapshotMissingFractionIsUnknownNotExhausted(t *testing.T) {
	body := []byte(`{
		"userStatus": {
			"email": "unknown@example.com",
			"cascadeModelConfigData": {
				"clientModelConfigs": [{
					"label": "Unknown model",
					"modelOrAlias": {"model": "unknown-model"},
					"quotaInfo": {}
				}]
			}
		}
	}`)

	snapshot, err := ParseSnapshot(body, time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(snapshot.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(snapshot.Models))
	}
	model := snapshot.Models[0]
	if model.RemainingFraction != nil {
		t.Fatalf("RemainingFraction = %v, want nil", *model.RemainingFraction)
	}
	if model.IsExhausted {
		t.Fatal("missing remaining fraction must not be classified as exhausted")
	}
}

func TestParseSnapshotRequiresEmail(t *testing.T) {
	_, err := ParseSnapshot([]byte(`{"userStatus":{"email":""}}`), time.Now())
	if err == nil {
		t.Fatal("ParseSnapshot returned nil error, want missing email error")
	}
	if !strings.Contains(err.Error(), "missing email") {
		t.Fatalf("error = %q, want missing email", err)
	}
}

func TestFlexInt64RejectsInvalidString(t *testing.T) {
	var value struct {
		Amount flexInt64 `json:"amount"`
	}
	err := json.Unmarshal([]byte(`{"amount":"not-a-number"}`), &value)
	if err == nil {
		t.Fatal("json.Unmarshal returned nil error, want parse failure")
	}
}

func TestReadLimitedBodyRejectsOversizedResponse(t *testing.T) {
	_, err := readLimitedBody(strings.NewReader(strings.Repeat("x", maxResponseBody+1)))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err = %v, want response size error", err)
	}
}
