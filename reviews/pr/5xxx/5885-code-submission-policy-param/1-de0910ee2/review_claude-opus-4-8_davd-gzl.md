# PR [#5885](https://github.com/gnolang/gno/pull/5885): feat(gnovm): add code_submission_policy vm param

URL: https://github.com/gnolang/gno/pull/5885
Author: moul | Base: master | Files: 6 | +237 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: de0910ee2 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5885 de0910ee2`

**TL;DR:** A chain operator can flip a new `code_submission_policy` param to `"permissioned"` and list a set of allowed addresses. Only those addresses may then upload packages (`MsgAddPackage`) or run scripts (`MsgRun`). Default is `"permissionless"`, so existing chains are unaffected.

**Verdict: REQUEST CHANGES** — enforcement logic verified correct, but `genproto`/`genproto2` are red because the amino schema (`vm.proto`) and codec (`pb3_gen.go`) were hand-edited instead of regenerated, and the new access-control path ships with no test.

## Summary
Adds two `vm` params, `code_submission_policy` and `code_submitters`, plus an ante-handler gate (`checkCodeSubmissionPolicy`) that rejects `MsgAddPackage` and `MsgRun` from any signer absent from the allowlist, before typechecking, on both `CheckTx` and `DeliverTx`. Motivation: the Go typechecker is superlinear on adversarial input with no gas bound, so permissioned mode lets operators restrict who can submit code while the permissionless path matures. The gate runs after the auth and session checks; default `"permissionless"` (and the pre-upgrade empty value) leave existing chains unaffected. The pinned crossrealm38 app hash moves because default genesis now persists two new `vm` param keys.

## Examples
| code_submission_policy | code_submitters | signer | add_package / run | exec (MsgCall) |
|---|---|---|---|---|
| permissionless (default) | — | any | allowed | allowed |
| `""` (pre-upgrade chain) | — | any | allowed | allowed |
| permissioned | `[A]` | A | allowed | allowed |
| permissioned | `[A]` | B | rejected | allowed |
| permissioned | `[]` (empty) | any | rejected | allowed |

## Glossary
- ante handler — pre-execution tx stage; its checks run on both CheckTx and DeliverTx.
- addpkg — the `MsgAddPackage` transaction that uploads a package/realm to the chain.
- amino — gno's deterministic serialization codec; `vm.proto`/`pb3_gen.go` are its generated schema and binary codec for the `Params` struct.

## Fix
The gate is `checkCodeSubmissionPolicy` at [`app.go:1134-1165`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/gnoland/app.go#L1134-L1165) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/gnoland/app.go#L1134), wired into the ante chain after the session check at [`app.go:185-187`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/gnoland/app.go#L185-L187) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/gnoland/app.go#L185). Params, validation, and the governance setter live in [`params.go`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/sdk/vm/params.go#L67-L73) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/params.go#L67). Verified on de0910ee2 by driving `checkCodeSubmissionPolicy` directly: permissioned rejects a non-allowlisted `add_package` and `run`, accepts an allowlisted signer, leaves `exec` untouched, and an empty allowlist under permissioned blocks every submitter. Enforcement is correct; the blockers are the ungenerated codec and the absent test. [repro](comment_claude-opus-4-8.md)

## Invariant catalog
Walked every class. Touched: **Gas** (the gate adds a metered `GetParams` read to every tx — see Suggestions); **Caller & access control** (tx-level identity via `msg.GetSigners()`, verified by the auth ante — correct layer, not the spoofable stack-walkers); **Error & panic handling** (`WillSetParam` panics on a bad governance address, consistent with the existing `storage_fee_collector` case; the gate returns an error result, no panic). Determinism holds: the check iterates ordered slices and uses the `allowed` map only for lookup; param JSON encoding is deterministic. No global mutable state, no realm-state or storage-deposit surface. Others not applicable (no VM-eval, coin/banker, or type-check change).

## Critical (must fix)
None.

## Warnings (should fix)
- **[schema and codec are generated, hand-edited here — CI red]** `gno.land/pkg/sdk/vm/pb3_gen.go:1173` — `pb3_gen.go` and `vm.proto` are generated from the `Params` struct; this PR hand-wrote the codec and never updated `vm.proto`, so both generator checks fail.
  <details><summary>details</summary>

  `vm.proto` still ends at `iter_next_cost_flat = 13` ([`vm.proto:83`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/sdk/vm/vm.proto#L83) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/vm.proto#L83)); it has no field 14/15 (grep-confirmed). `genproto` fails wanting to add `string code_submission_policy = 14` and `repeated string code_submitters = 15`; `genproto2` fails because the hand-written unmarshal in [`pb3_gen.go:1582`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/sdk/vm/pb3_gen.go#L1582) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/pb3_gen.go#L1582) differs from the generator's output. The regenerated code is behaviorally identical (the CI diff is only local-variable renames, `addr`/`elem` → `ev`/`rv`), so there is no wire-format risk, but both checks are merge-blocking. Note that this whole-struct binary codec is not what stores params (that path is per-field JSON in `tm2/pkg/sdk/params/amino_helper.go`), so it does not affect the app hash. Fix: add the two fields to the struct only, then run `make -C misc/genproto && make -C misc/genproto2` and commit the regenerated `vm.proto` and `pb3_gen.go`.
  </details>

## Nits
- `gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:71` — the consensus hash was bumped without a note; the file's bump-log ends on a different, already-merged PR.
  <details><summary>details</summary>

  The block above the constant logs each hash bump and why ([`apphash_crossrealm38_test.go:52-70`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go?plain=1#L52-L70) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52)). This PR changed the value at [`apphash_crossrealm38_test.go:71`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go?plain=1#L71) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L71) but added no entry, so the last note ("editing gnovm/stdlibs/math/rand/example_test.gno") now misattributes the current value. `TestAppHashCrossrealm38` passes on de0910ee2, so the value is right; add a one-line bump note stating the shift comes from the two new default `vm` params in genesis. Fix: add the bump-log line.
  </details>

## Missing Tests
- **[access-control path ships untested]** `gno.land/pkg/gnoland/app.go:1134` — no test exercises `checkCodeSubmissionPolicy` or the new `Params.Validate` branches; the PR's own "verify unauthorized MsgAddPackage is rejected" box is unchecked.
  <details><summary>details</summary>

  The gate reads only params and each message's signers, so a bare `std.Tx` drives it without signatures. The ready-to-add test at [`tests/code_submission_policy_test.go`](../../../../../reviews/pr/5xxx/5885-code-submission-policy-param/1-de0910ee2/tests/code_submission_policy_test.go) covers permissionless-allows-all, empty-policy compat, permissioned reject/allow for both `add_package` and `run`, `exec` untouched, empty-allowlist-blocks-all, and the three `Validate` rejections. It also guards the fail-open coupling in the next Suggestion: a rename of `MsgAddPackage.Type()` would make `TestCSP_PermissionedRejectsNonAllowlisted` fail. All cases pass on de0910ee2. Fix: add the test.
  </details>

## Suggestions
- `gno.land/pkg/gnoland/app.go:1135` — the gate reads the full `Params` struct on every transaction, even ones with no code message and even on a permissionless chain.
  <details><summary>details</summary>

  `checkCodeSubmissionPolicy` calls `vmk.GetParams(ctx)` unconditionally at [`app.go:1135`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/gnoland/app.go#L1135) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/gnoland/app.go#L1135), a metered per-field store read, before it looks at the messages. The ante handler already read the same struct one call earlier for gas config at [`app.go:150`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/gnoland/app.go#L150) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/gnoland/app.go#L150) and discarded it. Scanning `tx.GetMsgs()` for an `add_package`/`run` message first, and returning before the read when there is none, would drop the extra read for the common call/send/permissionless case. Fix: gate the param read on the presence of a code-submission message.
  </details>
- `gno.land/pkg/sdk/vm/params.go:186-205` — `Validate` accepts `permissioned` with an empty `CodeSubmitters`, which silently blocks all deployments chain-wide.
  <details><summary>details</summary>

  A single param proposal that sets `code_submission_policy=permissioned` without also setting `code_submitters` (or one that later clears the list) passes `Validate` ([`params.go:186-205`](https://github.com/gnolang/gno/blob/de0910ee2/gno.land/pkg/sdk/vm/params.go#L186-L205) · [↗](../../../../../.worktrees/gno-review-5885/gno.land/pkg/sdk/vm/params.go#L186)) and makes every `add_package`/`run` fail (`TestCSP_PermissionedEmptyAllowlistBricksAll` confirms). It is recoverable by a follow-up proposal, and could be an intentional freeze lever, so this is a decision not a defect. Fix: either reject permissioned-with-empty-allowlist in `Validate`, or leave a comment that the empty-list freeze is intentional.
  </details>

## Open questions
- The gate matches messages by literal `"vm"`/`"add_package"`/`"run"` at `app.go:1148-1152`, with no compile-time link to the message types' own `Type()`. If a future message type also feeds code through the typechecker, or an existing type string is renamed, the allowlist silently fails open. Not posted: the pattern is consistent with `sessionAlwaysDenied` (`app.go:1174`), no shared constant exists to reference, and the added test catches a rename of the current types.
- At genesis (block height 0) the gate still runs; a chain configured permissioned at genesis must include its initial package creators in `code_submitters` or genesis package loading fails. Not posted: default is permissionless, and a permissioned-at-genesis operator controls both lists.
