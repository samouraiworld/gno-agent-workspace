# PR #5589: chain/test13-rc6 (tier 3 — rc4 + rc5 + rc6)

**URL:** https://github.com/gnolang/gno/pull/5589
**Author:** aeddi | **Base:** chain/gnoland1 | **Files (full PR):** 300+ (truncated) | **+24,801 -**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7 (1M context)
**Scope of this file:** rc4 + rc5 + rc6 — audit/resilience tooling: `assert-migrations`, `audit-balances`, `state-diff`, `verify-txs-jsonl`, `verify-reproducibility`, `compare-gas-modes`, `audit-realm-imports`; valset-ops scripts (`rm-validator`, `change-power`, `batch-change`); the `mempackage` nil-skip fix; the rc6 ADR.

## Summary

This tier hardens the launch path with audit primitives and resilience patches. Three groups:

- **Audit scripts** (rc4 + rc6, ~1300 lines):
  - `assert-migrations.sh` — positive checks that every migration step (01-08) actually landed.
  - `audit-balances.sh` — diff per-signer ugnot between source-chain at `HALT_HEIGHT` and the replay node.
  - `state-diff.sh` — Render-output diff for a configurable realm set.
  - `verify-txs-jsonl.sh` — cardinality (`total_txs`) + spot-check vs the source RPC.
  - `verify-reproducibility.sh` — two independent local builds, SHA256 must match.
  - `compare-gas-modes.sh` (rc6) — A/B `strict` vs `source` gas-replay-mode smoketest, prints the failure-count delta.
  - `audit-realm-imports.sh` (rc6) — scan every addpkg tx (genesis + historical) for `import` paths that no longer resolve against the live `gnovm/stdlibs` + `examples/` tree.
