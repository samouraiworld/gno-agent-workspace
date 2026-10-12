# Claims: gnolang/gno#5995 round 4, abaa30adb, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:1129 | test in tm2/pkg/bft/node: LoadStateFromDBOrGenesisDocProvider(db, probeGenesisDocProvider(pk, 1)), then the same db with probeGenesisDocProvider(pk, 100); compare sm.LoadValidators/LoadConsensusParams at 1, 2, 100, 101 against a db that booted 100 directly | h=1 reset: `vals=1 err=<nil> params_err=<nil>`, fresh: `Could not find validator set for height #1`, `Could not find consensus params for height #1`; h=2 reset: `vals=1 err=<nil>`, fresh: `Could not find validator set for height #2`; h=100 and h=101 identical on both. --- PASS | tests/judge-1-stale-valset_test.go | hot |
| 2 | CONFIRMED | Nit | tm2/pkg/bft/node/node.go:1111 | at merge base cfcbaf484: handshake against initChainApp returning ResponseInitChain.Error, then probe abciResponsesKey:0; at head: the same handshake, and TestLoadStateFromDBOrGenesisDocProvider_AppliedGenesisWins for the persisted doc winning once the key exists | base: handshake err=<nil>, height-0 marker present=true; head: handshake err=error on replay: InitChain rejected the genesis: InitialHeight mismatch, height-0 marker present=false; AppliedGenesisWins --- PASS | tests/judge-2-base-saves-rejected_test.go | hot |

Hit rate per tier, from the rows above: hot 2/2 confirmed over 1 files, warm 0/0 confirmed over 4 files, cold 0/0 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| removed | gno.land/pkg/gnoland/app_test.go:411 | the TestInitChainer_PanicsOnValoperCoverageFailure doc comment still says consensus/replay.go:339-342 silently discards ResponseInitChain.Error, which the diff makes false | grep -rniE 'silently discard\|replay\.go:3[34][0-9]' gives 'app_test.go:411: // panic, not via the ResponseInitChain.Error field that tm2's' and ':412: // consensus/replay.go:339-342 silently discards.'; the only fix is the comment's wording, which this round does not return |
| removed | tm2/adr/pr5949_remove_secp256k1_validators.md:71 | ADR text (and gno.land/adr/pr5511_chain_upgrade_genesis_replay.md:240) states ResponseInitChain.Error is silently discarded | same grep: 'pr5949_remove_secp256k1_validators.md:71: as with the existing valoper-coverage assertion, `ResponseInitChain.Error` is silently'; an ADR is the governing document the no-critique rule covers |
| rollout | gno.land/pkg/gnoland/app.go:470 | a rejected InitChain persists stdlib writes into the app DB that the tm2-side reset does not undo, contradicting 'nothing is committed' | sed -n 466,472p app.go: 'ms := ctx.MultiStore()' / 'msCache := ms.MultiCacheWrap()'; the MultiWrite at :481 lands in the deliverState cache, which only Commit writes to the DB |
| removed | gno.land/pkg/gnoland/app.go:563 | StrictReplay now really refuses boot; an in-tree deployment setting it with failing genesis txs would stop booting | grep -rn StrictReplay non-test .go: only app.go:321,329,422,563,647; nothing sets it outside tests |
| removed | gno.land/pkg/gnoland/node_inmemory.go:46 | a caller of NewDefaultGenesisConfig type-asserts AppState as *GnoGenesisState and breaks on the value | grep NewDefaultGenesisConfig( gives contribs/gnodev/pkg/dev/node.go:716 (overrides with 'genesis.AppState = appstate', a value) and node_initial_height_test.go:87; the only '.(*gnoland.GnoGenesisState)' asserts are testscript env values (testscript_gnoland.go:287), which assign 'cfg.Genesis.AppState = genesis' as a value |
| lines | tm2/pkg/bft/node/node_genesis_restart_test.go:81 | the e2e test never stops n, so a listener bound in NewNode would make later package tests hit address-in-use | grep 'Listen(' node.go: 'node.go:665: if err := n.transport.Listen(*addr)', enclosing function 'node.go:624: func (n *Node) OnStart() error'; NewNode binds nothing |
| lines | tm2/pkg/bft/node/node.go:1129 | resetToGenesisDoc writes the doc and the state with two separate syncs; a crash between leaves the new doc beside the old state record | node.go:1092-1093 'if !applied { return resetToGenesisDoc(stateDB, freshDoc) }' re-runs on every boot until the height-0 key exists, so the next boot rewrites both from the provider and converges |

## Completeness

- InitChain consumers: `replay.go:339` is the only non-test `InitChainSync` caller and `InitChainAsync` has none, so the `res.IsErr()` guard has no unguarded sibling.
- Pointer `AppState`: no other site builds `AppState` as `&GnoGenesisState`, so the `node_inmemory.go` fix has no sibling.
- Nil `AppState`: `loadAppState` rejects nil in its default case, and that error now stops boot. The nil sites found (`app_test.go:364`, the gnogenesis txs and balances tests, `baseapp_test.go:371`) call `InitChainer` directly or edit a file, and none boots a node.
- Row 1's reach: the only consensus reader of old heights is `VerifyEvidence`, and it is commented out at `validation.go:165`, so the stale records show up only through `/validators`, `/consensus_params` and `status.go:94`.
- Round 3 row 2001, not reopened: `saveGenesisDoc` started stripping `AppState` in 5d0a4d703 (2026-05-18), after the height-0 save landed in 3aa0f0b7d (2024-07-23). A data dir from before 3aa0f0b7d still has `AppState` in its persisted doc and skips the reset branch, unless its genesis had a null `app_state`.
- Catalog: the diff touches tm2 node boot and the gnoland `InitChainer`, with no VM, gas, realm or banker code. The one class it touches is error and panic handling, and both of its rows are covered above.

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |
| fd79951a0 | none | the simplify pass: `DefaultNewNodeWithGenesisProvider` split into an unexported `newNodeWithDBProvider`, which the restart test calls instead of copying the node wiring; a fresh-agent pass over it returned nothing |
| db62dfa0f | none | the `app_test.go:411` comment the finder settled, which still said the handshake discards `ResponseInitChain.Error` |

`round stop` over `4c61d0582..abaa30adb` exits 0, no Critical or Warning, so the loop closes here. Row 1 adds an exported deleter to `tm2/pkg/bft/state` and changes what RPC answers for heights the corrected genesis never ran, so it goes to the author as a decision. Row 2 is a note for the pull request body, with no code change.

## Retro

- What failed: nothing died and no stage moved model (`review-retro.py --model claude-opus-5-5` exit 0); 17 minutes against the projected 8.
- What worked: the judge's reserved calls held, with `round check` at 0 hits and the overview clean before it returned. Both rows held, 2 of 2 CONFIRMED.
- Hit rate per tier: hot 2 of 2, both on `node.go`, the one hot file; warm and cold 0 rows.
- Upgrade: none this round. The round 3 upgrade, a call reserve for the Own PR judge, did not apply here because round 4 ran on the runner as launched before the edit. Its estimate stands until a round runs on it.
- Metrics: 2 agents, 57 turns, 92k output, 6.7M cache read, 0.34M cache write, about $5.90.
