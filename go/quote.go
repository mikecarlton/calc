// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TwelveData API response schema for quote endpoint
type QuoteResponse struct {
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	Exchange         string `json:"exchange"`
	Currency         string `json:"currency"`
	Datetime         string `json:"datetime"`
	Timestamp        int64  `json:"timestamp"`
	Open             string `json:"open"`
	High             string `json:"high"`
	Low              string `json:"low"`
	Close            string `json:"close"`
	Volume           string `json:"volume"`
	PreviousClose    string `json:"previous_close"`
	Change           string `json:"change"`
	PercentChange    string `json:"percent_change"`
	AverageVolume    string `json:"average_volume"`
	FiftyTwoWeekLow  string `json:"fifty_two_week.low"`
	FiftyTwoWeekHigh string `json:"fifty_two_week.high"`
	IsMarketOpen     bool   `json:"is_market_open"`
}

var tickerPattern = regexp.MustCompile(`^@([a-zA-Z]+)$`)

// isTickerSymbol checks if the input string is a ticker symbol (e.g., @aapl)
func isTickerSymbol(input string) (string, bool) {
	matches := tickerPattern.FindStringSubmatch(input)
	if len(matches) == 2 {
		return strings.ToUpper(matches[1]), true
	}
	return "", false
}

// fetchQuote fetches a stock quote from TwelveData API
func fetchQuote(symbol string) (*QuoteResponse, error) {
	apiKey, err := getAPIKey("twelvedata")
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.twelvedata.com/quote?symbol=%s&apikey=%s", symbol, apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP failure '%d' from TwelveData API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check for API error response
	var errorResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(body, &errorResponse); err == nil {
		if errorResponse.Status == "error" {
			return nil, fmt.Errorf("API error: %s", errorResponse.Message)
		}
	}

	var quote QuoteResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		return nil, fmt.Errorf("failed to parse quote response: %v", err)
	}

	// Validate we got a valid quote
	if quote.Symbol == "" || quote.Close == "" {
		return nil, fmt.Errorf("ticker symbol '%s' not found or incomplete data", symbol)
	}

	return &quote, nil
}

// getStockQuote fetches a stock quote and returns it as a Value
func getStockQuote(symbol string) (Value, error) {
	quote, err := fetchQuote(symbol)
	if err != nil {
		return Value{}, err
	}

	// Display verbose quote information if requested
	if options.debug || options.trace {
		printQuoteInfo(quote)
	}

	// Create a Value with the price
	priceNumber := newNumber(quote.Close)

	// Get the currency unit
	currencyCode := strings.ToLower(quote.Currency)
	var units Unit

	// Try to find the currency unit in the UNITS map
	if currencyUnit, ok := UNITS[currencyCode]; ok {
		units = currencyUnit
	} else {
		// Default to USD if currency not found
		if usdUnit, ok := UNITS["usd"]; ok {
			units = usdUnit
		}
	}

	return Value{
		number: priceNumber,
		units:  units,
	}, nil
}

// printQuoteInfo displays detailed quote information
func printQuoteInfo(quote *QuoteResponse) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Symbol:         %s\n", quote.Symbol)
	if quote.Name != "" {
		fmt.Fprintf(os.Stderr, "Name:           %s\n", quote.Name)
	}
	fmt.Fprintf(os.Stderr, "Exchange:       %s\n", quote.Exchange)
	fmt.Fprintf(os.Stderr, "Currency:       %s\n", quote.Currency)
	fmt.Fprintf(os.Stderr, "Price:          %s\n", quote.Close)

	if quote.Change != "" {
		change, _ := strconv.ParseFloat(quote.Change, 64)
		changeStr := fmt.Sprintf("%+.2f", change)
		if change < 0 {
			changeStr = red(changeStr)
		} else if change > 0 {
			changeStr = green(changeStr)
		}
		fmt.Fprintf(os.Stderr, "Change:         %s", changeStr)

		if quote.PercentChange != "" {
			pctChange, _ := strconv.ParseFloat(quote.PercentChange, 64)
			pctStr := fmt.Sprintf("%+.2f%%", pctChange)
			if pctChange < 0 {
				pctStr = red(pctStr)
			} else if pctChange > 0 {
				pctStr = green(pctStr)
			}
			fmt.Fprintf(os.Stderr, " (%s)", pctStr)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	if quote.Low != "" && quote.High != "" {
		fmt.Fprintf(os.Stderr, "Day Range:      %s - %s\n", quote.Low, quote.High)
	}

	if quote.FiftyTwoWeekLow != "" && quote.FiftyTwoWeekHigh != "" {
		fmt.Fprintf(os.Stderr, "52-Week Range:  %s - %s\n", quote.FiftyTwoWeekLow, quote.FiftyTwoWeekHigh)
	}

	if quote.Volume != "" {
		fmt.Fprintf(os.Stderr, "Volume:         %s\n", quote.Volume)
	}

	fmt.Fprintf(os.Stderr, "Market Status:  %s\n", marketStatus(quote.IsMarketOpen))
	fmt.Fprintf(os.Stderr, "Last Updated:   %s\n", quote.Datetime)
	fmt.Fprintf(os.Stderr, "\n")
}

func marketStatus(isOpen bool) string {
	if isOpen {
		return green("OPEN")
	}
	return "CLOSED"
}
