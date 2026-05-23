package main

// An interpreted error type implementing Error + Is(error) bool must have
// its custom Is dispatched when passed to the native errors.Is chain walk.
// At the boundary it is bridged via stdlib.BridgeErrorIs (the Error+Is
// composite); without that composite the single-method BridgeError swallows
// the Is method and errors.Is returns false.
import (
	"errors"
	"fmt"
	"io/fs"
)

type permErr struct{ msg string }

func (e permErr) Error() string        { return e.msg }
func (e permErr) Is(target error) bool { return target == fs.ErrPermission }

func main() {
	var err error = permErr{"denied"}
	fmt.Println("equals:", err == fs.ErrPermission)
	fmt.Println("is:", errors.Is(err, fs.ErrPermission))
	fmt.Println("is-other:", errors.Is(err, fs.ErrNotExist))
}

// Output:
// equals: false
// is: true
// is-other: false
