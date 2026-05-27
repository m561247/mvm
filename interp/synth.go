package interp

import (
	"github.com/mvm-sh/mvm/symbol"
	"github.com/mvm-sh/mvm/vm"
	"github.com/mvm-sh/mvm/vm/synth"
)

// attachSynthMethods walks every compiled type and asks the machine to
// install a synthesized rtype carrying that type's methods.
// No-op when synth.Enabled() is false (the default; opt in via MVM_SYNTH=1).
//
// Idempotency: each *vm.Type is attached at most once per Interp lifetime.
// The compiler aliases symbols under both bare and pkg-qualified keys
// (compiler.go:136), so the same *Type is reached twice per walk; and
// re-entrant Eval (test_cmd's package + _testmain) walks again.
// Without dedup the S1 stub pool exhausts after ~64 packages.
//
// vm.Type.RefreshRtype propagates the swap through t.derived, and
// Compiler.RefreshSynthRtype (called once after every type is attached)
// re-emits the c.Data slots whose stored rtype no longer matches the
// post-cascade Rtype -- Fnew sources, type descriptors, var storage.
// See [[project_synth_rtype_poc]] and [[project_symbolic_types_refactor]].
func (i *Interp) attachSynthMethods() error {
	if !synth.Enabled() {
		return nil
	}
	if i.synthAttached == nil {
		i.synthAttached = map[*vm.Type]bool{}
	}
	attached := false
	for _, sym := range i.Symbols {
		if sym.Kind != symbol.Type || sym.Type == nil {
			continue
		}
		if i.synthAttached[sym.Type] {
			continue
		}
		if err := i.AttachSynthMethods(sym.Type); err != nil {
			return err
		}
		i.synthAttached[sym.Type] = true
		attached = true
	}
	if attached {
		i.RebuildSynthStructRtypes()
		i.RefreshSynthRtype()
	}
	return nil
}
