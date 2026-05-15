// The mvm command interprets Go programs.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/modfs"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
	"github.com/mvm-sh/mvm/stdlib/stdmod"
)

// buildModFS builds the modfs the parser uses for both stdlib redirects
// and third-party imports, applying GOPROXY semantics from the Go
// toolchain:
//
//   - unset / empty: use the default public proxy
//   - "off":         disable network fetches (offline-only modfs)
//   - any URL list:  use the first URL entry as the proxy; "direct"/"off"
//     entries fall back to offline since modfs has no direct VCS path
func buildModFS() *modfs.FS {
	p := os.Getenv("GOPROXY")
	if p == "" {
		return modfs.New(modfs.Options{})
	}
	for _, part := range strings.FieldsFunc(p, func(r rune) bool { return r == ',' || r == '|' }) {
		switch strings.TrimSpace(part) {
		case "":
			continue
		case "off", "direct":
			return modfs.New(modfs.Options{Offline: true})
		default:
			return modfs.New(modfs.Options{Proxy: strings.TrimSpace(part)})
		}
	}
	return modfs.New(modfs.Options{Offline: true})
}

func wireFS(i *interp.Interp) {
	mfs := buildModFS()
	if err := mfs.Inject(stdmod.ModulePath, stdmod.Version, stdlib.EmbeddedStd()); err != nil {
		panic("modfs inject embedded std: " + err.Error())
	}
	i.SetStdlibFS(stdmod.FS(mfs))
	i.SetRemoteFS(mfs)
}

// traceFlag is a flag.Value for -x that doubles as a bool flag (-x = line trace)
// and a string-valued flag (-x=op, -x=all, -x=line,op).
type traceFlag struct{ line, op bool }

func (t *traceFlag) IsBoolFlag() bool { return true }

func (t *traceFlag) String() string {
	switch {
	case t.line && t.op:
		return "all"
	case t.line:
		return "line"
	case t.op:
		return "op"
	}
	return ""
}

func (t *traceFlag) Set(s string) error {
	if s == "true" { // bare -x
		t.line = true
		return nil
	}
	line, op := interp.ParseTraceModes(s)
	if !line && !op {
		return fmt.Errorf("unknown trace mode %q (want line, op, all, or comma list)", s)
	}
	t.line, t.op = line, op
	return nil
}

// newlineTracker wraps a writer and tracks whether the last byte written was a newline.
type newlineTracker struct {
	w       io.Writer
	written bool
	last    byte
}

func (t *newlineTracker) Write(p []byte) (int, error) {
	if len(p) > 0 {
		t.written = true
		t.last = p[len(p)-1]
	}
	return t.w.Write(p)
}

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionString() string {
	v, gv := "(devel)", ""
	if bi, ok := debug.ReadBuildInfo(); ok {
		gv = bi.GoVersion
		v = bi.Main.Version
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				v = s.Value
				break
			}
		}
	}
	return fmt.Sprintf("%.12s %s %s/%s", v, gv, runtime.GOOS, runtime.GOARCH)
}

func dispatch(args []string) error {
	if len(args) == 0 {
		return runCmd(nil)
	}
	switch args[0] {
	case "-h", "--help", "help":
		usage(os.Stdout)
		return nil
	case "run":
		return runCmd(args[1:])
	case "test":
		return testCmd(args[1:])
	case "version", "-v", "--version":
		fmt.Println(versionString())
		return nil
	}
	return runCmd(args)
}

const usageText = `Usage: mvm <command> [arguments]

Commands:
  run     run a Go source file, evaluate an expression, or start the REPL
  test    run Go tests in a package directory
  version print the mvm version, OS, and architecture
  help    show this help

Use "mvm <command> -h" for details on a command.
`

func usage(w io.Writer) { _, _ = fmt.Fprint(w, usageText) }

const runUsageText = `Usage: mvm run [options] [path] [args]
Options:
`

func runCmd(arg []string) error {
	var (
		str   string
		trace traceFlag
	)
	rflag := flag.NewFlagSet("run", flag.ContinueOnError)
	rflag.Usage = func() {
		_, _ = fmt.Fprint(os.Stdout, runUsageText)
		rflag.PrintDefaults()
	}
	rflag.StringVar(&str, "e", "", "string to eval")
	rflag.Var(&trace, "x", "trace mode (bare -x = line; -x=op, -x=all, -x=line,op)")
	if err := rflag.Parse(arg); err != nil {
		if err == flag.ErrHelp { // -h already printed usage
			return nil
		}
		return err
	}
	args := rflag.Args()

	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	wireFS(i)
	if trace.line {
		i.SetTracing(true)
	}
	if trace.op {
		i.SetTraceOps(true)
	}

	out := &newlineTracker{w: os.Stdout}
	i.SetIO(os.Stdin, out, os.Stderr)

	var err error
	switch {
	case str != "":
		i.AutoImportPackages()
		_, err = i.Eval(str, str)
	case len(args) == 0:
		i.AutoImportPackages()
		return i.Repl(os.Stdin)
	default:
		fpath := filepath.Clean(args[0])
		var buf []byte
		buf, err = os.ReadFile(fpath)
		if err != nil {
			return err
		}
		src := string(buf)
		if strings.HasPrefix(src, "#!") {
			if nl := strings.IndexByte(src, '\n'); nl >= 0 {
				src = src[nl:]
			} else {
				src = ""
			}
		}
		_, err = i.Eval(fpath, src)
	}
	// Ensure output ends with a newline so the shell prompt is not overwritten.
	if out.written && out.last != '\n' {
		_, _ = fmt.Fprintln(os.Stdout)
	}
	return err
}

