package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"

	flag "github.com/spf13/pflag"
)

var (
	token       = flag.StringP("token", "t", "xxxxxxxx", "api token for finnhub.io")
	symbol      = flag.StringP("symbol", "s", "NET", "symbol for eg: NET, WORK, EQIX")
	units       = flag.Float64P("units", "u", 1000, "No of stock units")
	stock_price = flag.Float64P("stock_price", "p", 0, "stock price")
	api_url     = "https://finnhub.io/api/v1"
)

func main() {
	flag.Parse()
	req, err := http.NewRequest("GET", api_url+"/quote", nil)
	if err != nil {
		log.Printf("Error making a new request: %s", err)
		os.Exit(1)
	}
	q := req.URL.Query()
	q.Add("token", *token)
	q.Add("symbol", *symbol)
	req.URL.RawQuery = q.Encode()

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error gettting response: %s", err)
		os.Exit(1)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %s", err)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		log.Printf("Got response code : %v\n", resp.StatusCode)
		log.Println(string(body))
		os.Exit(2)
	}

	quote := &Quote{}
	if *stock_price == 0 {
		err = json.Unmarshal(body, quote)
		if err != nil {
			log.Printf("Error unmarshalling struct: %s", err)
			os.Exit(1)
		}
		*stock_price = quote.C
	}

	// Get forex rates for INR and USD
	req, err = http.NewRequest("GET", api_url+"/forex/rates", nil)
	if err != nil {
		log.Printf("Error making a new request: %s", err)
		os.Exit(1)
	}
	q = req.URL.Query()
	q.Add("token", *token)
	req.URL.RawQuery = q.Encode()

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Error gettting response: %s", err)
		os.Exit(1)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %s", err)
		os.Exit(1)
	}
	forex := &Forex{}
	err = json.Unmarshal(body, forex)
	if err != nil {
		log.Printf("Error unmarshalling struct: %s", err)
		os.Exit(1)
	}

	usd_to_inr := forex.Quote.INR / forex.Quote.USD
	current_value := math.Round(*units * usd_to_inr * *stock_price)
	fmt.Printf(`Stock price : %.2f
Number of stocks: %.0f
Usd to INR: %.2f
Total value INR: %.0f
`, quote.C, *units, usd_to_inr, current_value)
}

type Quote struct {
	C  float64 `json:"c"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	O  float64 `json:"o"`
	Pc float64 `json:"pc"`
	T  int     `json:"t"`
}

type Forex struct {
	Quote struct {
		INR float64 `json:"INR"`
		USD float64 `json:"USD"`
		SGD float64 `json:"SGD"`
	} `json:"quote"`
}
