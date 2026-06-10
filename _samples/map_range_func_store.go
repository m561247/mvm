package main

// Func values stored from a map range into a func-typed slice must keep
// their identity: wrapping used to capture the live loop-var slot, so every
// stored func aliased one closure (goldmark renderer registration).

import "fmt"

type Fn func() string

func mk(s string) Fn { return func() string { return s } }

func main() {
	tmp := map[int]Fn{}
	for i := 0; i < 8; i++ {
		tmp[i] = mk(fmt.Sprintf("f%d", i))
	}
	funcs := make([]Fn, 8)
	for k, f := range tmp {
		funcs[k] = f
	}
	for i := 0; i < 8; i++ {
		if got, want := funcs[i](), fmt.Sprintf("f%d", i); got != want {
			fmt.Println("MISPAIRED:", i, got, want)
			return
		}
	}
	fmt.Println("ok")
}

// Output:
// ok
