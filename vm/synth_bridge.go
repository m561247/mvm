package vm

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mvm-sh/mvm/vm/synth"
)

// AttachSynthMethods installs t's interpreted methods on a fresh synthesized
// rtype via vm/synth and replaces t.Rtype.
// Native code that asserts the new rtype to an interface (fmt.Stringer,
// error, etc.) then dispatches the method directly, with no bridge proxy.
//
// Phase 1b: only shape S1 (func() string -- String/Error/GoString) on struct
// kinds.
// No-op for other shapes, other kinds, or when synth.Enabled() is false.
// Installs only the first matching method per call; multi-method support
// lands in Phase 2d.
//
// Re-allocation of existing values is out of scope: global slots populated
// before this call keep their old rtype.
// New values allocated via vm.NewValue against t.Rtype after this call see
// the synth rtype.
func (m *Machine) AttachSynthMethods(t *Type) error {
	if !synth.Enabled() || t == nil || t.Rtype == nil {
		return nil
	}
	if t.Rtype.Kind() != reflect.Struct {
		return nil
	}

	name, method, ok := m.firstS1Method(t)
	if !ok {
		return nil
	}

	rtype := t.Rtype
	ifcType := t
	methodSig := method.Rtype // func() string, no receiver

	handler := func(recv unsafe.Pointer) string {
		rv := reflect.NewAt(rtype, recv).Elem()
		ifc := Iface{Typ: ifcType, Val: FromReflect(rv)}
		fval := m.MakeMethodCallable(ifc, method)
		out, err := m.CallFunc(fval, methodSig, nil)
		if err != nil {
			return fmt.Sprintf("<synth dispatch error: %v>", err)
		}
		if len(out) != 1 {
			return ""
		}
		return out[0].String()
	}

	newRT, err := synth.AttachStructMethods(t.Rtype, t.PkgPath, synth.Method{
		Name:     name,
		Exported: true,
		Sig:      methodSig,
		Handler:  handler,
	})
	if err != nil {
		return err
	}
	t.Rtype = newRT
	return nil
}

// firstS1Method returns the first resolved method on t whose signature
// matches shape S1 (no args beyond receiver, single string return).
// Name filtering is intentionally absent: which method names matter is a
// stdlib-layer concern, not a vm concern.
// Any S1-shaped method becomes available to satisfy any interface that
// requires it (fmt.Stringer, error, fmt.GoStringer, user-defined).
func (m *Machine) firstS1Method(t *Type) (name string, mh Method, ok bool) {
	for i, method := range t.Methods {
		if !method.IsResolved() || i >= len(m.MethodNames) {
			continue
		}
		if method.Rtype == nil ||
			method.Rtype.NumIn() != 0 ||
			method.Rtype.NumOut() != 1 ||
			method.Rtype.Out(0).Kind() != reflect.String {
			continue
		}
		return m.MethodNames[i], method, true
	}
	return "", Method{}, false
}
