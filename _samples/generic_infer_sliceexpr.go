package main

// Generic inference through a slice expression (s[lo:]) and a make() argument.
func first[S ~[]E, E any](s S) E { return s[0] }

func main() {
	data := []int{7, 8, 9}
	println(first(data[1:]))       // arg is a slice expression with an omitted bound
	println(first(make([]int, 1))) // arg is a make() call
}

// Output:
// 8
// 0
