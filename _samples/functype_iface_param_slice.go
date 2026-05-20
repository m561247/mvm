package main

import (
	"fmt"
	"io"
)

// BUG (surfaced from bufio_test.go TestReaderWriteTo, line ~1171): a slice
// composite literal whose element is an INLINE func type with a qualified
// interface param/return -- []func(io.Reader) io.Reader{...} -- builds a zero
// (invalid) slice value, so len(rs) panics with
// "reflect: call of reflect.Value.Len on zero Value".
//
// Works for comparison:
//   - inline func type over builtins:  []func(int) int{...}
//   - named func type:                 type F func(io.Reader) io.Reader; []F{...}
//
// So the defect is specific to constructing the slice when the element is an
// inline (unnamed) func type whose signature references a qualified interface
// type. Distinct from the parenthesized-return parse bug fixed alongside this
// (see functype_paren_return.go).

func main() {
	rs := []func(io.Reader) io.Reader{
		func(r io.Reader) io.Reader { return r },
	}
	fmt.Println(len(rs))
}

// skip: inline func-type slice element with qualified iface param yields a zero slice value
