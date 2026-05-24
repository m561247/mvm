package main

// Regression: a parallel assignment swapping two pointer variables (a, b = b, a)
// must snapshot the RHS operands. A local's Value.ref is an addressable reflect.Value
// aliasing its source cell, so without the DetachRef pass in the batched multi-assign
// the first store writes through and corrupts the not-yet-consumed operand, collapsing
// both variables to one value. This is the reduced, dependency-free form of math/big's
// Example_fibonacci, which does `a.Add(a, b); a, b = b, a` and printed a wrong number
// before the fix.
func main() {
	x, y := 0, 1
	a, b := &x, &y
	for i := 0; i < 10; i++ {
		*a = *a + *b
		a, b = b, a
	}
	println(*b)
}

// Output:
// 89
