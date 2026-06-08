package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Regression for `mvm test github.com/samber/lo` -> retry.go
// "cannot infer type parameter T".
//
// `callbacks := Map(f, func(...) func(struct{}) {...})` then
// `NewThrottleByWithCount(interval, count, callbacks...)` must infer T=struct{}
// from callbacks's type. The inference reads the arg type via callFuncType /
// postfixType, which walk a parsed postfix right-to-left to locate the callee.
// A closure argument is emitted inline as its whole definition block
// (`Goto X_end; Label X; ...body...; Label X_end` then the value Ident "X"),
// and postfixType consumed only the trailing closure ident, derailing the arg
// walk. callFuncType then returned nil, leaving callbacks's type nil, so the
// later generic call could not infer T. Fixed by consuming the entire closure
// block as one operand in postfixType.
func TestGenericInferFromGenericCallResult(t *testing.T) {
	src := `package main

import "fmt"

func Map[T, R any](collection []T, transform func(item T, index int) R) []R {
	result := make([]R, len(collection))
	for i := range collection {
		result[i] = transform(collection[i], i)
	}
	return result
}

func Collect[T comparable](count int, f ...func(key T)) int {
	return len(f) + count
}

func main() {
	f := []func(){func() {}, func() {}}
	callbacks := Map(f, func(item func(), _ int) func(struct{}) {
		return func(struct{}) { item() }
	})
	fmt.Println(Collect(1, callbacks...))
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("infer_call_result.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	// len(callbacks)=2 + count(1) = 3.
	if got := stdout.String(); got != "3\n" {
		t.Errorf("stdout: got %q, want %q (stderr: %s)", got, "3\n", stderr.String())
	}
}
