package interp_test

import (
	"fmt"
	"testing"
)

// TestTypeSwitchConcurrency guards the type-switch guard temp being a frame-local,
// not a shared global slot that concurrent goroutines clobber. Each goroutine owns
// its *box; an out-of-sequence read means the switch dispatched on the wrong receiver.
func TestTypeSwitchConcurrency(t *testing.T) {
	const src = `
type reader interface{ read() int }
type box struct{ v int }
func (b *box) read() int { b.v++; return b.v }

func dispatch(e interface{}) int {
	switch r := e.(type) {
	case reader:
		return r.read()
	}
	return -1
}

func run() int {
	done := make(chan int, 8)
	for g := 0; g < 8; g++ {
		go func() {
			var e reader = &box{}
			bad, prev := 0, 0
			for i := 0; i < 20000; i++ {
				n := dispatch(e)
				if n != prev+1 {
					bad++
				}
				prev = n
			}
			done <- bad
		}()
	}
	total := 0
	for i := 0; i < 8; i++ {
		bad := <-done
		total += bad
	}
	return total
}
run()`
	i := newAutoImportInterp(t)
	r, err := i.Eval("test", src)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := fmt.Sprintf("%v", r); got != "0" {
		t.Errorf("concurrent type-switch corruption: %s out-of-sequence reads, want 0", got)
	}
}
