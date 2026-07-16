# PR [#5938](https://github.com/gnolang/gno/pull/5938): feat(gno.land): mount the bptree store with the fast index; reprice depth gas

URL: https://github.com/gnolang/gno/pull/5938
Author: jaekwon | Base: bptree-fastindex-working-tree | Files: 20 | +426 -100
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 27c5ece7e (latest — merged as squash `1e2e00e2f`)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5938 27c5ece7e`

Round 1 (deep — red-team, migration, blue-team and correctness lenses, plus a critic round and a claim-verification gate). The PR is merged; reviewed on its merits anyway, and the findings below are written as things to fix forward. It stacks on [#5937](https://github.com/gnolang/gno/pull/5937) (merged `dc305b6d6`), reviewed separately; the diff against its base carries none of 5937's files, so everything here is this PR's own delta.

**TL;DR:** gno.land keeps all its on-chain data in one big tree. This swaps that tree for a different, shallower design, turns on a side index that makes single-key lookups cheap, and re-prices what a transaction pays to read and write state.

**Verdict: REQUEST CHANGES** — mounting the bptree store puts a full scan of every retained block version on the production RPC path, ~100ms per query at 100K retained versions against IAVL's flat ~14µs and growing linearly toward the 705,600 the default prune strategy holds; separately the SET-read gas pin is 30% under the measurement its own comment cites.

## Summary

gno.land's `mainKey` state store moves from IAVL to the B+32 bptree with the fast index, and the three depth-gas genesis defaults are re-pinned: GET 3.0 → 1.0 read ops, SET-read stays 2.0, WRITE 4.4 → 5.4. Both changes are consensus-affecting, so the PR is correctly scoped to fresh chains and export/import forks. The gas arithmetic is internally consistent and the seven rebaselined goldens all move by exact multiples of the per-op deltas (−118,000 gas per uncached GET, +24,000 per SET), which is what "identical workloads, only prices changed" should look like.

Two problems survive that. The mount is what promotes tm2's per-open version discovery onto the production ABCI query path: every custom query re-opens the store at a height, and the bptree store's immutable load calls `MutableTree.Load` first, which scans every retained root record twice. IAVL's immutable load never did. And the SET-read pin of 2.0 is the *modeled* number that the PR's own cited provenance measured at 2.86 and explicitly corrected; the PR ships the model while newly labelling it "measured".

## Glossary

- app hash: the per-block commitment to application state; two honest nodes disagreeing on it halts the chain.
- bptree (B+32): the versioned copy-on-write B+tree with 32-way fanout now backing `mainKey`; different commitment structure from IAVL, so every app hash moves.
- fast index: the bptree's key-to-value side map serving a present-key point read in one flat DB read; outside the Merkle commitment, so it never moves the app hash or charged gas.
- depth-metered store: a store whose cache wrap charges tree-depth-scaled I/O gas rather than a flat cost.
- IAVL: the incumbent versioned Merkle binary tree this PR replaces on `mainKey`.
- hard fork: a change to a persisted format or a consensus gas constant; ships only on fresh genesis or a coordinated upgrade.
- txtar: the testscript integration tests under `gno.land/pkg/integration/testdata/`.

## Benchmarks / Numbers

Per-op gas deltas at the shipped pins (`ReadCostFlat` 59,000, `WriteCostFlat` 24,000):

| op | before | after | delta |
|---|---|---|---|
| uncached GET | 3.0 × 59,000 = 177,000 | 1.0 × 59,000 = 59,000 | −118,000 |
| SET (read part) | 2.0 × 59,000 = 118,000 | unchanged | 0 |
| SET (write part) | 4.4 × 24,000 = 105,600 | 5.4 × 24,000 = 129,600 | +24,000 |

Every rebaselined golden resolves to non-negative integer `(G, S)` under `−118,000·G + 24,000·S = Δ`:

| golden | before | after | Δ | implied |
|---|---|---|---|---|
| `restart_gas` addpkg | 2,780,212 | 2,476,212 | −304,000 | G=4, S=7 |
| `gnokey_gasfee` addpkg | 2,756,592 | 2,452,592 | −304,000 | G=4, S=7 |
| `gnokey_gasfee` call | 1,212,011 | 1,024,011 | −188,000 | G=2, S=2 |
| `gc` | 151,321,803 | 151,133,803 | −188,000 | G=2, S=2 |
| `stdlib_ibc_crypto_determinism` | 2,739,422 | 2,551,422 | −188,000 | G=2, S=2 |
| `stdlib_restart_compare` | 2,176,646 | 2,012,646 | −164,000 | G=2, S=3 |

Pins against the measurement the code cites as provenance ([PERFORMANCE.md](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L61-L67) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L61)):

| pin | shipped | modeled | measured @101M | verdict |
|---|---|---|---|---|
| `FixedGetReadDepth100` | 100 | 1.00 | 1.00 (by construction) | matches |
| `FixedSetReadDepth100` | 200 | 2.0 | **2.86** | 30% under |
| `FixedWriteDepth100` | 540 | 4.4 + 1.0 | 4.36 + 1.0 = 5.36 | ~1% over, fine |

Per-query store open, by retained-version count (`MultiImmutableCacheWrapWithVersion`, the exact call every custom ABCI query makes):

| retained versions | IAVL | bptree + fast index |
|---|---|---|
| 1,000 | 19.0µs | 209µs |
| 20,000 | 14.5µs | 17.6ms |
| 100,000 | 14.1µs | 100.9ms |

## Critical (must fix)

- **[every RPC query gets slower forever as the chain runs]** `gno.land/pkg/gnoland/app.go:106` — mounting the bptree store puts a full scan of every retained block version on the ABCI query path; at 100K retained versions a query costs 100.9ms against IAVL's flat 14.1µs, and the default prune strategy retains 705,600.
  <details><summary>details</summary>

  Every custom ABCI query re-opens the store at a height: [`handleQueryCustom`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/sdk/baseapp.go#L505) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/sdk/baseapp.go#L505) calls `MultiImmutableCacheWrapWithVersion(req.Height)`, defaulting `Height` to the latest block when the client sends none, so this is the path for `vm/qrender`, `vm/qeval` and `auth/accounts` alike. That reaches [`multiStore.LoadVersion`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/rootmulti/store.go#L258) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/rootmulti/store.go#L258), which constructs a fresh store per mounted key and calls its `LoadVersion`. IAVL's takes the immutable branch straight to [`GetImmutable(ver)`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/iavl/store.go#L177-L184) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/iavl/store.go#L177) — one root fetch. The bptree store's calls [`st.mtree.Load()`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/bptree/store.go#L187-L189) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/bptree/store.go#L187) first, and [`MutableTree.Load`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/mutable_tree.go#L487-L509) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/mutable_tree.go#L487) runs [`discoverVersions`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/nodedb.go#L473-L495) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/nodedb.go#L473), an unbounded iterator over every `PrefixRoot` key, then calls `LoadVersion(latest)` which scans a second time.

  The cost is linear in retained versions, and [`PruneSyncable`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/types/options.go#L42) keeps 705,600 of them — [the gno.land default](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app.go#L63) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app.go#L63) — so a node saturates about eight days in at one-second blocks, seven times beyond the largest point I measured. Archive nodes never stop growing. It runs under the mutex [`QuerySync`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bft/abci/client/local_client.go#L176-L182) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bft/abci/client/local_client.go#L176) shares with `DeliverTxSync` and `CommitSync`, so the scan blocks block production and an unauthenticated caller can hold it by issuing queries. The scan is tm2 code this PR does not touch, but `app.go:106` is what puts it in front of mainnet RPC — the mount and the fix have to land together. Test: [`query_path_version_scan_test.go`](tests/query_path_version_scan_test.go), red at this sha, green when the per-query open stops scanning. Fix: seek to the first and last root record instead of iterating them, or cache first/latest on the store and skip discovery on the immutable open.
  </details>

## Warnings (should fix)

- **[consensus gas is 30% under its own measurement]** `gno.land/pkg/sdk/vm/params.go:41` — the SET-read pin ships the modeled 2.0 that the cited provenance measured at 2.86 and corrected, and the comment newly calls it "measured".
  <details><summary>details</summary>

  [`minSetReadDepth100Default = int64(200) // 2.0 SET read ops (descent, measured with 10K cache)`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L41) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L41) is cited to [`PERFORMANCE.md`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L35) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L35), which states that where model and measurement disagree the measured number is authoritative. That file measures [SET reads at 2.86](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L66) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L66) at the same ~101M-key calibration point the pin claims, and records the correction explicitly: ["B+32 SET reads: modeled 2.0 → measured 2.86"](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L101-L102) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L101).

  The other two pins do track the measurement, which is what makes this one stand out: GET 1.00 is one flat read by construction, and WRITE 540 sits within 1% of the measured 4.36 + 1.0 index write. Only SET-read ships the pre-correction model, and the diff adds the word "measured" to a line that previously read just `// 2.0 SET read ops`. At `ReadCostFlat` 59,000 the gap is 0.86 read ops = ~50,700 gas undercharged on every SET, ~17% of the SET's total 247,600. These are consensus genesis defaults, so correcting later costs a governance re-tune or a fork. Fix: pin 286, or keep 200 and say in the comment that it is the model and knowingly under the measured cost.
  </details>

