package interp

import (
	"github.com/mvm-sh/mvm/symbol"
	"github.com/mvm-sh/mvm/vm/synth"
)

// attachSynthMethods walks every compiled type and asks the machine to
// install a synthesized rtype carrying that type's methods.
// No-op when synth.Enabled() is false (the default).
// See [[project_synth_rtype_poc]] for the design.
func (i *Interp) attachSynthMethods() error {
	if !synth.Enabled() {
		return nil
	}
	for _, sym := range i.Symbols {
		if sym.Kind != symbol.Type || sym.Type == nil {
			continue
		}
		if err := i.AttachSynthMethods(sym.Type); err != nil {
			return err
		}
	}
	return nil
}
