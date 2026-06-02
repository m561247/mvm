package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// errors.AsType[*fs.PathError] instantiates the [E error] shim with a pointer
// to a bridged type. The constraint check runs before the type arg is
// materialized, so *fs.PathError had a nil Rtype (its bridged base sits on
// ElemType) and was wrongly rejected as not satisfying error. The expression
// path (TestExpr) materializes earlier and never hit this; only the full
// package/file path did, which is why this uses a complete program.
// argImplementsIface now materializes the arg before the reflect checks.
func TestAsTypeBridgedPtrConstraint(t *testing.T) {
	src := `package main

import (
	"errors"
	"fmt"
	"io/fs"
)

func main() {
	var err error = &fs.PathError{Op: "open", Path: "x", Err: fmt.Errorf("boom")}
	pe, ok := errors.AsType[*fs.PathError](err)
	fmt.Println(ok, pe.Path)
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("astype.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	if got, want := stdout.String(), "true x\n"; got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}
