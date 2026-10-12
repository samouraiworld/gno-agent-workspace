# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 1 refuted kept out

## tm2/pkg/bft/node/node.go:1084 [gh](https://github.com/gnolang/gno/blob/4b45296ca80ee11748e28dff02a178f56800aa38/tm2/pkg/bft/node/node.go#L1084) · Missing test
State: CONFIRMED, band: Missing test, angle: removed
TL;DR: no test fails the first boot on a failed genesis doc write; the only write-failure fixture is armed after the first load returned
Check: Mutate node.go:1084 to drop the error check, then run go test ./tm2/pkg/bft/node: expect a red test, observe all green.
Details: With the first-boot error check removed the whole tm2/pkg/bft/node package passes. genesisDocSetErrDB (node_test.go:705) is used once, at node_test.go:722, where d.fail is set only after the first LoadStateFromDBOrGenesisDocProvider call returned, so it reaches resetToGenesisDoc's write and never the first-boot one. saveGenesisDoc has two callers, node.go:1084 and node.go:1146, and only the second is pinned. Fix, mechanical: a case setting d.fail before the first call and requiring ErrorContains 'injected write error'.
Evidence: mutation A: `ok  github.com/gnolang/gno/tm2/pkg/bft/node 2.226s`; grep -rn 'saveGenesisDoc(': node.go:1084, node.go:1146, node_test.go:123
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the first-boot branch persists the genesis doc before it is validated, so a doc with an invalid chain_id is pinned and the corrected file is refused as a changed genesis
Check: a provider returning ChainID "" makes the first load fail; a second load on the same db with ChainID "fixed" fails with 'genesis chain id mismatch'
Details: First load: `error in genesis file: Genesis doc must include non-empty chain_id`, after saveGenesisDoc at node.go:1084 already wrote the doc. Second load with chain_id fixed: `genesis chain id mismatch between persisted state ("") and source genesis ("fixed")`. resetToGenesisDoc runs sm.MakeGenesisState before its write (node.go:1142-1146), the shape this branch repeats without. Only a provider that skips ValidateAndComplete reaches it, since GenesisDocFromFile validates at types/genesis.go:178; the base behaves the same, and recovery is wiping a data dir with nothing committed, hence Nit. Fix, mechanical: build or validate the genesis state before saveGenesisDoc in the first-boot branch.
Evidence: head and base alike: `JUDGE2 first err=error in genesis file: Genesis doc must include non-empty chain_id` then `JUDGE2 second err=genesis chain id mismatch between persisted state ("") and source genesis ("fixed")`
Artifact: tests/judge-1-2-genesis-doc-probes_test.go

## tm2/adr/pr5995_initchain_rejection_aborts_boot.md:76 [gh](https://github.com/gnolang/gno/blob/4b45296ca80ee11748e28dff02a178f56800aa38/tm2/adr/pr5995_initchain_rejection_aborts_boot.md#L76) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: an in-memory node given a nil AppState booted at the merge base and now stops in the handshake, and the ADR's Consequences do not name it
Check: Boot NewInMemoryNode with Genesis.AppState=nil at base and at head: base boots, head returns the handshake error.
Details: loadAppState's switch is identical at both commits (head app.go:491-497, base app.go:504-510): nil and *GnoGenesisState fall to `invalid AppState of type %T`. The base ignored the response error; the head's res.IsErr() check at replay.go:346 aborts on it. The *GnoGenesisState half is named in the ADR's Decision, 'Pass the default genesis state by value', and the deleted app.go comment was already untrue at the base, where the nil case logged an error rather than a warning. No non-test caller in gno.land, contribs or tm2 passes a nil AppState. Rebanded from Suggestion: what remains is a Consequences bullet for out-of-tree NewInMemoryNode callers, or a `case nil` if the author wants that shape to keep booting; a decision for the author.
Evidence: head: `JUDGE1001 err=error during handshake: error on replay: InitChain rejected the genesis: invalid AppState of type <nil> node_built=false`; base: `JUDGE1001 err=<nil> node_built=true`
Artifact: tests/judge-1001-nil-appstate_test.go

## tm2/pkg/bft/node/node.go:1116 [gh](https://github.com/gnolang/gno/blob/4b45296ca80ee11748e28dff02a178f56800aa38/tm2/pkg/bft/node/node.go#L1116) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: once the height-0 key exists, a genesis file with another validator set boots with the persisted validators and the file's AppState, and nothing reports the difference
Check: LoadStateFromDBOrGenesisDocProvider with validator pk1, SaveABCIResponses(0), then again with validator pk2 and AppState v2: expect an error or a warning, observe err nil, pk1 kept, AppState v2
Details: The applied branch compares only ChainID (node.go:1096) and AppHash (node.go:1101), then takes freshDoc.AppState at node.go:1116. The probe returns err=<nil>, the persisted pk1 validator and AppState v2, at the head and at the merge base alike: the behaviour predates the branch, which kept the applied branch as it was. Reaching it needs InitChain to have succeeded with no block committed and the file swapped in that window. Whether to log or refuse the divergence is a decision for the author; not in this change's scope.
Evidence: head: `JUDGE1 err=<nil> validators_from_persisted_pk1=true appstate=v2`; base: the same line
Artifact: tests/judge-1-2-genesis-doc-probes_test.go
