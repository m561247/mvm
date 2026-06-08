package interp_test

import (
	"testing"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// An early return out of a range-over-func must call the iterator's stop, which
// resumes the pull coroutine so it runs its cleanup (here a defer) and exits.
// The buggy dropIterFrames popped the iterator without stopping it on the
// return path: the coroutine stayed suspended (leaked) and the deferred cleanup
// never ran. (break compiles to the Stop opcode, so only return/panic exercise
// dropIterFrames.)
func TestRangeFuncEarlyReturnStopsIterator(t *testing.T) {
	intp := interp.NewInterpreter(golang.GoSpec)
	intp.ImportPackageValues(stdlib.Values)
	if _, err := intp.Eval("setup", `
var cleaned bool
func seq(yield func(int) bool) {
	defer func() { cleaned = true }()
	for i := 0; ; i++ {
		if !yield(i) {
			return
		}
	}
}
func find() int {
	for v := range seq {
		if v == 3 {
			return v
		}
	}
	return -1
}
func run() bool {
	cleaned = false
	find()
	return cleaned
}
`); err != nil {
		t.Fatal(err)
	}
	res, err := intp.Eval("run", "run()")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Bool() {
		t.Fatal("iterator cleanup skipped on early return: stop() not called, coroutine leaked")
	}
}
