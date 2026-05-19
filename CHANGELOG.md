# Changelog

All notable changes to this project are documented in this file. The
format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `interp.ExitError` lets embedders catch interpreted `os.Exit` and
  `log.Fatal*` as a typed error rather than having the host process
  terminate.
  `i.Eval` returns `*interp.ExitError` whenever interpreted code reaches
  an exit path; `errors.As` recovers the exit code.
  The `mvm run` CLI translates the error back into a host `os.Exit(code)`
  so the user-facing exit status is unchanged.
  See ADR-018.

### Changed

- `interp.InstallStatsExitHook` is removed.
  Exit virtualization is now unconditional and wired automatically on
  first `Eval`, so the hook is no longer needed.
- `mvm test -stat` now prints the stats block *after* the test output
  (just before the package-level `PASS`/`FAIL` line) instead of before
  the driver runs.
  The `_testmain` driver wraps each test in a `t.Cleanup` that
  decrements an atomic counter; when the last test completes, mvm
  flushes `-stat` to stderr before native `testing.Main` reaches
  `os.Exit`.
  See ADR-018.

## [0.2.0] - 2026-05-18

### Added

- Dynamic network imports via the Go module proxy. `mvm run` and `mvm
  test` can pull third-party modules on demand, respecting `GOPROXY`
  (including `off` for offline-only operation).
- `mvm test <pkgpath>` runs `Test*` functions from a local directory or
  a remote import path. Accepts `go test`-compatible flags: `-v`,
  `-run`, `-count`, `-short`, etc. With a remote target like
  `github.com/<user>/<repo>`, fetches and runs the third-party test
  suite end-to-end.
- VM execution tracing. Bare `-x` gives a per-line trace, `-x=op` an
  opcode trace, `-x=all` both. The `MVM_TRACE=1` environment variable
  enables tracing from within `Eval`.
- `mvm version` subcommand prints the module version, Go toolchain, and
  OS/architecture.
- Old-style `// +build` build constraints are now supported alongside
  `//go:build`.
- `runtime.Callers`, `runtime.FuncForPC`, and `runtime.CallersFrames`
  are virtualized so interpreted stack frames report proper file:line
  and function names. Stack traces and `runtime/debug.Stack()` now show
  user code, not VM internals.
- `fmt.Formatter` bridge: user types implementing `Format(fmt.State,
  rune)` drive every `%`-verb via their own interpreted code. Unblocks
  `pkg/errors` formatting including `%+v` with stack frames.
- `errors.Is` walks `Unwrap` chains that mix native and interpreted
  error types.
- `reflect.TypeFor[T]()` (Go 1.22+) is provided via a generic shim.
- `reflect.Value.MethodByName` works on mvm `Iface` values.
- Composite-literal struct fields can shadow builtin names
  (e.g. `T{len: 5}` where `len` is the field name).

### Changed

- The standard library now ships as the `github.com/mvm-sh/std`
  synthetic module via the `stdmod` package. Third-party imports and
  stdlib share the same module-resolution pipeline (ADR-017).
- Stdlib bindings track Go 1.26.3. Symbols introduced in Go 1.25+ and
  1.26+ are isolated into build-tagged `*_go12N.go` files so mvm still
  builds against the floor `go 1.24`.
- The cross-package symbol table now uses canonical package-qualified
  keys throughout (Phase 1 + Phase 2 path B refactor). Closes a class
  of bugs where sibling imports with same-named types, funcs, methods,
  vars, or consts would clobber each other. Notable beneficiary:
  `golang.org/x/text` dual-imports of `language` and
  `internal/language`.
- Comments are stripped at the scanner level; the parser no longer
  needs to filter them, retiring a class of "comment leaks into
  Split-loop" bugs.
- The test driver runs `Test*` functions in source-declaration order
  instead of alphabetical order. Matches `go test` behavior and avoids
  order-dependent failures (e.g. uuid's `TestRandPool` exhausting the
  rand pool before `TestRandomUUID`).
