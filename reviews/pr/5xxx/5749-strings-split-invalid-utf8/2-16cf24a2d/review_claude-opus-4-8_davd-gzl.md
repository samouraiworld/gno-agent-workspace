# PR [#5749](https://github.com/gnolang/gno/pull/5749): fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests

URL: https://github.com/gnolang/gno/pull/5749
Author: davd-gzl | Base: master | Files: 4 | +1176 -6
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `16cf24a2d` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5749 16cf24a2d`

Round 2. Head advanced `1d2a53f5f` → `16cf24a2d`. The only PR-content change since round 1 is the master merge (~130 commits) and its one conflicted PR-owned file: `apphash_crossrealm38_test.go`, where the note history was extended and the pin moved `fb07264a…` → `3fc3614a…`. The strings fix and test ports are byte-identical to round 1. Verdict unchanged: APPROVE.

**TL;DR:** `strings.Split(s, "")` on a string holding invalid UTF-8 bytes used to swap those bytes for the replacement character `�` on every element except the last, so splitting then joining no longer gave back the original string. This PR removes that rewrite so the raw bytes survive, and back-fills the upstream Go test cases Gno never ported.

**Verdict: APPROVE** — clean master merge; the apphash re-pin to `3fc3614a…` is correct and now verifiable locally (`TestAppHashCrossrealm38` passes on head, where round 1 could not run it). One delta note: [#5723](https://github.com/gnolang/gno/pull/5723) has now merged into this branch, so the overflow test case round 1 flagged as blocked can be re-enabled — verified it passes. Still consensus-affecting; the apphash guard surfaces it correctly.

## Summary

`explode` (backing `Split`/`SplitN` with an empty separator) rewrote every invalid UTF-8 byte to the 3-byte U+FFFD encoding, but only for non-last elements, so the output was asymmetric and `Join(Split(s, ""), "")` no longer recovered `s`. `Split("\xff-\xff", "")` gave `["\xef\xbf\xbd", "-", "\xff"]` instead of `["\xff", "-", "\xff"]`. The fix deletes the re-encoding branch so `explode` slices the raw bytes, matching upstream go1.25.9 exactly. Blast radius is exactly `Split`/`SplitN` on empty separator — `explode` has one caller ([`strings.gno:239`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings.gno#L239) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings.gno#L239)).

Round 2 delta: the branch merged `origin/master`. The merge touched exactly one PR-owned file, the apphash guard, resolving a conflict between master's note history (crypto stdlib, foreign-markdown, errors stdlib, Example-test bumps) and this branch's strings note. The resolution keeps master's notes verbatim, re-appends the strings note, and moves the pinned hash to the combined value `3fc3614a…`. Master pins `28f55f0a…` ([`apphash_crossrealm38_test.go:73`](https://github.com/gnolang/gno/blob/16cf24a2d/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L73) on master is a different value), so the strings source change genuinely shifts the root and the re-pin is required.

## Examples

| Input | Buggy (`efbfbd` = `�`) | Fixed |
|---|---|---|
| `Split("\xff-\xff", "")` | `ff` `2d` `ff` reported as `efbfbd` `2d` `ff` | `ff` `2d` `ff` |
| `Join(Split("\xff-\xff", ""), "")` | `!= "\xff-\xff"` | `== "\xff-\xff"` |

Bytes shown hex. The last element always kept its raw bytes (the loop runs `i < n-1`), which is why only the non-last invalid bytes were rewritten — the asymmetry in the bug report.

## Glossary

- `explode` — internal `strings` helper; splits a string into one element per code point. Reached only by `Split`/`SplitN` when `sep == ""`.
- `MPStdlibAll` — `MemPackage` filter used when loading stdlibs into the chain store; keeps `_test.gno`/`_filetest.gno` (unlike `MPStdlibProd`). Genesis uses it at [`keeper.go:303`](https://github.com/gnolang/gno/blob/16cf24a2d/gno.land/pkg/sdk/vm/keeper.go#L303) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/keeper.go#L303).
- save set / app hash — the set of objects committed to the iavlStore each block; its Merkle root surfaces as the app hash. Stdlib package source is part of it.

## Fix

Unchanged from round 1. The re-encoding branch is gone; the loop slices `s[:size]` directly ([`strings.gno:26-30`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings.gno#L26-L30) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings.gno#L26-L30)), byte-identical to [upstream go1.25.9](https://github.com/golang/go/blob/go1.25.9/src/strings/strings.go#L18). The doc comment is updated to match ([`strings.gno:19`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings.gno#L19) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings.gno#L19)).

Consensus handling: because `strings.gno` source and the new test files are committed to genesis, the guard at [`apphash_crossrealm38_test.go:75-137`](https://github.com/gnolang/gno/blob/16cf24a2d/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L75-L137) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L75-L137) trips. The pin is `3fc3614a…` ([`apphash_crossrealm38_test.go:73`](https://github.com/gnolang/gno/blob/16cf24a2d/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L73) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L73)).

## Verification

- `TestAppHashCrossrealm38` **passes on head** (`3fc3614a…`, 5.36s). Round 1 could not run this locally (genesis stdlib load failed to type-check `errors/errors_test.gno`); with the errors stdlib now merged, it runs and confirms the merge-conflict re-pin is correct. This is the load-bearing check for the round.
- Revert-proof through the interpreter: with the re-encoding branch restored, `Split("\xff-\xff", "")` prints `efbfbd 2d ff` and `Join(...) == s` is false; with the fix, `ff 2d ff` and true. Ran as a temporary `gnovm/tests/files/` filetest.
- Master pins `28f55f0a…`, this PR pins `3fc3614a…` — the strings change genuinely shifts the root, so the re-pin is not a no-op.
- `go test -run 'TestStdlibs/strings$' ./gnovm/pkg/gnolang/` → `ok` (the ported test lines compile and pass under the interpreter).
- `git diff 1d2a53f5f 16cf24a2d -- gnovm/stdlibs/strings/` is empty: the fix and test ports are unchanged since round 1. Master never diverged on `strings.gno` (its only history there predates the split), so the merge lost nothing.
- CI green; only `Merge Requirements` is red (the approval-gate bot). No inline comments, no reviews on the PR.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- The two new files header-cite go1.25.9, while the audit that surfaced this framed Gno's baseline as go1.17. Not wrong — the fix and tests are pulled from current upstream on purpose (the old behavior matched go1.17, which is why this was snapshot lag). No action needed; flagging only so a future reader does not mistake the citation for a baseline claim. [`strings_test.gno:1`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings_test.gno#L1) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings_test.gno#L1).

## Missing Tests
- **[now unblocked by #5723]** [`strings_test.gno:198-201`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings_test.gno?plain=1#L198-L201) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings_test.gno#L198) — `TestRepeatCatchesOverflow` still skips the upstream `{"-", maxInt, "out of range"}` case, but its blocker has merged.
  <details><summary>details</summary>

  The TODO defers this case until [#5723](https://github.com/gnolang/gno/pull/5723) lands, because the oversized allocation used to host-panic and crash the whole test binary. #5723 (`2b21ea3ca`) is now in this branch via the master merge, and the oversized `make` is a recoverable panic. I re-enabled the case and ran it: `strings.Repeat("-", maxInt)` reaches `Builder.Grow(maxInt)`, the recover catches a panic whose message contains `out of range`, and `TestRepeatCatchesOverflow` passes with no binary crash. The remaining cost is that adding the line shifts the genesis Merkle root (`3fc3614a…` → `9bc11316…`, verified), so it needs another apphash bump — but the PR is already bumping that pin and already committing these test files, so folding the case in now is nearly free versus a follow-up.

  Ready-to-add: append one row to the `tests` slice at [`strings_test.gno:183`](https://github.com/gnolang/gno/blob/16cf24a2d/gnovm/stdlibs/strings/strings_test.gno?plain=1#L183) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings_test.gno#L183) and drop the TODO at 198-201:

  ```gno
  {"-", maxInt, "out of range"},
  ```

  Fix: re-enable the upstream case and remove the TODO, then re-pin `expectedCrossrealm38Hash`; or leave it deferred and update the TODO to note the blocker has merged.
  </details>

## Suggestions
- **[design observation, not this PR]** [`keeper.go:303`](https://github.com/gnolang/gno/blob/16cf24a2d/gno.land/pkg/sdk/vm/keeper.go#L303) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/keeper.go#L303) — stdlib `_test.gno` source is committed to genesis state.
  <details><summary>details</summary>

  Loading stdlibs with `MPStdlibAll` means the ~1171 lines of test source added here become part of the committed genesis/consensus state, not just the strings.gno fix. This is the status quo (every stdlib already ships its `_test.gno` into genesis — `errors_test.gno`, `bytes_test.gno`, etc.), so this PR is consistent and the pin bump is the right handling. The broader question — whether stdlib test code belongs in consensus state at all, vs. an `MPStdlibProd` genesis set — is worth a separate discussion, not a blocker here. Not posted: no change required in this PR.
  </details>

## Open questions
- This is a consensus-affecting behavior change: any realm calling `strings.Split`/`SplitN` on invalid UTF-8 now produces different results, so it must ship as a coordinated stdlib/software upgrade (the apphash guard correctly surfaces it). Is that acknowledged in the rollout plan, or does it need to ride the test13 chain-upgrade vehicle referenced in the apphash comment block? Not posted as an inline comment — it is a rollout-sequencing question, not a code change, and the guard already forces the coordination.
