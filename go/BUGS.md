# Bug Report for Go Calculator

## Critical Bugs

### 1. Unreachable Code After Panic (value.go:128-136)
**Location**: `go/value.go`, lines 128-136  
**Severity**: Medium (dead code, but doesn't affect functionality)

The `convertTo` function has unreachable code after a panic statement:

```go
} else {
    panic(fmt.Sprintf("Incomplete for %s -> %s", v.units[dim].name, unit.name))
    // At least one unit uses dynamic conversion
    if unit.factorFunction != nil {
        v.number = unit.factorFunction(v.number, v.units[dim].BaseUnit, unit.BaseUnit)
    } else if v.units[dim].factorFunction != nil {
        v.number = v.units[dim].factorFunction(v.number, v.units[dim].BaseUnit, unit.BaseUnit)
    } else {
        panic(fmt.Sprintf("No conversion method available for %s -> %s", v.units[dim].name, unit.name))
    }
}
```

**Issue**: The code after the first panic will never execute. The dynamic conversion logic should be moved before the panic, or the panic should be removed if dynamic conversion is intended.

**Fix**: Move the dynamic conversion check before the panic, or restructure the logic.

### 2. Potential Nil Pointer Dereference (value.go:123)
**Location**: `go/value.go`, line 123  
**Severity**: High (could cause runtime panic)

```go
factor := div(v.units[dim].factor, units[dim].factor)
if v.units[dim].factor != nil && unit.factor != nil {
```

**Issue**: `div()` is called with potentially nil pointers before checking if they're nil. If either factor is nil, this will cause a panic.

**Fix**: Check for nil before calling `div()`:
```go
if v.units[dim].factor != nil && unit.factor != nil {
    factor := div(v.units[dim].factor, units[dim].factor)
    v.number = mul(v.number, intPow(factor, v.units[dim].power))
    v.units[dim].BaseUnit = unit.BaseUnit
} else {
    // Handle dynamic conversion
}
```

### 3. Value Aliasing in Stack Duplicate (stack.go:114)
**Location**: `go/stack.go`, line 114  
**Severity**: Medium (could cause unexpected mutations)

```go
// TODO: need to copy value, otherwise they're aliased
s.values = append(s.values, s.values[len(s.values)-1])
```

**Issue**: When duplicating a value on the stack, the same `Value` struct is appended, which means both stack entries reference the same underlying `Number` pointer. Mutating one could affect the other.

**Fix**: Create a deep copy of the Value:
```go
func (s *Stack) dup() {
    if len(s.values) < 1 {
        die("Stack is empty for '%s', exiting", "duplicate")
    }
    
    // Create a copy of the value
    last := s.values[len(s.values)-1]
    copy := Value{
        number: newNumber(last.number.String()), // Create new Number
        units:  last.units, // Unit is a value type, so this is a copy
    }
    s.values = append(s.values, copy)
}
```

### 4. Value Mutation in Statistical Operations (stack.go:331, 362)
**Location**: `go/stack.go`, lines 331, 362  
**Severity**: Medium (could cause incorrect results)

```go
// Convert current to minVal's units for comparison
currentConverted := current.apply(minVal.units)
```

**Issue**: The `apply()` method mutates the receiver. When calling `current.apply()`, it modifies `current` in place, which could affect the original stack value if `current` is a reference to the actual stack element.

**Fix**: Since `Value` is a value type (not a pointer), this might be okay, but it's safer to work on a copy:
```go
currentCopy := current
currentConverted := currentCopy.apply(minVal.units)
```

## Logic Bugs

### 5. Comma Grouping Logic Error (number.go:506)
**Location**: `go/number.go`, line 506  
**Severity**: Medium (incorrect output formatting)

```go
if i > 0 && (len(integerPart)-i)%3 == 0 {
    result.WriteString(separator)
}
```

**Issue**: The grouping logic may not work correctly for all cases. For example:
- "1234" should become "1,234" but the current logic might place the comma incorrectly
- The condition `(len(integerPart)-i)%3 == 0` checks if the remaining length is divisible by 3, which may not align commas correctly

**Fix**: Use a more standard approach:
```go
for i, digit := range integerPart {
    if i > 0 && (len(integerPart)-i)%3 == 0 {
        result.WriteString(separator)
    }
    result.WriteRune(digit)
}
```

Actually, the logic should be: insert comma when `(len(integerPart) - i) % 3 == 0 && i > 0`. But this might still be off. Better approach:
```go
// Group from right to left
for i := len(integerPart) - 1; i >= 0; i-- {
    if i < len(integerPart)-1 && (len(integerPart)-1-i)%3 == 0 {
        result.WriteString(separator)
    }
    result.WriteRune(rune(integerPart[i]))
}
// Then reverse the result
```

Or simpler:
```go
var result strings.Builder
for i, digit := range integerPart {
    if i > 0 && (len(integerPart)-i)%3 == 0 {
        result.WriteString(separator)
    }
    result.WriteRune(digit)
}
```

Wait, let me trace through "1234":
- i=0, len=4, (4-0)%3=1, no comma, write '1'
- i=1, len=4, (4-1)%3=0, comma, write '2' → "1,2"
- i=2, len=4, (4-2)%3=1, no comma, write '3' → "1,23"
- i=3, len=4, (4-3)%3=1, no comma, write '4' → "1,234"

Actually that works! But let's check "12345":
- i=0: (5-0)%3=2, no comma, '1'
- i=1: (5-1)%3=1, no comma, '2' → "12"
- i=2: (5-2)%3=0, comma, '3' → "12,3"
- i=3: (5-3)%3=2, no comma, '4' → "12,34"
- i=4: (5-4)%3=1, no comma, '5' → "12,345"

That's correct! So the logic might actually be fine. But let me verify "1234567":
- Should be "1,234,567"
- i=0: (7-0)%3=1, no comma, '1'
- i=1: (7-1)%3=0, comma, '2' → "1,2"
- i=2: (7-2)%3=2, no comma, '3' → "1,23"
- i=3: (7-3)%3=1, no comma, '4' → "1,234"
- i=4: (7-4)%3=0, comma, '5' → "1,234,5"
- i=5: (7-5)%3=2, no comma, '6' → "1,234,56"
- i=6: (7-6)%3=1, no comma, '7' → "1,234,567"

Perfect! The logic is actually correct. This is not a bug.

### 6. Truncate Precision Loss (number.go:236-246)
**Location**: `go/number.go`, lines 236-246  
**Severity**: Low (precision loss in intermediate step)

```go
func truncate(x, y *Number) *Number {
    result := new(Number)
    result.Set(x.String())  // Converts to string and back, losing precision
    
    // Extract integer part
    intPart := new(big.Int)
    intPart.Quo(result.Rat.Num(), result.Rat.Denom())
    result.Rat.SetInt(intPart)
    
    return result
}
```

**Issue**: Converting to string and back (`x.String()`) may lose precision, especially if the number has more precision than the display precision setting.

**Fix**: Work directly with the rational number:
```go
func truncate(x, y *Number) *Number {
    result := new(Number)
    result.Rat = new(big.Rat)
    
    // Extract integer part directly
    intPart := new(big.Int)
    intPart.Quo(x.Rat.Num(), x.Rat.Denom())
    result.Rat.SetInt(intPart)
    
    return result
}
```

## Potential Issues

### 7. Temperature Unit Name Mismatch
**Location**: `go/unit.go`, temperature conversion functions  
**Severity**: Low (might work but inconsistent)

The unit definitions allow both "F"/"C" and "°F"/"°C" as input, but the `BaseUnit.name` is always set to "°F" or "°C". The conversion function checks for "°F" and "°C" specifically. This should work because the BaseUnit.name is always set to the degree symbol version, but it's worth verifying that all code paths use the BaseUnit.name consistently.

### 8. Missing Error Handling in Currency Conversion
**Location**: `go/currency.go`, `go/unit.go`  
**Severity**: Low (errors are panicked, which is acceptable for CLI tool)

Currency conversion errors are handled via panic, which is acceptable for a CLI tool, but could be improved with better error messages.

## Summary

**Critical fixes needed**:
1. Fix nil pointer check in `convertTo()` (line 123)
2. Remove unreachable code or restructure logic (line 128-136)
3. Fix value aliasing in `dup()` (line 114)

**Recommended fixes**:
4. Fix truncate to avoid precision loss (line 238)
5. Verify value mutation in statistical operations (lines 331, 362)

