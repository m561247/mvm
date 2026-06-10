package interp_test

import (
	"fmt"
	"testing"
)

// A spread call split across lines carries a trailing comma after the
// ellipsis (`f(a,\n b...,\n)`); spread detection must skip it or the slice
// is bound to the first variadic slot. Was gjson TestManyBasic:
// "reflect.Set: value of type []string is not assignable to type string".
func TestVariadicSpreadTrailingComma(t *testing.T) {
	src := `
f := func(prefix string, parts ...string) string {
	out := prefix
	for _, p := range parts {
		out += "," + p
	}
	return out
}
parts := []string{"a", "b"}
f(
	"p",
	parts...,
)`
	i := newAutoImportInterp(t)
	r, err := i.Eval("spread", src)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got, want := fmt.Sprintf("%v", r), "p,a,b"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
