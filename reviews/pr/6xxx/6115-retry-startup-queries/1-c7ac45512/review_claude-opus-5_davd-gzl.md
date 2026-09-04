# PR [#6115](https://github.com/gnolang/gno/pull/6115): fix(gpao): retry the startup queries instead of dying or guessing

URL: https://github.com/gnolang/gno/pull/6115
Author: gfanton | Base: master | Files: 4 | +268 -32
Reviewed by: davd-gzl | Model: claude-opus-5 (full) | Commit: `c7ac45512` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6115 c7ac45512`
Overview: [overview](../overview.md)

## Overview

gpao asked its node two questions before it started following the chain, each exactly once. The first asked where the tip is, which decides where to begin; the second asked for the block gas ceiling, which sizes every approval it signs. A node that was not answering yet turned the first into an exit and the second into a guess that stood for the life of the process. On a chain configured below that guess the [ante rejects every probe](https://github.com/gnolang/gno/blob/c7ac45512/tm2/pkg/sdk/auth/ante.go#L70-L79), so the daemon typechecks packages forever and approves none of them. This branch deletes the pre-loop tip query, resolves the height from the first poll that answers, and asks for the ceiling on every poll until the chain answers once. The field holding the ceiling becomes an `atomic.Int64`, because the block reader now writes it while the verifier goroutine reads it.

**Verdict: REQUEST CHANGES** — the ceiling retry is right and the atomic is complete, but resolving the tip inside the loop drops every block produced before the first tick, on a healthy start as much as a boot race, and the five lines that keep both behaviours leave the branch's own tests green (1 warning, 2 suggestions, 3 nits).

## Verify first

- [`contribs/gpao/oracle.go:317-321`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L317-L321) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L317) — the guard that decides which blocks are read at all. Run the repro in [`comment_claude-opus-5.md`](./comment_claude-opus-5.md), which drops [`tests/bootwindow_test.go`](./tests/bootwindow_test.go) into `contribs/gpao/` and reports 13 where it asks for 8, then reports 8 once the merge base's `oracle.go` is in place.
- [`contribs/gpao/oracle.go:796`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L796) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L796) — the only read of the ceiling outside the block reader. `grep -n blockMaxGas contribs/gpao/oracle.go` returns two writes, both in [`run`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L265) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L265), and this one read; confirm no later change adds a second reader that needs the value to be stable across two calls.

## Summary

Two startup queries become in-loop retries. The tip query is gone from before the loop and its result comes from [the first poll that answers](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L317-L321) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L317), and the ceiling is [asked for on every poll](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L304-L309) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L304) until [`queryBlockMaxGas`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L586) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L586) reports that the chain answered. The split between "the node said nothing" and "the chain answered" is what makes the retry terminate: the `-1` a chain sets to mean no bound [leaves the fallback standing](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L611-L614) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L611) and still [counts as an answer](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L597) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L597), so the loop does not ask forever on a chain that will never give a usable number.

One flag bounds the retry. `ceilingKnown` is a local in [`run`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L266) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L266), set on the one path that returns `answered`, and both queries take the loop's `ctx`, so a cancelled context ends them rather than being swallowed. Nothing makes "asked once" wrong on this chain. tm2 [changes consensus params only from `EndBlock.ConsensusParams`](https://github.com/gnolang/gno/blob/c7ac45512/tm2/pkg/bft/state/execution.go#L427-L430) · [↗](../../../../../.worktrees/gno-review-6115/tm2/pkg/bft/state/execution.go#L427), and gno.land's `EndBlocker` returns `ResponseEndBlock` at five sites where [the only one carrying a field](https://github.com/gnolang/gno/blob/c7ac45512/gno.land/pkg/gnoland/app.go#L1228-L1230) · [↗](../../../../../.worktrees/gno-review-6115/gno.land/pkg/gnoland/app.go#L1228) sets `ValidatorUpdates`, so `Block.MaxGas` is fixed at genesis for the chain's life.

## Benchmarks / Numbers

First block read with `-start-height 0`, against a node answering from the first call and producing one block every 50 ms, polled every 250 ms. From [`tests/bootwindow_test.go`](./tests/bootwindow_test.go), three runs each at the merge base and at the head, identical every time.

| Tree | First block read |
| --- | ---: |
| 2ed70a202, the merge base | 8 |
| c7ac45512 | 13 |
| c7ac45512 with the tip resolved before the loop | 8 |

Stderr lines written while the node refuses every connection, counted over 20 ticks by [`tests/lograte_test.go`](./tests/lograte_test.go).

| Tree | Total | Ceiling query | Status query |
| --- | ---: | ---: | ---: |
| c7ac45512 | 40 | 20 | 20 |
| c7ac45512 with the ceiling query below the status check | 20 | 0 | 20 |

At the default [one-second poll](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/main.go#L40) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/main.go#L40) moving the query is one stderr line and one round trip per second instead of two, for as long as the node stays down.

## Warnings (should fix)

- **[correctness]** [`contribs/gpao/oracle.go:317-321`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L317-L321) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L317) — `-start-height 0` resolves the tip on the first poll rather than at process start, so a `MsgAddPackage` landing in between is never read and stays `unknown` on `/status` for the rest of the run.
  <details><summary>details</summary>

  The merge base [resolved the tip before the ticker existed](https://github.com/gnolang/gno/blob/2ed70a202/contribs/gpao/oracle.go#L265-L271), so the first tick caught up whatever had landed since. Resolving it inside the loop moves the reference point forward by a poll interval and starts above it, and heights only move forward, so the blocks in between are read by nobody and [`statusUnknown`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/status.go#L25) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/status.go#L25) is what a submitter in that window reads for the rest of the run. Nothing in the log names the heights passed over. The measurement is 13 against 8 in the table above, and the same file passes at the merge base, so the branch causes it. The PR body calls this the price of not dying in the boot window; it is also paid on every healthy start, where the node answered and nothing was at risk.

  Fix: keep the pre-loop resolution as a best-effort attempt and let the new in-loop guard cover the failure, which costs five lines and leaves both existing run tests green.
  ```go
  height := o.cfg.startHeight
  if height <= 0 {
      // A node already up resolves the tip here, so the blocks it produces
      // while the first tick is pending are still read.
      if status, err := o.client.RPCClient.Status(ctx, nil); err == nil {
          height = status.SyncInfo.LatestBlockHeight + 1
      }
  }
  ```
  </details>

## Suggestions

- **[correctness]** [`contribs/gpao/oracle.go:589-591`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L589-L591) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L589) — the unanswered path returns `0` rather than the `defaultBlockMaxGas` its log names, so a caller ignoring `answered` clamps every gas amount to zero through `gasWantedFor`.
  <details><summary>details</summary>

  Both callers read `answered` and discard the number, so nothing is wrong today. A caller that does not would size its probe at zero gas, and [`gasWantedFor`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L756-L770) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L756) returns the ceiling on both of its branches when the ceiling is zero, which the ante refuses as a non-positive gas-wanted. Returning the value the log line already names removes the trap and reads the same.

  Fix: `return defaultBlockMaxGas, false`. Applied at c7ac45512, the whole `contribs/gpao` package stays green.
  </details>

- **[performance]** [`contribs/gpao/oracle.go:304-315`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L304-L315) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L304) — the ceiling query runs before the status query, so an unreachable node pays two failed round trips and two stderr lines per poll instead of one.
  <details><summary>details</summary>

  Both queries go to the same endpoint, so a failed `Status` already says the ceiling query will fail too. Moving the ceiling block below the status error check keeps its position relative to the block loop, so the window in which an approval can be signed at the fallback is unchanged, and drops the second line and the second round trip while the node is down. Measured at 20 lines against 40 over 20 ticks in the table above.

  Fix: swap the two blocks. Applied at c7ac45512, the whole `contribs/gpao` package stays green.
  </details>

## Nits

- **[clarity]** [`contribs/gpao/oracle.go:594-596`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L594-L596) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L594) — `gpao: using 3000000000 for block max gas` is now printed both when the chain reports exactly that number and when it reports something unusable and the fallback stands in. The merge base [passed the error into the same line](https://github.com/gnolang/gno/blob/2ed70a202/contribs/gpao/oracle.go#L572), which distinguished them. Not posted: an operator acts the same way in both cases, since the number in use is correct either way.

- **[clarity]** [`contribs/gpao/oracle.go:607-608`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L607-L608) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L607) — `blockMaxGasFrom` still takes an `err` and still branches on it, while its one production call site now [passes `nil`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L593) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L593) because the caller handles the error itself. Not posted: the parameter is a comment-and-signature leftover, and dropping it churns six assertions in [`gaswanted_test.go`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/gaswanted_test.go#L120-L142) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/gaswanted_test.go#L120) for no behaviour change.

- **[tests]** [`contribs/gpao/run_test.go:160-168`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/run_test.go#L160-L168) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/run_test.go#L160) — `ceilingLateRPC` cancels the run inside the first answered `ConsensusParams`, so nothing observes that the query stops after it. Deleting `ceilingKnown = true` leaves the whole package green. Not posted: the cost of that regression is one round trip per poll, and the flag's other half is covered.

## Verified

- The atomic is complete and production reads it once. `grep -n blockMaxGas contribs/gpao/oracle.go` gives two writes, [`:265`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L265) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L265) and [`:306`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L306) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L306), both on the block reader, and one read at [`:796`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L796) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L796), inside the approval that loads it once.
- The ceiling cannot go stale after the chain answers. tm2 changes consensus params only from [`abciResponses.EndBlock.ConsensusParams`](https://github.com/gnolang/gno/blob/c7ac45512/tm2/pkg/bft/state/execution.go#L427-L430) · [↗](../../../../../.worktrees/gno-review-6115/tm2/pkg/bft/state/execution.go#L427); `grep -n 'ResponseEndBlock{' gno.land/pkg/gnoland/app.go` gives five sites, four of them bare and [the fifth](https://github.com/gnolang/gno/blob/c7ac45512/gno.land/pkg/gnoland/app.go#L1228-L1230) · [↗](../../../../../.worktrees/gno-review-6115/gno.land/pkg/gnoland/app.go#L1228) carrying `ValidatorUpdates` alone.
- A fresh chain answers the ceiling query on its first poll. [`ConsensusParams`](https://github.com/gnolang/gno/blob/c7ac45512/tm2/pkg/bft/rpc/core/consensus.go#L85-L86) · [↗](../../../../../.worktrees/gno-review-6115/tm2/pkg/bft/rpc/core/consensus.go#L85) reads `LastBlockHeight + 1`, which is 1 before any block commits, and [`getHeight`](https://github.com/gnolang/gno/blob/c7ac45512/tm2/pkg/bft/rpc/core/blocks.go#L156-L158) · [↗](../../../../../.worktrees/gno-review-6115/tm2/pkg/bft/rpc/core/blocks.go#L156) accepts it, so the fallback window does not span a chain's first block.
- The race detector is not in the contribs test job, which [runs `go test -covermode=set -timeout 30m ./...`](https://github.com/gnolang/gno/blob/c7ac45512/.github/workflows/_ci-go.yml#L131) · [↗](../../../../../.worktrees/gno-review-6115/.github/workflows/_ci-go.yml#L131). `TestRunSurvivesTheBootRace` and `TestRunHonoursExplicitStartHeight` are green under `-race` at c7ac45512, so the new field and its two writers show no detected race.

## Open questions

- [`contribs/gpao/oracle.go:547`](https://github.com/gnolang/gno/blob/c7ac45512/contribs/gpao/oracle.go#L547) · [↗](../../../../../.worktrees/gno-review-6115/contribs/gpao/oracle.go#L547) charges `-max-spend` before `enable` runs, and the `verdictWillFail` branch returns without broadcasting, so the bound is consumed by approvals that never paid a fee. Not posted: it predates the branch and no line the diff touches sweeps it.
- Bounding consecutive poll failures before exiting is left to its own change, per the PR body, and the status listener answers 200 either way. A supervisor watching `/status` has no signal that the daemon has never reached the chain.
