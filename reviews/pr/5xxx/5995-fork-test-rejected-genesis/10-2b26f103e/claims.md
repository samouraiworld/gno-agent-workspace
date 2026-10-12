# Claims: gnolang/gno#5995 round 10, 2b26f103e, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 2 finders and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | tm2/pkg/bft/node/node.go:1092 | grep -n genesisDocKey tm2/pkg/bft/node/*_test.go shows only key comparisons (node_test.go:674, :711) and no raw write of a doc carrying AppState. Mutate node.go:1092 to `} else {` and run `go test ./tm2/pkg/bft/node -run 'LoadStateFromDBOrGenesisDocProvider\|E2E\|GenesisReset\|SaveGenesisDoc'`: expect red, observe green. | mutated run: `ok github.com/gnolang/gno/tm2/pkg/bft/node 2.229s`; unmutated named run: 17 PASS lines |  | hot |  |
| 2 | CONFIRMED | Nit | tm2/pkg/bft/node/node.go:1101 | Seed a memdb with saveGenesisDoc of a doc whose ChainID is "" (the pre-PR first-boot write order), then call LoadStateFromDBOrGenesisDocProvider with probeGenesisDocProvider(pk, 1): expect the boot to succeed or name the invalid persisted doc, observe the 'genesis chain id mismatch ... ("") ... ("genesis-doc-test")' error | judge test log: `boot error: genesis chain id mismatch between persisted state ("") and source genesis ("genesis-doc-test"): the genesis file backing this data dir has changed`, PASS | tests/judge-2-empty-chain-id_test.go | hot |  |
| 3 | CONFIRMED | Suggestion | tm2/pkg/bft/node/node.go:1119 | Read node.go:1114-1120 and resetToGenesisDoc (1146-1157): no logger call and no return value distinguishes a reset from a normal load; NewNode (node.go:415) logs nothing about it either | node.go:415 `state, genDoc, err := LoadStateFromDBOrGenesisDocProvider(stateDB, genesisDocProvider)` followed by createAndStartProxyAppConns; no logger call between node.go:1072 and 1157 |  | hot |  |
| 1001 | REFUTED | Suggestion | contribs/gnodev/pkg/dev/node.go:716 | A scratch test in contribs/gnodev/pkg/dev calling newNodeConfig(tmcfg, "dev", "gno.land", gnoland.GnoGenesisState{InitialHeight: 100}) then gnoland.NewInMemoryNode: expect an 'InitialHeight mismatch' error at head 2b26f103e and a node with no genesis state at merge base cfcbaf484 | contribs/gnodev/pkg/dev/node.go:341-342 `genesis := gnoland.DefaultGenState()` / `genesis.Balances = n.config.BalancesList` |  |  |  |

