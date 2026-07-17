# PR [#5938](https://github.com/gnolang/gno/pull/5938): feat(gno.land): mount the bptree store with the fast index; reprice depth gas

URL: https://github.com/gnolang/gno/pull/5938
Author: jaekwon | Base: bptree-fastindex-working-tree | Files: 20 | +426 -100
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: 27c5ece7e (latest — merged as squash `1e2e00e2f`)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5938 27c5ece7e`

Round 1 (deep — red-team, migration, blue-team and correctness lenses, plus a critic round and a claim-verification gate). The PR is merged as squash `1e2e00e2f`; reviewed on its merits anyway. Every Critical and Warning below names a fix that still applies to master as it stands, since the work lands as follow-up PRs rather than changes to this one. It stacks on [#5937](https://github.com/gnolang/gno/pull/5937) (merged `dc305b6d6`), reviewed separately; the diff against its base carries none of 5937's files, so everything here is this PR's own delta.

**TL;DR:** gno.land keeps all its on-chain data in one big tree. This swaps that tree for a different, shallower design, turns on a side index that makes single-key lookups cheap, and re-prices what a transaction pays to read and write state.

**Verdict: REQUEST CHANGES** — mounting the bptree store puts a scan of every retained block version on the ABCI query path, growing ~480x between 1K and 100K retained versions where IAVL is flat, and the default prune strategy retains 705,600; the scan has to land a fix before any chain launches from this mount. Separately the SET-read gas pin is 30% under the measurement its own comment cites.

The standard I applied to the two pricing trades this PR makes: an imprecision that still improves on what it replaces is an accepted trade and stays in Open questions, while a regression against what it replaces is a finding. The absent-key GET is charged 1.0 while it walks, but its worst case beats the IAVL era, so it is not a finding here. The query path is roughly three orders of magnitude worse than the IAVL mount it replaces, so it is.

## Summary

gno.land's `mainKey` state store moves from IAVL to the B+32 bptree with the fast index, and the three depth-gas genesis defaults are re-pinned: GET 3.0 → 1.0 read ops, SET-read stays 2.0, WRITE 4.4 → 5.4. Both changes are consensus-affecting, so the PR is correctly scoped to fresh chains and export/import forks. The gas arithmetic is internally consistent and every rebaselined golden moves by an exact combination of the per-op deltas (−118,000 gas per uncached GET, +24,000 per mutation), which is what "identical workloads, only prices changed" should look like.

Two problems survive that. The mount is what promotes tm2's per-open version discovery onto the production ABCI query path: every custom query re-opens the store at a height, and the bptree store's immutable load calls `MutableTree.Load` first, which scans every retained root record twice. IAVL's immutable load never did, so this is the one place the swap moves sharply backwards. And the SET-read pin of 2.0 is the *modeled* number that the PR's own cited provenance measured at 2.86 and explicitly corrected; the PR inherits the value untouched and the mount makes it far more accurate than it was under IAVL, but the comment it does change now calls the model "measured".

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

Every rebaselined golden resolves to non-negative integers under `−118,000·G + 24,000·S = Δ`, where G counts uncached GETs and S counts gas-charged mutations (sets plus deletes):

| golden | before | after | Δ | implied |
|---|---|---|---|---|
| `restart_gas` addpkg | 2,780,212 | 2,476,212 | −304,000 | G=4, S=7 |
| `gnokey_gasfee` addpkg | 2,756,592 | 2,452,592 | −304,000 | G=4, S=7 |
| `gnokey_gasfee` call | 1,212,011 | 1,024,011 | −188,000 | G=2, S=2 |
| `gc` | 151,321,803 | 151,133,803 | −188,000 | G=2, S=2 |
| `stdlib_ibc_crypto_determinism` | 2,739,422 | 2,551,422 | −188,000 | G=2, S=2 |
| `stdlib_restart_compare` | 2,176,646 | 2,012,646 | −164,000 | G=2, S=3 |

Pins against the measurement the code cites as provenance ([PERFORMANCE.md](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L60-L65) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L60)):

| pin | shipped | modeled | measured @101M | verdict |
|---|---|---|---|---|
| `FixedGetReadDepth100` | 100 | 1.00 | 1.00 (by construction) | matches |
| `FixedSetReadDepth100` | 200 | 2.0 | **2.86** | 30% under |
| `FixedWriteDepth100` | 540 | 4.4 + 1.0 | 4.36 + 1.0 = 5.36 | ~1% over, fine |

Per-query store open, by retained-version count (`MultiImmutableCacheWrapWithVersion`, the exact call every custom ABCI query makes). Absolute figures are machine-dependent and a second box came in ~2.5x faster; the growth column is the invariant:

| retained versions | IAVL | bptree + fast index |
|---|---|---|
| 1,000 | 19.0µs | 209µs |
| 20,000 | 14.5µs | 17.6ms |
| 100,000 | 14.1µs | 100.9ms |
| **growth 1K → 100K** | **flat** | **~480x** |

## Critical (must fix)

- **[queries get slower forever as the chain runs]** `gno.land/pkg/gnoland/app.go:106` — mounting the bptree store puts a scan of every retained block version on the custom ABCI query path; the per-query open grows ~480x between 1K and 100K retained versions where IAVL is flat, and the default prune strategy retains 705,600.
  <details><summary>details</summary>

  Every custom ABCI query re-opens the store at a height: [`handleQueryCustom`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/sdk/baseapp.go#L505) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/sdk/baseapp.go#L505) calls `MultiImmutableCacheWrapWithVersion(req.Height)`, defaulting `Height` to the latest block when the client sends none, so this is the path for `vm/qrender`, `vm/qeval` and `auth/accounts` alike. That reaches [`multiStore.LoadVersion`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/rootmulti/store.go#L258) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/rootmulti/store.go#L258), which constructs a fresh store per mounted key and calls its `LoadVersion`. IAVL's takes the immutable branch straight to [`GetImmutable(ver)`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/iavl/store.go#L177-L184) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/iavl/store.go#L177) — one root fetch. The bptree store's calls [`st.mtree.Load()`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/bptree/store.go#L187-L189) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/bptree/store.go#L187) first, and [`MutableTree.Load`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/mutable_tree.go#L487-L509) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/mutable_tree.go#L487) runs [`discoverVersions`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/nodedb.go#L473-L495) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/nodedb.go#L473), an unbounded iterator over every `PrefixRoot` key, then calls `LoadVersion(latest)` which scans a second time.

  The cost is linear in retained versions, and [`PruneSyncable`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/types/options.go#L42) keeps 705,600 of them — [the gno.land default](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app.go#L63) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app.go#L63) — so a node saturates about eight days in at one-second blocks, seven times beyond the largest point measured. Archive nodes never stop growing. It runs under the mutex [`QuerySync`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bft/abci/client/local_client.go#L176-L182) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bft/abci/client/local_client.go#L176) shares with `DeliverTxSync` and `CommitSync`, so the scan blocks block production and an unauthenticated caller can hold it by issuing queries. Absolute latency is machine-dependent — 100.9ms at 100K on my box, ~41ms on another — so the growth factor, not the millisecond figure, is the load-bearing number. `.store` queries escape it: [`handleQueryStore`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/sdk/baseapp.go#L457) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/sdk/baseapp.go#L457) serves from the live multistore without re-opening.

  Scope and standing: the PR's diff touches none of `nodedb.go`, `mutable_tree.go` or `store/bptree/store.go` — its whole contribution to this is the one constructor on this line, and the fix belongs in tm2. The mount is consensus-breaking and fresh-genesis-only, so no chain running today inherits this; the exposure begins with the next chain launched from it. That is what makes it a sequencing blocker rather than an incident, and both the mount and the scan are unchanged on master today, so the follow-up still has to happen. Test: [`query_path_version_scan_test.go`](tests/query_path_version_scan_test.go), red at this sha, green when the per-query open stops scanning. Fix: seek to the first and last root record rather than iterating every one, and drop the `Load()` call from the immutable branch of `bptree.Store.LoadVersion`, which needs the requested version's root and not the latest.
  </details>

## Warnings (should fix)

- **[a consensus gas pin labelled measured is the model]** `gno.land/pkg/sdk/vm/params.go:41` — the comment newly calls the SET-read pin measured, but the cited provenance measures 2.86 and records correcting the model's 2.0; the gap undercharges every SET by ~50,700 gas.
  <details><summary>details</summary>

  [`minSetReadDepth100Default = int64(200) // 2.0 SET read ops (descent, measured with 10K cache)`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L41) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L41) is cited to [`PERFORMANCE.md`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L34-L37) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L34), which states that where model and measurement disagree the measured number is authoritative. That file measures [SET reads at 2.86](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L64) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L64) at the same ~101M-key calibration point the pin claims, and records the correction explicitly: ["B+32 SET reads: modeled 2.0 → measured 2.86"](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L101-L102) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L101).

  The other two pins do track the measurement, which is what makes this one stand out: GET 1.00 is one flat read by construction, and WRITE 540 sits within 1% of the measured 4.36 + 1.0 index write. Only SET-read ships the pre-correction model. At `ReadCostFlat` 59,000 the gap is 0.86 read ops = ~50,700 gas undercharged on every SET, ~17% of the SET's total 247,600.

  Two things keep this off the blocker list. The PR does not choose 200: it inherits the value untouched and changes only the comment, which previously read just `// 2.0 SET read ops`. And the mount improves this pin's accuracy sharply rather than degrading it — [PERFORMANCE.md measures IAVL SET reads at 34.1](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L64) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L64), so the same 2.0 pin went from 17x under to 1.43x under. It is a Warning because the new "measured" label is false and because the value is a consensus default, not because the PR made pricing worse. Correctable live: [`p:fixed_set_read_depth_100`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L272-L273) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L272) is a governance-settable param on a running chain, so this needs no fork. Still live on master. Fix: pin 286 and rebaseline the goldens, or keep 200 and drop "measured" from the comment, saying it is the model and knowingly under the measured cost.
  </details>

