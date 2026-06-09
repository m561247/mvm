// A map composite literal with interface{} values must store raw Go values,
// not boxed vm.Iface wrappers, so native reflect-based code (json.Marshal)
// sees the concrete values.
package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	m := map[string]interface{}{"a": "x", "b": 1}
	out, err := json.Marshal(m)
	fmt.Println(string(out), err)
}

// Output:
// {"a":"x","b":1} <nil>
