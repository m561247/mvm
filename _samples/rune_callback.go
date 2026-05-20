package main

import (
	"fmt"
	"strings"
)

// Returning an untyped int constant from a func(rune) rune callback passed to
// a native function (strings.Map) must be converted to rune (int32) at the
// native-callback boundary, as the Go compiler would at the return statement.
func main() {
	fmt.Println(strings.Map(func(rune) rune { return 90 }, "abc"))
}

// Output:
// ZZZ
