package main

// Regression test: a generic function whose type-parameter name matches an
// existing package-level type name must not delete that type. registerFunc
// (goparser/func.go) installs temporary type-param placeholders at the bare
// symbol key; it now saves and restores the prior symbol (mirroring
// parseTypeParamList in goparser/generic.go) instead of deleting it
// unconditionally, so `type T int` survives the type parameter of
// `func F[T any]`.
//
// This is valid Go: a type parameter's scope is limited to its function, so
// the package-level `T` is still in scope at `var v T` in main. Before the
// fix mvm reported "undefined: T".

type T int

func F[T any](x T) T { return x }

func main() {
	var v T = 5
	println(int(v))
	println(F(3))
}

// Output:
// 5
// 3
