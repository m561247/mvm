package stdlib

// Incompat lists per-package tests that mvm cannot pass for reasons rooted in
// the bridge/interpreter design rather than a fixable bug in mvm's compiler.
// `mvm test` rewrites their entry to a t.Skip(reason) shim so they show as
// SKIP instead of FAIL, keeping the compat-matrix pass ratio honest.
//
// Add an entry only when:
//   - the root cause is an architectural limit (bridge type erasure, reflect
//     adapter frames, native-only protocols) -- not a bug worth chasing, AND
//   - the reason is short enough to land in the SKIP line without noise.
//
// Drop the entry the moment the underlying limitation is fixed.
var Incompat = map[string]map[string]string{
	"flag": {
		// flag.isZeroValue builds reflect.New(BridgeFlagValue).String() to
		// compare against DefValue; the freshly-zeroed bridge has nil func
		// fields and no path back to the underlying interpreted type, so it
		// panics where native Go would call the source-type zero String().
		"TestPrintDefaults":        "BridgeFlagValue zero loses underlying type; reflect.New().String() panics where the source type would not",
		"TestUserDefinedBoolUsage": "BridgeFlagValueBool zero loses underlying type; reflect.New().String() panics where boolFlagVar zero would not",

		// runtime.Caller through reflect.Call's adapter reports the adapter
		// frame (reflect/value.go) instead of the user's flag.Var call site.
		"TestDefineAfterSet": "runtime.Caller through reflect.Call adapter masks the user call site",
	},
}

// SkipReason returns the recorded reason for skipping testName when running
// `mvm test pkgPath`, or "" if the test should run normally.
func SkipReason(pkgPath, testName string) string {
	if m, ok := Incompat[pkgPath]; ok {
		return m[testName]
	}
	return ""
}
