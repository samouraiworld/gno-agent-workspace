# PR [#6134](https://github.com/gnolang/gno/pull/6134): perf(gnoland,bank): stop genesis balance loading being quadratic

URL: https://github.com/gnolang/gno/pull/6134
Author: moul | Base: master | Files: 7 | +329 -4
Reviewed by: davd-gzl | Model: claude-opus-5 (full, deep) | Commit: `7834d5d7e` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6134 7834d5d7e`
Overview: [overview](../overview.md)
Status: merged as [bdeccddf](https://github.com/gnolang/gno/commit/bdeccddf67414e666c0575be0a951ece45387e5e). Kept as a record; the draft beside it is not offered for posting, and every finding that survives on master is an issue rather than a review comment.

## Overview

Booting a chain from a genesis file writes every listed balance one address at a time, and each write first asked the store which coins that address already holds so it could delete the ones the new amount drops. The store answers that question by scanning every write still pending, so entry number 100,000 scanned the 99,999 before it and a four-second load became seven minutes. This branch adds a second write method that skips the question, and the genesis loop uses it for any address it has not written yet. That address can hold nothing, because the balance loop is the only writer of those keys and the store is empty when it starts. An address named twice in the file keeps the old path, which still asks. The store itself stays quadratic and is left to the issue.

**Verdict: COMMENT** — the store image is unchanged for every genesis shape I could build and the one Warning predates the branch, but the branch that decides between the two write paths is pinned by nothing, and substituting the fast path everywhere moves the committed genesis store hash while the whole package stays green (1 warning, 2 missing tests, 5 suggestions, 2 nits).

## Verify first

- [`gno.land/pkg/gnoland/app.go:785`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L785) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L785) — every balance the loader writes rests on this one boolean. Drop [`tests/applybalance_branch_test.go`](./tests/applybalance_branch_test.go) into `gno.land/pkg/gnoland/`, confirm `TestGenesisBalanceLoadAppHashMatchesSetCoinsOnly` is green, then confirm it reddens with `firstSighting := true` substituted.
- [`tm2/pkg/sdk/bank/keeper.go:533`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L533) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L533) — the `nil` here sends `setAccountTierCoins` to re-read the account the loader wrote three lines earlier. Decide whether the new method should take that account, before its shape is fixed by callers.
- [`contribs/gnogenesis/internal/fork/test.go:254`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L254) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L254) — read this against the line the branch moved at [`:228`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L228) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L228) and decide whether `--timeout` should cover the phase the clock now covers.

## Summary

[`InitCoins`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L526) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L526) is `SetCoins` without the drain loop, and both write the split tier through the shared [`writeSplitCoins`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L542-L546) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L542-L546), so the two cannot drift in how they persist. The drain is a prefix iteration, and [`cacheStore.dirtyItems`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/store.go#L414-L423) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/store/cache/store.go#L414-L423) answers any range query by walking the whole unsorted set, which is where the squared term lives. The precondition that lets genesis skip it holds: [`bank.InitGenesis`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/genesis.go#L29-L37) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/genesis.go#L29-L37) writes params and nothing else, and the balance loop is the only caller that reaches [`setSplitBalance`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/balance.go#L165-L173) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/balance.go#L165-L173) before genesis transactions run.

The new probe cannot feed the term it avoids: [`cacheStore.Get`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/store.go#L158) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/store/cache/store.go#L158) caches its result undirty, and [`setCacheValue`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/store.go#L462-L464) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/store/cache/store.go#L462-L464) enters the unsorted set only for a dirty write.

## Benchmarks / Numbers

Everything below was run at `7834d5d7e` on a 6-core AMD EPYC with `-benchtime 1x`, one `go test` invocation per benchmark so nothing shares the box, and reported as the median of five samples for the store table and three for the rest. Single samples here swing by a third, so no figure rests on one run.

`tm2/pkg/store/cache`, the loop shape against the cache store alone. Keys walked is the sum from 0 to n minus 1; per key is the median divided by it.

| Balances | Median | Growth | Keys walked | Per key |
| ---: | ---: | ---: | ---: | ---: |
| 1,000 | 35.28 ms | | 499,500 | 70.6 ns |
| 2,000 | 163.72 ms | 4.64x | 1,999,000 | 81.9 ns |
| 4,000 | 821.38 ms | 5.02x | 7,998,000 | 102.7 ns |
| 8,000 | 3.47 s | 4.22x | 31,996,000 | 108.4 ns |
| 16,000 | 14.69 s | 4.24x | 127,992,000 | 114.8 ns |
| 16,000, writes only | 19.37 ms | | 0 | |

Growth sits at or above the 4x a squared term predicts at every step, and per-key cost rises 63% across the range as the unsorted set outgrows the processor cache. The squared term alone is what carries the issue's extrapolation into days: master's own 413.99 s at 100,000 balances scales to 5.1 days at the 3,262,505 that genesis file holds, before the rising per-key cost is counted.

`tm2/pkg/sdk/bank`, the two keeper methods against a cache-wrapped multistore.

| Balances | `SetCoins` | `InitCoins` | Ratio |
| ---: | ---: | ---: | ---: |
| 10,000 | 6.19 s | 184 ms | 34x |
| 20,000 | 48.42 s | 399 ms | 121x |

`SetCoins` grows 7.82x per doubling here and `InitCoins` 2.17x. The description reports 4.29x and 2.17x from single samples; my own `SetCoins` samples ran 43.1 s to 56.7 s at 20,000, so a one-sample ratio on this shape is not reproducible either way. What is reproducible is the allocation column the benchmark already prints and the description does not quote: `SetCoins` goes 859,964 to 1,720,152 allocations and `InitCoins` 709,853 to 1,420,102, both exactly doubling on a doubled input. Both are linear in allocations, so the entire quadratic is allocation-free time inside the map scan, and a reader checks that without owning anyone's machine.

Two costs inside `applyBalance` itself, measured against a harness replicating its loop at 100,000 balances. The allocation and byte columns are deterministic; the wall clock is the median of three.

| Variant | Median | Allocations per balance | Bytes per balance |
| --- | ---: | ---: | ---: |
| As shipped | 3.32 s | 90.02 | 3,491 |
| Without the account probe | 3.08 s | 85.02 | 3,283 |
| Probe kept, account handed to `setAccountTierCoins` | 1.99 s | 71.01 | 2,746 |

The probe costs 5.00 allocations and 208 bytes per balance, which is the residual the description leaves unattributed at 14.8% and 21.3%. Handing the account through saves 19 allocations and 744 bytes per balance, four times what the probe costs.

## Warnings (should fix)

- **[the progress counter is frozen before anything starts watching it]** [`contribs/gnogenesis/internal/fork/test.go:254`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L254) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L254) — the branch moved the clock above `NewInMemoryNode` because InitChain runs inside it, and left the deadline, the progress ticker and the announcement below it.
  <details><summary>details</summary>

  `--timeout` is documented as "maximum time to wait for genesis replay to complete" at [`:76`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L76) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L76), default 30 minutes. The deadline is created at `:254`, after `NewInMemoryNode` returned at [`:230`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L230) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L230). The 30-second progress ticker at [`:251`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L251) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L251) and the `Replaying %d txs (timeout: %s)` line at [`:248`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L248) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L248) are below it too.

  All three watch `txProcessed`, which the result handler at [`:167`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L167) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L167) advances from `InitChainerConfig`, so it is final before the loop starts. `Ready()` returns `firstBlockSignal`, so what the loop waits for is block one and its commit, which is where the description says a 3,262,505-row genesis sat. The budget is real; it covers the phase after the one its help text names.

  The balance load this branch speeds up is the smaller half. Genesis transactions are delivered inside the same call, at [`app.go:615`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L615) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L615) for the in-memory path and [`:724`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L724) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L724) for the streaming one, both reached from `InitChainer`. The command knows this: it reads a final `txProcessed` the moment `n.Ready()` fires and compares it against the expected count at [`:288`](https://github.com/gnolang/gno/blob/7834d5d7e/contribs/gnogenesis/internal/fork/test.go#L288) · [↗](../../../../../.worktrees/gno-review-6134/contribs/gnogenesis/internal/fork/test.go#L288). So the whole replay finishes before the deadline exists, on any genesis, before or after this branch, and the announcement names a phase that is already over.

  This predates the branch and is in scope because the diff sweeps this exact class, four statements placed on the wrong side of a call that turned out to do the work, and moves one of them.

  Moving the deadline up does not make the load interruptible: `NewInMemoryNode` takes no context. What it changes is that the budget is spent rather than restarted, so a replay that took 25 minutes cannot then be given a fresh 30.

  Fix: create `timeoutCtx` and start the ticker above `NewInMemoryNode`, beside the clock, and print the announcement there.
  </details>

## Missing Tests

- **[the branch the whole change turns on is unpinned, and getting it wrong moves the genesis store hash]** [`gno.land/pkg/gnoland/app.go:785`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L785) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L785) — raised inline by tbruyelle at [`app.go:808`](https://github.com/gnolang/gno/pull/6134#discussion_r3933470179) and [`mock_test.go:182`](https://github.com/gnolang/gno/pull/6134#discussion_r3933494727); this is what it costs.
  <details><summary>details</summary>

  Two things keep the branch invisible. [`mockAuthKeeper.GetAccount`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/mock_test.go#L226) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/mock_test.go#L226) returns nil unconditionally, so every mock-backed test takes the fast path, and [`mockBankKeeper.InitCoins`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/mock_test.go#L181-L184) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/mock_test.go#L181-L184) counts into the same field as `SetCoins`, so nothing downstream can tell them apart. The one real-keeper repeat test, [`TestApplyBalanceWithARepeatedAddress`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app_test.go#L3927) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app_test.go#L3927), uses `ugnot` only, which is the sole account-tier denom, so the drain it would skip writes nothing either way.

  With `firstSighting := true` substituted, `go test ./gno.land/pkg/gnoland/` is green over the whole package. [`tests/applybalance_branch_test.go`](./tests/applybalance_branch_test.go) reddens on three counts: which method each entry took, the split-tier denom a repeat leaves behind, and the committed store hash.

  **Repro:**

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6134 -R gnolang/gno
  # copy tests/applybalance_branch_test.go from this review directory into gno.land/pkg/gnoland/
  sed -i 's|firstSighting := cfg.acck.GetAccount(ctx, bal.Address) == nil|firstSighting := true|' gno.land/pkg/gnoland/app.go
  go test -count=1 ./gno.land/pkg/gnoland/
  git checkout gno.land/pkg/gnoland/app.go
  rm gno.land/pkg/gnoland/applybalance_branch_test.go
  ```

  The store hash moving is the finding: a genesis file that names an address twice produces a different chain.

  ```
  --- FAIL: TestGenesisBalanceLoadAppHashMatchesSetCoinsOnly (0.00s)
          Error:    Not equal:
                    expected: []byte{0xb1, 0x43, 0x60, 0x84, 0xfd, 0x44, 0xaf, 0xff, …}
                    actual  : []byte{0x58, 0x26, 0x59, 0xff, 0xb, 0xc1, 0xd5, 0x12, …}
          Messages: genesis store hash must not move
  --- FAIL: TestApplyBalanceRepeatDrainsAStaleSplitDenom (0.00s)
          Error:    Not equal:
                    expected: 0
                    actual  : 7
          Messages: the dropped split-tier denom must be drained by the repeat
  --- FAIL: TestApplyBalanceTakesTheRightKeeperPath (0.00s)
      --- FAIL: .../repeated_vesting_address (0.00s)
      --- FAIL: .../plain_entry_after_a_vesting_one (0.00s)
      --- FAIL: .../repeated_address (0.00s)
  FAIL	github.com/gnolang/gno/gno.land/pkg/gnoland	18.503s
  ```

  Fix: add the file beside `TestApplyBalanceWithARepeatedAddress`. It asserts the effect rather than a call count, so it holds without changing `setCoinsCalls`, which the description keeps on purpose.
  </details>

