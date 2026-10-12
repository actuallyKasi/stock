package quote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testKey = "secret-key"

// newTestFinnhub starts a server with handler and returns a Finnhub provider
// pointed at it.
func newTestFinnhub(t *testing.T, handler http.HandlerFunc) *Finnhub {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewFinnhub(testKey, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
}

func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}

func TestPriceSuccess(t *testing.T) {
	f := newTestFinnhub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quote" {
			t.Errorf("path = %q, want /quote", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != "NET" {
			t.Errorf("symbol = %q, want NET", got)
		}
		w.Write([]byte(`{"c":123.45,"h":125,"l":120,"o":121,"pc":122,"t":1700000000}`))
	})

	got, err := f.Price(context.Background(), "NET")
	if err != nil {
		t.Fatalf("Price: %v", err)
	}
	if got != 123.45 {
		t.Errorf("Price = %v, want 123.45", got)
	}
}

func TestPriceNon200(t *testing.T) {
	f := newTestFinnhub(t, respond(http.StatusTooManyRequests, `{"error":"API limit reached"}`))

	_, err := f.Price(context.Background(), "NET")
	if err == nil {
		t.Fatal("Price: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error %q does not mention status 429", err)
	}
}

func TestPriceMalformedJSON(t *testing.T) {
	f := newTestFinnhub(t, respond(http.StatusOK, `{"c":`))

	if _, err := f.Price(context.Background(), "NET"); err == nil {
		t.Fatal("Price: expected error, got nil")
	}
}

func TestPriceUnknownSymbol(t *testing.T) {
	// Finnhub answers unknown symbols with 200 and a zeroed quote.
	f := newTestFinnhub(t, respond(http.StatusOK, `{"c":0,"d":null,"dp":null,"h":0,"l":0,"o":0,"pc":0,"t":0}`))

	_, err := f.Price(context.Background(), "NOPE")
	if err == nil {
		t.Fatal("Price: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "NOPE") {
		t.Errorf("error %q does not mention the symbol", err)
	}
}

func TestUSDToSuccess(t *testing.T) {
	f := newTestFinnhub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forex/rates" {
			t.Errorf("path = %q, want /forex/rates", r.URL.Path)
		}
		w.Write([]byte(`{"base":"EUR","quote":{"USD":1.25,"INR":100,"SGD":1.5}}`))
	})

	got, err := f.USDTo(context.Background(), "inr")
	if err != nil {
		t.Fatalf("USDTo: %v", err)
	}
	if got != 80 {
		t.Errorf("USDTo = %v, want 80", got)
	}
}

func TestUSDToNon200(t *testing.T) {
	f := newTestFinnhub(t, respond(http.StatusForbidden, `{"error":"You don't have access to this resource."}`))

	_, err := f.USDTo(context.Background(), "INR")
	if err == nil {
		t.Fatal("USDTo: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error %q does not mention status 403", err)
	}
}

func TestUSDToMalformedJSON(t *testing.T) {
	f := newTestFinnhub(t, respond(http.StatusOK, `not json`))

	if _, err := f.USDTo(context.Background(), "INR"); err == nil {
		t.Fatal("USDTo: expected error, got nil")
	}
}

func TestUSDToMissingCurrency(t *testing.T) {
	f := newTestFinnhub(t, respond(http.StatusOK, `{"base":"EUR","quote":{"USD":1.25}}`))

	if _, err := f.USDTo(context.Background(), "INR"); err == nil {
		t.Fatal("USDTo: expected error, got nil")
	}
}

func TestAPIKeySentAsHeader(t *testing.T) {
	var requests int
	f := newTestFinnhub(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.Header.Get("X-Finnhub-Token"); got != testKey {
			t.Errorf("X-Finnhub-Token = %q, want %q", got, testKey)
		}
		if strings.Contains(r.URL.String(), testKey) {
			t.Errorf("API key leaked into URL %q", r.URL)
		}
		if r.URL.Query().Has("token") {
			t.Errorf("unexpected token query param in %q", r.URL)
		}
		w.Write([]byte(`{"c":1,"quote":{"USD":1,"INR":1}}`))
	})

	ctx := context.Background()
	if _, err := f.Price(ctx, "NET"); err != nil {
		t.Fatalf("Price: %v", err)
	}
	if _, err := f.USDTo(ctx, "INR"); err != nil {
		t.Fatalf("USDTo: %v", err)
	}
	if requests != 2 {
		t.Errorf("server saw %d requests, want 2", requests)
	}
}

func TestErrorsDoNotContainAPIKey(t *testing.T) {
	f := newTestFinnhub(t, respond(http.StatusInternalServerError, "boom"))

	_, err := f.Price(context.Background(), "NET")
	if err == nil {
		t.Fatal("Price: expected error, got nil")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("error %q contains the API key", err)
	}
}

func TestValue(t *testing.T) {
	tests := []struct {
		name               string
		units, price, rate float64
		want               float64
	}{
		{"whole", 10, 2, 3, 60},
		{"rounds down", 1, 1.2, 1, 1},
		{"rounds half up", 1, 2.5, 1, 3},
		{"rounds up", 3, 33.33, 1.01, 101},
		{"zero units", 0, 123.45, 84.12, 0},
		{"realistic", 1000, 123.45, 84.12, 10384614},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Value(tt.units, tt.price, tt.rate); got != tt.want {
				t.Errorf("Value(%v, %v, %v) = %v, want %v", tt.units, tt.price, tt.rate, got, tt.want)
			}
		})
	}
}
