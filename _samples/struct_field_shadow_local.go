package main

// Struct composite-literal field name shadows an outer local of the same
// name used inside the value expression.

import (
	stderrors "errors"
	"fmt"
)

func main() {
	err := stderrors.New("test")
	a := struct {
		err    error
		target error
	}{err: fmt.Errorf("wrap: %s", err.Error()), target: err}
	fmt.Println("err:", a.err)
	fmt.Println("target:", a.target)
}

// Output:
// err: wrap: test
// target: test
