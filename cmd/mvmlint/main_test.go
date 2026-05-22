package main

import (
	"strings"
	"testing"
)

// lintString runs the checks over a source string and returns directive-filtered
// findings as "file:line:col: msg (check)" lines.
func lintString(src string) []string {
	fl := &fileLinter{src: src}
	fl.walk(src, 0, newCtx())
	return fl.results("x.go")
}

func countCheck(res []string, check string) int {
	n := 0
	for _, r := range res {
		if strings.Contains(r, "("+check+")") {
			n++
		}
	}
	return n
}

func TestSymkeySafe(t *testing.T) {
	const src = `package p
func (p *P) ok(name string) {
	p.Symbols[p.pkgKey("Foo")] = x       // qualified
	p.Symbols[QualifyName("pkg", "F")] = x // qualified
	p.Symbols["int"] = x                 // predeclared type
	p.Symbols["append"] = x              // predeclared builtin
	p.Symbols["pkg.Foo"] = x             // literal already qualified
	key := p.scope + "/_ts"
	p.Symbols[key] = x                   // scoped via local concat
	k2 := p.pkgKey("Bar")
	p.Symbols[k2] = x                    // local from qualifier
	name2 := s.PkgPath + "." + typeName
	p.SymAdd(0, name2, v)                // hand-built qualified key
	p.SymSet(name, x)                    // forwarded parameter
}
`
	if res := lintString(src); countCheck(res, "symkey") != 0 {
		t.Fatalf("expected 0 symkey findings, got %d:\n%s", countCheck(res, "symkey"), strings.Join(res, "\n"))
	}
}

func TestSymkeyUnsafe(t *testing.T) {
	const src = `package p
func (p *P) bad() {
	widget := "Widget"
	p.Symbols[widget] = x   // bare local from bare literal
	p.Symbols["Gadget"] = x // bare string literal
	p.SymSet("Gizmo", x)    // bare literal via helper
	p.SymAdd(0, "Cog", x)   // bare literal via helper arg 1
}
`
	if got := countCheck(lintString(src), "symkey"); got != 4 {
		t.Fatalf("expected 4 symkey findings, got %d:\n%s", got, strings.Join(lintString(src), "\n"))
	}
}

func TestSymkeyDirectiveSuppression(t *testing.T) {
	const src = `package p
func (p *P) x() {
	p.Symbols["Sprocket"] = x //mvm:symkey-ok intentional bare key
}
`
	if got := countCheck(lintString(src), "symkey"); got != 0 {
		t.Fatalf("directive should suppress, got %d findings", got)
	}
}

// TestSymkeyTypeParamShadow locks in the real bug shape this check surfaced:
// the bare-key write in registerFunc's generic branch (goparser/func.go:44).
func TestSymkeyTypeParamShadow(t *testing.T) {
	const src = `package p
func (p *P) registerFunc() {
	for _, tp := range params {
		p.Symbols[tp.name] = sym
	}
}
`
	if got := countCheck(lintString(src), "symkey"); got != 1 {
		t.Fatalf("expected the bare type-param placeholder write to be flagged, got %d", got)
	}
}

func TestPosbase(t *testing.T) {
	const safe = `package p
func (c *C) emit(t T) { p := t.Pos + c.PosBase; _ = p }
`
	if got := countCheck(lintString(safe), "posbase"); got != 0 {
		t.Fatalf("emit() may add PosBase, got %d findings", got)
	}
	const bad = `package p
func (c *C) other(t T) { p := t.Pos + c.PosBase; _ = p }
`
	if got := countCheck(lintString(bad), "posbase"); got != 1 {
		t.Fatalf("expected 1 posbase finding outside emit, got %d", got)
	}
}

// TestPosbaseCompoundAssign locks in detection of `+= PosBase`.
func TestPosbaseCompoundAssign(t *testing.T) {
	const src = `package p
func (c *C) other(t T) { pos := 0; pos += c.PosBase; _ = pos }
`
	if got := countCheck(lintString(src), "posbase"); got != 1 {
		t.Fatalf("expected += PosBase to be flagged, got %d", got)
	}
}

// TestPosbaseNonAdditive guards against a false positive from an unrelated `+`
// near a non-additive use of PosBase.
func TestPosbaseNonAdditive(t *testing.T) {
	const src = `package p
func (c *C) other(x int) { if c.PosBase > x + 1 { _ = x } }
`
	if got := countCheck(lintString(src), "posbase"); got != 0 {
		t.Fatalf("comparison use of PosBase must not be flagged, got %d", got)
	}
}

// TestSymkeyBoundaryMatching guards the boundary-aware qualifier/PkgPath checks
// against substring over-clearing.
func TestSymkeyBoundaryMatching(t *testing.T) {
	const src = `package p
func (p *P) f() {
	k1 := somethingpkgKey()
	p.Symbols[k1] = x                // "pkgKey(" only as a substring -> flag
	k2 := "Widget" + foo.PkgPathological
	p.Symbols[k2] = x                // "PkgPath" only as a substring -> flag
}
`
	if got := countCheck(lintString(src), "symkey"); got != 2 {
		t.Fatalf("expected 2 findings (substring matches must not clear), got %d:\n%s",
			got, strings.Join(lintString(src), "\n"))
	}
}
