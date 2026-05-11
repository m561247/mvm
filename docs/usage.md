# Usage Guide

This guide covers the `mvm` command line tool: its subcommands, execution
tracing, the `trap()` debugger, remote imports, and the environment variables it
reads. For internals see [architecture.md](architecture.md); for embedding mvm in
a Go or C host program see [`examples/`](../examples/).

## Install

```
go install github.com/mvm-sh/mvm@latest
```

Or run it straight from a clone of the repository with `go run .` in place of
`mvm` (all the examples below work either way).

## Commands

| Command   | What it does                                              |
|-----------|-----------------------------------------------------------|
| `run`     | run a Go source file, evaluate an expression, or start the REPL |
| `test`    | run `Test*` functions found in `*_test.go` files          |
| `version` | print the mvm version, Go version, and OS/arch            |
| `help`    | show the command list                                     |

`run` is the default command, so `mvm foo.go` is the same as `mvm run foo.go`.
Use `mvm <command> -h` for the flags of a command.

## run

```
mvm                                 # start the REPL
mvm run _samples/fib.go             # run a Go source file
mvm _samples/fib.go                 # same; "run" is the default
mvm run -e "fmt.Println(1+2)"       # evaluate an inline expression
mvm run -x _samples/fib.go          # run with line tracing (see below)
```

- **Source file.** The file is read and executed. A leading `#!` line (shebang)
  is stripped, so a script can start with `#!/usr/bin/env mvm`.
- **`-e <expr>`.** Evaluates a single Go expression or statement. The stdlib is
  auto-imported in this mode, so `fmt.Println(...)` resolves without an explicit
  `import`.
- **No arguments.** Starts an interactive REPL.
- **`-x`.** Enables execution tracing -- see [Execution tracing](#execution-tracing).

## test

```
mvm test                            # run tests in the current directory
mvm test ./pkg                      # run tests in a local package directory
mvm test github.com/google/uuid     # fetch a remote module and run its tests
mvm test ./pkg -v                   # verbose output
mvm test ./pkg -run TestFoo         # run only matching tests
mvm test -v                         # current directory, verbose
```

The target is either:

- **A local directory** (default `.`). Every `*.go` file in the directory is
  loaded; there must be at least one `*_test.go` file.
- **An import path** such as `github.com/google/uuid`. The module is fetched
  through the Go module proxy and held in memory -- see [Remote imports](#remote-imports).
  Its package is loaded as a whole so cross-file references resolve.

Test flags use the same names as `go test` (`-v`, `-run REGEX`, `-count N`,
`-short`, ...); mvm adds the `-test.` prefix `testing.Main` expects before
running. They follow the target (or stand alone when the target is omitted), so
the target, when given, comes first: `mvm test ./pkg -run TestFoo`, not
`mvm test -run TestFoo ./pkg`. Tests run in source-declaration order, not
alphabetical order.

`-x` enables execution tracing here too.

## Execution tracing

The `-x` flag (on `run` and `test`) and the `MVM_TRACE` environment variable
turn on a per-instruction trace printed to stderr. Both accept the same comma-
separated mode tokens:

| Want            | `-x` form                  | `MVM_TRACE` form          |
|-----------------|----------------------------|---------------------------|
| line tracing    | `-x`, `-x=line`            | `MVM_TRACE=1`, `MVM_TRACE=line` |
| bytecode tracing| `-x=op`, `-x=bytecode`     | `MVM_TRACE=op`            |
| both            | `-x=all`, `-x=line,op`     | `MVM_TRACE=all`           |

Tracing has effectively no cost when off: the VM hoists the trace state into a
register and the hot loop checks it with a single compare. See the
[Tracing](architecture.md#tracing) note in the architecture doc.

### Line tracing

One line per executed source line: `+ <file>:<line>: <source text>`. Consecutive
hits at the same position are deduplicated, and the prefix is indented by call
depth.

```
$ mvm run -x _samples/fib.go
+ _samples/fib.go:3: func fib(i int) int {
+ _samples/fib.go:10: func main() {
+ _samples/fib.go:11: 	println(fib(10))
+   _samples/fib.go:4: 	if i < 2 {
+   _samples/fib.go:7: 	return fib(i-2) + fib(i-1)
+     _samples/fib.go:4: 	if i < 2 {
+     _samples/fib.go:7: 	return fib(i-2) + fib(i-1)
...
```

### Bytecode tracing

One line per executed instruction: `+ [ip:.. sp:.. fp:..] [opcode operand] [top
of stack]`, where `ip` is the instruction pointer, `sp` the stack pointer, `fp`
the current frame pointer, and the trailing list is a snapshot of the top stack
slots.

```
$ mvm run -x=op -e "1+2"
+ [ip:0    sp:-1  fp:0  ]  [Push 1          ]  []
+ [ip:1    sp:0   fp:0  ]  [AddIntImm 2     ]  [0:1]
+ [ip:2    sp:0   fp:0  ]  [Exit            ]  [0:3]
```

## Interactive debugger: trap()

`trap()` is a builtin (no import needed). When the VM reaches it, execution
pauses and drops into an interactive prompt on stderr:

```
$ cat /tmp/t.go
package main

func main() {
	x := 1
	trap()
	_ = x
}
$ mvm run /tmp/t.go
trap at ip=7 (/tmp/t.go:5:6)
debug> help
  stack, bt  - dump call stack
  cont, c    - continue execution
  help, h    - show this help
debug> stack
=== Call Stack ===
--- Frame fp=4 ... (main) ---
  ...
debug> cont
```

Commands at the `debug>` prompt:

| Command       | Action               |
|---------------|----------------------|
| `stack`, `bt` | dump the call stack and memory |
| `cont`, `c`   | resume execution     |
| `help`, `h`   | show this list       |

Debug info (symbol names, source positions) is built lazily on the first
`trap()`, so programs that never call it pay nothing. See
[vm.md](modules/vm.md#trap-and-interactive-debug-mode) for the implementation.

## Remote imports

Both `run` and `test` accept import paths instead of local files. mvm resolves
them through the Go module proxy and keeps the sources in memory -- nothing is
written to disk.

The `GOPROXY` environment variable is honored with the usual Go semantics:

- unset or empty: use the default public proxy
- `off` or `direct`: offline; no network fetches (mvm has no direct-VCS path)
- a comma/pipe-separated list: the first URL entry is used as the proxy

```
mvm test github.com/google/uuid       # uses proxy.golang.org by default
GOPROXY=off mvm test ./pkg            # never touch the network
```

## Environment variables

| Variable     | Effect                                                             |
|--------------|--------------------------------------------------------------------|
| `MVM_TRACE`  | enable tracing at startup; same tokens as `-x` (`1`/`line`, `op`/`bytecode`, `all`, comma list) |
| `MVM_DEBUG`  | any non-empty value enables the compiler's data/code dumps         |
| `GOPROXY`    | module proxy used for remote imports (see above)                   |
| `MVMSTD`     | internal: override the path to the embedded standard library source |

## Tips

A static file server in one line, using the bundled stdlib:

```sh
mvm -e 'http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))'
```

`mvm test github.com/google/uuid` is a good sanity check that mvm runs real
third-party code, not just toy programs.

The repository ships [`_samples/`](../_samples/) (Go programs you can run
directly) and [`examples/`](../examples/) (embedding mvm in Go and C host
programs). For how the pipeline works under the hood, read
[architecture.md](architecture.md).