- **Operator scripts** (rc5) — `rm-validator.sh`, `change-power.sh` (atomic remove+add to work around v3's lack of in-place update), `batch-change.sh` (multi-op atomic batch with input validation).
- **Resilience patches** (rc5):
  - `gnovm/pkg/gnolang/machine.go` — defensive nil-skip in `PreprocessAllFilesAndSaveBlockNodes` for `IterMemPackage` returning `nil` (root-cause: split-write between pebble path index + IAVL body in `defaultStore.AddMemPackage` is not crash-consistent).
  - `assert-migrations.sh` (`4246860cc`) — accept empty string as "no pending update" because v3's `init()` doesn't seed `new_updates_available`.

The rc6 ADR (`gno.land/adr/pr5589_test13_hardfork_launch.md`) ties the design choices together and documents the launch verification ritual (§5).

## Test Results

- **`contribs/tx-archive/backup/...`** — PASS.
- **`contribs/gnogenesis/internal/fork/` non-replay tests** — PASS.
- **`gno.land/pkg/sdk/vm/...`** — PASS.
- **`misc/hf-glue/fixvalidator/`** — PASS.
- **No tests touch `gnovm/pkg/gnolang/machine.go`'s nil-skip branch** — confirmed by `grep`. Listed under Missing Tests.
- **No CI runs the bash audit scripts** — they're operator tooling, not unit tests. Their correctness is implicitly validated by the launch-prep runs against gnoland1, but a CI smoke (e.g. `verify-reproducibility.sh` against a 100-block synthetic source) would lock in the "make verify-X" contract listed in ADR §5.
- **Edge-case tests:** skipped per scope.

## Critical (must fix)

- [ ] **None new in this tier.** The chunked-fetch sort bug (tier 1) is still the launch-blocker.

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/machine.go:198-212` — **Defensive nil-skip in `PreprocessAllFilesAndSaveBlockNodes`.** The fix itself is correct and the commit message clearly identifies the root cause (`defaultStore.AddMemPackage` writes path index → IAVL body in two separate `Set` calls; SIGKILL between the two leaves the index pointing at a body that doesn't exist; `IterMemPackage` then yields a nil). But two follow-ups are needed:
  1. **No identifying info in the warning.** The branch logs `WARNING: IterMemPackage returned nil ...` with no path. An operator who sees this in the boot log can't tell which package's body is missing — needs to grep IAVL/pebble manually or wait for a downstream `import "..."` failure. Either change `IterMemPackage`'s contract to yield `(path, *MemPackage)` (so the warning can name the orphan) or have the iterator log the orphan internally before returning nil.
  2. **Proper fix is upstream-only.** Atomic index+body writes in `defaultStore.AddMemPackage` (pebble batch spanning both substores, or body-first + counter-last ordering crash-consistent by construction). The commit message correctly defers this; please file an issue if not already (recommend a follow-up `fix(gnovm): atomic index+body writes in defaultStore.AddMemPackage` so this tier's defensive skip becomes belt-and-suspenders, not load-bearing).

- [ ] `misc/hf-glue/scripts/audit-balances.sh:87-112` — **`query_balance_ugnot` swallows RPC errors and returns 0.** `gnokey query 2>/dev/null | awk ...` collapses unreachable-RPC, address-not-found, and "actual zero balance" into the same return value. If the source RPC drops mid-audit, every subsequent row reports `src=0, replay=X, delta=-X` — the report shows massive false divergence. Wrap the query in a status check (`if ! gnokey query ...; then echo "RPC error: $addr" >&2; return 1; fi`) and either skip-with-flag or fail the script.

- [ ] `misc/hf-glue/scripts/audit-balances.sh:42-43, 135-141` — **Default `SIGNER_LIMIT=200` audits the most-active signers, not the largest balances.** A 1-tx whale would be skipped. ADR §4 documents 13 known divergent accounts, but those came from a manual investigation of replay failures — re-deriving them from this script's defaults wouldn't surface them. Document loud-clearly that `SIGNER_LIMIT=0` is the production-launch mode (currently only mentioned at the bottom of the report). Also: the script samples by activity but the report claims "every diverged signer" — tighten the wording.

- [ ] `misc/hf-glue/scripts/state-diff.sh:198` — **Exit-code overflow.** `exit "$failed"` returns the failure count modulo 256. With the default 8 realms this is fine, but a `REALMS=$(cat huge_list)` invocation could produce 256+ failures and silently exit 0 (CI passes despite divergence). Replace with `exit $((failed > 0 ? 1 : 0))`. Same shape in `assert-migrations.sh:202`.

- [ ] `misc/hf-glue/scripts/assert-migrations.sh:186-193` — **Brittle render-string parsing for T1 tier size.** `grep -oE 'Tier T1 contains [0-9]+ members'` depends on the exact phrasing of `r/gov/dao/v3/memberstore`'s Render output. A future render touch ("Tier T1 has 1 member" → check breaks silently). Either (a) add a memberstore qeval helper (`MemberCount(T1)`) and call it instead, or (b) snapshot-pin the rendered string in a CI test that fires when the render output drifts.

- [ ] `misc/hf-glue/scripts/audit-balances.sh:118` — **Multi-sig txs only count `signer_info[0]`.** Comment at L30 acknowledges this. For test-13 there are very few multi-sig txs on gnoland1, but any multi-sig user whose first signer is fine but second-signer balance diverged would be missed. Worth iterating `signer_info[]` even if most txs have a single entry.

## Nits

- [ ] `misc/deployments/test13.gno.land/govdao-scripts/{add,rm,change-power}-validator.sh` — `set -eo pipefail` lacks `-u`. `batch-change.sh` is the gold standard in this directory (validates address regex, pubkey regex, power numeric). The other three should match: add `-u`, validate `^g1[0-9a-z]+$` / `^gpub1[0-9a-z]+$`, validate POWER as numeric in `add-validator.sh`.
- [ ] `misc/hf-glue/scripts/state-diff.sh:101-112` — normalize regexes are aggressive. `s|height=[0-9]*|height=<H>|g` could mask substantive content (e.g. a realm rendering "set the threshold to height=12345 by 2026-04-30"). Low risk for the current 8-realm default set; worth a comment noting the assumption.
- [ ] `misc/hf-glue/scripts/audit-realm-imports.sh:117-124` — block-import parser doesn't handle single-line `import ("x"; "y")`. Acknowledged in the comment. `gnofmt` doesn't produce that style, so 0% false-negatives in practice; OK to leave.
- [ ] `misc/hf-glue/scripts/verify-reproducibility.sh:35-37` — script wipes `$OUT_A` / `$OUT_B` but uses the same `$TMPDIR`-rooted ephemeral directories internally (via `migrations/build.sh`'s `mktemp -d`). Cross-machine reproducibility could break if any tmp path leaks into the genesis bytes; this script wouldn't catch that because both runs share the same machine's `$TMPDIR` base. Worth running once with `TMPDIR=/tmp/A` and once with `TMPDIR=/tmp/B` to surface that class of bug.
- [ ] `misc/hf-glue/scripts/compare-gas-modes.sh:69` — `extract_failures` regex `Failures:[[:space:]]+[0-9]+` depends on `gnogenesis fork test` output format. Low priority; pinning a CI test on this would be overkill.
- [ ] `misc/hf-glue/scripts/verify-txs-jsonl.sh:43` — `SPOT_SEED="${SPOT_SEED:-$RANDOM}"` defaults to a non-reproducible seed. Already printed at L65 — operator can capture-and-replay — but a more useful default would be `HALT_HEIGHT` (deterministic across runs against the same target).
- [ ] `assert-migrations.sh:34` — hardcoded `EXPECTED_T1=g1aeddlft...`. Sane default for the launch path, but should the Makefile/wrapper pass it explicitly so an alternative T1 can be tested without env-var tweaks? Worth a one-liner in the docstring on overriding it for the rc cluster smoke.

## Missing Tests

- [ ] **No unit test for the `IterMemPackage` nil-skip path** in `gnovm/pkg/gnolang/`. A test injecting a mock store that returns nil for one path + valid mpkgs for others would lock in the "boot continues, warning logged" contract. Currently the only validation is "it boots after a manual SIGKILL" — not reproducible in CI.
- [ ] **No CI gate on `verify-reproducibility.sh`.** A 100-block synthetic source genesis run twice in CI, asserting SHA match, would catch any future patch that introduces nondeterminism. Cheap to add; protects the launch-gate ritual from rotting silently.
- [ ] **No shellcheck on `misc/hf-glue/scripts/`.** Author has `# shellcheck disable=SC2206` at one point — already familiar with the tool. A `make lint` target invoking shellcheck on `scripts/*.sh` and `deployments/test13.gno.land/govdao-scripts/*.sh` is a 10-line addition that catches a meaningful set of bash regressions.
- [ ] **No test for `audit-realm-imports.sh`'s import parser.** Add a stub fixture (a tiny synthetic addpkg tx with all three import styles + a deliberately-broken import) and assert the count of detected/missing edges matches expected. Shields the parser from future cosmetic refactors.

## Suggestions

- **Upstream fix for `defaultStore.AddMemPackage`.** The mempackage nil-skip in this tier is symptomatic; the proper fix is atomic index+body writes (pebble batch or body-first ordering). File a follow-up issue and add a `// FIXME(upstream)` link in `machine.go:198`.
- **`audit-balances.sh`: include funded-but-non-signing accounts.** Current scope is `txs.jsonl` signers. Mainnet has accounts that received funds but never signed — they wouldn't appear here. Easy extension: union signers ∪ `app_state.balances[].address` from the source genesis, audit both, label which set each came from in the report.
- **`compare-gas-modes.sh`: dump per-error-class counts.** Currently the report has overall counts only. Adding `grep -c InsufficientFundsError`/`grep -c gasoverflow` etc. on each log would help the next operator drill into failure modes without re-running the smoketest.
- **`state-diff.sh`: companion `qeval`-based check** for realms whose Render hides state in maps. The script's own caveat at L18-22 names this gap; a small per-realm `expr` config (e.g. `r/sys/validators/v2.GetValidators()` length) would close it.
- **`verify-reproducibility.sh`: single-host TMPDIR perturbation.** Already noted under nits; would catch a real class of nondeterminism without needing two physical machines.
- **`assert-migrations.sh`: assert step 08's value is non-empty AND well-formed.** Currently checks for exact `gno.land/r/sys/validators/v3` — ties the assertion to the literal string. If a test ever needs to point at v4, the assertion has to change. Either parametrise on `EXPECTED_VALSET_REALM` or assert the value is a valid realm path (`gno.IsRealmPath` check).

## Questions for Author

1. The `PreprocessAllFilesAndSaveBlockNodes` nil-skip — is there a tracking issue for the upstream `AddMemPackage` atomic-write fix? If yes, add the link to the comment in `machine.go`. If not, OK to file?
2. `audit-balances.sh` defaults to `SIGNER_LIMIT=200`. Was the production launch-gating run in fact at `SIGNER_LIMIT=0` (full audit)? If yes, document the recommended invocation in `misc/hf-glue/Makefile`'s `audit-balances` target.
3. `assert-migrations.sh`'s `EXPECTED_T1` defaults to `g1aeddlft...`. Is this the post-rotation T1 for test-13's launch, or the launch playbook still calls a different address? Worth pinning in the ADR §4 if not already.
4. `verify-reproducibility.sh` — has it been run with deliberately-perturbed `$TMPDIR`? Any history of cross-machine SHA drift, or has every report so far traced back to "stale `$OUT/`" as the comment claims?
5. The `r/sys/validators/v2` step-01 partial-apply behaviour (one missing addr panics the whole batch) is documented as known in both ADR §4 and `assert-migrations.sh:162-179`. Is there a follow-up to make `NewPropRequest` either (a) tolerate-missing or (b) at minimum emit the index of the failing entry so an operator can prune-and-retry? Tracking question — not blocking this rc.

## Verdict

NEEDS DISCUSSION — same as tier 1: the chunked-fetch sort is the only blocker. The audit/resilience layer in this tier is the strongest part of the PR. The mempackage nil-skip is correctly scoped as defensive; the audit scripts together form a credible launch-gate as long as they're (a) all run with `SIGNER_LIMIT=0` / full coverage and (b) the chunked-fetch fix lands so `verify-txs-jsonl` and `verify-reproducibility` are running over a correctly-ordered txs.jsonl.
