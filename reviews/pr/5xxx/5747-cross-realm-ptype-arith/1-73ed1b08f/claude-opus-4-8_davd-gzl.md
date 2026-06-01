# PR #5747: fix(gnovm): cross-realm /p/-type arithmetic (gh#5736)

URL: https://github.com/gnolang/gno/pull/5747
Author: moul | Base: master | Files: 15 | +131 -82
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `73ed1b08f` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5747 73ed1b08f`

**Verdict: APPROVE** — fix is load-bearing and verified (master panics with the exact issue error, PR passes); the removed Copy-time hardening was provably non-load-bearing (syntactically routable via `copy()`), and the real cross-realm-write defense at the pointer-deref / conversion boundary is untouched and still blocks every victim-reaching write. Two non-blocking asks: the `/r/`-declared **array** retention branch is untested ([values.go:347-349](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L347-L349) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L347-L349) is uncovered), and the safety rests on an unguarded deep-copy-independence invariant worth a regression test. No ADR despite "non-trivial AI-assisted" repo rule.

## Summary

A `/r/` realm calling a helper that does in-place `uint256`-style arithmetic on a `/p/`-typed value handed in from another `/r/` realm panicked with `cannot directly modify readonly tainted object` — this hit the GnoSwap AMM math path (any realm calling `common.TickMathGetTickAtSqrtRatio`). Root cause: two Copy-time mechanisms re-stamped a freshly-allocated destination with the source's foreign `/r/` PkgID (`{Array,Struct}Value.Copy`) and carried the sticky readonly bit across (`TypedValue.Copy`'s `cp.N = tv.N`), so a local `*z = *x` value-copy inherited foreign taint. The PR makes Copy stamping type-driven (mirroring [#5706](https://github.com/gnolang/gno/pull/5706)'s `stampPkgID` split rule) and drops the `N_Readonly` carry on deep copies. The load-bearing observation (Morgan's): the same semantic write via `copy(dst, src)` never went through these paths, so the hardening was a syntactic speed-bump, not a defense.

```
value-copy of inlined array/struct  →  DEEP, independent fresh object  →  mutable locally, zero victim impact  ✅ now allowed
write through pointer/slice/field    →  aliases foreign object         →  PkgID gate at PopAsPointer2/doOpConvert  🔒 still blocked
```

## Glossary

- `N_Readonly` — sticky per-`TypedValue` bit meaning "this TV observes external/foreign state"; one of two arms of `IsReadonlyBy`.
- PkgID gate — the other arm: `tvoid.PkgID != m.Realm.ID` in [`IsReadonlyBy`](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/ownership.go#L530-L533) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/ownership.go#L530-L533). Fires on the *real* object's owning realm.
- `getDeclaredPkgID(t)` — walks Pointer/Declared/Struct wrappers to the named type's home-realm PkgID; zero for unnamed composites.
- `PopAsPointer2` → `IsReadonly` — the LHS-pointer evaluation gate for every assignment (`*p=`, `s[i]=`, `x.F=`). Where the real defense lives.
- borrow rule #1 — calling an `/r/`-declared function shifts `m.Realm` to that function's realm for its duration (why `common`'s helper runs under `common`'s authority).

## Fix

Two edits to [`gnovm/pkg/gnolang/values.go`](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go). (1) `{Array,Struct}Value.Copy` swap `if av.ObjectInfo.ID.PkgID.IsRealmPkg()` (runtime source stamp) for `if pid := getDeclaredPkgID(t); pid.IsRealmPkg()` ([values.go:347](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L347) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L347), [values.go:483](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L483) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L483)): `/r/`-declared types still inherit the type's `/r/` owner; everything else keeps the fresh `currentRealmID` stamp from `NewListArray`/`NewStruct`, so a `/p/`-typed copy belongs to the realm doing the copying. (2) `TypedValue.Copy` drops `cp.N = tv.N` for the `*ArrayValue`/`*StructValue` cases ([values.go:1095-1108](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L1095-L1108) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L1095-L1108)). The reference-type cases (`*SliceValue`, `*MapValue`, `PointerValue`) fall to `default: cp = tv`, which still preserves `N` and shares the foreign-stamped Base — so aliased writes remain caught.

## Why it's safe (the master invariant)

The deep-copy-independence argument, verified by reading every branch of `StructValue.Copy`/`ArrayValue.Copy`/`TypedValue.Copy`:

- A value-copy of an **inlined** array/struct produces a genuinely fresh, alias-free heap object (`NewStruct`/`NewListArray` + per-field `field.Copy`). Mutating it cannot affect any other object, real or unreal — so dropping `N_Readonly` and re-owning it to the copier is harmless.
- **Reference** fields (slice/map/pointer) hit `default: cp = tv`: the copy shares the original Base and *keeps* `N_Readonly`. Any write through them (`copy.Slice[i]=`, `*copy.Ptr=`, `copy.Ptr.F=`) routes to the foreign-stamped real object and is rejected by the PkgID gate at `PopAsPointer2 → IsReadonly`.
- Every victim-reaching vector — `*foreignPtr = …`, `&s[0]` write, `victim.Inner = localCopy`, pointer-conversion type-pun — is gated at the deref boundary or [`doOpConvert` Case 1](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/op_expressions.go#L751-L763) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/op_expressions.go#L751-L763), both keyed on `m.IsReadonly` (PkgID gate), both untouched by this PR.

Confirmed against the updated launder filetests: cases 3/4/5 in [`zrealm_launder_rdata_conv_val_then_addr.gno`](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/tests/files/zrealm_launder_rdata_conv_val_then_addr.gno#L78-L82) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/tests/files/zrealm_launder_rdata_conv_val_then_addr.gno#L78-L82) (`&s[0]`, `*victimPtr=`, `victim.Inner=localCopy`) stay `blocked: true`; only the no-victim-impact local-copy cases flip to `false`. `conv_val_pun` now allows the value-conversion but `victim slice[0]` reads unchanged (`slice0`).

## Verification

All run at `73ed1b08f` in the review worktree.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5747 -R gnolang/gno
go test -run 'TestFiles/zrealm' ./gnovm/pkg/gnolang/                          # ok — launder + crossrealm filetests
go test -run 'TestTestdata/addpkg_private' ./gno.land/pkg/integration/        # ok — panic shifted layer
go test -run 'AppHash' ./gno.land/pkg/sdk/vm/                                 # ok — no consensus-pin breakage
```

