# PR #5477: fix(gnoland): fix data races in integration test harness, add event collector test

**URL:** https://github.com/gnolang/gno/pull/5477
**Author:** ltzmaxwell | **Base:** master | **Files:** 6 | **+191 -12**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes two data races detected by `go test -race` in the integration test harness and adds an integration test exercising the event collector through a full GovDAO validator proposal flow.

**Race fix 1 — stderr/stdout buffers (`testscript_gnoland.go:760-818`):** Previously, `setupNode` passed `ts.Stderr()` (a `strings.Builder`) as the node's stderr writer. Background goroutines in the node subprocess (the stdout forwarder in `waitForProcessReady`, the node's own stderr output) would write to this builder concurrently with testscript's main goroutine reading it. The fix introduces `safeWriter` (`testscript_gnoland.go:740-758`), a mutex-protected `bytes.Buffer`, used for both stdout and stderr. On error, `stderrBuf.String()` is safely read and printed to `ts.Stderr()`.

**Race fix 2 — `runTestingNodeProcess` return signature (`process.go:333-348`):** The function previously called `require.NoError` internally (which calls `t.FailNow()` from a non-test goroutine — undefined behavior per Go docs). It now returns `(NodeProcess, error)`, and the caller in `setupNode` handles the error explicitly.

**Race detection infrastructure (`race_on.go`, `race_off.go`, `testscript_gnoland.go:254-265`):** A build-tagged `raceEnabled` constant (true under `-race`, false otherwise) and a `[race]` testscript condition allow `.txtar` tests to conditionally run only under the race detector.

**Integration test (`testdata/event_race.txtar`):** A comprehensive test that exercises the event collector by triggering a real `ValidatorAdded` event through the GovDAO proposal flow. Gated by `[!race] skip` to only run with `-race`. The test registers a valoper, creates a GovDAO proposal, votes, executes, and verifies the validator was added — exercising the full `Emit -> updateWith -> getEvents -> QueryEval` path.

**TestMain timeout (`process_test.go:31`):** Changed from hardcoded `30 * time.Second` to `nodeMaxLifespan` (120s), needed because `-race` builds are significantly slower.

## Test Results
- **Existing tests:** Not run locally (race tests require `-race` flag and are slow). PR author confirmed the races are fixed.
- **CI status:** Lint failure on `main / lint` job. This appears to be a pre-existing `main` branch issue, not caused by this PR's changes. The PR's own CI checks are not yet fully reported.
- **Edge-case tests:** Skipped — the `event_race.txtar` test IS the edge-case test; it exercises the exact code paths that were racing.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/integration/testscript_gnoland.go:766-788` — **`stdoutBuf` is written to but never read.** The `safeWriter` captures stdout from the node subprocess, but on error only `stderrBuf` is printed (line 784, 797, 807). If the node fails during startup, stdout output (which may contain useful diagnostic info like the address the node is listening on, or startup progress) is silently discarded. thehowl noted this in their review as well. Consider either: (a) printing both buffers on error, or (b) combining them into a single buffer (`var combinedBuf safeWriter` used for both Stdout and Stderr).

- [ ] `gno.land/pkg/integration/testscript_gnoland.go:766-818` — **No cleanup logging for successful runs.** The `stderrBuf` is only printed when `setupNode` encounters an error during startup. If the node starts successfully but a later testscript command fails, the stderr output is lost. The `process_test.go` tests handle this well (lines 56-59: `defer func() { t.Log(stdio.String()) }()`), but `setupNode` does not register an equivalent cleanup. Consider adding `ts.Defer(func() { ... })` to always log output on test completion, not just on startup failure.

## Nits

- [ ] `gno.land/pkg/integration/testdata/event_race.txtar:44` — The skip message `'requires -race build flag'` is clear, but the condition `[!race] skip` means the test is skipped in the vast majority of CI runs (only the specific `-race` job runs it). This is intentional, but it means the GovDAO proposal flow exercised here gets very little CI coverage. Consider adding a separate non-race version of the test (without the skip guard) that tests the proposal flow without the race detector, so the functional correctness is validated in every CI run.

- [ ] `gno.land/pkg/integration/race_on.go:1` / `race_off.go:1` — These files contain only a constant. An alternative is `//go:linkname` or checking `sync/atomic` alignment, but the build-tag approach is the cleanest and most idiomatic. No change needed, just noting this is well done.

## Missing Tests

- [ ] No unit test for `safeWriter` itself — concurrent Write/String calls. The struct is simple (mutex + buffer), but a `-race` unit test with multiple goroutines would validate the synchronization — `gno.land/pkg/integration/testscript_gnoland.go:740-758`.

## Suggestions

- Combine stdout and stderr into a single `safeWriter` and always log it on test completion (success or failure). This matches the pattern in `process_test.go:55-59` and ensures diagnostic output is never lost. This was also suggested by thehowl in their review.

- The `event_race.txtar` test is excellent documentation of the event collector's concurrency model (the ASCII-art call graph in the comments is particularly helpful). Consider extracting the concurrency explanation into a code comment in `events.go` near the collector type definition, so future developers don't need to find this test to understand why the collector doesn't need a mutex.

## Questions for Author

- The CI lint failure on `main / lint` — is this a pre-existing issue on the base branch, or does this PR introduce a new lint violation? If pre-existing, the PR should be fine to merge once the base branch lint is fixed.

## Verdict

**APPROVE** — The data race fixes are correct and well-motivated. The `safeWriter` approach is simple and effective. The `event_race.txtar` test is thorough and well-documented. The stdout-not-logged issue (warning #1) is minor and doesn't block merging. Already approved by thehowl.
