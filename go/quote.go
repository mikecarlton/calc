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

// BatchQuoteResponse is the response when fetching multiple quotes
type BatchQuoteResponse map[string]QuoteResponse

var tickerPattern = regexp.MustCompile(`^@([a-zA-Z]+)$`)

// Global cache for pre-fetched quotes
var preFetchedQuotes = make(map[string]Value)
var preFetchedQuoteData = make(map[string]*QuoteResponse)
var preFetchedQuoteTypeData = make(map[string]QuoteType)

// Global map to track quotes actually used in calculations (for -d detail option)
var usedQuotes = make(map[string]*QuoteResponse)
var usedQuoteTypes = make(map[string]QuoteType)

// isTickerSymbol checks if the input string is a ticker symbol (e.g., @aapl)
func isTickerSymbol(input string) (string, bool) {
	matches := tickerPattern.FindStringSubmatch(input)
	if len(matches) == 2 {
		return strings.ToUpper(matches[1]), true
	}
	return "", false
}

// preFetchStockQuotes scans all arguments and batch fetches stock quotes
func preFetchStockQuotes(args []string) {
	// Collect all unique ticker symbols
	symbolsMap := make(map[string]bool)
	for _, arg := range args {
		parts := strings.Fields(arg)
		for _, part := range parts {
			if ticker, ok := isTickerSymbol(part); ok {
				symbolsMap[ticker] = true
			}
		}
	}

	// If no symbols found, return early
	if len(symbolsMap) == 0 {
		return
	}

	// Convert map to slice
	symbols := make([]string, 0, len(symbolsMap))
	for symbol := range symbolsMap {
		symbols = append(symbols, symbol)
	}

	// Check which symbols need to be fetched (vs using cache)
	symbolsToFetch := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		if shouldFetchQuote(symbol, options.extended) {
			symbolsToFetch = append(symbolsToFetch, symbol)
		} else {
			// Try to get from cache
			cached, err := getLatestQuote(symbol, QuoteTypeRegular)
			if err == nil && cached != nil {
				// Convert cached quote to Value and store
				quote := &QuoteResponse{
					Symbol:           cached.Symbol,
					Name:             cached.Name,
					Exchange:         cached.Exchange,
					Currency:         cached.Currency,
					Datetime:         cached.Datetime,
					Timestamp:        cached.Timestamp,
					Open:             cached.Open,
					High:             cached.High,
					Low:              cached.Low,
					Close:            cached.Close,
					Volume:           cached.Volume,
					PreviousClose:    cached.PreviousClose,
					Change:           cached.Change,
					PercentChange:    cached.PercentChange,
					AverageVolume:    cached.AverageVolume,
					FiftyTwoWeekLow:  cached.FiftyTwoWeekLow,
					FiftyTwoWeekHigh: cached.FiftyTwoWeekHigh,
					IsMarketOpen:     cached.IsMarketOpen,
				}
				preFetchedQuotes[symbol] = quoteToValue(quote)
				preFetchedQuoteData[symbol] = quote
				preFetchedQuoteTypeData[symbol] = QuoteTypeRegular
			} else {
				// Cache miss, need to fetch
				symbolsToFetch = append(symbolsToFetch, symbol)
			}
		}
	}

	// Batch fetch all required symbols
	if len(symbolsToFetch) > 0 {
		quotes, err := fetchQuotes(symbolsToFetch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch quotes: %v\n", err)
			return
		}

		// Process each fetched quote
		for symbol, quote := range quotes {
			// Determine quote type
			quoteType := determineQuoteType(quote)

			// Check if this is a closing price
			isClosing := quoteType == QuoteTypeRegular && !quote.IsMarketOpen

			// Save to database cache
			if err := saveQuote(quote, quoteType, isClosing); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to cache quote for %s: %v\n", symbol, err)
			}

			// If this is a closing price, update yesterday's data if needed
			if isClosing {
				quoteDate := time.Unix(quote.Timestamp, 0)
				yesterday := quoteDate.AddDate(0, 0, -1).Format("2006-01-02")
				hasClosing, err := hasClosingPrice(symbol, yesterday)
				if err == nil && !hasClosing {
					if quote.PreviousClose != "" {
						if err := updateClosingPrice(symbol, yesterday, quote.PreviousClose); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to update previous closing price for %s: %v\n", symbol, err)
						}
					}
				}
			}

			// Store in memory cache
			preFetchedQuotes[symbol] = quoteToValue(quote)
			preFetchedQuoteData[symbol] = quote
			preFetchedQuoteTypeData[symbol] = quoteType
		}
	}
}

// getStockQuoteFromCache retrieves a pre-fetched stock quote
func getStockQuoteFromCache(symbol string) (Value, error) {
	value, ok := preFetchedQuotes[symbol]
	if !ok {
		// Fallback to individual fetch if not in cache
		return getStockQuote(symbol)
	}

	// Record that this quote was used (for -d detail option)
	if quoteData, ok := preFetchedQuoteData[symbol]; ok {
		usedQuotes[symbol] = quoteData
		if quoteType, ok := preFetchedQuoteTypeData[symbol]; ok {
			usedQuoteTypes[symbol] = quoteType
		}
	}

	return value, nil
}

