package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// Two functions each declare a local type named Person; both are passed to the
// same generic function. The types share a PkgPath.Name (main.Person), so the
// generic instance used to be cached under one mangled name, binding the first
// instantiation's rtype to the body's swap temps. The second call then tripped
// `reflect.Set: value of type main.Person is not assignable to type
// main.Person`. Each distinct declaration must get its own monomorphization.
func TestGenericLocalTypeCollision(t *testing.T) {
	src := `package main

import "fmt"

func swapFirst[E any](data []E) {
	data[0], data[1] = data[1], data[0]
}

func first() {
	type Person struct {
		Name string
		Age  int
	}
	p := []Person{{"A", 1}, {"B", 2}}
	swapFirst(p)
	fmt.Println(p)
}

func second() {
	type Person struct {
		Name string
		Age  int
	}
	p := []Person{{"C", 3}, {"D", 4}}
	swapFirst(p)
	fmt.Println(p)
}

func main() {
	first()
	second()
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("generic_local.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "[{B 2} {A 1}]\n[{D 4} {C 3}]\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}
