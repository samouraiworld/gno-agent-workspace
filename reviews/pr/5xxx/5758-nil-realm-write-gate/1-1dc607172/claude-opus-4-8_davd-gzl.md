# PR #5758: fix(gnovm): close the nil-realm cross-realm write hole for /p/ and stdlib

URL: https://github.com/gnolang/gno/pull/5758
Author: jaekwon | Base: master | Files: 18 | +402 -46
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `1dc607172` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5758 1dc607172`

**Verdict: APPROVE** — closes a real cross-realm write-laundering hole; the new /p/+stdlib write model is coherent, the write path stays strict, and all PR-relevant filetests pass. Two non-blocking cleanups: a now-false comment on a duplicate /p/ gate in `DidUpdate` (the `rlm==nil` branch), and a weakened stdlib invariant backstop. Neither affects correctness of the fix.

## Summary

Borrow rule #2/#3 set `m.Realm = nil` when a method or closure stamped `/p/` or stdlib was dispatched, because those packages had no `Realm`. With `m.Realm == nil`, `IsReadonly`/`isExternalRealm` short-circuit to "single-user mode, nothing is readonly" — so a `/p/`- or stdlib-method body could write directly to a *foreign* `/r/`-stamped object with no cross-realm check. That is a laundering vector: attacker code routed through a `/p/`-stamped receiver gains write access to a victim realm's data.

The fix gives `/p/` and stdlib packages a frozen, never-persisted `Realm` so the borrow lands on a non-nil realm and the normal PkgID-mismatch gate fires. `/p/` methods run as their own frozen realm (writes to other realms blocked; own data immutable post-init). Stdlib methods do *not* borrow at all — they keep the caller's realm, so they can write a caller-supplied out-param buffer (`base64.Encode(dst, src)`) but not a third realm's data, while still being allowed to mutate their own transient global state (`math/rand`).

```
before:  /p/-stamped recv  → borrow → m.Realm = nil → gate OFF → write foreign /r/  ✗ HOLE
after:   /p/-stamped recv  → borrow → m.Realm = /p/ frozen realm → PkgID mismatch → REJECT
         stdlib recv       → no borrow → m.Realm = caller /r/ → foreign != caller → REJECT
                                                              → own out-param == caller → ALLOW
