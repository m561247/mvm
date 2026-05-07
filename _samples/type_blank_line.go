package main

import "fmt"

type (
	A int

	B int
)

func main() {
	var a A = 1
	var b B = 2
	fmt.Println(a, b)
}

// Output:
// 1 2
