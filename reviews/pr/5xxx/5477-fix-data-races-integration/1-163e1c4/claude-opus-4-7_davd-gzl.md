# PR #5477: fix(gnoland): fix data races in integration test harness, add event collector test

URL: https://github.com/gnolang/gno/pull/5477
Author: ltzmaxwell | Base: master | Files: 6 | +191 -12
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5477 163e1c4` (then `gh -R gnolang/gno pr checkout 5477` inside it)

**Verdict: APPROVE** — race fixes are sound and the new test exercises a real chain path; only blockers are cosmetic (dead `tsLogWriter`, unused `stdoutBuf` already flagged by [@thehowl](https://github.com/gnolang/gno/pull/5477#discussion_r2042681497)).

## Summary

`go test -race` on the integration harness tripped two races: (1) `setupNode` passed testscript's internal `*strings.Builder` (`ts.Stderr()`) as the subprocess `cmd.Stderr`, so the node wrote to it from a background goroutine while testscript's main goroutine read it via `clearBuiltinStd`; (2) a `tsLogWriter` adapter called `ts.Logf` from the `waitForProcessReady` stdout-forwarding goroutine, touching testscript's internal log buffer concurrently. Both are replaced with a mutex-protected `safeWriter` buffer that decouples background writes from testscript internals, and `runTestingNodeProcess` no longer calls `require.NoError` from a non-test goroutine (undefined per Go testing docs). The PR also adds a `[race]` testscript condition (via build-tagged `raceEnabled` const) and `event_race.txtar`, a GovDAO-driven validator-add flow that exercises the eventless-mutex `collector` end-to-end under `-race`.

```
node subprocess                       testscript main goroutine
─────────────────                     ─────────────────────────
  stderr ──► strings.Builder ◄────── clearBuiltinStd()         (BEFORE: race)
  stdout ──► tsLogWriter ──► ts.Logf ◄── (same builder)        (BEFORE: race)

  stderr ──► safeWriter{mu,buf}  (mu-guarded)                  (AFTER: safe)
  stdout ──► safeWriter{mu,buf}  (mu-guarded, never read)      (AFTER: safe but swallowed)
