# PR #5663: test(misc/e2e): add gnovm audit and e2e regression scripts

**URL:** https://github.com/gnolang/gno/pull/5663
**Author:** louis14448 | **Base:** master | **Files:** 15 | **+948 -6**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Adds a Docker-based E2E test suite under `misc/e2e/` consisting of two
script families and a runner that aggregates results:

- **`audit/`** — eight POSIX-sh scripts, each pinned to a specific gnovm
  hardfork-era bugfix. Each script deploys a synthetic realm or runs a
  `maketx run` script that triggers the bug path, then asserts on the
  resulting output (`OK!`, `out of gas`, a Render value, etc.) and exits
  with status 1 when the symptom would indicate the fix is absent.
  Targets are: `runtime` stdlib removal (`afd7e4808`), `chan` preprocess
  rejection (`4bcd9828e`), uint64 overflow at compile time + iterative
  recursion gas exhaustion (`6a6fc4c71`, `3be0408f0`), per-byte alloc gas
  (`5d5f9213f`), `DidUpdate` on `DataByte` index assign (`a3a356e71`),
  `ArrayValue.Copy` deep-copy (`c64feef1d`), Go-order package-var init
  (`50ee56e64`), and panic+recover state rollback (`f87249327`).

- **`e2e/`** — three transaction-flow scripts: sequence-number replay
  protection, deploy-call-query of a counter realm, and a 10-tx
  sequential mempool stress against the same counter.

- **`run_tests.sh`** is extended with a `run_test` helper that tracks
  `PASS / FAIL / KNOWN` buckets, runs every script even on individual
  failure, prints a summary table at the end, and only exits 1 when
  `FAIL > 0`. `audit_cross_realm_recover` is wired with a `KNOWN_NOTE`
  because the targeted fix `f87249327` covers `NameExpr+recover` but
  not the broader method-receiver+recover pattern this script exercises.

- **`Makefile`** captures the `docker compose` exit code, re-greps the
  `gnokey-test` container logs for `[PASS]/[FAIL]/[KNOWN]/PASS:/FAIL:/
  KNOWN:` lines, prints them under a `TEST REPORT` banner, and exits
  with the original code.

- A `README.md` documents the layout, how to run, per-script targets,
  and the shared `common.sh` config variables.

All scripts use `#!/bin/sh` (POSIX, runs under BusyBox `ash` in the
Alpine `gnokey` image), no `bash`-isms in hot paths, source
`audit/common.sh` for `RPC`/`CHAINID`/`KEY` config, derive a unique
`SUFFIX=$(date +%s)` for realm pkg paths so deploys don't collide
between runs, and `trap 'rm -rf "$TMPDIR"' EXIT` to clean up.

The `test1` mnemonic embedded in `common.sh` and `run_tests.sh` is the
well-known integration-test mnemonic already published in
`gno.land/pkg/integration/node_testing.go` and many txtars — not a
secret.

## Test Results
- **Existing tests:** N/A — PR adds no Go tests; bash scripts not
  executed locally (require Docker compose / gnoland container).
- **Edge-case tests:** skipped (static review only, per instructions).
- **CI:** all automated checks pass.

## Critical (must fix)
None.

## Warnings (should fix)
- [ ] `misc/e2e/run_tests.sh:144` — `audit_cross_realm_recover` is wired
  with a `KNOWN_NOTE`, which means it can only ever land in `PASS` or
  `KNOWN` bucket — never `FAIL`. That's the intent today, but the
  current `run_test` semantics flip the logic too far: if a future
  master fixes the broader pattern and the script starts passing, the
  `KNOWN_NOTE` becomes silently misleading (it'll still claim "broader
  pattern not yet fixed" in PASS rows? No — re-reading lines 121-130,
  PASS path overrides `KNOWN_NOTE` correctly). The real risk is the
  opposite: if someone later adds *another* script that's expected to
  fail and forgets to remove the note, regressions stop blocking CI.
  Recommend (a) commenting in `run_tests.sh` that `KNOWN_NOTE` must
  only be set for scripts whose VULNERABLE output is the **expected
  baseline on master**, and (b) periodically pruning entries — or
  inverting the semantic so a `KNOWN`-marked test that unexpectedly
  passes is itself flagged.
