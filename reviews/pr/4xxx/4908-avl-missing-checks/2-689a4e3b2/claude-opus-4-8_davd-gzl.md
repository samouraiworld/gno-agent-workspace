# PR #4908: fix(avl): add missing checks in avl package

URL: https://github.com/gnolang/gno/pull/4908
Author: davd-gzl | Base: master | Files: 3 | +38 -8
Reviewed by: davd-gzl (AI agent, claude-opus-4-8) | Model: claude-opus-4-8 | Commit: `689a4e3b2` (stale — +17 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4908 689a4e3b2`

**Verdict: APPROVE** — round-2 follow-up; the latest commit `689a4e3b2` resolves all three of [@thehowl](https://github.com/gnolang/gno/pull/4908#discussion_r-2026-06-05)'s review points (negative-offset truncation, unreachable overflow guards, dead pager branch). Tests pass, CI green, two maintainer approvals (notJoon, thehowl). One disclosure note carries over: this review is authored by the PR author via AI agent.

## Summary
Bounds-checking pass on `p/nt/avl/v0` triaged from issue [#4440](https://github.com/gnolang/gno/issues/4440). Net of the whole PR: `GetByIndex` panics upfront on negative index, `TraverseByOffset` clamps negative offset to 0 (the one path that produced silently wrong results, not a panic), and `GetPageWithSize` panics on `pageSize <= 0` with the now-redundant `pageSize < 1` branch deleted. Round 1 (`35065d6b6`) flagged three things; the new commit fixes each.

## What changed since round 1 (`35065d6b6` → `689a4e3b2`)

| Round-1 finding | Status |
| --- | --- |
| `TraverseByOffset` silently truncates on negative offset | Fixed: clamp `offset < 0 → 0` at [`node.gno:388-392`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/node.gno#L388-L392) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L388-L392) + regression test |
| `calcHeightAndSize` overflow guards fire too late / unreachable | Fixed: both guards removed; hot path back to two lines at [`node.gno:270-273`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/node.gno#L270-L273) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L270-L273) |
| `pageSize < 1` branch dead after the new `pageSize <= 0` panic | Fixed: dead branch deleted, panic kept at [`pager.gno:56-58`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L56-L58) |

The clamp is the correct fix and fully covers the bug: the recursive `traverseByOffset` is reachable only through the exported `TraverseByOffset` (verified: [`node.gno:406`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/node.gno#L406) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L406) and the two `tree.gno` callers at [`tree.gno:105`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/tree.gno#L105) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/tree.gno#L105) / [`tree.gno:116`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/tree.gno#L116) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/tree.gno#L116)), so the clamp runs before any `delta := first.size - offset` can over-count. The pager never reaches this path with a negative offset anyway: `pageNumber < 1` returns early, so `startIndex = (pageNumber-1)*pageSize >= 0`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[disclosure: author = reviewer via AI]** — Carried over from round 1. This review is produced by an AI agent run by the PR author (davd-gzl). Fine as adversarial self-review, but if posted to the PR it must be marked as AI self-review per workspace AGENTS.md. Does not affect the merge decision: the substantive approvals come from notJoon and thehowl.

## Nits
- [`node.gno:111-113`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/node.gno#L111-L113) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node.gno#L111-L113) — policy now mildly inconsistent: `GetByIndex` panics on a negative index while `TraverseByOffset` clamps a negative offset. Both were sanctioned by thehowl (he offered clamp-or-panic for the offset), so this is fine; worth a one-word mention in the PR body that the two guards intentionally differ (index is a precise lookup, offset is a lenient cursor).
- [`pager.gno:57`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L57) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L57) — panic message `"GetPageWithSize: invalid page size"` omits the offending value; `ufmt.Sprintf("...: pageSize must be > 0, got %d", pageSize)` would match the clarity of the `GetByIndex` message. Optional.

## Missing Tests
- [`pager_test.gno`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/pager/pager_test.gno) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/pager/pager_test.gno) — no explicit test asserts the new `pageSize <= 0` panic. The negative-offset fix got a focused regression test ([`node_test.gno:82-103`](https://github.com/gnolang/gno/blob/689a4e3b2/examples/gno.land/p/nt/avl/v0/node_test.gno#L82-L103) · [↗](../../../../../.worktrees/gno-review-4908/examples/gno.land/p/nt/avl/v0/node_test.gno#L82-L103)); the pager panic did not. Low priority since the branch is trivial, but a `defer recover()` assertion would lock the contract against a future `<= 0` → `< 0` slip.

## Suggestions
- ADR no longer warranted. Round 1 suggested one; with the overflow guards dropped and the change reduced to plain bounds checks blessed by two maintainers, this falls under AGENTS.md "skip ADR for trivial bug fixes."

## Questions for Author
None — the round-1 question about why the `TraverseByOffset` guard was removed is now answered by the new commit, which re-adds it as a clamp.
