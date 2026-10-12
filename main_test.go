package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProvider struct {
	price, rate   float64
	priceErr      error
	priceCalls    int
	usdToCurrency string
	usdToCalls    int
}

func (f *fakeProvider) Price(ctx context.Context, symbol string) (float64, time.Time, error) {
	f.priceCalls++
	return f.price, time.Time{}, f.priceErr
}

func (f *fakeProvider) USDTo(ctx context.Context, currency string) (float64, time.Time, error) {
	f.usdToCalls++
	f.usdToCurrency = currency
	return f.rate, time.Time{}, nil
}

func TestRunUserSuppliedPrice(t *testing.T) {
	p := &fakeProvider{price: 999, rate: 84.12}
	var out bytes.Buffer

	err := run(context.Background(), &out, p, options{symbol: "NET", units: 1000, price: 123.45})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if p.priceCalls != 0 {
		t.Errorf("Price called %d times, want 0", p.priceCalls)
	}
	want := `Stock price : 123.45
Number of stocks: 1000
Usd to INR: 84.12
Total value INR: 10384614
`
	if got := out.String(); got != want {
		t.Errorf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunFetchesPrice(t *testing.T) {
	p := &fakeProvider{price: 10, rate: 80}
	var out bytes.Buffer

	err := run(context.Background(), &out, p, options{symbol: "NET", units: 3})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if p.priceCalls != 1 {
		t.Errorf("Price called %d times, want 1", p.priceCalls)
	}
	if p.usdToCurrency != "INR" {
		t.Errorf("USDTo currency = %q, want INR", p.usdToCurrency)
	}
	want := `Stock price : 10.00
Number of stocks: 3
Usd to INR: 80.00
Total value INR: 2400
`
	if got := out.String(); got != want {
		t.Errorf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunPriceError(t *testing.T) {
	wantErr := errors.New("no price")
	p := &fakeProvider{priceErr: wantErr}

	err := run(context.Background(), &bytes.Buffer{}, p, options{symbol: "NOPE", units: 1})
	if !errors.Is(err, wantErr) {
		t.Fatalf("run error = %v, want %v", err, wantErr)
	}
	if p.usdToCalls != 0 {
		t.Errorf("USDTo called %d times after Price failed, want 0", p.usdToCalls)
	}
}
