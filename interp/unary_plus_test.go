package interp_test

import "testing"

// TestUnaryPlusPrecedence guards unary `+` precedence: it was missing from the
// TokenProps table, so `ord == +1` mis-parsed and corrupted the compile stack.
func TestUnaryPlusPrecedence(t *testing.T) {
	run(t, []etest{
		{n: "eq_plus", src: `ord := 1; ord == +1`, res: "true"},
		{n: "eq_minus", src: `ord := 1; ord == -1`, res: "false"},
		{n: "and_eq_plus", src: `a, b, ord := 3, 1, 1; a > b && ord == +1`, res: "true"},
		{n: "unary_var", src: `ord := 5; +ord`, res: "5"},
		{n: "mixed_unary", src: `+5 * -3`, res: "-15"},
		{n: "short_circuit_chain", src: `
			t1, t2, s1, s2, ord := 2, 1, 2, 1, 1
			t1 == t2 ||
				(t1 > t2 && s1 > s2 && ord == +1) ||
				(t1 < t2 && s1 < s2 && ord == -1)`, res: "true"},
	})
}
