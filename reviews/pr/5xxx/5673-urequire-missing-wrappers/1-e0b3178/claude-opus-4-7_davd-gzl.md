# PR #5673: feat(examples/urequire): add missing uassert wrappers

URL: https://github.com/gnolang/gno/pull/5673
Author: davd-gzl | Base: master | Files: 1 | +68 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `e0b3178` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5673 e0b3178`

**Verdict: APPROVE** — purely additive parity work; six new `urequire` wrappers (`AbortsContains`, `PanicsContains`, `Nil`, `NotNil`, `TypedNil`, `NotTypedNil`) mirror their `uassert` counterparts byte-for-byte in shape, plus doc comments backfilled on pre-existing wrappers; only nit is a self-referential "Related: #5673" line in the PR body.

## Summary
`urequire` is the fail-fast sister of `uassert` — every exported `uassert.X` should have a `urequire.X` wrapper that calls the assertion and invokes `t.FailNow()` on failure. Before this PR, six of `uassert`'s 20 exported functions had no wrapper, leaving consumers to mix `uassert.AbortsContains` (returns bool) with `urequire.*` calls when fail-fast semantics were wanted. The PR adds the six missing wrappers and adds godoc comments to nine pre-existing wrappers that lacked them, bringing the file to full parity with `uassert` and to full godoc coverage.

## Fix
Before: 14 of 20 `uassert` functions had `urequire` wrappers; nine of those 14 had no doc comment. After: all 20 are wrapped following the established `if uassert.X(...) { return }; t.FailNow()` template ([`urequire.gno:78-84`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/urequire/v0/urequire.gno#L78-L84) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/urequire/v0/urequire.gno#L78-L84), [`urequire.gno:111-117`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/urequire/v0/urequire.gno#L111-L117) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/urequire/v0/urequire.gno#L111-L117), [`urequire.gno:167-201`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/urequire/v0/urequire.gno#L167-L201) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/urequire/v0/urequire.gno#L167-L201)), and every exported wrapper now has a one-or-two-line godoc that mirrors the source assertion ([`uassert.gno:122-153`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/uassert/v0/uassert.gno#L122-L153) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/uassert/v0/uassert.gno#L122-L153), [`uassert.gno:634-671`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/uassert/v0/uassert.gno#L634-L671) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/uassert/v0/uassert.gno#L634-L671)). Signatures, parameter names (`substr`, `value`, `obj`), and call shapes are identical to source; the only difference is the wrapper-level `t.Helper()` + `t.FailNow()` envelope, consistent with every other wrapper in the file.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`PR body`](https://github.com/gnolang/gno/pull/5673) — "Related: #5673" self-references this PR; either drop the line or replace with the actual related issue/PR (likely a copy-paste artifact from a template).
- [`urequire.gno:2`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/urequire/v0/urequire.gno#L2) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/urequire/v0/urequire.gno#L2) — pre-existing `XXX: codegen the package` is now closer to being achievable since the wrappers are mechanically uniform; worth filing a follow-up issue (out of scope for this PR).

## Missing Tests
- [`urequire_test.gno:5-10`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/urequire/v0/urequire_test.gno#L5-L10) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/urequire/v0/urequire_test.gno#L5-L10) — `TestPackage` already acknowledges that `t.FailNow()` behavior is not exercised; the new wrappers inherit this gap but don't widen it. Out of scope to fix here; flagged as pre-existing.

## Suggestions
- [`urequire.gno:7`](https://github.com/gnolang/gno/blob/e0b3178/examples/gno.land/p/nt/urequire/v0/urequire.gno#L7) · [↗](../../../../../.worktrees/gno-review-5673/examples/gno.land/p/nt/urequire/v0/urequire.gno#L7) — the `type TestingT = uassert.TestingT // XXX: bug, should work` line is unchanged. If the upstream alias bug is now fixed in master, an alias would let consumers drop the `uassert.` qualifier in their own helpers. Out of scope; would be a separate follow-up.

## Questions for Author
- Was "Related: #5673" meant to reference a tracking issue, or is it a stray line? (No issue #5673 exists.)
