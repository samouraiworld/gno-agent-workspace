# PR #5809: chore(boards2): Make public API non-crossing again, move to main folder

URL: https://github.com/gnolang/gno/pull/5809
Author: jefft0 | Base: master | Files: 39 | +286 -486
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 6d99daf0e (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5809 6d99daf0e`

**TL;DR:** boards2 had a separate `hub` sub-realm exposing read-only views (boards, threads, comments) over its data. A recent interrealm change forced those view functions to take a `cur realm` parameter, which made them crossing functions unusable as a plain query API. This PR folds the views back into the main `boards2/v1` realm as ordinary non-crossing functions, moves the safe view structs into a reusable `p/gnoland/boards/exts/hub` package, and deletes the now-unused namespace-gated `protected.gno`.

**Verdict: APPROVE** — pure move/refactor with behavior parity for every retained view; the only substantive change is dropping the caller-namespace gate on reads, which is intentional and matches the new "no caller authorization" doc. No blockers.

## Summary
Three things happen here. The view structs `Board`/`Thread`/`Comment`/`Flag`/`Member` and `format.gno` move from the realm `r/gnoland/boards2/v1/hub` into a stateless package `p/gnoland/boards/exts/hub` ([board.gno](https://github.com/gnolang/gno/blob/6d99daf0e/examples/gno.land/p/gnoland/boards/exts/hub/board.gno) · [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/p/gnoland/boards/exts/hub/board.gno)). The public `Get*` API moves into the realm as `boards2.GetBoard(id)` etc., non-crossing, reading `gBoards` directly instead of through the old protected `boards2.GetBoard(cross, ...)` ([hub.gno](https://github.com/gnolang/gno/blob/6d99daf0e/examples/gno.land/r/gnoland/boards2/v1/hub.gno) · [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/r/gnoland/boards2/v1/hub.gno)). The namespace gate `assertCallerHasBoardsNS` and its callers (`GetRealmPermissions`, protected `GetBoard`, `MustGetBoard`, `Iterate`) are deleted with `protected.gno`. Net: reads are now public to any caller, which the data already was on-chain.

## Glossary
- **crossing function** — a `func f(cur realm, ...)` that must be called with `cross(...)`; only these see a fresh `cur` whose `Previous()` is the immediate caller.
- **safe view** — a flattened, copied struct (`hub.Board` etc.) holding an unexported `ref` plus scalar fields, returned instead of the live `*boards.Board` to keep the realm's storage objects unmodifiable from outside.
- **AllReplies** — the `ThreadMeta` flat index holding every comment and reply in a thread; distinct from `thread.Replies`, which holds only top-level comments.

## Fix
Before: `hub.GetThread(cross, ...)` crossed into the hub sub-realm, which called back into `boards2.GetBoard(cross, ...)`, which ran `assertCallerHasBoardsNS` to require the caller live under `gno.land/r/gnoland/boards2/`. After: `boards2.GetThread(boardID, threadID)` is a plain function that reads `gBoards` and reuses the realm's existing `getThread` helper (which still falls back to `meta.HiddenThreads`, preserving hidden-thread lookup). The load-bearing constraint: the view structs had to leave the realm because a `p/` package cannot import an `r/` realm, and the realm now re-exports them as type aliases (`Board = hubexts.Board`) so external callers keep the same type names via `boards2`.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- `examples/gno.land/r/gnoland/boards2/v1/hub.gno:113` — [`GetFlags`](https://github.com/gnolang/gno/blob/6d99daf0e/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L113) · [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L113) doc comment opens "GetFlag returns..." (singular) for a function named `GetFlags`. Carried verbatim from the deleted hub package, not introduced here.

## Missing Tests
None. The moved `exts/hub` package keeps its full unit suite (`board_test.gno`/`comment_test.gno`/`thread_test.gno`, all pass), and the 22 `z_hub_*` filetests exercise every `boards2.Get*` view end to end.

## Suggestions
- `examples/gno.land/r/gnoland/boards2/v1/hub.gno:16-22` — [`Member` and `Flag` type aliases](https://github.com/gnolang/gno/blob/6d99daf0e/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L16-L22) · [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L16-L22) are re-exported but unused by any realm function: `GetMembers` returns `[]boards.User` and `GetFlags` returns `[]boards.Flag` (the boards-package types, not the hub ones).
  <details><summary>details</summary>

  The original hub package also defined `Member`/`Flag` without any function returning them, so this is carried-forward surface, not new. Re-exporting them still lets external callers name the types via `boards2`, so dropping them is optional. Flagging only so the author can decide whether the API surface is intentional.
  </details>
- `examples/gno.land/r/gnoland/boards2/v1/hub.gno:142-145` — [`GetComments` doc](https://github.com/gnolang/gno/blob/6d99daf0e/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L142-L145) · [↗](../../../../../.worktrees/gno-review-5809/examples/gno.land/r/gnoland/boards2/v1/hub.gno#L142-L145) says it returns "all thread comments **and replies**", but it iterates only `thread.Replies`, which holds top-level comments; nested replies live under each comment's `Replies` and in `meta.AllReplies`. Pre-existing behavior, identical before and after this PR; flagged because the PR re-introduces the comment in a new file.
  <details><summary>details</summary>

  Confirmed behaviorally: a thread with one top-level comment plus one nested reply returns `len(GetComments) == 1`, and `GetComment(board, thread, nestedReplyID)` returns `found == false`, because `getComment` looks the reply up in `thread.Replies` rather than the flat `meta.AllReplies` index. The same was true of the deleted `hub.GetComments`/`getComment`, so this is not a regression. To traverse the whole tree, callers chain `Comment.IterateReplies` (defined on the safe view) per top-level comment. The doc wording could be tightened to "top-level thread comments" and a one-line pointer to `IterateReplies`, but that is a separate cleanup the author may defer.
  </details>

## Open questions
- The PR body carries two TODOs: whether to drop `ref *boards.Board` from the `Board` struct and whether the new `hub.gno` header comment is final. Both are author-internal polish, no correctness impact, so not posted.
- Read access is now ungated. The data was already publicly readable on-chain (any node can dump realm storage), so removing the namespace gate does not expose anything new; the gate only ever blocked other on-chain realms from a convenience accessor. Not a finding, noted so the security-model change is on record.
