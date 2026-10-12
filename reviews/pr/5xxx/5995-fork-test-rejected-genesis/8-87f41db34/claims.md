# Claims: gnolang/gno#5995 round 8, 87f41db34, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 2 finders and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | tm2/pkg/bft/consensus/replay.go:353 | copy TestE2ERejectThenCorrectedGenesisBoots with an app returning req.Validators and a provider that sets doc.Validators = nil on the first build and keeps them on the second; expect the second build to succeed | head: 'validators seen by InitChain per boot: [0 0]', second build 'error during handshake: error on replay: validator set is nil in genesis and still empty after InitChain', FAIL. fixed: '[0 1]', PASS | tests/judge-1-nil-valset-refusal.go | hot |  |
| 1001 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:1144 | sed -n 1139,1147p and 1171,1178p tm2/pkg/bft/node/node.go: saveGenesisDoc returns nothing and discards db.SetSync's error; run: memdb wrapped so SetSync(genesisDocKey) fails once, refuse InitialHeight 1000, reset to 100, SaveABCIResponses(0), reload and read doc.InitialHeight | head: 'reset: doc.InitialHeight=100 state.InitialHeight=100', 'restart: doc.InitialHeight=1000 state.InitialHeight=100', expected: 100 actual: 1000, FAIL. fixed: 'reset surfaced the failed write: injected I/O error', PASS | tests/judge-1001-dropped-doc-write.go | hot |  |
| 1002 | CONFIRMED | Nit | tm2/adr/pr5949_remove_secp256k1_validators.md:71 | grep -rn 'silently' tm2/adr/pr5949_remove_secp256k1_validators.md; sed -n 343,349p tm2/pkg/bft/consensus/replay.go | pr5949 ADR:71 '`ResponseInitChain.Error` is silently' / :72 'discarded by tm2, so the check **panics**'; replay.go:346 'if res.IsErr() {' |  |  |  |
| 2003 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:1145 | read store.go's write sites and the DB interface | grep -n 'db.Set(\\|db.SetSync(' tm2/pkg/bft/state/store.go: 134, 203, 237, 330, 403, none checked; tm2/pkg/db/types.go:18-19 Set and SetSync return error |  | hot |  |
| 2001 | REFUTED | Warning | tm2/pkg/bft/state/store.go:83 | sed -n 82,90p tm2/pkg/bft/state/store.go | store.go:84-85 if err != nil { panic(err) } |  |  |  |

Hit rate per tier, from the rows above: hot 3/3 confirmed over 3 files, warm 0/0 confirmed over 3 files, cold 0/0 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | tm2/pkg/bft/node/node.go:1127 | a height-0 record with zero tx responses encodes to 0 bytes; a backend's Has (v != nil) could read it as absent and reset a committed chain | tests/solo-finder-1-empty-height0-record.go: len(Bytes())=0, yet after close+reopen genesisApplied=true on boltdb, goleveldb, lmdbdb, mdbxdb, pebbledb |
| claims | gno.land/pkg/gnoland/node_initial_height_test.go:129 | boot polls LastBlockHeight >= 999 and may compare app hashes read at different heights (999 vs proof block 1000) | tests/solo-finder-1-retry-test-height.go: first observed height=1000, app hash 38F9015A... unchanged over 8s; the proof block carries the post-999 hash, so either height yields the same hash |
| rollout | tm2/pkg/bft/node/node.go:1111 | gnoland start's on-disk genesis cache could serve the refused file's app state after the operator corrects genesis.json | gno.land/pkg/gnoland/genesis_state_ref.go:332-336 hash, err := hashFile(genesisPath); finalDir := filepath.Join(cacheRoot, hash): a corrected file gets a new cache |
| rollout | tm2/pkg/sdk/baseapp.go:352 | a refused hardfork boot's SetInitialVersion could persist and misalign a retry corrected to another InitialHeight | tm2/pkg/store/rootmulti/store.go:330-340 and iavl/store.go:147-154 set in-memory fields only (ms.initialVersion = v; st.initialVersion = v) |
| rollout | tm2/pkg/bft/consensus/replay.go:346 | StrictReplay refusals that old binaries ignored now abort boots of production nodes joining an existing network | grep -rn StrictReplay gno.land contribs tm2 (non-test): only the field at app.go:329 and the checks at 563 and 647; nothing sets it |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:83 | ADR orders gnolang/gno#1941 (height-0 save) before gnolang/gno#5684 (AppState stripped from the persisted doc) | git log -S: 3aa0f0b7d 2024-07-23 (gnolang/gno#1941) adds SaveABCIResponses(h.stateDB, 0; 5d0a4d703 2026-05-18 (gnolang/gno#5684) adds stripped.AppState = nil |
| claims | tm2/pkg/bft/node/node.go:1123 | genesisApplied godoc: the height-0 responses are never pruned | grep -rn '.Delete(\|DeleteSync(' tm2/pkg/bft/state tm2/pkg/bft/store tm2/pkg/bft/node (non-test): no hits |
| lines | gno.land/pkg/gnoland/node_inmemory.go:45 | another caller builds AppState as *GnoGenesisState and now fails to boot | grep -rn 'AppState: *&\|AppState = &' over the tree: no hits; NewDefaultGenesisConfig's one other caller, contribs/gnodev/pkg/dev/node.go:716-717, replaces AppState with a GnoGenesisState value |
| reach | tm2/pkg/bft/node/node.go:1127 | a genesis with zero tx responses saves an empty height-0 value (len 0), which a backend's Has (v != nil) could read as not applied, so every restart would reset a committed chain's state record | go test ./tm2/pkg/bft/node -run TestProbeEmptyGenesisMarker (CGO_ENABLED=1, scratch worktree at 87f41db34): 'len(NewABCIResponsesFromNum(0).Bytes()) = 0', then memdb, pebbledb, boltdb, lmdbdb, mdbxdb and goleveldb each print 'genesisApplied=true ... after restart LastBlockHeight=500 AppHash="committed-app-hash"' |
| reach | tm2/pkg/bft/node/node.go:1074 | a backend whose Get errors on a missing key would now fail the first boot of a fresh data dir rather than take the provider path | read each backend's Get: goleveldb maps errors.ErrNotFound to nil,nil (go_level_db.go:54), pebble maps pebble.ErrNotFound to nil,nil (pebbledb.go:64), lmdb and mdbx return nil on IsNotFound (lmdbdb.go:120, mdbxdb.go:121), boltdb and memdb return nil,nil |
| reach | tm2/pkg/bft/consensus/replay.go:348 | a refused InitChain could leave app-DB writes behind (BaseApp.InitChain runs SetInitialVersion and storeConsensusParams before the initChainer) that skew the retried boot on a persistent data dir | rootmulti/store.go:330-341 SetInitialVersion sets only in-memory fields; store.go:240-259 Commit drains the collector into one batch, and nothing commits after a refusal; loadStdlibs writes through msCache.MultiWrite into the deliverState cache (app.go:474-486) |
| reach | tm2/pkg/bft/consensus/replay.go:348 | a refused boot's genesis tx events could be indexed by the event store (started before the handshake) and then duplicated by the corrected boot | grep -rn 'EventTx{' tm2 gno.land (non-test): fired only at tm2/pkg/bft/state/execution.go:420, the block path, never from InitChain |
| removed | gno.land/pkg/gnoland/node_inmemory.go:47 | another site building AppState as *GnoGenesisState, or asserting NewDefaultGenesisConfig's AppState as a pointer, would be refused, or panic, now that the refusal stops boot | grep -rnE 'AppState(:\|\s*=)\s*&(gnoland\.)?GnoGenesisState' returned nothing; every non-test AppState assignment (start.go:491, node_testing.go:74/139, testscript_gnoland.go:296, gnodev node.go:717 and node_state.go:133, gnogenesis) assigns a value; the only pointer assertions (.(*gnoland.GnoGenesisState)) are testscript env values, not AppState |
| reach | tm2/pkg/bft/node/node.go:1126 | a pruner deleting ABCI responses by prefix would remove the height-0 marker and re-open the reset on a committed chain | grep -rn 'abciResponsesKey\\|PruneStates\\|pruneABCI\\|DiscardABCIResponses' (non-test): only the key builder at tm2/pkg/bft/state/store.go:35 |
| removed | tm2/pkg/bft/consensus/replay.go:348 | a live chain booted past a strict-replay refusal (which commits the delivered txs) would now refuse new nodes replaying its genesis | grep -rn StrictReplay (non-test, outside app.go): no setter anywhere; only tests enable it |
| removed | tm2/pkg/bft/consensus/replay.go:348 | a sibling InitChain caller still ignores the response Error | grep -rn '\.InitChain(\\|InitChainSync(' (non-test): replay.go:339 is the only consensus caller; local_client.go passes through; kvstore/helpers.go:34 is the example app's own setup |
| catalog | tm2/pkg/bft/node/node_test.go:563 | the new t.Parallel tests in tm2/pkg/bft/node and tm2/pkg/bft/consensus could race on shared state | go test -race -count=3 ./tm2/pkg/bft/node -run 'TestLoadStateFromDBOrGenesisDocProvider\|TestGenesisResetKeepsGoodDocOnInvalidFile\|TestE2ERejectThenCorrectedGenesisBoots' -> ok; ./tm2/pkg/bft/consensus -run 'TestHandshakeInitChainError\|TestHandshakeGenesisResponseDeliverTx' -> ok |

## Completeness

- Catalog walk: Determinism, Gas, Realm state safety, Caller and access control, Coin and banker, Storage deposit, VM-fault recoverability, VM semantics and Type-check touch no line of the diff. Error and panic handling is candidates 1001, 2001 and 2003. Twin input paths is gnoland start's streaming genesis path, decided as a separate issue and not reopened. Global mutable state: the finders' -race run over the new tests is in solo-finder-2's dropped rows.
- Siblings of the response-error refusal, swept by shape: every return between InitChainSync and the end of the appBlockHeight == 0 branch of ReplayBlocks. One sits after sm.SaveABCIResponses(h.stateDB, 0, ...): the 'validator set is nil' return at replay.go:367, candidate 1.
- Siblings of the fork test's transaction-count guard, swept by shape (len(...TxResponses), txCount, expected-vs-delivered comparisons) over contribs/gnogenesis, gno.land/pkg/gnoland, gno.land/cmd and tm2/pkg/bft/consensus outside tests: only progress counters in source_txs*.go and the mock app's index in replay.go:573-584. No second guard repeats the edited shape.
- Tests angle: with the read-error guard of 87f41db34 disabled (if false && err != nil ...), TestLoadStateFromDBOrGenesisDocProvider_GenesisDocReadErrorFailsBoot fails; with the reset disabled (if false && !applied), the gnoland test TestNodeRejectedGenesisRetryMatchesFreshBoot from 5fa47bd8e fails. Both new tests pin their lines.
- Written by the parent from `candidates/reflector.json`: the judge left this section at its placeholder, which the own-PR prompt never asked it to fill.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |
| aafa8a582 | 1 | `ReplayBlocks` refuses a genesis left with no validators before saving the height-0 ABCI responses, and the check after the save is gone; `TestE2ENoValidatorGenesisThenCorrectedBoots` fails with the check moved back after the save |
| 4b45296ca | 1001 | `saveGenesisDoc` returns the `SetSync` error and both callers fail the boot on it; `TestLoadStateFromDBOrGenesisDocProvider_ResetDocWriteErrorFailsBoot` fails with the error swallowed again |

Row 1002 stays: `tm2/adr/pr5949_remove_secp256k1_validators.md` records gnolang/gno#5949's own decision, and a historical ADR is left as written. Row 2003, `sm.SaveState` discarding its write error, is a convention of the whole state store, five write sites, predating the branch, and ships SKIP here.

## Retro

- What failed: the judge left Completeness at its placeholder and one row PLAUSIBLE, both closed by the parent; this is the second own-PR round in three to skip Completeness. 17 minutes against the projected 21.
- What worked: the claims and lines angles found two siblings of the branch's own refusal and read-error fixes, each confirmed by a run, and both ported tests fail under the matching mutation.
- Hit rate per tier: hot 3 of 3, rows 1, 1001 and 2003; row 1002 sits outside the diff and the risk table, and row 2001, refuted, carries no tier.
- Upgrade, applied: the own-PR branch of the solo judge's prompt in `scripts/workflows/review-pipeline.js` now asks it to replace the Completeness placeholder, which `round assemble` writes and nothing else fills. Estimate: one more Edit call per own-PR judge, under 1k output and about one turn of cache read, no change to the finding rate; one parent turn saved per own-PR round.
- Metrics: 3 agents, 98 turns, 163k output, 13.4M cache read, 0.54M cache write, about $10.24.
