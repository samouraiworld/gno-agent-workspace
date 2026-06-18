# Review: PR #5710
Posted: https://github.com/gnolang/gno/pull/5710#pullrequestreview-4520524582
Event: COMMENT

## Body
Correct fix. Verified on bac5ab8d1: removing the `gm.ConsumeGas` charges in `ComputeMapKey` drops both the read-only `Lookup` tx and the write-path `Insert` tx by an identical 5516 gas, and the per-key amounts are byte-identical to master.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5710-computemapkey-via-store-meter/2-bac5ab8d1/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/machine.go:201-205 [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/machine.go#L201) [posted](https://github.com/gnolang/gno/pull/5710#discussion_r3432195696)
`store.(*defaultStore)` never matches the production wrapper `transactionStore{*defaultStore}`, so this guard does nothing on real stores. The only store it does match, a bare `*defaultStore`, is the shared base, so it would stamp one tx's meter onto shared state. Delete the block.

## gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar:35-47 [↗](../../../../../.worktrees/gno-review-5710/gno.land/pkg/integration/testdata/compute_map_key_restore_gas.txtar#L35) [posted](https://github.com/gnolang/gno/pull/5710#discussion_r3432195700)
These `GAS USED` pins are whole-tx totals, so any unrelated gas-model change reds the test, and a single total never isolates the per-entry restore charge they are meant to guard. The sibling [`compute_map_key_big_bytes`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/tests/files/gas/compute_map_key_big_bytes.gno) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/tests/files/gas/compute_map_key_big_bytes.gno) / `compute_map_key_small_bytes` filetests are already off by +113 on master. Nit: the header says ~4728 for Tx2; it's actually ~5516.

## gnovm/pkg/gnolang/values_test.go:409-435 [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/values_test.go#L409) [posted](https://github.com/gnolang/gno/pull/5710#discussion_r3432195702)
This only checks that one scalar `int` key is charged something. It never charges the same key on both the write and restore paths and asserts the two are equal, so a future change that diverges them passes silently.

## gnovm/pkg/gnolang/store.go:73-77 [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L73) [posted](https://github.com/gnolang/gno/pull/5710#discussion_r3432195704)
The comment "Store does not charge gas itself" is wrong: [`consumeGas`](https://github.com/gnolang/gno/blob/bac5ab8d1/gnovm/pkg/gnolang/store.go#L1104-L1108) · [↗](../../../../../.worktrees/gno-review-5710/gnovm/pkg/gnolang/store.go#L1104) charges amino gas through this same store.
