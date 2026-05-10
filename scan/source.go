package scan

import (
	"fmt"
	"strings"
)

// Source describes a source text.
type Source struct {
	Name    string
	Base    int    // base byte offset in the unified position space
	Len     int    // length in bytes
	content string // source text for line/col resolution
}

// Sources is an ordered list of Source entries.
type Sources []Source

// Add registers a new source and returns its base offset.
func (ss *Sources) Add(name, src string) int {
	base := 0
	if n := len(*ss); n > 0 {
		last := (*ss)[n-1]
		base = last.Base + last.Len + 1 // +1 for implicit newline separator
	}
	*ss = append(*ss, Source{Name: name, Base: base, Len: len(src), content: src})
	return base
}

// find locates the source containing pos and returns it with the
// pos-relative local offset. Returns (nil, 0) when pos is out of range.
func (ss Sources) find(pos int) (*Source, int) {
	if len(ss) == 0 || pos < 0 {
		return nil, 0
	}
	i := len(ss) - 1
	for i > 0 && ss[i].Base > pos {
		i--
	}
	s := &ss[i]
	local := pos - s.Base
	if local < 0 || local > s.Len {
		return nil, 0
	}
	return s, local
}

// Resolve converts a global byte offset to (source name, line, col).
// Returns ("", 0, 0) if pos is out of range.
func (ss Sources) Resolve(pos int) (name string, line, col int) {
	s, local := ss.find(pos)
	if s == nil {
		return "", 0, 0
	}
	line, col = lineCol(s.content, local)
	return s.Name, line, col
}

// FormatPos converts a global byte offset to a "[file:]line:col" string.
func (ss Sources) FormatPos(pos int) string {
	name, line, col := ss.Resolve(pos)
	if name == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", name, line, col)
}

// LineText returns the source line containing pos, without the trailing
// newline. Returns "" if pos is out of range.
func (ss Sources) LineText(pos int) string {
	s, local := ss.find(pos)
	if s == nil {
		return ""
	}
	start := strings.LastIndexByte(s.content[:local], '\n') + 1
	end := len(s.content)
	if nl := strings.IndexByte(s.content[local:], '\n'); nl >= 0 {
		end = local + nl
	}
	return strings.TrimRight(s.content[start:end], " \t\r")
}

func lineCol(src string, offset int) (line, col int) {
	offset = min(offset, len(src))
	prefix := src[:offset]
	line = 1 + strings.Count(prefix, "\n")
	col = offset - strings.LastIndex(prefix, "\n")
	return line, col
}