- **[master's history describes gas that does not exist]** `gno.land/pkg/sdk/vm/params.go:82-84` — the merged commit message says SET/WRITE are estimator-driven at Fixed=0; the code pins Fixed = Min = 200/540, so the estimator never runs.
  <details><summary>details</summary>

  [`NewParams`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L82-L84) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L82) sets `FixedSetReadDepth100` and `FixedWriteDepth100` from the Min values, and [`effectiveSetReadDepth100`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/cache/store.go#L105-L108) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/cache/store.go#L105) returns the Fixed value whenever it is above zero, so at 200/540 the tree estimate is unreachable. `TestDefaultParams` asserts exactly that. This is a stale description of an abandoned iteration rather than a code defect: at `08e0e0f65` `NewParams` really did leave Fixed at zero, and `c3094bfcf` ("Merge origin/bptree-mount: keep final pinned-depth design") moved to the pins without updating the body.

  It matters because the squash landed the body verbatim as the commit message on master, so anyone reading `git log` concludes consensus gas self-corrects as state grows when it is in fact hard-pinned and needs governance to move. Two riders travel with it: the claim that `TestDefaultParams` pins "zero Fixed values are fully representable ... incl. a keeper round-trip" describes a test that only ever round-trips the non-zero defaults, and the trailing `ADR: gno.land/adr/prxxxx_mount_bptree_store.md` points at a filename that was never committed. Fix: a corrective note on the PR, since the squashed message cannot be edited in place.
  </details>

- **[bptree prices could silently end up on an IAVL chain]** `gno.land/pkg/gnoland/app_test.go:1517` — the depth pins are calibrated for this backend, but nothing asserts the app mounts it; reverting to IAVL leaves every gas and apphash golden green while SET-reads run 17x underpriced.
  <details><summary>details</summary>

  The pins encode bptree's costs specifically: GET 1.0 is a fast-index hit, WRITE 5.4 is 4.4 copy-on-write writes plus the index write. Charged gas never reads the tree, because [`effectiveSetReadDepth100`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/cache/store.go#L105-L108) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/cache/store.go#L105) short-circuits on the non-zero Fixed pin, so gas is a pure function of the params and comes out identical on either backend. That is exactly what leaves the pairing unenforced: reverting [the mount](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app.go#L106) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app.go#L106) to `iavl.StoreConstructor` keeps the four rebaselined goldens green and [`TestAppHashCrossrealm38`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L74) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L74) green too, since that test builds its store from [`common_test.go`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/common_test.go#L50) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/common_test.go#L50) rather than from the app. A chain in that state charges bptree prices for IAVL work, and [PERFORMANCE.md measures IAVL SET reads at 34.1](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L64) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L64) against the pinned 2.0.

  The one thing that reddens is `TestPruneStrategyNothing`, by accident: [its own multistore](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app_test.go#L1517) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app_test.go#L1517) mounts bptree over a DB the reverted app filled with IAVL data, so it fails on a format collision at [`app_test.go:1521`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app_test.go#L1521) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app_test.go#L1521) with a message naming no cause. Still live on master. Fix: assert in `gno.land/pkg/gnoland` that the app's `mainKey` store is a bptree store with the index on, so a revert names itself.
  </details>

- **[the fork tool rewrites consensus gas prices without saying so]** `contribs/gnogenesis/internal/fork/generate.go:453-457` — the reprice mutates the forked chain's gas params silently, while the same function narrates smaller decisions.
  <details><summary>details</summary>

  [The rewrite loop](https://github.com/gnolang/gno/blob/27c5ece7e/contribs/gnogenesis/internal/fork/generate.go#L453-L457) · [↗](../../../../../.worktrees/gno-review-5938/contribs/gnogenesis/internal/fork/generate.go#L453) calls `applyDefaultDepthParams` and moves on. An operator forking a chain gets no indication that the depth-gas params in their output genesis differ from the ones in their input, even though this is the fork's most consensus-relevant silent mutation. The fingerprint match is a heuristic — it infers "untuned" from an exact value match — so the one case where it guesses wrong is also the case with no trace in the output. Fix: log the before and after values when the rewrite fires.
  </details>

## Nits

- **[a reader can't tell which half of the file to believe]** `tm2/pkg/bptree/benchmarks/BENCHMARKS.md:3-8` — the new status header says the 100M run "has since been measured", but [`## TODO — the 100M run (the one that matters)`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/benchmarks/BENCHMARKS.md?plain=1#L247) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L247) and ["Confirm against the 100M run."](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/benchmarks/BENCHMARKS.md?plain=1#L264) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L264) are still there. The header dropped its link to that section but left the section standing.

- **[the new comments describe a failure the test never has]** `gno.land/pkg/integration/testdata/addpkg_outofgas.txtar:10-12` — both rewritten comments are disproven by the run: neither case fails "early in store access", and the second is not "slightly later" than the first.
  <details><summary>details</summary>

  The PR rewrites the two case comments to ["runs out of gas early in store access with gas 60000"](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L10-L12) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L10) and ["runs out of gas slightly later with gas 63000"](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L22) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L22). Running it, both cases report the same `GAS USED: 2847971` and fail with `gas used (2847971) exceeds tx's gas wanted (N) during operation: simulation`. gnokey simulates first, so the tx runs to completion under the simulation meter and gnokey synthesizes the out-of-gas error from the comparison; the 60000 and 63000 meters are never applied to VM execution, and the two cases differ only in the number printed in the message.

  The old comments named specific store calls and were also wrong, so the PR improved them; the note about the failure point shifting with the depth params is the honest part. But the replacement still asserts a mechanism the run contradicts. The hollowness is not this PR's doing: gnokey has simulated by default since `3ea1b47e2`, and 2,847,971 is far above 60000 at any pricing this PR could produce, so the file has covered nothing but gnokey's own arithmetic for a long time. The PR touches exactly these two comment lines, which is what makes it the moment to make them true. Fix: say the tx is rejected before execution because simulated gas exceeds gas-wanted, or add `-simulate skip` to both cases so they exercise the in-VM meter the file's title promises.
  </details>

## Missing Tests

- **[the fingerprint rule is enforced by a comment]** `contribs/gnogenesis/internal/fork/generate.go:676-681` — nothing reddens when the vm depth defaults change without a matching era fingerprint being appended, which is the exact bug this PR is fixing for the previous era.
  <details><summary>details</summary>

  [`untunedDepthFingerprints`](https://github.com/gnolang/gno/blob/27c5ece7e/contribs/gnogenesis/internal/fork/generate.go#L676-L698) · [↗](../../../../../.worktrees/gno-review-5938/contribs/gnogenesis/internal/fork/generate.go#L676) says "A new era fingerprint must be appended whenever the vm defaults change", and [`params.go:38-39`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L38-L39) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L38) repeats it from the other side. Both are comments. The next defaults change that forgets leaves a chain forked from the bptree era carrying 100/200/540 verbatim onto a chain priced differently, silently — the same failure this PR is repairing for the 300/200/440 era, and the reason the repair was needed is that the previous era had no such guard either.

  Test: [`depth_fingerprint_decay_test.go`](tests/depth_fingerprint_decay_test.go) pins the defaults as an append-only history and requires every superseded entry to appear in `untunedDepthFingerprints`. Green at this sha, red the moment a depth default moves without its predecessor being appended.
  </details>

## Verified

- The backend, not the reprice, is what moves the query cost: with only the store constructor differing, the per-query open is flat at ~14µs on IAVL across 1K–100K retained versions and rises 482x on bptree over the same range, reaching 100.9ms at 100K.
- Reverting the mount at `app.go:106` to `iavl.StoreConstructor` leaves the four rebaselined gas goldens and `TestAppHashCrossrealm38` green; only `TestPruneStrategyNothing` fails, on a DB-format collision that names no cause.
- Both rewritten `addpkg_outofgas` cases report the identical `GAS USED: 2847971` and are rejected by gnokey's simulate check, so neither runs under the 60000 or 63000 meter its comment describes.
- Green at `27c5ece7e`: the seven golden txtars this PR touches, `go test ./gno.land/pkg/sdk/vm/ -run 'TestDefaultParams'`, and `contribs/gnogenesis`'s fork suite.

## Open questions

- The GET pin prices a present-key hit at 1.0 read while an absent-key GET still walks ~3.66 and is charged the same 1.0, a wider gap than the SET-read Warning above. Not posted: it is a deliberate trade with a named fix, and unlike the query path its worst case still improves on the IAVL era, which is the line this review draws between an accepted imprecision and a finding.
- `iavlCapKey` survives as the store-key name in `app_test.go` and `common_test.go` while mounting bptree. Not posted: pure naming, no risk, and it churns lines a future backend swap would touch anyway.
- I could not isolate the two named causes of the crossrealm38 hash bump, so the review does not claim the new hash has no third cause. Not posted: the bump has two sufficient explanations and the underlying filetest still passes; the gap is in what the golden can prove, not in the value.
- The verdict was contested during review and the counter-argument is worth recording: the query scan lives entirely in tm2 code this PR does not touch, the author disclosed and ranked it as the top follow-up before merging, and the mount is fresh-genesis-only, so APPROVE with the scan named as a pre-launch blocker is defensible. I kept REQUEST CHANGES because a chain can launch from this mount before the follow-up lands, and the ~480x measured growth under the consensus mutex is what turns a ranked follow-up into a gate on that ordering. The measured latency is the review's own contribution; the mechanism was already known.
