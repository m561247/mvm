package interp_test

import (
	"testing"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// TestInitRunsOncePerEval guards a re-entrance regression where a package init
// function re-ran on every later Eval.
func TestInitRunsOncePerEval(t *testing.T) {
	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.AutoImportPackages()

	if _, err := i.Eval("m:init", "var initRuns int\nfunc init() { initRuns++ }"); err != nil {
		t.Fatalf("first eval: %v", err)
	}
	// A second, unrelated Eval must not re-run the init defined above.
	if _, err := i.Eval("m:more", "var x = 1\n_ = x"); err != nil {
		t.Fatalf("second eval: %v", err)
	}
	res, err := i.Eval("m:read", "initRuns")
	if err != nil {
		t.Fatalf("read eval: %v", err)
	}
	if got := res.Interface(); got != 1 {
		t.Fatalf("init ran %v times, want 1", got)
	}
}
