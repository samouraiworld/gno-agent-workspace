# Claims: gnolang/gno#5995 round 6, 4a3ca5bb6, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | REFUTED | Missing test | tm2/pkg/bft/node/node_test.go:611 | bash tests/judge-1-seed-helper-mutations.sh: delete the helper's SaveABCIResponses line, then its LastBlockHeight line, expect FAIL each time | node_test.go:611 sm.SaveABCIResponses(db, 0, sm.NewABCIResponsesFromNum(0)): deleting it fails CommittedStatePreserved | tests/judge-1-seed-helper-mutations.sh | cold |  |
| 2 | REFUTED | Nit | tm2/pkg/bft/consensus/replay_test.go:1206 | bash tests/judge-2-replay-save-order.sh: move the save above the res.IsErr() return, expect both subtests FAIL | replay_test.go:1246 _, loadErr := sm.LoadABCIResponses(stateDB, 0) | tests/judge-2-replay-save-order.sh | cold |  |
| 3 | REFUTED | Nit | tm2/pkg/bft/node/node_genesis_restart_test.go:62 | sed -n 60,68p tm2/pkg/bft/node/node_genesis_restart_test.go: what the provider calls before the branch, and whether that call can return an error | node_genesis_restart_test.go:61 doc, derr := types.GenesisDocFromFile(genesisFile) |  | cold |  |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 2 files, warm 0/0 confirmed over 4 files, cold 0/3 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| removed | tm2/pkg/bft/node/node_test.go:606 | extracting seedCommittedChain out of CommittedStatePreserved and ProbeErrorKeepsCommittedState could leave either test no longer pinning the reset gate or the probe-error path | scratch tree at 4a3ca5bb6, go test ./tm2/pkg/bft/node -run 'TestLoadStateFromDBOrGenesisDocProvider\|TestGenesisReset\|TestE2EReject' -count=1: baseline ok; probe error mutated to 'return false, nil': '--- FAIL: TestLoadStateFromDBOrGenesisDocProvider_ProbeErrorKeepsCommittedState'; 'if !applied {' mutated to 'if !applied \|\| true {': '--- FAIL: TestLoadStateFromDBOrGenesisDocProvider_AppliedGenesisWins', '--- FAIL: TestLoadStateFromDBOrGenesisDocProvider_CommittedStatePreserved' |
| removed | tm2/pkg/bft/node/node_genesis_restart_test.go:75 | with the boot-1 t.Logf gone, require.Error could pass on an unrelated boot failure and stop pinning the handshake fix | scratch tree, replay.go:346 mutated to 'if false && res.IsErr() {': 'node_genesis_restart_test.go:75: An error is expected but got nil', '--- FAIL: TestE2ERejectThenCorrectedGenesisBoots'; TestHandshakeInitChainError fails both subtests under the same mutation |
| lines | contribs/gnogenesis/internal/fork/test_test.go:240 | folding require.Error into require.ErrorContains could let a nil error pass TestExecTest_InitialHeightMismatch | testify v1.11.1 assert/assertions.go:1691 'if !Error(t, theError, msgAndArgs...) { return false }': ErrorContains fails on a nil error first |
| catalog | tm2/pkg/bft/node/node_test.go:680 | 'd, _ := probeGenesisDocProvider(pk, 100)()' discards an error (catalog: no swallowed errors) | sed -n 543,553p tm2/pkg/bft/node/node_test.go: the provider's only return is '&types.GenesisDoc{...}, nil' (line 553), so the discarded error is always nil |
| removed | gno.land/pkg/gnoland/node_inmemory.go:43 | the trimmed comment 'loadAppState rejects a *GnoGenesisState as an invalid AppState' could state something false | gno.land/pkg/gnoland/app.go:491-497: switch on GnoGenesisState and *GenesisStateRef only, 'default: return nil, fmt.Errorf("invalid AppState of type %T", appState)' |
| removed | tm2/pkg/bft/node/node.go:1118 | the trimmed genesisApplied comment claims the height-0 responses are saved only on InitChain success and never pruned; another writer or a delete would make it false | grep -rn 'SaveABCIResponses\|CalcABCIResponsesKey' --include='*.go' tm2 gno.land contribs, non-test: writers are replay.go:353 (height 0, after the res.IsErr return at 346) and execution.go:100 (block.Height); no Delete of the key |
| catalog | tm2/pkg/bft/node/node_test.go:606 | parallel tests added in node and consensus packages could race on shared state (catalog: global mutable state and concurrency) | go test -race ./tm2/pkg/bft/node -run 'TestLoadStateFromDBOrGenesisDocProvider\|TestGenesisReset\|TestE2EReject' -count=1: 'ok 1.049s'; go test -race ./tm2/pkg/bft/consensus -run TestHandshakeInitChainError -count=1: 'ok 1.044s' |

## Completeness

- Commit 4a3ca5bb6 read hunk by hunk for behaviour or test-strength change: no production line changes; node.go and node_inmemory.go lose comment lines only, and every remaining comment sentence was checked against the code (candidates 2 and the finder's drops at node.go:1118 and node_inmemory.go:43).
- Test strength: the shared seed helper is candidate 1, the trimmed InitChain-error header candidate 2; the folded require.Error is settled by a run, assert.ErrorContains(fakeT, nil, ...) returning ok=false with the fake failed (tests/judge-errorcontains-nil_test.go), and TestExecTest_InitialHeightMismatch passes at head in 1.134s.
- Sibling sweep by shape of the dropped dead branch, a test GenesisDocProvider checking an error its callee never returns: the only other 'if derr != nil' in tm2/pkg/bft/node tests is candidate 3.
- Catalog: skills/invariant-catalog.md classes are gno VM, realm and stdlib invariants; 4a3ca5bb6 touches no .gno, GnoVM or stdlib code. The Error and panic handling class reaches the discarded error at node_test.go:680, settled by the finder: the provider's only return carries a nil error.
- All changed tests pass at head: the node package's LoadStateFromDBOrGenesisDocProvider, GenesisReset and E2EReject tests, and TestHandshakeInitChainError.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |

Nothing to apply: no row held. `round stop` reads no Critical or Warning, so the loop stays closed at 4a3ca5bb6.

## Retro

- What failed: nothing; no agent died, no stage moved model (`review-retro.py --model claude-opus-5-5` exit 0), 10 minutes against the projected 22.
- What worked: the judge filled Completeness and returned no PLAUSIBLE row, on the prompt round 3 changed to hold back calls for its closing steps; the three reflector candidates each went to a run, the two test-strength ones to a mutation.
- Hit rate per tier: no row confirmed in any tier, over a commit that changes no executable line.
- Upgrade: none applied; a refactor-only head took a trivial round at $3.87, which is the shape the loop should keep for it.
- Metrics: 2 agents, 41 turns, 50k output, 4.4M cache read, 0.25M cache write, about $3.87.
