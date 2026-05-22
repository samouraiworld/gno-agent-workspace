# PR #5589: chain/test13-rc6 (tier 1 — base + rc1)

**URL:** https://github.com/gnolang/gno/pull/5589
**Author:** aeddi | **Base:** chain/gnoland1 | **Files (full PR):** 300+ (truncated) | **+24,801 -**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7 (1M context)
**Scope of this file:** test13-base + rc1 — replay engine import + curated gno-core fix-set + hf-glue chunked fetch / keyless genesis / T1 rotation migration

## Summary

This tier covers the base layer of the test-13 hardfork stack:

- **test13-base** (5 commits) imports PR #5486 (the `hf-glue` testbed for replay) and adds a few small fixes to make it run cleanly against `chain/gnoland1`.
- **rc1** (~50 commits) cherry-picks the gno-core fix-set curated on the launch-prep hackmd — these are upstream-merged PRs (`#5336`, `#5141`, `#5535`, `#5474`, `#5498`, `#5330`, `#5039`, `#5515`, `#5331`, `#5501`, `#5244`, `#5434`, `#5266`, `#5329`, `#5447`, `#5391`, `#5081`, `#5378`, `#5032`, `#5407`, `#5439`, `#5479`, `#5356`, `#5445`, `#5436`, plus ~25 other earlier-master cherry-picks). It then adds the rc1-original work: keyless-genesis support (`VALIDATOR_ADDR`/`VALIDATOR_PUBKEY`), the chunked tx-archive fetch helper, `init-node.sh` idempotency, the `make up` auto-restage hook, and the `govDAO T1 rotation + valset-reset` migration.

The rc1 originals concentrate in two areas:

1. `misc/hf-glue/scripts/fetch-txs-chunked.sh` — chunk-aware tx-archive driver, retries flaky RPCs.
2. `misc/deployments/gnoland-1/migrations/` (templates 01-08 + `build.sh`) — synthesises the post-replay migration jsonl. Steps 01 (valset reset to v2) and 02-04 (T1 rotation) belong to rc1; steps 05-08 are added in rc2/rc3.

`contribs/tx-archive/backup/signerinfo.go` (246 lines, brand-new) was reviewed in detail — it does the brute-force `(account_num, sequence)` resolution that lets the hardfork ante handler force-set account state before signature verification. The logic is dense but matches the ADR's documented strategy; cosmetic-only edge cases noted.

Per the agreed scope, cherry-picked commits are not re-reviewed in full — only verified for clean import and adverse hardfork interaction. None spotted.

## Test Results

- **`contribs/tx-archive/backup/...`** — PASS (`backup`, `writer/legacy`, `writer/standard`).
- **`contribs/gnogenesis/internal/fork/` non-replay tests** (`TestReadMigrationTxs|TestPatchGenesisModeAddPkg|TestRPCSource|TestVerifyGenesisFile`) — PASS.
- **`contribs/gnogenesis/internal/fork/TestExecTest_EmptyGenesis`** — FAIL locally with `chain/runtime/native.gno: function getSessionInfo does not have a body but is not natively defined`. This is a stdlib codegen mismatch in the local checkout (native bindings out of sync with `chain/runtime/native.gno`); not introduced by this tier. Upstream CI is green for the PR per `gh pr checks 5589`.
- **`misc/hf-glue/fixvalidator/`** — PASS.
- **Edge-case tests:** skipped per scope.

## Critical (must fix)

- [ ] `misc/hf-glue/scripts/fetch-txs-chunked.sh:76` — **Chunk concatenation order is wrong.** The line `for f in $(ls "$CHUNK_DIR"/*.jsonl | sort -t- -k1 -n); do` sorts by field 1 numerically with `-` as delimiter. Field 1 is everything before the first `-` in the full path (`out/source/txs`), which is identical for every chunk and starts with non-digits, so `sort -n` reads it as `0` for every entry. The fallback is the input order from `ls`, which is lexical: `1-20000.jsonl, 100001-120000.jsonl, 1000001-1020000.jsonl, ..., 20001-40000.jsonl, 200001-220000.jsonl, ...`. Verified locally — `printf "...1-20000\n...20001-40000\n...100001-120000\n" | sort -t- -k1 -n` returns `1-20000, 100001-120000, 20001-40000` (wrong). Consequence: at any `HALT_HEIGHT > ~100,000` (always true for gnoland1 → test-13), `txs.jsonl` is concatenated out of block order. `gnogenesis fork generate`'s `FetchTxs` (`contribs/gnogenesis/internal/fork/source_dir.go:115-160`) reads sequentially and **does not re-sort** — confirmed via grep + read of `buildHardforkGenesis` (no `sort.Slice` on `appState.Txs`). Out-of-order replay → account-sequence mismatches → most txs fail → `--skip-failing-genesis-txs` swallows them silently → state diverges massively from source. `state-diff` / `audit-balances` would surface this, but only post-build. Fix: replace with one of:
  - `find "$CHUNK_DIR" -maxdepth 1 -name '*.jsonl' | sort -V` (GNU `sort -V` is version-sort, handles numeric runs correctly), or
  - `for f in $(ls "$CHUNK_DIR"/*.jsonl | awk -F/ '{ split($NF, a, "-"); print a[1]"\t"$0 }' | sort -k1 -n | cut -f2-)`.
  Unit-test by listing 250+ chunks at realistic heights and asserting `head -n 1` of the assembled file's first tx has the lowest `block_height`.