const testUsageText = `Usage: mvm test [-x] [target] [test flags]
Runs Go tests found in *_test.go files of the given target.
Target may be a local directory (default ".") or an import path
(e.g. "github.com/google/uuid") fetched dynamically via the Go module proxy.
Test flags use the same names as "go test": -v for verbose output,
-run REGEX to select tests, -count N, -short, etc.
`

func isMvmTestFlag(a string) bool {
	switch a {
	case "-x", "--x", "-h", "-help", "--help":
		return true
	}
	return strings.HasPrefix(a, "-x=") || strings.HasPrefix(a, "--x=")
}

func splitTestArgs(arg []string) (mvmFlags []string, target string, testFlags []string) {
	target = "."
	n := 0
	for n < len(arg) && isMvmTestFlag(arg[n]) {
		n++
	}
	mvmFlags = arg[:n]
	rest := arg[n:]
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		target = rest[0]
		rest = rest[1:]
	}
	return mvmFlags, target, rest
}

func rewriteTestFlags(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		switch {
		case a == "-" || a == "--":
			out[i] = a
		case strings.HasPrefix(a, "--"):
			out[i] = "--test." + a[2:]
		case strings.HasPrefix(a, "-"):
			out[i] = "-test." + a[1:]
		default:
			out[i] = a
		}
	}
	return out
}

func testCmd(arg []string) error {
	var trace traceFlag
	tflag := flag.NewFlagSet("test", flag.ContinueOnError)
	tflag.Usage = func() {
		_, _ = fmt.Fprint(os.Stdout, testUsageText)
		tflag.PrintDefaults()
	}
	tflag.Var(&trace, "x", "trace mode (bare -x = line; -x=op, -x=all, -x=line,op)")

	mvmFlags, target, testFlags := splitTestArgs(arg)
	if err := tflag.Parse(mvmFlags); err != nil {
		if err == flag.ErrHelp { // -h already printed usage
			return nil
		}
		return err
	}

	os.Args = append([]string{"mvm-test"}, rewriteTestFlags(testFlags)...)

	i := interp.NewInterpreter(golang.GoSpec)
	i.ImportPackageValues(stdlib.Values)
	wireFS(i)
	i.AutoImportPackages()
	if trace.line {
		i.SetTracing(true)
	}
	if trace.op {
		i.SetTraceOps(true)
	}
	i.SetIO(os.Stdin, os.Stdout, os.Stderr)

	// Try target as a local directory first; fall back to import-path
	// resolution (modfs / stdlibfs / pkgfs) on miss.
	if absDir, aerr := filepath.Abs(target); aerr == nil {
		if entries, rerr := os.ReadDir(absDir); rerr == nil {
			if err := evalLocalDir(i, absDir, entries); err != nil {
				return err
			}
			return runTestDriver(i)
		}
	}
	i.SetIncludeTests(true)
	if _, err := i.Eval(target, ""); err != nil {
		return fmt.Errorf("loading %q: %w", target, err)
	}
	return runTestDriver(i)
}

func evalLocalDir(i *interp.Interp, absDir string, entries []os.DirEntry) error {
	var paths []string
	hasTest := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		paths = append(paths, filepath.Join(absDir, e.Name()))
		if strings.HasSuffix(e.Name(), "_test.go") {
			hasTest = true
		}
	}
	if !hasTest {
		return fmt.Errorf("no *_test.go files found in %s", absDir)
	}
	for _, p := range paths {
		buf, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if _, err := i.Eval(p, string(buf)); err != nil {
			return err
		}
	}
	return nil
}

func runTestDriver(i *interp.Interp) error {
	testNames := i.FuncNames("Test")
	if len(testNames) == 0 {
		fmt.Fprintln(os.Stderr, "testing: warning: no tests to run")
		return nil
	}

	var driver strings.Builder
	// Pass regexp.MatchString directly rather than wrapping it in an interpreted
	// closure: native testing.Main calls the matcher via reflect for each test
	// (and per slash-separated sub-name) when -run/-skip is set, so wrapping it
	// in `func(pat, name string) (bool, error) { return regexp.MatchString(...) }`
	// makes every match a re-entrant mvm Machine spin-up that copies the host
	// data segment. On large packages (e.g. golang.org/x/text/language) that
	// snowballed into minutes-long hangs and gigabytes of allocations under
	// `mvm test -run=X`. Passing the native func value avoids the bridge.
	driver.WriteString("testing.Main(regexp.MatchString, []testing.InternalTest{")
	for _, name := range testNames {
		fmt.Fprintf(&driver, "{Name: %q, F: %s},", name, name)
	}
	driver.WriteString("}, nil, nil)")
	_, err := i.Eval("_testmain", driver.String())
	return err
}
