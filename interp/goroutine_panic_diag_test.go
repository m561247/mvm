package interp_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// An unrecovered panic in a spawned goroutine must surface with mvm source
// context (position + stack), not a bare one-line message. The child machine
// inherits the parent's DebugInfo so capturePanic can render the location.
func TestGoroutinePanicCarriesSourceContext(t *testing.T) {
	i := newAutoImportInterp(t)
	var out, errBuf bytes.Buffer
	i.SetIO(os.Stdin, &out, &errBuf)

	// The goroutine derefs a nil pointer; main blocks on a channel so the
	// fault aborts the wait and surfaces (propagate policy).
	src := `
type T struct{ v int }
func boom() int {
	var p *T
	return p.v
}
go func() { _ = boom() }()
ch := make(chan int)
<-ch
`
	if _, err := i.Eval("gtest", src); err == nil {
		t.Fatal("expected a non-nil error from the goroutine fault")
	}
	got := errBuf.String()
	if !strings.Contains(got, "panic in goroutine") {
		t.Fatalf("missing goroutine-panic header:\n%s", got)
	}
	// The DebugInfo-backed render includes an "mvm stack:" section and the
	// panicking function's source location; the bare fallback has neither.
	if !strings.Contains(got, "mvm stack:") {
		t.Fatalf("missing mvm stack (no source context inherited):\n%s", got)
	}
	if !strings.Contains(got, "boom") || !strings.Contains(got, "gtest:") {
		t.Fatalf("missing source location of the panic:\n%s", got)
	}
}
