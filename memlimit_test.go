package gojq_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/itchyny/gojq"
)

func withMaxAlloc(t *testing.T, n int64) {
	t.Helper()
	gojq.MaxAlloc = n
	t.Cleanup(func() { gojq.MaxAlloc = 0 })
}

// runQuery runs the query and returns the emitted values, requiring that the
// iteration terminates. Errors are returned as values, like Query.Run does.
func runQuery(t *testing.T, query string, input any) []any {
	t.Helper()
	q, err := gojq.Parse(query)
	if err != nil {
		t.Fatal(err)
	}
	var vs []any
	iter := q.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			return vs
		}
		vs = append(vs, v)
	}
}

func requireAllocError(t *testing.T, vs []any) {
	t.Helper()
	if len(vs) == 0 {
		t.Fatal("expected an error value, got no values")
	}
	err, ok := vs[len(vs)-1].(error)
	if !ok {
		t.Fatalf("expected an error value, got: %#v", vs[len(vs)-1])
	}
	if expected := "allocation limit exceeded"; err.Error() != expected {
		t.Fatalf("expected error %q, got: %v", expected, err)
	}
}

func TestMaxAllocBombs(t *testing.T) {
	// The queries from issue #298 plus other single-shot allocation bombs.
	queries := []string{
		`[range(10000000)]`,
		`[0] | until(length >= 64000000; . + .)`,
		`def f: [f]; f`,
		`[range(10000000)] | length`,
		`"x" * 10000000`,
		`[range(10000000)] | add`,
		`[range(10000000)] | sort`,
		`[range(10000000)] | keys`,
		`[range(10000000)] | reverse`,
		`[range(10000000)] | implode`,
		`[range(10000000)] | tojson`,
		`[range(10000000)] | @csv`,
		`[range(10000000)] | map(.)`,
		`[[range(10000000)]] | flatten`,
		`[range(10000000)] | .[]`,
		`[range(10000000)] | length as $n | setpath([0]; $n)`,
		`range(10000000) | . as $x | [$x]`,
		`reduce range(10000000) as $x ([]; . + [$x])`,
		`[range(10000000)] | map(select(. >= 0))`,
		`[range(10000000)] | group_by(.)`,
		`[range(10000000)] | unique`,
		`[range(10000000)] | sort_by(.)`,
		`[range(10000000)] | min_by(.)`,
		`[range(10000000)] | index(0)`,
		`[range(10000000)] | rindex(0)`,
		`[range(10000000)] | contains([0])`,
		`[range(10000000)] | inside([0])`,
		`[range(10000000)] | .[0:10000000]`,
		`[range(10000000)] | to_entries`,
		`[range(10000000)] | with_entries(.)`,
		`[range(10000000)] | transpose`,
		`[range(10000000)] | splits("")`,
		`("a" * 10000000) | explode`,
		`("a" * 10000000) | split("a")`,
		`("a" * 10000000) | ascii_downcase`,
		`("a" * 10000000) | @base64`,
		`("a" * 10000000) | @uri`,
		`("a" * 10000000) | @html`,
		`("a" * 10000000) | @json`,
		`("a" * 10000000) | ltrimstr("b")`,
		`("a" * 10000000) | test("a")`,
		`("a" * 10000000) | match("a"; "g")`,
		`("a" * 10000000) | capture("(?<x>a)")`,
		`("[1," * 10000000) | fromjson`,
		`"1" + ("0" * 10000000) | tonumber`,
		`[limit(10000000; repeat(1))]`,
		`[range(10000000)] | first(.[])`,
		`[range(10000000)] | last(.[])`,
		`[range(10000000)] | nth(0; .[])`,
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			withMaxAlloc(t, 4<<20)
			requireAllocError(t, runQuery(t, query, nil))
		})
	}
}

func TestMaxAllocDisabledByDefault(t *testing.T) {
	if gojq.MaxAlloc != 0 {
		t.Fatalf("MaxAlloc should default to zero, got: %d", gojq.MaxAlloc)
	}
	vs := runQuery(t, `[range(100000)] | length`, nil)
	if len(vs) != 1 || vs[0] != 100000 {
		t.Fatalf("expected [100000], got: %#v", vs)
	}
}

