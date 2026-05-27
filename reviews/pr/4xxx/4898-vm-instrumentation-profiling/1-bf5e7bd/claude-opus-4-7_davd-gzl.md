# PR #4898: feat(gnovm): VM Instrumentation and Profiling System

URL: https://github.com/gnolang/gno/pull/4898
Author: notJoon | Base: master | Files: 22 | +4392 -12
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `bf5e7bd` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4898 bf5e7bd`

**Verdict: REQUEST CHANGES** — line-level cycle attribution is off-by-one (last op of every line credited to the next line), the inner-loop sample hook does an interface type-assertion + virtual call on every VM op when profiling is enabled regardless of mode, `lineStats.mu` is dead synchronization, `AllocationEvent.Kind` is in the schema but never populated, and the new 8 CLI flags ship without any doc/help-text in `docs/resources/gno-testing.md`.

## Summary

Adds an instrumentation-first profiling architecture: a new `instrumentation` package exposes `Sink`/`Capabilities` interfaces emitted by the VM (CPU/gas samples, allocation events, line samples); a new `profiler` package implements the sink with frame interning, incremental call-tree aggregation, per-function/per-line stats, and pprof-style output (text/toplist/calltree/JSON, plus interactive shell). The `gno test` command gains 8 flags (`-profile`, `-profile-type`, `-profile-format`, `-profile-sample-rate`, `-profile-output`, `-profile-stdout`, `-profile-interactive`, `-profile-line`) and wires `ProfileConfig` into `test.Test()` so a single profile spans multiple packages. The patch is ~4.4k LoC across 22 files; net additive except for naked-return cleanups in `machine.go`.

```
                ┌───────────────────────────────────┐
   VM op loop ──│ recordLineSampleIfNeeded()        │── per op
                │   nil-check on m.instrumentation  │
                │   capabilities iface assertion ── │── per op when profiling on
                │   WantsLineSamples() iface call ──│── per op
                │   captureLineSample() (allocates) │── per op when caps say yes
                └───────────────────────────────────┘
   incrCPU ──── m.profileState != nil ─── maybeEmitSample() ── sampleCounter % rate
                                                              ─ captureSampleContext (alloc)
   Alloc ────── instrumentation != nil ── OnAllocation ─── allocationStackInjector
                                                          ├─ captureFrameSnapshots (alloc)
                                                          └─ unsafe.Pointer(m) machineID
```

## Glossary

- `Sink`: the instrumentation interface the VM calls into (`OnSample`/`OnAllocation`/`OnLineSample`).
- `Capabilities`: optional interface (`WantsSamples`/`WantsAllocations`/`WantsLineSamples`) sinks implement so the VM can skip work.
- `ProfileConfig`: lifecycle holder in `gnovm/pkg/test`; idempotent `Start`/`Stop`, owns the `Profiler` and `SinkAdapter`.
- `SinkAdapter`: bridges `instrumentation.Sink` calls into the `profiler.Profiler` API.
- `baseline` / `prevCycles`: per-machine cumulative-counter snapshot used to derive per-sample deltas.
- `allocationStackInjector`: machine-scoped sink wrapper that captures `m.Frames` when an `AllocationEvent` arrives stack-less.

## Fix

Before: the VM exposed no extension hook for profiling and the only profiler in tree was the (unused) older path. After: every VM op-dispatch (`Run` loop) calls `recordLineSampleIfNeeded()`; every `incrCPU` conditionally calls `maybeEmitSample()`; every `Allocator.Allocate` conditionally calls `sink.OnAllocation`. Load-bearing gate is `m.profileState != nil` for samples and `m.instrumentation != nil` for line samples and allocations; when both are nil the cost reduces to a single nil-check per call site. `ProfileConfig.Start`/`Stop` are idempotent (so a CLI driver can own the lifecycle across multiple `Test()` calls) and `attachMachine` installs the sink per-machine. See [`gnovm/pkg/gnolang/machine.go#L1131-L1139`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine.go#L1131-L1139) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine.go#L1131-L1139), [`machine.go#L1313`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine.go#L1313) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine.go#L1313), [`gnovm/pkg/gnolang/machine_profiling.go#L230-L334`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine_profiling.go#L230-L334) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine_profiling.go#L230-L334), [`gnovm/pkg/test/test.go#L188-L274`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/test/test.go#L188-L274) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/test/test.go#L188-L274).

## Critical (must fix)

