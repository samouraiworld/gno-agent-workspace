# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 1 refuted kept out

## Body

- Nit: tm2/pkg/bft/node/node.go:1101 [gh](https://github.com/gnolang/gno/blob/2b26f103ee16b45e29bfd24c22357df409e98870/tm2/pkg/bft/node/node.go#L1101), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: two versions
  TL;DR: a data dir an older binary left with an empty persisted chain id stays unbootable after the file is fixed, until it is wiped
  Check: Seed a memdb with saveGenesisDoc of a doc whose ChainID is "" (the pre-PR first-boot write order), then call LoadStateFromDBOrGenesisDocProvider with probeGenesisDocProvider(pk, 1): expect the boot to succeed or name the invalid persisted doc, observe the 'genesis chain id mismatch ... ("") ... ("genesis-doc-test")' error
  Details: The boot fails on the chain id check before genesisApplied is reached. The path is reachable through gnoland start: StreamingGenesisProvider runs no ValidateAndComplete (none in gno.land/pkg/gnoland outside tests), and the merge base saved the doc before MakeGenesisState validated it. The check is an unchanged context line, so the merge base fails the same way; the diff stops new dirs from reaching this state and does not heal old ones. The data dir never applied a genesis, so the wipe loses nothing: Nit. Fix, a decision for the author: let an empty persisted chain id fall through to the reset when the height-0 key is absent, or name the invalid persisted doc in the error.
  Evidence: judge test log: `boot error: genesis chain id mismatch between persisted state ("") and source genesis ("genesis-doc-test"): the genesis file backing this data dir has changed`, PASS
  Artifact: tests/judge-2-empty-chain-id_test.go

## tm2/pkg/bft/node/node.go:1092 [gh](https://github.com/gnolang/gno/blob/2b26f103ee16b45e29bfd24c22357df409e98870/tm2/pkg/bft/node/node.go#L1092) · Missing test
State: CONFIRMED, band: Missing test, angle: two versions
TL;DR: no test pins the AppState == nil gate that keeps a pre-#1941 committed data dir off the reset
Check: grep -n genesisDocKey tm2/pkg/bft/node/*_test.go shows only key comparisons (node_test.go:674, :711) and no raw write of a doc carrying AppState. Mutate node.go:1092 to `} else {` and run `go test ./tm2/pkg/bft/node -run 'LoadStateFromDBOrGenesisDocProvider|E2E|GenesisReset|SaveGenesisDoc'`: expect red, observe green.
Details: With the gate widened to `} else {` the whole tm2/pkg/bft/node package passes. Every test seeds its persisted doc through saveGenesisDoc, which sets stripped.AppState = nil, so every persisted doc in the suite has a nil AppState and the widened branch takes exactly the paths the gated one does. A data dir committed before gnolang/gno#1941 has no height-0 key, and one whose doc was saved before gnolang/gno#5684 still carries AppState: only this gate keeps that dir off resetToGenesisDoc, which would rebuild its state record from the file. Fix, mechanical: a case writing amino.MarshalJSON of a doc with AppState set under genesisDocKey, a state record at LastBlockHeight 500 and no height-0 key, then asserting the boot keeps LastBlockHeight 500.
Evidence: mutated run: `ok  github.com/gnolang/gno/tm2/pkg/bft/node 2.229s`; unmutated named run: 17 PASS lines

## tm2/pkg/bft/node/node.go:1119 [gh](https://github.com/gnolang/gno/blob/2b26f103ee16b45e29bfd24c22357df409e98870/tm2/pkg/bft/node/node.go#L1119) · Suggestion
State: CONFIRMED, band: Suggestion, angle: line by line
TL;DR: a reset replaces the persisted genesis doc and state record and leaves no line in the node's log
Check: Read node.go:1114-1120 and resetToGenesisDoc (1146-1157): no logger call and no return value distinguishes a reset from a normal load; NewNode (node.go:415) logs nothing about it either
Details: LoadStateFromDBOrGenesisDocProvider takes no logger, and resetToGenesisDoc returns (state, genDoc, nil), the same shape as the normal load. NewNode calls it at node.go:415 and goes straight on to createAndStartProxyAppConns with no log line. An operator whose earlier boot failed before InitChain succeeded, and whose genesis.json changed fields other than chain id and app hash in between, sees a normal boot. Fix, a decision for the author: a returned flag or a logger parameter so NewNode logs the reset.
Evidence: node.go:415 `state, genDoc, err := LoadStateFromDBOrGenesisDocProvider(stateDB, genesisDocProvider)` followed by createAndStartProxyAppConns; no logger call between node.go:1072 and 1157
