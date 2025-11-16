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
  - `currency.go` - Currency conversion using OpenExchangeRates
  - `quote.go` - Stock quotes using TwelveData API

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

### Stock Quotes

The Go version uses TwelveData API for stock quotes. To use stock quotes:

1. **Get a free API key** from [TwelveData](https://twelvedata.com/)
   - Free tier includes 800 API calls per day
   - Supports real-time and historical stock data

2. **Set up the API key:**
   ```bash
   # Via environment variable
   export twelvedata=YOUR_API_KEY

   # Or on macOS using Keychain
   security add-generic-password -s twelvedata -a api_key -U -w YOUR_API_KEY
   ```

3. **Usage:**
   ```bash
   # Get stock quote
   ./calc @aapl

   # Perform calculations with stock prices
   ./calc @wday @msft +

   # With debug output for detailed quote info
   ./calc -d @aapl
   ```

The stock quote feature returns the latest close price with the appropriate currency (usually USD).
