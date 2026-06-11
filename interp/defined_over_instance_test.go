package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// A defined type over a generic instance (type BTree BTreeG[int]) shares the
// instance's canonical type, and both declare a same-named method.
// MethodByName's unnamed-receiver scan matched either by map order, so calls
// nondeterministically hit the wrong Delete: a 2-value := from a 1-result
// method underflowed codegen, and the other direction returned extra values
// (google/btree's backwards-compat wrappers).
func TestDefinedTypeOverGenericInstanceMethod(t *testing.T) {
	src := `package main

import "fmt"

type BTreeG[T any] struct{ n int }

func (t *BTreeG[T]) Delete(item T) (T, bool) {
	return item, true
}

type BTree BTreeG[int]

func (t *BTree) Delete(item int) int {
	i, _ := (*BTreeG[int])(t).Delete(item)
	return i
}

func main() {
	t := &BTree{}
	fmt.Println(t.Delete(5))
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, &stderr)
	if _, err := i.Eval("a.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	if got, want := stdout.String(), "5\n"; got != want {
		t.Errorf("stdout = %q, want %q\nstderr: %s", got, want, stderr.String())
	}
}
