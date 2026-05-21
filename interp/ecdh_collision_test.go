package interp

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/modfs"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Regression for `mvm test crypto/ecdh` -> "undefined: ecdh.PrivateKey".
//
// When a package is loaded as a test target, importingPkg is set to its
// path, so a bare identifier that names one of the package's own types
// resolves to that type. A struct field group like `Foo, Bar string`
// then mis-parsed: parseParamTypes processes right-to-left, and the lone
// leftmost ident `Foo` (which also names a type) made hasFirstParam
// report "type-only", so `Foo` became an unnamed embedded field of type
// Foo instead of a field NAME sharing the trailing string type. mvm then
// failed resolving the embedded type. The crypto/ecdh external test hit
// this via `map[ecdh.Curve]struct{ PrivateKey, PublicKey string; ... }`,
// where the field names PrivateKey/PublicKey collide with ecdh's own types.
func TestRemoteFieldNameMatchesLocalType(t *testing.T) {
	url, _ := startFakeProxy(t, remoteModule{
		path:    "example.com/x/coll",
		version: "v1.0.0",
		files: map[string]string{
			"go.mod": "module example.com/x/coll\n",
			"coll.go": `package coll

type Foo struct{ V int }
type Bar struct{ V int }

// Field names Foo, Bar (sharing the string type) collide with the
// package's own Foo/Bar types. The var is a deferred (Phase 2) decl.
var table = map[string]struct {
	Foo, Bar string
}{
	"k": {Foo: "a", Bar: "b"},
}

func Lookup(k string) string { return table[k].Foo + table[k].Bar }
`,
		},
	})

	var stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &bytes.Buffer{}, &stderr)
	i.SetRemoteFS(modfs.New(modfs.Options{Proxy: url}))
	i.SetIncludeTests(true)

	// Direct-target load (mirrors test_cmd's `i.Eval(target, "")`), which sets
	// importingPkg = "example.com/x/coll". Pre-fix this failed with
	// "undefined: Foo" because the field group was mis-parsed.
	if _, err := i.Eval("example.com/x/coll", ""); err != nil {
		t.Fatalf("loading target: %v\nstderr: %s", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "undefined") {
		t.Errorf("unexpected undefined error: %s", stderr.String())
	}
}
