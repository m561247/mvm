package interp

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Storing an interface-typed local into a map whose element type is a narrower
// bridged interface (here image.Image) used to panic in MapSet:
// "reflect.Value.SetMapIndex: value of type interface {} is not assignable to
// type image.Image". Interface locals are boxed as interface{} (or an mvm
// Iface), neither directly assignable to the map's image.Image slot. The fix
// in wrapForFunc bridges/unwraps to the concrete element first. Repro of the
// `mvm test image` TestDecode failure (golden[name] = g).
func TestIfaceMapStoreBridgedInterface(t *testing.T) {
	src := `package main

import (
	"fmt"
	"image"
)

func main() {
	m := make(map[string]image.Image)
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var g image.Image = img
	m["a"] = g
	fmt.Println(m["a"].Bounds())
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("test", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "panic") {
		t.Fatalf("got panic: %s", stderr.String())
	}
	if got, want := stdout.String(), "(0,0)-(2,2)\n"; got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}
