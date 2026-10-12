package quote

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestMassive starts a server with handler and returns a Massive provider
// pointed at it.
func newTestMassive(t *testing.T, handler http.HandlerFunc) *Massive {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewMassive(testKey, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
}

// checkMassiveRequest asserts r is a prev-day bar request for path that
// authenticates with a bearer token and not a query parameter.
func checkMassiveRequest(t *testing.T, r *http.Request, path string) {
	t.Helper()
	if r.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", r.Method)
	}
	if r.URL.Path != path {
		t.Errorf("path = %q, want %q", r.URL.Path, path)
	}
	if got, want := r.Header.Get("Authorization"), "Bearer "+testKey; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
	if r.URL.Query().Has("apiKey") {
		t.Errorf("unexpected apiKey query param in %q", r.URL)
	}
	if strings.Contains(r.URL.String(), testKey) {
		t.Errorf("API key leaked into URL %q", r.URL)
	}
}

const (
	// 2025-10-10 00:00 America/New_York, where Massive starts daily stock bars.
	netBarMillis = 1760068800000
	netPrevBody  = `{"ticker":"NET","queryCount":1,"resultsCount":1,"adjusted":true,` +
		`"results":[{"T":"NET","v":1.2e6,"vw":122.9,"o":121,"c":123.45,"h":125,"l":120,"t":1760068800000,"n":4321}],` +
		`"status":"OK","request_id":"abc","count":1}`
	inrPrevBody = `{"ticker":"C:USDINR","queryCount":1,"resultsCount":1,"adjusted":true,` +
		`"results":[{"T":"C:USDINR","v":100,"vw":84.1,"o":84,"c":84.12,"h":84.2,"l":83.9,"t":1760054400000,"n":100}],` +
		`"status":"OK","request_id":"def","count":1}`
)

func TestMassivePriceSuccess(t *testing.T) {
	m := newTestMassive(t, func(w http.ResponseWriter, r *http.Request) {
		checkMassiveRequest(t, r, "/v2/aggs/ticker/NET/prev")
		if got := r.URL.Query().Get("adjusted"); got != "true" {
			t.Errorf("adjusted = %q, want true", got)
		}
		w.Write([]byte(netPrevBody))
	})

	got, asOf, err := m.Price(context.Background(), "NET")
	if err != nil {
		t.Fatalf("Price: %v", err)
	}
	if got != 123.45 {
		t.Errorf("Price = %v, want 123.45", got)
	}
	if want := time.UnixMilli(netBarMillis); !asOf.Equal(want) {
		t.Errorf("as of = %v, want %v", asOf, want)
	}
}

func TestMassivePriceLowercaseSymbol(t *testing.T) {
	m := newTestMassive(t, func(w http.ResponseWriter, r *http.Request) {
		checkMassiveRequest(t, r, "/v2/aggs/ticker/NET/prev")
		w.Write([]byte(netPrevBody))
	})

	if _, _, err := m.Price(context.Background(), "net"); err != nil {
		t.Fatalf("Price: %v", err)
	}
}

func TestMassiveUSDToSuccess(t *testing.T) {
	m := newTestMassive(t, func(w http.ResponseWriter, r *http.Request) {
		checkMassiveRequest(t, r, "/v2/aggs/ticker/C:USDINR/prev")
		w.Write([]byte(inrPrevBody))
	})

	got, asOf, err := m.USDTo(context.Background(), "inr")
	if err != nil {
		t.Fatalf("USDTo: %v", err)
	}
	if got != 84.12 {
		t.Errorf("USDTo = %v, want 84.12", got)
	}
	if want := time.UnixMilli(1760054400000); !asOf.Equal(want) {
		t.Errorf("as of = %v, want %v", asOf, want)
	}
}

func TestMassiveUnauthorized(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			m := newTestMassive(t, respond(status, `{"status":"ERROR","request_id":"x","error":"Unknown API Key"}`))

			_, _, err := m.Price(context.Background(), "NET")
			if err == nil {
				t.Fatal("Price: expected error, got nil")
			}
			if !strings.Contains(err.Error(), "invalid or unauthorised Massive API key") {
				t.Errorf("error %q does not explain the key/plan problem", err)
			}
			if errors.Is(err, ErrRateLimited) {
				t.Errorf("error %q wraps ErrRateLimited for a %d", err, status)
			}
		})
	}
}

func TestMassiveRateLimited(t *testing.T) {
	m := newTestMassive(t, respond(http.StatusTooManyRequests, `{"status":"ERROR","error":"You've exceeded the maximum requests per minute"}`))

	_, _, err := m.USDTo(context.Background(), "INR")
	if err == nil {
		t.Fatal("USDTo: expected error, got nil")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("error %q does not wrap ErrRateLimited", err)
	}
	if !strings.Contains(err.Error(), "rate limited by Massive (free plan allows ~5 req/min)") {
		t.Errorf("error %q does not mention the free plan limit", err)
	}
}

func TestMassiveOtherStatus(t *testing.T) {
	body := strings.Repeat("x", 2*maxErrBody)
	m := newTestMassive(t, respond(http.StatusBadGateway, body))

	_, _, err := m.Price(context.Background(), "NET")
	if err == nil {
		t.Fatal("Price: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error %q does not mention status 502", err)
	}
	if !strings.Contains(err.Error(), "xxx...") || strings.Contains(err.Error(), body) {
		t.Errorf("error %q does not contain a truncated body", err)
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("error %q contains the API key", err)
	}
}

func TestMassiveEmptyResults(t *testing.T) {
	tests := []struct {
		name, body string
	}{
		{"zero count", `{"ticker":"NOPE","queryCount":0,"resultsCount":0,"adjusted":true,"status":"OK","request_id":"x"}`},
		{"empty results", `{"ticker":"NOPE","queryCount":0,"resultsCount":1,"adjusted":true,"results":[],"status":"OK"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMassive(t, respond(http.StatusOK, tt.body))

			_, _, err := m.Price(context.Background(), "nope")
			if err == nil {
				t.Fatal("Price: expected error, got nil")
			}
			if !strings.Contains(err.Error(), "no data for symbol NOPE") {
				t.Errorf("error %q does not report missing data for NOPE", err)
			}
		})
	}
}

func TestMassiveUSDToEmptyResults(t *testing.T) {
	m := newTestMassive(t, respond(http.StatusOK, `{"ticker":"C:USDXYZ","resultsCount":0,"status":"OK"}`))

	_, _, err := m.USDTo(context.Background(), "xyz")
	if err == nil {
		t.Fatal("USDTo: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no data for symbol C:USDXYZ") {
		t.Errorf("error %q does not report missing data for C:USDXYZ", err)
	}
}

func TestMassiveMalformedJSON(t *testing.T) {
	m := newTestMassive(t, respond(http.StatusOK, `{"results":[{"c":`))

	if _, _, err := m.Price(context.Background(), "NET"); err == nil {
		t.Error("Price: expected error, got nil")
	}
	if _, _, err := m.USDTo(context.Background(), "INR"); err == nil {
		t.Error("USDTo: expected error, got nil")
	}
}
