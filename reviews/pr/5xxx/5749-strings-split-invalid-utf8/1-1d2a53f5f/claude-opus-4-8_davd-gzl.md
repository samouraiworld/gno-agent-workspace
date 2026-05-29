# PR #5749: fix(gnovm/stdlibs/strings): keep invalid UTF-8 bytes in Split, add tests

URL: https://github.com/gnolang/gno/pull/5749
Author: davd-gzl | Base: master | Files: 4 | +1177 -6
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 1d2a53f5f (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5749 1d2a53f5f`

**Verdict: APPROVE** — `explode` now slices raw bytes, matching current upstream Go; the test ports are faithful and pin the round-trip; the consensus-visible Merkle-root shift is correctly handled by the regenerated `expectedCrossrealm38Hash`. CI green (only `Merge Requirements` red — the approval-gate bot). Two non-blocking notes below: the change is consensus-affecting, and ~1171 lines of stdlib test source now land in committed genesis state.

## Summary

`explode` (backing `Split`/`SplitN` with an empty separator) rewrote every invalid UTF-8 byte to the 3-byte U+FFFD encoding — but only for non-last elements, so the output was asymmetric and `Join(Split(s, ""), "")` no longer recovered `s`. `Split("\xff-\xff", "")` gave `["\xef\xbf\xbd", "-", "\xff"]` instead of `["\xff", "-", "\xff"]`. The fix deletes the re-encoding branch so `explode` slices the raw bytes, matching upstream go1.25.9 exactly. Blast radius is exactly `Split`/`SplitN` on empty separator — `explode` has one caller ([`strings.gno:239`](https://github.com/gnolang/gno/blob/1d2a53f5f/gnovm/stdlibs/strings/strings.gno#L239) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings.gno#L239)).

The PR also back-fills the upstream `strings_test.go` / `compare_test.go` cases Gno never ported (+1171 lines), including the invalid-UTF-8 `Split` cases that pin this fix and a previously-untested `TestTrimSpace`.

The interesting wrinkle is consensus: Gno commits stdlib **source** (including `_test.gno`) into genesis via `MPStdlibAll`, so editing `strings.gno` and adding test files both shift the iavlStore Merkle root. `TestAppHashCrossrealm38` is a guard that fires on exactly that shift; the author regenerated the pinned hash with a dated note.

## Glossary

- `explode` — internal `strings` helper; splits a string into one element per code point. Reached only by `Split`/`SplitN` when `sep == ""`.
- `MPStdlibAll` — `MemPackage` filter used when loading stdlibs into the chain store; keeps `_test.gno`/`_filetest.gno` (unlike `MPStdlibProd`). Genesis uses it at [`keeper.go:302`](https://github.com/gnolang/gno/blob/1d2a53f5f/gno.land/pkg/sdk/vm/keeper.go#L302) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/keeper.go#L302).
- save set / app hash — the set of objects committed to the iavlStore each block; its Merkle root surfaces as the app hash. Stdlib package source is part of it.

## Fix

Before: the loop body re-assigned `a[i] = string(utf8.RuneError)` whenever `DecodeRuneInString` returned `RuneError`, replacing the original bytes with `\xef\xbf\xbd`. The last element (`a[n-1]`) skips the loop, so only non-last invalid bytes were rewritten — the asymmetry in the bug report.

After: the branch is gone; the loop slices `s[:size]` directly ([`strings.gno:26-30`](https://github.com/gnolang/gno/blob/1d2a53f5f/gnovm/stdlibs/strings/strings.gno#L26-L30) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings.gno#L26-L30)), byte-identical to [upstream go1.25.9](https://github.com/golang/go/blob/go1.25.9/src/strings/strings.go#L18). The doc comment is updated to match ([`strings.gno:19`](https://github.com/gnolang/gno/blob/1d2a53f5f/gnovm/stdlibs/strings/strings.gno#L19) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings.gno#L19)).

Consensus handling: because `strings.gno` source (and the new test files) are committed to genesis, the run-`crossrealm38`-and-pin-the-commit-hash guard at [`apphash_crossrealm38_test.go:55-58`](https://github.com/gnolang/gno/blob/1d2a53f5f/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L55-L58) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L55-L58) trips. The author bumped the pin to `fb07264a…` with a 2026-05-29 note explaining the cause — the same workflow the crypto-stdlib batch used on 2026-05-26 (precedent in the same comment block).

## Verification

- `go test -run 'TestStdlibs/strings$' ./gnovm/pkg/gnolang/` → `ok` (the +1171 test lines compile and pass under the interpreter).
- `TestAppHashCrossrealm38` was the **only** failing test on the prior head; CI's observed hash (`fb07264a5218ef4257ab2eeab3c0f231db98aadb9ee307f52f2aa6bd0bb90460`) is exactly what the author pinned. On the current head `1d2a53f5f`, `main / test` passes.
- No example/filetest/txtar broke, confirming nothing else depended on the old re-encoding behavior.
- Local `TestAppHashCrossrealm38` could not be re-run in this worktree (genesis stdlib load type-checks `errors/errors_test.gno` and fails to resolve `fmt`/`testing` — pre-existing in master, environment-specific, unrelated to this PR). CI is authoritative here.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- The two new files header-cite go1.25.9, while the audit that surfaced this framed Gno's baseline as go1.17. Not wrong — the *fix* and *tests* are pulled from current upstream on purpose (the old behavior matched go1.17, which is why this was "snapshot lag"). No action needed; flagging only so a future reader doesn't mistake the citation for a baseline claim.

## Missing Tests
- **[deliberate gap, documented]** [`strings_test.gno:198-201`](https://github.com/gnolang/gno/blob/1d2a53f5f/gnovm/stdlibs/strings/strings_test.gno#L198-L201) · [↗](../../../../../.worktrees/gno-review-5749/gnovm/stdlibs/strings/strings_test.gno#L198-L201) — `TestRepeatCatchesOverflow` skips the upstream `{"-", maxInt, "out of range"}` case.
  <details><summary>details</summary>

  The skip is correct and well-documented: that path hits the allocator-overflow host panic that crashes the whole test binary until [PR #5723](https://github.com/gnolang/gno/pull/5723) lands. The TODO already pins the follow-up. No change requested — noting so the gap is tracked against #5723's merge.
  </details>

## Suggestions
- **[design observation, not this PR]** [`keeper.go:302`](https://github.com/gnolang/gno/blob/1d2a53f5f/gno.land/pkg/sdk/vm/keeper.go#L302) · [↗](../../../../../.worktrees/gno-review-5749/gno.land/pkg/sdk/vm/keeper.go#L302) — stdlib `_test.gno` source is committed to genesis state.
  <details><summary>details</summary>

  Loading stdlibs with `MPStdlibAll` means the ~1171 lines of test source added here become part of the committed genesis/consensus state, not just the strings.gno fix. This is the status quo (every stdlib already ships its `_test.gno` into genesis — `errors_test.gno`, `bytes_test.gno`, etc.), so this PR is consistent and the pin bump is the right handling. The broader question — whether stdlib test code belongs in consensus state at all, vs. an `MPStdlibProd` genesis set — is worth a separate discussion, not a blocker here.
  </details>

## Questions for Author
- This is a consensus-affecting behavior change: any realm calling `strings.Split`/`SplitN` on invalid UTF-8 now produces different results, so it must ship as a coordinated stdlib/software upgrade (the apphash guard correctly surfaces it). Is that acknowledged in the rollout plan, or does it need to ride the test13 chain-upgrade vehicle referenced in the apphash comment block?
