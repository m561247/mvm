package interp_test

import (
	"fmt"
	"testing"
)

// TestCrossUnitFuncValueAddress guards code/address alignment across successive
// top-level Evals. Each Eval leaves its init/main call shims in the VM code;
// those shims are not in the compiler's Code, so a later Eval's code must be
// trimmed back into alignment before being pushed. Otherwise a function defined
// in the later unit and called by VALUE (its stored compiler-code offset) lands
// at the wrong VM address -- exactly how `mvm test` ran external-package
// examples (referenced as testing.InternalExample.F) after the internal unit's
// init shims shifted m.code. See interp.evalCompiled / vm.Machine.TrimCode.
func TestCrossUnitFuncValueAddress(t *testing.T) {
	i := newAutoImportInterp(t)
	// Unit 1: an init func leaves init-call shims in m.code after the run.
	if _, err := i.Eval("unit1", `var inited int; func init() { inited = 7 }; func a() int { return inited }`); err != nil {
		t.Fatalf("unit1: %v", err)
	}
	// Unit 2: define a func, take its value, and call through it. With the
	// shims left dangling this called the wrong address and crashed.
	r, err := i.Eval("unit2", `func b(x int) int { return x*x + 1 }; fn := b; fn(6)`)
	if err != nil {
		t.Fatalf("unit2: %v", err)
	}
	if got := fmt.Sprintf("%v", r); got != "37" {
		t.Errorf("fn(6) = %q, want 37", got)
	}
}
