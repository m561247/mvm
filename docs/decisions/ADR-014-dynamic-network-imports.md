# ADR-014: Dynamic network imports via Go module proxy

**Status:** accepted
**Date:** 2026-05-04

## Context

Up to this point, the parser could resolve imports from two sources:

- The user's `pkgfs` (a local directory tree, typically a checkout).
- The embedded `stdlibfs` for generics-first stdlib packages
  (`cmp`, `slices`, `maps`, `iter`).

Both require the source to be present before evaluation starts. That is
fine for CLI use against a checked-out repo, but blocks two scenarios:

- **WASM embedding.** A browser-hosted interpreter has no filesystem.
  Today it can run only programs whose imports resolve to native
  bindings (`stdlib.Values`) or to the embedded src tree.
- **Ad-hoc evaluation of code that imports third-party packages.** Users
  who want to paste a snippet importing `github.com/google/uuid` need to
  pre-populate `pkgfs` themselves; there is no `go get`-equivalent.

We want imports to "just work" for any module reachable from the public
Go module proxy, with the constraints that nothing is written to disk
(so WASM stays viable, and host machines aren't littered with caches)
and no source-code changes are required in `comp/`, `vm/`, or anywhere
that already speaks to `goparser`.

## Decision

Add a third tier to the parser's FS chain: `pkgfs` -> `stdlibfs` ->
`remotefs`. The new tier is any `fs.FS`; the canonical implementation
is the new `modfs` package, which fetches modules from a Go module
proxy on demand and caches them entirely in memory.

Rationale for the specific choices:

- **Use the Go module proxy protocol** rather than scraping git hosts.
  The proxy gives us a uniform, versioned, immutable URL scheme
  (`/<module>/@latest`, `/<module>/@v/<ver>.zip`) and works for any
  module published to it, including private ones via configurable proxy
  URLs. Vanity-import resolution and git scraping would each need their
  own implementations.
- **Plug in via `fs.FS`, not a new code path.** Every existing parser
  call site that reads imports already speaks `fs.FS`. Adding a third
  fallback in `goparser/import.go` is six lines; everything else (zip
  decompression, HTTP, caching) lives in `modfs/` and is opaque to the
  rest of the system.
- **In-memory only.** No `GOMODCACHE`, no `os` calls in modfs. Module
  zips are decoded into a `map[string][]byte` per module and served
  from there.
- **Synchronous fetch in WASM is acceptable for v1.** `fs.FS.Open` is
  synchronous; in `GOOS=js GOARCH=wasm` builds, `net/http.Get` blocks
  the goroutine but yields to the JS event loop. The first import
  triggers a visible delay; for a v1 this is the right trade against
  the design complexity of an async pre-warm step.

### Resolution strategy

Module path resolution is the awkward part: the parser only sees an
*import* path, but the proxy needs the *module* path. modfs uses a
single mechanism: shortest-first probing over path-prefix candidates
with at least two components, with negative-result caching. The
first prefix the proxy answers 200 for is taken as the module path.

This is the simplest correct algorithm. It costs one extra wasted
probe per first-encounter of a github-style path (probing the bare
`github.com/<user>` segment), but that's a single 404 cached in
`f.missing` for the rest of the session. We deliberately avoided
introducing host shortcuts (`github.com/X/Y`-style heuristics) because
a proper resolver -- driven by `go.mod` `require` directives parsed
out of fetched modules -- will replace this layer entirely (see
[modfs](../modules/modfs.md) TODO). Building one heuristic to retire
it shortly would be churn.

Vanity-import resolution (`<importpath>?go-get=1`, parsing `go-import`
meta tags) is also not implemented. Same reasoning: the planned
`go.mod`-driven resolver subsumes it.

### Versioning

`@latest` always. There is no transitive `go.mod` walking and no
user-facing pinning option: if module `A` imports module `B`, the
parser triggers a separate modfs lookup for `B`, which probes
`@latest` for `B` independently. This means `@latest` may pull a
different version of `B` than `A` was tested against. Pinning is
deferred to a future iteration that parses `go.mod`'s `require` and
`replace` directives (see TODO in [modfs](../modules/modfs.md)); that
work covers reproducibility, version drift, and module-boundary
correctness in one pass, so an interim `Options.Pin` knob would be
throw-away surface area.

## Consequences

**Easier:**

- Users can paste any `import "github.com/foo/bar"` snippet and have
  it work without local setup, given network reachability.
- WASM builds gain access to the full module ecosystem (subject to the
  WASM build constraints of each module).
- The interpreter remains hermetic from the host filesystem; embedders
  can disable network imports by simply not installing a `remotefs`.

**Harder / weaker:**

- No `go.sum` verification, no checksum DB. A compromised proxy can
  serve substituted source. Embedders who care about this must
  configure `Options.Client` with their own verifying transport, or
  point `Options.Proxy` at a trusted proxy.
- Not reproducible: `@latest` results drift over time. The future
  `go.mod`-parsing iteration (see TODO in [modfs](../modules/modfs.md))
  is the planned fix.
- Nested major-version modules
  (`github.com/foo/bar/v2/sub` where `bar` and `bar/v2` are separate
  modules) resolve to the wrong module via shortest-first probing.
  Same future iteration eliminates the heuristic entirely.
- The `modfs.FS.locate` mutex is held across HTTP requests. Fine for a
  single-threaded parser; concurrent embedders need a future migration
  to per-module-path locking (e.g., `singleflight.Group`).
- Module caches grow without bound for the lifetime of the FS. Long-
  running hosts will eventually want eviction.

## Alternatives considered

- **Scrape `github.com` directly via raw URLs.** Simpler URL scheme
  but no version pinning, no canonical zip layout, and no module
  identity. Rejected.
- **Run `go mod download` in a subprocess.** Requires a Go toolchain on
  the host, fails in WASM, and writes to `GOMODCACHE`. Rejected.
- **Pre-fetch all imports synchronously at REPL start.** Would require
  parsing imports up-front before evaluation. Rejected: not actually
  simpler than handling the first-import delay, and breaks REPL
  ergonomics.
