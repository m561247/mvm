package stdlib

import _ "embed"

//go:generate go run gen_stdzip.go

//go:embed src.zip
var stdZip []byte

// EmbeddedStd returns the Go-module-proxy-format zip bytes for the std
// module snapshot baked into this binary.
func EmbeddedStd() []byte { return stdZip }
