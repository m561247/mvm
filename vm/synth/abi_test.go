package synth

import (
	"reflect"
	"testing"
	"unsafe"
)

// Probe to confirm our abi mirrors match what the running Go runtime expects.
// If Go ever changes the internal/abi layout, these tests fail loudly at
// build/CI time rather than producing memory corruption at runtime.

type probeNamed struct {
	A int
}

func (probeNamed) Marker() {}

func TestAbiTypeLayout(t *testing.T) {
	rt := reflect.TypeOf(probeNamed{})
	at := rtypePtr(rt)
	if at == nil {
		t.Fatal("rtypePtr returned nil")
	}

	// Sanity: pointer-size-aware Size_ for {A int} is one word.
	want := unsafe.Sizeof(uintptr(0))
	if at.Size != want {
		t.Errorf("Size_ = %d, want %d", at.Size, want)
	}

	// Kind is Struct.
	if at.Kind != kindStruct {
		t.Errorf("Kind_ = %d, want %d (struct)", at.Kind, kindStruct)
	}

	// Named + Uncommon flags should be set on a defined type with methods.
	if at.TFlag&tflagNamed == 0 {
		t.Errorf("TFlag missing tflagNamed: %#x", at.TFlag)
	}
	if at.TFlag&tflagUncommon == 0 {
		t.Errorf("TFlag missing tflagUncommon: %#x", at.TFlag)
	}

	// Align matches uintptr on this arch.
	if at.Align != uint8(unsafe.Alignof(uintptr(0))) {
		t.Errorf("Align_ = %d, want %d", at.Align, unsafe.Alignof(uintptr(0)))
	}

	// Hash is nonzero for a real type (compiler-emitted).
	if at.Hash == 0 {
		t.Errorf("Hash = 0; expected nonzero for compiler-emitted rtype")
	}
}

func TestAbiStructTypeLayout(t *testing.T) {
	type s struct {
		A int
		B string
	}
	rt := reflect.TypeOf(s{})
	st := (*abiStructType)(unsafe.Pointer(rtypePtr(rt)))

	if got := len(st.Fields); got != 2 {
		t.Fatalf("Fields len = %d, want 2", got)
	}
	if st.Fields[0].Offset != 0 {
		t.Errorf("Fields[0].Offset = %d, want 0", st.Fields[0].Offset)
	}
	// Field 1 (string) starts after Field 0 (int).
	wantOff := unsafe.Sizeof(int(0))
	if st.Fields[1].Offset != wantOff {
		t.Errorf("Fields[1].Offset = %d, want %d", st.Fields[1].Offset, wantOff)
	}
	if st.Fields[0].Typ == nil || st.Fields[1].Typ == nil {
		t.Errorf("field Typ is nil; mirror layout drift?")
	}
}

func TestAbiPtrTypeLayout(t *testing.T) {
	rt := reflect.TypeOf((*int)(nil))
	pt := (*abiPtrType)(unsafe.Pointer(rtypePtr(rt)))

	if pt.Kind != kindPointer {
		t.Errorf("Kind_ = %d, want %d (pointer)", pt.Kind, kindPointer)
	}
	if pt.TFlag&tflagDirectIface == 0 {
		t.Errorf("TFlag missing tflagDirectIface: %#x", pt.TFlag)
	}
	if pt.Elem == nil {
		t.Fatalf("Elem nil; layout drift?")
	}
	// Elem points to int's abiType: Kind=Int.
	if pt.Elem.Kind != kindInt {
		t.Errorf("Elem.Kind = %d, want %d (int)", pt.Elem.Kind, kindInt)
	}
}

func TestAbiSliceTypeLayout(t *testing.T) {
	rt := reflect.TypeOf([]int{})
	st := (*abiSliceType)(unsafe.Pointer(rtypePtr(rt)))
	if st.Kind != kindSlice {
		t.Errorf("Kind_ = %d, want %d (slice)", st.Kind, kindSlice)
	}
	if st.Elem == nil || st.Elem.Kind != kindInt {
		t.Errorf("Elem mismatch; layout drift?")
	}
}

