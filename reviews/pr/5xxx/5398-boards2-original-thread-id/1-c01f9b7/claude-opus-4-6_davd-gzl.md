# PR #5398: chore(boards2): add original thread ID to `hub` realm thread reposts

**URL:** https://github.com/gnolang/gno/pull/5398
**Author:** jeronimoalbi | **Base:** master | **Files:** 4 | **+32 -17**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary
This PR adds an `OriginalThreadID` field to the hub realm's `Thread` struct in the boards2 package. Previously, reposts only exposed `OriginalBoardID` (the board where the original thread lives), but not the ID of the original thread itself. The new field is populated from `ref.ParentID` in `NewSafeThread()` — for reposts, `ParentID` holds the original thread's ID (set during `NewRepost()` in the boards package at `p/gnoland/boards/thread.gno:85`). For non-repost threads, both `OriginalBoardID` and `OriginalThreadID` are zero.

The change touches four files: the `Thread` struct and its constructor in `hub/thread.gno`, and three filetests (`z_1_a`, `z_1_b`, `z_1_c`) which verify the new field's value for original threads, edited threads, and reposts respectively. The repost filetest (`z_1_c`) was improved to create two source threads so it can verify `OriginalThreadID = 2` (not just 1), confirming the field tracks the actual thread ID.

## Test Results
- **Existing tests:** PASS (all 22 hub tests/filetests pass)
- **Edge-case tests:** skipped

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `examples/gno.land/r/gnoland/boards2/v1/hub/thread.gno:93` — The field `OriginalThreadID` is populated from `ref.ParentID`, but `ParentID` has dual semantics in the boards package: for reposts it holds the original thread's ID, for comments/replies it holds the parent comment's ID. The mapping is safe because `NewSafeThread` panics on non-thread posts (`IsThread(ref)` check at line 85), but this non-obvious relationship deserves a comment explaining why `ParentID` is the correct source, e.g.: `// For reposts, ParentID holds the original thread's ID (set in NewRepost)`.

## Nits
- [ ] `examples/gno.land/r/gnoland/boards2/v1/hub/filetests/z_1_c_filetest.gno:21` — The comment `// ID = 2` is helpful but the variable name is still `srcThreadID` — consider adding a brief note that the test intentionally creates two threads to verify the ID is the thread ID (2), not just always 1.

## Missing Tests
None — the existing filetest `z_1_c_filetest.gno` was updated to use two source threads, verifying `OriginalThreadID = 2` (not just 1), which adequately covers the new field.

## Suggestions
- Add a comment on the `OriginalThreadID` field assignment in `NewSafeThread` explaining the `ParentID` → `OriginalThreadID` mapping, e.g.: `// For reposts, ParentID holds the original thread's ID (set in NewRepost)`.

## Questions for Author
- Is there a reason `OriginalThreadID` isn't also exposed through the `Render` path or any board-level query? Currently it's only accessible via `GetThread`.

## Verdict
APPROVE — Clean, well-tested change; only minor documentation suggestions.