```
ok  github.com/gnolang/gno/gnovm/pkg/gnolang        3.775s
ok  github.com/gnolang/gno/gno.land/pkg/integration 7.112s
ok  github.com/gnolang/gno/gno.land/pkg/sdk/vm      6.524s
```

Load-bearing check — reverted `values.go` to master, rebuilt, re-ran the new fixture:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5747 -R gnolang/gno
git checkout origin/master -- gnovm/pkg/gnolang/values.go
go build -o /tmp/gno_master ./gnovm/cmd/gno
( cd examples/gno.land/r/tests/issue5736_bar && GNOROOT=$PWD/../../../../../.. /tmp/gno_master test -v . )
git checkout HEAD -- gnovm/pkg/gnolang/values.go
```

```
panic: cannot directly modify readonly tainted object (use a method or crossing function): z<VPBlock(1,0)>.arr[(const (0 int))]
    gno.land/p/onbloc/uint256/bitwise.gno ...
FAIL: 0 build errors, 1 test errors
```

The exact issue panic without the patch; passes with it. Fix is doing the work.

## Critical (must fix)
None.

## Warnings (should fix)

- **[safety rests on an unguarded invariant]** [`gnovm/pkg/gnolang/values.go:1095-1108`](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L1095-L1108) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L1095-L1108) — dropping `N_Readonly` is safe *only* because `cv.Copy` is an alias-free deep duplication of inlined value types; a future refactor that shares sub-structure would silently re-open laundering with no test to catch it.
  <details><summary>details</summary>

  The whole security argument reduces to "the deep copy shares nothing mutable with the source." That invariant is implicit — it lives in the interplay between `StructValue.Copy`'s per-field `field.Copy` and `TypedValue.Copy`'s `default: cp = tv` branch (which deliberately *keeps* `N_Readonly` and shares Base for slice/map/pointer fields). Today it holds. If someone later "optimizes" `StructValue.Copy` to shallow-copy inlined arrays, or makes `TypedValue.Copy` handle `*SliceValue` by aliasing, the `N_Readonly` drop becomes a real laundering hole and every existing test still passes. Fix: add a regression filetest that constructs a struct with both an inlined array field and a slice field sourced from a foreign realm, value-copies it locally, and asserts (a) the inlined-array write succeeds with no victim impact and (b) the slice-field write is still `blocked: true`. That pins the alias boundary the safety claim depends on.
  </details>

- **[no ADR for a non-trivial AI-assisted PR]** PR body / commits — `gno/AGENTS.md` states "Every non-trivial AI-assisted PR must include an ADR"; this is explicitly "direct continuation of #5706 / Phase 3" and changes documented interrealm-v2 Copy semantics, yet ships no `gnovm/adr/` entry.
  <details><summary>details</summary>

  #5706 updated `gnovm/adr/interrealm_v2.md`; this PR alters the same model (value-copies produce locally-stamped fresh objects; cross-realm writes gated at the deref boundary, not the copy boundary) but only promises a `docs/resources/gno-interrealm-v2.md` §8 follow-up. The Copy-vs-deref distinction is exactly the kind of assumption an ADR exists to record for future contributors and auditors. Fix: add an ADR (or extend interrealm_v2.md) capturing the "Copy boundary is not a defense layer" decision and the `copy()`-asymmetry rationale, ideally in this PR.
  </details>

## Nits

- [`gnovm/pkg/gnolang/values.go:1106-1107`](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L1106-L1107) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L1106-L1107) — the `*StructValue` case comment is just `// See ArrayValue case above.` The array comment correctly explains *why* the bit is dropped; consider a one-line note here that the struct case additionally relies on `default: cp = tv` preserving `N` for reference-type fields, since that's the non-obvious half a reader checking struct safety will want.
- Commit `9e56b0c` / `73ed1b08` carry `Co-Authored-By: Claude …` and `🤖 Generated with Claude Code`. `gno/AGENTS.md` mandates `Assisted-By` (NOT `Co-Authored-By`) for AI credit. Minor, but it's a written repo rule.