## Warnings (should fix)

- [ ] `misc/hf-glue/scripts/fetch-txs-chunked.sh:61-62` — **Cache check accepts truncated chunk files.** `if [[ -s "$chunk_file" ]] || [[ -f "$chunk_file.done" ]]; then ... cached`. The `||` makes "any nonempty file" sufficient. If a previous run was killed mid-write (network drop, ctrl-C, OOM), `tx-archive backup -overwrite` may have left a partially-written `.jsonl` on disk with no matching `.done`. Next run treats that file as cached and never re-fetches, producing a permanently-truncated chunk that flows into the assembled `txs.jsonl`. Fix: `if [[ -f "$chunk_file.done" ]]; then ...` — only the success marker is reliable, and the inner `wc -l` already tolerates a missing chunk file via `2>/dev/null`.

- [ ] `misc/deployments/gnoland-1/migrate-from-gnoland1.sh:1-120` — **Stub script still in tree, README still points reviewers at it.** Whole file is a TODO block ending in `exit 1` ("ERROR: ... is not yet implemented"). Real flow now lives in `misc/hf-glue/scripts/migrate.sh` (rc1 imported it from PR #5486). `misc/deployments/gnoland-1/README.md:48,62,72` still reference the stub as "the critical missing piece". Either delete the stub and update README to point at `make migrate` in `misc/hf-glue/`, or replace the stub body with a one-line redirect (`exec "$REPO/misc/hf-glue/scripts/migrate.sh" "$@"`). Leaving it as-is mis-leads anyone arriving via `misc/deployments/gnoland-1/`.

- [ ] `misc/deployments/gnoland-1/migrations/04_withdraw_manfred_execute.gno.tmpl:44-60` — **`findWithdrawProposal` matches by author + title only.** If any historical tx between fork-base and migration replay also produced a "Member Withdrawal Proposal" authored by `OLD_T1_ADDR`, `findWithdrawProposal` returns the highest-pid match (line 55-56 keeps overwriting `best`), so it'll pick the most recent one — which IS the one we want from migration step 03. But the migration runs after historical replay, so any source-chain proposal with the same author+title that happened to land at a later block height (impossible at this point because manfred has no plan to file his own withdrawal historically) would be picked instead. Fragile by accident, not by design. Fix: tighten the match (e.g. assert pid > maxObservedAtMigrationStart, or read pid back from migration 03's output and pin it explicitly via a placeholder).

- [ ] `contribs/tx-archive/backup/backup.go:85-91` — **Whole-backup buffering when `populateSignerInfo` is on.** With ~5M txs on gnoland1 (and the `signerResolver` enabled by default for non-watch backups), the whole tx stream is held in `bufferedTxs []*gnoland.TxWithMetadata` until `Finalize`, then flushed in one pass. Memory grows linearly with the source chain. The chunked fetch script (Critical above) sidesteps this by running smaller bounded backups per chunk — but a user calling `tx-archive backup` directly on the full range will OOM. Either document this loudly (README + `--from`/`--to` help text) or add `--chunked` semantics to the writer (per-batch flush, but only after the next-success back-patch within that batch is resolved). Lower-priority since the chunked script is the recommended path.

## Nits

