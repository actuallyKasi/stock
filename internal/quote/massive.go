package quote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MassiveBaseURL is the default Massive (formerly Polygon.io) API base URL.
const MassiveBaseURL = "https://api.massive.com"

// Massive is a Provider backed by the Massive API. It uses the previous-day
// aggregate bar endpoints, which are available on the free Basic plans, so
// prices and rates are end-of-day closes.
type Massive struct {
	apiKey string
	cfg    config
}

// NewMassive returns a Massive provider authenticating with apiKey.
func NewMassive(apiKey string, opts ...Option) *Massive {
	return &Massive{apiKey: apiKey, cfg: newConfig(MassiveBaseURL, opts)}
}

type massiveAggs struct {
	ResultsCount int          `json:"resultsCount"`
	Results      []massiveBar `json:"results"`
}

type massiveBar struct {
	// Ticker must be declared: encoding/json matches keys case-insensitively,
	// so without it the "T" key would be decoded into Start.
	Ticker string  `json:"T"`
	C      float64 `json:"c"`
	Start  int64   `json:"t"` // bar start, Unix milliseconds
}

// Price returns the previous trading day's close of symbol and the time of
// that bar.
func (m *Massive) Price(ctx context.Context, symbol string) (float64, time.Time, error) {
	symbol = strings.ToUpper(symbol)
	c, t, err := m.prevClose(ctx, symbol)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("massive price for %s: %w", symbol, err)
	}
	return c, t, nil
}

// USDTo returns the previous day's USD to currency close and the time of that
// bar.
func (m *Massive) USDTo(ctx context.Context, currency string) (float64, time.Time, error) {
	currency = strings.ToUpper(currency)
	c, t, err := m.prevClose(ctx, "C:USD"+currency)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("massive USD/%s rate: %w", currency, err)
	}
	return c, t, nil
}

// prevClose fetches the previous-day bar for ticker and returns its close.
func (m *Massive) prevClose(ctx context.Context, ticker string) (float64, time.Time, error) {
	u := m.cfg.baseURL + "/v2/aggs/ticker/" + url.PathEscape(ticker) + "/prev?adjusted=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.cfg.client.Do(req)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody+1))
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return 0, time.Time{}, fmt.Errorf("invalid or unauthorised Massive API key / plan does not include this data (%s)", resp.Status)
		case http.StatusTooManyRequests:
			return 0, time.Time{}, fmt.Errorf("%w by Massive (free plan allows ~5 req/min)", ErrRateLimited)
		default:
			return 0, time.Time{}, fmt.Errorf("unexpected status %s: %s", resp.Status, truncate(string(body), maxErrBody))
		}
	}

	var aggs massiveAggs
	if err := json.NewDecoder(resp.Body).Decode(&aggs); err != nil {
		return 0, time.Time{}, fmt.Errorf("decoding response: %w", err)
	}
	if aggs.ResultsCount == 0 || len(aggs.Results) == 0 {
		return 0, time.Time{}, fmt.Errorf("no data for symbol %s", ticker)
	}
	bar := aggs.Results[0]
	return bar.C, time.UnixMilli(bar.Start).UTC(), nil
}
