//go:build ignore

package main

const N = 10000000

func sieve() int {
	s := make([]bool, N+1)
	count := 0
	for i := 2; i <= N; i++ {
		if !s[i] {
			count++
			for j := i * i; j <= N; j += i {
				s[j] = true
			}
		}
	}
	return count
}

func main() {
	println(sieve())
}
