package main

// A pointer-receiver method promoted from a VALUE embed (Write from an embedded
// bytes.Buffer) lives in *E's method set, not E's. Dispatch must retry the lookup
// on the addressable field's address, and the mutation must write back. (review #2)

import (
	"bytes"
	"fmt"
	"io"
)

type valBuf struct {
	bytes.Buffer
	closed bool
}

func (v *valBuf) Close() error { v.closed = true; return nil }

func main() {
	vb := &valBuf{}
	var iw io.Writer = vb // *valBuf satisfies io.Writer via promoted bytes.Buffer.Write
	n, err := fmt.Fprint(iw, "hello")
	fmt.Println(n, err, vb.String())
}

// Output:
// 5 <nil> hello