- **[the ordering the comment calls load-bearing is load-bearing, and nothing holds it]** [`tm2/pkg/sdk/bank/keeper.go:532-536`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L532-L536) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L532-L536) — swapping the account-tier write and the split-tier write leaves every test in the tree green, and the swapped order leaks a split-tier key on the failing input.
  <details><summary>details</summary>

  The comment at [`:532`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L532) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L532) says the account tier goes first because it is the only step that can fail. That is true and reachable: `setAccountTierCoins` calls `ensureAccount`, which returns whatever account exists, and [`BaseSessionAccount.SetCoins`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/std/account.go#L245-L250) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/std/account.go#L245-L250) rejects any non-zero amount. On a session account `InitCoins` returns an error, and with the two writes swapped it returns that error having already written the split key.

  [`tests/initcoins_order_test.go`](./tests/initcoins_order_test.go) is the input that separates them. At head it passes; with the writes swapped it reads `expected: 0, actual: 9`, and `go test ./tm2/pkg/sdk/bank/` without it is `ok` under the same swap.

  Fix: add the file beside `initcoins_test.go`. The comment stays as written.
  </details>

## Suggestions

- **[the new method re-reads the account its only caller wrote three lines earlier]** [`tm2/pkg/sdk/bank/keeper.go:533`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L533) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L533) — the `nil` costs 19 allocations and 744 bytes per balance, four times what the probe this branch adds costs.
  <details><summary>details</summary>

  `nil` sends `setAccountTierCoins` into [`ensureAccount`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L250-L257) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L250-L257), which reads the account key and amino-decodes it. [`applyBalance`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L802) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L802) wrote that account three lines earlier and still holds it. The parameter exists for exactly this: [`:259-260`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L259-L260) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L259-L260) reads "acc is the account if the caller already has it, else nil".

  So the loop reads the account store twice per balance, once for the new probe and once inside `InitCoins`. Over 100,000 balances the second read is 3.32 s against 1.99 s, and the deterministic columns are 9,001,585 allocations against 7,101,478 and 349,057,952 bytes against 274,638,656.

  `SetCoins` passes `nil` too, and it has the same caller, so the same saving is available there. Its own drain makes it the slow path either way, which is why this is worth doing on the new method rather than both.

  Fix: give `InitCoins` the account as a parameter and pass `acc` from `applyBalance`, while the method has one caller and its shape is still free.
  </details>

