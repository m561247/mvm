package main

// Same legal mutual struct cycle as embed_mutual_cycle_promoted.go, but the cycle
// CONTAINER (Named) carries a method, so it materializes through the reserved
// method-bearing path (maybeReserveStruct) rather than the plain placeholder path.
// That path must defer its layout the same way when an embedded by-value struct
// (Common) is still an in-flight placeholder mid-cycle.

import "fmt"

type hidden struct {
	Alias *Named
}

type Named struct {
	Common
	Source string
}

func (n *Named) Tag() string { return n.Source }

type Common struct {
	Type string
	hidden
}

func main() {
	c := Common{Type: "root"}
	c.Alias = &Named{Common{Type: "nested"}, "src"}
	fmt.Println(c.Alias.Type, c.Alias.Tag(), len(c.Alias.Type))
}

// Output:
// nested src 6
