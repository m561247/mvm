package main

// Interpreted types implementing xml.Marshaler (MarshalXML) and
// encoding.TextMarshaler (MarshalText) are encoded through the stdlib/xmlx
// shim, both top-level and nested in a struct field. Without it, native
// encoding/xml reflects over a synthetic rtype whose method set lacks those
// methods and emits the default codec (e.g. <int>1</int> for Animal).

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

func (a Animal) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var s string
	switch a {
	default:
		s = "unknown"
	case Gopher:
		s = "gopher"
	case Zebra:
		s = "zebra"
	}
	return e.EncodeElement(s, start)
}

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

type Size int

func (s Size) MarshalText() ([]byte, error) {
	switch s {
	case 1:
		return []byte("small"), nil
	case 2:
		return []byte("large"), nil
	}
	return []byte("unrecognized"), nil
}

type Zoo struct {
	XMLName xml.Name `xml:"zoo"`
	Animals []Animal `xml:"animal"`
}

func main() {
	out, _ := xml.Marshal(Animal(1)) // scalar MarshalXML
	fmt.Println(string(out))

	out, _ = xml.Marshal(Size(2)) // scalar MarshalText
	fmt.Println(string(out))

	z := Zoo{Animals: []Animal{Gopher, Zebra, Unknown}} // struct + XMLName + custom field
	out, _ = xml.Marshal(&z)
	fmt.Println(string(out))

	var back Zoo // round-trip
	if err := xml.Unmarshal(out, &back); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(back.Animals)
}

// Output:
// <Animal>gopher</Animal>
// <Size>large</Size>
// <zoo><animal>gopher</animal><animal>zebra</animal><animal>unknown</animal></zoo>
// [1 2 0]
