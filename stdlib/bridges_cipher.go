package stdlib

import (
	"crypto/cipher"
	"reflect"

	"github.com/mvm-sh/mvm/vm"
)

// BridgeStream bridges cipher.Stream so an interpreted type implementing
// XORKeyStream can be stored in native cipher.Stream slots (e.g.
// cipher.StreamReader.S) and called back from native code.
type BridgeStream struct {
	FnXORKeyStream func(dst, src []byte)
	Val            any
	Ifc            vm.Iface
}

// XORKeyStream implements cipher.Stream.
func (b *BridgeStream) XORKeyStream(dst, src []byte) { b.FnXORKeyStream(dst, src) }

func init() {
	vm.InterfaceBridges[reflect.TypeOf((*cipher.Stream)(nil)).Elem()] = reflect.TypeOf((*BridgeStream)(nil))
	vm.ValBridgeTypes[reflect.TypeOf((*BridgeStream)(nil))] = true
}
