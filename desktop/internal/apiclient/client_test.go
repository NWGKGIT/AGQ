package apiclient

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// testClient points a Client at a httptest server.
func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox does not permit loopback listeners: %v", err)
		}
		t.Fatalf("listen on loopback: %v", err)
	}
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	port, err := strconv.Atoi(strings.TrimPrefix(srv.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatalf("unexpected test server URL %q", srv.URL)
	}
	return New(port)
}

func TestCurrentAccountDecodesResponse(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/account/current" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"state": "ACTIVE",
			"is_live": true,
			"email": "user@example.com",
			"accounts": [{"id": 1, "email": "user@example.com", "plan_name": "Pro",
				"first_seen": "2026-07-01T10:00:00Z", "last_seen": "2026-07-01T11:59:00Z",
				"latest_snapshot": {"email": "user@example.com", "plan_name": "Pro",
					"captured_at": "2026-07-01T11:59:00Z", "staleness_seconds": 15,
					"prompt_credits_available": 10, "prompt_credits_monthly": 100,
					"flow_credits_available": 5, "flow_credits_monthly": 50,
					"models": [{"label": "Gemini 3 Pro", "model_id": "g3p",
						"remaining_fraction": 0.8, "remaining_pct": 80, "is_exhausted": false}]}}],
			"as_of": "2026-07-01T12:00:00Z"
		}`))
	}))

	got, err := c.CurrentAccount()
	if err != nil {
		t.Fatalf("CurrentAccount returned error: %v", err)
	}
	if !got.IsLive || got.State != "ACTIVE" {
		t.Fatalf("state = %q is_live = %v", got.State, got.IsLive)
	}
	if len(got.Accounts) != 1 || got.Accounts[0].LatestSnapshot == nil {
		t.Fatalf("accounts = %+v", got.Accounts)
	}
	models := got.Accounts[0].LatestSnapshot.Models
	if len(models) != 1 || models[0].RemainingFraction == nil || *models[0].RemainingFraction != 0.8 {
		t.Fatalf("models = %+v", models)
	}
}

func TestSnapshotsBuildsQuery(t *testing.T) {
	var gotPath, gotQuery string
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"email": "a@b.c", "snapshots": []}`))
	}))

	if _, err := c.Snapshots("a@b.c", 10, "2026-07-01T00:00:00Z"); err != nil {
		t.Fatalf("Snapshots returned error: %v", err)
	}
	if gotPath != "/api/accounts/a@b.c/snapshots" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "limit=10") || !strings.Contains(gotQuery, "before=") {
		t.Fatalf("query = %q", gotQuery)
	}
}

func TestTimeseriesPassesParams(t *testing.T) {
	var gotQuery string
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"range": "30d", "agg": "min", "days": [
			{"date": "2026-07-01", "providers": {"Gemini": 0.5, "Anthropic": null, "OpenAI": 0.9}}
		]}`))
	}))

	got, err := c.Timeseries("30d", "min")
	if err != nil {
		t.Fatalf("Timeseries returned error: %v", err)
	}
	if !strings.Contains(gotQuery, "range=30d") || !strings.Contains(gotQuery, "agg=min") {
		t.Fatalf("query = %q", gotQuery)
	}
	if len(got.Days) != 1 {
		t.Fatalf("days = %+v", got.Days)
	}
	providers := got.Days[0].Providers
	if providers["Gemini"] == nil || *providers["Gemini"] != 0.5 {
		t.Fatalf("gemini = %v", providers["Gemini"])
	}
	if providers["Anthropic"] != nil {
		t.Fatalf("anthropic = %v, want nil", providers["Anthropic"])
	}
}

func TestErrorEnvelopeIsSurfaced(t *testing.T) {
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "no snapshots found for account", "detail": ""}`))
	}))

	_, err := c.LatestSnapshot("missing@example.com")
	if err == nil || !strings.Contains(err.Error(), "no snapshots found") {
		t.Fatalf("err = %v, want daemon error message", err)
	}
}

func TestConnectionFailureIsUnreachable(t *testing.T) {
	// Point at a loopback port whose listener has already closed.
	listener, listenErr := net.Listen("tcp4", "127.0.0.1:0")
	if listenErr != nil {
		if strings.Contains(listenErr.Error(), "operation not permitted") {
			t.Skipf("sandbox does not permit loopback listeners: %v", listenErr)
		}
		t.Fatalf("listen on loopback: %v", listenErr)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	_, err := New(port).Health()
	if !errors.Is(err, ErrDaemonUnreachable) {
		t.Fatalf("err = %v, want ErrDaemonUnreachable", err)
	}
}
