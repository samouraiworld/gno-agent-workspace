# Claims: gnolang/gno#5995 round 3, 4c61d0582, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 2 finders and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | tm2/pkg/bft/node/node.go:1106 | go test ./tm2/pkg/bft/node -run TestSoloFinder1_HasError -v -count=1 | finder-1 artifact rerun at head: 'returned err=<nil> LastBlockHeight=0' / 'persisted after restart: LastBlockHeight=0 AppHash=""'. It does not build at the merge base (uses head-only probeGenesisDocProvider), so the base side is 1001's artifact: 'persisted LastBlockHeight=500 AppHash="committed-app-hash"', PASS. | tests/solo-finder-1-has-error-resets-state.go | warm |
| 2 | CONFIRMED | Missing test | tm2/pkg/bft/node/node_genesis_restart_test.go:38 | remove node.go:1088-1090 and run go test ./tm2/pkg/bft/node -run TestE2ERejectThenCorrectedGenesisBoots | mutation: node.go:1088-1090 removed; 'boot1 err=error during handshake: error on replay: InitChain rejected the genesis: InitialHeight mismatch: refusing to boot', '--- PASS: TestE2ERejectThenCorrectedGenesisBoots', '--- FAIL: TestLoadStateFromDBOrGenesisDocProvider_RejectedGenesisRetry'; tree restored. |  | cold |
| 1001 | CONFIRMED | Warning | tm2/pkg/bft/node/node.go:1106 | cp tests/solo-finder-2-has-error-resets-state.go tm2/pkg/bft/node/zz_has_error_test.go && go test ./tm2/pkg/bft/node -run TestZZHasErrorResetsCommittedChain -v -count=1 | head 4c61d0582, rerun from the file: 'boot err=<nil>; persisted LastBlockHeight=0 AppHash=""', expected 500 actual 0, --- FAIL. merge base cfcbaf484 via go test -overlay: 'boot err=<nil>; persisted LastBlockHeight=500 AppHash="committed-app-hash"', --- PASS. | tests/solo-finder-2-has-error-resets-state.go | warm |
| 1002 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:1088 | read: base tm2/pkg/bft/consensus/replay.go:339-347; head node.go:1088-1091 | base tm2/pkg/bft/consensus/replay.go:339-347: `res, err := proxyApp.Consensus().InitChainSync(req); if err != nil { return nil, err }` then `sm.SaveABCIResponses(h.stateDB, 0, abciResponse)`; head node.go:1088 `if !genesisApplied(stateDB) {` |  | warm |
| 1003 | CONFIRMED | Suggestion | tm2/pkg/bft/state/execution.go:170 | sed -n 168,180p and 450,458p tm2/pkg/bft/state/execution.go; sed -n 985,989p tm2/pkg/sdk/baseapp.go | sed execution.go:168-180 and 450-458: no res.IsErr() after CommitSync; baseapp.go:987 `res.Error = ABCIError(errors.New("baseapp expects MultiStore with 'base' Store"))`; app.go:107 `baseApp.MountStoreWithDB(baseKey, ...)` |  |  |
| 2001 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:1088 | cp tests/judge-2001-no-height0-marker-resets-state.go tm2/pkg/bft/node/zz_no_marker_test.go && go test ./tm2/pkg/bft/node -run TestJudgeNoHeight0MarkerKeepsState -v -count=1 | judge artifact at head: 'boot err=<nil>; persisted LastBlockHeight=0 AppHash=""', expected 500 actual 0, FAIL; merge base via -overlay: 'persisted LastBlockHeight=500 AppHash="committed-app-hash"', PASS; git log -S names 3aa0f0b7d 2024-07-23 | tests/judge-2001-no-height0-marker-resets-state.go | warm |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 0 files, warm 4/4 confirmed over 5 files, cold 1/1 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach: extremes | tm2/pkg/bft/node/node.go:1105 | a genesis with zero tx responses stores an empty height-0 blob (amino of ABCIResponses{} is 0 bytes), which a backend whose Has is Get()!=nil could read as absent, so every restart would reset the state | go test ./tm2/pkg/bft/node -run TestSoloFinder1_EmptyGenesisResponsesHas: 'len(NewABCIResponsesFromNum(0).Bytes()) = 0 (nil=true)', then memdb, pebbledb, goleveldb, boltdb, lmdbdb and mdbxdb each print 'Has=true err=<nil> genesisApplied=true' |
| reach: restart states | tm2/pkg/bft/node/node.go:1088 | a data dir created before height-0 ABCI responses were saved could reach the reset branch with committed blocks | git log -S'SaveABCIResponses(h.stateDB, 0' gives 3aa0f0b7d 2024-07-23 (gnolang/gno#1941); git log -S'genDoc.AppState == nil' gives 5d0a4d703 2026-05-18 (gnolang/gno#5684). The reset sits inside the AppState==nil branch, and only post-gnolang/gno#5684 data dirs persist a nil AppState for a gno.land genesis, by which point height 0 was already saved. |
| reach: restart states | tm2/pkg/bft/node/node.go:1104 | height-0 ABCI responses pruned on a long-running chain, so genesisApplied flips to false | grep -rn 'CalcABCIResponsesKey' (non-test) gives only store.go:34/179/203 and node.go:1105: no delete or prune of the key anywhere |
| reach: callers | tm2/pkg/bft/consensus/replay.go:346 | an in-tree gno.land boot path with a nil or *GnoGenesisState AppState that used to boot past the ignored Error and now aborts | Every non-test GenesisDoc literal in gno.land/contribs/misc sets AppState (node_testing.go:120, node_inmemory.go:37, common/config.go:31; genesis_state_ref.go:339 and start.go:423 assign it at :353 and :491). grep '\.\(\*(gnoland\.)?GnoGenesisState\)' only hits ts.Value(envKeyGenesis), never AppState. The only non-test returns of ResponseInitChain with Error are gno.land app.go:401 (persistent_kvstore.go:119 and application.go:64 return none). |
| reach: partial boot | tm2/pkg/bft/node/node.go:1088 | on a retry the gnoland streaming provider serves a stale cached AppState for a corrected genesis file | gno.land/pkg/gnoland/genesis_state_ref.go:336 'finalDir := filepath.Join(cacheRoot, hash)': the cache dir is keyed on the source hash, so a corrected file maps to a new cache |
| rollout | tm2/pkg/bft/node/node.go:1106 | a zero-tx genesis stores an empty height-0 value that some backend's Has reports as absent, resetting a running chain on every boot | probe on all 6 backends after close and reopen, len(NewABCIResponsesFromNum(0).Bytes())=0: 'boltdb/goleveldb/lmdbdb/mdbxdb/pebbledb after reopen genesisApplied=true Get(nil?)=false'; memdb true |
| removed | gno.land/pkg/gnoland/node_inmemory.go:46 | another caller still builds a *GnoGenesisState or nil AppState, which now aborts boot instead of booting empty | grep -rn -E '(\.AppState *=\|AppState: )' non-test go: every assignment is a GnoGenesisState value or *GenesisStateRef (gnodev node.go:717 overrides NewDefaultGenesisConfig's with a value; gnogenesis DefaultGenesis uses DefaultGenState()) |
| removed | tm2/pkg/bft/consensus/replay.go:347 | another in-tree app returns ResponseInitChain.Error on an ordinary genesis and now refuses to boot | grep for ResponseInitChain with Error: only gno.land/pkg/gnoland/app.go:401 outside tests; baseapp.InitChain only forwards initChainer's response |
| rollout | tm2/pkg/sdk/baseapp.go:352 | SetInitialVersion from a rejected InitChain persists into the corrected retry | rootmulti/store.go:330, iavl/store.go:147, bptree/store.go:75: SetInitialVersion only sets in-memory fields; nothing is written to the DB |
| rollout | tm2/pkg/bft/node/node.go:1088 | a pre-#1941 data dir with committed blocks but no height-0 key enters the reset | git log -S: height-0 save added in 3aa0f0b7d (2024-07-23), AppState stripped from the persisted doc in 5d0a4d703 (2026-05-18); a doc persisted earlier carries a gno.land AppState, so the genDoc.AppState == nil branch is never entered for it |

## Completeness

- Sibling sweep of the added res.IsErr() guard by shape, every <X>Sync( call on the handshake and block-execution path followed by a Go-error check alone: replay.go:239 InfoSync, execution.go:170 and :452 CommitSync, :238 BeginBlockSync, :257 EndBlockSync. Only Commit has a live producer of ResponseBase.Error (baseapp.go:987), candidate 1003; BaseApp.Info, BeginBlock and EndBlock set no Error (awk over their bodies printed nothing).
- Apps returning ResponseInitChain with Error, which now abort boot: gno.land app.go:401 (loadAppState failure, the intended abort, including StrictReplay and an invalid AppState type), BaseApp.InitChain passing the initChainer's Error through (baseapp.go:373); non-test tm2 apps (kvstore, BaseApplication) return an empty response. Test apps setting Error: only the two new tests.
- Sibling sweep of the NewDefaultGenesisConfig pointer fix by shape, &GnoGenesisState{ anywhere: parity_test.go:53 and fork/inspect_test.go only, none passed to InitChain as AppState.
- Tests the diff adds, reddened by mutation: dropping the res.IsErr() block turns both TestHandshakeInitChainError subtests red; dropping the genesisApplied gate turns TestLoadStateFromDBOrGenesisDocProvider_RejectedGenesisRetry red and leaves TestE2ERejectThenCorrectedGenesisBoots green (candidate 2).
- Pruning of the height-0 marker the gate relies on: CalcABCIResponsesKey has one writer (store.go:203) and no delete anywhere in tm2 or gno.land, so the comment's 'never pruned' holds at the head.
- Invariant catalog: Error & panic handling is the class the diff touches (genesisApplied swallowing the Has error, candidates 1 and 1001); determinism, gas, realm state, caller, coin, storage deposit and VM classes do not apply to tm2 boot code; pickyApp.seen is appended and read on one goroutine through the local client, no shared state.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |
| e753cd391 | 1, 1001 | `genesisApplied` returns `(bool, error)` and `LoadStateFromDBOrGenesisDocProvider` returns a failed `Has`; `TestLoadStateFromDBOrGenesisDocProvider_ProbeErrorKeepsCommittedState` ports the repro and fails with the error swallowed again |
| abaa30adb | 2 | `TestE2ERejectThenCorrectedGenesisBoots` builds both boots through `NewNode` on one set of memdb databases, and fails with the reset gate at `node.go:1088` removed |

Rows 1002 and 2001 are left for the author: each is a decision, not a mechanical fix.

Row 1003 gets no issue. On master at 02606904e, `BaseApp.Commit` sets `ResponseCommit.Error` only when the base store is missing, and `LoadLatestVersion` and `LoadVersion` both run `initFromMainStore`, which returns `baseapp expects MultiStore with 'base' Store` first (`tm2/pkg/sdk/baseapp.go:178-182`). An app that loads never reaches that branch, so dropping the field costs nothing a node can hit. The `"err", res` log key at `execution.go:511` stays a one-line nit for whoever next edits that function.

## Retro

- What failed: the solo judge spent its 30 calls before its last `round check`, whose output it sent to `/dev/null`, so the parent re-ran it and cleared one bare `#<number>` row. No agent died, no stage moved model (`review-retro.py --model claude-opus-5-5` exit 0), 18 minutes against the projected 7.
- What worked: both finders reached the `Has` error reset independently from different angles (lines and rollout), and every row the judge kept held, 6 of 6 CONFIRMED.
- Hit rate per tier: warm 4 of 4 rows over 5 warm files, cold 1 of 1 over 6, no hot file; row 1003 sits outside the diff and has no tier. One round gives no ordering between tiers.
- Upgrade, applied: the Own PR solo judge's prompt in `scripts/workflows/review-pipeline.js` now holds back 6 calls for `round assemble`, the overview and the two gates, as the reader-facing branch already holds back 10. Estimate: turns per judge unchanged at its 30-call cap, about 6 fewer judging calls on a round that exhausts it and none on one that does not, output, cache read and cache write unchanged within a percent, finding rate unchanged; one parent turn saved per exhausted round. Estimate until the outcome table measures it.
- Metrics: 3 agents, 95 turns, 143k output, 12.0M cache read, 0.43M cache write, about $8.68.
