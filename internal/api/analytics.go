package api

import (
	"net/http"
	"sort"
	"time"

	"agq-daemon/internal/domain"
)

// SessionGapThreshold is the minimum gap between consecutive snapshots that is
// treated as a logout/login boundary in the timeline. It sits comfortably above
// the poller's five-minute failure backoff so a run of failed polls is not
// mistaken for the account signing out.
const SessionGapThreshold = 10 * time.Minute

// dateFormat is the calendar-day key used to bucket the time series.
const dateFormat = "2006-01-02"

type timeseriesDay struct {
	Date      string              `json:"date"`
	Providers map[string]*float64 `json:"providers"`
}

type timeseriesResponse struct {
	Range string          `json:"range"`
	Agg   string          `json:"agg"`
	Days  []timeseriesDay `json:"days"`
}

func (s *Server) analyticsTimeseriesHandler(w http.ResponseWriter, r *http.Request) {
	days := 7
	switch r.URL.Query().Get("range") {
	case "", "7d":
		days = 7
	case "30d":
		days = 30
	default:
		s.writeError(w, http.StatusBadRequest, "invalid range (want 7d or 30d)", nil)
		return
	}

	agg := r.URL.Query().Get("agg")
	switch agg {
	case "":
		agg = "avg"
	case "avg", "min":
	default:
		s.writeError(w, http.StatusBadRequest, "invalid agg (want avg or min)", nil)
		return
	}

	since := s.now().UTC().AddDate(0, 0, -days)
	samples, err := s.store.GetFractionSamplesSince(since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query time series", err)
		return
	}

	// Group fractions by calendar day, then by provider. A day appears only if
	// at least one sample fell on it; within a present day every provider is
	// emitted, using null where that provider had no data.
	byDay := make(map[string]map[string][]float64)
	for _, sample := range samples {
		provider := classifyProvider(sample.Label)
		if provider == "" || sample.RemainingFraction == nil {
			continue
		}
		day := sample.CapturedAt.UTC().Format(dateFormat)
		providers := byDay[day]
		if providers == nil {
			providers = make(map[string][]float64)
			byDay[day] = providers
		}
		providers[provider] = append(providers[provider], *sample.RemainingFraction)
	}

	dates := make([]string, 0, len(byDay))
	for day := range byDay {
		dates = append(dates, day)
	}
	sort.Strings(dates)

	resp := timeseriesResponse{Range: rangeLabel(days), Agg: agg, Days: []timeseriesDay{}}
	for _, day := range dates {
		providers := byDay[day]
		values := make(map[string]*float64, len(providerOrder))
		for _, provider := range providerOrder {
			if vals := providers[provider]; len(vals) > 0 {
				v := aggregate(vals, agg)
				values[provider] = &v
			} else {
				values[provider] = nil
			}
		}
		resp.Days = append(resp.Days, timeseriesDay{Date: day, Providers: values})
	}

	s.writeJSON(w, http.StatusOK, resp)
}

type breakdownRow struct {
	Email            string   `json:"email"`
	Label            string   `json:"label"`
	ModelID          string   `json:"model_id"`
	CurrentFraction  *float64 `json:"current_fraction"`
	StartingFraction *float64 `json:"starting_fraction,omitempty"`
	Consumed         *float64 `json:"consumed,omitempty"`
	ResetTime        *string  `json:"reset_time,omitempty"`
}

func (s *Server) analyticsBreakdownHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.GetBreakdown()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query breakdown", err)
		return
	}

	out := make([]breakdownRow, 0, len(rows))
	for _, row := range rows {
		br := breakdownRow{
			Email:            row.Email,
			Label:            row.Label,
			ModelID:          row.ModelID,
			CurrentFraction:  row.CurrentFraction,
			StartingFraction: row.StartingFraction,
			Consumed:         row.Consumed,
		}
		if row.ResetTime != nil {
			reset := row.ResetTime.UTC().Format(time.RFC3339)
			br.ResetTime = &reset
		}
		out = append(out, br)
	}

	// Default ordering is most-consumed first; rows without a computed
	// consumption sort last. The frontend re-sorts client-side from here.
	sort.SliceStable(out, func(i, j int) bool {
		ci, cj := out[i].Consumed, out[j].Consumed
		switch {
		case ci == nil:
			return false
		case cj == nil:
			return true
		default:
			return *ci > *cj
		}
	})

	s.writeJSON(w, http.StatusOK, map[string]any{
		"rows": out,
	})
}