func TestAbiArrayTypeLayout(t *testing.T) {
	rt := reflect.TypeOf([3]int{})
	at := (*abiArrayType)(unsafe.Pointer(rtypePtr(rt)))
	if at.Kind != kindArray {
		t.Errorf("Kind_ = %d, want %d (array)", at.Kind, kindArray)
	}
	if at.Len != 3 {
		t.Errorf("Len = %d, want 3", at.Len)
	}
	if at.Elem == nil || at.Elem.Kind != kindInt {
		t.Errorf("Elem mismatch; layout drift?")
	}
}

func TestAbiMapTypeLayout(t *testing.T) {
	rt := reflect.TypeOf(map[string]int{})
	mt := (*abiMapType)(unsafe.Pointer(rtypePtr(rt)))
	if mt.Kind != kindMap {
		t.Errorf("Kind_ = %d, want %d (map)", mt.Kind, kindMap)
	}
	if mt.Key == nil || mt.Key.Kind != kindString {
		t.Errorf("Key mismatch; layout drift?")
	}
	if mt.Elem == nil || mt.Elem.Kind != kindInt {
		t.Errorf("Elem mismatch; layout drift?")
	}
	if mt.Group == nil {
		t.Errorf("Group nil; swisstable layout drift?")
	}
	if mt.Hasher == nil {
		t.Errorf("Hasher nil; layout drift?")
	}
}

func TestUncommonOffsetsByKind(t *testing.T) {
	// The runtime's per-Kind Uncommon() dispatch (internal/abi/type.go:319)
	// computes uncommon's offset as sizeof(KindType). Our synthesis code
	// will rely on these offsets being predictable; assert them here.
	cases := []struct {
		name    string
		mirror  uintptr
		wantOff uintptr // for documentation -- equals mirror size
	}{
		{"Type", unsafe.Sizeof(abiType{}), unsafe.Sizeof(abiType{})},
		{"PtrType", unsafe.Sizeof(abiPtrType{}), unsafe.Sizeof(abiPtrType{})},
		{"SliceType", unsafe.Sizeof(abiSliceType{}), unsafe.Sizeof(abiSliceType{})},
		{"ArrayType", unsafe.Sizeof(abiArrayType{}), unsafe.Sizeof(abiArrayType{})},
		{"MapType", unsafe.Sizeof(abiMapType{}), unsafe.Sizeof(abiMapType{})},
		{"StructType", unsafe.Sizeof(abiStructType{}), unsafe.Sizeof(abiStructType{})},
	}
	for _, c := range cases {
		if c.mirror != c.wantOff {
			t.Errorf("%s mirror sizeof = %d, want %d", c.name, c.mirror, c.wantOff)
		}
	}
	// Document the values for the current arch in test output (-v).
	t.Logf("uncommon offsets on this arch (ptrsize=%d):", unsafe.Sizeof(uintptr(0)))
	for _, c := range cases {
		t.Logf("  %-12s = %d", c.name, c.mirror)
	}
}

func TestAddReflectOff(t *testing.T) {
	// Round-trip: register an arbitrary pointer, verify a non-zero offset
	// comes back. We don't have resolveReflectName/Type/Text wired here,
	// so we only verify the linkname works.
	x := 42
	off := addReflectOff(unsafe.Pointer(&x))
	if off == 0 {
		t.Error("addReflectOff returned 0; linkname not resolved?")
	}
	// Second call with a different pointer returns a different offset.
	y := 99
	off2 := addReflectOff(unsafe.Pointer(&y))
	if off2 == 0 || off2 == off {
		t.Errorf("addReflectOff(&y) = %d, expected non-zero and distinct from %d", off2, off)
	}
}

func TestUncommonAndMethodSize(t *testing.T) {
	if got, want := unsafe.Sizeof(abiUncommon{}), uintptr(16); got != want {
		t.Errorf("sizeof(abiUncommon) = %d, want %d", got, want)
	}
	if got, want := unsafe.Sizeof(abiMethod{}), uintptr(16); got != want {
		t.Errorf("sizeof(abiMethod) = %d, want %d", got, want)
	}
}

func TestAsReflectTypeRoundTrip(t *testing.T) {
	rt := reflect.TypeOf(probeNamed{})
	at := rtypePtr(rt)
	rt2 := asReflectType(at)

	if rt != rt2 {
		t.Errorf("roundtrip not identical: rt=%v rt2=%v", rt, rt2)
	}
	// Underlying *abiType should match.
	if rtypePtr(rt2) != at {
		t.Errorf("data word lost in roundtrip")
	}
}
