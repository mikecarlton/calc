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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// OpenExchangeRates API schema
type ExchangeRates struct {
	Disclaimer string             `json:"disclaimer"`
	License    string             `json:"license"`
	Timestamp  int64              `json:"timestamp"`
	Base       string             `json:"base"`
	Rates      map[string]float64 `json:"rates"`
}

// Global rates cache
var rates *ExchangeRates

func getAPIKey(source string) (string, error) {
	if apiKey := os.Getenv(source); apiKey != "" {
		return apiKey, nil
	}

	// On macOS, try Keychain if env var not found
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("security", "find-generic-password", "-s", source, "-a", "api_key", "-w")
		output, err := cmd.Output()
		if err == nil {
			apiKey := strings.TrimSpace(string(output))
			if apiKey != "" {
				return apiKey, nil
			}
		}
	}

	// Return error with helpful message if no key found
	return "", fmt.Errorf(`Please set api_key in security (macos) or the environment, e.g.
  export %s=$api_key
or
  security add-generic-password -s %s -a api_key -U -w $api_key`, source, source)
}

func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, "data", "currency")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// returns the appropriate API URL for current or historical rates
func getRatesURL(date string) string {
	baseURL := "https://openexchangerates.org/api"
	if date != "" {
		return fmt.Sprintf("%s/historical/%s.json", baseURL, date)
	}
	return fmt.Sprintf("%s/latest.json", baseURL)
}

// returns the cache file path for rates
func getCacheFile(date string) (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}

	if date != "" {
		return filepath.Join(cacheDir, fmt.Sprintf("%s-rates.json", date)), nil
	}
	return filepath.Join(cacheDir, "rates.json"), nil
}

// performs HTTP GET request with optional token authorization
func httpGet(url, token string) (*ExchangeRates, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP failure '%d' from '%s'", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var exchangeRates ExchangeRates
	if err := json.Unmarshal(body, &exchangeRates); err != nil {
		return nil, err
	}

	return &exchangeRates, nil
}

// loads rates from cache file
func loadRatesFromCache(cacheFile string) (*ExchangeRates, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var exchangeRates ExchangeRates
	if err := json.Unmarshal(data, &exchangeRates); err != nil {
		return nil, err
	}

	return &exchangeRates, nil
}

// saves rates to cache file
func saveRatesToCache(rates *ExchangeRates, cacheFile string) error {
	data, err := json.MarshalIndent(rates, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// checks if the latest rates cache is expired (> 1 hour)
func isRatesCacheExpired(rates *ExchangeRates) bool {
	if rates == nil {
		return true
	}

	// Only check expiration for latest rates (not historical)
	// Historical rates never expire
	if options.date == "" {
		cacheTime := time.Unix(rates.Timestamp, 0)
		return time.Since(cacheTime) > time.Hour
	}

	return false
}

// loads exchange rates from cache or API
func getRates() (*ExchangeRates, error) {
	if rates != nil && !isRatesCacheExpired(rates) {
		return rates, nil
	}

	cacheFile, err := getCacheFile(options.date)
	if err != nil {
		return nil, err
	}

	// Try loading from cache first
	if cachedRates, err := loadRatesFromCache(cacheFile); err == nil {
		if !isRatesCacheExpired(cachedRates) {
			rates = cachedRates
			return rates, nil
		}
	}

	// Cache miss or expired, fetch from API
	apiKey, err := getAPIKey("openexchangerates")
	if err != nil {
		return nil, err
	}

	url := getRatesURL(options.date)
	fetchedRates, err := httpGet(url, apiKey)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := saveRatesToCache(fetchedRates, cacheFile); err != nil {
		// Log error but don't fail the conversion
		fmt.Fprintf(os.Stderr, "Warning: failed to save rates to cache: %v\n", err)
	}

	rates = fetchedRates
	return rates, nil
}

// convertCurrency converts a Number from one currency to another
func convertCurrency(amount *Number, from, to string) (*Number, error) {
	rates, err := getRates()
	if err != nil {
		return nil, err
	}

	fromCurrency := strings.ToUpper(from)
	toCurrency := strings.ToUpper(to)

	// Validate currencies exist in rates
	if fromCurrency != rates.Base {
		if _, exists := rates.Rates[fromCurrency]; !exists {
			return nil, fmt.Errorf("unable to find exchange rate for %s", fromCurrency)
		}
	}

	if toCurrency != rates.Base {
		if _, exists := rates.Rates[toCurrency]; !exists {
			return nil, fmt.Errorf("unable to find exchange rate for %s", toCurrency)
		}
	}

	// Convert via USD base (same logic as Ruby version)
	if fromCurrency == rates.Base {
		// Converting from USD to target currency: amount * rate
		rate := rates.Rates[toCurrency]
		rateNumber := newNumber(strconv.FormatFloat(rate, 'f', -1, 64))
		return mul(amount, rateNumber), nil
	} else if toCurrency == rates.Base {
		// Converting from source currency to USD: amount / rate
		rate := rates.Rates[fromCurrency]
		rateNumber := newNumber(strconv.FormatFloat(rate, 'f', -1, 64))
		return div(amount, rateNumber), nil
	} else {
		// This should be handled by the unit system for non-USD to non-USD conversions
		return nil, fmt.Errorf("invalid usage: convert %s -> %s (must go through USD)", fromCurrency, toCurrency)
	}
}

// Supported currency codes
var supportedCurrencies = map[string]string{
	"usd": "USD",
	"$":   "USD",
	"eur": "EUR",
	"€":   "EUR",
	"gbp": "GBP",
	"£":   "GBP",
	"yen": "JPY",
	"jpy": "JPY",
	"¥":   "JPY",
	"btc": "BTC",
}

// getCurrencyCode normalizes currency symbols to standard codes
func getCurrencyCode(symbol string) (string, bool) {
	code, exists := supportedCurrencies[strings.ToLower(symbol)]
	return code, exists
}

