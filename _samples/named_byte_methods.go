package main

import "fmt"

type (
	A byte
	B byte
)

func (a A) String() string { return "A-string" }
func (b B) String() string { return "B-string" }

func main() {
	var a A = 0
	var b B = 0
	var ai any = a
	var bi any = b
	fmt.Println(ai.(fmt.Stringer).String())
	fmt.Println(bi.(fmt.Stringer).String())
}

// Output:
// A-string
// B-string
