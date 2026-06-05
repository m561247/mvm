package interp_test

import "testing"

// The reflect.Value func-identity cache is keyed by {code address, rtype} on the
// persistent Machine; these guard the cross-Eval (REPL) invariants keying relies
// on: code addresses are monotonic across Evals, and a failed Eval's rollback
// never reuses an address that already cached a wrapper.

// A redefined func in a later Eval must not collapse onto the old func's wrapper.
func TestCrossEval_RedefineKeepsIdentity(t *testing.T) {
	i := newAutoImportInterp(t)
	if _, err := i.Eval("e0", `import "reflect"`); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := i.Eval("e1", `type cf func() int
var f cf = func() int { return 1 }
var rf = reflect.ValueOf(f)`); err != nil {
		t.Fatalf("e1: %v", err)
	}
	r, err := i.Eval("e2", `var g cf = func() int { return 2 }
reflect.ValueOf(g) == rf`)
	if err != nil {
		t.Fatalf("e2: %v", err)
	}
	if got := r.Interface(); got != false {
		t.Errorf("reflect.ValueOf(g) == rf: got %v, want false (distinct funcs collided)", got)
	}
	r2, err := i.Eval("e3", `rf.Call(nil)[0].Int()`)
	if err != nil {
		t.Fatalf("e3: %v", err)
	}
	if got := r2.Interface(); got != int64(1) {
		t.Errorf("rf.Call -> %v, want 1 (cached wrapper points at wrong func)", got)
	}
}

// A failed Eval between caching and reuse must not corrupt the cached wrapper.
func TestCrossEval_RollbackThenReuse(t *testing.T) {
	i := newAutoImportInterp(t)
	if _, err := i.Eval("e0", `import "reflect"`); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := i.Eval("e1", `type cf func() int
var f cf = func() int { return 11 }
var rf = reflect.ValueOf(f)`); err != nil {
		t.Fatalf("e1: %v", err)
	}
	if _, err := i.Eval("e2", `var bad = undefXYZ`); err == nil {
		t.Fatal("e2 expected to fail")
	}
	r, err := i.Eval("e3", `rf.Call(nil)[0].Int()`)
	if err != nil {
		t.Fatalf("e3: %v", err)
	}
	if got := r.Interface(); got != int64(11) {
		t.Errorf("rf.Call after rollback -> %v, want 11", got)
	}
}

// A cached func re-bridged and invoked in a later Eval, after globals grew and a
// referenced global was mutated, must read the new value (not a stale snapshot).
func TestCrossEval_GlobalsVisibleAfterGrowth(t *testing.T) {
	i := newAutoImportInterp(t)
	if _, err := i.Eval("e0", `import "reflect"`); err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := i.Eval("e1", `var base = 100
type cf func() int
var f cf = func() int { return base }
var rf = reflect.ValueOf(f)`); err != nil {
		t.Fatalf("e1: %v", err)
	}
	r, err := i.Eval("e2", `var a, b, c, d, e2v, ff, gg, hh int = 1, 2, 3, 4, 5, 6, 7, 8
_ = a + b + c + d + e2v + ff + gg + hh
base = 999
reflect.ValueOf(f).Call(nil)[0].Int()`)
	if err != nil {
		t.Fatalf("e2: %v", err)
	}
	if got := r.Interface(); got != int64(999) {
		t.Errorf("re-bridged f after growth -> %v, want 999 (stale globals)", got)
	}
}
