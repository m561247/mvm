package vm

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

// Concurrent misses on one key must build once and hand every caller the same
// value; a plain get-then-set would let racing callers each build a distinct one.
func TestFuncWrapTableGetOrBuildSingleFlight(t *testing.T) {
	tbl := newFuncWrapTable()
	key := funcWrapKey{code: 7, rtype: reflect.TypeOf(func() {})}

	const n = 64
	var builds atomic.Int64
	build := func() reflect.Value {
		builds.Add(1)
		// A fresh pointer per build: distinct builds have distinct .Pointer()
		// (two identical func literals would share a code pointer, hiding dups).
		return reflect.ValueOf(new(int))
	}

	got := make([]reflect.Value, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range got {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			got[idx] = tbl.getOrBuild(key, build)
		}(i)
	}
	close(start)
	wg.Wait()

	if b := builds.Load(); b != 1 {
		t.Errorf("build ran %d times, want 1", b)
	}
	for i := 1; i < n; i++ {
		if got[i].Pointer() != got[0].Pointer() {
			t.Fatalf("caller %d got a different wrapper than caller 0", i)
		}
	}

	// A different key builds independently.
	other := tbl.getOrBuild(funcWrapKey{code: 8, rtype: key.rtype}, build)
	if other.Pointer() == got[0].Pointer() {
		t.Error("distinct keys returned the same wrapper")
	}
	if b := builds.Load(); b != 2 {
		t.Errorf("build ran %d times after second key, want 2", b)
	}
}
