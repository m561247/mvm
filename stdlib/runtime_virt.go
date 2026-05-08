package stdlib

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/mvm-sh/mvm/vm"
)

// frameKey is the dedup key for sentinelByFrame: the bytecode position
// IP plus the source position Pos uniquely identify a call site within
// a single program. Two calls to runtime.Callers from the same line
// reuse the same sentinel, which bounds runtimeFuncMeta growth at the
// number of distinct interpreted call sites rather than the number of
// stack captures.
type frameKey struct {
	IP  int
	Pos uint32
}

var sentinelByFrame sync.Map // frameKey -> *runtime.Func

// init registers a PackagePatcher for "runtime" so that interpreted code
// calling runtime.Callers / runtime.FuncForPC observes the interpreter's
// own call stack instead of the host Go stack frames inside vm.Machine.
//
// The replacement Callers walks m's frame-pointer chain and produces one
// synthetic uintptr per yielded interpreter frame. Each synthetic PC is
// the address of a freshly allocated *runtime.Func sentinel registered
// with vm.RegisterRuntimeFunc, so vm.nativeMethodLookup intercepts the
// later Name()/FileLine() calls on the sentinel and returns the recorded
// metadata. PCs that do not match a sentinel fall through to the host
// runtime.FuncForPC, preserving behavior for code that captured real
// host PCs.
func init() {
	RegisterPackagePatcher("runtime", patchRuntime)
}

func patchRuntime(_ *vm.Machine, values map[string]vm.Value) {
	values["Callers"] = vm.FromReflect(reflect.ValueOf(func(skip int, pcs []uintptr) int {
		// makeCallFunc spawns a fresh Machine for native callbacks, so the
		// Machine passed to the patcher is not necessarily the one running
		// when this bridge fires. Resolve the live one through the
		// active-machine stack.
		active := vm.ActiveMachine()
		if active == nil {
			return 0
		}
		return mvmCallers(active, skip, pcs)
	}))
	values["FuncForPC"] = vm.FromReflect(reflect.ValueOf(mvmFuncForPC))
}

// mvmCallers fills pcs with synthetic PCs for the live interpreter stack.
// The skip semantics mirror runtime.Callers: skip=0 identifies Callers
// itself. mvm has no Callers vm-frame, so we drop the first (skip-1)
// interpreter frames before recording.
func mvmCallers(m *vm.Machine, skip int, pcs []uintptr) int {
	if len(pcs) == 0 {
		return 0
	}
	di := m.DebugInfo()
	if di == nil {
		return 0
	}
	drop := skip - 1
	if drop < 0 {
		drop = 0
	}
	n := 0
	m.WalkCallStack(func(f vm.StackFrame) bool {
		if drop > 0 {
			drop--
			return true
		}
		if n >= len(pcs) {
			return false
		}
		rf := internSentinel(di, f)
		// pkg/errors' Frame.pc() does (uintptr(f) - 1) so we add 1.
		pcs[n] = uintptr(unsafe.Pointer(rf)) + 1
		n++
		return true
	})
	return n
}

// internSentinel returns a *runtime.Func sentinel for the given frame,
// reusing a previously allocated one when the (IP, Pos) call site has
// been seen before. First-encounter sentinels are registered with
// vm.RegisterRuntimeFunc. The intern cache bounds runtimeFuncMeta size
// at O(distinct call sites) instead of O(stack captures).
func internSentinel(di *vm.DebugInfo, f vm.StackFrame) *runtime.Func {
	key := frameKey{IP: f.IP, Pos: uint32(f.Pos)} //nolint:gosec
	if v, ok := sentinelByFrame.Load(key); ok {
		return v.(*runtime.Func)
	}
	file, line, _ := di.Sources.Resolve(int(f.Pos))
	name := qualifyFuncName(di.FuncAt(f.IP), file)
	if file == "" {
		file = "?"
		line = 0
	} else {
		file = "modfs/" + file
	}
	rf := vm.NewRuntimeFuncSentinel()
	vm.RegisterRuntimeFunc(rf, name, file, line)
	actual, loaded := sentinelByFrame.LoadOrStore(key, rf)
	if loaded {
		// Lost the race: another goroutine interned a sentinel for the
		// same key. The metadata is identical so it doesn't matter
		// which one wins; just drop ours and use theirs. The orphaned
		// sentinel stays in runtimeFuncMeta -- a small bounded leak
		// proportional to race count, not capture count.
		return actual.(*runtime.Func)
	}
	return rf
}

// mvmFuncForPC accepts either a sentinel pc produced by mvmCallers or a
// real host pc. For sentinels it returns the registered *runtime.Func;
// otherwise it delegates to runtime.FuncForPC so non-mvm uses still work.
func mvmFuncForPC(pc uintptr) *runtime.Func {
	if pc == 0 {
		return runtime.FuncForPC(pc)
	}
	// pkg/errors stores PCs as Frame(pc+1) and queries via Frame(pc).pc()
	// which subtracts 1, so the sentinel is reachable at pc-1. Try pc as
	// well in case a caller skipped the +1 convention.
	candidate := (*runtime.Func)(unsafe.Pointer(pc - 1)) //nolint:govet,gosec
	if vm.LookupRuntimeFunc(candidate) != nil {
		return candidate
	}
	candidate = (*runtime.Func)(unsafe.Pointer(pc)) //nolint:govet,gosec
	if vm.LookupRuntimeFunc(candidate) != nil {
		return candidate
	}
	return runtime.FuncForPC(pc)
}

// qualifyFuncName turns a debug-info label such as "TestFormatNew" into
// "github.com/pkg/errors.TestFormatNew" using the import-path prefix of
// the function's source file. file has the form "<pkgPath>/<filename>"
// (set by goparser's source registry). Labels that are already qualified
// (containing '.') are returned unchanged.
func qualifyFuncName(label, file string) string {
	if label == "" {
		return "?"
	}
	if strings.ContainsRune(label, '.') {
		return label
	}
	short := label
	if i := strings.LastIndexByte(label, '/'); i >= 0 {
		short = label[i+1:]
	}
	pkgPath, _ := splitPathFile(file)
	if pkgPath == "" {
		return label
	}
	return pkgPath + "." + short
}

func splitPathFile(file string) (dir, name string) {
	i := strings.LastIndexByte(file, '/')
	if i < 0 {
		return "", file
	}
	return file[:i], file[i+1:]
}