## Missing Tests

- **[/r/-declared array retention branch uncovered]** [`gnovm/pkg/gnolang/values.go:347-349`](https://github.com/gnolang/gno/blob/73ed1b08f/gnovm/pkg/gnolang/values.go#L347-L349) · [↗](../../../../../.worktrees/gno-review-5747/gnovm/pkg/gnolang/values.go#L347-L349) — no test exercises `ArrayValue.Copy`'s `pid.IsRealmPkg()` true-branch; the symmetric `StructValue.Copy` branch is covered.
  <details><summary>details</summary>

  Coverage over the `zrealm` filetest suite shows block `values.go:347.50,349.3` (`cp.ObjectInfo.SetPkgID(pid)` for arrays) with hit count 0, while the struct equivalent at `483.50,485.3` is hit — matching codecov's "1 missing line" on the PR. This is the security-relevant half for `/r/`-declared **array** types: it ensures an `/r/foo`-declared array value-copy is *not* downgraded to the copier's realm (which would change persisted ownership). An `/r/`-declared named array type (`type T [N]U` in an `/r/` realm) is a real shape. Fix: add a filetest that value-copies an `/r/`-declared array value inside a different realm and asserts the copy is still owned by the declaring realm (e.g. a subsequent cross-realm write to it is blocked, mirroring the existing struct-side coverage).

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5747 -R gnolang/gno
  go test -run 'TestFiles/zrealm' -coverprofile=/tmp/cov.out ./gnovm/pkg/gnolang/ >/dev/null
  grep 'values.go:347.50,349.3' /tmp/cov.out   # trailing 0 = uncovered
  ```

  ```
  github.com/gnolang/gno/gnovm/pkg/gnolang/values.go:347.50,349.3 1 0
  ```
  </details>

## Suggestions
None.

## Questions for Author

- The `docs/resources/gno-interrealm-v2.md` §8 update you flagged — land it in this PR or a follow-up? Given it documents the exact Copy-vs-deref boundary this PR establishes, in-PR is safer against drift.
- Confirm the `/r/`-retention correctness relies on `getDeclaredPkgID(tv.T)` always seeing the named (Declared) type at every `Copy` call site — same envelope as #5706's `stampPkgID`. Are there call paths where `tv.T` is the unnamed underlying type for an `/r/`-declared value (would silently drop the `/r/` owner)?
