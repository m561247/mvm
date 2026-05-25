# ADR-020: Type references resolved by identity slot, not by name

**Status:** accepted
**Date:** 2026-05-25

## Context

`comp.Compiler` embeds `*goparser.Parser` and shares its single, mutable
`symbol.SymMap`. The compiler resolves names flatly: `symAt` checks the
`CompilingPkg`-qualified key, then the bare key -- it does NO scope walk
(see [ADR-003](ADR-003-scope-as-path.md)). The contract is that the parser
resolves a name and bakes the exact resolvable key into the token's `Str`, and
the compiler does a flat lookup.

Under that contract, type references crossed the parser->comp boundary as NAME
strings that the compiler re-resolved against the shared, mutable symbol table.
This couples compile-time type correctness to the parse-time symbol-table STATE.
It is fine for stable names (package types, builtins) but structurally fragile
for transient, instance-specific type identity -- generic type parameters. A
type parameter would have to live in the table by name for the compiler to find
it, yet must not persist (it would collide across instantiations) -- a lifecycle
hazard. Name-substitution monomorphization (see
[ADR-011](ADR-011-generics-monomorphization.md)) copes by baking concrete type
NAMES into the instantiated body, which breaks when a type argument's name is
not resolvable in the defining package (an interpreted or test-local type, e.g.
`errors.AsType[timeout]`).

## Decision

Carry RESOLVED TYPE IDENTITY in the IR instead of a name.

- A type-reference token carries its resolved `*vm.Type` (`Token.ResolvedType`),
  attached by `registerType`, the bare type-ident handler in `parseExpr`, and
  `zeroInitLocals`. Type assertions and type switches already did this.
- The compiler binds a carried type to a shared global type slot via
  `zeroTypeSlot(typ)` -- a `Data` slot holding the type's zero value (the `Fnew`
  source), deduped by `reflect.Type`. It bypasses the symbol-table name lookup.
- The slot is shared by rtype (correct: distinct types with the same rtype have
  the same zero value), but the precise `*vm.Type` rides the pushed compile-stack
  SYMBOL, so method resolution -- which is still name-keyed -- stays exact even
  when two interpreted types share an rtype (placeholder collisions).
- Name-path type idents route their lazy slot allocation through the same
  `zeroTypeSlot`, so a name ident, a carried-type ident, and a composite literal
  all patch one `Fnew` slot.

This makes a type a first-class VM object: a slot in global memory referenced by
identity. `typeSym`/`typeIndex` (type-DESCRIPTOR slots, used by make-elem/key and
`TypeAssert`) are the descriptor counterpart; `zeroTypeSlot` is the zero-value
(`Fnew`) counterpart.

## Consequences

- The compiler no longer re-resolves type NAMES for `make`/`new`/conversions/
  composites/var-zero-init; type correctness is decoupled from the mutable
  symbol table's name state.
- Robust against name shadowing and unresolvable names (interpreted/test-local
  types) -- the foundation for instantiating generics with such type arguments.
- Behavior-preserving on the existing substitution base: no runtime change, a
  pure foundation.
- Methods stay name-keyed (`SymMap.MethodByName` uses the symbol `Name`) -- this
  is inherent (methods are registered by name), so the carried-type symbol keeps
  its `Name` for method lookup.
- A type can occupy two `Data` slots: a zero-value slot (`zeroTypeSlot`, for
  `Fnew`) and a type-descriptor slot (`typeSym`, for make-elem/`TypeAssert`).
  Minor duplication; both are deduped.
- This does not by itself change monomorphization (still name substitution,
  [ADR-011](ADR-011-generics-monomorphization.md)). The intended follow-on is
  instantiation that ATTACHES the resolved `*vm.Type` to type-param idents (no
  name substitution, no symbol-table binding) -- which this IR enables and which
  removes a class of symbol-lifecycle hazards.
