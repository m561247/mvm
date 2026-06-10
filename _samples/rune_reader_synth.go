package main

// An interpreted io.RuneReader passed to a native API (regexp over a
// reader) exercises synth shape S37: ReadRune() (rune, int, error).

import (
	"fmt"
	"io"
	"regexp"
)

type rr struct {
	s   string
	pos int
}

func (r *rr) ReadRune() (rune, int, error) {
	if r.pos >= len(r.s) {
		return 0, 0, io.EOF
	}
	c := rune(r.s[r.pos])
	r.pos++
	return c, 1, nil
}

func main() {
	re := regexp.MustCompile(`b+`)
	loc := re.FindReaderIndex(&rr{s: "aabbbcc"})
	fmt.Println(loc[0], loc[1])
}

// Output:
// 2 5
