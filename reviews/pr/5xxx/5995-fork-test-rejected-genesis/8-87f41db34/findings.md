# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 1 refuted kept out

## Body

- Nit: tm2/adr/pr5949_remove_secp256k1_validators.md:71 [gh](https://github.com/gnolang/gno/blob/87f41db347325ab339a33c87227443292d4f8e08/tm2/adr/pr5949_remove_secp256k1_validators.md#L71), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: removed
  TL;DR: pr5949's ADR still gives tm2 silently discarding ResponseInitChain.Error as the reason the pubkey-type gate panics, which replay.go:346 no longer does
  Check: grep -rn 'silently' tm2/adr/pr5949_remove_secp256k1_validators.md; sed -n 343,349p tm2/pkg/bft/consensus/replay.go
  Details: Lines 71-72 read that ResponseInitChain.Error 'is silently discarded by tm2, so the check panics'. replay.go:346-348 now returns 'InitChain rejected the genesis' on res.IsErr(), and the branch rewrote the same reason in app.go:427-430 and app_test.go:408-411 but not here. The ADR records PR 5949's decision as of its merge, so amending it is the author's call; the file is outside the diff, so it can only be a Body bullet.
  Evidence: pr5949 ADR:71 '`ResponseInitChain.Error` is silently' / :72 'discarded by tm2, so the check **panics**'; replay.go:346 'if res.IsErr() {'
- Nit: tm2/pkg/bft/consensus/replay.go:353 [gh](https://github.com/gnolang/gno/blob/87f41db347325ab339a33c87227443292d4f8e08/tm2/pkg/bft/consensus/replay.go#L353), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: lines
  TL;DR: ReplayBlocks saves the height-0 responses before refusing a genesis with no validators, so the restart after adding one keeps the persisted doc with none and is refused again
  Check: copy TestE2ERejectThenCorrectedGenesisBoots with an app returning req.Validators and a provider that sets doc.Validators = nil on the first build and keeps them on the second; expect the second build to succeed
  Details: ValidateAndComplete (types/genesis.go:131-165) accepts an empty validator list; only Validate rejects it, and neither the provider nor MakeGenesisState calls Validate. sm.SaveABCIResponses(h.stateDB, 0, ...) at replay.go:353 runs before the 'validator set is nil in genesis and still empty after InitChain' return at replay.go:367, so genesisApplied reads true on the restart and the corrected file is ignored: InitChain saw 0 validators on both boots. The merge base has no reset at all, so it refuses every correction the same way; the branch's sweep of refusals that must not mark the genesis applied missed this one. Recovery is wiping a data dir holding nothing committed, hence Nit. Fix, run: the same condition checked above the save (wasStateFresh && len(res.Validators) == 0 && len(h.genDoc.Validators) == 0); the test then sees InitChain get 0 then 1 validators and the second boot succeed, and the TestHandshake tests in tm2/pkg/bft/consensus pass.
  Evidence: head: 'validators seen by InitChain per boot: [0 0]', second build 'error during handshake: error on replay: validator set is nil in genesis and still empty after InitChain', FAIL. fixed: '[0 1]', PASS
  Artifact: tests/judge-1-nil-valset-refusal.go

## tm2/pkg/bft/node/node.go:1144 [gh](https://github.com/gnolang/gno/blob/87f41db347325ab339a33c87227443292d4f8e08/tm2/pkg/bft/node/node.go#L1144) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: resetToGenesisDoc reports success when the overwrite of the persisted genesis doc fails, so the refused doc becomes authoritative once the corrected genesis is applied
Check: sed -n 1139,1147p and 1171,1178p tm2/pkg/bft/node/node.go: saveGenesisDoc returns nothing and discards db.SetSync's error; run: memdb wrapped so SetSync(genesisDocKey) fails once, refuse InitialHeight 1000, reset to 100, SaveABCIResponses(0), reload and read doc.InitialHeight
Details: saveGenesisDoc ends in db.SetSync(genesisDocKey, b) with the error dropped, and resetToGenesisDoc calls it, then SaveState, then returns the corrected doc to the handshake. With one injected SetSync failure on the doc key the reset returns doc.InitialHeight=100, and after the height-0 key is written the next load returns the persisted doc at InitialHeight=1000 beside a state record at 100. On a chain that has since committed blocks, ReplayBlocks' fix-a then realigns the state to the refused height and the StateBlockHeight > StoreBlockHeight panic follows, by read. At the merge base the doc is written only when absent, so a dropped write leaves no doc and the next boot writes it again; the overwrite is new here. Needs an I/O error from SetSync, so latent. Fix, run: saveGenesisDoc returns the SetSync error and both callers return it; the test then sees the reset fail with 'injected I/O error', and every LoadStateFromDBOrGenesisDocProvider test, TestE2ERejectThenCorrectedGenesisBoots and TestNodeStartStop pass. The state write's own dropped error is candidate 2003.
Evidence: head: 'reset: doc.InitialHeight=100 state.InitialHeight=100', 'restart: doc.InitialHeight=1000 state.InitialHeight=100', expected: 100 actual: 1000, FAIL. fixed: 'reset surfaced the failed write: injected I/O error', PASS
Artifact: tests/judge-1001-dropped-doc-write.go

## tm2/pkg/bft/node/node.go:1145 [gh](https://github.com/gnolang/gno/blob/87f41db347325ab339a33c87227443292d4f8e08/tm2/pkg/bft/node/node.go#L1145) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: resetToGenesisDoc's sm.SaveState discards the write error, as every write in the state store does
Check: read store.go's write sites and the DB interface
Details: sm.SaveState returns nothing, and saveState discards the error db.SetSync returns at tm2/pkg/bft/state/store.go:134, which the DB interface declares at tm2/pkg/db/types.go:19. The store discards the error at all five of its write sites, 134, 203, 237, 330 and 403, so this is a store-wide convention predating the branch, not a defect the reset adds; the reset only joins the callers. Ships SKIP: out of this change's scope.
Evidence: grep -n 'db.Set(\|db.SetSync(' tm2/pkg/bft/state/store.go: 134, 203, 237, 330, 403, none checked; tm2/pkg/db/types.go:18-19 Set and SetSync return error
