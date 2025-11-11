# CLAUDE.md

## Overview

This is a calculator project with two implementations:
- **Ruby**: A complete, production-ready RPN (Reverse Polish Notation) calculator (`calc.rb`)
- **Go**: A work-in-progress port of the Ruby calculator to Go (`go/` directory)

The calculator operates as a stack-based calculator supporting arithmetic operations, units conversions, and various mathematical functions.

## Commands

* The ruby version is stable and should not be modified.
* After every set of changes, always verify that the code compiles and successfully runs a simple calculation
* Never delete tests unless requested to
* The Go version is switching to represent the Number structure as a Go Rational
* * The intent is to preserve precision to the greatest extent possible.  E.g. 1 divided by 3 and thenmultiplied by 3 should always return exactly 1

### Running the Calculator

**Ruby version (working):**
```bash
ruby calc.rb [arguments]
```

**Go version (in development):**
```bash
cd go && go build . && ./calc [arguments]
```

Note: The Go version currently has compilation errors and is incomplete.

### Testing Examples

```bash
# Basic arithmetic
ruby calc.rb 3 4 +
ruby calc.rb 10 2 /
ruby calc.rb 5 2 **

# With units
ruby calc.rb 100 cm m
ruby calc.rb 32 f c

# Stack operations
ruby calc.rb 3 4 d + x -
```

## Architecture

### Ruby Implementation
- **Single-file architecture** (`calc.rb`) with all functionality
- **Stack-based computation** using Array as stack
- **Units system** with dimensional analysis and automatic conversions
- **Number types**: Integers, BigDecimal for floats, Rationals for fractions
- **External integrations**: Stock quotes (IEX), currency conversion (OpenExchangeRates)

### Go Implementation (WIP)
- **Modular architecture** split across multiple files:
  - `calc.go` - Main entry point and argument processing
  - `number.go` - Arbitrary precision arithmetic using `math/big`
  - `stack.go` - Stack operations and management  
  - `value.go` - Values with units and operations
  - `unit.go` - Units definitions and conversions
  - `options.go` - Command-line option handling

### Key Patterns
- **RPN evaluation**: All operations work on stack elements
- **Units as types**: Units are strongly typed with dimensional analysis
- **Operator dispatch**: Operations are looked up in maps/tables
- **Error handling**: Ruby uses exceptions; Go uses error returns

### Current Status
- Ruby version is complete and fully functional
- Go version has compilation errors and missing implementations
- Both versions share the same conceptual architecture but different implementation approaches

## Development Notes

When working on the Go version:
- Fix compilation errors related to undefined `newNumber` function in `unit.go`
- Complete missing arithmetic operations in number.go
- Implement parsing functions for numbers and time formats
- Add support for constants and mathematical functions

## Stock quotes

- We no longer use iex.cloud service, it has been discontinued
- We use twelvedata.com
- the api key will be stored in keychain under the name 'twelvedata'
- A sample end point is https://api.twelvedata.com/time_series\?interval\=1min\&apikey\=APIKEY\&symbol\=wday
- The historical data endpoint is documented here: https://twelvedata.com/docs#time-series
- We will use the historical data endpoint each time the '-D' date parameter is used
  - we should request 30 days, daily value only (interval 1 day)
  - all responses should be saved in a sqlite database stored at
    /Users/mike/Library/Mobile\ Documents/com~apple~CloudDocs/quotes/quotes.sqlite3
  - historical quotes are stored in their own table
  - the schema should match and include all details returned by the twelvedata api
  - we should check the database before each historical query and not make a query if the symbol on that date is already
    saved
- The real time endpoint is documented here: https://twelvedata.com/docs#quote
  - all responses should be saved in the same sqlite database stored at
    /Users/mike/Library/Mobile\ Documents/com~apple~CloudDocs/quotes/quotes.sqlite3
  - we should query the real time each time a quote is requested and no historical date is given ('-D')
  - real time quotes are stored in their own table
  - the schema should match and include all details returned by the twelvedata api
  - we should save all real time quotes for a day
  - if the most recent quote indicates the market is closed for the day, we should not query again, but should just return the closing value
  - the parameter 'is_market_open' and the date can be used to determine if the market is open or now (we could be getting pre-market value, meaning closing from yesterday)
