# Calculator Implementations: Ruby vs Go

This document describes the two implementations of the CLI calculator: the original Ruby version (`calc.rb`) and the newer Go version (`go/calc.go`).

## Overview

Both implementations provide a reverse Polish notation (RPN) stack-based calculator with:
- Arbitrary precision arithmetic
- Unit conversion and dimensional analysis
- Multiple number formats (decimal, hex, binary, octal)
- Stack manipulation operations
- Statistical operations
- Currency conversion (Ruby only)
- Stock quotes (Ruby only)

## Architecture Comparison

### Ruby Version (`calc.rb`)

**Language**: Ruby 2.7+
**Structure**: Single monolithic file (~1250 lines)
**Key Classes**:
- `Stack`: Main calculator engine with input parsing
- `Denominated`: Value with units (numerator/denominator)
- `Unit`: Unit definitions with conversion factors
- `Constant`: Mathematical constants (π, e, ∞, C)
- `BigDecimal`: Used for arbitrary precision arithmetic

**Design Philosophy**:
- Object-oriented with extensive monkey-patching
- Uses Ruby's dynamic typing and metaprogramming
- String-based pattern matching for input parsing
- Forwardable module for delegation

### Go Version (`go/calc.go`)

**Language**: Go
**Structure**: Modular package with multiple files:
- `calc.go`: Main entry point and input processing
- `stack.go`: Stack operations and display
- `value.go`: Value type with units and operations
- `unit.go`: Unit system and conversions
- `number.go`: Arbitrary precision number representation
- `options.go`: Command-line option parsing
- `currency.go`: Currency conversion (if present)

**Design Philosophy**:
- Strongly typed with explicit error handling
- Uses `math/big.Rat` for arbitrary precision
- Struct-based with methods
- Compile-time type safety

## Number Representation

### Ruby Version

- **Integers**: Ruby `Integer` (arbitrary precision)
- **Rationals**: Ruby `Rational` class
- **Floats**: `BigDecimal` for precision control
- **Parsing**: String extensions with `.int`, `.float`, `.ipv4` methods
- **Formatting**: Custom `to_s` methods with precision control

### Go Version

- **All Numbers**: `Number` type embedding `*big.Rat` (rational numbers)
- **Unified Representation**: All numbers stored as rationals (numerator/denominator)
- **Parsing**: `NewFromString()` with regex patterns for decimal, hex, binary, octal
- **Formatting**: `String()` method with configurable precision

**Advantage**: Go version uses a unified rational number system, avoiding floating-point precision issues entirely.

## Units System

### Ruby Version

**Structure**:
- Units stored as `numerator` and `denominator` on `Denominated` objects
- Each `Unit` has:
  - `name`, `desc`, `dimension`, `si` flag
  - `factor` (conversion factor) or `ifactor` (inverse)
  - Can be a `Proc` for dynamic conversion (currency, temperature)

**Unit Application**:
- If value has no units: apply unit
- If value has compatible units: convert to new unit
- Units checked for compatibility in arithmetic operations

**Supported Dimensions**:
- Time (s, mn, hr)
- Length (m, mm, cm, km, in, ft, yd, mi)
- Volume (l, ml, cl, dl, foz, cp, pt, qt, gal)
- Mass (g, kg, oz, lb)
- Temperature (c, f) with special offset handling
- Currency (usd, eur, gbp, yen, btc, $, €, £, ¥)

### Go Version

**Structure**:
- `Unit` is an array of `UnitPower` (one per dimension)
- `BaseUnit` contains:
  - `name`, `description`, `dimension`
  - `factor` (static) or `factorFunction` (dynamic)
  - `delta` flag for temperature deltas

**Unit Application**:
- Similar logic: apply if empty, convert if compatible
- More structured with explicit dimension checking
- Supports SI prefixes (k, M, G, m, μ, etc.) for base units
- Derived units (J, N, V, W, Ω) as composite units

**Supported Dimensions**:
- Mass, Length, Time, Volume, Temperature, Currency, Current
- Similar unit set to Ruby version
- Better support for derived units (joules, newtons, volts, watts, ohms)

**Advantage**: Go version has more structured unit system with explicit dimension tracking and better derived unit support.

## Operations

### Arithmetic Operations

