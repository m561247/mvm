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
	vm.RegisterArgProxy(xml.Marshal, 0, newMarshalProxy)
	vm.RegisterArgProxy(xml.MarshalIndent, 0, newMarshalProxy)
	vm.RegisterArgProxyMethod((*xml.Encoder)(nil), "Encode", 0, newMarshalProxy)
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
	if ifc.Typ != nil && containsUnmarshaler(m, ifc.Typ) {
		return reflect.ValueOf(&unmarshalProxy{m: m, ifc: ifc})
	}
	return m.BridgeForAny(ifc)
}

// marshalProxy wraps a mvm Iface so native encoding/xml reflection discovers
// an xml.Marshaler whose MarshalXML re-enters the xmlx encode walker driving
// the native encoder with full Iface metadata.
type marshalProxy struct {
	m   *vm.Machine
	ifc vm.Iface
}

// MarshalXML implements xml.Marshaler. Native xml derives start.Name from this
// proxy's Go type name, so rootStart restores the real element name first.
func (p *marshalProxy) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	rv := p.ifc.Val.Reflect()
	e := &encoder{m: p.m, custom: map[*vm.Type]bool{}}
	return e.marshalElement(enc, rootStart(p.ifc.Typ, rv), rv, p.ifc.Typ)
}

// newMarshalProxy installs the proxy only when ifc's type transitively contains
// an interpreted MarshalXML/MarshalText; otherwise it defers to the default
// any-bridging so native encoding/xml marshals exactly as before.
func newMarshalProxy(m *vm.Machine, ifc vm.Iface) reflect.Value {
	if ifc.Typ != nil && containsMarshaler(m, ifc.Typ) {
		return reflect.ValueOf(&marshalProxy{m: m, ifc: ifc})
	}
	return m.BridgeForAny(ifc)
}

// encoder threads the machine and a per-Marshal memo of containsMarshaler
// answers through the walk (see decoder for the caching-scope rationale).
type encoder struct {
	m      *vm.Machine
	custom map[*vm.Type]bool
}

// marshalElement encodes rv (typed via mvm type typ) as the element named by
// start.
func (e *encoder) marshalElement(enc *xml.Encoder, start xml.StartElement, rv reflect.Value, typ *vm.Type) error {
	if !rv.IsValid() {
		return nil
	}
	if typ == nil {
		return enc.EncodeElement(rv.Interface(), start)
	}
	if typ.Rtype.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return e.marshalElement(enc, start, rv.Elem(), typ.ElemType)
	}
	// Interpreted MarshalXML: dispatch, handing it the native encoder + start.
	if method, ok := e.m.MethodByName(typ, "MarshalXML"); ok {
		recv := marshalReceiver(typ, rv, method)
		args := []reflect.Value{reflect.ValueOf(enc), reflect.ValueOf(start)}
		return dispatchErrMethod(e.m, "MarshalXML", xmlMarshalFnType, args, recv, method)
	}
	// Interpreted MarshalText (encoding.TextMarshaler): native xml wraps the
	// returned text in the element.
	if method, ok := e.m.MethodByName(typ, "MarshalText"); ok {
		recv := marshalReceiver(typ, rv, method)
		text, err := dispatchMarshalText(e.m, recv, method)
		if err != nil {
			return err
		}
		return enc.EncodeElement(string(text), start)
	}
	// No custom marshaler below: let native xml encode the whole subtree
	// (byte-identical to the native bridge).
	if !e.hasCustom(typ) {
		return enc.EncodeElement(rv.Interface(), start)
	}
	switch typ.Rtype.Kind() {
	case reflect.Struct:
		return e.marshalStruct(enc, start, rv, typ)
	case reflect.Slice, reflect.Array:
		for i := range rv.Len() {
			if err := e.marshalElement(enc, start, rv.Index(i), typ.ElemType); err != nil {
				return err
			}
		}
		return nil
	default:
		return enc.EncodeElement(rv.Interface(), start)
	}
}

