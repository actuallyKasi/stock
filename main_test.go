package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/actuallyKasi/stock/internal/quote"
)

// env returns a getenv func backed by vars.
func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestParseOptionsAPIKey(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{"env only", nil, map[string]string{apiKeyEnv: "env-key"}, "env-key"},
		{"flag only", []string{"--api-key", "flag-key"}, nil, "flag-key"},
		{"flag overrides env", []string{"--api-key", "flag-key"}, map[string]string{apiKeyEnv: "env-key"}, "flag-key"},
		{"shorthand overrides env", []string{"-k", "flag-key"}, map[string]string{apiKeyEnv: "env-key"}, "flag-key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := parseOptions(tt.args, env(tt.env), &bytes.Buffer{})
			if err != nil {
				t.Fatalf("parseOptions: %v", err)
			}
			if o.apiKey != tt.want {
				t.Errorf("apiKey = %q, want %q", o.apiKey, tt.want)
			}
		})
	}
}

func TestRunMissingAPIKey(t *testing.T) {
	for _, args := range [][]string{nil, {"--price", "123.45"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			newProvider := func(string) quote.Provider {
				t.Fatal("provider built without an API key")
				return nil
			}
			var out bytes.Buffer
			err := run(context.Background(), args, env(nil), &out, &bytes.Buffer{}, newProvider)
			if err == nil {
				t.Fatal("run succeeded without an API key")
			}
			for _, want := range []string{apiKeyEnv, "--api-key", "https://massive.com/dashboard/keys"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
			if out.Len() != 0 {
				t.Errorf("unexpected output %q", out.String())
			}
		})
	}
}

const (
	netPrevBody = `{"ticker":"NET","queryCount":1,"resultsCount":1,"adjusted":true,` +
		`"results":[{"T":"NET","v":1.2e6,"vw":122.9,"o":121,"c":123.45,"h":125,"l":120,"t":1760068800000,"n":4321}],` +
		`"status":"OK","request_id":"abc","count":1}`
	inrPrevBody = `{"ticker":"C:USDINR","queryCount":1,"resultsCount":1,"adjusted":true,` +
		`"results":[{"T":"C:USDINR","v":100,"vw":84.1,"o":84,"c":84.12,"h":84.2,"l":83.9,"t":1760054400000,"n":100}],` +
		`"status":"OK","request_id":"def","count":1}`
	sgdPrevBody = `{"ticker":"C:USDSGD","queryCount":1,"resultsCount":1,"adjusted":true,` +
		`"results":[{"T":"C:USDSGD","v":100,"vw":1.29,"o":1.29,"c":1.2934,"h":1.3,"l":1.28,"t":1760054400000,"n":100}],` +
		`"status":"OK","request_id":"ghi","count":1}`
)

// newMassiveServer starts a fake Massive API that serves the NET, USD/INR and
// USD/SGD prev-day bars to requests authenticated with key. It returns a
// provider factory pointed at the server and a pointer to the request paths
// served.
func newMassiveServer(t *testing.T, key string) (func(string) quote.Provider, *[]string) {
	t.Helper()
	bodies := map[string]string{
		"/v2/aggs/ticker/NET/prev":      netPrevBody,
		"/v2/aggs/ticker/C:USDINR/prev": inrPrevBody,
		"/v2/aggs/ticker/C:USDSGD/prev": sgdPrevBody,
	}
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer "+key {
			http.Error(w, `{"status":"ERROR"}`, http.StatusUnauthorized)
			return
		}
		body, ok := bodies[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	newProvider := func(apiKey string) quote.Provider {
		return quote.NewMassive(apiKey, quote.WithBaseURL(srv.URL), quote.WithHTTPClient(srv.Client()))
	}
	return newProvider, &paths
}

