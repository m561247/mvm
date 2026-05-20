package main

import "fmt"

// A func type whose result is a parenthesized multi-value list, e.g. (int, error),
// must parse even when the func type is the element type of a slice composite
// literal: []func(int) (int, error){...}. The trailing composite-literal brace
// previously made parseFuncParams miss the parenthesized return list, so the
// parser hit ParenBlock with no case ("not implemented: ... ParenBlock").

func main() {
	fns := []func(int) (int, error){
		func(x int) (int, error) { return x + 1, nil },
	}
	r, err := fns[0](41)
	fmt.Println(r, err)
}

// Output:
// 42 <nil>
