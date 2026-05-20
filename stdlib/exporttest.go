package stdlib

import (
	"reflect"
	"strings"
)

// TestValues holds faithful stand-ins for symbols that a stdlib package's own
// export_test.go injects into the package under test. Bridged stdlib packages
// are native, so those internal-only symbols don't exist on the bridge; here
// we reproduce the ones that can be built from exported API (or self-contained
// ported source) so external *_test.go files using them can run.
//
// Merged over Values ONLY by `mvm test` (see TestOverlay); `mvm run` never
// sees these, so the real package surface stays clean.
//
// Only faithfully-reproducible symbols belong here. Ones that read a package's
// unexported state -- e.g. (*strings.Replacer).Replacer()/PrintTrie(), which
// return the internal algorithm value/trie -- cannot be reproduced (and would
// need a method attached to a native type, which mvm can't dispatch), so they
// are left to `mvm test`'s drop-on-compile-error retry.
var TestValues = map[string]map[string]reflect.Value{
	"strings": {
		// export_test.go: StringFind(pattern, text) == Index(text, pattern).
		"StringFind": reflect.ValueOf(func(pattern, text string) int {
			return strings.Index(text, pattern)
		}),
		// export_test.go returns makeStringFinder's Boyer-Moore skip tables;
		// no exported API yields them, so the finder is ported below.
		"DumpTables": reflect.ValueOf(func(pattern string) ([]int, []int) {
			f := makeStringFinder(pattern)
			return f.badCharSkip[:], f.goodSuffixSkip
		}),
	},
}

// TestOverlay returns each TestValues package merged over its Values base, so
// a single ImportPackageValues installs the package with both its real bridge
// symbols and the test-only stand-ins.
func TestOverlay() map[string]map[string]reflect.Value {
	out := make(map[string]map[string]reflect.Value, len(TestValues))
	for pkg, syms := range TestValues {
		m := make(map[string]reflect.Value, len(Values[pkg])+len(syms))
		for k, v := range Values[pkg] {
			m[k] = v
		}
		for k, v := range syms {
			m[k] = v
		}
		out[pkg] = m
	}
	return out
}

// stringFinder, makeStringFinder, and longestCommonSuffix are ported verbatim
// from $GOROOT/src/strings/search.go (BSD-licensed, The Go Authors) so that
// DumpTables can reproduce the exact Boyer-Moore skip tables search_test.go
// asserts on. Keep in sync if the upstream algorithm changes.
type stringFinder struct {
	pattern        string
	badCharSkip    [256]int
	goodSuffixSkip []int
}

func makeStringFinder(pattern string) *stringFinder {
	f := &stringFinder{
		pattern:        pattern,
		goodSuffixSkip: make([]int, len(pattern)),
	}
	last := len(pattern) - 1

	// Bad-character table: bytes not in the pattern skip its whole length.
	for i := range f.badCharSkip {
		f.badCharSkip[i] = len(pattern)
	}
	for i := 0; i < last; i++ {
		f.badCharSkip[pattern[i]] = last - i
	}

	// Good-suffix table, first pass: next index starting a prefix of pattern.
	lastPrefix := last
	for i := last; i >= 0; i-- {
		if strings.HasPrefix(pattern, pattern[i+1:]) {
			lastPrefix = i + 1
		}
		f.goodSuffixSkip[i] = lastPrefix + last - i
	}
	// Second pass: repeats of the suffix starting from the front.
	for i := 0; i < last; i++ {
		lenSuffix := longestCommonSuffix(pattern, pattern[1:i+1])
		if pattern[i-lenSuffix] != pattern[last-lenSuffix] {
			f.goodSuffixSkip[last-lenSuffix] = lenSuffix + last - i
		}
	}

	return f
}

func longestCommonSuffix(a, b string) (i int) {
	for ; i < len(a) && i < len(b); i++ {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			break
		}
	}
	return
}