func TestRunEndToEnd(t *testing.T) {
	tests := []struct {
		currency string
		want     string
	}{
		{"INR", `Symbol:            NET
Price (USD):       123.45  (close 2025-10-10)
Units:             1000
USD→INR:           84.12   (close 2025-10-10)
Total value (INR): 10384614
`},
		{"sgd", `Symbol:            NET
Price (USD):       123.45  (close 2025-10-10)
Units:             1000
USD→SGD:           1.29    (close 2025-10-10)
Total value (SGD): 159670
`},
	}
	for _, tt := range tests {
		t.Run(tt.currency, func(t *testing.T) {
			newProvider, paths := newMassiveServer(t, "env-key")
			var out bytes.Buffer
			args := []string{"--symbol", "net", "--currency", tt.currency}
			err := run(context.Background(), args, env(map[string]string{apiKeyEnv: "env-key"}), &out, &bytes.Buffer{}, newProvider)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got := out.String(); got != tt.want {
				t.Errorf("output:\n%s\nwant:\n%s", got, tt.want)
			}
			if len(*paths) != 2 {
				t.Errorf("requests = %q, want 2", *paths)
			}
		})
	}
}

func TestRunAPIKeyFlagUsedForRequests(t *testing.T) {
	newProvider, _ := newMassiveServer(t, "flag-key")
	args := []string{"--api-key", "flag-key"}
	err := run(context.Background(), args, env(map[string]string{apiKeyEnv: "env-key"}), &bytes.Buffer{}, &bytes.Buffer{}, newProvider)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunUserSuppliedPrice(t *testing.T) {
	for _, flagName := range []string{"--price", "-p", "--stock_price"} {
		t.Run(flagName, func(t *testing.T) {
			newProvider, paths := newMassiveServer(t, "k")
			var out, stderr bytes.Buffer
			args := []string{flagName, "100", "--units", "2.5"}
			err := run(context.Background(), args, env(map[string]string{apiKeyEnv: "k"}), &out, &stderr, newProvider)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			want := `Symbol:            NET
Price (USD):       100.00  (user-supplied)
Units:             2.5
USD→INR:           84.12   (close 2025-10-10)
Total value (INR): 21030
`
			if got := out.String(); got != want {
				t.Errorf("output:\n%s\nwant:\n%s", got, want)
			}
			if want := []string{"/v2/aggs/ticker/C:USDINR/prev"}; strings.Join(*paths, ",") != strings.Join(want, ",") {
				t.Errorf("requests = %q, want %q", *paths, want)
			}
			deprecated := strings.Contains(stderr.String(), "--stock_price has been deprecated, use --price instead")
			if deprecated != (flagName == "--stock_price") {
				t.Errorf("stderr = %q, deprecation warning printed = %v", stderr.String(), deprecated)
			}
		})
	}
}

type fakeProvider struct {
	priceErr   error
	usdToCalls int
}

func (f *fakeProvider) Price(ctx context.Context, symbol string) (float64, time.Time, error) {
	return 0, time.Time{}, f.priceErr
}

func (f *fakeProvider) USDTo(ctx context.Context, currency string) (float64, time.Time, error) {
	f.usdToCalls++
	return 1, time.Time{}, nil
}

func TestRunPriceError(t *testing.T) {
	wantErr := errors.New("no price")
	p := &fakeProvider{priceErr: wantErr}
	newProvider := func(string) quote.Provider { return p }

	err := run(context.Background(), []string{"-s", "NOPE"}, env(map[string]string{apiKeyEnv: "k"}), &bytes.Buffer{}, &bytes.Buffer{}, newProvider)
	if !errors.Is(err, wantErr) {
		t.Fatalf("run error = %v, want %v", err, wantErr)
	}
	if p.usdToCalls != 0 {
		t.Errorf("USDTo called %d times after Price failed, want 0", p.usdToCalls)
	}
}

func TestHelpMatchesREADME(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseOptions([]string{"--help"}, env(nil), &stderr)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("parseOptions(--help) error = %v, want ErrHelp", err)
	}
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	// The README shows the help text as an indented code block.
	indented := "    " + strings.ReplaceAll(strings.TrimSuffix(stderr.String(), "\n"), "\n", "\n    ")
	indented = strings.ReplaceAll(indented, "    \n", "\n")
	if !strings.Contains(string(readme), indented) {
		t.Errorf("README.md does not contain the --help output:\n%s", indented)
	}
}
