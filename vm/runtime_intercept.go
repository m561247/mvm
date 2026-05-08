package vm

import (
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// RuntimeFuncInfo holds the synthesized Name/FileLine for a *runtime.Func
// sentinel allocated by the bridged runtime.FuncForPC. The sentinel is a
// fresh &runtime.Func{} pointer; the host runtime never sees it because
// nativeMethodLookup intercepts Name and FileLine before any host method
// runs.
type RuntimeFuncInfo struct {
	Name string
	File string
	Line int
}

var runtimeFuncMeta sync.Map // *runtime.Func -> *RuntimeFuncInfo

// activeMachine tracks the currently running Machine so that native
// bridges (e.g. the runtime.Callers replacement) can find it.
// Single-machine-at-a-time semantics: concurrent Machines on different
// goroutines will race on this slot. Run threads the previous value
// through its call chain via SetActiveMachine + defer-restore, which is
// correct for the synchronous nesting that makeCallFunc/CallFunc
// produce. Concurrent goroutine execution is no worse off than under
// the previous global-stack implementation, which had the same race.
var activeMachine atomic.Pointer[Machine]

// SetActiveMachine atomically replaces the current Machine and returns
// the previous value. Pair with `defer SetActiveMachine(prev)` to
// restore on return.
func SetActiveMachine(m *Machine) (prev *Machine) {
	return activeMachine.Swap(m)
}

// ActiveMachine returns the Machine currently set via SetActiveMachine,
// or nil if none.
func ActiveMachine() *Machine {
	return activeMachine.Load()
}

// runtimeFuncPtrType is *runtime.Func, used to detect intercepted receivers.
var runtimeFuncPtrType = reflect.TypeOf((*runtime.Func)(nil))

// runtimeFuncSentinel embeds runtime.Func together with padding so each
// allocation gets a unique address. runtime.Func itself is a zero-sized
// struct (opaque{} field), and Go reuses a single pointer for all
// zero-size allocations -- using it bare would collapse every
// registered frame onto the same key.
//
// The padding is sized at 24 bytes (> 16) so allocations bypass Go's
// tiny allocator. With a 1-byte struct the tiny allocator can pack
// consecutive sentinels exactly 1 byte apart, which makes
// mvmFuncForPC's `pc-1 / pc` two-step lookup alias adjacent sentinels:
// pcs[i+1]-1 == sentinel_i, so frame i+1 prints frame i's metadata.
// 24 bytes lands the struct in a regular 8-aligned size class, so
// distinct sentinels are at least 8 bytes apart.
type runtimeFuncSentinel struct {
	rf runtime.Func
	_  [24]byte
}

// NewRuntimeFuncSentinel returns a fresh *runtime.Func whose address is
// unique. Use it together with RegisterRuntimeFunc to mark a PC as
// virtualized.
func NewRuntimeFuncSentinel() *runtime.Func {
	s := &runtimeFuncSentinel{}
	return (*runtime.Func)(unsafe.Pointer(s))
}

// RegisterRuntimeFunc associates Name/File/Line metadata with rf so that
// interpreted code calling rf.Name() / rf.FileLine() observes the
// recorded values instead of the host runtime's lookup. rf must be a
// non-nil pointer obtained from NewRuntimeFuncSentinel so the address is
// distinct from any other registered sentinel.
func RegisterRuntimeFunc(rf *runtime.Func, name, file string, line int) {
	if rf == nil {
		return
	}
	runtimeFuncMeta.Store(rf, &RuntimeFuncInfo{Name: name, File: file, Line: line})
}

// LookupRuntimeFunc returns the registered metadata for rf, or nil if rf
// was not produced by the mvm bridge.
func LookupRuntimeFunc(rf *runtime.Func) *RuntimeFuncInfo {
	v, ok := runtimeFuncMeta.Load(rf)
	if !ok {
		return nil
	}
	return v.(*RuntimeFuncInfo)
}

// runtimeFuncShim returns a bound-method reflect.Value that satisfies
// (*runtime.Func).Name or (*runtime.Func).FileLine using the side-table
// entry for rv. Returns the zero reflect.Value if rv is not a tracked
// sentinel or name is not one of the intercepted methods.
func runtimeFuncShim(rv reflect.Value, name string) reflect.Value {
	if !rv.IsValid() || rv.Type() != runtimeFuncPtrType || rv.IsNil() {
		return reflect.Value{}
	}
	rf, ok := rv.Interface().(*runtime.Func)
	if !ok {
		return reflect.Value{}
	}
	info := LookupRuntimeFunc(rf)
	if info == nil {
		return reflect.Value{}
	}
	switch name {
	case "Name":
		return reflect.MakeFunc(reflect.TypeOf(func() string { return "" }),
			func(_ []reflect.Value) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(info.Name)}
			})
	case "FileLine":
		return reflect.MakeFunc(reflect.TypeOf(func(uintptr) (string, int) { return "", 0 }),
			func(_ []reflect.Value) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(info.File), reflect.ValueOf(info.Line)}
			})
	}
	return reflect.Value{}
}
