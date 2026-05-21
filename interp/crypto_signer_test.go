//go:build go1.25

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

// crypto.SignMessage(signer crypto.Signer, ...) does an interface upgrade
// signer.(crypto.MessageSigner) and calls SignMessage when present.
// An interpreted type that implements Public+Sign+SignMessage must be
// bridged with a host proxy that satisfies crypto.MessageSigner, not just
// crypto.Signer, or the upgrade fails and Sign (the error path here) runs.
//
// Before vm.bestInterfaceBridge, bridgeIface had no registered bridge for
// crypto.Signer and passed the raw interpreted struct pointer to the native
// call, panicking with "reflect: Call using *struct { P1 int } as type
// crypto.Signer".
func TestCryptoMessageSignerBridge(t *testing.T) {
	src := `package main

import (
	"crypto"
	"errors"
	"io"
)

type onlyMsg struct{ tag string }

func (s *onlyMsg) Public() crypto.PublicKey { return nil }

func (s *onlyMsg) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	return nil, errors.New("Sign should not be called")
}

func (s *onlyMsg) SignMessage(_ io.Reader, msg []byte, _ crypto.SignerOpts) ([]byte, error) {
	return append([]byte(s.tag), msg...), nil
}

func main() {
	sig, err := crypto.SignMessage(&onlyMsg{tag: "msg:"}, nil, []byte("hi"), nil)
	if err != nil {
		println("err:", err.Error())
		return
	}
	println("sig:", string(sig))
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
	if got := stdout.String(); !strings.Contains(got, "sig: msg:hi") {
		t.Errorf("MessageSigner upgrade not taken: stdout=%q stderr=%q", got, stderr.String())
	}
}
