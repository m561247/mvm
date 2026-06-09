package main

// A named type over []byte must keep its identity across an interface{} so a
// type switch matches its own case, not `case []byte`. Two fixes: type-switch
// on a native concrete read from a slice uses exact identity (not AssignableTo),
// and a string->named-[]byte conversion (json.RawMessage) carries the dst rtype.
// (rs/zerolog TestFieldsMap)

import (
	"encoding/json"
	"net"
)

func cls(v interface{}) string {
	switch v.(type) {
	case []byte:
		return "[]byte"
	case net.IP:
		return "net.IP"
	case json.RawMessage:
		return "json.RawMessage"
	default:
		return "other"
	}
}

func main() {
	kv := make([]interface{}, 2)
	kv[0] = net.IP{0x20, 0x01}
	kv[1] = json.RawMessage(`{"x":1}`)
	b := []byte("z")
	println(cls(kv[0]))
	println(cls(kv[1]))
	println(cls(b))
}

// Output:
// net.IP
// json.RawMessage
// []byte
