# stock
Show your investment value in a particular stock

# Installation

    brew install go
    go install github.com/actuallyKasi/stock@latest

Prices and exchange rates come from [Massive](https://massive.com) (formerly
Polygon.io). Create a free account and copy your API key from
https://massive.com/dashboard/keys, then export it:

    export MASSIVE_API_KEY=your-key

The free plan gives end-of-day data (the previous trading day's close) and
allows about 5 requests per minute. Each run makes up to two requests: one for
the stock price, unless you pass `--price`, and one for the exchange rate.

# Usage

    Usage: stock [flags]

    Flags:
      -k, --api-key string    Massive API key (overrides $MASSIVE_API_KEY)
      -s, --symbol string     stock symbol, e.g. NET, EQIX (default "NET")
      -u, --units float       number of stock units (default 1000)
      -p, --price float       stock price in USD (default: previous close)
      -c, --currency string   currency to report the total value in (default "INR")

    The API key is read from $MASSIVE_API_KEY unless --api-key is given.

`--stock_price` is a deprecated alias for `--price`.

Example:

    $ stock --symbol NET --units 1000
    Symbol:            NET
    Price (USD):       123.45  (close 2026-10-09)
    Units:             1000
    USD→INR:           84.12   (close 2026-10-09)
    Total value (INR): 10384614
