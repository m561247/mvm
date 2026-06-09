// Two sibling closures each declaring a func-local anonymous struct with the
// same field names but different field types must not share a type symbol key.
// The collision left the first literal's Fnew unpatched (nil slice), so
// building it panicked with "reflect: slice index out of range".
package main

import "fmt"

func run(f func()) { f() }

func main() {
	run(func() {
		cases := []struct {
			field  string
			output string
		}{
			{"a", "x"},
		}
		fmt.Println(cases)
	})
	run(func() {
		cases := []struct {
			field  interface{}
			output string
		}{
			{0, "y"},
		}
		fmt.Println(cases)
	})
}

// Output:
// [{a x}]
// [{0 y}]
