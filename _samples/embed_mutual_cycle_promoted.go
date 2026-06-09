package main

// Legal mutual struct cycle broken by a pointer, where a field is read PROMOTED
// THROUGH the cycle. hidden reaches Common only via *struct{ Common }; Common
// embeds hidden by value. Reading c.Alias.Type promotes Type from the embedded
// Common inside the anonymous struct: that anon struct's layout must embed the
// fully-sized Common, not the size-0 in-flight placeholder seen mid-cycle.
// (cf. x/text/unicode/cldr's Elem types.)

import "fmt"

type hidden struct {
	Alias *struct {
		Common
		Source string
	}
}

type Common struct {
	Type string
	hidden
}

func (c *Common) GetType() string { return c.Type }

func main() {
	c := Common{Type: "root"}
	c.Alias = &struct {
		Common
		Source string
	}{Common{Type: "nested"}, "src"}
	fmt.Println(c.Alias.Type, c.Alias.Source, len(c.Alias.Type))
}

// Output:
// nested src 6
