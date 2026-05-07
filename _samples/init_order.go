package main

// Regression: var-init dep analysis follows free-function calls.
// FromTable's init calls computeUsingTable() which reads Table; the
// parser walks computeUsingTable's body, records the Table reference
// on its Reads, and topo-sorts so Table runs first.
import "fmt"

func computeUsingTable() byte { return Table[0] }

var (
	FromTable = computeUsingTable()
	Table     = [256]byte{0xaa, 0xbb, 0xcc, 0xdd}
)

func main() {
	fmt.Printf("Table=%x FromTable=%x\n", Table[0], FromTable)
}

// Output:
// Table=aa FromTable=aa
