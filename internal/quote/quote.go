// Package quote fetches stock prices and forex rates from market data providers.
package quote

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"
)

// DefaultTimeout is the timeout of the HTTP client used when none is supplied.
const DefaultTimeout = 10 * time.Second

// ErrRateLimited is wrapped by provider errors when the API rejects a request
// for exceeding the plan's rate limit (HTTP 429).
var ErrRateLimited = errors.New("rate limited")

// Provider fetches stock prices and USD exchange rates. Each method also
// returns the time the value is as of, or the zero time if the provider does
// not report one.
type Provider interface {
	// Price returns the latest price of symbol in USD.
	Price(ctx context.Context, symbol string) (float64, time.Time, error)
	// USDTo returns how many units of currency one USD buys.
	USDTo(ctx context.Context, currency string) (float64, time.Time, error)
}

// Value returns the value of units shares at price (USD), converted with rate
// and rounded to the nearest whole unit of the target currency.
func Value(units, price, rate float64) float64 {
	return math.Round(units * rate * price)
}

type config struct {
	baseURL string
	client  *http.Client
}

// Option configures a provider.
type Option func(*config)

// WithBaseURL overrides the provider's API base URL, for example to point it
// at a test server.
func WithBaseURL(u string) Option {
	return func(c *config) { c.baseURL = u }
}

// WithHTTPClient overrides the HTTP client used for requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) { c.client = hc }
}

func newConfig(baseURL string, opts []Option) config {
	c := config{baseURL: baseURL, client: &http.Client{Timeout: DefaultTimeout}}
	for _, o := range opts {
		o(&c)
	}
	return c
}
