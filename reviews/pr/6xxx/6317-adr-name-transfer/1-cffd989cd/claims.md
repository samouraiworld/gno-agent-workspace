# Claims: https://github.com/gnolang/gno/pull/6317 round 1, cffd989cd, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | gno.land/adr/pr6317_name_transfer.md:155 | grep -n 'func Propose\\|Set(\\|NewSimpleExecutor' examples/gno.land/r/sys/users/admin.gno; grep -n 'GenesisTxResultHandler' gno.land/pkg/gnoland/app.go; grep -n 'type SimpleExecutor' -A4 examples/gno.land/r/gov/dao/types.gno | grep admin.gno: lines 143-184 build proposals with no Set; app.go:325 PanicOnFailingTxResultHandler, app.go:333-334 Noop on SkipFailingTxs |  | hot |  |
| 2 | CONFIRMED | Nit | gno.land/adr/pr6317_name_transfer.md:414 | sed -n 370,380p contribs/gnogenesis/internal/fork/generate.go; grep -rln 'migration-tx\\|migrationTxs' contribs/gnodev gno.land/pkg/integration; grep -n -i 'gnodev\\|txtar' gno.land/adr/pr6317_name_transfer.md | generate.go:373-377 'Append --migration-tx files at the END of appState.Txs'; grep -rln 'migration-tx\|migrationTxs' contribs/gnodev gno.land/pkg/integration: no hit; grep ADR for gnodev/fresh genesis activation: none |  | hot |  |
| 4 | CONFIRMED | Nit | gno.land/adr/pr6317_name_transfer.md:91 | grep -n 'func Resolve\\|return data\\|makeUserDataSafe' examples/gno.land/r/sys/users/users.gno; sed -n 88,106p gno.land/adr/pr6317_name_transfer.md | users.gno:27 'data := raw.(*UserData)' returned unchanged; users.gno:48,53 rotree.Wrap(..., makeUserDataSafe) |  | hot |  |
| 3 | REFUTED | Nit | gno.land/adr/pr6317_name_transfer.md:287 | sed -n 500,525p gno.land/pkg/sdk/vm/keeper.go; sed -n 464,467p gno.land/adr/pr6317_name_transfer.md | keeper.go:509-514 `if err != nil { return err }` after keeper.go:505-506 `if usersPkg == nil { return nil }`: the same asymmetry holds today |  | hot |  |

Hit rate per tier, from the rows above: hot 3/4 confirmed over 1 files, warm 0/0 confirmed over 0 files, cold 0/0 confirmed over 0 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach | gno.land/adr/pr6317_name_transfer.md:448 | Suspected false claim that buildHardforkGenesis drops inherited PastChainIDs and inherited txs lose their chain-id override | sed -n 448,458p contribs/gnogenesis/internal/fork/generate.go: 'appState.PastChainIDs = []string{originalChainID}'; grep -n isPastChainID gno.land/pkg/gnoland/app.go:907 applies metadata.ChainID only when isPastChainID(pastChainIDs, ...): the claim holds |
| reach | gno.land/adr/pr6317_name_transfer.md:620 | Suspected false claim that Amino JSON writes zero-valued fields without omitempty | grep -n omitempty tm2/pkg/amino/json_encode.go:359: 'Unlike Amino:binary, we don't skip null fields unless "omitempty"': the claim holds |
| reach | gno.land/adr/pr6317_name_transfer.md:637 | Suspected false claim that gpao's dedup key hashes only the submitted MemPackage | grep -n -A3 '^func candidateKey' contribs/gpao/oracle.go:756-758: sha256 of amino.MustMarshal(mpkg) keyed by path; mpkg comes from MsgAddPackage (main.go:4): the claim holds |
| reach | gno.land/adr/pr6317_name_transfer.md:97 | Suspected false claim that mutating a read-only-tree copy leaves the real record untouched | sed -n 124,155p examples/gno.land/r/sys/users/users_test.gno: TestROStores calls roData.Delete(0, cur) and asserts nameStore's record stays not deleted: the claim holds |
| reach | gno.land/adr/pr6317_name_transfer.md:135 | Suspected false claim that NewFilterByTier restricts only voting and any member can create | sed -n 40,70p examples/gno.land/r/gov/dao/impl/v0/govdao.gno: PreCreateProposal checks only membership; PostCreateProposal stores tatv=[T1] for FilterByTier T1 at creation: the claim holds |

