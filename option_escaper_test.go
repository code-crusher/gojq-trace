package gojq_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/itchyny/gojq"
)

func ExampleWithEscaper() {
	query, err := gojq.Parse(`"abcde" | split("") | @goprint "goprint: \(.)"`)
	if err != nil {
		log.Fatalln(err)
	}
	code, err := gojq.Compile(
		query,
		gojq.WithEscaper("goprint", func(x any) string {
			return fmt.Sprint(x)
		}),
	)
	if err != nil {
		log.Fatalln(err)
	}
	iter := code.Run(nil)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalln(err)
		}
		fmt.Printf("%v\n", v)
	}

	// Output:
	// goprint: [a b c d e]
}

func ExampleWithEscaper_override() {
	// The escaper can also override the built-in format strings.
	query, err := gojq.Parse(`. | @text, @upper`)
	if err != nil {
		log.Fatalln(err)
	}
	code, err := gojq.Compile(
		query,
		gojq.WithEscaper("text", func(x any) string {
			return "text: " + fmt.Sprint(x)
		}),
		gojq.WithEscaper("upper", func(x any) string {
			return fmt.Sprintf("[%v]", x)
		}),
	)
	if err != nil {
		log.Fatalln(err)
	}
	iter := code.Run("abc")
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalln(err)
		}
		fmt.Printf("%v\n", v)
	}

	// Output:
	// text: abc
	// [abc]
}

func ExampleWithEscaper_formatFunction() {
	// The escaper is also used by the `format` function.
	query, err := gojq.Parse(`format("goprint")`)
	if err != nil {
		log.Fatalln(err)
	}
	code, err := gojq.Compile(
		query,
		gojq.WithEscaper("goprint", func(x any) string {
			return fmt.Sprint(x)
		}),
	)
	if err != nil {
		log.Fatalln(err)
	}
	iter := code.Run([]any{"a", "b"})
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalln(err)
		}
		fmt.Printf("%v\n", v)
	}

	// Output:
	// [a b]
}

func TestWithEscaperError(t *testing.T) {
	query, err := gojq.Parse("@f")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "a-b", "a b", "a.b", "a/b"} {
		func(name string) {
			expected := fmt.Sprintf("invalid name for escaper: %q", name)
			defer func() {
				if got := recover(); got != expected {
					t.Errorf("expected: %v, got: %v", expected, got)
				}
			}()
			if _, err := gojq.Compile(query, gojq.WithEscaper(name, nil)); err != nil {
				t.Fatal(err)
			}
			t.Errorf("expected a panic for %q", name)
		}(name)
	}
}
