package main

import "strings"

func main() {
	// Slicing an untyped string const: the operand has no concrete Type
	// at compile time (only a Value). Used to nil-deref in the lang.Slice
	// arity sniff.
	const str = "0123456789"
	println(str[1:])
	println(str[:3])
	println(str[2:5])

	// Slicing expression results (call, concat).
	println(strings.Repeat("a", 4)[:2])
	println(("x" + strings.Repeat("y", 3))[1:])

	// 3-index slice arity must still be detected.
	a := []int{1, 2, 3, 4, 5}
	b := a[1:3:4]
	println(len(b), cap(b))
}

// Output:
// 123456789
// 012
// 234
// aa
// yyy
// 2 3
