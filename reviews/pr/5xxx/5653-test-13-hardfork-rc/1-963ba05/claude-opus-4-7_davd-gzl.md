# PR #5653: feat: test-13 hardfork release candidate

URL: https://github.com/gnolang/gno/pull/5653
Author: aeddi | Base: master | Files: 87 | +6343 -68
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `963ba05` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5653 963ba05`

**Verdict: NEEDS DISCUSSION** — design is sound and the `SkipSigVerificationKey` boundary holds, but two failing CI jobs (`gno-checks/lint`, one `main/test`), 13 commits tagged `TMP` whose intent is unstated, and operational hot spots (per-tx `--patch-txs` shell pipeline, large `gen-genesis.sh`, sequence semantics for caller-swap patches) need author clarification before merge.

## Summary

End-to-end tooling that produces a test-13 hardfork genesis from gnoland1's state. Two phases: (1) a fresh base genesis with the test-13 valset, faucets, and a single-use `r/test13/rotate` realm; (2) replay of gnoland1's 1.4M+ historical txs on top, with 27 per-tx caller/body patches that compensate for API drift (post-#5669) and an admin-list narrowing in `r/gnoland/boards2/v1`. The change surface includes a new `gnogenesis fork` subpackage (`patch_txs`, `inspect`, `annotated_tx`, `migration_load`, `source_annotation`, valoper-seed extensions), a `SkipSigVerificationKey` ante bypass restricted to `Source=patched` replay, three new `GnoTxMetadata` provenance fields, and a 1,757-line bash pipeline (`gen-genesis.sh`).

The novel security boundary is `SkipSigVerificationKey` — set only on a per-tx `ctxFn` inside [`deliverGenesisTx`](https://github.com/gnolang/gno/blob/963ba05/gno.land/pkg/gnoland/app.go#L740-L742) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L740-L742) when `metadata.Source == SourcePatched`. `BaseApp.DeliverTx` (the production block path) never threads a `ContextFn`, so the key is structurally unreachable outside InitChain replay — verified across [`tm2/pkg/sdk/helpers.go:61-77`](https://github.com/gnolang/gno/blob/963ba05/tm2/pkg/sdk/helpers.go#L61-L77) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/helpers.go#L61-L77) and the InitChainer call graph.

## Glossary

- `SkipSigVerificationKey` — context-value sentinel that makes the auth ante skip Phase 3 (pubkey set + sig verify + sequence increment) for that tx.
- `SourcePatched` — provenance tag set by `applyPatchTxs` after it matches and rewrites a historical tx; the only `Source` value that triggers `SkipSigVerificationKey`.
- `keyOf(meta)` — `(block_height, signer_info[0].address, signer_info[0].sequence)` triple used to match patch entries against the historical stream.
- `txn_dir_to_jsonl` — shell helper in `gen-genesis.sh` that converts per-tx `transactions/<bucket>/h<height>/{meta.json,body.gno|pkg/}` directories into AnnotatedTx jsonl lines.
- `AnnotatedTx` — `{Tx, Metadata, Reason}` jsonl schema for `--patch-txs` and `--migration-tx` inputs; `Reason` flows into `metadata.Note` at consume time.
- TMP — author tag on 13 commit subjects in this PR; intent (squash? extract?) is not documented.

## Fix

Before: there was no hardfork-genesis tooling beyond `gnogenesis fork generate`; per-tx body rewriting, provenance tagging, and the post-replay-friendly sig-bypass were absent, and operators inspecting a hardfork genesis had no way to tell which tx came from where.

After: `gnogenesis fork` grows `patch_txs` / `migration_load` / `inspect`, `GnoTxMetadata` grows `Source`/`Note`/`OriginalTx`, the ante grows a single sentinel-gated bypass, and the test-13 deployment ships a self-contained `gen-genesis.sh` plus a per-tx-directory layout under `transactions/{base,patched,migration}` that the script converts into AnnotatedTx jsonl on the fly.

The load-bearing invariant is the binding `SkipSigVerificationKey ⇔ Source=patched`, set in exactly one place ([`app.go:740-742`](https://github.com/gnolang/gno/blob/963ba05/gno.land/pkg/gnoland/app.go#L740-L742) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L740-L742)) and only on the per-tx ctxFn that `BaseApp.Deliver` consumes in InitChain.

## Benchmarks / Numbers

| Surface | Count | Notes |
|---|---|---|
| New files in `contribs/gnogenesis/internal/fork/` | 10 (5 prod + 5 test) | annotated_tx, inspect, migration_load, patch_txs, source_annotation + tests |
| TMP-tagged commits | 13 / 37 | Author has not explained the TMP convention in the PR body |
| Patch entries under `transactions/patched/` | 27 across 8 buckets | validator-noops×17, boards2-cascade×9, unrestrict×3, others×... |
| `gen-genesis.sh` length | 1,757 lines | Replaces the prior `build.sh` + 3 phase scripts + `lib/common.sh` + CHECKSUMS |
| CI status | 3 fail / pass total | `gno-checks/lint`, `main/test` (the longer one), `main/test` x1 — see [CI checks](https://github.com/gnolang/gno/blob/963ba05/.review-context/ci-checks.txt) · [↗](../../../../../.worktrees/gno-review-5653/.review-context/ci-checks.txt) |

## Critical (must fix)

- **[red CI on RC]** [ci-checks.txt](https://github.com/gnolang/gno/blob/963ba05/.review-context/ci-checks.txt) · [↗](../../../../../.worktrees/gno-review-5653/.review-context/ci-checks.txt) — `gno-checks / lint` (1m22s) and `main / test` (21m54s) are failing on HEAD; the green `main / test` is a separate matrix entry.
  <details><summary>details</summary>

  The PR is titled "release candidate" but has red required CI. The other `main / test` matrix entry is green and the rest of the pipeline (gnogenesis, build, docker, scenarios, codecov, stdlibs, e2e) is green, so the failures are localized — but for an RC labeled `a/everyone` the bar is "green or explained". The bot-comment "force green CI check" override is checked, so the failures will not block bot-merge gates; that makes the human review the only check on whether the failures are spurious or load-bearing. Fix: triage both failing jobs, then either fix or document why the failure is acceptable for this PR. The override should not be a substitute for triage on an RC of this size.
  </details>

## Warnings (should fix)

- **[explain TMP commits]** [pr-view.json#commits](https://github.com/gnolang/gno/blob/963ba05/.review-context/pr-view.json) · [↗](../../../../../.worktrees/gno-review-5653/.review-context/pr-view.json) — 13 of 37 commits are subject-tagged `TMP` (`SkipSigVerificationKey`, AnnotatedTx, `--patch-txs`, fork inspect, all three provenance fields, valoper-seed --caller, fork test guards, InitChainer error logging, …) with no explanation of the convention.
  <details><summary>details</summary>

  TMP normally means "to be squashed" or "to be extracted to a separate PR" (the precedent here is the gov-dao scripts that already moved to #5658). These 13 commits are not WIP — they implement production code paths (the sig-verification bypass, the provenance schema, the patch-txs pipeline) that ship in this PR. If they are intended to land as-is, retitle them; if they will be extracted, list which ones in the PR body and call out the dependency order. The reviewer should not have to guess whether `TMP add --patch-txs flag` is permanent. Fix: edit the PR body with a short table (commit subject → final disposition: squash into X / extract to PR #Y / keep as-is).
  </details>

- **[shell-pipeline patch generation is fragile]** [`misc/deployments/test13.gno.land/gen-genesis.sh:682-825`](https://github.com/gnolang/gno/blob/963ba05/misc/deployments/test13.gno.land/gen-genesis.sh#L682-L825) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L682-L825) — patch jsonl is assembled by ~143 lines of bash + jq that inlines `.gno` bodies into `tx.msg[0].package.files[0].body` and `pkg_dir` walks into `files`, with no validation that the resulting line decodes as AnnotatedTx until `gnogenesis fork generate` rejects it.
  <details><summary>details</summary>

  The conversion logic lives in shell because the per-tx directory layout was chosen for diffability, but every operator hand-editing a `transactions/patched/<bucket>/h<height>/meta.json` is one mis-escaped quote away from a malformed jsonl that fails opaquely at chain replay. The PR comment thread already shows moul asking why a Go binary (`emit-migration-txs`) was used and getting a "use gnogenesis subcommand" push-back — the same argument applies to `txn_dir_to_jsonl`: a `gnogenesis fork txn-dir-to-jsonl <dir>` subcommand would let you typecheck the AnnotatedTx struct at emit time and ship test coverage rather than relying on integration-level shell smoke. Fix: track this as a follow-up issue at minimum; for the RC, add a `gnogenesis fork verify-patches <dir>` step that decodes each emitted line before `fork generate` runs.
  </details>

- **[sequence semantics of caller-swap patches]** [`transactions/patched/boards2-cascade/h126810/meta.json`](https://github.com/gnolang/gno/blob/963ba05/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/transactions/patched/boards2-cascade/h126810/meta.json) + [`tm2/pkg/sdk/auth/ante.go:189-281`](https://github.com/gnolang/gno/blob/963ba05/tm2/pkg/sdk/auth/ante.go#L189-L281) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/auth/ante.go#L189-L281) — boards2-cascade patches rewrite `MsgCall.Caller` from `g16jpf0…` to the GovDAO multisig but keep `metadata.SignerInfo` pointing at `g16jpf0…` (needed for `keyOf` matching); the ante then force-sets g16jpf0's seq via [`app.go:777-792`](https://github.com/gnolang/gno/blob/963ba05/gno.land/pkg/gnoland/app.go#L777-L792) · [↗](../../../../../.worktrees/gno-review-5653/gno.land/pkg/gnoland/app.go#L777-L792), and the `SkipSigVerificationKey` short-circuit skips the seq-increment for the new caller (`g1rp7…`), so the multisig's seq does not advance for these 9 patched txs.
  <details><summary>details</summary>

  Functionally this is OK on a fresh chain — post-replay seq for `g1rp7…` = count of non-patched txs by `g1rp7…` during replay, and any further txs from that account use the on-chain value. But it deserves an explicit invariant comment somewhere central (probably in `applyPatchTxs` or next to `SkipSigVerificationKey`) so the next person reading this doesn't have to reconstruct the reasoning from three files. The current comment block on `SkipSigVerificationKey` describes the "patched body invalidates sig" angle but is silent on the seq-skipping side effect. Fix: add a 3-line "side effects" paragraph to the `SkipSigVerificationKey` doc covering (a) seq not incremented for the caller, (b) pubkey not set on first-use, (c) `validateSignerInfo` force-set still runs against the original SignerInfo address (not the patched caller).
  </details>

- **[unaddressed review thread]** [moul on misc/govdao-scripts/add-validator-v3.sh](https://github.com/gnolang/gno/pull/5653#discussion_r3228074228) — "I've merged the improved version of the GovDAO scripts. Please use the new format: #5426". The govdao scripts were moved into this PR's earlier commits via #5426 and the `add-validator-v3.sh` file no longer appears in HEAD, so the thread should be marked resolved.
  <details><summary>details</summary>

  Confirmed: the file is absent from the current diff (`pr-view.json` does not list it). Either resolve the thread on GitHub or add a one-line note that the rename has already happened. Not a code issue, just a hygiene loose end.
  </details>

- **[validator-noops semantics]** [`transactions/patched/validator-noops/`](https://github.com/gnolang/gno/blob/963ba05/misc/deployments/test13.gno.land/transactions/patched/validator-noops/) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/transactions/patched/validator-noops/) — 17 patches replace v2 valset proposal bodies with `func main(cur realm) {}`; the patches preserve manfred's fee deduction "faithfully" but the bodies are no-ops.
  <details><summary>details</summary>

  This is the documented design — v2 is unseeded on test-13 by intent because EndBlocker reads from v3 + `GenesisDoc.Validators`. The risk is not the no-op itself but the principle: the chain's post-replay state silently differs from gnoland1's halt-state by the union of (5 absent valoper profiles, 2 absent boards2 boards + their member history, 17 v2 add/remove valset events). The PR body lists this honestly. Fix: nothing required for the code, but the operator handoff should include a single-page "test-13 vs gnoland1 halt-state diff" that the next person picking up the chain can audit against, beyond the `fork inspect` output (which only counts categories, not state effects). Worth tracking as a follow-up.
  </details>

- **[no `gnogenesis fork inspect` invariant test]** [`contribs/gnogenesis/internal/fork/inspect_test.go`](https://github.com/gnolang/gno/blob/963ba05/contribs/gnogenesis/internal/fork/inspect_test.go) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/inspect_test.go) — the report-shape tests are good but there is no end-to-end test that a real `gnogenesis fork generate` output, fed into `inspect`, agrees with the source counts emitted at assembly time.
  <details><summary>details</summary>

  This is the kind of regression that only fires when the `Source` annotation order changes silently (e.g. someone moves `annotateSource(_, SourceBase)` past the `applyPatchTxs` call by mistake). A 30-line test that runs `execGenerate` against a tiny fixture genesis with one patched tx and one migration tx, then runs `inspectReport` on the result, would lock in the contract. Fix: add a generate→inspect round-trip test.
  </details>

## Nits

- [`misc/deployments/test13.gno.land/gen-genesis.sh:1-80`](https://github.com/gnolang/gno/blob/963ba05/misc/deployments/test13.gno.land/gen-genesis.sh#L1-L80) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/gen-genesis.sh#L1-L80) — 1,757-line bash file is hard to navigate; consider splitting into `lib/phase1.sh` + `lib/phase2.sh` + `lib/txn-loader.sh` so the entry point fits in one screen. The author's earlier refactor consolidated everything in one file deliberately — fair tradeoff, just calling out the readability cost.
- [`contribs/gnogenesis/internal/fork/inspect.go:97`](https://github.com/gnolang/gno/blob/963ba05/contribs/gnogenesis/internal/fork/inspect.go#L97) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/inspect.go#L97) — a `Source` value not in `groups` is silently bucketed as "unannotated". Consider warning instead so a typo in a future `Source` constant doesn't disappear into the unannotated bin.
- [`misc/deployments/test13.gno.land/transactions/patched/validator-noops/h1008282/meta.json`](https://github.com/gnolang/gno/blob/963ba05/misc/deployments/test13.gno.land/transactions/patched/validator-noops/h1008282/meta.json) · [↗](../../../../../.worktrees/gno-review-5653/misc/deployments/test13.gno.land/transactions/patched/validator-noops/h1008282/meta.json) — height 1008282 is documented in the PR body as the original `HALT_HEIGHT`, but the current `HALT_HEIGHT=1485629`. The presence of a patch at the old halt height is fine (it's still a pre-halt tx) but the bucket name and the `h<height>` directory naming make it slightly confusing. Not actionable.

## Missing Tests

- **[no integration test for `--patch-txs` end-to-end]** [`contribs/gnogenesis/internal/fork/patch_txs_test.go`](https://github.com/gnolang/gno/blob/963ba05/contribs/gnogenesis/internal/fork/patch_txs_test.go) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/patch_txs_test.go) — unit tests cover the match/swap logic but there is no test that runs `gnogenesis fork generate --patch-txs` against a real-ish base genesis + historical stream and confirms the output has `Source=patched`, `OriginalTx` set, and replays cleanly under `gnogenesis fork test`.
  <details><summary>details</summary>

  The end-to-end path is exercised manually via the test-13 `gen-genesis.sh` audit step, but that's an out-of-tree script. A txtar or in-tree test that builds a 5-tx genesis, patches one tx, runs `fork generate` + `fork test`, and asserts replay-success would catch a regression in the generate→deliverGenesisTx→ante chain. Fix: add `TestForkGenerate_PatchTxsRoundTrip`.
  </details>

- **[no test that `SkipSigVerificationKey` cannot escape genesis-replay context]** [`tm2/pkg/sdk/auth/ante_test.go:537-580`](https://github.com/gnolang/gno/blob/963ba05/tm2/pkg/sdk/auth/ante_test.go#L537-L580) · [↗](../../../../../.worktrees/gno-review-5653/tm2/pkg/sdk/auth/ante_test.go#L537-L580) — `TestAnteHandlerSkipSigVerificationKey` asserts the positive case (key set → sig bypassed) but no negative test pins the invariant that no production ABCI path threads a `ContextFn` that could leak the key.
  <details><summary>details</summary>

  The boundary is structural (BaseApp.DeliverTx → getContextForTx → runTx, no ctxFn) but a regression test that pulls the call graph for `BaseApp.DeliverTx` and asserts no caller sets `SkipSigVerificationKey{}` is fragile. A weaker but cheaper check: a unit test that constructs a non-genesis ctx with `SkipSigVerificationKey{}` set, runs the ante, AND asserts an alarm sounds (e.g. wrap the bypass with a `panic("SkipSigVerificationKey set outside InitChain — refusing to bypass")` when `BlockHeader().Height != 0` and `Source != patched`). The current code accepts any caller that flips the key. Fix: at minimum document the threat model in the `SkipSigVerificationKey` doc — at best, gate the bypass on a corroborating context value that only `deliverGenesisTx` sets.
  </details>

## Suggestions

- [`contribs/gnogenesis/internal/fork/test.go:344-353`](https://github.com/gnolang/gno/blob/963ba05/contribs/gnogenesis/internal/fork/test.go#L344-L353) · [↗](../../../../../.worktrees/gno-review-5653/contribs/gnogenesis/internal/fork/test.go#L344-L353) — `countDeliverableTxs` walks every tx every time; not hot but worth a comment that it's also called once per test run, so the O(N) doesn't matter.

## Questions for Author

- Why are 13 commits subject-tagged `TMP`? Are they intended to be squashed, extracted into separate PRs (like the gov-dao scripts moved into #5658), or kept as-is in the final history? If the latter, can the prefix be dropped before merge?
- Per the `boards2-cascade` patches: post-replay, the GovDAO T1 multisig (`g1rp7…`) is on record as the creator of 2 boards and the inviter of N members on test-13. On gnoland1 the creator was `g16jpf0…`. Is this divergence acceptable to the test-13 stakeholders (i.e. the multisig members), and does any post-replay realm code key on the creator address?
- For the `--patch-txs` mechanism: is the long-term plan to keep patches per-chain (every hardfork ships its own `transactions/patched/`) or to upstream the API-drift patches into the gnoland1 examples directory? The `boards2-permissions` patch already mirrors a migration the official boards2 source did — the duplication may be self-defeating.
- The `fork test` `--skip-failing-genesis-txs` flag exists to match production node behavior, but `gen-genesis.sh:1684-1693` then parses the verbose log and rejects on any failure unless suppressed. Why not require `0` failures by default in the audit step (i.e. drop `--skip-failing-genesis-txs` from the audit invocation) so the audit becomes truly strict?
- `SkipSigVerificationKey` is keyed only on `Source=patched`. If a future migration tx (Source=migration) needs the same sig bypass, what's the intended extension point — broaden the predicate in `deliverGenesisTx`, or add a per-source flag?
- Has anyone other than the author run `gen-genesis.sh` end-to-end against the cached gnoland1 archive and reproduced the expected SHAs from the PR description (`66ad49b1…` for base, `989f7758…` for the final)? Cross-validator reproducibility is the chain's only protection against a bad genesis at boot.