func (s *Server) analyticsStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetAnalyticsStats(s.now().UTC())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query stats", err)
		return
	}
	s.writeJSON(w, http.StatusOK, stats)
}

type sparklinePoint struct {
	CapturedAt        time.Time `json:"captured_at"`
	RemainingFraction *float64  `json:"remaining_fraction"`
}

type sparklineModel struct {
	Label   string           `json:"label"`
	ModelID string           `json:"model_id"`
	Points  []sparklinePoint `json:"points"`
}

func (s *Server) accountSparklinesHandler(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	since := s.now().UTC().AddDate(0, 0, -7)

	samples, err := s.store.GetAccountFractionSamplesSince(email, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query sparklines", err)
		return
	}

	// Group points per model, preserving the store's ascending-time ordering.
	// Null fractions are retained so the frontend can draw gaps.
	order := make([]string, 0)
	models := make(map[string]*sparklineModel)
	for _, sample := range samples {
		key := sample.ModelID + "\x00" + sample.Label
		model := models[key]
		if model == nil {
			model = &sparklineModel{
				Label:   sample.Label,
				ModelID: sample.ModelID,
				Points:  []sparklinePoint{},
			}
			models[key] = model
			order = append(order, key)
		}
		model.Points = append(model.Points, sparklinePoint{
			CapturedAt:        sample.CapturedAt,
			RemainingFraction: sample.RemainingFraction,
		})
	}

	out := make([]sparklineModel, 0, len(order))
	for _, key := range order {
		out = append(out, *models[key])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Label < out[j].Label })

	s.writeJSON(w, http.StatusOK, map[string]any{
		"email":  email,
		"models": out,
	})
}

type timelineEvent struct {
	Type  string              `json:"type"`
	At    time.Time           `json:"at"`
	Quota map[string]*float64 `json:"quota"`
}

func (s *Server) accountTimelineHandler(w http.ResponseWriter, r *http.Request) {
	email := r.PathValue("email")
	since := s.now().UTC().AddDate(0, 0, -7)

	refs, err := s.store.GetSnapshotRefsSince(email, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query timeline", err)
		return
	}

	events := make([]timelineEvent, 0)
	if len(refs) > 0 {
		// The first snapshot always begins a session. A gap wider than the
		// threshold closes the prior session (logout on the snapshot before the
		// gap) and opens a new one (login on the snapshot after it). The final
		// session is left open because it may still be active.
		event, err := s.timelineEvent("login", refs[0])
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to summarize timeline quota", err)
			return
		}
		events = append(events, event)

		for i := 1; i < len(refs); i++ {
			if refs[i].CapturedAt.Sub(refs[i-1].CapturedAt) <= SessionGapThreshold {
				continue
			}
			logout, err := s.timelineEvent("logout", refs[i-1])
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to summarize timeline quota", err)
				return
			}
			login, err := s.timelineEvent("login", refs[i])
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to summarize timeline quota", err)
				return
			}
			events = append(events, logout, login)
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"email":  email,
		"events": events,
	})
}

func (s *Server) timelineEvent(kind string, ref domain.SnapshotRef) (timelineEvent, error) {
	models, err := s.store.GetSnapshotModels(ref.ID)
	if err != nil {
		return timelineEvent{}, err
	}
	return timelineEvent{
		Type:  kind,
		At:    ref.CapturedAt,
		Quota: summarizeByProvider(models),
	}, nil
}

func aggregate(vals []float64, agg string) float64 {
	if agg == "min" {
		min := vals[0]
		for _, v := range vals[1:] {
			if v < min {
				min = v
			}
		}
		return min
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func rangeLabel(days int) string {
	if days == 30 {
		return "30d"
	}
	return "7d"
}