**Both Support**:
- `+`, `-`, `*`, `/`, `%` (modulo)
- `**` or `pow` (exponentiation)
- Aliases: `.` and `•` for `*`, `÷` for `/`

**Ruby-Specific**:
- `lcm`, `gcd` (integer only)

**Go-Specific**:
- More explicit type checking for dimensionless operations

### Unary Operations

**Both Support**:
- `chs` (change sign)
- `t` or `truncate` (truncate to integer)
- `round` (round to integer)
- `[` (floor), `]` (ceiling)
- `r` (reciprocal)
- `sqrt` or `√` (square root)
- `log`, `log2`, `log10`
- `sin`, `cos`, `tan`
- `rand` (random number)
- `~` (bitwise NOT, integers only)

**Ruby-Specific**:
- `!` (factorial, integer only)
- `i` (invert units)

**Go-Specific**:
- `num` (remove units)
- `mask` (IPv4 mask)

### Bitwise Operations

**Both Support** (integers only):
- `&` (AND), `|` (OR), `^` (XOR)
- `<<` (left shift), `>>` (right shift)
- `~` (NOT)

### Stack Reduction

**Both Support**:
- `@op` prefix reduces entire stack with operation
- Example: `@+` sums all values on stack

## Stack Operations

### Common Operations

**Both Support**:
- `x` (exchange top 2 elements)
- `d` or `dup` (duplicate top element)
- `p` or `pop` (pop top element)
- `clear` (clear entire stack)

### Stack Statistics

**Both Support**:
- `mean` (average)
- `max` (maximum)
- `min` (minimum) - Ruby uses `min`, Go uses `mini`
- `size` (stack size)

**Modifier**: Append `!` to replace stack with result (Ruby and Go)

## Input Parsing

### Ruby Version

**Pattern Matching**:
- Uses `StringScanner` for sequential pattern matching
- Ordered list of regex patterns with actions
- Patterns include: numbers, units, constants, operations, registers

**Special Inputs**:
- Time: `H:MM:SS` or `M:SS` format
- ASCII: `'string'` converts to integer
- Stock quotes: `@TICKER` (requires IEX API key)
- IPv4: `a.b.c.d` format
- Rationals: `num/den` format

**Registers**:
- `>NAME`: Store value in register
- `<NAME`: Retrieve value from register
- `>:NAME`: Clear register

### Go Version

**Parsing Strategy**:
- Sequential parsing with multiple parse functions
- `parseNumber()`, `parseBase60()`, `parseIPv4()`, `parseUnits()`
- Operator lookup tables

**Special Inputs**:
- Base-60: `H:MM:SS` or `M:SS` (parsed as regular number)
- IPv4: `a.b.c.d` format
- Constants: `pi`, `e`, `c`, `G`, `acre`, `hectare`

**Missing Features** (compared to Ruby):
- No ASCII string conversion
- No stock quotes
- No registers/variables
- No rational input format (`num/den`)

## Command-Line Options

### Ruby Version

```
-t          Trace operations
-b          Show binary representation
-x          Show hex representation
-i          Show IPv4 representation
-a          Show ASCII representation
-c N        Extract column N from stdin (negative from end)
-d REGEX    Delimiter for column extraction (default: whitespace)
-p N        Set precision (default: 2)
-g          Group numbers with ','
-f          Show prime factorization
-s          Show statistics
-q          Quiet (don't show stack)
-o          One-line output
-D DATE     Date for currency rates
-v          Verbose (repeat for more)
-u          Show units list
-h          Show help
```

### Go Version

```
-t          Trace operations
-b          Show binary representation
-o          Show octal representation
-x          Show hex representation
-X          Show hex (including floats)
-i          Show IPv4 representation
-r          Show rational representation (n/d)
-g          Group numbers with ',' or '_'
-s          Show statistics
-O          One-line output
-S          Disable superscript powers
-c N        Extract column N from stdin
-p N        Set precision (default: 4)
-D DATE     Date for currency rates
--debug     Show debug information
--base      Display units as base units only
-h          Show help
```

**Key Differences**:
- Go has octal (`-o`) and hex float (`-X`) support
- Go has rational display (`-r`) and superscript control (`-S`)
- Go has `--base` option for unit display
- Ruby has ASCII (`-a`), factorization (`-f`), and column delimiter (`-d`)
- Default precision: Ruby 2, Go 4

