# Invariant catalog

Recurring gnolang/gno bug classes, as a mandatory checklist for PRs touching gno code (loaded by `review.md`, which gates when it applies). Walk every class. For each, the diff either touches it (check, cite a finding only if it's violated) or plainly does not (no output). This is the floor, not a substitute for reading every line.

- Determinism: no dependence on Go map iteration order, `time.Now()`/wall clock, real randomness, goroutine scheduling, floating-point, or pointer/address identity in any state-affecting or output path; no network or filesystem reads on those paths.
- Gas: new VM ops and stdlib calls charge gas; any gas-cost change is consensus-affecting and called out; gas and allocation accounting can't overflow or go negative, and is charged before the work it bills.
- Realm state safety: no partial state writes on a path that can panic mid-update; re-entrancy: state mutated before a cross-realm or external call commits before that call or is guarded, so it can't be re-entered mid-update; object persistence (`saveObject`/`copyValueWithRefs`) and pointer identity hold across transaction boundaries.
- Caller & access control: privileged operations authenticate the caller via the threaded `cur realm` (`cur.Previous().Address()`, `cur.Previous().PkgPath()`); flag auth that instead relies on the stack-walking `chain/runtime/unsafe` primitives (`unsafe.PreviousRealm()`, `unsafe.OriginCaller()`), spoofable (tx.origin phishing, borrow-rule delegation) unless meant for tx-level identity and paired with `runtime.AssertOriginCall()`. Flag too any privileged op with no caller check at all.
- Coin & banker: coin math can't overflow or underflow; no negative or zero-amount sends; banker operations are authorized.
- Storage deposit: per-realm deposit is charged and refunded in proportion to the storage byte delta (`processStorageDeposit`, `RealmStorageDiffs`); `diff*price` can't overflow (`overflow.Mulp`); a release refunds on the realm's original deposit ratio (`rlm.Deposit*released/rlm.Storage`), never the current `StoragePrice`, and never unlocks past `rlm.Deposit` or releases past `rlm.Storage`; `StoragePrice`/`DefaultDeposit` param changes are consensus-affecting.
- Global mutable state & concurrency: no package-level `var` mutated at runtime (lazy caches, toggles, shared allocators); it races once code runs in parallel (the new `gno test -p` path, and the touched package's own `t.Parallel()` unit tests). A var set once via `sync.OnceValue` or only at `init()` and read-only after is safe to read concurrently and is the fix shape, not a finding. CI runs no `-race`, so when a PR adds parallelism or a global cache, run `go test -race` on every Go package the PR touches (including `cmd/...`, not just `pkg/...`) and compare against master.
- Error & panic handling: `.go` returns errors on recoverable failure and panics only on programmer error; `.gno` panics for user-facing failure. No swallowed errors (`_ =` on an error, returned `err` left unchecked); `recover` wraps exactly the intended call, not a wider block.
- VM-fault recoverability: any runtime fault user `.gno` can trigger (index/slice out of range, nil deref, nil-interface method call, uncomparable-type compare, makeslice len/cap) must surface through the VM Exception machinery (`panic(&Exception{...})`, `m.Panic`, `m.pushPanic`) so gno `recover()` catches it; a bare Go panic re-raises past `runOnce` (machine.go) and escapes the VM uncatchable. Assert with a `recover()` filetest, not an `// Error:` directive.
- VM semantics vs Go: any change to gno evaluation (interpreter `op_*.go`, but also `preprocess.go`, `types.go`, `values.go`, `uverse.go`) must keep valid gno evaluating exactly as Go does: assignment/argument order, tuple and multi-assign, struct/blank-field/interface equality, type-switch nil cases, string<->rune/byte conversion, map-key hashability, iota scoping, slice/string backing-array aliasing. Assert parity with an `// Output:` filetest against a side-by-side Go run; divergence is a correctness bug even when nothing faults.
- Type-check & preprocess: new builtins/symbols resolve under `gno lint`; preprocessing rejects what the VM can't run (e.g. illegal recursive types).

## Realm audit patterns

Walk these for a PR that adds or changes a realm, on top of the classes above. Each is a vulnerable/fixed fixture pair plus a text-scan rule in the audit pattern harness, which lives in the unmerged [PR 5835](https://github.com/gnolang/gno/pull/5835) under `misc/audit-pattern-harness/`. Follow-up guidance in [PR 5936](https://github.com/gnolang/gno/pull/5936). Neither path exists on master, so read them from a worktree of that branch.

Run the rules against the fixtures to see each shape, then match the diff by hand:

```bash
# workdir: .worktrees/gno-review-5835/misc/audit-pattern-harness
GNOROOT=<pr-worktree> make run AUDIT_PATTERN_FLAGS='-gno-bin <path-to-gno>'
```

Every rule is a heuristic line scan, which the harness README states outright. A hit is a line to inspect, never a finding: confirm the mechanism against the code and land it in the severity model like any other finding.

| Class | Vulnerable shape | Fix |
|---|---|---|
| `origin-caller-auth` | authorization compares against `unsafe.OriginCaller()`, which hides an intermediate realm caller | compare `cur.Previous().Address()` |
| `current-guard` | `cur.Previous()` read with no `cur.IsCurrent()` earlier in the same function | guard with `if !cur.IsCurrent() { panic(...) }` first |
| `payment-user-call` | `unsafe.OriginSend()` gated on `IsUser()`, which also admits ephemeral run realms | gate on `IsUserCall()` |
| `realm-only-gate` | a realms-only gate written `if x.IsUserCall() { panic }`, which a `maketx run` script walks through | reject on `IsUser()` |
| `callback-param` | a caller-supplied `func()` stored and later invoked with this realm's authority | expose a narrow value setter, take no callback |
| `interface-realm-param` | an interface method declares a `cur realm` parameter, handing a realm capability to an arbitrary implementer | pass inert data (`caller address`) instead |
| `exported-pointer-leak` | exported `var X *T` or an exported getter returning `*T` into package state | return a value copy, keep the var unexported |
| `render-map-iteration` | `Render` ranges over a Go map, so output order is nondeterministic | iterate an `avl.Tree` |
| `render-markdown` | caller-controlled text concatenated into `Render` output | wrap in `md.EscapeText` |

### Caller identity predicates

The three predicates are not interchangeable, and confusing them is a live defect class rather than a theoretical one. From [`frame.gno:78-107`](https://github.com/gnolang/gno/blob/f380c15f7/gnovm/stdlibs/chain/runtime/frame.gno#L78-L107):

- `IsUserCall()` is `pkgPath == ""`. True only for a direct `gnokey maketx call` from an account.
- `IsUserRun()` is true when `pkgPath` equals `<domain>/e/<addr>/run`, the ephemeral realm `gnokey maketx run` executes a script in ([`keeper.go:1018`](https://github.com/gnolang/gno/blob/f380c15f7/gno.land/pkg/sdk/vm/keeper.go#L1018)).
- `IsUser()` is `IsUserCall() || IsUserRun()`.

So a realm that must reject users tests `IsUser()`, and a realm that must accept only a direct user call tests `IsUserCall()`. Rejecting on `IsUserCall()` to mean "the caller is a realm" is wrong: `gnokey maketx run` gives an attacker a non-empty `pkgPath` and walks straight through. That is the Critical in [PR 5976](https://github.com/gnolang/gno/pull/5976). `chain/banker` itself takes the strict side, gating `BankerTypeOriginSend` on `IsUserCall()` ([`banker.gno:96`](https://github.com/gnolang/gno/blob/f380c15f7/gnovm/stdlibs/chain/banker/banker.gno#L96)).

The harness as it stands in PR 5835 has no rule for this direction: `payment-user-call` only fires on an `OriginSend()` with no preceding `IsUserCall()`, so it stays silent on PR 5976. The `realm-only-gate` row above is a local addition sitting uncommitted in the 5835 worktree, not yet offered upstream.

### On the `current-guard` rule

The rule fires on any `.Previous()` not preceded by `.IsCurrent()` in the same function, which matches nearly every realm in `examples/`: a survey during the [PR 5951](https://github.com/gnolang/gno/pull/5951) review found zero of 28 realm files carrying the guard. The preprocessor also rejects any realm value other than `cur` or `cross(rlm)` in that position, and `cross()` validates `IsCurrent` itself. Treat a bare hit as unproven: file it only with a demonstrated path that reaches the read with a forgeable realm value.
