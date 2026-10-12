## Goal
Make Massive the CLI's data provider, remove Finnhub, and tidy up configuration and the README.

## Scope
- `main` builds the Massive provider. Delete the Finnhub implementation and its tests.
- **API key:** read `MASSIVE_API_KEY` from the environment, with an optional `--api-key` / `-k` flag that overrides it. Remove the old `--token` flag and its fake default. If no key is set and a fetch is needed, exit non-zero with a message pointing to https://massive.com/dashboard/keys.
- **Flags:**
  - Keep `-s/--symbol` (default `NET`) and `-u/--units`.
  - Rename `--stock_price` to `--price`/`-p`, and keep `--stock_price` as a deprecated alias using `pflag`'s `MarkDeprecated`.
  - Add `--currency`/`-c` (default `INR`) so the conversion target isn't hard-coded.
- **Output:** show the symbol, the price and its "as of" date, the units, the USD→currency rate and its date, and the total value. Example:
  ```
  Symbol:            NET
  Price (USD):       123.45  (close 2026-10-09)
  Units:             1000
  USD→INR:           84.12   (close 2026-10-09)
  Total value (INR): 10384614
  ```
- **README:** rewrite Usage and Installation. Cover `go install github.com/actuallyKasi/stock@latest`, getting a Massive key, `MASSIVE_API_KEY`, a note that the free plan gives end-of-day data at about 5 req/min, and the new flags. Remove the Finnhub references.
- **Out of scope:** new providers, caching, and multiple symbols.

## Acceptance criteria
- [ ] `grep -ri finnhub .` matches nothing outside git history.
- [ ] With neither `MASSIVE_API_KEY` nor `--api-key` set, the CLI exits non-zero with a helpful message. With `--price` given, it still needs the key for the FX rate.
- [ ] `--api-key` takes precedence over the env var. Cover this with a unit test that calls the config or `run` function with an injected env lookup rather than real `os.Getenv`.
- [ ] `--stock_price` still works and prints a deprecation warning.
- [ ] An end-to-end test of `run` against an `httptest` Massive server checks the full output for `--currency INR` and `--currency SGD`.
- [ ] The README matches the actual `--help` output.

## Validation
Add the tests above. The workflow's validation commands must pass.
