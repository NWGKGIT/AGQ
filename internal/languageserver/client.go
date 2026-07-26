package languageserver

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"agq-daemon/internal/domain"
)

const (
	// RequestTimeout is the default timeout for regular quota polls.
	RequestTimeout = 10 * time.Second

	// ProbeTimeout is the default timeout for discovering the active port.
	ProbeTimeout = 2 * time.Second

	getUserStatusPath = "/exa.language_server_pb.LanguageServerService/GetUserStatus"
	requestBody       = `{"metadata":{"ideName":"antigravity","extensionName":"antigravity","ideVersion":"1.0.0","locale":"en"}}`
	maxResponseBody   = 8 << 20
)

// Client wraps HTTP access to the local language server.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a language-server client with the provided request timeout.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = RequestTimeout
	}
	return &Client{httpClient: newHTTPClient(timeout)}
}

// FetchSnapshot calls GetUserStatus for one detected language server process.
func (c *Client) FetchSnapshot(ctx context.Context, info *domain.ProcessInfo) (*domain.QuotaSnapshot, error) {
	url := endpointURL(info.Scheme, info.ActivePort)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	setGetUserStatusHeaders(req, info.CsrfToken)

	resp, err := c.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	return ParseSnapshot(body, time.Now())
}

// ProbeActivePort tries each candidate port over HTTPS then HTTP and returns
// the first port that responds with a valid GetUserStatus payload.
func (c *Client) ProbeActivePort(ports []int, csrfToken string) (activePort int, scheme string) {
	for _, port := range ports {
		for _, s := range []string{"https", "http"} {
			if c.probePort(port, s, csrfToken) {
				return port, s
			}
		}
	}
	return 0, ""
}

// ParseSnapshot converts a GetUserStatus JSON response into a quota snapshot.
func ParseSnapshot(body []byte, capturedAt time.Time) (*domain.QuotaSnapshot, error) {
	var raw userStatusResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	us := raw.UserStatus
	if us.Email == "" {
		return nil, fmt.Errorf("response missing email")
	}

	if capturedAt.IsZero() {
		capturedAt = time.Now()
	}
	capturedAt = capturedAt.UTC()

	snap := &domain.QuotaSnapshot{
		Email:                  us.Email,
		PlanName:               us.UserTier.Name,
		CapturedAt:             capturedAt,
		PromptCreditsAvailable: int64(us.PlanStatus.AvailablePromptCredits),
		PromptCreditsMonthly:   int64(us.PlanStatus.PlanInfo.MonthlyPromptCredits),
		FlowCreditsAvailable:   int64(us.PlanStatus.AvailableFlowCredits),
		FlowCreditsMonthly:     int64(us.PlanStatus.PlanInfo.MonthlyFlowCredits),
		// Persist normalized fields only. The full upstream payload can contain
		// account metadata the monitor does not need to retain.
		RawJSON: "{}",
	}

	for _, cfg := range us.CascadeModelConfigData.ClientModelConfigs {
		m := domain.ModelQuota{
			Label:   cfg.Label,
			ModelID: cfg.ModelOrAlias.Model,
		}

		if cfg.QuotaInfo.RemainingFraction != nil {
			frac := *cfg.QuotaInfo.RemainingFraction
			pct := frac * 100
			m.RemainingFraction = &frac
			m.RemainingPct = &pct
			m.IsExhausted = frac == 0
		}

		if cfg.QuotaInfo.ResetTime != "" {
			if t, err := time.Parse(time.RFC3339, cfg.QuotaInfo.ResetTime); err == nil {
				m.ResetTime = &t
				m.PoolResetTime = &t
				ms := t.UnixMilli() - capturedAt.UnixMilli()
				m.TimeUntilResetMs = &ms
			}
		}

		snap.Models = append(snap.Models, m)
	}

	return snap, nil
}

func (c *Client) probePort(port int, scheme, csrfToken string) bool {
	req, err := http.NewRequest(http.MethodPost, endpointURL(scheme, port), bytes.NewBufferString(requestBody))
	if err != nil {
		return false
	}
	setGetUserStatusHeaders(req, csrfToken)

	resp, err := c.client().Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := readLimitedBody(resp.Body)
	if err != nil {
		return false
	}

	var result userStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.UserStatus.Email != ""
}

func (c *Client) client() *http.Client {
	if c == nil || c.httpClient == nil {
		return newHTTPClient(RequestTimeout)
	}
	return c.httpClient
}

func endpointURL(scheme string, port int) string {
	return fmt.Sprintf("%s://127.0.0.1:%d%s", scheme, port, getUserStatusPath)
}

func setGetUserStatusHeaders(req *http.Request, csrfToken string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("X-Codeium-Csrf-Token", csrfToken)
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

func readLimitedBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseBody+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBody {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseBody)
	}
	return data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// flexInt64 unmarshals a JSON value that may be either a number or a quoted
// decimal string. Antigravity returns either form depending on tier.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		v, err := n.Int64()
		if err != nil {
			return fmt.Errorf("flexInt64: invalid number %q: %w", n, err)
		}
		*f = flexInt64(v)
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("flexInt64: cannot unmarshal %s", string(b))
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("flexInt64: cannot parse string %q: %w", s, err)
	}
	*f = flexInt64(v)
	return nil
}
