# ADR-018: Virtualized process exit via panic-based `ExitError`

**Status:** accepted
**Date:** 2026-05-19

## Context

Interpreted code paths that terminate the process -- `os.Exit`, `log.Fatal*`, native bridges like `testing.Main` -- previously routed straight to the host runtime's `os.Exit`.
That had three concrete costs:

- **REPL.** `os.Exit(0)` typed in the REPL killed the REPL session instead of returning the prompt.
- **Embedders.** Untrusted interpreted code from a `Eval` could terminate the host process; there was no error signal an embedder could catch.
- **`mvm test -stat`.** The recently-added `-stat` summary had to print *before* the test summary, via a `setupStats` once-guard + `InstallStatsExitHook` os.Exit wrapper, because native `testing.Main` ends in `os.Exit(MainStart(...).Run())` and bypasses host defers.

Two earlier alternatives were considered and rejected.

- **`//go:linkname` to replace `syscall.Exit`.** The runtime already publishes `syscall.Exit` via linkname; adding a second is a duplicate-symbol link error.
  `os.Exit` itself cannot be replaced via linkname because it has a body.
- **Assembly-level monkey-patch.** Works on amd64/arm64 but fragile across Go versions and inlining.
  Not viable for a maintained codebase.

## Decision

Replace the `os.Exit` and `log.Fatal*` bindings with stubs that `panic` an `*interp.ExitError`, and surface that panic cleanly through the VM's recover path.

### `interp.ExitError`

```go
type ExitError struct{ Code int }
func (e *ExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }
```

`ExitError` is returned from `i.Eval` (and therefore from `i.Run`) whenever interpreted code's exit path is taken.
Callers use `errors.As` (or a type assertion) to recover the exit code; treating it as a generic error and printing it is fine for embedders that don't care.

### Bindings (`interp/interpreter.go`)

`installExitVirtualization`, called from `patchStdlibOverrides` (which already runs once on first `Eval` after `ImportPackageValues` has populated `i.Packages`):

- `os.Exit(code)` -> `panic(&ExitError{Code: code})`.
- `log.Fatal(args...)` -> `log.Print(args...); panic(&ExitError{Code: 1})`. Same shape for `Fatalf` (`Printf`) and `Fatalln` (`Println`), preserving the configured logger's prefix/flags/Writer.

No-op for packages the embedder did not import (the `ok` guards keep stripped-down bindings working).

### `vm.recoverPanic` shape check (`vm/vm.go`)

`recoverPanic` already returns `*PanicError` for in-VM panics and wraps everything else via `capturePanic` (which adds source snippets and an mvm stack).
`ExitError` should *not* be wrapped -- it's a clean signal, not a crash.
The recovery branch becomes:

```go
if e, ok := r.(error); ok {
    if _, isRuntimeErr := r.(runtime.Error); !isRuntimeErr {
        *err = e
        return
    }
}
*err = m.capturePanic(r)
```

The check is on shape, not concrete type: any `error` value that is not a `runtime.Error` (i.e., not a nil deref / type-assert mismatch / etc.) flows through unwrapped.
This keeps `vm` free of an `interp` dependency, and gives embedders the same hook for any future signal type they define.
Genuine runtime crashes (`runtime.Error`) keep the `capturePanic` path with full mvm diagnostics.

### CLI translation (`main.go`)

```go
func main() {
    if err := dispatch(os.Args[1:]); err != nil {
        var ee *interp.ExitError
        if errors.As(err, &ee) {
            os.Exit(ee.Code)
        }
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

mvm's own internal `os.Exit` calls (the host-side dispatch error path) stay -- they are not interpreted code.

## What this does *not* solve: native `testing.Main`'s os.Exit

`testing.Main` is a host-compiled native function (bridged into the interpreter via `stdlib/ext/testing.go`).
Its body literally is `os.Exit(MainStart(matchStringOnly(matchString), tests, benchmarks, nil, examples).Run())`.
The `os.Exit` there is the host-compiled os.Exit, not the interpreter's binding -- so the (A) virtualization does not intercept it, and `-stat` still has to flush before the driver invocation via a once-guard.

The intended follow-up was to switch the `_testmain` driver from `testing.Main` to `testing.RunTests`, which returns `ok bool` and does not call `os.Exit`.
Investigation found this is **blocked**: `testing.RunTests` reads package-private state -- `cpuList`, `*timeout`, `*count`, `*parallel`, `*match`, `*skip` -- that is only initialized by `testing.M.Run()`'s call to the unexported `parseCpuList()` (after `flag.Parse()`).
Calling `RunTests` without that setup either crashes on nil-flag-pointer deref or silently runs zero tests (the outer `for procs := range cpuList` loop body never executes when `cpuList` is nil).

None of the public testing entry points expose a way to drive that setup without also calling `os.Exit`.
Viable paths, all out of scope for this ADR:

- **Vendor `testing/internal/testdeps`** so mvm can call `testing.MainStart(testdeps.TestDeps{}, ...)` and own the `.Run()` exit code.
  Internal-package rules forbid importing it directly; the vendor copy needs upstream sync as testing evolves.
- **`//go:linkname testing.parseCpuList`** (pull-direction).
  Go 1.24+ rejects pull-direction linknames to stdlib internals unless built with `-ldflags=-checklinkname=0`, which breaks `go install`.
- **Subprocess isolation.** Fork mvm with a child mode that runs `testing.Main`; parent owns the exit code and prints stats.
  Clean, but adds testdata cwd / IO / signal plumbing and doubles load time.

Until one of those is taken, the test driver stays on `testing.Main` and `setupStats` remains a `sync.OnceFunc` that the test path flushes manually before each driver invocation.

## What changes in observable behavior

- Embedded `Eval` calls that hit interpreted `os.Exit` or `log.Fatal*` now return `*interp.ExitError` rather than terminating the host.
- `mvm run` still translates `*interp.ExitError` into a host `os.Exit(code)`, so the user-facing exit code is unchanged.
- REPL: an `os.Exit(0)` typed at the prompt currently returns an error and stays at the prompt (the `Repl` loop continues past `Eval` errors). Honoring the exit by terminating the REPL is a one-liner in `interp.Repl` and a separate change.
- `mvm test -stat` behavior under `testing.Main` is unchanged from before this ADR (stats print before the test summary), pending the deferred (D) work.
- Goroutines: an interpreted `os.Exit` inside a `go func() { ... }()` panics that goroutine with `ExitError`, which mvm's per-goroutine `recoverPanic` surfaces.
  The host process is not terminated by that goroutine alone -- matching native Go's behavior where a panicking goroutine without recover kills the whole process from the *host's* perspective, but here mvm contains it.
  This is a behavior change worth knowing about; concurrency-heavy interpreted programs that relied on `os.Exit` from a worker goroutine to kill the host need to surface the exit code through the main goroutine.

## Files

- `interp/interpreter.go` -- `ExitError`, `installExitVirtualization` (replaces `InstallStatsExitHook`), wired from `patchStdlibOverrides`.
- `vm/vm.go` -- `recoverPanic` shape check.
- `main.go` -- `main()` translates `*interp.ExitError`; `setupStats` keeps the `sync.OnceFunc` flush.
