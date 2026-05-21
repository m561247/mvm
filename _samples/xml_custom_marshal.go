package main

// An interpreted type implementing xml.Unmarshaler (UnmarshalXML), nested
// inside a struct field ([]Animal), is decoded correctly through the mvm-aware
// stdlib/xmlx shim. Without it, native encoding/xml reflects over a synthetic
// rtype whose method set lacks UnmarshalXML and falls back to parsing the
// CharData "gopher" as the underlying int (strconv.ParseInt error).

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Animal int

const (
	Unknown Animal = iota
	Gopher
	Zebra
)

func (a *Animal) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	default:
		*a = Unknown
	case "gopher":
		*a = Gopher
	case "zebra":
		*a = Zebra
	}
	return nil
}

func main() {
	blob := `<animals><animal>gopher</animal><animal>zebra</animal><animal>bee</animal></animals>`
	var zoo struct {
		Animals []Animal `xml:"animal"`
	}
	if err := xml.Unmarshal([]byte(blob), &zoo); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(zoo.Animals)
}

// Output:
// [1 2 0]
