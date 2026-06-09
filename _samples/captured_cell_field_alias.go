// Assigning a method's struct-field return to a captured variable must store
// a detached copy in the closure cell, not a reflect value aliasing the source
// field: a later reset of the field (e.g. a pooled object reuse) must not
// mutate the captured variable. Was zerolog TestSlogHandler_HandlePropagatesContext.
package main

import "fmt"

type box struct{ v interface{} }

func (b *box) get() interface{} { return b.v }

func call(f func()) { f() }

func main() {
	b := &box{v: "original"}
	var got interface{}
	call(func() { got = b.get() })
	b.v = nil // reset the source field, like a pooled object reuse
	fmt.Println(got)
}

// Output:
// original
