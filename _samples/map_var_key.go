package main

func main() {
	l := "x"
	m := map[string]bool{l: true}
	println(m[l])

	k1, k2 := "a", "b"
	m2 := map[string]int{k1: 1, "c": 3, k2: 2}
	println(m2["a"], m2["b"], m2["c"])
}

// Output:
// true
// 1 2 3
