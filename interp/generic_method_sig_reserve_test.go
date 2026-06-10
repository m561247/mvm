package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/modfs"
	"github.com/mvm-sh/mvm/stdlib"
)

// A generic type instantiated with a named type in a method SIGNATURE
// (gjson: func (t Result) All() iter.Seq2[Result, Result]) must not
// materialize the type argument at parse time: that runs before
// preregisterMethods, so the reserve gate would see no methods and stamp a
// methodless identity that AttachSynthMethods cannot fill ("has no
// reservation at attach"). Was github.com/tidwall/gjson.
func TestGenericMethodSigReserve(t *testing.T) {
	url, _ := startFakeProxy(t, remoteModule{
		path:    "example.com/x/j",
		version: "v1.0.0",
		files: map[string]string{
			"go.mod": "module example.com/x/j\n",
			"j.go": `package j

type Seq2[K, V any] func(yield func(K, V) bool)

type Type int

func (t Type) String() string { return "x" }

type Result struct {
	Type Type
	Raw  string
}

func (t Result) String() string { return t.Raw }

func (t Result) All() Seq2[Result, Result] {
	return func(yield func(Result, Result) bool) {}
}
`,
		},
	})

	var stdout bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, os.Stderr)
	i.SetRemoteFS(modfs.New(modfs.Options{Proxy: url}))

	src := `import "example.com/x/j"; var r j.Result; r.Raw = "hi"; println(r.String())`
	if _, err := i.Eval("test", src); err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got, want := stdout.String(), "hi\n"; got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}
