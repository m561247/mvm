package main

import (
	"fmt"
	"runtime"
)

func inner() {
	pcs := make([]uintptr, 8)
	n := runtime.Callers(0, pcs)
	for i := 0; i < n; i++ {
		fn := runtime.FuncForPC(pcs[i])
		if fn == nil {
			fmt.Printf("frame %d: nil func\n", i)
			continue
		}
		file, line := fn.FileLine(pcs[i])
		fmt.Printf("frame %d: %s @ %s:%d\n", i, fn.Name(), file, line)
	}
	fmt.Printf("total frames: %d\n", n)
}

func middle() {
	inner()
}

func main() {
	middle()
}

// Output:
// frame 0: _samples.inner @ modfs/_samples/runtime_callers.go:10
// frame 1: _samples.middle @ modfs/_samples/runtime_callers.go:24
// frame 2: _samples.main @ modfs/_samples/runtime_callers.go:28
// total frames: 3
