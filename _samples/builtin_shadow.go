package main

// Regression: a user-defined function shadowing a predeclared builtin name
// (here `close`, exactly as math's huge_test.go does) was wrongly dispatched to
// the builtin and rejected with "invalid argument count for close".
// compileBuiltin now only treats Kind=Builtin symbols (plus unsafe.*) as
// builtins, so the shadowing function is called normally.
func close(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}

func main() {
	println(close(1.0, 1.0))
}

// Output:
// true