func TestMaxAllocAllowsSmallQueries(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	queries := []string{
		`.`,
		`.foo`,
		`length`,
		`keys`,
		`[range(100)]`,
		`[range(100)] | add`,
		`[range(100)] | sort`,
		`[range(100)] | map(. + 1)`,
		`[range(100)] | tojson | fromjson | length`,
		`"hello" | ascii_upcase`,
		`"a,b,c" | split(",")`,
		`{"a": [1, 2, 3]} | to_entries`,
		`[range(100)] | @csv`,
		`reduce range(1000000) as $x (0; . + 1)`,
		`[range(100)] | group_by(.) | length`,
		`[range(100)] | min_by(.)`,
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			vs := runQuery(t, query, map[string]any{"foo": 1})
			if len(vs) == 0 {
				t.Fatal("expected a value")
			}
			for _, v := range vs {
				if err, ok := v.(error); ok {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMaxAllocSharedInput(t *testing.T) {
	withMaxAlloc(t, 1<<10)
	// A large input document provided by the caller is not charged, so these
	// queries can pass it around as long as they do not copy it.
	input := make([]any, 10000)
	for i := range input {
		input[i] = strings.Repeat("x", 100)
	}
	for _, query := range []string{`.`, `. | length`, `length`, `. | .`} {
		vs := runQuery(t, query, input)
		if len(vs) != 1 {
			t.Fatalf("expected one value for %q, got: %#v", query, vs)
		}
		if err, ok := vs[0].(error); ok {
			t.Fatalf("unexpected error for %q: %v", query, err)
		}
	}
}

func TestMaxAllocPerRun(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	// The budget is per run; a previous run exhausting it must not affect the
	// next one.
	requireAllocError(t, runQuery(t, `[range(10000000)]`, nil))
	vs := runQuery(t, `[range(100)] | length`, nil)
	if len(vs) != 1 || vs[0] != 100 {
		t.Fatalf("expected [100], got: %#v", vs)
	}
}

func TestMaxAllocConcurrentRuns(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q, err := gojq.Parse(`[range(10000000)]`)
			if err != nil {
				t.Error(err)
				return
			}
			if i%2 == 0 {
				iter := q.Run(nil)
				for {
					v, ok := iter.Next()
					if !ok {
						t.Error("expected an error value")
						return
					}
					if err, ok := v.(error); ok {
						if err.Error() != "allocation limit exceeded" {
							t.Errorf("unexpected error: %v", err)
						}
						return
					}
				}
			}
			q2, err := gojq.Parse(`[range(100)] | length`)
			if err != nil {
				t.Error(err)
				return
			}
			iter := q2.Run(nil)
			for {
				v, ok := iter.Next()
				if !ok {
					return
				}
				if err, ok := v.(error); ok {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if v != 100 {
					t.Errorf("expected 100, got: %v", v)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestMaxAllocCustomFunction(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	q, err := gojq.Parse(`huge`)
	if err != nil {
		t.Fatal(err)
	}
	code, err := gojq.Compile(q, gojq.WithFunction("huge", 0, 0,
		func(any, []any) any {
			return make([]any, 10000000)
		}))
	if err != nil {
		t.Fatal(err)
	}
	iter := code.Run(nil)
	v, ok := iter.Next()
	if !ok {
		t.Fatal("expected an error value")
	}
	if err, ok := v.(error); !ok || err.Error() != "allocation limit exceeded" {
		t.Fatalf("expected allocation limit error, got: %#v", v)
	}
}

func TestMaxAllocCustomIteratorFunction(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	q, err := gojq.Parse(`huge`)
	if err != nil {
		t.Fatal(err)
	}
	code, err := gojq.Compile(q, gojq.WithIterFunction("huge", 0, 0,
		func(any, []any) gojq.Iter {
			return gojq.NewIter(make([]any, 10000000))
		}))
	if err != nil {
		t.Fatal(err)
	}
	iter := code.Run(nil)
	v, ok := iter.Next()
	if !ok {
		t.Fatal("expected an error value")
	}
	if err, ok := v.(error); !ok || err.Error() != "allocation limit exceeded" {
		t.Fatalf("expected allocation limit error, got: %#v", v)
	}
}

func TestMaxAllocNotCatchable(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	// The limit error terminates the run; try/catch and the alternative
	// operator must not swallow it and continue allocating.
	for _, query := range []string{
		`try ([range(10000000)] | length) catch "caught"`,
		`([range(10000000)] | length) // "alternative"`,
		`label $out | try ([range(10000000)] | length) catch break $out`,
	} {
		vs := runQuery(t, query, nil)
		requireAllocError(t, vs)
		for _, v := range vs {
			if s, ok := v.(string); ok {
				t.Fatalf("query %q caught the limit error as %q", query, s)
			}
		}
	}
}

func TestMaxAllocStopsAfterError(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	// The iterator terminates on the limit error instead of emitting a partial
	// result from the interrupted collection.
	vs := runQuery(t, `[range(10000000)] | length`, nil)
	if len(vs) != 1 {
		t.Fatalf("expected exactly one value, got %d: %#v", len(vs), vs)
	}
	requireAllocError(t, vs)
}

func TestMaxAllocMarshal(t *testing.T) {
	withMaxAlloc(t, 1<<10)
	if _, err := gojq.Marshal(strings.Repeat("x", 1<<20)); err == nil ||
		err.Error() != "allocation limit exceeded" {
		t.Fatalf("expected allocation limit error, got: %v", err)
	}
	if s, err := gojq.Marshal("ok"); err != nil || string(s) != `"ok"` {
		t.Fatalf("expected %q, got: %q, %v", `"ok"`, s, err)
	}
}

func TestMaxAllocStreaming(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	// A query that emits many scalars without retaining them must not trip the
	// limit; only live value memory is bounded, not total throughput.
	q, err := gojq.Parse(`range(1000000)`)
	if err != nil {
		t.Fatal(err)
	}
	iter := q.Run(nil)
	n := 0
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			t.Fatalf("unexpected error: %v", err)
		}
		n++
	}
	if n != 1000000 {
		t.Fatalf("expected 1000000 values, got: %d", n)
	}
}

// nestedArray builds a value nested depth levels deep without allocating
// through the interpreter, mimicking a caller-supplied input document.
func nestedArray(depth int) any {
	var v any = []any{}
	for range depth {
		v = []any{v}
	}
	return v
}

func TestMaxAllocDeepValue(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	input := nestedArray(1 << 20)
	// The caller-provided input is not charged, so these queries reach the
	// Go-recursion guards of compare, flatten, encode, and update instead of
	// the value meter. They must error rather than overflow the stack.
	queries := []string{
		`. == .`,
		`. != .`,
		`flatten`,
		`[.] | flatten`,
		`tojson`,
		`tostring`,
		`@json`,
	}
	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			vs := runQuery(t, query, input)
			requireAllocError(t, vs)
		})
	}
}

func TestMaxAllocLongPath(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	path := make([]any, 1<<20)
	for i := range path {
		path[i] = 0
	}
	q, err := gojq.Parse(`setpath($p; 1)`)
	if err != nil {
		t.Fatal(err)
	}
	code, err := gojq.Compile(q, gojq.WithVariables([]string{"$p"}))
	if err != nil {
		t.Fatal(err)
	}
	iter := code.Run(nil, path)
	v, ok := iter.Next()
	if !ok {
		t.Fatal("expected an error value")
	}
	if err, ok := v.(error); !ok || err.Error() != "allocation limit exceeded" {
		t.Fatalf("expected allocation limit error, got: %#v", v)
	}
}

func TestMaxAllocDeepMarshal(t *testing.T) {
	withMaxAlloc(t, 4<<20)
	if _, err := gojq.Marshal(nestedArray(1 << 20)); err == nil ||
		err.Error() != "allocation limit exceeded" {
		t.Fatalf("expected allocation limit error, got: %v", err)
	}
	// Preview must not crash on the same value; it truncates the output.
	if s := gojq.Preview(nestedArray(1 << 20)); s == "" {
		t.Fatal("expected a non-empty preview")
	}
}
