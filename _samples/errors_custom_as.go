package main

// An interpreted error type implementing Error + As(any) bool must have its
// custom As dispatched by errors.As (replaced by errorsx.mvmAs). At the
// boundary it is bridged via stdlib.BridgeErrorAs (the Error+As composite);
// without that composite the single-method BridgeError has no As method and
// errors.As never reaches the interpreted body.
import (
	"errors"
	"fmt"
	"io/fs"
)

type asErr struct{ msg string }

func (e asErr) Error() string { return e.msg }

func (e asErr) As(target any) bool {
	pe, ok := target.(**fs.PathError)
	if !ok {
		return false
	}
	*pe = &fs.PathError{Op: "custom", Path: "/", Err: errors.New(e.msg)}
	return true
}

func main() {
	var err error = asErr{"an error"}
	var pe *fs.PathError
	fmt.Println("as:", errors.As(err, &pe))
	fmt.Println("pathError:", pe)
}

// Output:
// as: true
// pathError: custom /: an error
