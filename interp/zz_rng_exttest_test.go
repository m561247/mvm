package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/modfs"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
	"github.com/mvm-sh/mvm/stdlib/stdmod"
)

// Regression for oklog/ulid TestMonotonicSafe. In an external `package X_test`
// unit, a `var name = expr` whose name also denotes an (unexported) type in the
// package under test -- ulid's `type rng` vs the test's `var rng = rand.New(..)`
// -- must declare a variable with its type inferred from expr, not be read as an
// unnamed var of that type. Before the fix, `rng` got the interface type (nil
// value), so `ulid.Monotonic(rng, 0)` received a nil reader and the 100-goroutine
// loop nil-deref'd inside bufio.
func TestExternalTestVarNameMatchesPkgType(t *testing.T) {
	url, _ := startFakeProxy(t, remoteModule{
		path:    "example.com/x/rmod",
		version: "v1.0.0",
		files: map[string]string{
			"go.mod": "module example.com/x/rmod\n",
			"rmod.go": `package rmod

// rng (unexported) shares its name with the external test's local var.
type rng interface{ Int63n(n int64) int64 }

var _ rng

func Stamp() uint64 { return 1 }
`,
			"rmod_test.go": `package rmod_test

import (
	"math/rand"
	"testing"
	"time"

	"example.com/x/rmod"
)

func TestRng(t *testing.T) {
	_ = rmod.Stamp()
	var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	var buf [8]byte
	if n, err := rng.Read(buf[:]); err != nil || n != 8 {
		t.Fatalf("rng.Read: n=%d err=%v (var rng mis-typed by package type rng)", n, err)
	}
}
`,
		},
	})

	var stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &bytes.Buffer{}, &stderr)
	mfs := modfs.New(modfs.Options{Proxy: url})
	if err := mfs.Inject(stdmod.ModulePath, stdmod.Version, stdlib.EmbeddedStd()); err != nil {
		t.Fatalf("inject std: %v", err)
	}
	i.SetStdlibFS(stdmod.FS(mfs))
	i.SetRemoteFS(mfs)
	i.SetIncludeTests(true)

	// Loading the target compiles its external `package X_test` sources; before
	// the fix this failed with "undefined: Read" (rng resolved to the type).
	if _, err := i.Eval("example.com/x/rmod", ""); err != nil {
		t.Fatalf("load target: %v\nstderr: %s", err, stderr.String())
	}
	i.PublishCompiledPackage("example.com/x/rmod")
	if _, err := i.EvalFiles(i.ExternalTestSources()); err != nil {
		t.Fatalf("load external tests: %v\nstderr: %s", err, stderr.String())
	}
}
