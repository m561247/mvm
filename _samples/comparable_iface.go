package main

// `comparable` is recognized in type-parameter constraint position
// (goparser/generic.go: `case "comparable"`), but not as an embedded element
// of an interface type definition. Parsing `interface { comparable; error }`
// tries to resolve `comparable` as an ordinary type name and fails with
// `undefined: comparable`. Surfaced by `mvm test errors` on go1.26, whose
// wrap_test.go declares `type compError interface { comparable; error }` and
// uses it to constrain the generic errors.AsType[E].
import "fmt"

type compError interface {
	comparable
	error
}

func first[E compError](xs []E) (E, bool) {
	var zero E
	for _, x := range xs {
		if x != zero {
			return x, true
		}
	}
	return zero, false
}

func main() {
	_, ok := first[error](nil)
	fmt.Println(ok)
}

// skip: `comparable` not supported as an embedded interface element (only in
// type-parameter constraint position). Reports `undefined: comparable`.
// Output:
// false
