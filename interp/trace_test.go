package interp_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// runTraced compiles src under name with the given trace modes and returns
// whatever the VM wrote to stderr.
func runTraced(t *testing.T, name, src string, line, op bool) string {
	t.Helper()
	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	var stderr bytes.Buffer
	i.SetIO(os.Stdin, &stderr, &stderr)
	if line {
		i.SetTracing(true)
	}
	if op {
		i.SetTraceOps(true)
	}
	if _, err := i.Eval(name, src); err != nil {
		t.Fatalf("Eval: %v", err)
	}
	return stderr.String()
}

const traceSampleSrc = `package main

func main() {
	a := 1
	b := 2
	c := a + b
	_ = c
}
`

func TestTracingLines(t *testing.T) {
	t.Parallel()
	out := runTraced(t, "trace_test.go", traceSampleSrc, true, false)
	wantInOrder := []string{
		"+ trace_test.go:4: \ta := 1",
		"+ trace_test.go:5: \tb := 2",
		"+ trace_test.go:6: \tc := a + b",
		"+ trace_test.go:7: \t_ = c",
	}
	prev := 0
	for _, want := range wantInOrder {
		idx := strings.Index(out[prev:], want)
		if idx < 0 {
			t.Errorf("trace output missing %q (after offset %d)\nfull output:\n%s", want, prev, out)
			continue
		}
		prev += idx + len(want)
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "+ ") {
			t.Errorf("unexpected non-trace line in stderr: %q", line)
		}
	}
}

func TestTracingOps(t *testing.T) {
	t.Parallel()
	out := runTraced(t, "ops_test.go", traceSampleSrc, false, true)
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "+ op ") {
			t.Errorf("non op-trace line in stderr: %q", line)
		}
	}
	if !strings.Contains(out, "ip:") || !strings.Contains(out, "sp:") || !strings.Contains(out, "op:[") {
		t.Errorf("op-trace output missing expected fields:\n%s", out)
	}
	if !strings.Contains(out, "Push") {
		t.Errorf("expected at least one Push instruction in op trace:\n%s", out)
	}
}

func TestTracingBoth(t *testing.T) {
	t.Parallel()
	out := runTraced(t, "both_test.go", traceSampleSrc, true, true)
	idxLine := strings.Index(out, "+ both_test.go:4: \ta := 1")
	if idxLine < 0 {
		t.Fatalf("missing line trace in combined output:\n%s", out)
	}
	if !strings.Contains(out, "+ op ") {
		t.Errorf("missing op trace in combined output:\n%s", out)
	}
	if !strings.Contains(out[idxLine:], "+ op ") {
		t.Errorf("expected op trace after line trace; output:\n%s", out)
	}
}

func TestTracingDedupsRepeatedLine(t *testing.T) {
	t.Parallel()
	const src = `package main

func main() {
	for i := 0; i < 3; i++ {
		_ = i
	}
}
`
	out := runTraced(t, "dedup_test.go", src, true, false)
	bodyLines := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "_ = i") {
			bodyLines++
		}
	}
	if bodyLines < 3 {
		t.Errorf("expected at least 3 emissions of loop body line, got %d\noutput:\n%s", bodyLines, out)
	}
}

func TestParseTraceModes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		wantLine bool
		wantOp   bool
	}{
		{"", false, false},
		{"1", true, false},
		{"line", true, false},
		{"op", false, true},
		{"bytecode", false, true},
		{"all", true, true},
		{"line,op", true, true},
		{"  line , op ", true, true},
		{"unknown", false, false},
		{"line,unknown,op", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			line, op := interp.ParseTraceModes(tc.in)
			if line != tc.wantLine || op != tc.wantOp {
				t.Errorf("ParseTraceModes(%q) = (%v,%v), want (%v,%v)", tc.in, line, op, tc.wantLine, tc.wantOp)
			}
		})
	}
}
