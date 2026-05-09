package main

import (
	"io/fs"
	"testing"

	"github.com/mvm-sh/mvm/stdlib/stdmod"
)

// TestBuildModFS exercises the GOPROXY parsing in buildModFS. The shape
// of the resulting modfs (offline vs network-backed) is internal; this
// test only asserts construction never fails or returns nil.
func TestBuildModFS(t *testing.T) {
	cases := []string{
		"",                                    // default proxy
		"off",                                 // explicit disable -> offline
		"direct",                              // VCS-only -> offline
		"https://example.com/proxy",           // single URL
		"https://example.com/proxy,direct",    // first wins
		"off,https://example.com/proxy",       // first wins (offline)
		" https://example.com/proxy , direct", // whitespace tolerated
	}
	for _, goproxy := range cases {
		t.Setenv("GOPROXY", goproxy)
		if got := buildModFS(); got == nil {
			t.Errorf("GOPROXY=%q: buildModFS returned nil", goproxy)
		}
	}
}

// TestEmbeddedStdResolves checks that stdlib imports resolve through
// the default stdlib redirect FS, by virtue of the embedded std zip
// injected at startup. This is the path NewInterpreter installs for
// callers that don't go through wireFS (tests, embed users).
func TestEmbeddedStdResolves(t *testing.T) {
	stdlibFS := stdmod.DefaultFS()

	if _, err := fs.Stat(stdlibFS, "slices"); err != nil {
		t.Fatalf("stat slices: %v", err)
	}
	data, err := fs.ReadFile(stdlibFS, "slices/slices.go")
	if err != nil {
		t.Fatalf("read slices/slices.go: %v", err)
	}
	if len(data) == 0 {
		t.Error("slices/slices.go empty")
	}
}
