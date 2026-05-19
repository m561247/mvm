package interp_test

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/vm"
)

func TestOsExitReturnsExitError(t *testing.T) {
	i := newAutoImportInterp(t)
	_, err := i.Eval("exit", "os.Exit(42)")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ee *interp.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *interp.ExitError, got %T: %v", err, err)
	}
	if ee.Code != 42 {
		t.Errorf("Code = %d, want 42", ee.Code)
	}
}

func TestLogFatalReturnsExitError(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	i := newAutoImportInterp(t)
	_, err := i.Eval("fatal", `log.Fatal("boom")`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ee *interp.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *interp.ExitError, got %T: %v", err, err)
	}
	if ee.Code != 1 {
		t.Errorf("Code = %d, want 1", ee.Code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Errorf("log output missing %q:\n%s", "boom", buf.String())
	}
}

// TestRuntimeErrorStillPanicError guards the vm.recoverPanic shape check:
// a runtime.Error (here a nil deref via out-of-bounds index) must keep the
// capturePanic wrapping with mvm diagnostics, not slip through as a bare
// error like ExitError does.
func TestRuntimeErrorStillPanicError(t *testing.T) {
	i := newAutoImportInterp(t)
	_, err := i.Eval("oob", `var a = []int{1, 2}; _ = a[5]`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *vm.PanicError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *vm.PanicError, got %T: %v", err, err)
	}
}
