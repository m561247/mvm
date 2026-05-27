// Package synth synthesizes Go rtypes with interpreted-method metadata,
// so native code can invoke methods on directly interpeter objects.
//
// The mirrors here track internal/abi.Type, StructType, UncommonType, and
// Method byte-for-byte.
// Layout drift across Go versions is caught by abi_test.go probes that
// verify against a real native rtype.
package synth
