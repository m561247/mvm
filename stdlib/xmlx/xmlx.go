// Package xmlx is a mvm-aware shim for the encoding/xml functions that
// must honour xml.Unmarshaler methods (UnmarshalXML) defined on
// interpreted types. Native encoding/xml reflects over a synthetic native
// rtype for an interpreted type whose method set does not include the
// interpreted UnmarshalXML, so it falls back to the default codec (e.g.
// decoding the CharData "gopher" into the int underlying type). This shim
// re-implements just enough of the decode walk to reach interpreted custom
// unmarshalers, delegating every pure-native subtree wholesale to the
// native xml.Decoder.
//
// Dispatch is wired through vm.RegisterArgProxy: the second argument to
// xml.Unmarshal is wrapped as an *unmarshalProxy whose UnmarshalXML
// re-enters the walker with full Iface metadata. Native encoding/xml
// reflection sees the proxy as an ordinary xml.Unmarshaler.
//
// The proxy is installed only when the destination type transitively
// contains an interpreted UnmarshalXML (see containsCustom); otherwise the
// argument falls back to default any-bridging and native xml handles it
// exactly as before. This keeps pure-native decoding byte-identical to the
// native bridge. Mirrors stdlib/jsonx and stdlib/gobx.
//
// Not yet handled in a hand-walked (custom-bearing) struct: ",attr" fields,
// nested-path tags ("a>b>c"), ",chardata"/",cdata"/",comment"/",innerxml",
// and the encode direction (xml.Marshal). Those subtrees are only walked
// when they sit alongside a custom unmarshaler; pure-native structs using
// them are delegated whole and keep working.
package xmlx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/mvm-sh/mvm/vm"
)

func init() {
	vm.RegisterArgProxy(xml.Unmarshal, 1, newUnmarshalProxy)
}

// unmarshalProxy wraps a mvm Iface so native encoding/xml reflection
// discovers an xml.Unmarshaler whose UnmarshalXML re-enters the xmlx
// walker with full Iface metadata.
type unmarshalProxy struct {
	m   *vm.Machine
	ifc vm.Iface
}

// UnmarshalXML implements xml.Unmarshaler. The decoder is positioned just
// after start has been read; the walker consumes the element to its match.
func (p *unmarshalProxy) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	return (&decoder{m: p.m, custom: map[*vm.Type]bool{}}).unmarshalElement(dec, start, p.ifc)
}

// newUnmarshalProxy installs the proxy only when ifc's type transitively
// contains an interpreted UnmarshalXML; otherwise it defers to the default
// any-bridging so native encoding/xml decodes exactly as before.
func newUnmarshalProxy(m *vm.Machine, ifc vm.Iface) reflect.Value {
	if ifc.Typ != nil && containsCustom(m, ifc.Typ) {
		return reflect.ValueOf(&unmarshalProxy{m: m, ifc: ifc})
	}
	return m.BridgeForAny(ifc)
}

// decoder threads the machine and a per-Unmarshal memo of containsCustom
// answers through the walk. Caching by *vm.Type pointer is safe only because
// the cache lives for a single Unmarshal call: those pointers are stable for
// the life of the owning machine, so a process-wide cache could go stale.
type decoder struct {
	m      *vm.Machine
	custom map[*vm.Type]bool
}

// unmarshalElement decodes the element whose start has already been read
// into the destination described by ifc (expected to box a pointer).
func (d *decoder) unmarshalElement(dec *xml.Decoder, start xml.StartElement, ifc vm.Iface) error {
	rv := ifc.Val.Reflect()
	if !rv.IsValid() {
		return errors.New("xml: invalid unmarshal destination")
	}
	if ifc.Typ == nil {
		return dec.DecodeElement(rv.Interface(), &start)
	}
	if ifc.Typ.Rtype.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return errors.New("xml: nil pointer unmarshal destination")
		}
		return d.decodeElement(dec, start, rv.Elem(), ifc.Typ.ElemType)
	}
	if rv.CanAddr() {
		return d.decodeElement(dec, start, rv, ifc.Typ)
	}
	return dec.DecodeElement(rv.Interface(), &start)
}

