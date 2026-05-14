package stdlib

import "github.com/mvm-sh/mvm/goparser"

// reflectGenericShim provides interpreted definitions for reflect symbols
// that cannot be expressed as a single reflect.ValueOf binding. Currently:
// reflect.TypeFor[T any]() Type, added in Go 1.22. Body is exactly the
// fallback branch of the upstream implementation (which mvm reaches in all
// cases because we do not optimize the zero-value branch).
const reflectGenericShim = `package reflect

func TypeFor[T any]() Type {
	return TypeOf((*T)(nil)).Elem()
}
`

func init() {
	goparser.RegisterGenericShim("reflect", reflectGenericShim, []string{"Type", "TypeOf"})
}