```

## Glossary
- **borrow** — `PushFrameCall` shifting `m.Realm` to a callable's realm so the readonly gate is evaluated against the right authority.
- **stamp / PkgID** — every real object records the PkgID of the realm that constructed it (`oid.PkgID`); the gate compares it to `m.Realm.ID`.
- **frozen realm** — a `Realm` whose `IsRealm()` stays false, so it is recreated deterministically on load and never persisted.
- **StageAdd / StageRun** — package deploy/init vs. message execution; `/p/` writes are legal at StageAdd, rejected at StageRun.
- **own-package exemption** — a package reading/copying (and, for stdlib, writing) its own `PkgID`-stamped data regardless of `m.Realm`.

## Fix

`isImmutableLibraryPath` ([`mempackage.go:299`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/mempackage.go#L299) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/mempackage.go#L299)) classifies `/p/`+stdlib (excluding `_test` overlays). `NewPackage` and `fillPackage` now create/recreate a frozen realm for those paths ([`nodes.go:1391`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/nodes.go#L1391) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/nodes.go#L1391), [`store.go:585-592`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/store.go#L585-L592) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/store.go#L585-L592)). Borrow rule #2 skips stdlib receivers, rule #3 skips stdlib closures ([`machine.go:2376`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L2376) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L2376), [`machine.go:2408`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L2408) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L2408)). `IsReadonly` splits into a strict write-guard (own-write exemption for stdlib only) and `isReadonlyForCopy` (read-taint, own-package exemption for all) ([`machine.go:2700-2728`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L2700-L2728) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L2700-L2728)). The five value-read ops switch to the copy variant; every actual mutation site (`PopAsPointer2`, uverse builtins) keeps strict `IsReadonly`/`isExternalRealm`. `maybeFinalize` skips immutable realms so the frozen realm is never persisted ([`op_call.go:442`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/op_call.go#L442) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/op_call.go#L442)). A debug-only `assertBorrowedRealm` tripwire fires if a realm-bearing package is ever borrowed with a nil realm ([`machine.go:80`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L80) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L80)).

The model's security rests on two facts I verified:
- The stdlib own-write exemption keys on `m.Package.PkgID.IsStdlibPkg()`, and `m.Package` is set per-frame to the *executing* function's package ([`machine.go:2286`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L2286) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L2286)). Attacker `/r/` code can never satisfy `IsStdlibPkg()`, and users cannot deploy a stdlib-classified path. The exemption additionally requires `oid.PkgID == m.Package.PkgID`, so a stdlib package can't write another stdlib package's data.
- The read-taint exemption only sets the readonly *flag* on a copied value; writes through any resulting pointer still go through strict `IsReadonly` at `PopAsPointer2` ([`machine.go:2778-2824`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L2778-L2824) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L2778-L2824)).

## Critical (must fix)
None.

## Warnings (should fix)

- **[comment now contradicts the fix; duplicate /p/ gate]** [`realm.go:275-300`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/realm.go#L275-L300) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/realm.go#L275-L300) — the `rlm==nil` branch of `DidUpdate` still carries a `/p/`-immutability gate justified by "m.Realm becomes nil when a method is dispatched on a /p/-stamped receiver via the borrow rule" — the exact behavior this PR removes.
  <details><summary>details</summary>

  This gate was added in #5669 (Phase 3), before `/p/` packages had a realm. After this PR a `/p/`-stamped receiver borrows `m.Realm` to its frozen `/p/` realm (non-nil), so the write lands in the new gate at [`realm.go:343-348`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/realm.go#L343-L348) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/realm.go#L343-L348), not the `rlm==nil` branch. All non-read `DidUpdate` callers pass `m.Realm`, so `rlm==nil` now only happens in single-user mode (`m.Realm==nil`), where reaching this branch with a *real* `/p/`-stamped object in StageRun is hard to construct. The result is two `/p/` gates that look like they guard different cases but the first's stated case can no longer occur. On a security gate, a comment that asserts the opposite of the new invariant is a real audit hazard. Fix: either delete the `rlm==nil` `/p/` gate if you confirm it is now unreachable, or rewrite the comment to describe the actual residual nil-realm path it still covers (and add a filetest that exercises it, so it doesn't silently rot).
  </details>

- **[invariant backstop looser than the guarantee it protects]** [`realm.go:333`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/realm.go#L333) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/realm.go#L333) — the `DidUpdate` external-object backstop now no-ops for *any* stdlib-stamped `po`, but the authoritative pre-check only allows stdlib writes to the executing package's *own* stdlib data.
  <details><summary>details</summary>

  The pre-check exemption is `m.Package.PkgID.IsStdlibPkg() && oid.PkgID == m.Package.PkgID` ([`machine.go:2769`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/machine.go#L2769) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/machine.go#L2769)) — stdlib A may write only A's own data. The `DidUpdate` backstop, whose whole purpose is to panic when a pre-check is missing, accepts `poPkgID.IsStdlibPkg()` for *any* stdlib package. So if a future pre-check regression let stdlib A write stdlib B's object, this detector would silently no-op instead of catching it. The looseness isn't exploitable today (the pre-check is the real gate, and stdlib state is transient/never-persisted), but it weakens the one tripwire meant to catch exactly this class of regression. Fix: tighten to `poPkgID.IsStdlibPkg() && m.Package != nil && poPkgID == m.Package.PkgID`, matching the pre-check.
  </details>

## Nits

- [`op_call.go:439-447`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/op_call.go#L439-L447) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/op_call.go#L439-L447) — `maybeFinalize` comment says "m.Realm==nil only happens for /p/ and stdlib" in the old form; the new code correctly checks `!m.Realm.ID.IsImmutablePkg()` instead. The updated comment is fine, just confirm no remaining prose elsewhere still claims `/p/`/stdlib have a nil realm.

## Missing Tests

- **[regression for the StageAdd /p/ init-write path]** [`realm.go:343`](https://github.com/gnolang/gno/blob/1dc607172/gnovm/pkg/gnolang/realm.go#L343) · [↗](../../../../../.worktrees/gno-review-5758/gnovm/pkg/gnolang/realm.go#L343) — the immutability gate is keyed on `m.Stage == StageRun`, so `/p/` init writes (StageAdd) are exempt. The launder filetests cover the *blocked* StageRun path; none asserts that a `/p/` package's own `init()` mutating its globals still succeeds.
  <details><summary>details</summary>

  This is the load-bearing exemption that keeps every existing `/p/` package deployable. CI integration (deploying `examples/` `/p/` packages) exercises it transitively and is green, but a dedicated filetest — a `/p/` with a non-trivial `init()` writing package-level state, imported and used from `/r/` — would pin the StageAdd-vs-StageRun boundary against accidental tightening. Cheap insurance for a gate whose two stages have opposite outcomes.
  </details>

## Questions for Author

- The reopening of the hole is caught only by a debug-only assert (`assertBorrowedRealm`, `debugAssert` builds). In a production build, any future path that yields a `/p/`/stdlib `PackageValue` with `Realm==nil` silently reopens the laundering hole. I confirmed `NewPackage` and `fillPackage` are the two paths that set the realm — are those provably the only construction routes a runtime-used immutable package can take? If a cheap always-on guard at the four borrow sites is too hot for the call path, is the debug tripwire considered sufficient given the severity of the failure mode?
- `isImmutableLibraryPath` excludes paths by `strings.HasSuffix(pkgPath, "_test")`. Is there any legitimate non-overlay package path that ends in `_test` and *should* carry a frozen realm? (I believe not, given `/p/` and `/r/` `_test` overlays self-exclude and only stdlib dot-free `_test` names slip through `IsStdlib` — confirming intent.)

---

Test notes (ran in worktree at 1dc607172): the 6 new filetests pass; the model's strict-write-path and per-frame `m.Package` invariants verified by reading. Full `TestFiles -test.short` and `-tags debugAssert TestFiles` were run — the only failures (`types/eql_0f0`, `types/or_f0` type-checker message diffs; `zrealm15`, `zrealm17` debugAssert "non-escaped object should not have zero hash") reproduce identically on `origin/master` and are unrelated to this PR. The new `assertBorrowedRealm` does not false-fire.
