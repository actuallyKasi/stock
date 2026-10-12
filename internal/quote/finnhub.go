package quote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// FinnhubBaseURL is the default Finnhub API base URL.
const FinnhubBaseURL = "https://finnhub.io/api/v1"

// maxErrBody caps how much of an error response body is included in errors.
const maxErrBody = 200

// Finnhub is a Provider backed by the finnhub.io API.
type Finnhub struct {
	apiKey string
	cfg    config
}

// NewFinnhub returns a Finnhub provider authenticating with apiKey.
func NewFinnhub(apiKey string, opts ...Option) *Finnhub {
	return &Finnhub{apiKey: apiKey, cfg: newConfig(FinnhubBaseURL, opts)}
}

type finnhubQuote struct {
	C  float64 `json:"c"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	O  float64 `json:"o"`
	Pc float64 `json:"pc"`
	T  int64   `json:"t"`
}

type finnhubForex struct {
	Base  string             `json:"base"`
	Quote map[string]float64 `json:"quote"`
}

// Price returns the current price of symbol.
func (f *Finnhub) Price(ctx context.Context, symbol string) (float64, error) {
	var q finnhubQuote
	if err := f.get(ctx, "/quote", url.Values{"symbol": {symbol}}, &q); err != nil {
		return 0, fmt.Errorf("finnhub quote for %s: %w", symbol, err)
	}
	if q.C == 0 {
		return 0, fmt.Errorf("finnhub quote for %s: no price returned (unknown symbol?)", symbol)
	}
	return q.C, nil
}

// USDTo returns the USD to currency exchange rate.
func (f *Finnhub) USDTo(ctx context.Context, currency string) (float64, error) {
	currency = strings.ToUpper(currency)
	var fx finnhubForex
	if err := f.get(ctx, "/forex/rates", nil, &fx); err != nil {
		return 0, fmt.Errorf("finnhub forex rates: %w", err)
	}
	usd, to := fx.Quote["USD"], fx.Quote[currency]
	if usd == 0 || to == 0 {
		return 0, fmt.Errorf("finnhub forex rates: no USD/%s rate returned", currency)
	}
	return to / usd, nil
}

func (f *Finnhub) get(ctx context.Context, path string, query url.Values, v any) error {
	u := f.cfg.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Finnhub-Token", f.apiKey)

	resp, err := f.cfg.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody+1))
		err := fmt.Errorf("unexpected status %s: %s", resp.Status, truncate(string(body), maxErrBody))
		if resp.StatusCode == http.StatusTooManyRequests {
			err = fmt.Errorf("%w: %w", ErrRateLimited, err)
		}
		return err
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		s = s[:n] + "..."
	}
	return strings.TrimSpace(s)
}
