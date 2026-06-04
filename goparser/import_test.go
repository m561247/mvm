package goparser

import (
	"slices"
	"testing"

	"github.com/mvm-sh/mvm/lang/golang"
)

func TestExtractImports(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "single and grouped",
			src: `package p
import "fmt"
import (
	"go/scanner"
	alias "go/token"
	_ "strings"
)`,
			want: []string{"fmt", "go/scanner", "go/token", "strings"},
		},
		{
			name: "embedded source in raw string is ignored",
			src: `package p
import (
	"fmt"
	"testing"
)
const bsrc = ` + "`" + `
package b
import (
	"a"
	"html/template"
)
` + "`" + `
`,
			want: []string{"fmt", "testing"},
		},
		{
			name: "raw-string single import line ignored",
			src: `package p
import "go/types"
var src = ` + "`" + `import "lib"` + "`" + `
`,
			want: []string{"go/types"},
		},
		{
			name: "import keyword in comments ignored",
			src: `package p
// import "commented/out"
import "regexp" /* import "also/not" */
`,
			want: []string{"regexp"},
		},
		{
			name: "quoted token in block comment not harvested",
			src: `package p
var s = ` + "`" + `import  /* ERROR "8:9" */  // blanks` + "`" + `
import "strings"
`,
			want: []string{"strings"},
		},
	}
	p := NewParser(golang.GoSpec, false)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := p.extractImports(tc.src)
			if !slices.Equal(got, tc.want) {
				t.Errorf("extractImports() = %q, want %q", got, tc.want)
			}
		})
	}
}
