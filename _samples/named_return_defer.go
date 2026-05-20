package main

// A named return modified by a deferred closure must propagate to the caller
// (Go returns 42 here). It currently returns 7: a named return that is
// captured by a closure is not promoted to a heap cell (CellSlot) -- its
// zero-initialization bypasses the assignment path that does the promotion --
// so the closure captures a value snapshot and the write is lost. This also
// breaks the idiomatic `func() (err error) { defer func(){ recover(); err = ...
// }(); ... }` pattern (strings_test TestRepeatCatchesOverflow).

func f() (x int) {
	defer func() { x = 42 }()
	return 7
}

func main() {
	println(f())
}

// Output:
// 42