- [ ] `misc/deployments/gnoland-1/migrations/build.sh:124` — `g1[0-9a-z]{38}` regex's class is broader than bech32's data charset (`qpzry9x8gf2tvdw0s3jn54khce6mua7l`) — `b`, `i`, `o`, `1` are excluded by the spec but allowed by `[0-9a-z]`. Won't reject a real address (a true bech32 g1-addr never contains `b/i/o`), but the comment "the bech32 data-charset for gpub1 pubkeys excludes the digit `1`" overstates what the regex enforces. Cosmetic.
- [ ] `misc/deployments/gnoland-1/migrations/build.sh:262` — same charset overstatement on `NEW_T1_ADDR` validator regex `^g1[0-9a-z]{38}$`. Cosmetic.
- [ ] `misc/hf-glue/scripts/migrate.sh:118` — `go run -C "$REPO/misc/hf-glue/fixvalidator" . --valset-list ...` recompiles fixvalidator on every migrate run. `go build` once into `out/bin/fixvalidator` is cheaper for the cluster scenario, where this is invoked per-node.
- [ ] `misc/deployments/gnoland-1/migrations/build.sh:336-337` — Cross-reference comment helpful: "creator=manfred even though manfred is no longer T1 after step 04, because MsgAddPackage requires only namespace authz, which step 05 disabled". The current `# MsgAddPackage uses 'creator' (not 'caller')` line is good but doesn't explain why this is safe under post-rotation T1 state.

## Missing Tests

- [ ] **Chunk concat order under realistic chunk count.** Add a fixture in `misc/hf-glue/scripts/` (or shellspec/bats) that exercises `fetch-txs-chunked.sh` against ≥100 stub chunks at heights covering the lex-sort divergence ([`1-20000`, `20001-40000`, …, `4980001-5000000`]) and asserts the assembled `txs.jsonl` is in monotonic block-height order. This would have caught the Critical above immediately.
- [ ] **Resilience check on `signerResolver.Populate` first-success brute-force.** No unit test in `contribs/tx-archive/backup/` exercises the back-patching of `pendingFails` when `lo > hi` (i.e. `ss.seq > ss.finalSeq` due to a brute-force overshoot). The current code returns `lo, error` from `bruteForceSignerSequence` and the caller falls back to `ss.seq` — silently — which produces wrong (but cosmetic) sequence numbers on failed txs. A unit test asserting the silent-fallback path is hit would document the behavior.
- [ ] **Migration jsonl smoke test.** No CI exercises `misc/deployments/gnoland-1/migrations/build.sh` against a mock RPC. The amino-printed output of `vm/qeval` for `r/sys/validators/v2.GetValidators()` is regex-extracted (line 124); a schema change in `r/sys/validators/v2` (rename `Validator.Address` → `Address.Bech32`, etc.) would silently strip the OLD_ADDRS. A scripted smoke that asserts the produced jsonl validates against the gnogenesis schema on a known input would catch this.

## Suggestions

- Replace the brittle `awk` placeholder substitution in `migrations/build.sh:235-240` with `sed`-with-delimiter-detection or a Go helper that reads the template + a `key=value` map. Bash-quoting `"$NEW_T1_ADDR"` etc. into an awk replacement string already needed an `ENVIRON[]` workaround (line 235); it's one edge case away from breaking on a `$portfolio` containing `\n` or `\\`.
- `init-node.sh:58-61` mutates `config.toml` via `sed`. The `gnoland config edit` subcommand (or `set-config`) is a more durable contract than text-rewriting a TOML file. The `tcp://127.0.0.1:26657 → 0.0.0.0` substitution will silently no-op the day someone changes the default in the upstream config template.
- The `fetch-txs-chunked.sh:38-48` retry loop has linear backoff (`sleep $((attempt * 5))`). Exponential or jittered backoff is friendlier to the upstream RPC. Low priority.

## Questions for Author

1. Has `fetch-txs-chunked.sh` ever been exercised end-to-end against gnoland1's real halt height (i.e. enough chunks to surface the lex-sort bug)? If yes, what mitigated it — was the assembled `txs.jsonl` post-sorted somewhere I missed, or did the launch always run with a single non-chunked tx-archive pass? Important to know whether prior `verify-reproducibility` runs were actually exercising the chunked path.
2. The `migrate-from-gnoland1.sh` stub: intentionally kept as a TODO marker, or was it superseded silently by the hf-glue import? If superseded, OK to delete in a followup.
3. `contribs/tx-archive/backup/backup.go:89` — buffering all txs when `populateSignerInfo` is on. Was OOM observed against gnoland1's tx volume in practice, or has every real run gone through the chunked path? Worth a doc note in `cmd/backup.go --help` either way.
4. `signerinfo.go:113-141` — the comment at L131-133 says "subsequent txs may fail verification". For the test-13 launch this is fine (sigs are skipped via `--skip-genesis-sig-verification`), but a future fork that wants to keep signature checks would need a louder failure mode here. Worth a follow-up issue?

## Verdict

NEEDS DISCUSSION — the chunked-fetch sort bug is a launch-blocker if anyone runs the chunked path on the real gnoland1 history. Easy fix; just needs to land before the `verify-reproducibility` SHA is treated as authoritative across machines. Other findings are addressable in followups; the rc1 originals are otherwise solid and the cherry-pick set imports cleanly.
