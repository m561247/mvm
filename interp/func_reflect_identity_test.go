package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// reflect.ValueOf(f) == reflect.ValueOf(f) must hold for a top-level func, as in
// Go (one global funcval). funcWrappers memoises the bridge wrapper to preserve
// it; previously each ValueOf minted a fresh wrapper that compared unequal. This
// is the Masterminds/semver TestParseConstraint pattern.
func TestFuncReflectIdentity(t *testing.T) {
	src := `package main

import (
	"fmt"
	"reflect"
)

type cfunc func(int) int

func gte(x int) int { return x + 1 }
func lt(x int) int  { return x - 1 }

var ops = map[string]cfunc{">=": gte, "<": lt}

func main() {
	var f cfunc = gte
	// Same func, two separate ValueOf calls: equal, as in native Go.
	fmt.Println(reflect.ValueOf(f) == reflect.ValueOf(f))
	// The semver pattern: direct ref vs map lookup of the same func.
	fmt.Println(reflect.ValueOf(gte) == reflect.ValueOf(ops[">="]))
	// Distinct funcs must stay distinct.
	fmt.Println(reflect.ValueOf(gte) == reflect.ValueOf(ops["<"]))
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("func_reflect_identity.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "true\ntrue\nfalse\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}
