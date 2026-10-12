# Claims: gnolang/gno#5995 round 5, 1a1a31310, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 5 | CONFIRMED | Suggestion | tm2/pkg/bft/consensus/replay.go:239 | an app whose Info returns Error, run through NewHandshaker(...).Handshake | go test -overlay with tests/parent-5-info-error-ignored_test.go ./tm2/pkg/bft/consensus -run TestZZInfoErrorIgnored -v: 'handshake err=<nil>', --- PASS | tests/parent-5-info-error-ignored_test.go | warm |
| 1 | REFUTED | Warning | tm2/pkg/bft/node/node.go:1106 | go test -run TestJudgeRejectedGenesisRestart -v ./gno.land/pkg/gnoland with tests/judge-r1-rejected-genesis-restart_test.go copied in, at head and at the merge base | replay.go:346 if res.IsErr() { return nil, fmt.Errorf("InitChain rejected the genesis: %w", res.Error) } | tests/judge-r1-rejected-genesis-restart_test.go | hot |
| 2 | REFUTED | Warning | tm2/pkg/bft/node/node.go:1124 | grep -rn 'CalcABCIResponsesKey' tm2 gno.land contribs outside _test.go for a Delete | tm2/pkg/bft/state/store.go:203 db.Set(CalcABCIResponsesKey(height), abciResponses.Bytes()) |  | hot |
| 3 | REFUTED | Nit | contribs/gnogenesis/internal/fork/test.go:266 | awk over gno.land/pkg/gnoland/app.go deliverGenesisTx for every return before cfg.GenesisTxResultHandler, and both tx loops | gno.land/pkg/gnoland/app.go:824 if metadata != nil && metadata.Failed { |  | warm |
| 4 | REFUTED | Nit | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:81 | git merge-base --is-ancestor 3aa0f0b7d 5d0a4d703; git show of each for SaveABCIResponses(…, 0, …) and stripped.AppState = nil | 5d0a4d703 node.go: stripped.AppState = nil |  | warm |

Hit rate per tier, from the rows above: hot 0/2 confirmed over 1 files, warm 1/3 confirmed over 5 files, cold 0/0 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach | tm2/pkg/bft/node/node.go:1124 | a zero-tx genesis saves a nil height-0 value, which a Get-based Has could read as absent and reset a committed chain's state record | go test ./tm2/pkg/bft/node -run TestSoloEmptyGenesisMarker: 'zero-tx height-0 value: len=0 nil=true', then 'genesisApplied=true err=<nil>' on memdb, goleveldb, pebbledb, boltdb, lmdbdb and mdbxdb |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:74 | the operator-retry consequence is pinned only against a mock app; gno.land's real InitChainer could leave app-side writes that break a retry on the same DB | tests/solo-finder-gnoland-retry_test.go, one memdb shared by a refused and a corrected NewInMemoryNode: '--- PASS' in all three shapes, first block at 100 (app side corrected), 1 (doc 100 to 1), 100 (doc 1 to 100) |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:79 | 'Only RPC reads those heights': a non-RPC reader of stale per-height validator records below InitialHeight | grep LoadValidators callers: state/validation.go:175 sits inside a /* */ block (VerifyEvidence commented out); state/execution.go:281 reads block.Height-1 only when block.Height > state.InitialHeight |
| removed | gno.land/pkg/gnoland/node_inmemory.go:46 | a sibling constructor or consumer still building or asserting *GnoGenesisState after the default moved to a value | grep -rnE 'AppState(\s*:\|\s*=)\s*&' and 'AppState.(*' over the tree: no hit; the one non-test caller, contribs/gnodev/pkg/dev/node.go:717, overwrites AppState with a value |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:31 | 'every app's response passes' ReplayBlocks: another InitChain caller could still drop the response Error | grep '.InitChainSync(' non-test: tm2/pkg/bft/consensus/replay.go:339 is the only caller; grep 'ResponseInitChain{' non-test: gno.land/pkg/gnoland/app.go:401 is the only producer of an Error response |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:82 | history claim that the height-0 save (gnolang/gno#1941) predates AppState stripping (gnolang/gno#5684) | git log -S 'SaveABCIResponses(h.stateDB, 0' -- replay.go: 3aa0f0b7d (gnolang/gno#1941); git log -S 'stripped.AppState = nil' -- node.go: 5d0a4d703 (gnolang/gno#5684) |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:45 | execTest receives the refusal from NewInMemoryNode rather than from Start | tm2/pkg/bft/node/node.go:452 runs doHandshake inside NewNode, which NewInMemoryNode returns; the retry run logged 'error during handshake: error on replay: InitChain rejected the genesis: InitialHeight mismatch' |
| removed | gno.land/pkg/gnoland/app.go:497 | the deleted note that AppState may be nil in minimal setups: a nil AppState now aborts boot with 'invalid AppState of type <nil>' | NewInMemoryNode callers (integration/node_testing.go, integration/process.go, gnodev dev/node.go:717, gnogenesis fork/test.go) all pass a GnoGenesisState value; a nil AppState left in a test setup fails that test in CI |
| two-versions | gno.land/pkg/gnoland/app.go:563 | StrictReplay's refusal was discarded before and is now enforced, changing boot for any deployment that sets it | grep StrictReplay non-test: only app.go:321-329 (declaration) and 563/647 (reads); nothing outside tests sets it |
| removed | tm2/adr/pr5949_remove_secp256k1_validators.md:71 | another ADR still says ResponseInitChain.Error is silently discarded by tm2 | gno calibration: the ADR is the governing document the no-critique rule covers; it records PR 5949's own decision |

## Completeness

- ADR Context, Decision and Consequences read line by line against the code: every runtime statement maps to one of candidates 1-4 or to a run listed in their verdicts; none failed.
- Catalog: skills/invariant-catalog.md holds realm audit patterns and caller-identity predicates; the diff touches no .gno, GnoVM or stdlib code, so no class applies.
- Sibling sweep by shape, an ABCI Sync call in the handshake whose response Error is unread: InitChainSync is fixed, InfoSync at replay.go:239 is candidate 5; the only non-test InitChainSync caller is replay.go:339.
- Changed tests: TestHandshakeInitChainError, TestE2ERejectThenCorrectedGenesisBoots, the five LoadStateFromDBOrGenesisDocProvider tests and TestExecTest_InitialHeightMismatch pass at head.
- The ADR's runtime claims, read against the head by the judge: a refused genesis fails boot and a corrected one boots on the same data dir with the real gno.land app, both at `InitialHeight` 100 and 999; nothing deletes an ABCI responses key, `state/store.go:203` being the only writer; `deliverGenesisTx` skips the handler only for `metadata.Failed`, which the expected count excludes; gnolang/gno#1941 (3aa0f0b7d) is an ancestor of gnolang/gno#5684 (5d0a4d703). All four held.
- Written by the parent from `candidates/reflector.json` and the judge's returned summary: the section held its placeholder when the parent re-assembled the round.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |

Nothing applied. No Critical or Warning; row 5 is a latent Suggestion that predates the branch, settled CONFIRMED by the parent with `tests/parent-5-info-error-ignored_test.go`, and gets no issue: no in-tree `Info` implementation sets `ResponseBase.Error`, the same reason as round 3 row 1003.

## Retro

- What failed: the judge returned with one row PLAUSIBLE and the Completeness section unfilled; the parent ran the missing check, a handshake against an app whose `Info` returns an error, `handshake err=<nil>`, and wrote the section. 18 minutes against the projected 9.
- What worked: every runtime claim of the new ADR was run and held, four rows REFUTED with their runs.
- Hit rate per tier: warm 1 of 3, row 5, a latent gap in a warm file outside the changed lines; hot 0 of 2; cold none.
- Upgrade: none applied. Both misses sit in the solo judge's own-PR prompt, which round 3's retro already changed to hold back calls for the closing steps; this round ran on that prompt and still missed them, so the next change waits for a round whose transcript shows where the calls went.
- Metrics: 2 agents, 52 turns, 100k output, 6.8M cache read, 0.37M cache write, about $6.35.
