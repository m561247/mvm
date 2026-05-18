package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/modfs"
	"github.com/mvm-sh/mvm/stdlib"
)

// TestIssue9MultiReturnTupleAssignCrossPkg is the regression test for
// github.com/mvm-sh/mvm/issues/9. parseImportLine used to register `_` as a
// Kind=symbol.Pkg entry for `import _ "path"`, polluting the symbol table; a
// later `tag, _ = f(tag)` would resolve the blank LHS to that Pkg symbol
// instead of Kind==Unset, miss the blank-shortcut at comp/compiler.go's
// lang.Assign n>1 loop, and fall into the FieldRefSet default branch -- which
// then wrote the bool return into the struct slot, panicking with
// "reflect.Set: value of type bool is not assignable to type struct {P<n> int}".
// Fixed in goparser/decl.go by skipping SymSet when the alias name is "_".
func TestIssue9MultiReturnTupleAssignCrossPkg(t *testing.T) {
	url, _ := startFakeProxy(t,
		remoteModule{
			path:    "example.com/x/inner",
			version: "v1.0.0",
			files: map[string]string{
				"go.mod": "module example.com/x/inner\n",
				"inner.go": `package inner

type Tag struct{ X int }
`,
			},
		},
		remoteModule{
			path:    "example.com/x/outer",
			version: "v1.0.0",
			files: map[string]string{
				"go.mod": "module example.com/x/outer\n",
				"outer.go": `package outer

import "example.com/x/inner"

func f(t inner.Tag) (inner.Tag, bool) { return t, true }

func init() {
	var tag inner.Tag
	tag, _ = f(tag)
	_ = tag
}
`,
			},
		},
	)

	var stdout bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, os.Stderr)
	i.SetRemoteFS(modfs.New(modfs.Options{Proxy: url}))

	if _, err := i.Eval("test", `import _ "example.com/x/outer"`); err != nil {
		t.Fatalf("Eval: %v", err)
	}
}
