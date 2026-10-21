# Review: [#6061](https://github.com/gnolang/gno/pull/6061)
Event: COMMENT
Verdict: APPROVE. The findings left are Nits and one test assertion; a cgroup at its limit delays a node cut by one poll rather than blocking it.
Model: claude-opus-5-5, quick review
Commit: 115d118c79c72256f69dcd5e419104f794316b8f
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6061 115d118c7`
Round: 2. The head moved from a4d6089 to 115d118c7 through d93c06f64, 7bf249c7b, 4caf70d48 and 115d118c7, the two merges carrying no hunk. Round 1's zero-Available reading is carried, as the Nit at parallel_linux.go:62 and the SKIP at testscript_gnoland.go:215-217; its cgroupDir test gap is carried as the SKIP at parallel_linux.go:54; the page-cache charge, the cgroup v1 and ancestor reads and the examples Makefile worker count are not raised again by this round's finders. 3 finders, no reflector, 8 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted. The parent moved the zero-Available finding onto the line its edit changes and banded it Nit once a pressure probe showed 7 and 12 failed readings in 2000, every other reading cutting, banded parallel_test.go:60 as Test, and held both Suggestions, unrun.

## Body
> AI review, claude-opus-5-5, quick review, reviewed at [115d118](https://github.com/gnolang/gno/pull/6061/commits/115d118c79c72256f69dcd5e419104f794316b8f), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6061-bound-test-memory-by-memory/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6061-bound-test-memory-by-memory/2-115d118/claims.md) · Status: APPROVE

## tm2/pkg/testutils/parallel_linux.go:62 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_linux.go#L62) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L62) · Nit
Nit: `mi.Available > 0` turns a cgroup at its [`memory.max`](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_linux.go#L74-L77) into a failed read, so [`resize`](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L215-L217) skips the cut that reading calls for and reads again on the next poll. [The darwin reader](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_darwin.go#L31) returns the same way.

## SKIP gno.land/pkg/integration/testscript_gnoland.go:215-217 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L215-L217) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L215) · Nit
Nit: `ReadMemInfo` reports a zero `Available` as a failed read, so exiting on `!ok` skips a cut in a cgroup at its `memory.max`.

Not posted: the same defect as the Nit at parallel_linux.go:62, whose one edit closes it.

## SKIP gno.land/pkg/integration/testscript_gnoland.go:246 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L246) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L246) · Missing test
Missing test: replacing `b.running > b.limit` with `true` leaves every `NodeBudget` test green.

Not posted: PLAUSIBLE on the finder's read, the mutation not run.

## SKIP gno.land/pkg/integration/testscript_gnoland.go:313 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L313) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L313) · Missing test
Missing test: `acquireSlot`'s `held` flag has no test, and `gnoland.txtar` starts a node twice in one script, which only that flag keeps to one slot.

Not posted: PLAUSIBLE on the finder's read, the mutation not run.

## SKIP gnovm/cmd/gno/test.go:261 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/cmd/gno/test.go#L261) · [↗](../../../../../.worktrees/gno-review-6061/gnovm/cmd/gno/test.go#L261) · Missing test
Missing test: no Go test or txtar passes `-p`, so deleting the clamp at `GOMAXPROCS` stays green.

Not posted: PLAUSIBLE on the finder's read, the mutation not run.

## SKIP tm2/pkg/testutils/parallel_linux.go:54 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_linux.go#L54) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_linux.go#L54) · Missing test
Missing test: deleting the cgroup v1 bail or the clamp at each ancestor leaves every test green.

Not posted: PLAUSIBLE on the finder's read, the mutation not run.

## SKIP tm2/pkg/testutils/parallel_test.go:54 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_test.go#L54) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_test.go#L54) · Missing test
Missing test: `TestReadMemInfo` skips whenever `ReadMemInfo` returns false, so a reader that never reads passes as a skip, and no CI job runs the darwin reader.

Not posted: PLAUSIBLE on the finder's read, the mutation not run.

## examples/Makefile:31 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/examples/Makefile#L31) · [↗](../../../../../.worktrees/gno-review-6061/examples/Makefile#L31) · Nit
Nit: the `grep -xE '0*[1-9][0-9]*'` filter rejects `+8`, which [`strconv.Atoi`](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel.go#L24) accepts. It also passes a 20-digit value that `gno test` cannot parse, and [the recipe](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/examples/Makefile#L45-L53) still exits 0.

## gno.land/pkg/integration/testscript_gnoland.go:163 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L163) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L163) · Nit
Nit: the `reserve` set here is non-zero even when `min` equals `max`, as on four cores or fewer, so `resize` reads memory every second for a limit that cannot move.

## SKIP gno.land/pkg/integration/testscript_gnoland.go:167 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L167) · [↗](../../../../../.worktrees/gno-review-6061/gno.land/pkg/integration/testscript_gnoland.go#L167) · Nit
Nit: the floor `min: n` is `MaxParallel()`, where the description says the budget shrinks back towards two.

Not posted: a mistake in the description, and UNVERIFIED on the finder's read.

## SKIP tm2/pkg/testutils/parallel.go:24 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel.go#L24) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel.go#L24) · Nit
Nit: `strconv.Atoi` takes `GNO_TEST_MAX_PARALLEL=+8` as 8 for the Go suites while the Makefiles fall back to 4.

Not posted: the same mismatch as the Nit at examples/Makefile:31.

## tm2/pkg/testutils/parallel_test.go:60 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_test.go#L60) · [↗](../../../../../.worktrees/gno-review-6061/tm2/pkg/testutils/parallel_test.go#L60) · Test
Test: the 256 MiB floor fails a correct reading in any cgroup limited below it, since [`clampToCgroup`](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_linux.go#L72) caps `Total` at the cgroup limit.

<details><summary>repro</summary>

The run fails on a correct 200 MiB reading, which is the finding.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6061 -R gnolang/gno
go test -c -o /tmp/tu.test ./tm2/pkg/testutils
systemd-run --user --scope -q -p MemoryMax=200M /tmp/tu.test -test.run 'TestReadMemInfo$' -test.v
rm /tmp/tu.test
```

```
"209715200" is not greater than "268435456"
--- FAIL: TestReadMemInfo
```
</details>

## SKIP gnovm/pkg/gnolang/files_test.go:66 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/pkg/gnolang/files_test.go#L66) · [↗](../../../../../.worktrees/gno-review-6061/gnovm/pkg/gnolang/files_test.go#L66) · Suggestion
Suggestion: `poolSize` takes `GNO_TEST_MAX_PARALLEL` unclamped, where `gno test` [lowers the same value](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/cmd/gno/test.go#L261-L262) to `GOMAXPROCS`.

Not posted: a Suggestion this round neither applied nor ran both ways; the store count it rests on is unmeasured.

## SKIP gnovm/pkg/gnolang/files_test.go:186 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/pkg/gnolang/files_test.go#L186) · [↗](../../../../../.worktrees/gno-review-6061/gnovm/pkg/gnolang/files_test.go#L186) · Suggestion
Suggestion: this `freshStore` channel adds a second budget beside [`TestFiles`'s own](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/pkg/gnolang/files_test.go#L76).

Not posted: a Suggestion this round neither applied nor ran both ways; the store count it rests on is unmeasured.
