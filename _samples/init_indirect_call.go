package main

// Var-init dep analysis can't see through a function value fetched at
// runtime via an opaque path. Src.Fn() returns the real `first` via
// interface dispatch; the trailing call hits the returned func value
// without surfacing a slot/name reference the analyzer can follow.
// Same shape as `m["k"]()`: the map index and subsequent call carry
// no compile-time pointer at the chosen callee.
//
// As with init_iface.go, Table is given a sibling-dep on Sentinel so
// the missing edge actually changes the topo order.
import "fmt"

type FnSource interface{ Fn() func() byte }

type S struct{}

func (S) Fn() func() byte { return first }

func first() byte { return Table[0] }

func computeTable(b byte) [256]byte { return [256]byte{0xaa, b + 0xa9} }

var (
	Src      FnSource = S{}
	FromFn            = Src.Fn()()
	Sentinel byte     = 1
	Table             = computeTable(Sentinel)
)

func main() {
	fmt.Printf("Table=%x FromFn=%x\n", Table[0], FromFn)
}

// skip: indirect call through a func value fetched at runtime is opaque to var-init dep analysis.
// Output:
// Table=aa FromFn=aa
