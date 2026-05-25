package main

// A zero-value map/slice compares != nil in package context (ptr/func/chan are
// fine); `mvm run -e` evaluates the same source correctly.
// All three lines below should print true once fixed.
import "maps"

func main() {
	var m map[string]int
	var s []int
	println(m == nil)
	println(s == nil)
	println(maps.Clone(m) == nil)
}

// skip: nil map/slice zero-value compares != nil in package context (pre-existing).