// marshalStruct hand-walks a struct's element fields, emitting each as a child
// element of start. Only reached for structs that transitively contain a custom
// marshaler. Attribute, chardata, nested-path, and XMLName fields are not
// emitted here (same limits as the decode walk).
func (e *encoder) marshalStruct(enc *xml.Encoder, start xml.StartElement, rv reflect.Value, typ *vm.Type) error {
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	rtype := typ.Rtype
	for i := range rtype.NumField() {
		sf := rtype.Field(i)
		if !sf.IsExported() || sf.Type == xmlNameType {
			continue
		}
		name, isElem := parseXMLTag(sf.Tag.Get("xml"))
		if !isElem || strings.Contains(name, ">") {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		fv := rv.Field(i)
		var ftyp *vm.Type
		if i < len(typ.Fields) {
			ftyp = typ.Fields[i]
		}
		child := xml.StartElement{Name: xml.Name{Local: name}}
		ft := fv.Type()
		if ft.Kind() == reflect.Slice && ft.Elem().Kind() != reflect.Uint8 {
			var elemTyp *vm.Type
			if ftyp != nil {
				elemTyp = ftyp.ElemType
			}
			for j := range fv.Len() {
				if err := e.marshalElement(enc, child, fv.Index(j), elemTyp); err != nil {
					return err
				}
			}
		} else if err := e.marshalElement(enc, child, fv, ftyp); err != nil {
			return err
		}
	}
	return enc.EncodeToken(start.End())
}

// hasCustom is the per-Marshal memoized form of containsMarshaler (see
// decoder.hasCustom for why only top-level answers are cached).
func (e *encoder) hasCustom(typ *vm.Type) bool {
	if typ == nil {
		return false
	}
	if res, ok := e.custom[typ]; ok {
		return res
	}
	res := containsMarshaler(e.m, typ)
	e.custom[typ] = res
	return res
}

// rootStart derives the element start for a top-level marshaled value,
// replacing the proxy-derived name native xml supplies. It mirrors native
// xml's name precedence for the common cases: a struct's XMLName field
// (runtime value, else tag), otherwise the type name.
func rootStart(typ *vm.Type, rv reflect.Value) xml.StartElement {
	for typ != nil && typ.Rtype.Kind() == reflect.Pointer {
		typ = typ.ElemType
		// typ and rv deref in lockstep, but rv may bottom out first (nil link).
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				rv = reflect.Value{}
			} else {
				rv = rv.Elem()
			}
		}
	}
	if typ == nil {
		return xml.StartElement{}
	}
	if typ.Rtype.Kind() == reflect.Struct {
		for i := range typ.Rtype.NumField() {
			sf := typ.Rtype.Field(i)
			if sf.Type != xmlNameType {
				continue
			}
			if rv.IsValid() && rv.Kind() == reflect.Struct {
				if n, ok := rv.Field(i).Interface().(xml.Name); ok && n.Local != "" {
					return xml.StartElement{Name: n}
				}
			}
			if tag, _ := parseXMLTag(sf.Tag.Get("xml")); tag != "" {
				return xml.StartElement{Name: xmlNameFromTag(tag)}
			}
			break
		}
	}
	return xml.StartElement{Name: xml.Name{Local: typ.Name}}
}

// xmlNameFromTag splits an XMLName tag ("namespace local" or "local") into a Name.
func xmlNameFromTag(tag string) xml.Name {
	if i := strings.LastIndex(tag, " "); i >= 0 {
		return xml.Name{Space: tag[:i], Local: tag[i+1:]}
	}
	return xml.Name{Local: tag}
}

// marshalReceiver builds the Iface receiver for a marshal method.
// MakeMethodCallable does not auto-address, so a pointer-receiver method needs
// a pointer Val.
func marshalReceiver(typ *vm.Type, rv reflect.Value, method vm.Method) vm.Iface {
	if method.PtrRecv {
		if rv.CanAddr() {
			return vm.Iface{Typ: typ, Val: vm.FromReflect(rv.Addr())}
		}
		p := reflect.New(rv.Type())
		p.Elem().Set(rv)
		return vm.Iface{Typ: typ, Val: vm.FromReflect(p)}
	}
	return vm.Iface{Typ: typ, Val: vm.FromReflect(rv)}
}

