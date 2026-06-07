package interp_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mvm-sh/mvm/interp"
)

// TestGoroutinePanicSurfacesAsExit checks an unrecovered goroutine panic is
// surfaced (logged) and turned into a non-zero exit instead of being silently
// swallowed -- here while main is blocked on a channel the dead goroutine owned,
// which would otherwise deadlock.
func TestGoroutinePanicSurfacesAsExit(t *testing.T) {
	i := newAutoImportInterp(t)
	var stderr bytes.Buffer
	i.SetIO(nil, &bytes.Buffer{}, &stderr)

	_, err := i.Eval("gopanic", `
		done := make(chan bool)
		go func() { panic("boom") }()
		<-done
	`)

	var ee *interp.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *interp.ExitError, got %T: %v", err, err)
	}
	if ee.Code != 2 {
		t.Errorf("Code = %d, want 2", ee.Code)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Errorf("goroutine panic not surfaced on stderr:\n%s", stderr.String())
	}
}

// TestGoroutineClosureCaptureStackDepth guards a closure-capture stack
// under-reservation: on a goroutine's tight stack the worker overran mem and
// died before wg.Done(), hanging wg.Wait(). See compiler reserveDepth.
func TestGoroutineClosureCaptureStackDepth(t *testing.T) {
	i := newAutoImportInterp(t)
	r, err := i.Eval("gostack", `
		func pairs(n int) func(func(int, int) bool) {
			return func(yield func(int, int) bool) {
				for i := 0; i < n; i++ {
					if !yield(i, i*i) {
						return
					}
				}
			}
		}
		func work(n int) int {
			total := 0
			for k, v := range pairs(n) {
				total += k + v
			}
			return total
		}
		res := make([]int, 5)
		wg := sync.WaitGroup{}
		wg.Add(5)
		for i := 0; i < 5; i++ {
			go func(i int) {
				res[i] = work(i + 1)
				wg.Done()
			}(i)
		}
		wg.Wait()
		sum := 0
		for _, v := range res {
			sum += v
		}
		sum
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	// work(n) = sum_{k<n} (k + k*k): 0,2,8,20,40 for n=1..5 -> total 70.
	if got := fmt.Sprintf("%v", r); got != "70" {
		t.Errorf("got %q, want %q", got, "70")
	}
}
