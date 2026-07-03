# PR [#5900](https://github.com/gnolang/gno/pull/5900): perf(gnovm): skip store probe for synthetic package paths

URL: https://github.com/gnolang/gno/pull/5900
Author: omarsy | Base: master | Files: 8 | +36 -20
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 69be9a6ed (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5900 69be9a6ed`

**TL;DR:** Every transaction's preprocessing spins up a throwaway machine under the compiler-internal package path `.dontcare`, and the store used to charge that as a full disk read even though the package can never be on disk. The PR skips that read for dot-prefixed synthetic paths, making every transaction exactly 59,000 gas cheaper with no behavior change.

**Verdict: APPROVE** — the skip is safe: dot-prefixed paths never collide with a valid user or stdlib path, `.uverse` still resolves from cache, `.dontcare` still returns nil, and reverting the change re-adds exactly 59,000 gas per transaction.

## Summary
`GetPackage` classifies a synthetic path (`.uverse`, `.dontcare`) with the new `IsSyntheticPath` predicate and returns from the per-transaction object cache only, skipping the backend read and the `pkgGetter`. The backend read was a guaranteed miss charged as one flat I/O read of 59,000 gas (`ReadCostFlat`, `tm2/pkg/store/types/gas.go:404`), incurred once per transaction because `evalConst` builds a `.dontcare` machine during preprocessing. The `pkgGetter` could never resolve a synthetic path either, so both skipped steps were pure waste. Net effect: every transaction drops exactly 59,000 gas, and the seven affected gas goldens are updated to match.

## Glossary
- gas: metered CPU/memory cost; consensus-relevant, so any change to it is a behavior change.
- MemPackage: in-memory set of a package's source files; the unit loaded, type-checked, run.
- preprocess: the static pass resolving names, types, and blocks before execution; where the `.dontcare` machine is built.
- Store: the backing store for packages and objects (`defaultStore`), layered over a tm2 CommitStore/IAVL.

## Fix
Before, `GetPackage` always ran `GetObjectSafe` (cache then backend) then the `pkgGetter` for any miss. Now, when `IsSyntheticPath(pkgPath)` holds it consults `cacheObjects` only and returns nil on a miss, at [`store.go:365-374`](https://github.com/gnolang/gno/blob/69be9a6ed/gnovm/pkg/gnolang/store.go#L365-L374) · [↗](../../../../../.worktrees/gno-review-5900/gnovm/pkg/gnolang/store.go#L365). The load-bearing constraint is that a dot-prefixed path is never a persisted or getter-resolvable package: `.uverse` is re-installed into the cache every transaction by `ClearObjectCache` ([`store.go:1189`](https://github.com/gnolang/gno/blob/69be9a6ed/gnovm/pkg/gnolang/store.go#L1189) · [↗](../../../../../.worktrees/gno-review-5900/gnovm/pkg/gnolang/store.go#L1189)) and GC-skipped, so it always hits the cache branch; `.dontcare` is never cached or stored, so it always returns nil, exactly as the old backend-miss-then-getter-miss path did.

Verified on 69be9a6ed: reverting the store change makes the `addpkg` golden report `GAS USED: 2815592`, exactly 59,000 above the PR's `2756592`, confirming the fix removes one flat store read per transaction and nothing else.

## Benchmarks / Numbers
| check | value |
|-------|-------|
| flat read cost eliminated per tx | 59,000 gas (`ReadCostFlat`, `tm2/pkg/store/types/gas.go:404`) |
| `addpkg r/hello` gas | 2,815,592 → 2,756,592 (−59,000) |
| `call Hello()` gas | 1,271,011 → 1,212,011 (−59,000) |
| goldens updated | 6 txtar files, each delta exactly −59,000 |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The seven gas goldens fully pin the behavior: the addpkg/call/simulate/restart/gc/determinism flows all assert the post-change gas, and reverting the store change breaks each by exactly 59,000. A dedicated unit test asserting `IsSyntheticPath` classification would duplicate what the goldens already enforce end-to-end.

## Suggestions
None.

## Open questions
- The predicate is a leading-dot prefix, matching the two synthetic paths that exist today (`.uverse`, `.dontcare`). Any future dot-prefixed synthetic path automatically inherits the skip, which is the intended convention (the predicate sits alongside `IsRealmPath`/`IsEphemeralPath` for exactly this reason). No action; noting the coupling for whoever adds a third synthetic path.
