package gojq_test

import (
	"fmt"
	"log"

	"github.com/itchyny/gojq"
)

func ExampleToSeq() {
	query, err := gojq.Parse(".foo | ..")
	if err != nil {
		log.Fatalln(err)
	}
	for v := range gojq.ToSeq(query.Run(map[string]any{"foo": []any{1, 2, 3}})) {
		fmt.Printf("%#v\n", v)
	}

	// Output:
	// []interface {}{1, 2, 3}
	// 1
	// 2
	// 3
}

func ExampleToSeq2() {
	query, err := gojq.Parse(".foo | ..")
	if err != nil {
		log.Fatalln(err)
	}
	for v, err := range gojq.ToSeq2(query.Run(map[string]any{"foo": []any{1, 2, 3}})) {
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%#v\n", v)
	}

	// Output:
	// []interface {}{1, 2, 3}
	// 1
	// 2
	// 3
}

func ExampleToSeq2_halt() {
	query, err := gojq.Parse("1, halt, 2")
	if err != nil {
		log.Fatalln(err)
	}
	for v, err := range gojq.ToSeq2(query.Run(nil)) {
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%#v\n", v)
	}

	// Output:
	// 1
}

func ExampleToSeq2_haltError() {
	query, err := gojq.Parse("1, halt_error, 2")
	if err != nil {
		log.Fatalln(err)
	}
	for v, err := range gojq.ToSeq2(query.Run(nil)) {
		if err, ok := err.(*gojq.HaltError); ok {
			if err.Value() == nil {
				break
			}
			fmt.Printf("halted with value: %v\n", err.Value())
			continue
		}
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%#v\n", v)
	}

	// Output:
	// 1
}

func ExampleToSeq2_error() {
	query, err := gojq.Parse("1, error(\"foo\"), 2")
	if err != nil {
		log.Fatalln(err)
	}
	for v, err := range gojq.ToSeq2(query.Run(nil)) {
		if err != nil {
			fmt.Printf("error: %s\n", err)
			continue
		}
		fmt.Printf("%#v\n", v)
	}

	// Output:
	// 1
	// error: error: foo
	// 2
}

func ExampleToSeq2_break() {
	query, err := gojq.Parse("range(5)")
	if err != nil {
		log.Fatalln(err)
	}
	var vs []int
	for v, err := range gojq.ToSeq2(query.Run(nil)) {
		if err != nil {
			log.Fatalln(err)
		}
		if v.(int) >= 2 {
			break
		}
		vs = append(vs, v.(int))
	}
	fmt.Println(vs)

	// Output:
	// [0 1]
}