// fetchQuotes fetches stock quotes from TwelveData API (supports batch requests)
func fetchQuotes(symbols []string) (map[string]*QuoteResponse, error) {
	if len(symbols) == 0 {
		return map[string]*QuoteResponse{}, nil
	}

	apiKey, err := getAPIKey("twelvedata")
	if err != nil {
		return nil, err
	}

	// Join symbols with comma for batch request
	symbolList := strings.Join(symbols, ",")

	// Add extended_hours parameter if requested
	extendedParam := ""
	if options.extended {
		extendedParam = "&prepost=true"
	}

	url := fmt.Sprintf("https://api.twelvedata.com/quote?symbol=%s&apikey=%s%s",
		symbolList, apiKey, extendedParam)

	// Print URL in blue if debug mode is enabled
	if options.debug {
		fmt.Fprintf(os.Stderr, "%s\n", blue(url))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP failure '%d' from TwelveData API", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Print JSON response in green if debug mode is enabled
	if options.debug {
		fmt.Fprintf(os.Stderr, "%s\n", green(string(body)))
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

	results := make(map[string]*QuoteResponse)

	// Handle single symbol response (returns object) vs batch (returns map)
	if len(symbols) == 1 {
		var quote QuoteResponse
		if err := json.Unmarshal(body, &quote); err != nil {
			return nil, fmt.Errorf("failed to parse quote response: %v", err)
		}

		// Validate we got a valid quote
		if quote.Symbol == "" || quote.Close == "" {
			return nil, fmt.Errorf("ticker symbol '%s' not found or incomplete data", symbols[0])
		}

		results[strings.ToUpper(symbols[0])] = &quote
	} else {
		// Batch response
		var batchResponse BatchQuoteResponse
		if err := json.Unmarshal(body, &batchResponse); err != nil {
			return nil, fmt.Errorf("failed to parse batch quote response: %v", err)
		}

		for symbol, quote := range batchResponse {
			if quote.Symbol == "" || quote.Close == "" {
				fmt.Fprintf(os.Stderr, "Warning: incomplete data for symbol '%s'\n", symbol)
				continue
			}
			q := quote // Create a copy to avoid pointer issues
			results[strings.ToUpper(symbol)] = &q
		}
	}

	return results, nil
}

// fetchQuote fetches a single stock quote (legacy function, now uses batch API)
func fetchQuote(symbol string) (*QuoteResponse, error) {
	results, err := fetchQuotes([]string{symbol})
	if err != nil {
		return nil, err
	}

	quote, ok := results[strings.ToUpper(symbol)]
	if !ok {
		return nil, fmt.Errorf("ticker symbol '%s' not found", symbol)
	}

	return quote, nil
}

// determineQuoteType determines the type of quote based on market hours and data
func determineQuoteType(quote *QuoteResponse) QuoteType {
	// If we're not requesting extended hours, always treat as regular
	if !options.extended {
		return QuoteTypeRegular
	}

	// Load Eastern timezone
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}

	quoteTime := time.Unix(quote.Timestamp, 0).In(loc)

	// Pre-market: before 9:30 AM ET
	marketOpen := time.Date(quoteTime.Year(), quoteTime.Month(), quoteTime.Day(), 9, 30, 0, 0, loc)
	if quoteTime.Before(marketOpen) {
		return QuoteTypePreMarket
	}

	// Market hours: 9:30 AM - 4:00 PM ET
	marketClose := time.Date(quoteTime.Year(), quoteTime.Month(), quoteTime.Day(), 16, 0, 0, 0, loc)
	if quoteTime.Before(marketClose) {
		return QuoteTypeRegular
	}

	// Post-market: after 4:00 PM ET
	return QuoteTypePostMarket
}

// getStockQuote fetches a stock quote with caching and returns it as a Value
func getStockQuote(symbol string) (Value, error) {
	// Check if we should use cached data
	if !shouldFetchQuote(symbol, options.extended) {
		cached, err := getLatestQuote(symbol, QuoteTypeRegular)
		if err == nil && cached != nil {
			if options.debug || options.trace {
				fmt.Fprintf(os.Stderr, "Using cached quote for %s from %s\n", symbol, cached.Date)
			}

			// Convert cached quote to QuoteResponse for display and processing
			quote := &QuoteResponse{
				Symbol:           cached.Symbol,
				Name:             cached.Name,
				Exchange:         cached.Exchange,
				Currency:         cached.Currency,
				Datetime:         cached.Datetime,
				Timestamp:        cached.Timestamp,
				Open:             cached.Open,
				High:             cached.High,
				Low:              cached.Low,
				Close:            cached.Close,
				Volume:           cached.Volume,
				PreviousClose:    cached.PreviousClose,
				Change:           cached.Change,
				PercentChange:    cached.PercentChange,
				AverageVolume:    cached.AverageVolume,
				FiftyTwoWeekLow:  cached.FiftyTwoWeekLow,
				FiftyTwoWeekHigh: cached.FiftyTwoWeekHigh,
				IsMarketOpen:     cached.IsMarketOpen,
			}

			return quoteToValue(quote), nil
		}
	}

	// Fetch fresh quote
	quote, err := fetchQuote(symbol)
	if err != nil {
		return Value{}, err
	}

	// Determine quote type
	quoteType := determineQuoteType(quote)

	// Check if this is a closing price (market just closed or after hours)
	isClosing := quoteType == QuoteTypeRegular && !quote.IsMarketOpen

	// Save to cache
	if err := saveQuote(quote, quoteType, isClosing); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache quote: %v\n", err)
	}

	// If this is a closing price, also check if we need to update yesterday's data
	if isClosing {
		quoteDate := time.Unix(quote.Timestamp, 0)
		yesterday := quoteDate.AddDate(0, 0, -1).Format("2006-01-02")
		hasClosing, err := hasClosingPrice(symbol, yesterday)
		if err == nil && !hasClosing {
			// Update yesterday's quote with the previous_close value
			if quote.PreviousClose != "" {
				if err := updateClosingPrice(symbol, yesterday, quote.PreviousClose); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to update previous closing price: %v\n", err)
				}
			}
		}
	}

	// Display verbose quote information if requested
	if options.debug || options.trace {
		printQuoteInfo(quote)
		fmt.Fprintf(os.Stderr, "Quote Type: %s\n", quoteType)
		if isClosing {
			fmt.Fprintf(os.Stderr, "Closing Price: Yes\n")
		}
	}

	return quoteToValue(quote), nil
}

