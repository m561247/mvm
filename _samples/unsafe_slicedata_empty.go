package main

// unsafe.SliceData on a len==0 cap>0 slice returns the underlying array
// pointer (the bridge used to Index(0) and panic out of range).

import "unsafe"

func main() {
	s := make([]int, 0, 4)
	p := unsafe.SliceData(s)
	println(p != nil)

	var nilSlice []int
	println(unsafe.SliceData(nilSlice) == nil)
}

// Output:
// true
// true
