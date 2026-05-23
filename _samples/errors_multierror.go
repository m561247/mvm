package main

// Multi-error (Unwrap() []error) bridging is NOT supported. An interpreted
// type whose Unwrap returns []error should let errors.Is walk every branch,
// but bridge selection (vm.wrapIface) keys on method NAME only: "Unwrap"
// collides with the single-error Unwrap() error bridge, and a single Go
// struct cannot declare both signatures. The Error+Unwrap composite wires
// its func() error field to a method returning []error, so the chain walk
// panics ("reflect: call of reflect.Value.Elem on slice Value"). Supporting
// it needs signature-aware bridge selection (a deeper VM change), and even
// then errors/wrap_test.go + join_test.go stay blocked on separate gaps.
import (
	"errors"
	"fmt"
	"io/fs"
)

type multiErr []error

func (m multiErr) Error() string   { return "multi" }
func (m multiErr) Unwrap() []error { return []error(m) }

func main() {
	leaf := fmt.Errorf("wrap: %w", fs.ErrPermission)
	var err error = multiErr{errors.New("other"), leaf}
	fmt.Println("is:", errors.Is(err, fs.ErrPermission))
}

// skip: multierror Unwrap() []error not bridged; selection keys on method
// name so it collides with Unwrap() error and panics. Needs signature-aware
// selection.
// Output:
// is: true
