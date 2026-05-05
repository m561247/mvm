package main

import (
	"testing"
)

func TestNewRemoteFS(t *testing.T) {
	cases := []struct {
		goproxy string
		wantNil bool
	}{
		{"", false},                          // default proxy
		{"off", true},                        // explicit disable
		{"direct", true},                     // VCS-only, no proxy support
		{"https://example.com/proxy", false}, // single URL
		{"https://example.com/proxy,direct", false},    // first wins
		{"off,https://example.com/proxy", true},        // first is off
		{" https://example.com/proxy , direct", false}, // whitespace tolerated
	}
	for _, c := range cases {
		t.Setenv("GOPROXY", c.goproxy)
		got := newRemoteFS()
		if (got == nil) != c.wantNil {
			t.Errorf("GOPROXY=%q: got nil=%v, want nil=%v", c.goproxy, got == nil, c.wantNil)
		}
	}
}
