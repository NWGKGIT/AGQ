// Package apiclient is a typed HTTP client for the AGQ daemon's local JSON
// API. The Wails app binds thin wrappers around it so the React frontend
// never issues HTTP requests itself.
package apiclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrDaemonUnreachable wraps connection-level failures so the frontend can
// distinguish "daemon not running" from an API error.
var ErrDaemonUnreachable = errors.New("daemon unreachable")

// Client talks to one daemon instance on loopback.
type Client struct {
	baseURL string
	http    *http.Client
}

// New creates a client for the daemon listening on the given port.
func New(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// apiError is the daemon's error envelope.
type apiError struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

func (c *Client) get(path string, out any) error {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDaemonUnreachable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		var ae apiError
		if json.Unmarshal(body, &ae) == nil && ae.Error != "" {
			return fmt.Errorf("daemon: %s (HTTP %d)", ae.Error, resp.StatusCode)
		}
		return fmt.Errorf("daemon: HTTP %d for %s", resp.StatusCode, path)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// Health calls GET /api/health.
func (c *Client) Health() (Health, error) {
	var out Health
	err := c.get("/api/health", &out)
	return out, err
}

// Status calls GET /api/status.
func (c *Client) Status() (DaemonStatus, error) {
	var out DaemonStatus
	err := c.get("/api/status", &out)
	return out, err
}

// Accounts calls GET /api/accounts.
func (c *Client) Accounts() (AccountsResponse, error) {
	var out AccountsResponse
	err := c.get("/api/accounts", &out)
	return out, err
}

// CurrentAccount calls GET /api/account/current.
func (c *Client) CurrentAccount() (CurrentAccount, error) {
	var out CurrentAccount
	err := c.get("/api/account/current", &out)
	return out, err
}

// LatestSnapshot calls GET /api/accounts/{email}/latest.
func (c *Client) LatestSnapshot(email string) (Snapshot, error) {
	var out Snapshot
	err := c.get("/api/accounts/"+url.PathEscape(email)+"/latest", &out)
	return out, err
}

// Snapshots calls GET /api/accounts/{email}/snapshots. A zero limit uses the
// daemon default; an empty before fetches the newest page.
func (c *Client) Snapshots(email string, limit int, before string) (SnapshotsResponse, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if before != "" {
		q.Set("before", before)
	}
	path := "/api/accounts/" + url.PathEscape(email) + "/snapshots"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out SnapshotsResponse
	err := c.get(path, &out)
	return out, err
}

// Sparklines calls GET /api/accounts/{email}/sparklines.
func (c *Client) Sparklines(email string) (SparklinesResponse, error) {
	var out SparklinesResponse
	err := c.get("/api/accounts/"+url.PathEscape(email)+"/sparklines", &out)
	return out, err
}

// Timeline calls GET /api/accounts/{email}/timeline.
func (c *Client) Timeline(email string) (TimelineResponse, error) {
	var out TimelineResponse
	err := c.get("/api/accounts/"+url.PathEscape(email)+"/timeline", &out)
	return out, err
}

// ModelsLatest calls GET /api/models/latest.
func (c *Client) ModelsLatest() (ModelsLatestResponse, error) {
	var out ModelsLatestResponse
	err := c.get("/api/models/latest", &out)
	return out, err
}

// Timeseries calls GET /api/analytics/timeseries.
func (c *Client) Timeseries(rangeKey, agg string) (TimeseriesResponse, error) {
	q := url.Values{}
	if rangeKey != "" {
		q.Set("range", rangeKey)
	}
	if agg != "" {
		q.Set("agg", agg)
	}
	path := "/api/analytics/timeseries"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out TimeseriesResponse
	err := c.get(path, &out)
	return out, err
}

// Breakdown calls GET /api/analytics/breakdown.
func (c *Client) Breakdown() (BreakdownResponse, error) {
	var out BreakdownResponse
	err := c.get("/api/analytics/breakdown", &out)
	return out, err
}

// Stats calls GET /api/analytics/stats.
func (c *Client) Stats() (Stats, error) {
	var out Stats
	err := c.get("/api/analytics/stats", &out)
	return out, err
}
