package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// &funcvar used to yield *interface{} (func slots are interface{} boxes),
// breaking reflect.MakeFunc (fn.Type() reported interface) and *p=f through a
// *func. AddrLocal now retypes the slot to its func type.
func TestAddrFuncVar(t *testing.T) {
	src := `package main

import (
	"fmt"
	"reflect"
)

func main() {
	// reflect.MakeFunc round-trip through a *func target.
	swap := func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{in[1], in[0]}
	}
	var intSwap func(int, int) (int, int)
	fn := reflect.ValueOf(&intSwap).Elem()
	fmt.Println(fn.Type().Kind())
	intSwap = reflect.MakeFunc(fn.Type(), swap).Interface().(func(int, int) (int, int))
	a, b := intSwap(1, 2)
	fmt.Println(a, b)

	// &f reports the declared func type, not *interface{}.
	var f func(int) int
	fmt.Printf("%T\n", &f)

	// Assignment through a *func pointer dispatches to the new closure, and a
	// captured closure stays callable after its address is taken.
	base := 10
	g := func(x int) int { return x + base }
	p := &g
	*p = func(x int) int { return x - base }
	fmt.Println(g(1))
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.ImportPackageConsts(stdlib.ConstValues)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("addr_func.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	want := "func\n2 1\n*func(int) int\n-9\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}
