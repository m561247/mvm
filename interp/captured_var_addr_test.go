package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Regression for `mvm test github.com/samber/lo` -> ExampleNewDebounce panic
// `reflect.Value.Convert: *int cannot be converted to *int32`.
//
// A local captured by a closure is promoted to a heap cell. Taking its address
// (`&i`) was broken two ways: (1) for a `:=` var the cell inherited the value's
// generic ref, so a sized-numeric (`i := int32(0)`) yielded *int not *int32; the
// non-cell path masks this by typing the slot via vm.New, skipped for cells.
// (2) the cell's ref was non-addressable, so &i pointed at a transient copy and
// writes through it (atomic ops here) never reached the cell -- the closure and
// later reads saw the stale value. Fixed by converting `:=` numeric values to
// the declared type before HeapAlloc, and by making HeapAlloc detach numeric
// values into fresh addressable storage (enabling CellGet's num<-ref resync).
func TestCapturedVarAddressTypeAndWrite(t *testing.T) {
	src := `package main

import (
	"fmt"
	"sync/atomic"
)

func main() {
	i := int32(0)
	seen := func() int32 { return atomic.LoadInt32(&i) }
	atomic.AddInt32(&i, 1)
	atomic.AddInt32(&i, 1)
	fmt.Println(i, seen())

	// Plain pointer write-through to a captured sized-numeric var.
	j := int32(5)
	_ = func() int32 { return j }
	p := &j
	*p = 9
	fmt.Println(j)
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("captured_addr.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "2 2\n9\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout: got %q, want %q (stderr: %s)", got, want, stderr.String())
	}
}
