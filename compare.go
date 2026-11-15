package gojq

import (
	"cmp"
	"encoding/json"
	"math"
	"math/big"
	"strings"
)

// Compare l and r, and returns jq-flavored comparison value.
// The result will be 0 if l == r, -1 if l < r, and +1 if l > r.
// This comparison is used by built-in operators and functions.
func Compare(l, r any) int {
	return binopTypeSwitch(l, r,
		cmp.Compare,
		func(l, r float64) int {
			switch {
			case lt(l, r):
				return -1
			case l == r:
				return 0
			default:
				return 1
			}
		},
		(*big.Int).Cmp,
		strings.Compare,
		func(l, r []any) int {
			for i := range min(len(l), len(r)) {
				if cmp := Compare(l[i], r[i]); cmp != 0 {
					return cmp
				}
			}
			return cmp.Compare(len(l), len(r))
		},
		func(l, r map[string]any) int {
			lk, rk := funcKeys(l), funcKeys(r)
			if cmp := Compare(lk, rk); cmp != 0 {
				return cmp
			}
			for _, k := range lk.([]any) {
				if cmp := Compare(l[k.(string)], r[k.(string)]); cmp != 0 {
					return cmp
				}
			}
			return 0
		},
		func(l, r any) int {
			return cmp.Compare(typeIndex(l), typeIndex(r))
		},
	)
}

func lt(l, r float64) bool {
	return l < r || math.IsNaN(l)
}

func typeIndex(v any) int {
	switch v := v.(type) {
	default:
		return 0
	case bool:
		if !v {
			return 1
		}
		return 2
	case int, float64, *big.Int, json.Number:
		return 3
	case string:
		return 4
	case []any:
		return 5
	case map[string]any:
		return 6
	}
}

// cloneConst clones the constant value embedded in the bytecode so that
// mutable objects (arrays and objects) are never shared between multiple
// executions of a compiled query. Constant values are created at compile
// time, but their copies are given to the virtual machine at execution time,
// otherwise the values returned by Run could be mutated by the user and
// affect the following executions (issue #235).
func cloneConst(v any) any {
	switch v := v.(type) {
	case []any:
		w := make([]any, len(v))
		for i, x := range v {
			w[i] = cloneConst(x)
		}
		return w
	case map[string]any:
		w := make(map[string]any, len(v))
		for k, x := range v {
			w[k] = cloneConst(x)
		}
		return w
	default:
		return v
	}
}
