package main

// skip: pre-existing bug. `id, end := parseTag()` inside a closure with
// named return `id` calls addLocalVar(id) which SymAdd-overwrites the
// named-return Symbol with a fresh LocalVar of Type=nil. The compiler
// then crashes in typeSym(nil) when emitting the next assignment to id.
// Go allows := if at least one LHS is new (here `end`); existing names
// must rebind without redeclaration. Fix candidates: in
// goparser/assign.go's define loop, skip addLocalVar when the ident is
// already a LocalVar in the current funcScope; or have addLocalVar
// preserve Type when the name already resolves. Surfaced 2026-05-14
// while exercising reflect.TypeFor[Elem]() in unicode/cldr from the
// internal/language test load.

import "fmt"

type Tag struct{ a int }

func parseTag() (Tag, int) { return Tag{a: 7}, 42 }

func driver(fn func() (id Tag, skip bool)) {
	id, skip := fn()
	fmt.Println(id, skip)
}

func main() {
	driver(func() (id Tag, skip bool) {
		id, end := parseTag()
		_ = end
		id.a += 1
		return id, false
	})
}
