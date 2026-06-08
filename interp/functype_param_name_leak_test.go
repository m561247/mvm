package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Regression for `mvm test github.com/samber/lo` -> ExampleNewDebounceBy panic
// `index out of range [-1]`.
//
// A func TYPE's parameter names are documentation only, but parseTypeExpr's func
// case registered them as locals in the enclosing scope (it shares that code with
// real func-literal signatures). When a func-type param name collided with an
// outer param -- here `[]func(key string, count int){}` inside a method
// `reset(key string)` -- the leaked `key` rebound the method's `key` to a wrong
// frame slot (the receiver shifts param indices, so the slots no longer coincide).
// A later use of `key` then loaded garbage (a func value), corrupting the call.
// Fixed by registering func-signature params only for a genuine declaration/
// literal (parseFunc sets regFuncSig); a func TYPE leaves it unset.
func TestFuncTypeParamNameNoLeak(t *testing.T) {
	src := `package main

import "fmt"

type deb struct {
	callbacks []func(key string, count int)
}

func (d *deb) reset(key string) {
	// The func-type param name 'key' collides with the method param 'key'.
	callbacks := append([]func(key string, count int){}, d.callbacks...)
	for i := range callbacks {
		callbacks[i](key, i+1)
	}
}

// A func-type alias whose param name also collides with an outer param.
func run(key string, f func(key string, n int)) {
	type handler func(key string, n int)
	var h handler = f
	h(key, 7)
}

func main() {
	d := &deb{callbacks: []func(key string, count int){
		func(userID string, count int) { fmt.Println("cb1", userID, count) },
		func(userID string, count int) { fmt.Println("cb2", userID, count) },
	}}
	d.reset("samuel")
	run("john", func(userID string, n int) { fmt.Println("run", userID, n) })
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("functype_leak.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "cb1 samuel 1\ncb2 samuel 2\nrun john 7\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout: got %q, want %q (stderr: %s)", got, want, stderr.String())
	}
}
