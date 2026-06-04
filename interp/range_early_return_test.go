package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// An early `return` from inside a range loop compiles to the fused GetLocalReturn
// opcode (single-local return, no defers). That opcode used to skip the iterator
// unwind that plain Return does, leaking the loop iterator on m.iterStack. The
// next outer range step then read the inner (string) iterator and tried to assign
// a rune to a string loop var: "reflect.Set: value of type int32 is not
// assignable to type string". Minimized from `mvm test unicode/utf8`
// (TestDecodeInvalidSequence -> runtimeDecodeRune).
func TestRangeEarlyReturnIteratorLeak(t *testing.T) {
	src := `package main

import "fmt"

var tests = []string{"ab", "cd", "ef"}

// firstRune ranges over a string and returns from inside the loop.
func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return -1
}

func main() {
	for _, s := range tests {
		fmt.Printf("%q %#x\n", s, firstRune(s))
	}
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("range_early_return.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "\"ab\" 0x61\n\"cd\" 0x63\n\"ef\" 0x65\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout:\n got %q\nwant %q", got, want)
	}
}