- **[line samples credit the wrong line]** [`gnovm/pkg/gnolang/machine.go:1313`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine.go#L1313) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine.go#L1313) — `recordLineSampleIfNeeded` is called BEFORE the op's `incrCPU`, so each sample carries the cumulative cycle count from before this op ran; the delta computed in `RecordLineSample` is therefore the cost of the PREVIOUS op, attributed to the CURRENT op's line.
  <details><summary>details</summary>

  **Shape:** for an op sequence `(line=10, cost=25), (line=10, cost=5), (line=11, cost=10), (line=11, cost=3)`, samples fire at the head of each op with `m.Cycles ∈ {0, 25, 30, 40}`. Deltas recorded: line10 += 0 (first sample, baseline=0), line10 += 25, line11 += 5, line11 += 10. Total line10 = 25, total line11 = 15. Real costs: line10 = 30, line11 = 13. The last op of line 10 (5 cycles) leaks to line 11; the last op of line 11 (3 cycles) never gets sampled (no next op to take a sample).

  **Mechanism:** in [`gnovm/pkg/gnolang/machine.go#L1298-L1314`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine.go#L1298-L1314) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine.go#L1298-L1314) the order is `PopOp → recordLineSampleIfNeeded → switch{case OpExec: incrCPU(...); doOpExec}`. `incrCPU` mutates `m.Cycles` after the line sample is taken. `RecordLineSample` at [`gnovm/pkg/profiler/profiler.go#L885-L920`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L885-L920) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L885-L920) computes `deltaCycles = cycles - baseline.prevLineCycles` and credits that delta to the line carried in the sample — but `cycles` is the value at the start of the current op, so `delta` describes the previous op.

  **Why it matters:** the headline use case of the PR is "find expensive lines in user code." For tight functions where ops on different lines have different costs (e.g. a heavy `OpExec` on the last line of a function before returning), the attribution shifts cost off the actual hot line. The interactive `list` view will mislead users about which line to optimize. For CPU profiling in `RecordSample` (driven by `maybeEmitSample` AFTER `incrCPU`), attribution is correct — only the line-level path is broken.

  **Fix:** move `recordLineSampleIfNeeded()` to AFTER the op dispatch (i.e. after the `switch`/`incrCPU` for the case), OR capture the line BEFORE the op runs but emit the sample AFTER, computing the delta correctly. Simplest: keep the call at the bottom of the loop iteration so `m.Cycles` already reflects the op that just ran on the line that was just executed.
  </details>

## Warnings (should fix)

- **[per-op interface assertion in hot path]** [`gnovm/pkg/gnolang/machine_profiling.go:294-298`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine_profiling.go#L294-L298) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine_profiling.go#L294-L298) — when profiling is enabled but line profiling is off (CPU or gas profile without `-profile-line`), every VM op pays `instrumentationCapabilities()` (type-assertion) + `caps.WantsLineSamples()` (interface call) before short-circuiting.
  <details><summary>details</summary>

  The cheap nil-check claim ("zero additional work (only a nil-check)") holds when profiling is disabled. When it's enabled, `m.instrumentation != nil`, so the function falls through to the capabilities check on every op. Interface assertions are not free in a per-op loop — the VM dispatches millions of ops per second. The cleaner pattern is to cache `wantsLineSamples bool` on the machine when the sink is installed (one assignment in `refreshInstrumentationSink`) and read that boolean field, avoiding interface vtable lookups inside the loop. Same applies to `maybeEmitSample` at [`machine_profiling.go:230-249`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine_profiling.go#L230-L249) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine_profiling.go#L230-L249). Fix: store cached capability booleans on `profilingState` once at sink install, branch on those.
  </details>

- **[lineStats.mu is dead synchronization]** [`gnovm/pkg/profiler/line_level.go:19-55`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/line_level.go#L19-L55) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/line_level.go#L19-L55) — `lineStats` embeds `lineStat` + a `sync.Mutex`, but every WRITE site (`p.lineSamples[file][line].count++` etc., e.g. [`profiler.go:451-457`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L451-L457) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L451-L457), [`profiler.go:916-917`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L916-L917) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L916-L917)) holds only `p.mu` and never `lineStats.mu`. The accessor methods (`Count()`, `Cycles()`, `Allocations()`, `AllocBytes()`) lock `lineStats.mu` but read fields the writers never lock — so the mutex provides no protection while creating the impression that the accessors are thread-safe.
  <details><summary>details</summary>

  Either drop the inner mutex (the project-wide invariant is "everything goes through `p.mu`"), or push the mutex down by ensuring all writers also lock it. The current state is the worst of both worlds: cost of lock-acquire on read, false sense of safety. Fix: remove `mu sync.Mutex` from `lineStats` and the locking from the getter methods.
  </details>

- **[AllocationEvent.Kind unused by emitter]** [`gnovm/pkg/gnolang/alloc.go:177-184`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/alloc.go#L177-L184) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/alloc.go#L177-L184) — the schema field `Kind string` is defined in [`gnovm/pkg/instrumentation/instrumentation.go#L36`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/instrumentation/instrumentation.go#L36) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/instrumentation/instrumentation.go#L36) but the `Allocator.Allocate` site that emits events sets only `Bytes` and `Objects`. Every callsite in `alloc.go` (`AllocateString`, `AllocatePointer`, `AllocateStruct`, `AllocateMap`, …) routes through `Allocate(size)`, losing the allocation kind.
  <details><summary>details</summary>

  dongwon8247's review comment on Dec 16 specifically asked for distinguishing pointer-bearing vs. primitive allocations for GC overhead analysis. As shipped, downstream consumers cannot filter or aggregate by kind — the field is dead. Either populate `Kind` from each caller (cheapest fix: take a `kind string` parameter on a new `allocateWithKind`, route through it, default callers to `""`) or delete the field from the schema until it's wired. Recommend the former since the schema is what external sinks will be coded against.
  </details>

- **[machine identity via `unsafe.Pointer(m)`]** [`gnovm/pkg/gnolang/machine_profiling.go:200, 260, 332`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/gnolang/machine_profiling.go#L200) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/gnolang/machine_profiling.go#L200) — using the Machine's address as `MachineID` means a freed-and-reallocated Machine can collide with a stale baseline entry. The profiler handles the in-flight reuse case (the "currentCycles <= prevCycles" branch resets the delta, see [`profiler.go:407-414`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L407-L414) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L407-L414)) but the `baselines` map grows monotonically per address, never collected; under a long-running session that creates many short-lived machines, it leaks.
  <details><summary>details</summary>

  Two issues: (a) `unsafe.Pointer(m)` is a `uintptr` cast that isn't tracked by GC — fine since Go doesn't move objects, but the comment "best-effort identifier" understates the staleness risk; (b) `p.baselines map[uintptr]*sampleBaseline` is only ever inserted into (via `ensureBaseline`), never expired. Across thousands of machines this is unbounded. Fix: either expose a `Machine.Identity()` that uses a monotonically increasing counter (collision-free), or expose a `RetireMachine(id)` hook the test runner calls when a machine is `Release`d to evict the baseline entry.
  </details>

- **[`packageFromFunction` heuristic mislabels nested-type methods]** [`gnovm/pkg/profiler/profiler.go:1187-1219`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L1187-L1219) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L1187-L1219) — for `funcName = "pkg.Type.SubType.Method"` (4 dot-separated parts), the function falls through the `len(parts) == 3 && uppercase` branch and returns `strings.LastIndex(funcName, ".")`, yielding package `"pkg.Type.SubType"` instead of `"pkg"`. Similarly `gno.land/p/demo.Type.Method` (the realistic 3-part case where parts[1] is `land/p/demo` — lowercase 'l') falls back to LastIndex, returning `"gno.land/p/demo.Type"`.
  <details><summary>details</summary>

  The heuristic only correctly handles `pkg.Type.Method` where `parts[1]` happens to start with an uppercase letter, and `pkg.(*Type).Method`. Anything else lands in the LastIndex branch and mis-extracts the package. The downstream consequence is `canonicalFilePath(file, funcName)` produces wrong paths and the `list` view's source lookup fails. Fix: use the `PkgPath` carried on `ProfileLocation`/`FrameSnapshot` instead of re-parsing the function name string. The frame already has it; reach for the data, not the regex.
  </details>

- **[silent fallback on bad profile-type/format strings]** [`gnovm/pkg/test/test.go:127-148`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/test/test.go#L127-L148) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/test/test.go#L127-L148) — `ProfileConfig.GetProfileType()` and `getProfileFormat()` are case-sensitive switches with a silent `default` fallback. A user invoking `-profile-type=CPU` (uppercase) gets CPU because the default is CPU; `-profile-type=cpuu` (typo) also gets CPU. There is no error path, no warning. Same for `-profile-format`.
  <details><summary>details</summary>

  Profiling is opt-in and expensive; failing silently to the default is a debug-eating footgun. Fix: validate on `ProfileConfig.Start()` and return an error for unrecognized values. Also normalize to lowercase before comparing so `-profile-type=CPU` works as the user expects.
  </details>

- **[default `profile.out` clobbers in parallel runs]** [`gnovm/cmd/gno/test.go:219-223`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/cmd/gno/test.go#L219-L223) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/cmd/gno/test.go#L219-L223) — the default output filename is the literal string `"profile.out"`. Multiple concurrent `gno test` invocations in the same cwd overwrite each other; reruns silently overwrite the previous artifact.
  <details><summary>details</summary>

  Minimal fix: append a PID or timestamp to the default name; or refuse to write when the file exists unless `-profile-output` is given explicitly. Either way the silent overwrite is surprising for a tool that takes minutes to produce its output.
  </details>

- **[isFilteredFunction over-aggressive]** [`gnovm/pkg/profiler/profiler.go:1165-1169`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L1165-L1169) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L1165-L1169) — the filter is `strings.HasPrefix(name, "testing.")`. Any user realm whose fully-qualified function name begins with `testing.` (legal in Gno) would be excluded from stats. Realistic risk is low (Gno realm paths are usually `gno.land/...`), but the filter is unanchored to package boundary.
  <details><summary>details</summary>

  Use the frame's `PkgPath` field directly: `frame.PkgPath == "testing"`. The frame already carries it; no string-prefix gymnastics needed. Fix at the two callsites in `updateFunctionStats` and `updateFunctionLineStats`.
  </details>

- **[`Identity()` not part of `MachineInfo` interface]** [`gnovm/pkg/profiler/types.go:4-8`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/types.go#L4-L8) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/types.go#L4-L8), [`profiler.go:1233-1246`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L1233-L1246) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L1233-L1246) — `machineKey` does a type assertion to an unexported `identifier` interface on every `RecordSample`/`RecordLineSample` call. Both real call sites already implement it. Fix: lift `Identity() uintptr` into the `MachineInfo` interface so it's a direct call.

- **[unbounded growth of frame and function tables]** [`gnovm/pkg/profiler/profiler.go:30-58, 269`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L30-L58) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L30-L58) — `frameStore` interns every distinct frame; `funcStats` keys every distinct function; `lineSamples` keys every (file,line). For long-running test suites or REPL-style profiling this is unbounded. There is no cap, eviction, or watermark. Fix: at minimum document the lifetime expectation (one profile session, callers must `StopProfiling` to release); ideally cap and emit a warning when caps are hit.

- **[docs not updated for 8 new flags]** [`docs/resources/gno-testing.md`](https://github.com/gnolang/gno/blob/bf5e7bd/docs/resources/gno-testing.md) · [↗](../../../../../.worktrees/gno-review-4898/docs/resources/gno-testing.md) — the file lists `-v`, `-debug`, etc., but does not mention `-profile`/`-profile-type`/`-profile-format`/`-profile-sample-rate`/`-profile-output`/`-profile-stdout`/`-profile-interactive`/`-profile-line`. A user-facing feature this large with zero documentation will not be discoverable. Fix: add a "Profiling tests" subsection with the 8 flags, an example invocation, and a screenshot/example of the toplist/calltree output.

## Nits

- [`gnovm/pkg/profiler/list.go:117-135`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/list.go#L117-L135) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/list.go#L117-L135) — `if startLine == -1 { startLine = 1; endLine = len(lines) }` is dead code: `findFunctionBounds` returns either `(positive, positive, nil)` or `(-1, -1, errors.New(...))`; the `err != nil` check 5 lines above already exits.
- [`gnovm/pkg/profiler/line_level.go:117-130`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/line_level.go#L117-L130) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/line_level.go#L117-L130) — header uses `%-8s` (width 8) but body uses `%7d` (width 7); columns misalign.
- [`gnovm/pkg/test/test.go:308-318`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/test/test.go#L308-L318) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/test/test.go#L308-L318) — `defer file.Close()` ignores the close error in `DefaultProfileWriter.WriteProfile`; if `Close` fails, the partial profile is silently truncated.
- [`gnovm/pkg/profiler/profiler.go:279-313`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L279-L313) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L279-L313) — `NewProfiler(params ...any)` with positional `any` arguments and runtime type-switching is brittle vs. a typed `NewProfilerWithOptions(opts Options)`. The variadic API has no compiler help when callers reorder arguments.
- [`gnovm/pkg/test/test.go:160-175`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/test/test.go#L160-L175) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/test/test.go#L160-L175) — `GetSampleRate` returns `100` for CPU/Gas but `profiler.NewProfiler` defaults `1000` in [`profiler.go:283`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L283) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L283). Both code paths claim "default" — pick one and centralize.
- [`gnovm/pkg/profiler/profiler.go:163-184`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L163-L184) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L163-L184) — `ProfileGoroutine` is declared but unused (`TODO: not supported yet`); shipping a public enum value users can pass that does nothing is confusing. Either implement or omit.
- 26 commits, ~half are `fix:` against earlier commits in the same PR. Squashing before merge would clean up `git log` (style choice — Gno's history is rebase-merge friendly per the existing log).

## Missing Tests

- **[off-by-one attribution]** [`gnovm/pkg/profiler/profiler_test.go`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler_test.go) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler_test.go) — no test asserts cycle attribution at a line-transition boundary. A test that runs 2 ops on line A (costs `a1`, `a2`) and 2 ops on line B (costs `b1`, `b2`) and asserts `line_A_cycles == a1+a2` and `line_B_cycles == b1+b2` would catch the Critical above.
- **[deep call graphs]** no test exercises a stack with > ~5 frames; `convertCallTreeNode` recursion has no depth limit. Adversarial test: 1000-deep recursion crashing the renderer.
- **[shell `list` command without source]** [`gnovm/cmd/gno/profile_shell_test.go`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/cmd/gno/profile_shell_test.go) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/cmd/gno/profile_shell_test.go) — covers `help/exit/text/json/clear/unknown` but skips `list <func>` and `top <N>`. Both have non-trivial branching (limit validation, source-not-found fallback in `writeLineStatsOnly`).
- **[output file failures]** no test covers `os.Create(pc.OutputFile)` returning an error (read-only dir, permission denied). The function returns the error, but the surrounding lifecycle never asserts the file is created.
- **[concurrent machines]** the `baselines map` per-machine logic is tested (`TestProfilerSeparatesBaselinesPerMachine`), but with synthetic identities. No integration test runs two real machines concurrently against one sink.

## Suggestions

- [`gnovm/pkg/instrumentation/instrumentation.go:57-69`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/instrumentation/instrumentation.go#L57-L69) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/instrumentation/instrumentation.go#L57-L69) — make `Capabilities` part of `Sink` rather than an optional companion interface. The current "type-assert to detect" pattern is what motivates the hot-path warning above; baking it in removes the assertion and forces every sink to declare intent.
- [`gnovm/pkg/profiler/profiler.go:32-58`](https://github.com/gnolang/gno/blob/bf5e7bd/gnovm/pkg/profiler/profiler.go#L32-L58) · [↗](../../../../../.worktrees/gno-review-4898/gnovm/pkg/profiler/profiler.go#L32-L58) — `frameStore` could expose a `framesSnapshot()` zero-copy view (slice over the underlying `frames`) for callers that don't mutate; current implementation copies on every `StopProfiling`.
- Consider exporting `pprof` protobuf output as an additional format. The existing JSON shape is custom; `go tool pprof` users would benefit from a `-profile-format=pprof` that emits compatible profiles. The schema is already runtime/pprof-shaped per the `instrumentation` package comments.
- Pull profiling configuration parsing into a small `ParseFlags(flag.FlagSet) (*ProfileConfig, error)` so future tools (e.g. `gno run -profile`) don't duplicate the flag block.

## Questions for Author

- Was the line-attribution off-by-one intentional? Looking at the `incrCPU → maybeEmitSample` ordering (correct) vs. `loop-top → recordLineSampleIfNeeded → switch → incrCPU` (off by one for lines), the asymmetry looks accidental. If intentional, the rationale belongs in a comment on `recordLineSampleIfNeeded`.
- Why two defaults for `SampleRate` (100 in `ProfileConfig.GetSampleRate`, 1000 in `NewProfiler`)?
- The PR description claims "zero additional work (only a nil-check)" when profiling is disabled. Did you benchmark VM throughput with and without the patch on a representative workload (e.g. `gno test ./gnovm/tests/...`)? A go-bench comparison would close the loop on the perf claim.
- dongwon8247's Dec 16 comment asks for distinguishing pointer-bearing vs. primitive allocations for GC analysis. Is `AllocationEvent.Kind` the intended hook, and if so when do you plan to populate it from `Allocator`?
- Was `gnoenv.RootDir()` chosen for `TestProfilingSpansMultiplePackages` because the previous empty rootDir produced an unrelated test failure? The fallback chain in `gnoenv.GuessRootDir` involves a `go list` exec which is slow — `t.TempDir()` would be cheaper if the test only needs a writable root.
