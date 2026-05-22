package main

// Regression: hex floating-point literals (a 'p'/'P' binary exponent) used to
// be mis-scanned -- 0x1p-1022 stopped at 'p', leaving 'p' as an undefined
// identifier. The scanner now reads them as a single Float token.
func main() {
	println(0x1p4)
	println(0x1.8p3)
	println(0x1.fp4)
	println(-0x1p+8)
}

// Output:
// 16
// 12
// 31
// -256
