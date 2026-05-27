# PR #5644: feat(example/bptree): simplify `Get` to return `nil` as "no value"

URL: https://github.com/gnolang/gno/pull/5644
Author: davd-gzl | Base: master | Files: 41 | +198 -217
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `390325740` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5644 390325740`

**Verdict: APPROVE** — mechanical, self-consistent API simplification; CI green, jefft0 already approved; only soft concerns are a stale `// AVL tree` comment in `r/sys/users/users.gno` and a slight test-style inconsistency (some sites use `Has`, others `Get + NotEqual nil`); no correctness regressions found.

## Summary

`BPTree.Get(key)` drops the `(any, bool)` return for plain `any`, matching the still-open `avl` mirror in PR #5314. A missing key returns `nil`; callers needing to distinguish a stored-`nil` value from absence must use `Has`. 41 files migrate mechanically — 31 production sites convert `v, ok := tree.Get(k); if !ok` to either `v := tree.Get(k); if v == nil` (when v is consumed) or `tree.Has(k)` (existence-only) — and tests follow suit. The `bptree.ITree`, `bptree/rotree.IReadOnlyTree`, and `bptree/v0/PLAN.md` `Get` signatures are all updated in lockstep, and the cross-validation test `TestAVLCrossValidation` is rewritten to bridge the asymmetry between `avl.Get → (val, ok)` and `bpt.Get → val` via an extra `bpt.Has(key)` call.

## Glossary
- `bptree` — `examples/gno.land/p/nt/bptree/v0`, the B+ tree implementation whose `Get` is being simplified.
- `rotree` — `examples/gno.land/p/nt/bptree/v0/rotree`, the read-only wrapper that applies an optional `makeEntrySafeFn` before returning values.
- `makeEntrySafeFn` — caller-supplied `func(any) any` that rewrites a stored value before exposure (typically a defensive copy that nulls sensitive fields).
- PR #5314 — the open companion PR doing the same change for `avl`; the rotree-shape choice in this PR matches #5314 exactly.

## Fix

