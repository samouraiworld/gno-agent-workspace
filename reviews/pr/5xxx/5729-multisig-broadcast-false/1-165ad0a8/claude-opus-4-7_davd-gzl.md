# PR #5729: fix: interact-with-gnokey multisig needs -broadcast=false

URL: https://github.com/gnolang/gno/pull/5729
Author: jefft0 | Base: master | Files: 1 | +1 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `165ad0a8` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5729 165ad0a8`

**Verdict: APPROVE** — one-line doc fix completing the migration started in [#4965](https://github.com/gnolang/gno/pull/4965); without it the multisig walkthrough is unrunnable from step 2 onward.

## Summary

PR [#4965](https://github.com/gnolang/gno/pull/4965) flipped `gnokey maketx -broadcast` to default `true`, then walked every example in the docs and the txtar suite to either drop the now-redundant `-broadcast` or add `-broadcast=false` where the caller actually wanted the JSON. The multisig section of [`docs/users/interact-with-gnokey.md`](https://github.com/gnolang/gno/blob/165ad0a8/docs/users/interact-with-gnokey.md) · [↗](../../../../../.worktrees/gno-review-5729/docs/users/interact-with-gnokey.md) was missed: step 2 creates the unsigned tx with `gnokey maketx send ... > "$TX_PAYLOAD"`, which with the new default tries to broadcast an unsigned tx instead of writing JSON to the file. This PR adds the missing `-broadcast=false` on that one line ([`docs/users/interact-with-gnokey.md:824`](https://github.com/gnolang/gno/blob/165ad0a8/docs/users/interact-with-gnokey.md#L824) · [↗](../../../../../.worktrees/gno-review-5729/docs/users/interact-with-gnokey.md#L824)).

## Fix

Before: `gnokey maketx send ... -to <addr> multisig-abc > "$TX_PAYLOAD"` — with `-broadcast=true` default, this would attempt to broadcast and `$TX_PAYLOAD` would not contain a usable unsigned-tx JSON, breaking every subsequent step (`gnokey sign --tx-path "$TX_PAYLOAD" ...`). After: `-broadcast=false` is inserted before the positional `multisig-abc` argument, restoring the prior behavior — write the unsigned tx JSON to stdout so the file is suitable for distributing to signers. Same flag-before-positional ordering as the existing example at [`docs/users/interact-with-gnokey.md:626-633`](https://github.com/gnolang/gno/blob/165ad0a8/docs/users/interact-with-gnokey.md#L626-L633) · [↗](../../../../../.worktrees/gno-review-5729/docs/users/interact-with-gnokey.md#L626-L633).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`docs/users/interact-with-gnokey.md:824`](https://github.com/gnolang/gno/blob/165ad0a8/docs/users/interact-with-gnokey.md#L824) · [↗](../../../../../.worktrees/gno-review-5729/docs/users/interact-with-gnokey.md#L824) — line is now ~210 chars; the rest of the doc uses one-flag-per-line backslash continuation. Wrapping would match local style and make the new `-broadcast=false` visible in a typical 100-col diff view. Out of scope for a one-line fix, but worth a follow-up.

## Missing Tests

None. Docs change, no executable test surface. The walkthrough itself is the test — anyone copy-pasting through the multisig section will hit the failure mode.

## Suggestions

- [`docs/resources/test-halt-height.md:96,132,154,166`](https://github.com/gnolang/gno/blob/165ad0a8/docs/resources/test-halt-height.md#L96) · [↗](../../../../../.worktrees/gno-review-5729/docs/resources/test-halt-height.md#L96) — same migration sweep left behind redundant bare `-broadcast` flags (no `=true`). Functionally fine since true is the new default, but noise. Either a follow-on cleanup or leave alone; not this PR's problem.

## Questions for Author

None.