```

## Glossary

- `safeWriter` — new mutex-protected `bytes.Buffer` wrapping subprocess stdout/stderr (testscript_gnoland.go:743-758).
- `collector` — generic `[]T` event buffer in `gno.land/pkg/gnoland/events.go`; appended by `updateWith()` (listener callback), drained by `getEvents()` (EndBlocker). No mutex; PR claims both calls always run on the consensus `receiveRoutine`.
- `[race]` condition — true under `-race` builds via build-tagged const `raceEnabled` (race_on.go / race_off.go).
- `nodeMaxLifespan` — 120s subprocess lifetime constant, now also used as `TestMain` timeout.

## Fix

`setupNode` (`testscript_gnoland.go:760-818`) now declares local `var stdoutBuf, stderrBuf safeWriter` and wires both as `ProcessConfig.Stdout` / `ProcessConfig.Stderr`. On startup failure, `stderrBuf.String()` is flushed to `ts.Stderr()` before `ts.Fatalf`. `runTestingNodeProcess` (`process.go:333-348`) now returns `(NodeProcess, error)` instead of asserting internally. `TestMain` (`process_test.go:31`) uses `nodeMaxLifespan` instead of a hard-coded 30s, needed because `-race` builds load heavy genesis packages well past 30s. The `[!race] skip` guard in `event_race.txtar` means the new test only runs when the binary is compiled with `-race` — the race detector is the entire point of the test.

## Critical (must fix)

None.

## Warnings (should fix)

- **[dead code, was the bug]** [`testscript_gnoland.go:731-738`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/testscript_gnoland.go#L731-L738) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/testscript_gnoland.go#L731-L738) — `tsLogWriter` type/method are defined but have zero callers anywhere in the tree.
  <details><summary>details</summary>

  `grep -rn tsLogWriter gno.land/` returns only the definition. Per the commit message, this adapter was the source of race fix #2 — it called `ts.Logf` from the `waitForProcessReady` goroutine. The fix replaced its usage with `safeWriter`, but the type was never removed. Leaving it in invites someone to wire it back up and re-introduce the exact race this PR fixes. Fix: delete the type and method (8 lines).
  </details>

- **[@thehowl](https://github.com/gnolang/gno/pull/5477#discussion_r2042681497) [silent stdout]** [`testscript_gnoland.go:766-808`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/testscript_gnoland.go#L766-L808) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/testscript_gnoland.go#L766-L808) — `stdoutBuf` is written into but never read; on error only `stderrBuf` is surfaced.
  <details><summary>details</summary>

  `pcfg.Stdout = &stdoutBuf` captures the subprocess stdout — which contains the `READY:<address>` line and any startup logging — but the three error paths (lines 784, 797, 807) all print only `stderrBuf.String()`. If a node fails to emit `READY` (the common failure mode), the diagnostic is in stdout, not stderr, so the test author sees "unable to start ... node" with no context. Fix: either merge into a single combined `safeWriter` used for both Stdout and Stderr (thehowl's suggestion), or print both buffers on each error path.
  </details>

- **[diagnostics lost after successful start]** [`testscript_gnoland.go:760-818`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/testscript_gnoland.go#L760-L818) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/testscript_gnoland.go#L760-L818) — buffers are only flushed on startup failure; failures during the testscript body silently discard node output.
  <details><summary>details</summary>

  `process_test.go:55-59` uses `defer func() { t.Log(stdio.String()) }()` so node output is always logged. `setupNode` does not register an equivalent `ts.Defer(...)`, so if `gnoland start` succeeds but a later `gnokey` command fails, the node's stderr (panics, slog errors) is dropped. For a `-race`-gated test exercising consensus, this is exactly when the operator most needs the log. Fix: add `ts.Defer(func() { fmt.Fprint(ts.Stderr(), stderrBuf.String()) })` (or combined buffer) after successful startup.
  </details>

## Nits

- [`testdata/event_race.txtar:43-44`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/testdata/event_race.txtar#L43-L44) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/testdata/event_race.txtar#L43-L44) — `[!race] skip` means the GovDAO → valoper → proposal → execute flow only runs in the `-race` lane. The flow itself is non-trivial and would be useful to exercise in every CI run; consider duplicating without the guard, or splitting the race-only assertion from the functional one.
- [`race_on.go:1`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/race_on.go#L1) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/race_on.go#L1) / [`race_off.go:1`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/race_off.go#L1) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/race_off.go#L1) — build-tagged const is the cleanest approach; no change requested, just noting it's well done.
- [`testscript_gnoland.go:31`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/process_test.go#L31) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/process_test.go#L31) — `TestMain` now uses `nodeMaxLifespan` (120s). Worth a one-line comment explaining the bump is for `-race` builds; the previous 30s value will look surprisingly high to a reader who hasn't read the commit.

## Missing Tests

- **[concurrent safeWriter]** [`testscript_gnoland.go:740-758`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/testscript_gnoland.go#L740-L758) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/testscript_gnoland.go#L740-L758) — no unit test for `safeWriter.Write` / `String` under concurrent goroutines.
  <details><summary>details</summary>

  The struct is trivial (mutex + buffer), but a focused `-race` test with N goroutines hammering `Write` and one calling `String` would (a) document the invariant for future contributors and (b) fail loudly if someone later switches to a non-locking `io.Writer`. `event_race.txtar` covers the integration path but not the `safeWriter` contract itself.
  </details>

## Suggestions

- [`testscript_gnoland.go:766`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/integration/testscript_gnoland.go#L766) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/integration/testscript_gnoland.go#L766) — collapse `stdoutBuf` and `stderrBuf` into a single `safeWriter` (per thehowl), and `ts.Defer` a flush regardless of outcome. Matches the `process_test.go` pattern and addresses both warnings above in one change.
- [`gno.land/pkg/gnoland/events.go:11-15`](https://github.com/gnolang/gno/blob/163e1c4/gno.land/pkg/gnoland/events.go#L11-L15) · [↗](../../../../../.worktrees/gno-review-5477/gno.land/pkg/gnoland/events.go#L11-L15) — the absence-of-mutex invariant the new test verifies is non-obvious; document it on the `collector` type ("safe without a mutex because both `updateWith` and `getEvents` run sequentially on `receiveRoutine`"). The `event_race.txtar` header is the only place this is explained, and it's a place no one looks until the race detector fires.

## Questions for Author

- The `main / lint` CI failure on this PR looks like a tar extraction collision in the toolchain cache (`File exists` on `golang.org/toolchain@v0.0.1-go1.24.4`), not a code lint violation. Is this pre-existing infra noise on `master`, or should we rerun?
- Any plan to fold `tsLogWriter` removal into this PR, or leave it for a follow-up? It's a four-line cleanup and removes the only remaining tripwire that could regress fix #2.
