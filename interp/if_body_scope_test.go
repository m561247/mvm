package interp_test

import (
	"fmt"
	"testing"
)

// A := inside an if/else-if body must not leak into later else-if conditions
// or sibling bodies: parseIf parses the whole chain in one scope, so each
// body gets its own sub-scope. Was tidwall/pretty appendPrettyObject (a
// shadowed `max :=` in the then-body made `max != -1` read the uninitialized
// body slot in the else-if condition).
func TestIfBodyScope(t *testing.T) {
	cases := []struct{ n, src, res string }{
		{"elseif_cond", `x := -1; r := ""; if x == 0 { x := 5; _ = x } else if x != -1 { r = "bad" } else { r = "good" }; r`, "good"},
		{"chained", `x := -1; r := ""; if x == 0 { x := 1; _ = x } else if x == 1 { x := 2; _ = x } else if x != -1 { r = "bad" } else { r = "good" }; r`, "good"},
		{"sibling_bodies", `x := 1; a := 0; if x == 1 { y := 10; a = y } else { y := 20; a = y }; a`, "10"},
		{"init_visible_in_body", `a := 0; if v := 7; v > 0 { a = v }; a`, "7"},
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
