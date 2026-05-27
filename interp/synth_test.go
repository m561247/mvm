package interp

import (
	"bytes"
	"os"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
	"github.com/mvm-sh/mvm/vm/synth"
)

func TestSynthStringerEndToEnd(t *testing.T) {
	t.Setenv("MVM_SYNTH", "1")

	const src = `package main

import "fmt"

type Greeter struct {
	Name string
}

func (g Greeter) String() string { return "hello " + g.Name }

func main() {
	var s fmt.Stringer = Greeter{Name: "world"}
	fmt.Print(s.String())
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("synth_test.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	if got, want := stdout.String(), "hello world"; got != want {
		t.Errorf("stdout = %q, want %q\nstderr: %s", got, want, stderr.String())
	}
}

// TestSynthPtrStringerEndToEnd is the pointer-receiver counterpart of
// TestSynthStringerEndToEnd: Phase 2a synthesizes a *T rtype via
// attachPtrType and wires PtrToThis so &T satisfies fmt.Stringer.
func TestSynthPtrStringerEndToEnd(t *testing.T) {
	t.Setenv("MVM_SYNTH", "1")

	const src = `package main

import "fmt"

type Counter struct {
	N int
}

func (c *Counter) String() string { return fmt.Sprintf("count=%d", c.N) }

func main() {
	c := &Counter{N: 7}
	var s fmt.Stringer = c
	fmt.Print(s.String())
}
`
	var stdout, stderr bytes.Buffer
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	i.SetIO(os.Stdin, &stdout, &stderr)

	if _, err := i.Eval("synth_ptr_test.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	if got, want := stdout.String(), "count=7"; got != want {
		t.Errorf("stdout = %q, want %q\nstderr: %s", got, want, stderr.String())
	}
}

// TestSynthKindsValueRecv exercises the Phase 2b kind catalog end-to-end:
// each named non-struct kind (primitive, slice, array, map) with a value
// receiver Stringer must satisfy fmt.Stringer and dispatch through the
// synthesized rtype.
func TestSynthKindsValueRecv(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "int",
			src: `package main
import "fmt"
type Code int
func (c Code) String() string { return fmt.Sprintf("code=%d", int(c)) }
func main() { var s fmt.Stringer = Code(7); fmt.Print(s.String()) }
`,
			want: "code=7",
		},
		{
			name: "string",
			src: `package main
import "fmt"
type Path string
func (p Path) String() string { return "path:" + string(p) }
func main() { var s fmt.Stringer = Path("x"); fmt.Print(s.String()) }
`,
			want: "path:x",
		},
		{
			name: "slice",
			src: `package main
import "fmt"
type IntList []int
func (l IntList) String() string { return fmt.Sprintf("list len=%d", len(l)) }
func main() { var s fmt.Stringer = IntList{1, 2, 3}; fmt.Print(s.String()) }
`,
			want: "list len=3",
		},
		{
			name: "array",
			src: `package main
import "fmt"
type Triple [3]int
func (t Triple) String() string { return fmt.Sprintf("triple[0]=%d", t[0]) }
func main() { var s fmt.Stringer = Triple{9, 8, 7}; fmt.Print(s.String()) }
`,
			want: "triple[0]=9",
		},
		{
			name: "map",
			src: `package main
import "fmt"
type Counts map[string]int
func (c Counts) String() string { return fmt.Sprintf("counts len=%d", len(c)) }
func main() {
	c := Counts{"a": 1, "b": 2}
	var s fmt.Stringer = c
	fmt.Print(s.String())
}
`,
			want: "counts len=2",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("MVM_SYNTH", "1")
			i := NewInterpreter(golang.GoSpec)
			i.ImportPackageValues(stdlib.Values)
			var stdout, stderr bytes.Buffer
			i.SetIO(os.Stdin, &stdout, &stderr)
			if _, err := i.Eval(c.name+".go", c.src); err != nil {
				t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
			}
			if got := stdout.String(); got != c.want {
				t.Errorf("stdout = %q, want %q\nstderr: %s",
					got, c.want, stderr.String())
			}
		})
	}
}

// TestSynthAttachIdempotent verifies that a single Eval consumes the
// expected number of S1 slots: one per distinct synth-attached *Type.
// The compiler aliases each Type symbol under bare and pkg-qualified keys
// (compiler.go:136), so without per-*Type dedup the walker would attach the
// same type twice, doubling slot consumption.
func TestSynthAttachIdempotent(t *testing.T) {
	t.Setenv("MVM_SYNTH", "1")

	const src = `package main

import "fmt"

type T struct{ N int }

func (t T) String() string { return fmt.Sprintf("n=%d", t.N) }

func main() {
	var s fmt.Stringer = T{N: 3}
	fmt.Print(s.String())
}
`
	i := NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	var stdout, stderr bytes.Buffer
	i.SetIO(os.Stdin, &stdout, &stderr)

	before := synth.SlotsUsedS1()
	if _, err := i.Eval("a.go", src); err != nil {
		t.Fatalf("Eval: %v\nstderr: %s", err, stderr.String())
	}
	after := synth.SlotsUsedS1()
	if got, want := after-before, uint32(1); got != want {
		t.Errorf("SlotsUsedS1 delta = %d, want %d (alias dedup broken)", got, want)
	}
}