- [ ] `misc/e2e/audit/audit_chan_type.sh:34-39` — uses `FAIL:` /
  `PASS:` instead of the `❌ VULNERABLE` / `✅ PATCHED` vocabulary every
  other audit script uses. The Makefile's `TEST REPORT` greps for
  `[PASS]|[FAIL]|[KNOWN]` (set by `run_tests.sh`, not these inner
  strings), so functionally harmless — but a reader scanning container
  logs for `FAIL` will hit this script even when it succeeded.
  Suggest aligning to `✅ PATCHED` / `❌ VULNERABLE` like its peers.
- [ ] `misc/e2e/audit/audit_security.sh:14,71-72` — emoji and "🏁 Audit
  Complete." trailing line don't match the per-script convention used
  by the other seven audit scripts (each prints one `🧪` header and a
  single PATCHED/VULNERABLE line). Cosmetic but the script bundles two
  checks (overflow + recursion) which means a single `[PASS]` summary
  row hides which sub-check actually ran — consider splitting into two
  scripts or printing two distinct verdict lines.
- [ ] `misc/e2e/audit/audit_runtime_pkg.sh:36` — the grep pattern
  `unknown import|cannot find|not found|unavailable|no package` is
  generous and will match any network-level failure too (e.g. a
  transient "node not found"), masking a node outage as a successful
  patch verification. Tightening to the exact gnovm message
  (`unknown import path "runtime"`) would make the check stricter
  without losing coverage. Same concern applies to
  `audit_gas_alloc.sh:34` (`out of gas|gas limit|exceeded` — `exceeded`
  alone could match unrelated errors).

## Nits
- [ ] `misc/e2e/audit/common.sh` and `misc/e2e/run_tests.sh:8` — the
  `test1` mnemonic is duplicated in two places. PR description
  acknowledges this. Prefer sourcing `common.sh` from `run_tests.sh`
  too (or extracting `TEST1_MNEMONIC` / `TEST1_ADDR` to a single
  config file under `misc/e2e/`).
- [ ] `misc/e2e/audit/common.sh` — no `set -e` / `set -u`. If
  `mktemp -d` ever fails (out of inodes, /tmp readonly), `$TMPDIR`
  stays empty and subsequent `cat > "$TMPDIR/foo.gno"` writes to
  `/foo.gno` — recoverable but ugly. Consider a single `: "${TMPDIR:?
  mktemp failed}"` after the mktemp call in each audit script, or
  factor that into `common.sh`.
- [ ] `misc/e2e/audit/*.sh` — `SUFFIX=$(date +%s)` collides if two
  scripts targeting the same package path run within the same wall
  second (they don't currently, since `run_tests.sh` runs them
  sequentially and each uses a different subpath, but if anyone
  adds parallelism later, two suffixes will collide). Prefer
  `mktemp -u` or `$$` (PID) appended.
- [ ] `misc/e2e/e2e/e2e_counter.sh:74` and
  `misc/e2e/e2e/e2e_mempool_stress.sh:82-83` — `sleep 2` / `sleep 5`
  are unnecessary: `gnokey maketx run -broadcast` uses
  `BroadcastTxCommit` (see `tm2/pkg/crypto/keys/client/broadcast.go:
  138`), which blocks until the tx is included in a block. The state
  is queryable immediately after broadcast returns. Removing the
  sleeps speeds the suite without losing reliability.
- [ ] `misc/e2e/run_tests.sh:136-139` — formatting alignment broken:
  `"audit_security"` and `"audit_gas_alloc"` use one fewer space than
  `"audit_runtime_pkg"`. `gofmt`-equivalent for shell would be nice;
  visually distracting but trivial.
- [ ] `misc/e2e/audit/audit_chan_type.sh:5` — comment claims the
  pre-fix behavior was "node panicked only at runtime when the
  channel was actually used". Verify this matches the fix's actual
  scope — `4bcd9828e` rejects at preprocess and runtime per its
  message. Comment is fine; just confirming.
