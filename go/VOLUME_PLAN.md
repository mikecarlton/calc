# Plan: Volume ↔ Cubic Length Conversions

## Goal
Enable conversions between volume units (liters, gallons, etc.) and cubic length units (m³, cm³, in³, ft³, etc.), recognizing that:
- 1 liter = 1000 cm³ = 0.001 m³ (by definition)
- Volume dimension should be compatible with Length³ dimension

## Current State

### Unit System Structure
- Units are represented as arrays of `UnitPower`, one per dimension
- `Volume` is a separate dimension from `Length`
- `compatible()` function checks if units have same power in ALL dimensions
- Volume units (l, ml, gal, etc.) have `Volume` dimension with power=1
- Length units (m, cm, in, ft, etc.) have `Length` dimension with power=1

### Current Limitations
- Volume and Length are separate dimensions, so they're never compatible
- No way to convert between 1 liter and 1000 cm³
- **Note**: The unit parser already handles cubic length units! `cm^3` or `cm³` is automatically parsed as Length dimension with power=3, so no custom unit definitions are needed.

## Proposed Solution

### Phase 1: Update Compatibility Check
**File**: `go/unit.go`

Modify the `compatible()` function to recognize that:
- Volume (power=1) is equivalent to Length³ (power=3)
- This is a special case that needs explicit handling

**Implementation**:
```go
func (u *Unit) compatible(other Unit) bool {
    // Check standard compatibility (same power in all dimensions)
    standardCompatible := true
    for i := range u {
        if u[i].power != other[i].power {
            standardCompatible = false
            break
        }
    }
    if standardCompatible {
        return true
    }
    
    // Special case: Volume (power=1) is compatible with Length³ (power=3)
    // Check if one has Volume=1 and other has Length=3 (and all other dimensions match)
    uHasVolume := u[Volume].power == 1 && u[Length].power == 0
    otherHasLength3 := other[Volume].power == 0 && other[Length].power == 3
    
    otherHasVolume := other[Volume].power == 1 && other[Length].power == 0
    uHasLength3 := u[Volume].power == 0 && u[Length].power == 3
    
    if (uHasVolume && otherHasLength3) || (otherHasVolume && uHasLength3) {
        // Check all other dimensions match
        for i := range u {
            if i == int(Volume) || i == int(Length) {
                continue // Skip Volume and Length, already checked
            }
            if u[i].power != other[i].power {
                return false
            }
        }
        return true
    }
    
    return false
}
```

### Phase 2: Add Volume ↔ Length³ Conversion Function
**File**: `go/unit.go`

Create a conversion function similar to `temperatureConvert` and `currencyConvert`:

```go
// volumeToLength3 converts volume units to cubic length units
// Base conversion: 1 liter = 1000 cm³ = 0.001 m³
func volumeToLength3(amount *Number, from, to BaseUnit) *Number {
    // Determine conversion path:
    // 1. If converting from Volume to Length³:
    //    - Convert volume unit to liters (base volume unit)
    //    - Convert liters to cm³ (1 l = 1000 cm³)
    //    - Convert cm³ to target length³ unit
    
    // 2. If converting from Length³ to Volume:
    //    - Convert length³ unit to cm³ (base cubic length)
    //    - Convert cm³ to liters (1000 cm³ = 1 l)
    //    - Convert liters to target volume unit
    
    // For now, we'll use cm³ as the intermediate unit
    // 1 liter = 1000 cm³
    
    if from.dimension == Volume && to.dimension == Length {
        // Volume → Length³
        // First, convert volume to liters (if not already)
        // Then convert liters to cm³
        // Then convert cm³ to target length³
        
        // Convert from volume unit to liters
        liters := amount
        if from.factor != nil {
            // Convert to liters: amount / from.factor (since factor is relative to liters)
            liters = div(amount, from.factor)
        }
        
        // Convert liters to cm³: 1 l = 1000 cm³
        cm3 := mul(liters, newNumber(1000))
        
        // Convert cm³ to target length³ unit
        // The target unit's factor is relative to meters (base length unit)
        // We need to convert: cm³ → m³ → target³
        
        // First convert cm³ to m³: 1 cm = 0.01 m, so 1 cm³ = (0.01)³ m³ = 0.000001 m³
        // Or: 1 cm³ = 1 / 1,000,000 m³
        m3 := div(cm3, newNumber(1_000_000))
        
        // Now convert m³ to target length³
        // to.factor converts from meters to target length unit
        // For cubic: multiply by (to.factor)³
        if to.factor != nil {
            cmToTarget := to.factor
            m3ToTarget3 := mul(mul(cmToTarget, cmToTarget), cmToTarget) // (factor)³
            return mul(m3, m3ToTarget3)
        }
        
        // If target is m³, we're done
        if to.name == "m" {
            return m3
        }
        
        panic(fmt.Sprintf("Unsupported volume to length³ conversion: %s -> %s", from.name, to.name))
    }
    
    if from.dimension == Length && to.dimension == Volume {
        // Length³ → Volume
        // First, convert length³ to cm³
        // Then convert cm³ to liters
        // Then convert liters to target volume unit
        
        // Convert from length³ to cm³
        // from.factor converts from meters to source length unit
        // First convert source³ to m³, then m³ to cm³
        
        // Convert source³ to m³: divide by (from.factor)³
        m3 := amount
        if from.factor != nil {
            sourceToM := from.factor
            source3ToM3 := div(div(newNumber(1), sourceToM), mul(sourceToM, sourceToM)) // 1/(factor)³
            m3 = mul(amount, source3ToM3)
        }
        
        // Convert m³ to cm³: 1 m³ = 1,000,000 cm³
        cm3 := mul(m3, newNumber(1_000_000))
        
        // Convert cm³ to liters: 1000 cm³ = 1 l
        liters := div(cm3, newNumber(1000))
        
        // Convert liters to target volume unit
        if to.factor != nil {
            // to.factor is conversion from liters to target volume
            return mul(liters, to.factor)
        }
        
        panic(fmt.Sprintf("Unsupported length³ to volume conversion: %s -> %s", from.name, to.name))
    }
    
    panic(fmt.Sprintf("Invalid volume/length³ conversion: %s -> %s", from.name, to.name))
}
```