// dispatchMarshalText dispatches a func() ([]byte, error) MarshalText method.
func dispatchMarshalText(m *vm.Machine, ifc vm.Iface, method vm.Method) ([]byte, error) {
	fval := m.MakeMethodCallable(ifc, method)
	out, err := m.CallFunc(fval, textMarshalFnType, nil)
	if err != nil {
		return nil, err
	}
	if len(out) != 2 {
		return nil, fmt.Errorf("MarshalText: expected 2 returns, got %d", len(out))
	}
	var data []byte
	if out[0].IsValid() && !out[0].IsZero() {
		data = out[0].Bytes()
	}
	if out[1].IsValid() && !out[1].IsNil() {
		if e, ok := out[1].Interface().(error); ok {
			return data, e
		}
	}
	return data, nil
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
			return dispatchErrMethod(d.m, "UnmarshalXML", xmlUnmarshalFnType, args, *recv, method)
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
			return dispatchErrMethod(d.m, "UnmarshalText", textUnmarshalFnType, args, *recv, method)
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
	xmlMarshalFnType    = reflect.TypeOf((func(*xml.Encoder, xml.StartElement) error)(nil))
	textMarshalFnType   = reflect.TypeOf((func() ([]byte, error))(nil))
	xmlNameType         = reflect.TypeOf(xml.Name{})
)

// dispatchErrMethod dispatches a single-error-return method (UnmarshalXML,
// UnmarshalText, MarshalXML) through the VM. name is used only for the arity
// error.
func dispatchErrMethod(m *vm.Machine, name string, fnType reflect.Type, args []reflect.Value, ifc vm.Iface, method vm.Method) error {
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

// hasCustom is the per-Unmarshal memoized form of containsUnmarshaler. Only
// top-level answers are cached: sub-results computed mid-cycle can be wrong,
// since a node's answer may depend on an ancestor not yet resolved.
func (d *decoder) hasCustom(typ *vm.Type) bool {
	if typ == nil {
		return false
	}
	if res, ok := d.custom[typ]; ok {
		return res
	}
	res := containsUnmarshaler(d.m, typ)
	d.custom[typ] = res
	return res
}

// containsUnmarshaler / containsMarshaler report whether typ or any
// transitively reachable field, element, key, or embedded type defines the
// corresponding interpreted custom (un)marshal methods. The mvm type graph
// forms pointer cycles for recursive types, so a pointer-keyed visited set
// terminates.
func containsUnmarshaler(m *vm.Machine, typ *vm.Type) bool {
	return graphHasMethod(m, typ, map[*vm.Type]bool{}, "UnmarshalXML", "UnmarshalText")
}

func containsMarshaler(m *vm.Machine, typ *vm.Type) bool {
	return graphHasMethod(m, typ, map[*vm.Type]bool{}, "MarshalXML", "MarshalText")
}

// graphHasMethod reports whether typ or any reachable type has an interpreted
// method named a or b.
func graphHasMethod(m *vm.Machine, typ *vm.Type, seen map[*vm.Type]bool, a, b string) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	if _, ok := m.MethodByName(typ, a); ok {
		return true
	}
	if _, ok := m.MethodByName(typ, b); ok {
		return true
	}
	for _, f := range typ.Fields {
		if graphHasMethod(m, f, seen, a, b) {
			return true
		}
	}
	if graphHasMethod(m, typ.ElemType, seen, a, b) {
		return true
	}
	if graphHasMethod(m, typ.KeyType, seen, a, b) {
		return true
	}
	for _, e := range typ.Embedded {
		if graphHasMethod(m, e.Type, seen, a, b) {
			return true
		}
	}
	return false
}
