package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	flag "github.com/spf13/pflag"

	"github.com/actuallyKasi/stock/internal/quote"
)

// targetCurrency is the currency the portfolio value is reported in.
const targetCurrency = "INR"

type options struct {
	symbol string
	units  float64
	// price is the user-supplied stock price; 0 means fetch it.
	price float64
}

func main() {
	token := flag.StringP("token", "t", "xxxxxxxx", "api token for finnhub.io")
	symbol := flag.StringP("symbol", "s", "NET", "symbol for eg: NET, WORK, EQIX")
	units := flag.Float64P("units", "u", 1000, "No of stock units")
	stockPrice := flag.Float64P("stock_price", "p", 0, "stock price")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	p := quote.NewFinnhub(*token)
	opts := options{symbol: *symbol, units: *units, price: *stockPrice}
	if err := run(ctx, os.Stdout, p, opts); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, w io.Writer, p quote.Provider, o options) error {
	price := o.price
	if price == 0 {
		var err error
		if price, err = p.Price(ctx, o.symbol); err != nil {
			return err
		}
	}

	rate, err := p.USDTo(ctx, targetCurrency)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, `Stock price : %.2f
Number of stocks: %.0f
Usd to INR: %.2f
Total value INR: %.0f
`, price, o.units, rate, quote.Value(o.units, price, rate))
	return err
}