Before: `BPTree.Get` returned `(value, exists)` so a stored `nil` could be distinguished from a missing key. After: it returns `any`, and `Has` is the only way to tell those apart — documented at [`p/nt/bptree/v0/PLAN.md:366-367`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/PLAN.md#L366-L367) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/PLAN.md#L366-L367) and on the `Get` doc itself at [`tree.gno:69`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/tree.gno#L69) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/tree.gno#L69). The `rotree` wrapper at [`rotree.gno:100-108`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/rotree/rotree.gno#L100-L108) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/rotree/rotree.gno#L100-L108) early-returns `nil` when the underlying tree returns `nil`, skipping `getSafeValue` — same shape as PR #5314's avl/rotree, so the two packages stay symmetric.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`examples/gno.land/r/sys/users/users.gno:63-64`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/r/sys/users/users.gno#L63-L64) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/r/sys/users/users.gno#L63-L64) — `makeUserDataSafe`'s trailing comment still references "this AVL tree" and the gone `(exists bool)` shape; both are obsolete (tree is bptree, return is plain `any`). Drop the comment, or rewrite as "deleted entries surface as `Get → nil` because this function intercepts them".
  <details><summary>details</summary>

  The semantics the comment used to warn about ("`(exists bool)` is true even for deleted") are now resolved by the new API — `Get` returns nil for both missing and deleted (intercepted) entries, which is exactly what callers want. The comment now misleads more than it informs.
  </details>

- [`examples/gno.land/p/nt/bptree/v0/tree.gno:65-67`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/tree.gno#L65-L67) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/tree.gno#L65-L67) — the documented `if value, ok := tree.Get("key").(MyType); ok { ... }` pattern is syntactically valid and works for missing keys (`nil.(MyType)` yields `(zero, false)`), but the rest of the PR migrates callers to the `if v := tree.Get(k); v != nil { ... v.(MyType) ... }` two-step. Aligning the doc example with the dominant migration pattern would make the codebase read more consistently.

- [`examples/gno.land/r/sys/validators/v3/cache_test.gno:30,41`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/r/sys/validators/v3/cache_test.gno#L30) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/r/sys/validators/v3/cache_test.gno#L30) and [`r/gnops/valopers/rotate_test.gno:63-64,72-73`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/r/gnops/valopers/rotate_test.gno#L63-L64) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/r/gnops/valopers/rotate_test.gno#L63-L64) use `urequire.NotEqual(t, nil, rawEntry, ...)` to assert presence, while [`r/gnops/valopers/valopers_test.gno:177`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/r/gnops/valopers/valopers_test.gno#L177) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/r/gnops/valopers/valopers_test.gno#L177) migrates the same shape to `urequire.True(t, signingRegistry.Has(...), ...)`. Both work, but the inconsistency is gratuitous — those sites only need existence; `Has` is cleaner and avoids the `(nil, value)` arg order trap (the assertion would silently pass on `nil == nil`).

- [`examples/gno.land/p/nt/bptree/v0/PLAN.md:368-369`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/PLAN.md#L368-L369) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/PLAN.md#L368-L369) — `Remove` semantics still documented as returning `(nil, false)` for a missing key. That hasn't changed (this PR only touches `Get`), but worth a sentence in the PR body or PLAN noting why `Remove` keeps the two-value shape — same reason `list.Get` does (out-of-range vs missing-vs-stored-nil are genuinely distinct, whereas the `Get(key) (val, exists)` distinction collapsed into a single `Has` query).

## Missing Tests

- [`examples/gno.land/p/nt/bptree/v0/rotree/rotree_test.gno`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/rotree/rotree_test.gno) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/rotree/rotree_test.gno) — no explicit test for the stored-nil + non-nil `makeEntrySafeFn` interaction. With the current implementation at [`rotree.gno:102-108`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/rotree/rotree.gno#L102-L108) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/rotree/rotree.gno#L102-L108), `tree.Set(k, nil)` then `roTree.Get(k)` returns `nil` without calling `makeEntrySafeFn` — same shape as PR #5314's avl/rotree, so this is by design, but a pinning test (asserting "stored-nil short-circuits the safe-fn") would lock the contract in.
  <details><summary>details</summary>

  No production caller stores `nil` through a bptree-wrapped `rotree` today (only `p/moul/addrset` does the `Set(k, nil)` pattern, and it's avl-backed and never wrapped in rotree), so the behavior is unobservable. But the asymmetry is real: if a future user wraps a nil-storing bptree and expects `makeEntrySafeFn` to fire for every present key, they'll be quietly wrong. A 6-line test would prevent the silent decay.
  </details>

## Suggestions

- The PR body says "This is a breaking change" but doesn't include the formatted `BREAKING CHANGE:` block jefft0 asked for ([https://github.com/gnolang/gno/pull/5644#issuecomment-3247770818](https://github.com/gnolang/gno/pull/5644#issuecomment-3247770818)). Add it before merge to make the changelog/migration story explicit.

- Worth landing PR #5314 (avl mirror) and PR #5644 (this one) as a coordinated pair so the two ITree implementations don't drift on which one has the simplified signature.

## Questions for Author

- Is the stored-nil + makeEntrySafeFn short-circuit in rotree intentional parity with PR #5314, or just a side effect of the mechanical migration? If intentional, a one-line comment at [`rotree.gno:104`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/p/nt/bptree/v0/rotree/rotree.gno#L104) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/p/nt/bptree/v0/rotree/rotree.gno#L104) saying "stored nil is opaque — makeEntrySafeFn is not invoked" would prevent future "why isn't my safe-fn called?" confusion.

- The pre-existing `AdminRemoveModerator` uses `Set(addr, false)` rather than `Remove(addr)` ([`r/gnoland/blog/admin.gno:43`](https://github.com/gnolang/gno/blob/39032574/examples/gno.land/r/gnoland/blog/admin.gno#L43) · [↗](../../../../../.worktrees/gno-review-5644/examples/gno.land/r/gnoland/blog/admin.gno#L43) FIXME). Migrating `isModerator` to `Has` doesn't change behavior here (`Has` returns true for "removed" mods, same as the old `_, found := Get(...)` did), so this PR introduces no new bug — but the FIXME is now arguably easier to fix in a follow-up since `Remove` is the obvious one-line change. Worth a tracking issue?
