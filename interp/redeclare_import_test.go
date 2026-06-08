package interp_test

import (
	"testing"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
)

// TestRedeclareAsImport guards the file-block vs package-block collision: a
// top-level name (var/const/type/func) that clashes with an imported package
// name in the same file. Go rejects it ("X already declared through import");
// mvm previously clobbered the shared bare-key symbol, yielding a runtime
// nil-deref (var) or a misleading "is not a type" (type). It must now be a
// clean, located redeclaration error. A local shadow of an import is valid Go
// and must still resolve.
func TestRedeclareAsImport(t *testing.T) {
	run(t, []etest{
		{n: "var_vs_import", src: `import "sort"; var sort = 1; func run() int { return 0 }; run()`, err: "redeclared in this block"},
		{n: "const_vs_import", src: `import "sort"; const sort = 1; func run() int { return 0 }; run()`, err: "redeclared in this block"},
		{n: "type_vs_import", src: `import "sort"; type sort = int; func run() int { return 0 }; run()`, err: "redeclared in this block"},
		{n: "func_vs_import", src: `import "sort"; func sort() {}; func run() int { return 0 }; run()`, err: "redeclared in this block"},
		{n: "grouped_var_vs_import", src: `import "sort"; var ( a = 1; sort = 2 ); func run() int { return a }; run()`, err: "redeclared in this block"},

		// Valid Go: a local name shadows the imported package -- distinct scoped
		// key, must resolve to the local, never trip the check.
		{n: "local_shadow_ok", src: `import "sort"; func run() int { sort := 42; return sort }; run()`, res: "42"},
	})
}

// TestRedeclareVsAutoImport guards against a false positive: in REPL/-e/test
// mode every loaded package is ambient-bound under its short name (sort, bytes,
// ...). A top-level decl of such a name with NO explicit import is valid Go and
// must shadow the convenience binding, not be rejected as a redeclaration.
func TestRedeclareVsAutoImport(t *testing.T) {
	for _, name := range []string{"sort", "bytes", "time"} {
		t.Run(name, func(t *testing.T) {
			intp := interp.NewInterpreter(golang.GoSpec)
			intp.ImportPackageValues(stdlib.Values)
			intp.AutoImportPackages() // ambient-bind sort/bytes/time as Pkg symbols
			if _, err := intp.Eval("t", "var "+name+" = 42\n"); err != nil {
				t.Fatalf("var %s = 42 with no explicit import: unexpected error %v", name, err)
			}
			r, err := intp.Eval("t2", name+"\n")
			if err != nil {
				t.Fatalf("read back %s: %v", name, err)
			}
			if got := r.Interface(); got != 42 {
				t.Fatalf("%s = %v, want 42", name, got)
			}
		})
	}
}
