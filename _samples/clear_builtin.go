package main

func main() {
	s := []int{1, 2, 3}
	clear(s)
	println(s[0], s[1], s[2])

	m := map[string]int{"a": 1, "b": 2}
	clear(m)
	println(len(m))

	b := []byte{7, 8, 9}
	defer func() { println(b[0], b[1], b[2]) }()
	defer clear(b)
}

// Output:
// 0 0 0
// 0
// 0 0 0
