package interp_test

import (
	"fmt"
	"testing"
)

// A condition switch keeps its operand on the stack through the EqualSet
// chain; without a default clause the no-match exit must drop it. The leak
// (one slot per no-match execution) overflowed the frame in a hot loop.
// Was gjson revSquash: "index out of range [257] with length 257".
func TestSwitchOperandLeak(t *testing.T) {
	cases := []struct{ n, src, res string }{
		{"no_match_loop", `n := 0; for i := 0; i < 2000; i++ { switch i % 251 { case -1: n++ } }; n`, "0"},
		{"rare_match_loop", `n := 0; for i := 0; i < 2000; i++ { switch i % 251 { case 7: n++ } }; n`, "8"},
		{"multi_value_loop", `n := 0; for i := 0; i < 2000; i++ { switch i % 251 { case 1, 2, 3: n++ } }; n`, "24"},
		{"last_case_match", "n := 0\nfor i := 0; i < 2000; i++ {\n\tswitch 7 {\n\tcase 6:\n\tcase 7:\n\t\tn++\n\t}\n}\nn", "2000"},
		{"with_default", "n := 0\nfor i := 0; i < 2000; i++ {\n\tswitch i % 251 {\n\tcase 7:\n\tdefault:\n\t\tn++\n\t}\n}\nn", "1992"},
		// Return-terminated bodies empty the compile-time stack model; the
		// drop emitted at the no-match merge must not underflow it.
		{"return_bodies", "f := func(s int) string {\n\tswitch s {\n\tcase 1:\n\t\treturn \"small\"\n\tcase 2:\n\t\treturn \"large\"\n\t}\n\treturn \"other\"\n}\nout := \"\"\nfor i := 0; i < 2000; i++ {\n\tout = f(3)\n}\nout", "other"},
	}
	for _, c := range cases {
		t.Run(c.n, func(t *testing.T) {
			i := newAutoImportInterp(t)
			r, err := i.Eval(c.n, c.src)
			if err != nil {
				t.Fatalf("eval %q: %v", c.src, err)
			}
			if got := fmt.Sprintf("%v", r); got != c.res {
				t.Errorf("got %q, want %q", got, c.res)
			}
		})
	}
}
