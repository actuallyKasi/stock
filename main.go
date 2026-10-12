package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	flag "github.com/spf13/pflag"

	"github.com/actuallyKasi/stock/internal/quote"
)

// apiKeyEnv is the environment variable holding the Massive API key.
const apiKeyEnv = "MASSIVE_API_KEY"

// keysURL is where users get a Massive API key.
const keysURL = "https://massive.com/dashboard/keys"

type options struct {
	apiKey   string
	symbol   string
	units    float64
	currency string
	// price is the user-supplied stock price; 0 means fetch it.
	price float64
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr, newMassive)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newMassive(apiKey string) quote.Provider {
	return quote.NewMassive(apiKey)
}

// parseOptions parses the command line args, falling back to getenv for the
// API key. Usage and deprecation warnings are written to stderr.
func parseOptions(args []string, getenv func(string) string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("stock", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVarP(&o.apiKey, "api-key", "k", "", "Massive API key (overrides $"+apiKeyEnv+")")
	fs.StringVarP(&o.symbol, "symbol", "s", "NET", "stock symbol, e.g. NET, EQIX")
	fs.Float64VarP(&o.units, "units", "u", 1000, "number of stock units")
	fs.Float64VarP(&o.price, "price", "p", 0, "stock price in USD (default: previous close)")
	fs.Float64Var(&o.price, "stock_price", 0, "stock price in USD")
	fs.MarkDeprecated("stock_price", "use --price instead")
	fs.StringVarP(&o.currency, "currency", "c", "INR", "currency to report the total value in")
	fs.SortFlags = false
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: stock [flags]\n\nFlags:\n%s\nThe API key is read from $%s unless --api-key is given.\n",
			fs.FlagUsages(), apiKeyEnv)
	}

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if o.apiKey == "" {
		o.apiKey = getenv(apiKeyEnv)
	}
	if o.apiKey == "" {
		return options{}, fmt.Errorf("no Massive API key: set $%s or pass --api-key (get a key at %s)", apiKeyEnv, keysURL)
	}
	o.symbol = strings.ToUpper(o.symbol)
	o.currency = strings.ToUpper(o.currency)
	return o, nil
}

// run parses args, fetches the price and exchange rate from the provider
// built by newProvider, and writes the portfolio value to stdout.
func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer, newProvider func(apiKey string) quote.Provider) error {
	o, err := parseOptions(args, getenv, stderr)
	if err != nil {
		return err
	}
	p := newProvider(o.apiKey)

	price, priceNote := o.price, "user-supplied"
	if price == 0 {
		var asOf time.Time
		if price, asOf, err = p.Price(ctx, o.symbol); err != nil {
			return err
		}
		priceNote = closeNote(asOf)
	}

	rate, rateAsOf, err := p.USDTo(ctx, o.currency)
	if err != nil {
		return err
	}

	return report(stdout, o, price, priceNote, rate, closeNote(rateAsOf))
}

// closeNote describes a close as of t, or returns "" if t is unknown.
func closeNote(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return "close " + t.UTC().Format(time.DateOnly)
}

// report writes the portfolio summary to w with labels and the price and rate
// columns aligned.
func report(w io.Writer, o options, price float64, priceNote string, rate float64, rateNote string) error {
	priceStr := strconv.FormatFloat(price, 'f', 2, 64)
	rateStr := strconv.FormatFloat(rate, 'f', 2, 64)
	valueWidth := max(len(priceStr), len(rateStr))
	withNote := func(v, note string) string {
		if note == "" {
			return v
		}
		return fmt.Sprintf("%-*s  (%s)", valueWidth, v, note)
	}

	rows := [][2]string{
		{"Symbol:", o.symbol},
		{"Price (USD):", withNote(priceStr, priceNote)},
		{"Units:", strconv.FormatFloat(o.units, 'f', -1, 64)},
		{"USD→" + o.currency + ":", withNote(rateStr, rateNote)},
		{"Total value (" + o.currency + "):", strconv.FormatFloat(quote.Value(o.units, price, rate), 'f', 0, 64)},
	}
	labelWidth := 0
	for _, r := range rows {
		labelWidth = max(labelWidth, utf8.RuneCountInString(r[0]))
	}

	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "%-*s %s\n", labelWidth, r[0], r[1])
	}
	_, err := io.WriteString(w, b.String())
	return err
}