// quoteToValue converts a QuoteResponse to a Value
func quoteToValue(quote *QuoteResponse) Value {
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
	}
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

// printDetailedQuoteSummary prints detailed information for all quotes used in calculations
func printDetailedQuoteSummary() {
	if len(usedQuotes) == 0 {
		return
	}

	// Sort symbols for consistent output
	symbols := make([]string, 0, len(usedQuotes))
	for symbol := range usedQuotes {
		symbols = append(symbols, symbol)
	}

	// Simple bubble sort to avoid importing sort package
	for i := 0; i < len(symbols); i++ {
		for j := i + 1; j < len(symbols); j++ {
			if symbols[i] > symbols[j] {
				symbols[i], symbols[j] = symbols[j], symbols[i]
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
	// Print header row
	fmt.Fprintf(os.Stderr, "%-8s %-10s %12s %18s %20s %20s %15s %15s %12s %-6s %-11s %-19s %s\n",
		"Symbol", "Exchange", "Price", "Change", "Day Range", "52-Week Range", "Volume", "Avg Volume", "Prev Close", "Status", "Type", "Updated", "Name")

	for _, symbol := range symbols {
		quote := usedQuotes[symbol]
		quoteType := usedQuoteTypes[symbol]

		// Format price with currency
		priceStr := fmt.Sprintf("%s %s", quote.Close, quote.Currency)

		// Format change with percentage
		changeStr := ""
		if quote.Change != "" && quote.PercentChange != "" {
			change, _ := strconv.ParseFloat(quote.Change, 64)
			pctChange, _ := strconv.ParseFloat(quote.PercentChange, 64)
			changeText := fmt.Sprintf("%+.2f (%+.2f%%)", change, pctChange)
			if change < 0 {
				changeStr = red(changeText)
			} else if change > 0 {
				changeStr = green(changeText)
			} else {
				changeStr = changeText
			}
		}

		// Format day range
		dayRangeStr := ""
		if quote.Low != "" && quote.High != "" {
			dayRangeStr = fmt.Sprintf("%s - %s", quote.Low, quote.High)
		}

		// Format 52-week range
		weekRangeStr := ""
		if quote.FiftyTwoWeekLow != "" && quote.FiftyTwoWeekHigh != "" {
			weekRangeStr = fmt.Sprintf("%s - %s", quote.FiftyTwoWeekLow, quote.FiftyTwoWeekHigh)
		}

		// Format quote type
		typeStr := string(quoteType)
		if quoteType == QuoteTypeRegular {
			typeStr = "regular"
		}

		// Print the row
		fmt.Fprintf(os.Stderr, "%-8s %-10s %12s %18s %20s %20s %15s %15s %12s %-6s %-11s %-19s %s\n",
			quote.Symbol,
			quote.Exchange,
			priceStr,
			changeStr,
			dayRangeStr,
			weekRangeStr,
			quote.Volume,
			quote.AverageVolume,
			quote.PreviousClose,
			marketStatus(quote.IsMarketOpen),
			typeStr,
			quote.Datetime,
			quote.Name)
	}
	fmt.Fprintf(os.Stderr, "\n")
}
