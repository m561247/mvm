package main

import (
	"encoding/json"
	"fmt"
)

// Color embeds the native stdlib type json.RawMessage (type RawMessage []byte
// with its own MarshalJSON/UnmarshalJSON) as a value field, exercising the
// native-marshaler delegation in the jsonx walker.
type Color struct {
	Space string
	Point json.RawMessage
}

type RGB struct {
	R uint8
	G uint8
	B uint8
}

func main() {
	// Decode: the nested object is captured verbatim into the RawMessage field
	// (delayed parsing), not mangled as a base64 []byte.
	data := []byte(`{"Space":"RGB","Point":{"R":98,"G":218,"B":255}}`)
	var c Color
	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Println("decode err:", err)
		return
	}
	fmt.Println("raw:", string(c.Point))

	// The captured raw bytes re-decode into a concrete struct.
	var rgb RGB
	if err := json.Unmarshal(c.Point, &rgb); err != nil {
		fmt.Println("redecode err:", err)
		return
	}
	fmt.Println("rgb:", rgb.R, rgb.G, rgb.B)

	// Encode: the RawMessage value field is emitted verbatim, not base64.
	b, _ := json.Marshal(c)
	fmt.Println("encoded:", string(b))

	// Encode: a *json.RawMessage pointer field is also emitted verbatim.
	h := json.RawMessage(`{"precomputed":true}`)
	wrap := struct {
		Header *json.RawMessage `json:"header"`
		Body   string           `json:"body"`
	}{Header: &h, Body: "hi"}
	pb, _ := json.Marshal(wrap)
	fmt.Println("ptr:", string(pb))
}

// Output:
// raw: {"R":98,"G":218,"B":255}
// rgb: 98 218 255
// encoded: {"Space":"RGB","Point":{"R":98,"G":218,"B":255}}
// ptr: {"header":{"precomputed":true},"body":"hi"}
