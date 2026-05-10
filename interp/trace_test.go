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

func TestTracing(t *testing.T) {
	const src = `package main

func main() {
	a := 1
	b := 2
	c := a + b
	_ = c
}
`
	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)

	var stderr bytes.Buffer
	i.SetIO(os.Stdin, &stderr, &stderr)
	i.SetTracing(true)

	if _, err := i.Eval("trace_test.go", src); err != nil {
		t.Fatalf("Eval: %v", err)
	}

	out := stderr.String()
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

func TestTracingDedupsRepeatedLine(t *testing.T) {
	const src = `package main

func main() {
	for i := 0; i < 3; i++ {
		_ = i
	}
}
`
	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)

	var stderr bytes.Buffer
	i.SetIO(os.Stdin, &stderr, &stderr)
	i.SetTracing(true)

	if _, err := i.Eval("dedup_test.go", src); err != nil {
		t.Fatalf("Eval: %v", err)
	}

	out := stderr.String()
	if strings.Count(out, "_ = i") < 3 {
		t.Errorf("expected loop body line printed once per iteration; got:\n%s", out)
	}
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
