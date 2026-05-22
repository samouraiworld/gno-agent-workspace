# PR #5706: feat(gnovm): type-driven PkgID stamping for *StructValue at allocation & realm .seal

URL: https://github.com/gnolang/gno/pull/5706
Author: jaekwon | Base: master | Files: 67 | +563 -269
Reviewed by: davd-gzl | Model: claude-opus-4-7

Verdict: REQUEST CHANGES — pinned `TestAppHashCrossrealm38` apphash is stale and CI currently red; minor doc/sed artifacts ("borrow rule #2ed", one missed "borrow rule #N borrow" doubling); lint swallow-recover masks unrelated preprocess panics whenever a filetest has `// Error:`. Core stamping change and `realm.seal` mechanism look sound.

## Summary

Three orthogonal changes shipped in one PR. (1) `stampPkgID(oi, t)` gains a Type argument: when `t` is a `/r/`-declared named struct (directly or wrapped in pointer/declared layers), the stamp is the type's home realm; otherwise it falls back to `alloc.currentRealmID`. The only allocator caller passing a non-nil `t` is `NewStruct`; `NewListArray/NewMap/NewSliceFrom*/NewHeapItem/NewBlock/BoundMethodValue/doOpFuncLit/op_exec HIV reinit` all pass `nil` to preserve current-realm stamping for wrappers and anonymous composites. Net effect: a zero-value `var X foreign.T` declaration now mints a `foreign.T`-stamped StructValue, plugging the foreign-typed-placeholder corner from PR [#5669](https://github.com/gnolang/gno/pull/5669#discussion_r3280591967). (2) A dot-named `.seal` no-op method on `gRealmType` whose Gno-side counterpart cannot be declared (parser rejects leading-dot idents), making `realm` only structurally satisfiable by runtime's `.grealm`. (3) `gno lint` now wraps per-filetest `PreprocessFiles` in a per-call `defer recover()` gated on the presence of a `// Error:` directive.

## Glossary

- `stampPkgID(oi, t)` — allocator helper. Now type-driven for /r/-declared named types.
- `getDeclaredPkgID(t)` — walks Pointer/Declared/Struct wrappers; returns the outermost named type's `PkgID` or zero.
- `borrow rule #1/#2/#3` — renamed in this PR from `Layer 1/2/3`. Rule 1 = /r/-declared callable, Rule 2 = /p/ method on real foreign-stamped receiver, Rule 3 = /p/-declared FuncLit closure.
- `checkConstructionTime` — pre-existing gate that panics if a composite literal of an /r/-declared type is constructed outside its home realm.
- `gRealmType` / `.grealm` — uverse-declared `realm` interface and its native impl.

## Fix

