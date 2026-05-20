package main

import (
	"fmt"
	"strings"
)

// A panic raised by a native bridged method (strings.Builder.Grow on a
// negative count) must cross the native call boundary into the interpreted
// defer/recover machinery so recover() catches it.
func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
		}
	}()
	var b strings.Builder
	b.Grow(-1)
	fmt.Println("unreachable")
}

// Output:
// recovered: strings.Builder.Grow: negative count
