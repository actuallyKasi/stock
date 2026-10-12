## Goal
Add a Massive (formerly Polygon.io) implementation of the `quote.Provider` interface from the refactor issue. It should work on Massive's **free Basic plans**, which are end-of-day data limited to about 5 requests per minute.

## Massive API facts (verified from https://massive.com/docs)
- Base URL: `https://api.massive.com`. The legacy `api.polygon.io` still works but don't use it.
- Auth: send the key as `Authorization: Bearer <key>`, never in the query string.
- **Stock price:** `GET /v2/aggs/ticker/{TICKER}/prev?adjusted=true`, available on Stocks Basic. Response: `{"status":"OK","ticker":"NET","resultsCount":1,"results":[{"T":"NET","c":<close>,"o":..,"h":..,"l":..,"v":..,"vw":..,"t":<unix ms>}]}`. Use `results[0].c`.
- **FX rate:** `GET /v2/aggs/ticker/C:USD{CUR}/prev`, for example `C:USDINR`, available on Currencies Basic. The response shape is the same as above, and `results[0].c` is the USD to CUR close.
- Don't use the snapshot endpoint (`/v2/snapshot/...`) or `/v1/conversion/{from}/{to}`. Both need paid tiers.

## Scope
- `internal/quote/massive.go`: `NewMassive(apiKey string, opts ...)` with an injectable base URL and `*http.Client`.
- Map errors clearly:
  - 401/403: "invalid or unauthorised Massive API key / plan does not include this data"
  - 429: "rate limited by Massive (free plan allows ~5 req/min)"
  - `resultsCount == 0` or empty `results`: "no data for symbol X"
  - any other non-2xx: include the status and a truncated body
- Upper-case the symbol and currency before building paths.
- Expose the bar's timestamp too, for example `Price` returns or a separate method gives `time.Time`, so the CLI can show "as of" dates. Keep the change to the interface minimal and update the Finnhub implementation and fakes if you change it.
- **Out of scope:** wiring Massive into `main`, removing Finnhub, and README changes. A later issue does those.

## Acceptance criteria
- [ ] `httptest` tests assert the exact request paths (`/v2/aggs/ticker/NET/prev`, `/v2/aggs/ticker/C:USDINR/prev`) and the `Authorization: Bearer` header, and that no `apiKey` query param is sent.
- [ ] Tests cover success, 401, 429, empty results and malformed JSON.
- [ ] A lowercase symbol input (`net`) requests `NET`.
- [ ] `go test ./...` passes without network access.

## Validation
Add the tests above. The workflow's validation commands must pass.
