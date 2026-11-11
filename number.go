package gojq

import (
	"encoding/json"
	"math"
	"math/big"
	"strings"
)

// ToInt returns the int value of v, and whether v is a number.
//
// This method is used by built-in operators and functions to convert numbers
// to integers, and accepts number types (int, float64, *big.Int, and
// json.Number); it returns false for other types. This method truncates
// floating-point numbers toward zero, and clamps numbers out of the int range
// (and infinities) to math.MinInt or math.MaxInt. NaN is converted to
// math.MinInt.
func ToInt(v any) (int, bool) {
	switch v := v.(type) {
	case int:
		return v, true
	case float64:
		return floatToInt(v), true
	case *big.Int:
		if v.IsInt64() {
			if i := v.Int64(); math.MinInt <= i && i <= math.MaxInt {
				return int(i), true
			}
		}
		if v.Sign() > 0 {
			return math.MaxInt, true
		}
		return math.MinInt, true
	case json.Number:
		return ToInt(parseNumber(v))
	default:
		return 0, false
	}
}

// ToFloat64 returns the float64 value of v, and whether v is a number.
//
// This method is used by built-in operators and functions to convert numbers
// to floating-point numbers, and accepts number types (int, float64, *big.Int,
// and json.Number); it returns false for other types. Note that large integers
// may lose precision, and numbers out of the float64 range are converted to
// infinities.
func ToFloat64(v any) (float64, bool) {
	switch v := v.(type) {
	case int:
		return float64(v), true
	case float64:
		return v, true
	case *big.Int:
		return bigToFloat(v), true
	case json.Number:
		return ToFloat64(parseNumber(v))
	default:
		return 0.0, false
	}
}

func floatToInt(x float64) int {
	if math.MinInt <= x && x < math.MaxInt {
		return int(x)
	}
	if x > 0 {
		return math.MaxInt
	}
	return math.MinInt
}

func bigToFloat(x *big.Int) float64 {
	if x.IsInt64() {
		return float64(x.Int64())
	}
	// Convert through big.Float (a binary copy) rather than x.String(), whose
	// base-10 conversion allocates large superlinear scratch inside math/big: a
	// 10 MB integer divided by a small number reached ~300 MB, none of it seen
	// by the value meter. Float64 already yields +/-Inf on overflow.
	f, _ := new(big.Float).SetPrec(53).SetInt(x).Float64()
	return f
}

func parseNumber(v json.Number) any {
	s := v.String()
	if len(s) <= 20 {
		if i, err := v.Int64(); err == nil && math.MinInt <= i && i <= math.MaxInt {
			return int(i)
		}
	}
	// Decimal conversion uses superlinear scratch inside math/big. Mirror the
	// encoder's 384-bytes-per-word safety factor in the reverse direction and
	// reject by O(1) digit count before SetString starts work. Counting a sign
	// or decimal syntax byte as a digit is deliberately conservative for
	// absurdly large numbers and avoids a linear pre-scan on the attack path.
	if MaxAlloc > 0 && len(s) > 20 {
		digits := int64(len(s))
		words := (digits + 18) / 19 // at most 19 decimal digits per 64-bit word
		if words > MaxAlloc/384 {
			panic(&allocLimitError{})
		}
	}
	if strings.ContainsAny(s, ".eE") {
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	if bi, ok := new(big.Int).SetString(s, 10); ok {
		return bi
	}
	if strings.HasPrefix(s, "-") {
		return math.Inf(-1)
	}
	return math.Inf(1)
}