// decodeElement decodes one already-started element into the addressable
// reflect.Value dst (typed via mvm type typ).
func (d *decoder) decodeElement(dec *xml.Decoder, start xml.StartElement, dst reflect.Value, typ *vm.Type) error {
	if typ == nil {
		return delegateDecode(dec, start, dst)
	}
	// Interpreted UnmarshalXML on this type: dispatch, handing it the native
	// decoder so its body (d.DecodeElement(&s, &start)) consumes the element.
	if method, ok := d.m.MethodByName(typ, "UnmarshalXML"); ok {
		recv, err := pointerReceiver(typ, dst)
		if err != nil {
			return err
		}
		if recv != nil {
			args := []reflect.Value{reflect.ValueOf(dec), reflect.ValueOf(start)}
			return invokeUnmarshal(d.m, "UnmarshalXML", xmlUnmarshalFnType, args, *recv, method)
		}
	}
	// Interpreted UnmarshalText (encoding.TextUnmarshaler): native xml feeds
	// such leaf elements their accumulated CharData. Decode the element's text
	// via the native decoder, then dispatch UnmarshalText.
	if method, ok := d.m.MethodByName(typ, "UnmarshalText"); ok {
		recv, err := pointerReceiver(typ, dst)
		if err != nil {
			return err
		}
		if recv != nil {
			var text string
			if err := dec.DecodeElement(&text, &start); err != nil {
				return err
			}
			args := []reflect.Value{reflect.ValueOf([]byte(text))}
			return invokeUnmarshal(d.m, "UnmarshalText", textUnmarshalFnType, args, *recv, method)
		}
	}
	// No custom unmarshaler anywhere below: let native xml decode the whole
	// subtree (byte-identical to the native bridge).
	if !d.hasCustom(typ) {
		return delegateDecode(dec, start, dst)
	}
	switch typ.Rtype.Kind() {
	case reflect.Struct:
		return d.decodeStruct(dec, dst, typ)
	case reflect.Pointer:
		if dst.IsNil() {
			if !dst.CanSet() {
				return delegateDecode(dec, start, dst)
			}
			dst.Set(reflect.New(typ.Rtype.Elem()))
		}
		return d.decodeElement(dec, start, dst.Elem(), typ.ElemType)
	default:
		return delegateDecode(dec, start, dst)
	}
}

// decodeStruct hand-walks a struct element's child tokens, dispatching each
// child to its matching field. Pure-native fields are delegated; slice
// fields accumulate one element per occurrence; custom-bearing fields
// recurse. Only reached for structs that transitively contain a custom
// unmarshaler.
func (d *decoder) decodeStruct(dec *xml.Decoder, dst reflect.Value, typ *vm.Type) error {
	rtype := typ.Rtype
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			fi, ftyp := lookupElemField(rtype, typ, t.Name.Local)
			if fi < 0 {
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			fv := dst.Field(fi)
			if !fv.CanSet() {
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			ft := fv.Type()
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() != reflect.Uint8 {
				var elemTyp *vm.Type
				if ftyp != nil {
					elemTyp = ftyp.ElemType
				}
				ev := reflect.New(ft.Elem()).Elem()
				if err := d.decodeElement(dec, t, ev, elemTyp); err != nil {
					return err
				}
				fv.Set(reflect.Append(fv, ev))
			} else if err := d.decodeElement(dec, t, fv, ftyp); err != nil {
				return err
			}
		case xml.EndElement:
			// The only EndElement reachable at this level is start's match;
			// nested ends are consumed by recursion / Skip / DecodeElement.
			return nil
		}
	}
}

// delegateDecode hands the element to the native decoder, decoding into a
// temporary when dst is not addressable.
func delegateDecode(dec *xml.Decoder, start xml.StartElement, dst reflect.Value) error {
	if dst.CanAddr() {
		return dec.DecodeElement(dst.Addr().Interface(), &start)
	}
	tmp := reflect.New(dst.Type())
	if err := dec.DecodeElement(tmp.Interface(), &start); err != nil {
		return err
	}
	if dst.CanSet() {
		dst.Set(tmp.Elem())
	}
	return nil
}

// lookupElemField finds the struct field decoded from a child element named
// name. An explicit xml tag name wins over a bare field name. Attribute,
// chardata, skipped, and nested-path fields are not matched here.
func lookupElemField(rtype reflect.Type, typ *vm.Type, name string) (int, *vm.Type) {
	fallback := -1
	for i := range rtype.NumField() {
		sf := rtype.Field(i)
		if !sf.IsExported() {
			continue
		}
		tname, isElem := parseXMLTag(sf.Tag.Get("xml"))
		if !isElem || strings.Contains(tname, ">") {
			continue
		}
		if tname == name { // tname is never "" here, so this is a real tag match
			return i, fieldTypeAt(typ, i)
		}
		if tname == "" && sf.Name == name && fallback < 0 {
			fallback = i
		}
	}
	if fallback >= 0 {
		return fallback, fieldTypeAt(typ, fallback)
	}
	return -1, nil
}

