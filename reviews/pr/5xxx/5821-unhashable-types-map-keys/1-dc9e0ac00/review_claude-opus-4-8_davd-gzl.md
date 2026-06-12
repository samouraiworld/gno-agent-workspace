# PR #5821: fix(gnovm): recover panics when having unhashable type as map key

URL: https://github.com/gnolang/gno/pull/5821
Author: Villaquiranm | Base: master | Files: 9 | +95 -9
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: dc9e0ac00 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5821 dc9e0ac00`

**TL;DR:** In Gno, putting an uncomparable value (a slice, map, func, or a struct/array containing one) into a `map[interface{}]V` could crash the whole VM with an error that user code could not `recover()` from. This PR turns those crashes into normal catchable panics whose message matches Go's, so a realm can guard against bad keys.

**Verdict: APPROVE** — correct, minimal fix; behavior matches Go for every shape I tested, including the subtle outer-type-vs-leaf-type distinction; all five new filetests and the regression suite pass on the current head (dc9e0ac00). No open concerns.

## Summary

`(*TypedValue).ComputeMapKey` is the hashing entry point for every map key operation (`m[k]`, `m[k]=v`, `delete`). Statically uncomparable key types (`map[[]int]V`) are rejected at preprocess, but when the static key type is an interface the dynamic value can still be uncomparable, and that only surfaces at hash time. Previously two paths went wrong: `*SliceType` panicked with a Gno-specific message (`slice type cannot be used as map key`), and the `default` branch panicked via raw `panic(fmt.Sprintf(...))` for map/func/nested cases. The raw `panic` was a host-level Go panic, not a Gno `&Exception{}`, so user `recover()` could not catch it and the VM exited. The fix adds one early guard: `if !isComparable(tv.T)` panic with a Gno `Exception` carrying Go's exact message `runtime error: hash of unhashable type <type>`, then deletes the `*SliceType` case and converts the `default` branch to the same Exception form as a defensive fallback.

## Glossary

- `&Exception{}` — a Gno-level panic value that user `recover()` can catch, as opposed to a Go `panic(string)` that escapes the interpreter and kills the VM.
- `isComparable(Type)` — structural walk returning whether a type supports `==`; recurses into array elements and struct fields, memoizes structs, returns true for interfaces (their dynamic value is checked later).
- filetest — a `.gno` file under `gnovm/tests/files/` with an `// Output:` or `// Error:` trailer asserting the program's exact output.

## Fix

Before: `ComputeMapKey` reached a per-shape `switch`, where `*SliceType` and `default` carried wrong or uncatchable panics. After: an `isComparable(tv.T)` guard runs first and panics a catchable `Exception` naming the outer dynamic type, so `[1]map[int]int` and `struct{[]int}` report the enclosing type rather than the inner leaf, exactly as Go does. The load-bearing constraint is that `tv.T` at this point is the concrete dynamic type (the value was already unboxed from the interface), which is what makes naming the outer type correct. See [`gnovm/pkg/gnolang/values.go:1586-1594`](https://github.com/gnolang/gno/blob/dc9e0ac00/gnovm/pkg/gnolang/values.go#L1586-L1594) · [↗](../../../../../.worktrees/gno-review-5821/gnovm/pkg/gnolang/values.go#L1586-L1594) and the converted fallback at [`gnovm/pkg/gnolang/values.go:1690-1696`](https://github.com/gnolang/gno/blob/dc9e0ac00/gnovm/pkg/gnolang/values.go#L1690-L1696) · [↗](../../../../../.worktrees/gno-review-5821/gnovm/pkg/gnolang/values.go#L1690-L1696).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/values.go:1690-1696`](https://github.com/gnolang/gno/blob/dc9e0ac00/gnovm/pkg/gnolang/values.go#L1690-L1696) · [↗](../../../../../.worktrees/gno-review-5821/gnovm/pkg/gnolang/values.go#L1690-L1696) — the `default` branch is now unreachable for uncomparable types (the early `isComparable` guard catches them all), so its comment "catches `*SliceType`, `*MapType`, `*FuncType`" describes the guard, not this branch. Reachable types still missing a `case` (e.g. a future comparable composite) would land here with a misleading "unhashable" message, but no such type exists today. Harmless; kept as a belt-and-suspenders fallback.

## Missing Tests

- **[regression guard absent]** [`gnovm/tests/files/types/`](https://github.com/gnolang/gno/blob/dc9e0ac00/gnovm/tests/files/types/) · [↗](../../../../../.worktrees/gno-review-5821/gnovm/tests/files/types/) — the five new filetests all exercise the *failure* path (uncomparable dynamic keys); none asserts that comparable dynamic keys (`int`, `string`, a comparable struct) boxed into `map[interface{}]V` still dedup and look up. A future change to the new `isComparable` guard could silently break the happy path without any filetest noticing.
  <details><summary>details</summary>

  I confirmed the happy path works on the current head (a `map[interface{}]int` with `int`, `string`, and `struct{a,b int}` keys deduped on overwrite and looked up correctly), so this is a coverage gap, not a bug. Low priority: the broader `maplit*`/`range*` suite covers comparable keys at non-interface static types, just not boxed-into-interface ones.
  </details>

## Suggestions

None.

## Open questions

- `isComparable`'s `*ChanType` arm panics with a raw host `panic("channel type is not yet supported")` ([`gnovm/pkg/gnolang/type_check.go:1172-1173`](https://github.com/gnolang/gno/blob/dc9e0ac00/gnovm/pkg/gnolang/type_check.go#L1172-L1173) · [↗](../../../../../.worktrees/gno-review-5821/gnovm/pkg/gnolang/type_check.go#L1172-L1173)) — the same uncatchable-panic class this PR fixes. Unreachable today: channels are rejected at preprocess (`channels are not permitted`), so no channel value can reach a map key. Not posted: predates this PR, no reachable path, out of scope.
