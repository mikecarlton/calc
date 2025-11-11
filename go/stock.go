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
	"strings"
	"time"
)

// IEXQuoteResponse represents the JSON response from IEX Cloud API
type IEXQuoteResponse struct {
	Symbol          string  `json:"symbol"`
	CompanyName     string  `json:"companyName"`
	LatestPrice     float64 `json:"latestPrice"`
	Currency        string  `json:"currency"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"changePercent"`
	Low             float64 `json:"low"`
	High            float64 `json:"high"`
	Week52Low       float64 `json:"week52Low"`
	Week52High      float64 `json:"week52High"`
	YtdChange       float64 `json:"ytdChange"`
	MarketCap       int64   `json:"marketCap"`
	PeRatio         float64 `json:"peRatio"`
	LatestUpdate    int64   `json:"latestUpdate"`
	IsUSMarketOpen  string  `json:"isUSMarketOpen"`
}

// QuoteRow represents a formatted quote row for table display
type QuoteRow []string

var globalQuotes []QuoteRow
var globalQuoteHeaders []string

func httpGetStock(url string) ([]byte, error) {
	if options.trace {
		fmt.Fprintf(os.Stderr, "[get(%s)]\n", url)
	}
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response: %v", err)
	}

	return body, nil
}

func formatValue(val interface{}, currency string, unit string, delta bool) string {
	var result string
	
	switch v := val.(type) {
	case float64:
		if delta {
			result = fmt.Sprintf("%+.02f", v)
			if v < 0 {
				result = red(result)
			} else {
				result = green(result)
			}
		} else {
			result = fmt.Sprintf("%.02f", v)
		}
	case int64:
		if delta {
			result = fmt.Sprintf("%+d", v)
			if v < 0 {
				result = red(result)
			} else {
				result = green(result)
			}
		} else {
			result = fmt.Sprintf("%d", v)
		}
	case string:
		result = v
	default:
		result = fmt.Sprintf("%v", val)
	}
	
	return result + unit
}

func getStockQuote(symbol string) (*Value, error) {
	apiKey, err := getAPIKey("iex")
	if err != nil {
		return nil, fmt.Errorf("IEX API key not found: %v", err)
	}

	quoteURL := fmt.Sprintf("https://api.iex.cloud/v1/data/core/quote/%s?token=%s", symbol, apiKey)
	
	body, err := httpGetStock(quoteURL)
	if err != nil {
		return nil, fmt.Errorf("Unable to get quote: %v", err)
	}

	var responses []IEXQuoteResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		return nil, fmt.Errorf("Failed to parse quote response: %v", err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("No quote data returned")
	}

	response := responses[0]
	if response.LatestPrice == 0 || response.Currency == "" {
		return nil, fmt.Errorf("Invalid quote data")
	}

	// Handle verbose output
	if options.verbose > 0 {
		cur := "$"
		if response.Currency == "USD" {
			cur = "$"
		}

		quote := []string{
			formatValue(response.Symbol, "", "", false),
			formatValue(response.LatestPrice, cur, "", false),
			formatValue(response.Change, cur, "", true),
			formatValue(response.ChangePercent*100, "", "%", true),
			formatValue(response.Low, cur, "", false),
			formatValue(response.High, cur, "", false),
			formatValue(response.Week52Low, cur, "", false),
			formatValue(response.Week52High, cur, "", false),
			formatValue(response.YtdChange*100, "", "%", true),
			formatValue(float64(response.MarketCap)/1e9, cur, "B", false),
			formatValue(response.PeRatio, "", "", false),
			formatValue(time.Unix(response.LatestUpdate/1000, 0).Format("2006-01-02 03:04:05 PM"), "", "", false),
		}

		if options.verbose > 1 {
			quote = append(quote, response.CompanyName)
		}

		// Initialize headers if not done yet
		if globalQuoteHeaders == nil {
			isOpen := response.IsUSMarketOpen == "true"
			globalQuoteHeaders = []string{"Symbol", "Price", "Change", "Change%", "Low", "High", "52wLow", "52wHigh", "YTD%", "MktCap", "P/E", "Time"}
			if isOpen {
				globalQuoteHeaders[1] = "Price"
			} else {
				globalQuoteHeaders[1] = "Price*"
			}
			if options.verbose > 1 {
				globalQuoteHeaders = append(globalQuoteHeaders, "Company")
			}
		}

		globalQuotes = append(globalQuotes, quote)
	}

	// Create the value to return
	price := newNumberFromFloat64(response.LatestPrice)
	currency := strings.ToLower(response.Currency)
	
	var units Unit
	if currency == "usd" {
		// Add USD currency unit if we have currency support
		// For now, just return the raw number
	}

	return &Value{number: price, units: units}, nil
}

func showQuotes() {
	if len(globalQuotes) == 0 {
		return
	}
	
	// Simple table printing - could be enhanced
	fmt.Fprintf(os.Stderr, "\n")
	
	// Print headers
	if globalQuoteHeaders != nil {
		for i, header := range globalQuoteHeaders {
			if i > 0 {
				fmt.Fprintf(os.Stderr, "  ")
			}
			fmt.Fprintf(os.Stderr, "%-12s", header)
		}
		fmt.Fprintf(os.Stderr, "\n")
		
		// Print separator
		for i := range globalQuoteHeaders {
			if i > 0 {
				fmt.Fprintf(os.Stderr, "  ")
			}
			fmt.Fprintf(os.Stderr, "%-12s", strings.Repeat("-", 12))
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
	
	// Print data rows
	for _, quote := range globalQuotes {
		for i, cell := range quote {
			if i > 0 {
				fmt.Fprintf(os.Stderr, "  ")
			}
			fmt.Fprintf(os.Stderr, "%-12s", cell)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
}