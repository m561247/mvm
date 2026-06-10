package main

// A ptr-recv method promoted through two embedding levels and dispatched
// via an interface must get the embedded field's address: with the bare
// struct in the receiver cell, an inner method call on the receiver mutated
// a detached copy (goldmark ast.ReplaceChild).

type Node interface {
	Parent() Node
	SetParent(Node)
	First() Node
	Append(self, v Node)
	Replace(self, v1, v2 Node)
}

type Base struct {
	parent, first Node
}

func (b *Base) Parent() Node     { return b.parent }
func (b *Base) SetParent(p Node) { b.parent = p }
func (b *Base) First() Node      { return b.first }
func (b *Base) Append(self, v Node) {
	b.first = v
	v.SetParent(self)
}
func (b *Base) Replace(self, v1, v2 Node) {
	b.insert(self, v2)
	b.remove(v1)
}
func (b *Base) insert(self, v Node) { b.first = v; v.SetParent(self) }
func (b *Base) remove(v Node)       { v.SetParent(nil) }

type Mid struct{ Base }
type Para struct{ Mid }
type Leaf struct{ Mid }

func main() {
	p := &Para{}
	n := &Leaf{}
	p.Append(p, n)
	var iface Node = n
	parent := iface.Parent()
	t := &Leaf{}
	parent.Replace(parent, iface, t)
	println("replaced:", p.First() == Node(t))
	println("detached:", n.Parent() == nil)
}

// Output:
// replaced: true
// detached: true
