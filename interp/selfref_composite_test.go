package interp_test

import (
	"fmt"
	"testing"
)

// Self-referential named composites (type P *P / S []S / M map[int]M) must
// parse, materialize (donor layout + SetElem patch), and behave like gc.
// Was go-cmp cycleTests: "undefined: P".
func TestSelfRefNamedComposites(t *testing.T) {
	src := `
package main

import "fmt"

type (
	P *P
	S []S
	M map[int]M
)

func main() {
	x := new(P)
	*x = x
	fmt.Println(*x == x, **x == x)

	s := S{nil}
	s[0] = s
	fmt.Println(len(s[0][0][0]) == 1)

	m := M{0: nil}
	m[0] = m
	fmt.Println(len(m[0][0]) == 1)

	fmt.Printf("%T %T %T\n", x, s, m)
}
`
	i := newAutoImportInterp(t)
	if _, err := i.Eval("selfref", src); err != nil {
		t.Fatalf("eval: %v", err)
	}
}

// Two same-named func-local self-ref types must get distinct identities; a
// shared-carrier cache keyed on the donor layout collided them.
func TestSelfRefSameNameNoCollision(t *testing.T) {
	src := `
package main

import "fmt"

func f1() any {
	type S []S
	s := S{nil}
	s[0] = s
	return s
}

func f2() any {
	type S [][]S
	return S{}
}

func main() {
	a, b := f1(), f2()
	fmt.Printf("%T %T\n", a, b)
}
`
	i := newAutoImportInterp(t)
	if _, err := i.Eval("selfref_collision", src); err != nil {
		t.Fatalf("eval: %v", err)
	}
}

// Self-ref map with an array elem ([2]M is 2 ptr words; donor uses a
// same-shape stand-in). Was an internal "Type on zero Value" panic.
func TestSelfRefMapArrayElem(t *testing.T) {
	src := `
package main

import "fmt"

type M map[int][2]M

func main() {
	m := M{0: [2]M{nil, nil}}
	m[0] = [2]M{m, nil}
	if len(m) != 1 || len(m[0][0]) != 1 {
		panic("cycle broken")
	}
	fmt.Println("ok")
}
`
	i := newAutoImportInterp(t)
	if _, err := i.Eval("selfref_map_array", src); err != nil {
		t.Fatalf("eval: %v", err)
	}
}

// Equality across mixed rtypes must match gc: same-pointee pointers compare
// by address (P vs *P), different pointees at the same address stay inequal,
// and named vs unnamed struct/array with identical underlying compare equal.
func TestMixedRtypeEquality(t *testing.T) {
	cases := []struct{ n, src, res string }{
		{"selfref_ptr", `type P *P; f := func() bool { x := new(P); *x = x; return *x == x && **x == x }; fmt.Sprint(f())`, "true"},
		{"diff_pointee_same_addr", `type S struct{ F int; G string }; f := func() bool { s := S{1, "x"}; var a any = &s; var b any = &s.F; return a == b }; fmt.Sprint(f())`, "false"},
		{"named_vs_unnamed_array", `type A [2]int; a1 := A{1, 2}; a2 := [2]int{1, 2}; fmt.Sprint(a1 == a2)`, "true"},
		{"named_vs_unnamed_struct", `type T struct{ X int }; s1 := T{1}; s2 := struct{ X int }{1}; fmt.Sprint(s1 == s2)`, "true"},
		{"named_vs_unnamed_struct_ne", `type T struct{ X int }; s1 := T{1}; s2 := struct{ X int }{2}; fmt.Sprint(s1 == s2)`, "false"},
	}
	for _, c := range cases {
		t.Run(c.n, func(t *testing.T) {
			i := newAutoImportInterp(t)
			r, err := i.Eval(c.n, c.src)
			if err != nil {
				t.Fatalf("eval %q: %v", c.src, err)
			}
			if got := fmt.Sprintf("%v", r); got != c.res {
				t.Errorf("got %q, want %q", got, c.res)
			}
		})
	}
}

// Elided composite literals in map literals: the key needs &T{} addressing
// and the value must re-infer its own type ident ({k}: {v} shared a stale
// ctype). Was go-cmp StringerMapKey: compile-stack underflow.
func TestElidedMapComposites(t *testing.T) {
	cases := []struct{ n, src, res string }{
		{"key_and_value", `type T struct{ X string }; m := map[*T]*T{{"hello"}: {"world"}}; r := ""; for k, v := range m { r = k.X + v.X }; r`, "helloworld"},
		{"key_only", `type T struct{ X string }; m := map[*T]string{{"x"}: "a"}; r := ""; for k, v := range m { r = k.X + v }; r`, "xa"},
		{"value_only", `type T struct{ X string }; m := map[string]*T{"a": {"x"}}; m["a"].X`, "x"},
	}
	for _, c := range cases {
		t.Run(c.n, func(t *testing.T) {
			i := newAutoImportInterp(t)
			r, err := i.Eval(c.n, c.src)
			if err != nil {
				t.Fatalf("eval %q: %v", c.src, err)
			}
			if got := fmt.Sprintf("%v", r); got != c.res {
				t.Errorf("got %q, want %q", got, c.res)
			}
		})
	}
}

// A map write whose key was read from an unexported field carries flagRO;
// MapSet must strip it. Was go-cmp resolveReferences SetMapIndex panic.
func TestMapSetUnexportedKey(t *testing.T) {
	src := `
package main

import (
	"fmt"
	"reflect"
)

type ptr struct {
	p uintptr
	t reflect.Type
}

type leaf struct{ p ptr }

type node struct{ Metadata any }

func main() {
	v := 42
	n := &node{Metadata: leaf{p: ptr{p: reflect.ValueOf(&v).Pointer(), t: reflect.TypeOf(&v)}}}
	seen := make(map[ptr]bool)
	seen[n.Metadata.(leaf).p] = true
	fmt.Println(len(seen))
}
`
	i := newAutoImportInterp(t)
	if _, err := i.Eval("rokey", src); err != nil {
		t.Fatalf("eval: %v", err)
	}
}

// A synth *T boxed in an interface map key must be hashable: derived and
// reserved pointer rtypes carry tflagRegularMemory or runtime.typehash
// panics "hash of unhashable type" when the map crosses the bridge.
func TestSynthPtrIfaceKeyHash(t *testing.T) {
	src := `
package main

import (
	"fmt"
	"reflect"
)

type Stringer string

func (s Stringer) String() string { return string(s) }

func newStringer(s string) fmt.Stringer { return (*Stringer)(&s) }

func main() {
	y := map[interface{}]string{newStringer("hello"): "goodbye"}
	var i any = y
	fmt.Println(reflect.ValueOf(i).Len())
}
`
	i := newAutoImportInterp(t)
	if _, err := i.Eval("ptrhash", src); err != nil {
		t.Fatalf("eval: %v", err)
	}
}