- **[the benchmark the file documents runs without bound]** [`tm2/pkg/sdk/bank/initcoins_bench_test.go:52`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_bench_test.go#L52) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_bench_test.go#L52) — `BenchmarkCoinsLoadSetCoins100000` has no completion I could measure, and `-timeout` does not stop a benchmark in progress.
  <details><summary>details</summary>

  The file's own instruction at [`:17`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_bench_test.go#L17) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_bench_test.go#L17) is `-bench 'CoinsLoad'`, which matches all six, and the 100,000 pair sits behind the four that finish in about a minute. Extrapolating the measured 20,000 figure quadratically puts one iteration past twenty minutes.

  `-timeout` does not help. A five-second timeout let a 37.7-second benchmark run to completion and exit 0:

  ```bash
  # from a local clone of gnolang/gno, in tm2/:
  go test ./pkg/sdk/bank/ -run XXX -bench '^BenchmarkCoinsLoadSetCoins20000$' -benchtime 1x -timeout 5s
  ```

  ```
  BenchmarkCoinsLoadSetCoins20000-6   	       1	37720650596 ns/op
  PASS
  ok  	github.com/gnolang/gno/tm2/pkg/sdk/bank	37.749s
  ```

  No CI job reaches it, since nothing under `.github/workflows/` passes `-bench`, which is why nothing goes red and why it survives to a terminal.

  Fix: drop the 100,000 pair or gate it on `testing.Short()`. The 10,000 and 20,000 pair already carries the demonstration.
  </details>

- **[the equivalence tests run against the store configuration the branch says shows nothing]** [`tm2/pkg/sdk/bank/initcoins_test.go:61-65`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_test.go#L61-L65) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_test.go#L61-L65) — `setupTestEnv` hands back the bare IAVL store, and the branch's own benchmark comment says InitChain runs against a cache-wrapped one.
  <details><summary>details</summary>

  [`initcoins_bench_test.go:28-32`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_bench_test.go#L28-L32) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_bench_test.go#L28-L32) adds `env.ctx = env.ctx.WithMultiStore(env.ctx.MultiStore().MultiCacheWrap())` and says benchmarking without it "would measure tree cost only and show nothing". The equivalence tests compare iteration order over the same store, and iteration order under a genesis load is supplied by the cache layer's `dirtyItems` rather than by the tree, so the two runs are compared under a store the loader never uses.

  Fix: add the same wrap line to `TestInitCoinsMatchesSetCoinsOnFreshAddress` and to `TestInitCoinsWholeLoadMatchesSetCoins` at [`:114-115`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_test.go#L114-L115) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_test.go#L114-L115).
  </details>

- **[the negative test passes for a partial drain, a full wipe and a no-op alike]** [`tm2/pkg/sdk/bank/initcoins_test.go:143-146`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/initcoins_test.go#L143-L146) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/initcoins_test.go#L143-L146) — `require.NotEqual` asserts only that something differs, and the exact post-state is available.
  <details><summary>details</summary>

  This is the one assertion in the tree that reddens when `InitCoins` starts draining, so it earns its place. What it does not pin is what `InitCoins` left, which is `1aaa,20bbb`. As written it also passes if `InitCoins` wiped the address or wrote nothing.

  Fix: assert the value.
  </details>

- **[the store benchmark ships one control and it is a single point]** [`tm2/pkg/store/cache/bench_quadratic_test.go:43`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/store/cache/bench_quadratic_test.go#L43) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/store/cache/bench_quadratic_test.go#L43) — the comment promises to isolate two terms, and the control that would attribute the cost is the one missing.
  <details><summary>details</summary>

  `BenchmarkWritesOnly16000` shows writes are fast, not that they are linear, which needs a second size. More useful is the pair the file does not have: the same iteration count over a clean cache against a dirty one. That pair is the file's whole thesis, and separating it from "iteration is expensive" is what tells the next reader where to look.

  Fix: add a second size to the writes-only control and an iterate-only benchmark at both cache states.
  </details>

## Nits

- **[a paragraph about creating the account now also sits above the probe]** [`gno.land/pkg/gnoland/app.go:773-776`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L773-L776) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L773-L776) repeats [`:787-790`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L787-L790) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L787-L790) word for word. At the merge base those four lines appear once, so the branch added the copy. Deleting `:773-776` is the whole fix.
- [`tm2/pkg/sdk/bank/keeper.go:483-485`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L483-L485) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L483-L485) still tells a caller that `SetCoins` costs `O(denoms currently held)`, which [`:513-515`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/keeper.go#L513-L515) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/keeper.go#L513-L515) states thirty lines below is false, and [`balance.go:178`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/bank/balance.go#L178) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/bank/balance.go#L178) repeats it for `splitCoins`. Not posted: a finding about what a code comment says stays in the review file.

## Verified

- Every genesis shape produces master's store bytes. A list carrying account-tier only, split-tier only, both tiers, vesting, a repeat in the same tier, a repeat dropping a split denom, a repeat clearing a schedule and a repeat adding a denom commits to the same hash under `InitCoins`-on-first-sighting as under unconditional `SetCoins`, at [`tests/applybalance_branch_test.go`](./tests/applybalance_branch_test.go).
- No path writes a split-tier key without an account, so the account probe answers the question the drain would. `setSplitBalance` is reached from `SetCoins`, `InitCoins` and `writeSplitCoins`, each preceded by `SetAccount` in the loader, from `AddCoins` through `ensureAccount`, and from `SubtractCoins`, which cannot debit an address never credited.
- `InitCoins` is unreachable from a realm or the ante handler. It is absent from [`vm.BankKeeperI`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/sdk/vm/types.go#L18-L34) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/sdk/vm/types.go#L18-L34) and [`auth.BankKeeperI`](https://github.com/gnolang/gno/blob/7834d5d7e/tm2/pkg/sdk/auth/types.go#L25-L33) · [↗](../../../../../.worktrees/gno-review-6134/tm2/pkg/sdk/auth/types.go#L25-L33); the full interface is held only by [`InitChainerConfig.bankk`](https://github.com/gnolang/gno/blob/7834d5d7e/gno.land/pkg/gnoland/app.go#L396) · [↗](../../../../../.worktrees/gno-review-6134/gno.land/pkg/gnoland/app.go#L396).
- The call chain the new comment names exists, with one hop unnamed: `NewInMemoryNode` reaches `node.NewNode`, then `doHandshake`, then `Handshaker.Handshake`, then `ReplayBlocks`, then `InitChainSync`, which fires only at height zero.
- `go test -race -count=1` green at this head on `tm2/pkg/sdk/bank`, `tm2/pkg/store/cache`, `gno.land/pkg/gnoland` and `contribs/gnogenesis/internal/fork`.

## Existing threads

- tbruyelle, APPROVED, with two open inline comments: [`app.go:808`](https://github.com/gnolang/gno/pull/6134#discussion_r3933470179) asks for a mock-backed test of the branch, and [`mock_test.go:182`](https://github.com/gnolang/gno/pull/6134#discussion_r3933494727) asks for a separate `initCoinsCalls` counter. Neither is addressed at this head. The first cannot be met as asked, because the mock account keeper answers nil for every address; the second would break the `setCoinsAtRecompute` assertion the description keeps on purpose, which the shipped test avoids by asserting the effect instead.

## Open questions

- The fast path's benefit is proportional to how many addresses are first sightings, and nothing counts or reports repeats. A genesis file that is mostly repeats keeps the quadratic and looks the same from outside. Not posted: no decision rests on it in this branch.
- `AGENTS.md` requires an ADR for a non-trivial AI-assisted change, and this one adds an exported keeper method, widens an exported interface and edits the genesis critical path. Whether the trigger fired is not readable from outside the PR. Not posted: contribution-policy compliance is never a code finding.
- `perf` is not among the conventional types `AGENTS.md` lists, and the scope is two comma-separated words. Nothing enforces either and master carries `perf(gnovm)` commits. Not posted.
