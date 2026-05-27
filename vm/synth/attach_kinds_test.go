package synth

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

// stringerSig is the reflect.Type for func() string (shape S1 without recv).
var stringerSig = reflect.TypeOf((func() string)(nil))

func stringerT() reflect.Type {
	return reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
}

// stubHandler returns a HandlerS1 that records the call and returns out.
func stubHandler(called *bool, out string) HandlerS1 {
	return func(recv unsafe.Pointer) string {
		_ = recv
		*called = true
		return out
	}
}

func TestAttachPrimitiveMethodsInt(t *testing.T) {
	called := false
	rt, err := AttachPrimitiveMethods(reflect.TypeOf(int(0)),
		"MyInt", "test", Method{
			Name: "String", Exported: true, Sig: stringerSig,
			Handler: stubHandler(&called, "myint"),
		})
	if err != nil {
		t.Fatalf("AttachPrimitiveMethods: %v", err)
	}
	if got, want := rt.Kind(), reflect.Int; got != want {
		t.Errorf("Kind = %v, want %v", got, want)
	}
	if got, want := rt.NumMethod(), 1; got != want {
		t.Errorf("NumMethod = %d, want %d", got, want)
	}
	if !rt.Implements(stringerT()) {
		t.Fatal("rt.Implements(fmt.Stringer) = false")
	}

	v := reflect.New(rt).Elem()
	v.SetInt(42)
	s, ok := v.Interface().(fmt.Stringer)
	if !ok {
		t.Fatal("Interface().(fmt.Stringer) failed")
	}
	if got, want := s.String(), "myint"; got != want {
		t.Errorf("s.String() = %q, want %q", got, want)
	}
	if !called {
		t.Error("handler not invoked")
	}
}

func TestAttachPrimitiveMethodsString(t *testing.T) {
	called := false
	rt, err := AttachPrimitiveMethods(reflect.TypeOf(""),
		"MyStr", "test", Method{
			Name: "String", Exported: true, Sig: stringerSig,
			Handler: stubHandler(&called, "mystr"),
		})
	if err != nil {
		t.Fatalf("AttachPrimitiveMethods: %v", err)
	}
	if !rt.Implements(stringerT()) {
		t.Fatal("not Stringer")
	}
	v := reflect.New(rt).Elem()
	v.SetString("hi")
	s := v.Interface().(fmt.Stringer)
	if got, want := s.String(), "mystr"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !called {
		t.Error("handler not invoked")
	}
}

func TestAttachPrimitiveMethodsRejectsStruct(t *testing.T) {
	_, err := AttachPrimitiveMethods(reflect.TypeOf(struct{}{}),
		"X", "test", Method{Sig: stringerSig, Handler: stubHandler(new(bool), "")})
	if err == nil {
		t.Fatal("expected error for non-primitive kind")
	}
}

func TestAttachSliceMethods(t *testing.T) {
	called := false
	rt, err := AttachSliceMethods(reflect.TypeOf([]int(nil)),
		"MySlice", "test", Method{
			Name: "String", Exported: true, Sig: stringerSig,
			Handler: stubHandler(&called, "myslice"),
		})
	if err != nil {
		t.Fatalf("AttachSliceMethods: %v", err)
	}
	if got, want := rt.Kind(), reflect.Slice; got != want {
		t.Errorf("Kind = %v, want %v", got, want)
	}
	if rt.Elem() != reflect.TypeOf(int(0)) {
		t.Errorf("Elem = %v, want int", rt.Elem())
	}
	if !rt.Implements(stringerT()) {
		t.Fatal("not Stringer")
	}
	v := reflect.MakeSlice(rt, 3, 3)
	v.Index(0).SetInt(1)
	s := v.Interface().(fmt.Stringer)
	if got, want := s.String(), "myslice"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !called {
		t.Error("handler not invoked")
	}
}

func TestAttachArrayMethods(t *testing.T) {
	called := false
	layout := reflect.ArrayOf(4, reflect.TypeOf(int(0)))
	rt, err := AttachArrayMethods(layout, "MyArr", "test", Method{
		Name: "String", Exported: true, Sig: stringerSig,
		Handler: stubHandler(&called, "myarr"),
	})
	if err != nil {
		t.Fatalf("AttachArrayMethods: %v", err)
	}
	if got, want := rt.Kind(), reflect.Array; got != want {
		t.Errorf("Kind = %v, want %v", got, want)
	}
	if rt.Len() != 4 {
		t.Errorf("Len = %d, want 4", rt.Len())
	}
	if !rt.Implements(stringerT()) {
		t.Fatal("not Stringer")
	}
	v := reflect.New(rt).Elem()
	v.Index(0).SetInt(7)
	s := v.Interface().(fmt.Stringer)
	if got, want := s.String(), "myarr"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !called {
		t.Error("handler not invoked")
	}
}

func TestAttachMapMethods(t *testing.T) {
	called := false
	layout := reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(int(0)))
	rt, err := AttachMapMethods(layout, "MyMap", "test", Method{
		Name: "String", Exported: true, Sig: stringerSig,
		Handler: stubHandler(&called, "mymap"),
	})
	if err != nil {
		t.Fatalf("AttachMapMethods: %v", err)
	}
	if got, want := rt.Kind(), reflect.Map; got != want {
		t.Errorf("Kind = %v, want %v", got, want)
	}
	if rt.Key() != reflect.TypeOf("") {
		t.Errorf("Key = %v, want string", rt.Key())
	}
	if !rt.Implements(stringerT()) {
		t.Fatal("not Stringer")
	}
	v := reflect.MakeMap(rt)
	v.SetMapIndex(reflect.ValueOf("a"), reflect.ValueOf(1))
	s := v.Interface().(fmt.Stringer)
	if got, want := s.String(), "mymap"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !called {
		t.Error("handler not invoked")
	}
}
