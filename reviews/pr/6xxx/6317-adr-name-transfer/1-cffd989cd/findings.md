# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 1 refuted kept out

## gno.land/adr/pr6317_name_transfer.md:91 [gh](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/adr/pr6317_name_transfer.md#L91) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: ResolveID is not pinned to return the authoritative pointer, and the guard of ADR:104-105 refuses a copy
Check: grep -n 'func Resolve\|return data\|makeUserDataSafe' examples/gno.land/r/sys/users/users.gno; sed -n 88,106p gno.land/adr/pr6317_name_transfer.md
Details: ResolveName and ResolveAddress return the stored pointer (users.gno:12, users.gno:27-32) while the rotree getters return makeUserDataSafe copies (users.gno:48-56). ADR:91 calls ResolveID part of a 'read-only API' one line above the copies paragraph, and AcceptTransfer (ADR:230) and the id-bound proposals call mutators on its result, which pass the addressStore.Get(...) == u guard only for the stored pointer. The offer -> accept filetest of ADR:677 fails on a copy, so an implementation cannot ship the wrong reading unnoticed.
Evidence: users.gno:27 'data := raw.(*UserData)' returned unchanged; users.gno:48,53 rotree.Wrap(..., makeUserDataSafe)

## gno.land/adr/pr6317_name_transfer.md:155 [gh](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/adr/pr6317_name_transfer.md#L155) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: Activation is told to refuse while address-bound users proposals are open, but r/sys/users records nothing when such a proposal is created, and a refused genesis tx stops the boot
Check: grep -n 'func Propose\|Set(\|NewSimpleExecutor' examples/gno.land/r/sys/users/admin.gno; grep -n 'GenesisTxResultHandler' gno.land/pkg/gnoland/app.go; grep -n 'type SimpleExecutor' -A4 examples/gno.land/r/gov/dao/types.gno
Details: ProposeUpdateName (admin.gno:143-166) and ProposeDeleteUser (admin.gno:170-184) only return dao.NewProposalRequest(..., dao.NewSimpleExecutor(0, cur, cb, "")); the realm stores nothing, and SimpleExecutor keeps only callback, description and creationRealm (types.gno:185-188), so the activation tx can find open proposals only by walking GovDAO's proposal store. A refusal in the appended activation tx panics InitChain under the default PanicOnFailingTxResultHandler (app.go:59, app.go:325); booted with genesis SkipFailingTxs it is a Noop (app.go:333-334), so the chain boots with the flag unset and, since the entry point requires height 0 (ADR:414), transfers stay off. The source chain is halted by then, so the open proposal cannot be executed or rejected there. Capturing the dormant mode in the callback at creation, refusing to run once the flag is set, needs neither the scan nor a refusal at boot.
Evidence: grep admin.gno: lines 143-184 build proposals with no Set; app.go:325 PanicOnFailingTxResultHandler, app.go:333-334 Noop on SkipFailingTxs

## gno.land/adr/pr6317_name_transfer.md:414 [gh](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/adr/pr6317_name_transfer.md#L414) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: Only an upgrade genesis gets the activation tx, so a chain booted fresh from examples never activates and its transfers stay refused
Check: sed -n 370,380p contribs/gnogenesis/internal/fork/generate.go; grep -rln 'migration-tx\|migrationTxs' contribs/gnodev gno.land/pkg/integration; grep -n -i 'gnodev\|txtar' gno.land/adr/pr6317_name_transfer.md
Details: ADR:411-415 places the activation tx after replayed history and requires height 0; ADR:428-430 makes every transfer entry point require the flag. The only appender is the fork tool's --migration-tx (generate.go:373-377); no loader under contribs/gnodev or gno.land/pkg/integration references a migration tx, and the ADR names no other route. gnodev, a fresh testnet and the integration txtar of ADR:730-732 (offer/accept from a new key) therefore boot unactivated and can never call OfferTransfer or AcceptTransfer, while the keeper rule is live from block 1 (ADR:459-462).
Evidence: generate.go:373-377 'Append --migration-tx files at the END of appState.Txs'; grep -rln 'migration-tx|migrationTxs' contribs/gnodev gno.land/pkg/integration: no hit; grep ADR for gnodev/fresh genesis activation: none