Hit rate per tier, from the rows above: hot 3/3 confirmed over 4 files, warm 0/0 confirmed over 2 files, cold 0/0 confirmed over 6 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | contribs/gnogenesis/internal/fork/test.go:265 | tripwire comment says no current path skips a deliverable tx before the result handler | sed -n 725,860p gno.land/pkg/gnoland/app.go: the only return before `cfg.GenesisTxResultHandler(ctx, stdTx, res)` (app.go:859) is `if metadata != nil && metadata.Failed {` (app.go:824), which countDeliverableTxs excludes; the other returns at relative lines 42/63/121 are inside ctx-builder closures |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:90 | ADR attributes the height-0 save to gnolang/gno#1941 and the AppState strip to gnolang/gno#5684 | git log -S'SaveABCIResponses(h.stateDB, 0' -- replay.go -> 3aa0f0b7d feat(tm2): expose InitChain tx responses (gnolang/gno#1941); git log -S'stripped.AppState = nil' -- node.go -> 5d0a4d703 (gnolang/gno#5684) |
| claims | tm2/adr/pr5995_initchain_rejection_aborts_boot.md:84 | ADR says a nil AppState now fails the handshake with `invalid AppState of type <nil>` | app.go:496-497 loadAppState default branch `return nil, fmt.Errorf("invalid AppState of type %T", appState)`; %T of a nil interface prints <nil>, and replay.go now returns on res.IsErr() |
| claims | tm2/pkg/bft/node/node.go:1130 | genesisApplied godoc says the height-0 responses are never pruned and are written only after InitChain succeeds | grep -rn 'ABCIResponsesKey' (non-test): store.go:179 Get, store.go:203 Set, node.go:1134 Has, no Delete; SaveABCIResponses callers: execution.go:100 (block heights) and replay.go:357 (height 0, after the res.IsErr and no-validators returns) |
| two versions | gno.land/pkg/gnoland/node_inmemory.go:45 | a sibling building AppState as *GnoGenesisState, or a reader asserting the pointer, breaks with the value change | grep -rn '&gnoland\.GnoGenesisState\\|&GnoGenesisState\b\\|AppState *= *&\\|AppState: *&' (non-test): no hits; the `.(*gnoland.GnoGenesisState)` asserts at integration/testscript_gnoland.go:287,509,554,648 read ts.Value(envKeyGenesis), not GenesisDoc.AppState; NewDefaultGenesisConfig callers are gnodev node.go:716 (overwrites AppState with a value) and one test |
| line by line | gno.land/pkg/gnoland/node_initial_height_test.go:143 | a refused InitChain could leave app writes in the shared DB that the app-hash comparison misses (dbadapter base store not hashed) | app.go:470-481 loadStdlibs wraps ctx.MultiStore() (deliverState cache) and MultiWrite()s into it, not the DB; baseapp InitChain's SetInitialVersion is in-memory only (rootmulti/store.go:330-340, iavl/store.go:147-154, bptree/store.go:75-81) |
| line by line | tm2/pkg/bft/consensus/replay.go:346 | %w over res.Error, an abci.Error, may not be an error value | tm2/pkg/bft/abci/types/types.go:207-210 `type Error interface { AssertABCIError(); Error() string }`, so it satisfies error |
| removed | tm2/pkg/bft/node/node.go:1079 | The not-found branch lets the file's doc through with no chain-identity check when a state record already exists. This is the 'neither' cell beside the read-error hardening. | Unreachable without an outside writer. The first boot writes the doc with SetSync (node.go:1089) before LoadStateFromDBOrGenesisDoc saves the first state record (node.go:1123). grep -rn 'genesisDoc' over *.go hits only node.go:1057, so no tool writes or deletes the key. contribs/gnobr/main.go:100-136 patches only the state record. |
| reach | tm2/pkg/bft/node/node.go:1134 | genesisApplied probes a key that SaveABCIResponses writes with db.Set rather than SetSync, so a crash could lose the marker after InitChain succeeded. | store.go:203 is 'db.Set(CalcABCIResponsesKey(height), ...)'. On a fresh genesis, replay.go then runs sm.SaveState, whose doc says 'This flushes the writes (e.g. calls SetSync)'. That sync flushes the earlier Set on the log-ordered backends. On a non-fresh replay the key was already written by the first boot. |
| reach | tm2/pkg/bft/consensus/replay.go:346 | Honouring res.IsErr() could stop production nodes from booting when StrictReplay refuses a genesis whose txs were all delivered. | grep -rn 'StrictReplay' outside _test.go hits only gno.land/pkg/gnoland/app.go (field and reads). No production config sets it, and gnoland start maps --skip-failing-genesis-txs to SkipFailingTxs (start.go:275), not StrictReplay. |
| removed | gno.land/pkg/gnoland/node_inmemory.go:45 | Other in-tree sites may build a *GnoGenesisState AppState, which loadAppState refuses and the handshake now stops on. | grep -rnE 'AppState:\s*&\|AppState\s*=\s*&\|\.AppState = &' over *.go returns no hit. gnodev overrides the default with a value (node.go:717 'genesis.AppState = appstate'). The integration testscript converts its *GnoGenesisState to a value before assigning it (testscript_gnoland.go:288-296). |
| removed | tm2/pkg/bft/consensus/replay.go:339 | Another InitChain caller could still read only the Go error and miss the response Error. | grep -rn '\.InitChainSync(' outside _test.go hits only replay.go:339 and the appconn and local_client pass-throughs. kvstore/helpers.go:34 is a test-data initialiser. NewHandshaker has one caller, node.go:292. |
| removed | gno.land/pkg/gnoland/app.go:398 | Some doc or comment outside the diff may still claim that tm2 discards ResponseInitChain.Error. | grep -rniE 'ResponseInitChain\.Error\|InitChain.*(silently\|discard\|ignor)' over *.go, *.md and *.txtar. It hits only the rewritten comments, the pr5995 ADR, gno.land/adr/pr5511 line 240 (which describes the BaseApp early return, still accurate) and tm2/adr/pr5949 (a historical record, not to be reopened). |
| reach | tm2/pkg/sdk/baseapp.go:351 | A refused InitChain could leave persistent app writes, such as SetInitialVersion and storeConsensusParams, that a corrected retry inherits. | rootmulti SetInitialVersion only sets ms.initialVersion and the per-store in-memory fields (store.go:330-339). storeConsensusParams writes to the mounted main store, which reaches disk only on Commit. loadStdlibs MultiWrites into the deliverState cache (app.go:481), not the root store. |

