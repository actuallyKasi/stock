## Goal
Restructure `stock.go` so the quote and forex fetching sit behind a small interface that can be tested without the network. Fix the existing bugs along the way. This prepares for swapping Finnhub for Massive in a later issue, so **keep Finnhub as the only provider for now**.

## Scope
- Create `internal/quote` (or a similar package) containing:
  - `type Provider interface { Price(ctx context.Context, symbol string) (float64, error); USDTo(ctx context.Context, currency string) (float64, error) }`
  - A Finnhub implementation that takes a configurable base URL, API key and `*http.Client`, so tests can point it at `httptest.Server`.
- Move the portfolio maths into a pure function, for example `Value(units, price, rate float64) float64`, and test it.
- `main.go` stays thin. It parses flags, builds the provider, calls `run(ctx, ...) error`, and exits non-zero from `main` only. No `os.Exit` or `log.Fatal` inside library code.
- Use an HTTP client with a timeout (for example 10s) and `context`.
- Send the API key in a header, not the query string (Finnhub accepts `X-Finnhub-Token`), so it doesn't leak into logs or error messages.
- **Bugs to fix:**
  1. When `--stock_price` is passed, the output prints `quote.C` (0) as "Stock price" instead of the price actually used.
  2. The forex response's status code is never checked.
  3. The stock quote is fetched even when `--stock_price` is supplied. Skip the call in that case.
  4. A 200 response with an empty or zero quote (an unknown symbol) is treated as price 0. Return a clear error instead.
- **Out of scope:** adding Massive, and changing flag names or the output format beyond bug fix 1.

## Acceptance criteria
- [ ] The `Provider` interface and Finnhub implementation exist with an injectable base URL and HTTP client.
- [ ] `httptest` unit tests cover: a successful quote, a non-200 quote, a non-200 forex response, malformed JSON, an unknown symbol (zero price), and the API key sent as a header and absent from the URL.
- [ ] A unit test for `Value` covers rounding.
- [ ] A test (with a fake Provider) shows `run` prints the user-supplied price when `--stock_price` is set and doesn't call `Price`.
- [ ] `go test ./...` passes without network access.

## Validation
Add the tests above. The workflow's validation commands must pass.
