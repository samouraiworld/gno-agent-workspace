# PR #5653: feat: test-13 hardfork release candidate

URL: https://github.com/gnolang/gno/pull/5653
Author: aeddi | Base: master | Files: 86 | +6345 -68
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `f45cc5c88` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5653 f45cc5c88`

**TL;DR:** End-to-end tooling that builds the test-13 chain's starting state by replaying gnoland1's full transaction history on top of a fresh genesis, rewriting a handful of old transactions whose code no longer compiles against current master. The change ships a `gnogenesis fork` toolset, a one-time signature-verification bypass used only while replaying those rewritten transactions at chain-init, and a 1,759-line operator script that produces and self-audits the final `genesis.json`.

**Verdict: NEEDS DISCUSSION** — the red-CI blocker from round 1 is gone (all required jobs green on `f45cc5c88`) and the `SkipSigVerificationKey` boundary still holds, but four open items need author/maintainer input before merge on a consensus-critical genesis: the 13 `TMP`-tagged commits' final disposition, cross-validator reproduction of the locked genesis sha, the undocumented sequence/pubkey side effects of the caller-swap patches, and the absence of a generate→inspect/replay round-trip test in-tree.

## Round note

Re-review since round 1 (`963ba05`, +42 commits, mostly a master merge). PR-authored delta since round 1 is one file: `gen-genesis.sh` relaxes the cached-txs height pre-flight from strict equality to an upper bound and relocks `CHECKSUMS_DATA`. The security-critical Go code (`app.go`, `ante.go`, `helpers.go`, `GnoTxMetadata`) is byte-identical to round 1 and re-verified at this head. Round 1's red-CI Critical is dropped (CI green). Round 1's Q4 (audit strictness) is answered by the code and dropped. All other findings carry forward with anchors re-cut.

## Summary

Tooling that produces a test-13 hardfork genesis from gnoland1's state in two phases: a fresh base genesis with the test-13 valset, faucets, and a single-use `r/test13/rotate` realm, then a replay of gnoland1's 1.4M+ historical txs on top, with per-tx caller/body patches that compensate for API drift since the original txs were signed. The novel security surface is `SkipSigVerificationKey`, a context-value sentinel that makes the auth ante skip signature verification for a tx. It is set in exactly one place ([`app.go:740-742`](https://github.com/gnolang/gno/blob/f45cc5c88/gno.land/pkg/gnoland/app.go#L740-L742) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L740-L742)) and only when `metadata.Source == SourcePatched`. The production block path never reaches it: `BaseApp.DeliverTx` does not thread a `ContextFn`, and the only non-test caller of the ctxFn-accepting [`BaseApp.Deliver`](https://github.com/gnolang/gno/blob/f45cc5c88/tm2/pkg/sdk/helpers.go#L61-L77) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/helpers.go#L61-L77) is `deliverGenesisTx` at [`app.go:812`](https://github.com/gnolang/gno/blob/f45cc5c88/gno.land/pkg/gnoland/app.go#L812) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L812).

## Glossary

- `SkipSigVerificationKey` — context-value sentinel that makes the auth ante skip Phase 3 (pubkey set + sig verify + sequence increment) for that tx.
- `SourcePatched` — provenance tag (`"patched"`) set on a historical tx whose body was rewritten by `--patch-txs`; the only `Source` value that triggers `SkipSigVerificationKey`.
- `AnnotatedTx` — `{tx, metadata, reason}` jsonl schema for `--patch-txs` and `--migration-tx` inputs; `reason` flows into `metadata.Note`.
- `verify_checksum` — `gen-genesis.sh` helper that sha256s an artifact against the inline `CHECKSUMS_DATA` manifest; exact-match fail for listed paths, advisory note for unlisted paths.
- TMP — author tag on 13 commit subjects in this PR; final disposition undocumented.

## Examples

What changed in `gen-genesis.sh` since round 1, behaviorally: the cached-txs height pre-flight (jsonl-file source mode).

| Cache max tx height vs HALT_HEIGHT (1485629) | Round 1 (`-ne`) | This head (`-gt`) |
|---|---|---|
| 1485000 (complete cache, tx-less trailing blocks) | rejected (false positive) | accepted |
| 1485629 (exact) | accepted | accepted |
| 1485630 (extends past halt) | rejected | rejected |
| short / truncated cache | rejected at pre-flight | accepted at pre-flight, caught later by final genesis-sha checksum |

## Fix

`gen-genesis.sh` [lines 1487-1516](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L1487-L1516) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L1487-L1516) replaces the `MAX_HEIGHT -ne HALT_HEIGHT` reject with `MAX_HEIGHT -gt HALT_HEIGHT`. The pre-flight scans only tx-bearing blocks (the jsonl carries no tx-less blocks), so a complete archive whose final pre-halt blocks were empty has a max tx height below `HALT_HEIGHT` and was wrongly rejected. The load-bearing constraint is that exactness is still enforced downstream: the final `genesis.json` sha is locked in `CHECKSUMS_DATA` ([line 231](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L231) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L231)) and re-checked at [line 1703](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L1703) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L1703), so a truncated cache produces a wrong replay and fails the final checksum. Detection of a short cache moves from pre-flight to end-of-run; reproducibility is unaffected.

## Benchmarks / Numbers

| Surface | Count | Notes |
|---|---|---|
| PR-authored files changed since round 1 | 1 | `gen-genesis.sh` (height check + CHECKSUMS); `go.mod`/`go.sum` deltas are base-merge churn, identical to the new merge-base |
| TMP-tagged non-merge commits | 13 / 40 | unchanged from round 1; PR body still does not explain the convention |
| `gen-genesis.sh` length | 1,759 lines | +2 since round 1 |
| Required CI on head | 167 pass / 0 fail / 2 skip | `gno-checks/lint` and all `main/test` matrix entries now green |
| Final genesis sha (locked) | `56f56e13…` | was `64251ed9…` in round 1; relocked after the master merge |

## Critical (must fix)

None.

## Warnings (should fix)

- **[unclear which commits survive merge]** [`gno.land/pkg/gnoland/app.go:740`](https://github.com/gnolang/gno/blob/f45cc5c88/gno.land/pkg/gnoland/app.go#L740) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L740) — 13 of 40 non-merge commits are subject-tagged `TMP`, including the ones that ship production paths (`SkipSigVerificationKey`, the `GnoTxMetadata` provenance fields, `--patch-txs`, fork `inspect`), with no stated disposition.
  <details><summary>details</summary>

  `TMP` usually means squash-me or extract-me, but these 13 implement code that ships in this PR, not WIP scaffolding. The reviewer cannot tell whether `TMP add --patch-txs flag` is permanent, will be squashed, or will be extracted like the gov-dao scripts were to #5658. On a consensus-critical RC the final commit history matters for bisect and audit. Fix: in the PR body, map each `TMP` subject to its final disposition (squash into X / extract to PR #Y / keep and drop the prefix).
  </details>

- **[patch jsonl assembled in shell, only validated at replay]** [`misc/deployments/test13.gno.land/gen-genesis.sh:682-825`](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L682-L825) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L682-L825) — `txn_dir_to_jsonl` inlines `.gno` bodies into `tx.msg[0].package.files[0].body` and walks `pkg_dir` into `files` with jq, with no AnnotatedTx schema check until `gnogenesis fork generate` rejects a malformed line.
  <details><summary>details</summary>

  The per-tx directory layout was chosen for diffability, but every operator hand-editing a `transactions/patched/<bucket>/h<height>/meta.json` is one mis-escaped quote from a jsonl line that fails opaquely at chain replay. The maintainer already pushed back on a separate Go helper (`emit-migration-txs`) in favor of a `gnogenesis` subcommand; the same argument applies here. A `gnogenesis fork` subcommand that decodes the AnnotatedTx struct at emit time would typecheck the output and ship test coverage instead of relying on integration-level shell smoke. Fix: track as a follow-up; for the RC, add a decode-each-line verify step before `fork generate` runs.
  </details>

- **[caller-swap patches mutate state the docs don't mention]** [`gno.land/pkg/gnoland/app.go:769-793`](https://github.com/gnolang/gno/blob/f45cc5c88/gno.land/pkg/gnoland/app.go#L769-L793) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L769-L793) + [`tm2/pkg/sdk/auth/ante.go:189-281`](https://github.com/gnolang/gno/blob/f45cc5c88/tm2/pkg/sdk/auth/ante.go#L189-L281) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/auth/ante.go#L189-L281) — boards2-cascade patches rewrite `MsgCall.Caller` to the GovDAO multisig but keep `SignerInfo` pointing at the original signer for `keyOf` matching; the ante's `SkipSigVerificationKey` `continue` ([ante.go:194-196](https://github.com/gnolang/gno/blob/f45cc5c88/tm2/pkg/sdk/auth/ante.go#L194-L196) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/auth/ante.go#L194-L196)) skips pubkey-set and sequence-increment for these txs.
  <details><summary>details</summary>

  The `continue` short-circuits the entire signature loop body, so for a patched tx no pubkey is set on first use (ante.go:238 skipped) and no sequence is incremented (ante.go:275/278 skipped); the force-set at app.go:777-792 still runs against the original `SignerInfo` address, not the patched caller. On a fresh chain this is consistent, but the `SkipSigVerificationKey` doc block (app.go:734-742) describes only the body-invalidates-sig angle and is silent on the seq/pubkey side effects. The next person reading this reconstructs the reasoning from three files. Fix: add a short side-effects note to the `SkipSigVerificationKey` doc covering (a) sequence not incremented for the caller, (b) pubkey not set on first use, (c) the force-set still keys on the original `SignerInfo` address.
  </details>

- **[post-replay state silently diverges from gnoland1 halt-state]** [`misc/deployments/test13.gno.land/transactions/patched/validator-noops/`](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/transactions/patched/validator-noops/) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/transactions/patched/validator-noops/) — 17 patches replace v2 valset proposal bodies with `func main(cur realm) {}`, preserving the original fee deduction but no-oping the effect.
  <details><summary>details</summary>

  This is the documented design: v2 is unseeded on test-13 because the EndBlocker reads from v3 + `GenesisDoc.Validators`. The risk is not the no-op but that the post-replay state differs from gnoland1's halt-state by the union of the no-oped v2 events plus any absent valoper profiles and boards2 history. The code is correct; the gap is operator-facing. Fix: ship a one-page "test-13 vs gnoland1 halt-state diff" with the handoff so the next operator can audit it, beyond the category counts that `fork inspect` prints. Worth tracking as a follow-up, not a code change.
  </details>

- **[no generate→inspect round-trip test]** [`contribs/gnogenesis/internal/fork/inspect_test.go`](https://github.com/gnolang/gno/blob/f45cc5c88/contribs/gnogenesis/internal/fork/inspect_test.go) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/inspect_test.go) — the report-shape unit tests are good but no test runs a real `gnogenesis fork generate` output through `inspect` and checks the per-`Source` counts against what was assembled.
  <details><summary>details</summary>

  This regression fires only when the `Source` annotation order changes silently, e.g. someone moves `annotateSource(_, SourceBase)` past the `applyPatchTxs` call. A small test that runs `generate` against a tiny fixture genesis with one patched tx and one migration tx, then `inspect` on the result, locks the contract. Fix: add a generate→inspect round-trip test.
  </details>

## Nits

- [`misc/deployments/test13.gno.land/gen-genesis.sh:1`](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L1) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L1) — the 1,759-line single bash file is hard to navigate; a split into `lib/phase1.sh` + `lib/phase2.sh` + `lib/txn-loader.sh` would fit the entry point on one screen. The author consolidated everything in one file deliberately; flagging the readability cost only.
- [`contribs/gnogenesis/internal/fork/inspect.go:97`](https://github.com/gnolang/gno/blob/f45cc5c88/contribs/gnogenesis/internal/fork/inspect.go#L97) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/inspect.go#L97) — a `Source` value not in `groups` is silently bucketed as "unannotated"; a warning would stop a typo in a future `Source` constant from disappearing into that bin. Confirmed behaviorally: the four `Source*` constants in `types.go:202-205` are the only known buckets.
- [`misc/deployments/test13.gno.land/transactions/patched/validator-noops/h1008282/meta.json`](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/transactions/patched/validator-noops/h1008282/meta.json) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/transactions/patched/validator-noops/h1008282/meta.json) — `HALT_HEIGHT` is now `1485629` ([gen-genesis.sh:97](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L97) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L97)); a patch at h1008282 is a valid pre-halt tx, the old-halt-height-looking directory name is the only confusing part. Not actionable.

## Missing Tests

- **[no in-tree `--patch-txs` end-to-end test]** [`contribs/gnogenesis/internal/fork/patch_txs_test.go`](https://github.com/gnolang/gno/blob/f45cc5c88/contribs/gnogenesis/internal/fork/patch_txs_test.go) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/patch_txs_test.go) — unit tests cover the match/swap logic, but nothing runs `fork generate --patch-txs` against a real-ish base genesis + historical stream and asserts the output carries `Source=patched`, `OriginalTx` set, and replays cleanly under `fork test`.
  <details><summary>details</summary>

  The full path is exercised only by the out-of-tree `gen-genesis.sh` audit. An in-tree test that builds a small genesis, patches one tx, runs `fork generate` + `fork test`, and asserts replay-success would catch a regression in the generate→deliverGenesisTx→ante chain. Fix: add `TestForkGenerate_PatchTxsRoundTrip`.
  </details>

- **[no negative test pinning the SkipSigVerificationKey boundary]** [`tm2/pkg/sdk/auth/ante_test.go:542-580`](https://github.com/gnolang/gno/blob/f45cc5c88/tm2/pkg/sdk/auth/ante_test.go#L542-L580) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/auth/ante_test.go#L542-L580) — `TestAnteHandlerSkipSigVerificationKey` asserts the positive case (key set → sig bypassed) but no test pins that no production ABCI path can thread a `ContextFn` setting the key.
  <details><summary>details</summary>

  The boundary is structural: `BaseApp.DeliverTx` → `getContextForTx` → `runTx` never threads a ctxFn, and the only non-test caller of `BaseApp.Deliver(tx, ctxFns...)` is `deliverGenesisTx`. The current code accepts any caller that flips the key. Fix: at minimum, document the threat model on the `SkipSigVerificationKey` doc; at best, defense-in-depth gate the bypass to refuse when the block height is non-zero or the source is not patched.
  </details>

## Suggestions

- [`contribs/gnogenesis/internal/fork/test.go:338-353`](https://github.com/gnolang/gno/blob/f45cc5c88/contribs/gnogenesis/internal/fork/test.go#L338-L353) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/test.go#L338-L353) — `countDeliverableTxs` walks every tx on each call; not hot, but a one-line comment that it runs once per test invocation would settle the O(N).

## Open questions

- Cross-validator reproducibility: has anyone other than the author run `gen-genesis.sh` end-to-end against the cached gnoland1 archive and reproduced the locked final genesis sha (`56f56e13…` in `CHECKSUMS_DATA`)? It is the chain's only protection against a bad genesis at boot. Not posted: this is a coordination question for the test-13 stakeholders, not a code change.
- boards2-cascade creator divergence: post-replay, the GovDAO T1 multisig is on record as creator of 2 boards and inviter of N members; on gnoland1 the creator was the original signer. Acceptable to stakeholders, and does any post-replay realm code key on the creator address? Not posted: a stakeholder decision, not a code defect.
- `--patch-txs` long-term home: keep patches per-chain or upstream the API-drift fixes into the examples directory? Not posted: deferred-scope design.
- Round 1's Q4 (audit strictness) is answered by the code: the audit uses `--skip-failing-genesis-txs` only to collect the full failure list, then parses the verbose log and `exit 1` on any failure ([gen-genesis.sh:1686-1695](https://github.com/gnolang/gno/blob/f45cc5c88/misc/deployments/test13.gno.land/gen-genesis.sh#L1686-L1695) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L1686-L1695)). Zero failures are already required. Dropped.
- Round 1's stale review thread on `misc/govdao-scripts/add-validator-v3.sh`: the file exists on master but is not in this PR's diff (gov-dao scripts were extracted to #5658), so the thread is moot for this PR. Not a code finding.