Pre-PR, `Alloc.stampPkgID(oi)` always stamped `oi.PkgID = alloc.currentRealmID`. A `var X foreign.T` placeholder allocated in `/r/bar` therefore got `/r/bar`'s stamp despite holding `/r/foreign`-shaped data; field writes from `/r/bar` then passed the `DidUpdate` authority check by coincidence. Post-PR, `stampPkgID(&sv.ObjectInfo, t)` in `NewStruct` ([`alloc.go:582`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/alloc.go#L582)) consults [`getDeclaredPkgID`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/types.go#L2995-L3009); if it's a realm PkgID, that becomes the stamp. `defaultStructValue` ([`values.go:2670-2675`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/values.go#L2670-L2675)) routes through `NewStruct(st, …)` so zero-value defaults pick up the declared stamp automatically. The seal lifts a runtime-only HIV-identity defense (against user types satisfying `realm`) up to preprocess time via an undeclarable method name. The lint change scopes its `// Error:`-gated recover to per-filetest `PreprocessFiles` calls, leaving the outer top-level catchPanic intact for non-filetest paths.

## Critical (must fix)

- **[pinned apphash is stale; CI red]** [`gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:52`](../../../../../.worktrees/gno-review-5706/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52) — `TestAppHashCrossrealm38` fails; observed `daa5545…` vs pinned `77eeee7…`.
  <details><summary>details</summary>

  The pinned multistore commit hash was minted before type-driven stamping landed. With `NewStruct` now stamping declared-realm PkgIDs, the save set for the crossrealm38 scenario changes — exactly what the pin is designed to catch. Reproduced locally:

  ```bash
  # from a gno checkout:
  gh pr checkout 5706 -R gnolang/gno
  go test ./gno.land/pkg/sdk/vm/ -run TestAppHashCrossrealm38 -v 2>&1 | tail -10
  # expected: 77eeee7c455c8c8e99c4d27825ab52912125a2fb22ce20f8110b49e5b07277fd
  # actual:   daa554529ce43b80c3dedd658de8fe787e0a057e60aa01c897364dd80f9dfc65
  ```

  The pin's own docstring tells the author the right move ([`apphash_crossrealm38_test.go:48-51`](../../../../../.worktrees/gno-review-5706/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L48-L51)): "verify the change is actually consensus-breaking before updating this constant — re-run the zrealm_crossrealm38.gno filetest and inspect the save-set diff first." Since the whole point of this PR is to shift which objects enter the save set (foreign-typed zero-values are now stamped to declared realm and therefore become reachable as objects under that realm), a consensus-breaking change is *expected*. Fix: re-run the crossrealm38 filetest and inspect the opslog diff to confirm only intended save-set changes (no incidental shifts from /p/ or stdlib paths), then update the pin to `daa5545…`. Land alongside any chain-upgrade gating note the migration policy requires.
  </details>

## Warnings (should fix)

- **[lint silently drops filetests after // Error: gate]** [`gnovm/cmd/gno/lint.go:344-351`](../../../../../.worktrees/gno-review-5706/gnovm/cmd/gno/lint.go#L344-L351) — any preprocess panic is swallowed when `// Error:` is present, including unrelated regressions.
  <details><summary>details</summary>

  The IIFE wraps `PreprocessFiles` + `AddFileTest` and recovers from any panic when `expectsErr == true`. That's broader than the PR body implies — the swallow is panic-any, not panic-matches-Error-directive. If a future change makes preprocess panic for a different reason (e.g., a regression in type unification) inside a filetest that asserts an unrelated runtime `// Error:`, lint will silently stop processing that file. `ppkg.AddFileTest` is never called for the swallowed case, so downstream lint passes don't see the file either. Exact-message comparison "belongs to `gno test`," fine — but lint could at least record the recovered value and emit a `// Error:` debug breadcrumb when `-v` is set, or restrict the recover to filetests where `// PreprocessorError:` (or similar marker) is present. Fix: narrow the recover to only filetests known to fail preprocess (e.g., a separate `// PreprocessError:` directive, or check that the recovered value matches a preprocess-error type), and emit a diagnostic in verbose mode so swallowed panics are observable.
  </details>

- **[sed artifact: "borrow rule #2ed" / "borrow rule #1ed" in docs and examples]** [`examples/gno.land/r/tests/vm/launderrvictim/launderrvictim.gno:60`](../../../../../.worktrees/gno-review-5706/examples/gno.land/r/tests/vm/launderrvictim/launderrvictim.gno#L60), [`gno.land/pkg/integration/testdata/interrealm_v2.txtar:513`](../../../../../.worktrees/gno-review-5706/gno.land/pkg/integration/testdata/interrealm_v2.txtar#L513), [`docs/resources/gno-interrealm-v2.md:287`](../../../../../.worktrees/gno-review-5706/docs/resources/gno-interrealm-v2.md#L287) — three "Layer Ned" → "borrow rule #Ned" sites the mechanical pass missed.
  <details><summary>details</summary>

  Three sed bleed-throughs survived the cleanup pass. The original prose used "Layer 2ed" as a verb form ("Layer-2 borrowed"); the rewrite produced ungrammatical "borrow rule #2ed". The commit message for `1e44d3fa0` says doublings were dropped — these three weren't. Fix: replace "borrow rule #1ed" → "Rule-1 borrowed" (or similar grammatical form) and same for #2ed, at the three sites.
  </details>

- **[missed "borrow rule #N borrow" doubling in stdlib]** [`gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno:33`](../../../../../.worktrees/gno-review-5706/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L33) — `// SECURITY: stack-walks. Inside a /p/ helper called via borrow rule #2 borrow` — same redundant-noun doubling the cleanup pass claimed to remove.
  <details><summary>details</summary>

  Commit `1e44d3fa0`'s message: "drops the awkward 'borrow rule #N borrow' doublings that the mechanical rewrite introduced". One survived in stdlib. Stdlib doc strings get baked into user-facing docs (`runtime.CurrentRealm` is canonical) so worth fixing. Fix: `borrow rule #2 borrow` → `borrow rule #2`.
  </details>

- **[ADR section structurally drops borrow rule #1's bullet then narrates around it]** [`gnovm/adr/interrealm_v2.md:31-50`](../../../../../.worktrees/gno-review-5706/gnovm/adr/interrealm_v2.md#L31-L50) — the new prepended section introduces "three borrow rules" then bullets #1, prose for paragraphs, then bullets #2, then #3. Reads like a draft.
  <details><summary>details</summary>

  Quoting: "There are three borrow rules in v2, not one: — Borrow rule #1: …" then 6 paragraphs of explanation/example, then "— Borrow rule #2: …". The narrative for #1 ("Now, /r/ type values can only be constructed from within the /r/ realm in which they are declared … This is borrow rule #1 of three.") works, but ending the rule-1 narrative with another "This is borrow rule #1" — after the reader has been told this twice — and then introducing #2 on the next bullet feels like out-of-order drafting. Also `replaces ... three new borrow rules It does not matter` ([line 30](../../../../../.worktrees/gno-review-5706/gnovm/adr/interrealm_v2.md#L30)) is missing a period. Fix: restructure as three bullets at top, prose after; tighten line 30.
  </details>

- **[dangling reference to PLAN_TYPE_DRIVEN_STAMP.md]** PR body and commit `47e355f7c` body reference a design memo not committed to the repo.
  <details><summary>details</summary>

  The PR description says "Split rule (per design memo PLAN_TYPE_DRIVEN_STAMP.md)" and "See PLAN_TYPE_DRIVEN_STAMP.md for full design + interaction analysis." Grep finds no such file. Either the memo should land alongside (preferred, since it's referenced as load-bearing for the split-rule justification), or the reference should be dropped from commit/PR text. Fix: commit the memo to `gnovm/adr/` or remove the reference.
  </details>

## Nits

- [`gnovm/pkg/gnolang/alloc.go:425-426`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/alloc.go#L425-L426) — `checkConstructionTime`'s docstring says "Zero-value defaults … are not 'construction'" but the same allocator's `stampPkgID` now stamps zero-value defaults with the declared realm via `NewStruct` from `defaultStructValue`. The two docstrings disagree about whether default-zero counts as construction. Fix: harmonize the prose — `checkConstructionTime` doesn't fire on defaults, but `stampPkgID` does stamp them; both behaviors are intended, the comments just need to make that distinction explicit.
- [`gnovm/pkg/gnolang/alloc.go:442-457`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/alloc.go#L442-L457) — `stampPkgID` doc reads "Type-driven stamping makes '/r/realmA-typed values live in /r/realmA' strictly true" but `NewListArray/NewMap` *receive* `t` and pass it through, while `NewSliceFromList/NewSliceFromData/NewHeapItem` pass `nil`. The taxonomy (`wrappers vs anonymous composites`) lands at the call site, not the function itself; the docstring conflates them. Worth saying explicitly: `NewListArray(t)`, `NewMap(t)` pass-through but currently the only realm-driven decision is on struct-shaped types, so non-named array/map types stamp via current. Not a bug — but a careful reader will wonder why two functions take `t`.
- [`gnovm/tests/files/zrealm_crossrealm0.gno:8-11`](../../../../../.worktrees/gno-review-5706/gnovm/tests/files/zrealm_crossrealm0.gno#L8-L11) — the comment "Regression: a foreign-realm-typed package var carries the type's declared PkgID" is misleading — this isn't a regression, it's the new intended behavior. Suggest "Test:" or "Behavior:" instead of "Regression:".

## Missing Tests

- **[no test that .seal cannot be satisfied via embedding/promotion]** [`gnovm/tests/files/zrealm_seal_realm.gno`](../../../../../.worktrees/gno-review-5706/gnovm/tests/files/zrealm_seal_realm.gno) — only structural-implement is tested; embedding bypass (cf. existing `examples/gno.land/p/test/seal/filetests/z_seal_embedding_filetest.gno` discussion) isn't.
  <details><summary>details</summary>

  Existing seal filetests for the `/p/test/seal` package explicitly document that "the seal is BYPASSABLE via embedding when the implementation type is exported" ([`z_seal_embedding_filetest.gno:11-15`](../../../../../.worktrees/gno-review-5706/examples/gno.land/p/test/seal/filetests/z_seal_embedding_filetest.gno#L11-L15)). `.grealm` is an internal type — user code cannot reference it by name to embed it. But interface-value embedding (the case `z_seal_iface_embedding_filetest.gno` covers) might still leak: if a user struct embeds a `realm` interface field, does it promote `.seal`? Worth a filetest pinning the answer. Fix: add a filetest with `type fakeRealm struct { realm }` and confirm whether `var _ realm = fakeRealm{}` passes or fails preprocess.
  </details>

- **[no test for "throw an Error: directive at a non-preprocess panic"]** [`gnovm/cmd/gno/lint.go:345-351`](../../../../../.worktrees/gno-review-5706/gnovm/cmd/gno/lint.go#L345-L351) — the new recover swallows any panic when `// Error:` is present.
  <details><summary>details</summary>

  No test asserts the boundary: "lint passes a filetest with `// Error:` declaring a runtime error, when the preprocess passes." Or the inverse: "lint fails when the preprocess panics for a reason unrelated to the declared `// Error:`." Without these, the recover could mask future regressions. Fix: add a test in `gnovm/cmd/gno/lint_test.go` (or wherever `TestLintApp` lives) that pins both directions.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/alloc.go:582`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/alloc.go#L582) — Consider an assertion in `NewStruct` that `t != nil` when called from the doOpStructLit path (every composite literal has a type) and document that `NewStruct(nil, …)` is reserved for internal copy paths. Currently the only `NewStruct(nil)` caller would be `*StructValue.Copy(t, …)` ([`values.go:474`](../../../../../.worktrees/gno-review-5706/gnovm/pkg/gnolang/values.go#L474)) which then immediately re-stamps if the source had a realm PkgID. Belt-and-suspenders.
- [`gnovm/adr/interrealm_v2.md:154-159`](../../../../../.worktrees/gno-review-5706/gnovm/adr/interrealm_v2.md#L154-L159) — The new "Foreign Type Value Caveat" examples are excellent; the table format would render even more clearly than the four side-by-side code blocks. Optional.

## Questions for Author

- Apphash pin update: do you intend to update `expectedCrossrealm38Hash` in this PR or in a follow-up?  The pin's docstring says "verify is consensus-breaking" first — the type-stamp change *is* consensus-breaking by design (foreign-typed zero-values now stamp declared realm), so an update inside this PR seems right; just confirming.
- Is the `// Error:`-gated recover in lint meant to be temporary scaffolding until preprocess errors can be returned as values (rather than panics), or the long-term design?
- The PR body says `tests.gno` "added `NewTestRealmObject()` (non-crossing)" — adding a public constructor to a widely-used shared `/r/tests/vm` package surface is a low-key API addition. Any concern that other downstream filetests will start using it inadvertently?
