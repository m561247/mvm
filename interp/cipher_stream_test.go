package interp

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// cipher.StreamReader.S is the native interface cipher.Stream (XORKeyStream).
// Storing an interpreted implementation there used to panic with
// "reflect.Set: value of type *struct { P1 int } is not assignable to type
// cipher.Stream" because bridgeIface had no registered bridge for cipher.Stream
// and fell back to the raw interpreted struct pointer (same class as the
// crypto.Signer bug in crypto_signer_test.go). The BridgeStream registration
// fixes it: native StreamReader.Read calls back into the interpreted
// XORKeyStream through the bridge.
func TestCipherStreamBridge(t *testing.T) {
	src := `package main

import (
	"crypto/cipher"
	"strings"
)

type addStream struct{ k byte }

func (s *addStream) XORKeyStream(dst, src []byte) {
	for i := range src {
		dst[i] = src[i] + s.k
	}
}

func main() {
	sr := cipher.StreamReader{S: &addStream{k: 1}, R: strings.NewReader("abc")}
	out := make([]byte, 3)
	n, _ := sr.Read(out)
	println("read:", n, string(out))
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("test", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "panic") {
		t.Fatalf("got panic: %s", stderr.String())
	}
	// "abc" + 1 per byte = "bcd".
	if got := stdout.String(); !strings.Contains(got, "read: 3 bcd") {
		t.Errorf("cipher.Stream bridge dispatch failed: stdout=%q stderr=%q", got, stderr.String())
	}
}
