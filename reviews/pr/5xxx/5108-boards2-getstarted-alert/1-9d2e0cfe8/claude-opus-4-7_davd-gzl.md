# PR #5108: feat(boards2): boards requirement instruction md alert

URL: https://github.com/gnolang/gno/pull/5108
Author: alexiscolin | Base: master | Files: 100 (intent: 5 source + ~20 filetests) | +11026 -11074
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5108 9d2e0cfe8` (then `gh -R gnolang/gno pr checkout 5108` inside it)

Verdict: CLOSE — superseded by merged [`README.md`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/README.md) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/README.md) (#5135) and [`doc.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/doc.gno) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/doc.gno) (#5636), premise invalidated by removed-username-requirement (#5254), stale 3+ months with unanswered rebase request from [jefft0](https://github.com/gnolang/gno/pull/5108#issuecomment-2848614552).

## Summary

PR adds a `getStartedAlert()` blockquote on most boards2 render pages — boards list, board view, create-board, thread view, create/edit thread, reply, edit reply, flag, repost, invite — instructing readers to get a Gno address, GNOT, and a registered username. Author acknowledged mid-review (Feb 9) that the 3000 GNOT + username requirements only apply to open boards, not the global guidance the alert frames it as. Maintainer [jeronimoalbi](https://github.com/gnolang/gno/pull/5108#discussion_r1942691023) proposed instead writing a realm README — which landed as #5135 four months ago — and #5254 then dropped the username requirement entirely. The PR has not been touched since.

## Fix

Adds `noteAlert(title, msg)` and `getStartedAlert()` helpers in [`render.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/render.gno#L360-L373) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/render.gno#L360-L373); sprinkles `res.Write(getStartedAlert())` into eight render functions across [`render.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/render.gno#L87) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/render.gno#L87), [`render_board.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/render_board.gno#L36) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/render_board.gno#L36), [`render_post.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L316) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L316), [`render_reply.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/render_reply.gno#L179) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/render_reply.gno#L179), [`render_thread.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/render_thread.gno#L42) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/render_thread.gno#L42). Filetest goldens updated to match. The actual code addition is ~24 lines on top of the head commit `9d2e0cfe8` — the 11k +/- the PR view shows is merge-noise from a stale branch that pulled in unrelated commits (#5085, #5076, #5097, etc.) and never rebased past the master force-push.

## Why CLOSE, not REQUEST CHANGES

Three independent reasons, each sufficient on its own:

1. **Superseded.** The functional ask — "tell new users what they need" — is now in [`README.md`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/README.md) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/README.md) (#5135, merged) with a proper "Open Boards Quick Start" section, plus [`doc.gno`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/doc.gno) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/doc.gno) (#5636) for gnoweb's action overview. jeronimoalbi explicitly offered to write that README in the PR thread ([here](https://github.com/gnolang/gno/pull/5108#discussion_r1944122398)) and shipped it. Re-introducing the same content as a noisy banner on every render path is the wrong solution to a solved problem.

2. **Content is factually wrong against current master.** The alert hardcodes "GNOT to pay for transactions" + "registered username" as universal requirements. The username requirement was removed for open boards in #5254 ([commit `ef1836d36`](https://github.com/gnolang/gno/commit/ef1836d36)). The GNOT-balance requirement only gates open-board posting and is configurable via the function added in #5349 — not a flat "you need GNOT" gate. Author acknowledged this on Feb 9: *"I hadn't realised the 3,000 GNOT (and registered username) requirement applied only to open boards, so I agree that hardcoding that in the quick start isn't ideal."* No follow-up commit narrowed the content.

3. **Stale + abandoned.** Last author commit Feb 9. [Force-push on master](https://github.com/gnolang/gno/pull/5108#issuecomment-2848614552) requires a rebase, requested by jefft0 on May 4 — three weeks ago, no response. PR diff against current master shows the branch reverts since-fixed typos (`publicly` → `publically`, extra spaces before `!` / `?`), uses pre-rename globals (`Notice`, `Help`, `RealmLink` instead of `gNotice`, `gHelp`, `gRealmLink`), and uses the old `gno.land/p/nt/mux/v0` import. Rebasing would amount to rewriting the patch.

## Pre-existing review threads (already on the PR, not re-litigated here)

- [jeronimoalbi @ `render.gno:372`](https://github.com/gnolang/gno/pull/5108#discussion_r1942691023): GNOT + username only apply to open boards. Author acknowledged. Not fixed.
- [jefft0 @ PR-level (May 4)](https://github.com/gnolang/gno/pull/5108#issuecomment-2848614552): rebase request after master force-push. Unanswered.

## Critical (must fix)

None — closing the PR is the action; no point sharpening findings on a patch that needs to be discarded.

## Warnings (should fix)

None.

## Nits

None.

## Missing Tests

None.

## Suggestions

- If the author wants to revive the UX goal in a new PR, scope the alert to open-board surfaces only (`renderBoard` when `board.Type == BoardTypeOpen`, the create-thread / reply / repost / flag forms under an open board) and link to the [`README.md`](https://github.com/gnolang/gno/blob/9d2e0cfe8/examples/gno.land/r/gnoland/boards2/v1/README.md) · [↗](../../../../../.worktrees/gno-review-5108/examples/gno.land/r/gnoland/boards2/v1/README.md) "Open Boards Quick Start" section rather than restating its content inline. Drop the username clause entirely (gone since #5254). One short alert with a `Read more →` link beats a maintenance liability on every render path.

## Questions for Author

- Are you willing to close this in favor of a fresh, scoped follow-up? The current branch isn't worth rebasing — every load-bearing file has moved (variable renames, mux import path, route additions, typo fixes) and the alert text needs to be rewritten regardless.
