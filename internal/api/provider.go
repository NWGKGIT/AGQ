package api

import (
	"strings"

	"agq-daemon/internal/domain"
)

// Provider names group models by the vendor that serves them. They are the
// buckets the Analytics dashboard renders quota against.
const (
	ProviderGemini    = "Gemini"
	ProviderAnthropic = "Anthropic"
	ProviderOpenAI    = "OpenAI"
)

// providerOrder is the stable render order shared by every provider-grouped
// response, so a day or event always exposes the same keys.
var providerOrder = []string{ProviderGemini, ProviderAnthropic, ProviderOpenAI}

// classifyProvider maps a model label to a provider by case-insensitive
// substring match. It is the single label-matching helper used by every
// endpoint that groups by provider. An empty string means the label matched no
// known provider and should be excluded from provider aggregates.
func classifyProvider(label string) string {
	l := strings.ToLower(label)
	switch {
	case strings.Contains(l, "gemini"):
		return ProviderGemini
	case strings.Contains(l, "claude"),
		strings.Contains(l, "anthropic"),
		strings.Contains(l, "sonnet"),
		strings.Contains(l, "opus"),
		strings.Contains(l, "haiku"):
		return ProviderAnthropic
	case strings.Contains(l, "gpt"),
		strings.Contains(l, "openai"),
		strings.Contains(l, "codex"):
		return ProviderOpenAI
	default:
		return ""
	}
}

// summarizeByProvider averages the remaining fraction of models within each
// provider. Every provider in providerOrder is present in the result; a
// provider with no classified, non-null model maps to nil so the frontend
// always receives a complete, consistently shaped quota summary.
func summarizeByProvider(models []domain.ModelQuota) map[string]*float64 {
	sums := make(map[string]float64)
	counts := make(map[string]int)
	for _, m := range models {
		provider := classifyProvider(m.Label)
		if provider == "" || m.RemainingFraction == nil {
			continue
		}
		sums[provider] += *m.RemainingFraction
		counts[provider]++
	}

	out := make(map[string]*float64, len(providerOrder))
	for _, provider := range providerOrder {
		if counts[provider] > 0 {
			avg := sums[provider] / float64(counts[provider])
			out[provider] = &avg
		} else {
			out[provider] = nil
		}
	}
	return out
}
