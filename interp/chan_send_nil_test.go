package interp_test

import (
	"fmt"
	"testing"
)

// Sending a bare untyped nil on an interface-element channel must deliver the
// element type's nil, not crash: the compiler's iface-wrap is a no-op for an
// untyped nil, so it reaches the channel-send marshaler as an invalid Value.
func TestChannelSendBareNil(t *testing.T) {
	cases := []struct{ n, src, res string }{
		{"chan_error", `ch := make(chan error, 1); ch <- nil; e := <-ch; e == nil`, "true"},
		{"chan_iface", `ch := make(chan interface{}, 1); ch <- nil; v := <-ch; v == nil`, "true"},
		{"select_send", `ch := make(chan error, 1); select { case ch <- nil: }; e := <-ch; e == nil`, "true"},
	}
	for _, c := range cases {
		t.Run(c.n, func(t *testing.T) {
			i := newAutoImportInterp(t)
			r, err := i.Eval(c.n, c.src)
			if err != nil {
				t.Fatalf("eval %q: %v", c.src, err)
			}
			if got := fmt.Sprintf("%v", r); got != c.res {
				t.Errorf("got %q, want %q", got, c.res)
			}
		})
	}
}
