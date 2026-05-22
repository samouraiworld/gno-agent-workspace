# PR #5589: chain/test13-rc6 (tier 2 — rc2 + rc3)

**URL:** https://github.com/gnolang/gno/pull/5589
**Author:** aeddi | **Base:** chain/gnoland1 | **Files (full PR):** 300+ (truncated) | **+24,801 -**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7 (1M context)
**Scope of this file:** rc2 + rc3 — `r/sys/validators/v3` deployment + sysnames disable/restore wrap + multi-validator genesis support + eager-eval fix + EndBlocker wiring (`vm:p:valset_realm_path`)

## Summary

This tier wires the `r/sys/validators/v3` flow into the test-13 hardfork. v3 (PR #5485) is cherry-picked unchanged in `256454169`; the rest of the tier is glue:

- **`095cee651`** — adds migration steps 05/07 (templates `05_disable_sysnames.gno.tmpl` + `07_restore_sysnames.gno.tmpl`) and the `migrations/build.sh` block that wraps the v3 `addpkg` (step 06) with a govDAO-driven flip of `vm:p:sysnames_pkgpath` to `""` and back. Without the flip, `r/sys/names` rejects the addpkg ("namespace `sys` doesn't match caller").
- **`e6953c9b6`** — wires step 06 (the actual `MsgAddPackage` for `gno.land/r/sys/validators/v3`) into `migrations/build.sh`. Patches the tx's `creator` field to manfred (the pre-rotation T1) so `--skip-genesis-sig-verification` honors it as `OriginCaller`.
- **`8135d8ddc`** — `misc/deployments/test13.gno.land/govdao-scripts/add-validator.sh`. Operator script that builds an ad-hoc `MsgRun` invoking `valr.NewValsetChangeExecutor(...)` + `dao.MustCreateProposal/Vote/Execute` on the running chain. Used to validate the v3 flow end-to-end.
- **`ec3e2837f`** — `misc/hf-glue/fixvalidator/main.go` gains `--valset-list <path>` (multi-validator genesis input from a `<name> <power> <pubkey>` file), `--emit-json` (print the resolved valset in `NEW_VALSET_JSON` shape), and an internal `parseValsetList` helper.
- **`281d51ed9`** — patches `examples/gno.land/r/sys/validators/v3/poc.gno`. Moves the `changesFn()` evaluation out of the executor callback and into `NewValsetChangeExecutor` itself. Lazy eval closes the callback over the caller's `/e/<addr>/run` realm, which `dao.MustCreateProposal` cannot persist ("cannot persist function from the private realm"). Eager eval captures the resulting `[]validators.Validator` slice (plain data) instead.
- **`f6a7cdd79`** — adds migration step 08 (`08_set_valset_realm.gno.tmpl`) and the `migrations/build.sh` block for it. Sets `vm:p:valset_realm_path = "gno.land/r/sys/validators/v3"` so the EndBlocker's `getValsetRealmParam` doesn't fall through to "" stored from the gnoland1 era.

## Test Results

- **`gno.land/pkg/sdk/vm/...`** — PASS (`TestParams|TestValset` and surrounding). Covers `WillSetParam` for the `valset_realm_path` key, `validateValsetUpdate`, etc.
- **`misc/hf-glue/fixvalidator/`** (the `--valset-list` parser) — PASS.
- **`examples/gno.land/r/sys/validators/v3/`** — no `_test.gno` or `_filetest.gno` present in the realm directory (cherry-picked from #5485 unchanged + the rc3 eager-eval patch). Listed under Missing Tests.
- **Edge-case tests:** skipped per scope.

## Critical (must fix)

- [ ] **None new in this tier.** The rc1 chunked-fetch sort bug (in tier1 review) remains the launch-blocker.

## Warnings (should fix)

- [ ] `gno.land/pkg/sdk/vm/params.go:152-156` — **`getValsetRealmParam` treats stored `""` as authoritative.** This is upstream code from PR #5485 (cherry-picked here), but the hardfork specifically relies on its fragility. Pattern:
  ```go
  func (vm *VMKeeper) getValsetRealmParam(ctx sdk.Context) string {
      valsetRealm := ValsetRealmDefault           // "gno.land/r/sys/validators/v3"
      vm.prmk.GetString(ctx, ValsetRealmParamPath, &valsetRealm)
      return valsetRealm
  }
  ```
  `getIfExists` (`tm2/pkg/sdk/params/keeper.go:232`) leaves the receiver alone if the key is missing — but if the key is present with value `""`, it overwrites the default to `""`. After `loadAppState` runs `vm.InitGenesis` → `SetParams(ctx, gs.Params)` (`gno.land/pkg/sdk/vm/genesis.go:47-59`), every Params field gets written including `ValsetRealmPath`. Source-genesis Params from gnoland1 (struct created before the field existed) deserialize to the Go zero-value `""`, so the post-fork chain stores `vm:p:valset_realm_path = ""`. EndBlocker then reads `valsetRealm = ""` and silently drops every v3 valset update. ADR §2 step 08 is the migration that repairs this, but it's a one-shot fix — any future hardfork from a chain that didn't seed the param will re-discover the trap. Two suggestions:
  1. Coerce `""` → default in the getter: `if valsetRealm == "" { valsetRealm = ValsetRealmDefault }`.
  2. Reject `""` in `Params.Validate()` (line 97 currently accepts it: `if p.ValsetRealmPath != "" && !gno.IsRealmPath(...)`). With validation, `SetParams` from a hardfork-era genesis would fail loudly instead of silently storing the broken value.
  Either change makes step 08 defensive rather than load-bearing.

- [ ] `examples/gno.land/r/sys/validators/v3/poc.gno:35-66` — **Eager-eval fix changes semantics, not just internals.** The fix is correct for the persistence-failure problem (good — and the comment explains it). But the contract of `NewValsetChangeExecutor(changesFn)` has changed: callers can no longer rely on `changesFn` running at execute time. For static valset constructions (the only known callers today: `add-validator.sh`, `migrations/01_reset_valset.gno.tmpl` via v2's `NewPropRequest`, and the ones in `deployments/test13.gno.land/govdao-scripts/`), this is fine. But anyone passing a `changesFn` that reads on-chain state (e.g. "compute the new valset based on whichever stakes are highest at vote-time") will silently mis-evaluate — `changesFn` runs once at proposal creation, not at execution. Add an explicit caveat to the doc comment: "changesFn is invoked at proposal-creation time, NOT at execution time. Do not pass a function that reads time-varying chain state expecting it to be re-evaluated when the proposal executes." Pair with a runtime check that panics if `changesFn` returns different values on two consecutive calls (defensive, but cheap and surfaces the trap immediately).

- [ ] `misc/deployments/gnoland-1/migrations/build.sh:316-364` (rc2/rc3 additions) — **Migration jsonl ordering depends on the script's source order; the relationship is implicit.** Steps 05 → 06 → 07 → 08 must run in exactly this order: 05 disables namespace check, 06 deploys v3, 07 restores, 08 points the valset realm at v3. The script appends them in source order (`render_tx ... >>"$OUT_JSONL"` 4 times). `gnogenesis fork generate` then appends the entire jsonl to `appState.Txs` (`contribs/gnogenesis/internal/fork/generate.go:222`) — preserving order. So far so good. But there's no in-file assertion that the steps stay in this order — a future contributor reordering blocks for readability could silently break the launch (e.g. moving step 08 before 06 → param set before realm exists → still works, since the param check at vm runtime is just a string read; but moving step 06 before 05 → addpkg fails namespace check → migration tx fails → `--skip-failing-genesis-txs` swallows it → v3 not deployed → all subsequent valset updates lost). Add either (a) numbered comments at each `render_tx` block restating the dependency, or (b) a smoke test in CI that runs `migrations/build.sh` against a stub fixture and asserts the produced jsonl matches a golden file.

## Nits

- [ ] `misc/deployments/test13.gno.land/govdao-scripts/add-validator.sh:16` — `set -eo pipefail` (no `-u`). With `-u`, an unset `GNOKEY_NAME` etc. would fail loudly, but defaults are in place lines 18-22, so this is benign in practice. Worth adding for consistency with other scripts in the tree (`set -euo pipefail` is the project convention).
- [ ] `misc/deployments/test13.gno.land/govdao-scripts/add-validator.sh:39-69` — Direct shell substitution of `$ADDR`, `$PUB_KEY`, `$POWER` into the heredoc-emitted `.gno` file. Operator script, untrusted input is unlikely, but worth a `^g1[0-9a-z]{38}$` / `^gpub1[0-9a-z]+$` / `^[0-9]+$` validation pass before the heredoc.
- [ ] `examples/gno.land/r/sys/validators/v3/poc.gno:3-11` — Import-ordering reshuffle (`"strings"` moved out of the first import block, then back next to other stdlib imports). Cosmetic and unrelated to the eager-eval fix; should arguably have been a separate commit. Doesn't affect correctness — `gofmt`/`gnofmt` orders imports the same way regardless. Easy to drop into squash.
- [ ] `misc/hf-glue/fixvalidator/main.go` (whole change) — `--valset-list` parses `<name> <power> <pubkey>` per line. The format diverges from gno-cluster's `INITIAL_VALSET=(...)` array literal — the comment says "strip the `INITIAL_VALSET=(` wrapper and quotes before feeding it in", which is a manual transform users will get wrong. Either (a) detect and strip the wrapper inside `parseValsetList`, or (b) emit a one-liner shell snippet in the help text that does the strip.

## Missing Tests

- [ ] **No test for `getValsetRealmParam` empty-string fallthrough.** `gno.land/pkg/sdk/vm/params_test.go` should cover: (i) param unset → default returned, (ii) param set to non-empty realm path → that path returned, (iii) param set to `""` → returns `""` (current behavior — and the test will fail loudly if the suggested coercion-to-default change lands, prompting whoever does it to update both).
- [ ] **No filetest for `NewValsetChangeExecutor` eager-eval semantics.** A filetest under `examples/gno.land/r/sys/validators/v3/filetests/` that constructs an executor with a stateful `changesFn` and asserts the captured slice matches the value at construction-time (not execute-time) would document the new contract. Currently the test surface for v3 is mostly in `gno.land/pkg/gnoland/app_test.go` — those exercise EndBlocker integration but not the realm-internal `changesFn` behavior.
- [ ] **No CI smoke for `migrations/build.sh`.** A scripted run against a stub `RPC_URL` (canned `vm/qeval` response, ephemeral filesystem `PV_KEY`) that asserts the produced `migrations.jsonl` parses cleanly via `gnogenesis fork generate --migration-tx` (just the parse — no full replay). Would catch (a) template-placeholder bugs, (b) `jq`/`awk` quoting regressions in `render_template`, (c) step-ordering breakages.
- [ ] **No test for `parseValsetList` error paths.** `misc/hf-glue/fixvalidator/main_test.go` (introduced in `ec3e2837f`) covers the happy path; missing: malformed lines (2-field, 4-field, non-numeric power, malformed pubkey, address-pubkey mismatch when both are explicit).

## Suggestions

- Make `getValsetRealmParam` treat `""` as default, and tighten `Params.Validate()` to reject `""`. See Warning above. Cheap, removes a class of silent-divergence bugs from any future hardfork.
- Add a defensive `assert-migrations` check (rc4 territory) for "migration step 08 actually wrote a non-empty `vm:p:valset_realm_path`" — currently the assertion exists (rc4) but only checks the value matches the expected path. Easy to fail open if something writes "" later. Treat empty as a positive failure.
- The 05/06/07 disable-deploy-restore wrap is a working but ugly contract. ADR's Alternatives section rejects "fold v3 addpkg into pre-history genesis-mode txs" because it would desynchronise the SHA from independent rebuilds. A cleaner long-term primitive: a govDAO-gated `MsgAddSysPackage` (or `--allow-sys-namespace` flag on `MsgAddPackage`) that bypasses the namespace check based on caller authz alone. Out of scope for test-13 launch; worth filing as a follow-up issue.
- `add-validator.sh` (test13 deployment) is a useful operator primitive but doesn't echo the resulting proposal id back. Worth piping `gnokey maketx run` output through `jq` and printing the on-chain pid at end so the operator has a handle for follow-up queries (e.g. did it execute?).

## Questions for Author

1. The eager-eval fix in `poc.gno` — was the lazy-eval pattern in v3's original PR (#5485) intentional, or did it just happen to slip through review because no integration test exercised the propose+execute flow against `dao.MustCreateProposal`'s persistence? Worth a follow-up commit to upstream #5485 with this fix? (rc3 currently patches the cherry-picked v3 in-place; a clean upstream fix would let other forks of v3 inherit it.)
2. `getValsetRealmParam`'s empty-string trap: do you have signal on whether other params fields could have the same fallthrough trap (e.g. a future field added to `Params` after some chain genesis is sealed)? If yes, suggest a generalised "coerce zero-value to default" behaviour in `prefixParamsKeeper.GetString` or a wrapper.
3. `migrations/build.sh` step ordering — is there an automated check anywhere that the steps run in 01→08 order, or is it asserted only by reading the source? (This relates to the Warning above.)

## Verdict

NEEDS DISCUSSION — no new launch-blocker beyond tier 1. The eager-eval fix is correct; the sysnames-wrap is unavoidable given the design constraints; the EndBlocker wiring works. The `getValsetRealmParam` empty-string fragility deserves an upstream hardening even if the migration step 08 is sufficient for test-13 specifically.
