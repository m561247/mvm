package synth

import (
	"reflect"
	"unsafe"

	// Required because of go:linkname directives below.
	_ "unsafe"
)

// addReflectOff registers a pointer into the runtime's reflect-offset table
// and returns the corresponding NameOff / TypeOff / TextOff. Reflect uses
// this internally to make StructOf-built rtypes participate in the same
// off-resolution machinery as compiler-emitted rtypes.
//
// reflect.addReflectOff is tagged with `//go:linkname addReflectOff` at the
// definition site (see reflect/type.go), permitting external linkname access
// without -checklinkname=0 on Go 1.23+.
//
//go:linkname addReflectOff reflect.addReflectOff
//go:noescape
func addReflectOff(ptr unsafe.Pointer) int32

// rtypePtr extracts the *abiType from a reflect.Type interface value.
// reflect.Type is a non-empty interface with header layout (itab, data); the
// data word is a *rtype pointer which is identical in layout to *abiType.
//
// Returns nil if t is nil.
//
//go:nosplit
func rtypePtr(t reflect.Type) *abiType {
	if t == nil {
		return nil
	}
	return (*abiType)((*[2]unsafe.Pointer)(unsafe.Pointer(&t))[1])
}

// asReflectType wraps a *abiType pointer as a reflect.Type interface value
// without going through reflect.toType (which would require a separate
// linkname). We borrow a stable rtype itab from a sample reflect.TypeOf
// call, then patch the data word.
//
// The sample is built once per Go process and reused; the itab pointer is
// stable for the life of the process.
//
//go:nosplit
func asReflectType(t *abiType) reflect.Type {
	if t == nil {
		return nil
	}
	out := sampleReflectType // copy carries the rtype itab
	(*[2]unsafe.Pointer)(unsafe.Pointer(&out))[1] = unsafe.Pointer(t)
	return out
}

// sampleReflectType is a reflect.Type whose itab is the canonical
// (*rtype, reflect.Type) itab. We swap its data word in asReflectType.
var sampleReflectType reflect.Type = reflect.TypeOf(struct{}{})
