package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Regression for `mvm test github.com/bmatcuk/doublestar/v4` ->
// TestSkipDirInGlobWalk corrupted the package-level SkipDir sentinel.
//
// The compiler materializes a local slot's own storage (vm.New) only at the
// first TEXTUAL assignment. A named return assigned first in a later branch
// reached SetLocal with an unmaterialized slot, so assignSlot adopted the
// source ref verbatim. When the source was a global's value returned by a
// callee, the slot aliased the global's storage and a following `e = nil`
// wrote through it, nulling the global for the rest of the run.
// Fixed in vm.assignSlot: never adopt a settable ref; detach into fresh storage.
func TestNamedReturnAssignNoGlobalAlias(t *testing.T) {
	src := `package main

import (
	"errors"
	"fmt"
)

var sentinel = errors.New("skip")
var word = "hello"

func getErr() error  { return sentinel }
func getStr() string { return word }

// The if-branch is the first textual assignment to e (it gets the slot-
// materializing vm.New); the else branch, executed here, does not.
func walk(which bool) (e error) {
	if which {
		if e = getErr(); e != nil {
			return
		}
	} else {
		if e = getErr(); e != nil {
			e = nil
			return
		}
	}
	return
}

func wstr(which bool) (s string) {
	if which {
		if s = getStr(); s != "" {
			return
		}
	} else {
		if s = getStr(); s != "" {
			s = ""
			return
		}
	}
	return
}

func main() {
	walk(false)
	wstr(false)
	fmt.Println(sentinel, word)
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("named_return_alias.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "skip hello\n"
	if got := stdout.String(); got != want {
		t.Errorf("globals clobbered through a named-return slot: got %q, want %q", got, want)
	}
}
