# Stdlib Test Gap Hunt

Find bugs in Gno's stdlib by running Go 1.25.9 tests that haven't been ported.

Gno tracks Go 1.25.9. Many `gno/gnovm/stdlibs/*` packages are direct ports of Go's stdlib with `_test.gno` files derived from upstream `_test.go`. Tests present upstream but missing in Gno are an unaudited surface — if a missing test fails when ported, it points at a real divergence in Gno's port.

## Layout

All artifacts live under `audits/stdlib-test-gap/`:

- `README.md` — this design.
- `run.sh` — the survey script (Phase 1).
- `report.md` — Phase 1 output.
- `bugs.md` — aggregated Phase 2 findings (issue-ready).
- `findings/<batch>.md` — per-batch findings before aggregation.
- `.cache/go1.25.9/<pkg>/` — cached upstream `_test.go` files (gitignored).

## Baseline

Phase 2 runs against `gnolang/gno@master` with PR
[#5723](https://github.com/gnolang/gno/pull/5723) cherry-picked into the
worktree. Without it, allocator-overflow class bugs surface as
unrecoverable host panics that mask everything downstream. The cherry-pick
gives a clean baseline to find *additional* bugs.

## Phase 1 — Survey

Script: `audits/stdlib-test-gap/run.sh` (starts with the `NOT AUDITED` disclaimer per repo convention).

For each subdir under `gno/gnovm/stdlibs/` that matches a Go stdlib package:

1. Extract `^func (Test|Example)\w+` from all local `_test.gno` files → Gno set.
2. Fetch matching Go 1.25.9 `_test.go` files from `https://raw.githubusercontent.com/golang/go/go1.25.9/src/<pkg>/`, cache under `audits/stdlib-test-gap/.cache/go1.25.9/<pkg>/`.
3. Same grep → Go set.
4. Diff: tests in Go but not in Gno → "missing".

Output: `audits/stdlib-test-gap/report.md`. One table per package:

| package | Go tests | Gno tests | missing | sample missing names |

Sorted by missing count, descending.

Name-level diff only — won't detect ported tests with cases removed inside the function body. Acceptable for triage; subcase analysis is out of scope.

Packages to include (mirror Go's stdlib): `bufio`, `bytes`, `encoding/*`, `errors`, `hash/*`, `html`, `io`, `math`, `path`, `regexp`, `sort`, `strconv`, `strings`, `time`, `unicode`. Gno-only packages (`chain`, `gno`, `testing`, native shims) are skipped.

## Phase 2 — Drill in

Pick a target package using the report. Bias toward:

- Non-trivial gaps (~10–50 missing tests).
- Packages exercising tricky semantics: `strconv`, `strings`/`bytes` (UTF-8 boundaries), `path` (cleaning), `time` (formatting), `regexp` (engine quirks).
- Skip packages dominated by reflect/unsafe/generics/goroutines/`io.Reader` helpers Gno can't run (e.g. most of `crypto/*`).

Workflow:

1. Create worktree `.worktrees/gno-stdlib-test-port/` per repo rule (no in-place edits to `gno/`).
2. Port missing tests one at a time, simplifying away Gno-incompatible constructs (subtests via `t.Run` → flatten if needed, `testing/quick` → skip, generics → skip).
3. Run with `gno test ./gnovm/stdlibs/<pkg>/`.
4. Each failure → minimal repro + entry in `audits/stdlib-test-gap/bugs.md`.
5. Stop at ~3–5 confirmed bugs or when the well runs dry. Return to user before filing upstream issues.

## Out of scope

- AST-level subcase diff.
- Auto-classifying which missing tests are portable.
- Filing issues on gnolang/gno (batch, review with user first).
- Touching anything outside the matched packages.
- Modifying Gno stdlib code to fix the bugs found (separate task).