- **[master's history describes gas that does not exist]** `gno.land/pkg/sdk/vm/params.go:82-84` — the merged commit message says SET/WRITE are estimator-driven at Fixed=0; the code pins Fixed = Min = 200/540, so the estimator never runs.
  <details><summary>details</summary>

  [`NewParams`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L82-L84) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L82) sets `FixedSetReadDepth100` and `FixedWriteDepth100` from the Min values, and [`effectiveSetReadDepth100`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/cache/store.go#L105-L108) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/cache/store.go#L105) returns the Fixed value whenever it is above zero, so at 200/540 the tree estimate is unreachable. `TestDefaultParams` asserts exactly that. This is a stale description of an abandoned iteration rather than a code defect: at `08e0e0f65` `NewParams` really did leave Fixed at zero, and `c3094bfcf` ("Merge origin/bptree-mount: keep final pinned-depth design") moved to the pins without updating the body.

  It matters because the squash landed the body verbatim as the commit message on master, so anyone reading `git log` concludes consensus gas self-corrects as state grows when it is in fact hard-pinned and needs governance to move. Two riders travel with it: the claim that `TestDefaultParams` pins "zero Fixed values are fully representable ... incl. a keeper round-trip" describes a test that only ever round-trips the non-zero defaults, and the trailing `ADR: gno.land/adr/prxxxx_mount_bptree_store.md` points at a filename that was never committed. Fix: a corrective note on the PR, since the squashed message cannot be edited in place.
  </details>

## Nits

- **[a reader can't tell which half of the file to believe]** `tm2/pkg/bptree/benchmarks/BENCHMARKS.md:3-8` — the new status header says the 100M run "has since been measured", but [`## TODO — the 100M run (the one that matters)`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/benchmarks/BENCHMARKS.md?plain=1#L247) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L247) and ["Confirm against the 100M run."](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/benchmarks/BENCHMARKS.md?plain=1#L264) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L264) are still there. The header dropped its link to that section but left the section standing.

## Missing Tests

- **[the fingerprint rule is enforced by a comment]** `contribs/gnogenesis/internal/fork/generate.go:676-681` — nothing reddens when the vm depth defaults change without a matching era fingerprint being appended, which is the exact bug this PR is fixing for the previous era.
  <details><summary>details</summary>

  [`untunedDepthFingerprints`](https://github.com/gnolang/gno/blob/27c5ece7e/contribs/gnogenesis/internal/fork/generate.go#L676-L698) · [↗](../../../../../.worktrees/gno-review-5938/contribs/gnogenesis/internal/fork/generate.go#L676) says "A new era fingerprint must be appended whenever the vm defaults change", and [`params.go:38-39`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L38-L39) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L38) repeats it from the other side. Both are comments. The next defaults change that forgets leaves a chain forked from the bptree era carrying 100/200/540 verbatim onto a chain priced differently, silently — the same failure this PR is repairing for the 300/200/440 era, and the reason the repair was needed is that the previous era had no such guard either.
  </details>

## Verified

- Reverting the mount is what moves the query cost, not the reprice: swapping only the constructor in `MultiImmutableCacheWrapWithVersion` between `iavl.StoreConstructor` and `storebptree.FastStoreConstructor` takes the per-query open from a flat 14.1µs to 100.9ms at 100K retained versions, with the bptree row growing 482x across the range and the IAVL row flat.
- The estimator is unreachable at the shipped defaults: `effectiveGetReadDepth100`/`effectiveSetReadDepth100`/`effectiveWriteDepth100` each return the Fixed value when it is above zero, and all three Fixed values are non-zero, so `expectedDepth100(tree.Size())` never enters charged gas and the fast index cannot move it.
- Every rebaselined golden's delta decomposes into whole GET and SET counts against the derived per-op deltas, with no residue — consistent with the workloads being unchanged and only prices moving.
- Green at `27c5ece7e`: `go test ./gno.land/pkg/sdk/vm/ -run 'TestDefaultParams'`.

## Open questions

- The GET pin prices a present-key hit at 1.0 read while an absent-key GET still walks ~3.66 reads and is charged the same 1.0. Not posted: it is a deliberate, documented trade with a named fix, and the worst case is better than the IAVL-era status quo it replaces.
- `iavlCapKey` survives as the store-key name in `app_test.go` and `common_test.go` while mounting bptree. Not posted: pure naming, no risk, and it churns lines a future backend swap would touch anyway.