### Phase 3: Update Conversion Logic
**File**: `go/value.go`

Modify `convertTo()` and `apply()` to handle Volume ↔ Length³ conversions:

1. **In `convertTo()`**: When converting between Volume and Length³, use the conversion function
2. **In `apply()`**: When applying a cubic length unit to a volume value (or vice versa), convert appropriately

**Key changes**:
- Check if conversion is Volume ↔ Length³
- If so, use `volumeToLength3` conversion function
- Handle the dimension change (Volume power=1 ↔ Length power=3)

### Phase 4: Testing

Test cases to implement:
1. `1 l cm^3` → should convert 1 liter to 1000 cm³
2. `1000 cm^3 l` → should convert 1000 cm³ to 1 liter
3. `1 l m³` → should convert 1 liter to 0.001 m³ (using superscript)
4. `1 gal cm^3` → should convert 1 gallon to appropriate cm³
5. `1 ft^3 l` → should convert 1 cubic foot to liters
6. `1 l + 500 cm^3` → should add 1 liter + 500 cm³ (compatible units)
7. `1 m³ gal` → should convert 1 m³ to gallons
8. `1 in^3 ml` → should convert 1 cubic inch to milliliters

## Implementation Order

1. **Phase 1**: Update `compatible()` function - enables the system to recognize Volume and Length³ as compatible
2. **Phase 2**: Implement `volumeToLength3()` conversion function - core conversion logic
3. **Phase 3**: Update conversion logic in `value.go` - wires everything together
4. **Phase 4**: Add tests - verify correctness

**Note**: No need to define cubic length units or update parsing - the existing system already handles `cm^3`, `m³`, etc. as Length dimension with power=3.

## Edge Cases to Handle

1. **SI prefixes on cubic length**: `km³`, `mm³` should work
2. **Mixed operations**: Adding volume and cubic length should work
3. **Display**: Cubic length units should display with superscript (m³) or ^3 notation
4. **Precision**: Ensure exact conversions (1 l = 1000 cm³ exactly)
5. **Negative powers**: Handle cases like m³/s (volume flow rate)

## Notes

- The relationship 1 l = 1000 cm³ is by definition, so this should be exact
- 1 m³ = 1000 l (since 1 m = 100 cm, so 1 m³ = 1,000,000 cm³ = 1000 l)
- **No custom unit definitions needed**: The parser already handles `cm^3`, `m³`, `in^3`, `ft³`, etc. automatically
- Length unit factors are relative to meters (base unit), so:
  - cm factor = 0.01 (1 cm = 0.01 m)
  - in factor = 0.0254 (1 in = 0.0254 m)
  - For cubic: cm³ factor relative to m³ = (0.01)³ = 0.000001
- Conversion path: Volume → liters → cm³ → m³ → target length³