## Display Formatting

### Ruby Version

- Multi-column output with aligned decimals
- Units shown in separate column
- Time units formatted as `H:MM:SS` or `M:SS`
- Supports multiple formats simultaneously (decimal, hex, binary, IPv4, ASCII, factor)

### Go Version

- Similar multi-column output
- Better decimal point alignment
- Time units formatted similarly
- Supports decimal, hex, binary, octal, IPv4, rational formats
- Units with superscript powers by default (e.g., `m²`, `s⁻¹`)
- Can disable superscripts with `-S` flag

## External Dependencies

### Ruby Version

**Required**:
- Standard library: `bigdecimal`, `forwardable`, `fileutils`, `date`, `json`, `net/http`, `open3`, `strscan`
- External APIs:
  - IEX Cloud API (for stock quotes)
  - OpenExchangeRates API (for currency conversion)
- macOS Keychain integration for API keys

### Go Version

**Required**:
- Standard library: `math/big`, `bufio`, `fmt`, `os`, `strings`, `regexp`, `strconv`
- External APIs:
  - Currency conversion API (if `currency.go` exists)
- No keychain integration (uses environment variables)

## Error Handling

### Ruby Version

- Uses `die()` function for fatal errors
- Exception-based with rescue blocks
- Color-coded error messages (red)
- Stack traces in trace mode

### Go Version

- Uses `die()` function for fatal errors
- Panic/recover for unexpected errors
- Explicit error returns from functions
- More structured error messages

## Performance Considerations

### Ruby Version

- Interpreted language - slower startup
- Dynamic typing overhead
- String operations are efficient
- Good for interactive use

### Go Version

- Compiled binary - fast startup
- Static typing - no runtime type checks
- Efficient memory management
- Better for scripting/automation

## Feature Parity

### Features in Ruby but not Go

1. **Stock Quotes**: `@TICKER` syntax with IEX API
2. **ASCII Conversion**: `'string'` to integer
3. **Registers/Variables**: `>NAME`, `<NAME`, `>:NAME`
4. **Prime Factorization**: `-f` flag
5. **Column Delimiter**: `-d REGEX` option
6. **Rational Input**: `num/den` format
7. **GCD/LCM**: Integer operations
8. **Factorial**: `!` operator

### Features in Go but not Ruby

1. **Octal Support**: `-o` flag and `0o` prefix
2. **Hex Float Support**: `-X` flag and `0xp` format
3. **Rational Display**: `-r` flag shows n/d format
4. **Superscript Control**: `-S` flag
5. **Base Units Display**: `--base` flag
6. **Derived Units**: Better support (J, N, V, W, Ω)
7. **SI Prefixes**: Automatic generation (km, mm, etc.)
8. **IPv4 Mask**: `mask` operation
9. **Constants**: `G`, `acre`, `hectare` in addition to `pi`, `e`, `c`

## Code Organization

### Ruby Version

- **Lines**: ~1250
- **Structure**: Single file, class-based
- **Maintainability**: Monolithic but well-organized with clear class boundaries
- **Extensibility**: Easy to add via monkey-patching (Ruby style)

### Go Version

- **Lines**: ~2000+ across multiple files
- **Structure**: Modular package design
- **Maintainability**: Better separation of concerns
- **Extensibility**: Type-safe, requires more boilerplate

## Use Cases

### Choose Ruby Version If:

- You need stock quotes or currency conversion
- You want registers/variables
- You need ASCII string conversion
- You prefer dynamic, flexible syntax
- You're on macOS and want keychain integration

### Choose Go Version If:

- You need better performance
- You want octal or hex float support
- You prefer compiled binaries
- You need better unit system (SI prefixes, derived units)
- You want type safety and explicit error handling
- You're building automation scripts

## Conclusion

Both implementations provide a powerful RPN calculator with unit support. The Ruby version is more feature-rich with external integrations (stocks, currency) and dynamic features (registers, ASCII). The Go version focuses on mathematical precision, better unit system, and performance, making it better suited for computational tasks and automation.

The Go version appears to be a modernization effort, trading some convenience features for better architecture, type safety, and performance.