// parseXMLTag returns the element name from a struct field's xml tag and
// whether the field is decoded from a child element. isElem is false for
// "-" and for attr/chardata/comment/innerxml/any fields, which native xml
// fills from sources other than a name-matched child element.
func parseXMLTag(tag string) (name string, isElem bool) {
	if tag == "-" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, opt := range parts[1:] {
		switch opt {
		case "attr", "chardata", "cdata", "comment", "innerxml", "any":
			return name, false
		}
	}
	return name, true
}

func fieldTypeAt(typ *vm.Type, i int) *vm.Type {
	if i < len(typ.Fields) {
		return typ.Fields[i]
	}
	return nil
}

// pointerReceiver builds a pointer Iface receiver for invoking UnmarshalXML.
// Returns (nil, nil) if dst can neither be addressed nor pre-allocated.
func pointerReceiver(typ *vm.Type, dst reflect.Value) (*vm.Iface, error) {
	var recv reflect.Value
	switch {
	case typ.Rtype.Kind() == reflect.Pointer:
		if dst.IsNil() {
			if !dst.CanSet() {
				return nil, errors.New("xml: non-settable pointer destination")
			}
			dst.Set(reflect.New(typ.Rtype.Elem()))
		}
		recv = dst
	case dst.CanAddr():
		recv = dst.Addr()
	}
	if !recv.IsValid() {
		return nil, nil
	}
	return &vm.Iface{Typ: typ, Val: vm.FromReflect(recv)}, nil
}

var (
	xmlUnmarshalFnType  = reflect.TypeOf((func(*xml.Decoder, xml.StartElement) error)(nil))
	textUnmarshalFnType = reflect.TypeOf((func([]byte) error)(nil))
)

// invokeUnmarshal dispatches an error-returning unmarshal method (UnmarshalXML
// or UnmarshalText) through the VM. name is used only for the arity error.
func invokeUnmarshal(m *vm.Machine, name string, fnType reflect.Type, args []reflect.Value, ifc vm.Iface, method vm.Method) error {
	fval := m.MakeMethodCallable(ifc, method)
	out, err := m.CallFunc(fval, fnType, args)
	if err != nil {
		return err
	}
	if len(out) != 1 {
		return fmt.Errorf("%s: expected 1 return, got %d", name, len(out))
	}
	if out[0].IsValid() && !out[0].IsNil() {
		if e, ok := out[0].Interface().(error); ok {
			return e
		}
	}
	return nil
}

// --- custom-unmarshaler detection ---

// hasCustom is the per-Unmarshal memoized form of containsCustom. Only
// top-level answers are cached: sub-results computed mid-cycle can be wrong,
// since a node's answer may depend on an ancestor not yet resolved.
func (d *decoder) hasCustom(typ *vm.Type) bool {
	if typ == nil {
		return false
	}
	if res, ok := d.custom[typ]; ok {
		return res
	}
	res := containsCustom(d.m, typ)
	d.custom[typ] = res
	return res
}

// containsCustom reports whether typ or any transitively reachable field,
// element, key, or embedded type defines an interpreted UnmarshalXML or
// UnmarshalText. The mvm type graph forms pointer cycles for recursive
// types, so a pointer-keyed visited set terminates.
func containsCustom(m *vm.Machine, typ *vm.Type) bool {
	return containsCustomWalk(m, typ, map[*vm.Type]bool{})
}

func containsCustomWalk(m *vm.Machine, typ *vm.Type, seen map[*vm.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	if _, ok := m.MethodByName(typ, "UnmarshalXML"); ok {
		return true
	}
	if _, ok := m.MethodByName(typ, "UnmarshalText"); ok {
		return true
	}
	for _, f := range typ.Fields {
		if containsCustomWalk(m, f, seen) {
			return true
		}
	}
	if containsCustomWalk(m, typ.ElemType, seen) {
		return true
	}
	if containsCustomWalk(m, typ.KeyType, seen) {
		return true
	}
	for _, e := range typ.Embedded {
		if containsCustomWalk(m, e.Type, seen) {
			return true
		}
	}
	return false
}