## Completeness

- Catalog: the PR adds one ADR and no code, so the gno delta skips the invariant catalog walk. No test, fixture or guard is edited, so there is no sibling shape to sweep.
- Thin angle, claims: swept by the judge, every ADR claim read held. `verifier.gno:227-228` returns true while `!isEnabled`; `runtime.GetSessionInfo` exists (`gnovm/stdlibs/chain/runtime/native.gno:32`); no `r/sys` realm sets `private = true`; `preregister.gno:26-37` lists exactly the ten names of ADR:538-539; `checkCLASignature` runs at `keeper.go:1018`, `keeper.go:1142` and `keeper_inert.go:299`; `ErrNameTaken`, `ErrDeletedUser`, `ErrAlreadyHasName` exist in `users/errors.gno`; `params.go:539-540` defaults an empty `SysNamesPkgPath`; namereg `Register` panics on a returned error (`users.gno:71,80`), matching ADR:230-231. No new candidate, so `candidates/reflector.json` holds none.
- Candidate 1: a design that needs neither the open-proposal scan nor a boot-time refusal is a callback that records at creation whether the registry was active and refuses to run once the flag is set. Not run, so it stays out of the draft.
- Candidate 3 refuted: today's `checkNamespacePermission` already passes on an undeployed names realm (`keeper.go:505-506`) and refuses on a call error (`keeper.go:512-513`), and ADR:464-467 ships keeper and realms behind one upgrade.
- Candidate 4 confirmed and skipped: the ADR:677 offer and accept test fails on a copy.
- Candidate 1, parent's final check: the draft's `SkipFailingTxs` clause was dropped. `sed -n 374,386p;636,660p gno.land/pkg/gnoland/app.go` shows `StrictReplay`, which ADR:455 requires, panicking on any failed non-skipped genesis tx whatever the result handler, so a boot with the activation tx failed and the flag unset needs an operator ignoring the ADR. The "source chain is halted" bullet was dropped as unmeasured.
- No scratch worktree was created: every candidate was a Nit on the ADR's text, settled by reading the head, and nothing was run.
- The round's `overview.md` was already on disk, written by the overview agent, and was not rewritten. Only the lines `round check` flagged were edited: four shas dropped (95, 96, 123, 296), `approved` to `agreed` (191), `pull request changes` to `pull request edits` (282), and five bare `#<number>` made `gnolang/gno#<number>` (225, 238, 246, 254, 258). It still names no generating model.

## Retro

From `./scripts/review-retro.py <workflow dir> --model claude-opus-5-5`: 2 agents, 57 turns, 93k output, 7.3M cache read, about $6.13 measured against $5 projected, 20 min against 21.

- What failed: the judge banded candidate 1 on a `SkipFailingTxs` boot path without reading `StrictReplay`, which the ADR itself requires and which panics on any failed genesis tx; the parent's final check caught it and the clause was dropped. Every turn ran on claude-opus-5-5, no fallback, no refusal.
- What worked: the solo shape held a 747-line ADR in one read; 3 of 4 candidates confirmed, and the refuted one was refuted with the proving lines.
- Hit rate per tier: hot 3/4 over 1 file; warm and cold empty.
- Upgrade, proposed and not applied this turn: a judge checking a failure path that depends on a node or genesis flag reads every flag the target itself mandates before banding, estimated one extra read per such candidate.
