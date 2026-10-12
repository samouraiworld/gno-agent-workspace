# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 0 refuted kept out

## Body

- Suggestion: tm2/pkg/bft/state/execution.go:170 [gh](https://github.com/gnolang/gno/blob/4c61d0582649ececdaa0ffa5c4ebdcabdeacc702/tm2/pkg/bft/state/execution.go#L170), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: removed
  TL;DR: CommitSync callers check only the Go error and drop ResponseCommit.Error, the one sibling of the InitChain guard with a live producer
  Check: sed -n 168,180p and 450,458p tm2/pkg/bft/state/execution.go; sed -n 985,989p tm2/pkg/sdk/baseapp.go
  Details: execution.go:170-177 and :452-457 return on err alone; the comments say ResponseCommit has no error code, but BaseApp.Commit sets res.Error when the multistore has no 'base' store (baseapp.go:987). Latent: gno.land mounts baseKey (app.go:107). Predates the diff; in scope as a missed member of the class the diff's res.IsErr() guard sweeps. Also :455 logs `"err", res` where err was meant.
  Evidence: sed execution.go:168-180 and 450-458: no res.IsErr() after CommitSync; baseapp.go:987 `res.Error = ABCIError(errors.New("baseapp expects MultiStore with 'base' Store"))`; app.go:107 `baseApp.MountStoreWithDB(baseKey, ...)`

## tm2/pkg/bft/node/node.go:1106 [gh](https://github.com/gnolang/gno/blob/4c61d0582649ececdaa0ffa5c4ebdcabdeacc702/tm2/pkg/bft/node/node.go#L1106) · Warning
State: CONFIRMED, band: Warning, angle: catalog: error & panic handling / lines
TL;DR: a failing Has on abciResponsesKey:0 reads as 'genesis not applied' and persists the genesis state record over a committed chain
Check: go test ./tm2/pkg/bft/node -run TestSoloFinder1_HasError -v -count=1
Details: genesisApplied is `ok, err := stateDB.Has(sm.CalcABCIResponsesKey(0)); return err == nil && ok` (node.go:1123-1124), so an error is the same answer as an absent key. Every restart reaches the gate, since saveGenesisDoc drops AppState and the re-invoke branch always runs. resetToGenesisDoc then calls sm.SaveState with MakeGenesisState(freshDoc): LastBlockHeight 0, empty AppHash, no error returned. The next boot, the read error gone, loads that record and ReplayBlocks panics on StoreBlockHeight > StateBlockHeight + 1 (replay.go:395 at head), so the node needs a state repair or a resync. The trigger is a read error on that one key while the genesisDoc Get just succeeded: rare, but the outcome is a persisted write that cannot be undone. At the merge base the branch reads no extra key, and loadState panics on a Get error rather than writing. Return the Has error out of LoadStateFromDBOrGenesisDocProvider (genesisApplied as (bool, error)), which is mechanical. Same defect and same edit as candidate 1001; the round-2 decision to key the gate on the marker is untouched by it.
Evidence: finder-1 artifact rerun at head: 'returned err=<nil> LastBlockHeight=0' / 'persisted after restart: LastBlockHeight=0 AppHash=""'. It does not build at the merge base (uses head-only probeGenesisDocProvider), so the base side is 1001's artifact: 'persisted LastBlockHeight=500 AppHash="committed-app-hash"', PASS.
Artifact: tests/solo-finder-1-has-error-resets-state.go
State: CONFIRMED, band: Warning, angle: rollout
TL;DR: one read error at boot rebuilds and persists genesis state over a running chain's state record
Check: cp tests/solo-finder-2-has-error-resets-state.go tm2/pkg/bft/node/zz_has_error_test.go && go test ./tm2/pkg/bft/node -run TestZZHasErrorResetsCommittedChain -v -count=1
Details: Same mechanism as candidate 1, reproduced independently: the wrapper DB fails Has once, LoadStateFromDBOrGenesisDocProvider returns nil error, and sm.LoadState afterwards reads LastBlockHeight 0 and an empty AppHash; the same file at the merge base keeps 500 and the committed hash. One edit closes both: Return the Has error out of LoadStateFromDBOrGenesisDocProvider (genesisApplied as (bool, error)), which is mechanical.
Evidence: head 4c61d0582, rerun from the file: 'boot err=<nil>; persisted LastBlockHeight=0 AppHash=""', expected 500 actual 0, --- FAIL. merge base cfcbaf484 via go test -overlay: 'boot err=<nil>; persisted LastBlockHeight=500 AppHash="committed-app-hash"', --- PASS.
Artifact: tests/solo-finder-2-has-error-resets-state.go

## tm2/pkg/bft/node/node_genesis_restart_test.go:38 [gh](https://github.com/gnolang/gno/blob/4c61d0582649ececdaa0ffa5c4ebdcabdeacc702/tm2/pkg/bft/node/node_genesis_restart_test.go#L38) · Missing test
State: CONFIRMED, band: Missing test, angle: reach / tests
TL;DR: TestE2ERejectThenCorrectedGenesisBoots stays green with the reset gate deleted
Check: remove node.go:1088-1090 and run go test ./tm2/pkg/bft/node -run TestE2ERejectThenCorrectedGenesisBoots
Details: cfg.ResetTestRoot sets DBBackend memdb (config.go:334), DefaultDBProvider calls dbm.NewDB per NewNode (node.go:82-85), and memdb's creator returns NewMemDB() each call (mem_db.go:16), so the second build starts on an empty DB and takes the first-boot branch. With node.go:1088-1090 deleted the e2e test passes while TestLoadStateFromDBOrGenesisDocProvider_RejectedGenesisRetry fails, so only the unit test pins the reset and no test runs the handshake after it. Fix shape for the author: a DBProvider returning one shared memdb per ctx.ID across both builds, or a disk backend.
Evidence: mutation: node.go:1088-1090 removed; 'boot1 err=error during handshake: error on replay: InitChain rejected the genesis: InitialHeight mismatch: refusing to boot', '--- PASS: TestE2ERejectThenCorrectedGenesisBoots', '--- FAIL: TestLoadStateFromDBOrGenesisDocProvider_RejectedGenesisRetry'; tree restored.

## tm2/pkg/bft/node/node.go:1088 [gh](https://github.com/gnolang/gno/blob/4c61d0582649ececdaa0ffa5c4ebdcabdeacc702/tm2/pkg/bft/node/node.go#L1088) · Suggestion
State: CONFIRMED, band: Suggestion, angle: rollout
TL;DR: a data dir where the current release ran a rejected InitChain already carries the height-0 marker, so the corrected genesis is never picked up there
Check: read: base tm2/pkg/bft/consensus/replay.go:339-347; head node.go:1088-1091
Details: Merge base replay.go:339-347 saves abciResponsesKey:0 straight after InitChainSync with no res.Error check, so a release binary that saw a rejection marked the genesis applied. On this head, genesisApplied is then true at node.go:1088 and the persisted doc stays, only AppState refreshed, so a corrected InitialHeight or validator set in genesis.json is ignored and InitChain refuses again. The merge base ignores the corrected doc too (it never refreshes doc fields), so this is not a regression; it bounds the retry path to data dirs first booted by this binary. Settled by reading those lines; the two-binary run was not built. Whether upgrade notes tell an operator to wipe such a dir is a decision for the author.
Evidence: base tm2/pkg/bft/consensus/replay.go:339-347: `res, err := proxyApp.Consensus().InitChainSync(req); if err != nil { return nil, err }` then `sm.SaveABCIResponses(h.stateDB, 0, abciResponse)`; head node.go:1088 `if !genesisApplied(stateDB) {`
State: CONFIRMED, band: Suggestion, angle: rollout
TL;DR: a committed chain with no height-0 ABCI responses key, genesis applied before 3aa0f0b7d, is reset to genesis state on its first boot of this binary
Check: cp tests/judge-2001-no-height0-marker-resets-state.go tm2/pkg/bft/node/zz_no_marker_test.go && go test ./tm2/pkg/bft/node -run TestJudgeNoHeight0MarkerKeepsState -v -count=1
Details: The height-0 save in ReplayBlocks landed in 3aa0f0b7d (gnolang/gno#1941, 2024-07-23). A data dir whose InitChain ran before it has no marker, so the gate at node.go:1088 resets its state record exactly as in candidate 1, with no read error needed. Latent: whether any live data dir predates that commit was not measured here. A gate that also requires no committed block (block store height 0) closes this and candidates 1 and 1001 together while keeping the marker semantics round 2 chose; that is a decision for the author.
Evidence: judge artifact at head: 'boot err=<nil>; persisted LastBlockHeight=0 AppHash=""', expected 500 actual 0, FAIL; merge base via -overlay: 'persisted LastBlockHeight=500 AppHash="committed-app-hash"', PASS; git log -S names 3aa0f0b7d 2024-07-23
Artifact: tests/judge-2001-no-height0-marker-resets-state.go
