package main

// An interpreted type implementing encoding.TextUnmarshaler (UnmarshalText),
// nested as a slice element ([]Size), is decoded through the stdlib/xmlx shim:
// native encoding/xml feeds such leaf elements their accumulated CharData,
// which xmlx routes to the interpreted UnmarshalText. Without it the synthetic
// rtype lacks the method and "small" is parsed as the underlying int.

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Size int

const (
	Unrecognized Size = iota
	Small
	Large
)

func (s *Size) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	default:
		*s = Unrecognized
	case "small":
		*s = Small
	case "large":
		*s = Large
	}
	return nil
}

func main() {
	blob := `<inventory><size>small</size><size>large</size><size>huge</size></inventory>`
	var inv struct {
		Sizes []Size `xml:"size"`
	}
	if err := xml.Unmarshal([]byte(blob), &inv); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(inv.Sizes)
}

// Output:
// [1 2 0]