- [ ] `misc/e2e/run_tests.sh:1` — script is `sh` not `bash`, but
  uses `local`-free helpers (good). The hardcoded `/usr/bin/gnokey`
  full path in the first half (lines 20-108) is unnecessary; the
  later half just calls `gnokey` via PATH. Inconsistent style.
- [ ] `misc/e2e/Makefile:10` — the grep used for the post-hoc
  summary will silently produce no output if container logs were
  evicted or rotated (very unlikely for a `docker compose` run, but
  the `2>/dev/null` swallows any error). Reasonable trade-off.

## Missing Tests
- [ ] No assertion that the `KNOWN` exit path is itself observable —
  if `audit_cross_realm_recover` ever silently becomes a no-op (e.g.
  realm fails to deploy), the script returns non-zero and `KNOWN`
  fires regardless. A failed deploy should be a `FAIL`, not a
  `KNOWN`. Consider distinguishing "the script ran and the symptom
  was VULNERABLE" (legitimate KNOWN) from "the script crashed before
  asserting anything" (should be FAIL). One approach: have each
  audit script exit with a distinct code (e.g. `2` for VULNERABLE,
  `3` for setup failure) and have `run_test` only accept exit `2` as
  KNOWN. Currently any non-zero is treated as the "known" symptom.
- [ ] No e2e test exercises a non-trivial cross-realm call path,
  which is precisely the area that produced multiple recent fixes
  (`f87249327`, NEWTENDG-* class). The PR adds an audit for the
  panic+recover symptom but no positive e2e that a correct
  cross-realm call commits as expected.
- [ ] `e2e_mempool_stress.sh` is sequential, not concurrent. Real
  mempool stress would fire txs in parallel and race them. The
  current implementation effectively tests `BroadcastTxCommit`
  back-to-back, not mempool ordering. Either rename
  (`e2e_sequential_txs.sh`) or extend to actually stress concurrency
  via `&` + `wait`.

## Suggestions
- Move `common.sh` to `misc/e2e/common.sh` (PR description suggests
  this); the relative `../audit/common.sh` from `e2e/e2e/*.sh` is
  awkward.
- Rename `misc/e2e/e2e/` → `misc/e2e/consensus/` or
  `misc/e2e/transactions/` (also suggested in PR description).
- Add a `--filter <regex>` option to `run_tests.sh` so a developer
  iterating on one fix can re-run only its audit.
- Consider running shellcheck in CI for `misc/e2e/**/*.sh` to lock
  in the POSIX-sh contract. A `shellcheck -s sh` pass on these
  scripts would be useful pre-merge.
- The audit scripts hard-code message patterns (`out of gas`,
  `unknown import`); a future gnovm refactor of error strings will
  silently turn audits into false PASS / false FAIL. A regression
  test for the error messages themselves (in Go) would back-stop
  the bash assertions.

## Questions for Author
- Is the intent that `audit_cross_realm_recover` graduate from KNOWN
  to PASS once the broader recover pattern is fixed? If so, who owns
  flipping the marker? Worth a TODO in the comment at line 141-143
  with an issue link.
- Why are the dispatch lines (134-150) calling absolute paths under
  `/e2e/...` (the container-mounted volume) instead of relative to
  `$SCRIPT_DIR`? It works because of the docker-compose mount, but
  it ties `run_tests.sh` to that specific deployment shape and means
  the script can't be run standalone outside the container. A
  `SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"` + relative paths
  would let a developer run `./run_tests.sh` against a
  locally-running gnoland.
- Any reason `audit_security.sh` bundles two unrelated fixes
  (`6a6fc4c71` integer overflow + `3be0408f0` recursion)? Splitting
  into `audit_int_overflow.sh` and `audit_recursion.sh` would
  improve the per-fix granularity the rest of the suite already
  provides.

## Verdict
APPROVE — the suite is well-structured, POSIX-sh-correct, each audit
script does fail with non-zero status on the vulnerable symptom (the
failure path is real, not informational), and the `run_tests.sh`
aggregator correctly propagates `FAIL > 0` to a non-zero exit. The
nits and warnings above are quality-of-life improvements, not
blockers. Recommend addressing the `KNOWN` semantic ambiguity
(distinguishing setup-failure from expected-vulnerable) and the
relative-path issue in a follow-up.
