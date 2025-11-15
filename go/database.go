// Copyright 2024 Mike Carlton
// Released under terms of the MIT License:
//   http://www.opensource.org/licenses/mit-license.php

package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type QuoteType string

const (
	QuoteTypePreMarket  QuoteType = "pre-market"
	QuoteTypeRegular    QuoteType = "regular"
	QuoteTypePostMarket QuoteType = "post-market"
)

type CachedQuote struct {
	Symbol           string
	Date             string // YYYY-MM-DD format
	QuoteType        QuoteType
	IsClosing        bool
	Name             string
	Exchange         string
	Currency         string
	Datetime         string
	Timestamp        int64
	Open             string
	High             string
	Low              string
	Close            string
	Volume           string
	PreviousClose    string
	Change           string
	PercentChange    string
	AverageVolume    string
	FiftyTwoWeekLow  string
	FiftyTwoWeekHigh string
	IsMarketOpen     bool
	CreatedAt        time.Time
}

var db *sql.DB

// initDatabase initializes the SQLite database for quote caching
func initDatabase() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	dataDir := filepath.Join(homeDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "stock-quotes.sqlite3")
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Create the quotes table
	schema := `
	CREATE TABLE IF NOT EXISTS quotes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		date TEXT NOT NULL,
		quote_type TEXT NOT NULL,
		is_closing BOOLEAN NOT NULL DEFAULT 0,
		name TEXT,
		exchange TEXT,
		currency TEXT,
		datetime TEXT,
		timestamp INTEGER,
		open TEXT,
		high TEXT,
		low TEXT,
		close TEXT,
		volume TEXT,
		previous_close TEXT,
		change TEXT,
		percent_change TEXT,
		average_volume TEXT,
		fifty_two_week_low TEXT,
		fifty_two_week_high TEXT,
		is_market_open BOOLEAN,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(symbol, date, quote_type)
	);

	CREATE INDEX IF NOT EXISTS idx_symbol_date ON quotes(symbol, date);
	CREATE INDEX IF NOT EXISTS idx_symbol_date_type ON quotes(symbol, date, quote_type);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %v", err)
	}

	return nil
}

// closeDatabase closes the database connection
func closeDatabase() {
	if db != nil {
		db.Close()
	}
}

// saveQuote saves or updates a quote in the database
func saveQuote(quote *QuoteResponse, quoteType QuoteType, isClosing bool) error {
	if db == nil {
		if err := initDatabase(); err != nil {
			return err
		}
	}

	// Get the date from the quote timestamp
	quoteDate := time.Unix(quote.Timestamp, 0).Format("2006-01-02")

	// Insert or replace the quote (keeping only the latest for each type per day)
	query := `
	INSERT OR REPLACE INTO quotes (
		symbol, date, quote_type, is_closing,
		name, exchange, currency, datetime, timestamp,
		open, high, low, close, volume,
		previous_close, change, percent_change, average_volume,
		fifty_two_week_low, fifty_two_week_high, is_market_open
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		quote.Symbol, quoteDate, quoteType, isClosing,
		quote.Name, quote.Exchange, quote.Currency, quote.Datetime, quote.Timestamp,
		quote.Open, quote.High, quote.Low, quote.Close, quote.Volume,
		quote.PreviousClose, quote.Change, quote.PercentChange, quote.AverageVolume,
		quote.FiftyTwoWeekLow, quote.FiftyTwoWeekHigh, quote.IsMarketOpen,
	)

	return err
}

// getLatestQuote retrieves the most recent cached quote for a symbol
func getLatestQuote(symbol string, quoteType QuoteType) (*CachedQuote, error) {
	if db == nil {
		if err := initDatabase(); err != nil {
			return nil, err
		}
	}

	query := `
	SELECT symbol, date, quote_type, is_closing,
		name, exchange, currency, datetime, timestamp,
		open, high, low, close, volume,
		previous_close, change, percent_change, average_volume,
		fifty_two_week_low, fifty_two_week_high, is_market_open,
		created_at
	FROM quotes
	WHERE symbol = ? AND quote_type = ?
	ORDER BY date DESC, created_at DESC
	LIMIT 1
	`

	var cached CachedQuote
	err := db.QueryRow(query, symbol, quoteType).Scan(
		&cached.Symbol, &cached.Date, &cached.QuoteType, &cached.IsClosing,
		&cached.Name, &cached.Exchange, &cached.Currency, &cached.Datetime, &cached.Timestamp,
		&cached.Open, &cached.High, &cached.Low, &cached.Close, &cached.Volume,
		&cached.PreviousClose, &cached.Change, &cached.PercentChange, &cached.AverageVolume,
		&cached.FiftyTwoWeekLow, &cached.FiftyTwoWeekHigh, &cached.IsMarketOpen,
		&cached.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &cached, nil
}

// updateClosingPrice updates a previous day's quote to mark it as closing price
func updateClosingPrice(symbol, date, closePrice string) error {
	if db == nil {
		if err := initDatabase(); err != nil {
			return err
		}
	}

	query := `
	UPDATE quotes
	SET is_closing = 1, close = ?
	WHERE symbol = ? AND date = ? AND quote_type = ?
	`

	_, err := db.Exec(query, closePrice, symbol, date, QuoteTypeRegular)
	return err
}

// hasClosingPrice checks if we have a closing price for a given symbol and date
func hasClosingPrice(symbol, date string) (bool, error) {
	if db == nil {
		if err := initDatabase(); err != nil {
			return false, err
		}
	}

	query := `
	SELECT COUNT(*)
	FROM quotes
	WHERE symbol = ? AND date = ? AND quote_type = ? AND is_closing = 1
	`

	var count int
	err := db.QueryRow(query, symbol, date, QuoteTypeRegular).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// isMarketHours checks if the current time is within market hours (9:30 AM - 4:00 PM ET)
func isMarketHours() bool {
	// Load Eastern timezone
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback to UTC if timezone loading fails
		loc = time.UTC
	}

	now := time.Now().In(loc)

	// Check if it's a weekend
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	// Market hours: 9:30 AM - 4:00 PM ET
	marketOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, loc)
	marketClose := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, loc)

	return now.After(marketOpen) && now.Before(marketClose)
}

// shouldFetchQuote determines if we should fetch a fresh quote or use cached data
func shouldFetchQuote(symbol string, requestExtended bool) bool {
	// If market is open and we're requesting regular quotes, always fetch
	if isMarketHours() && !requestExtended {
		return true
	}

	// If market is closed and we're requesting regular quotes, check cache
	if !isMarketHours() && !requestExtended {
		// Try to get cached quote
		cached, err := getLatestQuote(symbol, QuoteTypeRegular)
		if err != nil || cached == nil {
			return true // No cache, need to fetch
		}

		// Check if cached quote is from today
		today := time.Now().Format("2006-01-02")
		if cached.Date == today {
			return false // Have fresh cache, don't fetch
		}

		return true // Cache is old, need to fetch
	}

	// For extended hours requests, always fetch
	return true
}
