package main

// Regression: var-init dep analysis follows method calls on concrete
// types. FromMethod's init invokes R{}.First(), whose body reads Table.
// The parser-side analysis (Phase 1.5 in ParseAll) walks R.First's body,
// finds the Table reference, and topo-sorts Table before FromMethod.
import "fmt"

type R struct{}

func (R) First() byte { return Table[0] }

var (
	FromMethod = R{}.First()
	Table      = [256]byte{0xaa, 0xbb}
)

func main() {
	fmt.Printf("Table=%x FromMethod=%x\n", Table[0], FromMethod)
}

// Output:
// Table=aa FromMethod=aa
