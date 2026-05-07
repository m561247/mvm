package main

// Var-init dep analysis can't see through interface method dispatch:
// Iface.First() goes through a runtime vtable lookup, so the analyzer
// (whether parser-side or bytecode-derived) doesn't know FromIface
// reaches Table through R.First.
//
// To make the failure observable rather than masked by topo-luck, the
// init gives Table a compile-order dep on a sibling Sentinel. Without
// this, Kahn happens to schedule Table before FromIface and the bug
// hides.
import "fmt"

type Reader interface{ First() byte }

type R struct{}

func (R) First() byte { return Table[0] }

func computeTable(b byte) [256]byte { return [256]byte{0xaa, b + 0xa9} }

var (
	Iface     Reader = R{}
	FromIface        = Iface.First()
	Sentinel  byte   = 1
	Table            = computeTable(Sentinel)
)

func main() {
	fmt.Printf("Table=%x FromIface=%x\n", Table[0], FromIface)
}

// skip: interface method dispatch is opaque to var-init dep analysis.
// Output:
// Table=aa FromIface=aa