## Completeness

- Catalog: `skills/invariant-catalog.md` holds realm audit patterns and caller identity predicates only, and none of its classes reaches tm2 node boot or the gnogenesis fork test.
- Pointer `AppState`, the shape `NewDefaultGenesisConfig` dropped: a grep at the head for `AppState` assigned `&GnoGenesisState` in `.go` files returns nothing.
- The `InitChain` response guard: `replay.go:339` is the only non-test `InitChainSync` caller at the head, so no sibling handshake reads the response without the `res.IsErr()` check.
- The `saveGenesisDoc` write error: `state/store.go:134` still discards its `SetSync` error, the settled store-wide convention.
- c121978e7 now persists the doc `ValidateAndComplete` filled in, a completed `GenesisTime` included, where the merge base persisted the raw doc. Read as an improvement, no candidate.
- gnodev's `--genesis-file` path keeps only the file's balances and txs, so the head's `InitialHeight` check never reaches it (candidate 1001, refuted).

## Applied

| Commit | Rows | What changed |
| --- | --- | --- |
| 5d27c66b4 | 1 | `TestLoadStateFromDBOrGenesisDocProvider_LegacyDocKeepsCommittedState` writes a doc carrying `AppState` under `genesisDocKey`, a state record at height 500 and no height-0 key; with the gate widened to `} else {` it fails, 500 against 0 |

Rows 2 and 3 are held for the author. Row 2 predates the branch, base and head alike, and a dir that never applied a genesis loses nothing to a wipe. Row 3 asks for a log line, which needs a logger or a returned flag on `LoadStateFromDBOrGenesisDocProvider`, a signature change.

## Retro

- What failed: the reach finder's gnodev candidate, 1001, refuted: gnodev keeps only the file's balances and txs. The judge's Completeness reads the catalog as realm patterns only, though it carries the Error and panic handling and Twin input paths classes this diff touches; the finders walked those, per their dropped rows.
- What worked: the output projection, 144k against 148k spent, after round 9's repricing; 15 minutes against the projected 22. The two-versions read found the one gate no test held, and its ported test fails under the mutation.
- Hit rate per tier: hot 3 of 3 confirmed; the refuted row sits outside the risk table.
- Upgrade, applied: the invariant catalog gains a "Data an older binary wrote" class, since row 1's gate exists only for such data and every fixture in the suite was written by the current code. Estimate: one more class each finder walks, a line read per round.
- Metrics: 3 agents, 89 turns, 148k output, 10.7M cache read, 0.46M cache write, about $8.77.