- `mvm test` applies dynamic network imports the same way `mvm run`
  does.
- Generated `op_string.go`, `token_string.go`, and `kind_string.go` are
  committed. `make generate` is idempotent; CI runs a periodic full
  regeneration check instead of regenerating stdlib on every commit.
- `mvm test`'s CLI flag layout follows `go test` conventions:
  mvm-specific flags appear before the target, test flags after.

### Fixed

- ActiveMachine concurrency races: multiple goroutines sharing the
  active-machine pointer no longer interleave incorrectly.
- `CallFunc` concurrency follows Go spec rules; per-callback allocation
  snowballing under repeated invocation is gone.
- Many parser edge cases: stray comments inside `var (...)` blocks and
  `switch` bodies; composite literals with non-constant keys,
  parenthesized type conversions in struct field values, and
  array-type keys.
- `(*time.Duration)(d).String()` no longer hangs in infinite recursion.
- Generic instantiation: pointer-type-arg names no longer break the
  mangling guard. Closes a `make generate` hang on `net/http`.
- Struct and array pass-by-value: callee field/index writes no longer
  leak back to the caller's storage. Value-receiver method bodies see
  their own copy of the receiver.
- Bitops: shift truncation, `&^` (and-not), and boolean `Not` now
  produce correct results.
- Numeric conversions are correctly applied in multi-return
  assignments.
- Interface bridging: identical structural types are distinguished
  correctly; pointer-receiver methods are reachable through
  user-defined interfaces; method-set metadata survives cross-package
  type registration.
- `reflect.DeepEqual` and `==` work across the mvm/native value
  boundary, including wrapped types from interface bridges.
- `runtime.Func` sentinel `pc-1` lookups are checkptr-clean: the
  intercept side table is keyed by `uintptr` rather than
  `unsafe.Pointer`.
- Closure naming follows Go's `OuterFunc.funcN` stack-trace convention.
- Mixed keyed/unkeyed slice composite literals (e.g. `[]int{2: 7, 9}`)
  produce the correct length and place each element at the right
  index.
- `runtime.Callers` file paths match between `mvm run` and the
  `interp.TestFile` harness.
- Range over invalid subjects, integer with two iteration variables,
  and channel with two iteration variables now emit clean compile-time
  errors with source locations instead of late VM panics.
- Malformed expressions report a parse error instead of panicking.
- `import "golang.org/x/text/language"` compiles end-to-end. The
  original blocker was a SIGSEGV in `vm.patchRtype`; subsequent fixes
  resolved a chain of cross-pkg resolution failures, reflect-via-mvm
  dispatch gaps, and named-return zero-init issues.

### Performance

- New `CallImmFast` opcode skips `detachByValueArgs` at direct-call
  sites whose callee has no Struct or Array parameter. fib(35) drops
  from 369 to 333 ms/op; similar wins on numeric-call-heavy workloads.
- `runtimeFuncMeta` is now interned per call site, bounding memory at
  `O(distinct call sites)` rather than `O(captures)`. Removes a slow
  leak under repeated `runtime.Callers` use.
- Hot-loop micro-optimizations in `vm.Run`: pointer-based instruction
  fetch, hoisted trace-flag check, and several opcode bodies extracted
  to release register pressure.
- Variable-dependency init-order analysis runs only when its result is
  actually needed.

### Removed

- `comp/dump.go` (dead experiment helper).
- `Symbol.RecvType` Phase-1 cache field. Receiver-type binding in
  Phase 2 method bodies now goes through the unified `symGet`
  qualified-probe path; the cache is obsolete after Phase 2 path B
  step 2.
- Stale `FIXME` in `comp/compiler.go`'s `lang.Range` handler ("handle
  all iterator types"). All official Go range subjects (integer,
  array, slice, string, map, channel, function) have been supported
  for some time.

## [0.1.0] - 2026-05-04

Initial public release. Imported from
[mvertes/parscan](https://github.com/mvertes/parscan) at commit
d7aa040.
