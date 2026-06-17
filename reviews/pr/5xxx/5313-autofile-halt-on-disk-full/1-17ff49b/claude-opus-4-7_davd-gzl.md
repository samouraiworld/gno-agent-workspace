# PR #5313: fix(autofile): halt writes on disk space exhaustion with auto-recovery

URL: https://github.com/gnolang/gno/pull/5313
Author: davd-gzl | Base: master | Files: 4 | +243 -66
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `17ff49b` (stale — +2 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5313 17ff49b`

**Verdict: REQUEST CHANGES** — direction is right, but the reactive `isErrNoSpace` branch is effectively unreachable through `bufio.Writer`, `FlushAndSync` is not gated by `halted` so disk-full failures still surface there unsignaled, the mempool WAL's split-write pattern can interleave a halt between `tx` and `\n` corrupting the line, and the missing ADR violates the workspace AGENTS.md rule for non-trivial AI-assisted changes.

## Summary

Adds disk-space protection to `tm2/pkg/autofile.Group`. Before each `Write`/`WriteLine` (throttled to every 100 calls), `statfs` reads `Bavail * Bsize` for `g.Dir`; below a hard-coded 16 MB the group sets `halted=true` and returns `ErrDiskSpaceUnavailable`. Subsequent writes re-check on every call; when space recovers the group auto-resumes. Below 64 MB (`4 × limit`) a warning is logged. Windows/wasm are stubbed via a `^uint64(0)` "unsupported" sentinel. The motivating site is the consensus WAL on a node with a filling disk; today such writes silently succeed into a `bufio` buffer and the operator only finds out when `FlushAndSync` panics. After this PR, the validator halts ~one block earlier and with a clearer error class, but consensus still panics via the existing `cs.wal.WriteSync` paths in `bft/consensus/state.go` (lines 643-646 and 1365-1367) — this is not a graceful availability improvement, it just shifts the panic site.

## Glossary

- `Group` — `tm2/pkg/autofile` rotating-file writer; backs the consensus WAL, mempool WAL, and file event store.
- `bufio.Writer` (`headBuf`) — 40 KiB write buffer in front of the underlying `AutoFile`; defers actual disk I/O until `Flush`.
- `ensureDiskSpace` — new pre-write gate; runs `statfs` every 100 writes or always when halted.
- `Bavail` / `Bsize` — `syscall.Statfs_t` fields: blocks available to unprivileged users, block size in bytes.

## Fix

Before: `Write`/`WriteLine` wrote into `headBuf` and returned its (almost always nil) error; ENOSPC surfaced at `FlushAndSync` / `rotateFile` and panicked the WAL. After: a per-Group `mtx`-guarded throttle counts writes; every 100 calls (or every call while `halted`) the group `statfs(g.Dir)`s and rejects further writes with `ErrDiskSpaceUnavailable` once `Bavail*Bsize < 16 MiB`. The load-bearing constraint is that the syscall is sufficiently rare to avoid latency cost on the hot path; the load-bearing failure mode is that `halted` is only set in `writeBytes` and never by the actual `FlushAndSync` / `rotateFile` paths that produce real ENOSPC errors today. See [`group.go:245-268`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L245-L268) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L245-L268) and [`group.go:289-348`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L289-L348) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L289-L348).

## Critical (must fix)

- **[reactive ENOSPC branch is effectively dead code]** [`tm2/pkg/autofile/group.go:250-257`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L250-L257) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L250-L257) — `g.headBuf.Write(p)` returns ENOSPC at most once per 40 KiB of writes; for typical WAL traffic (~300 B per consensus message) the buffer absorbs 100+ writes before any flush, so `isErrNoSpace(err)` here virtually never fires.
  <details><summary>details</summary>

  `headBuf` is `bufio.NewWriterSize(head, 4096*10)` ([`group.go:106`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L106) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L106)). `bufio.Writer.Write` only calls the underlying `Write` when the buffer fills, so a small payload (the common WAL/event-store case) returns nil regardless of disk state. The ENOSPC the operator will actually hit comes out of `g.headBuf.Flush()` in [`FlushAndSync`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L279-L287) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L279-L287) and [`rotateFile`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L407-L431) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L407-L431), neither of which calls `g.halt()` or returns `ErrDiskSpaceUnavailable`. Concretely: today the consensus WAL's `WriteSync` flushes immediately ([`wal.go:238`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/wal/wal.go#L238) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/wal/wal.go#L238)) and the file event store flushes after every `Append` ([`file.go:87`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/state/eventstore/file/file.go#L87) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/state/eventstore/file/file.go#L87)) — those `FlushAndSync` calls return a raw `&os.PathError{Err: syscall.ENOSPC}`, the consensus path panics with the unchanged "Failed to write … to consensus wal" message, and `halted` is never set. The PR's claim of "halt writes on disk space exhaustion with auto-recovery" only holds for the proactive 100-write `statfs` gate; the reactive path is a comment-level promise the code doesn't keep. Fix: either move the `isErrNoSpace`/`halt` handling into `FlushAndSync` (and the panicking branches of `rotateFile`), or drop the dead reactive branch and explicitly document that `FlushAndSync` is the load-bearing ENOSPC site.
  </details>

- **[mempool WAL split-write can corrupt line on halt boundary]** [`tm2/pkg/bft/mempool/clist_mempool.go:270-277`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) — mempool issues `Write(tx)` then `Write("\n")` as two separate calls; the 100th write of the pair can trip the new check, halt the group, and reject the `\n` while the tx bytes already entered the buffer.
  <details><summary>details</summary>

  The two-call sequence pre-dates this PR but its failure mode changes here. Previously, both calls always succeeded against `bufio.Writer` and the eventual `FlushAndSync` would either succeed or fail atomically. Now, if `writesSinceLastCheck` is at 99 when the tx Write enters, the counter goes to 100, the `statfs` runs, finds <16 MiB, sets `halted=true`, returns `ErrDiskSpaceUnavailable` — but `g.headBuf.Write(p)` was never called (the gate returns first, see [`group.go:246-248`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L246-L248) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L246-L248)), so the tx bytes are not in the buffer. Fine. Worse: if the counter hits 100 on the `\n` Write *after* tx Write succeeded with counter=99, you get an unterminated tx line followed by `ErrDiskSpaceUnavailable` on the `\n`. The mempool logs and continues ([`clist_mempool.go:272`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/mempool/clist_mempool.go#L272) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L272), `// TODO: Notify administrators when WAL fails`), so the next successful tx will be concatenated onto the previous tx's bytes with no separator, corrupting the WAL stream on replay. Fix: convert mempool to a single `WriteLine(string(tx))` call (one `writeBytes`, atomic gate), or have `Write` accept a slice-of-slices to enforce all-or-nothing semantics under a single gate.
  </details>

## Warnings (should fix)

- **[`FlushAndSync` is not gated by `halted` and doesn't trip the halt]** [`tm2/pkg/autofile/group.go:279-287`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L279-L287) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L279-L287) — a halted group's `FlushAndSync` still attempts to write the buffered bytes to a full disk, and the resulting ENOSPC just bubbles up unwrapped.
  <details><summary>details</summary>

  Once `ensureDiskSpace` halts the group, any `headBuf` bytes from earlier successful writes are still sitting in memory. `FlushAndSync` will call `g.headBuf.Flush()`, which calls `af.file.Write(b)`, which returns `&os.PathError{Err: syscall.ENOSPC}`. The error returned to the caller (mempool, file event store, WAL) is the raw `*os.PathError` — no `ErrDiskSpaceUnavailable` wrap, no `g.halt()` call (the group was already halted, but in principle this is the first time the *real* disk-full signal arrives). Callers that don't `errors.Is(err, ErrDiskSpaceUnavailable)` lose the chance to disambiguate. Worse: if `ensureDiskSpace` returned nil because `availableDiskSpace` failed (the "fail-open" path at [`group.go:307-311`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L307-L311) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L307-L311)), the FlushAndSync ENOSPC is the only signal anything is wrong, and the group never halts. Fix: in `FlushAndSync`, call `isErrNoSpace` on the returned error and invoke `g.halt()` + wrap in `ErrDiskSpaceUnavailable` like `writeBytes` does.
  </details>

- **[`rotateFile` panics the process on ENOSPC]** [`tm2/pkg/autofile/group.go:407-431`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L407-L431) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L407-L431) — `g.headBuf.Flush()` and `g.Head.Sync()` inside `rotateFile` panic on any error; ENOSPC during rotation crashes the process before halt can engage.
  <details><summary>details</summary>

  Rotation runs inside `writeBytes` after a successful append ([`group.go:264-266`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L264-L266) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L264-L266)). The default head size limit is 10 MB; rotation is rare but happens every ~10 MB of consensus messages. If the disk is nearly full but above the 16 MB halt threshold (say 20 MB free), the ensureDiskSpace check returns nil → the append succeeds → rotation fires → `g.headBuf.Flush()` returns ENOSPC because the flush plus rotation overhead exceeded available space → `panic(err)` at [`group.go:411`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L411) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L411). The PR positions itself as "halt cleanly" but rotation still panics. Either widen the halt margin (and document why 16 MB), or convert rotation panics into halt-and-return-error.
  </details>

- **[per-validator non-uniform halt on different filesystems]** [`tm2/pkg/autofile/group.go:295-348`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L295-L348) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L295-L348) — the halt is a per-node liveness gate, not a consensus-state change; a validator with less free disk than its peers will halt (panic via `WriteSync`) while others continue, shrinking the validator set without any chain-level signal.
  <details><summary>details</summary>

  Not a determinism bug (chain state is unaffected) but a single-validator availability hazard. The 16 MB threshold is hard-coded; on a 1 TiB SSD it's 0.0015 % of capacity, on a 10 GB container volume it's 0.16 %. The choice of 16 MB is not justified in the diff (the `// 16 MB` comment at [`group.go:27-28`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L27-L28) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L27-L28) restates the value, not the reasoning). For a chain with N validators, the validator with the smallest free-disk margin is the first to halt; under correlated disk pressure (e.g., a snapshot rollout), >1/3 can halt together and stall consensus. Fix: surface the threshold as a config option (`GroupMinDiskSpaceLimit` was removed in commit `da444579` — the rationale "make it a fixed constant" trades off operator control for simplicity, which is the wrong direction for a liveness knob).
  </details>

- **[missing ADR for AI-assisted non-trivial change]** [workspace AGENTS.md](https://github.com/gnolang/gno/blob/17ff49b/AGENTS.md#L82-L97) · [↗](../../../../../.worktrees/gno-review-5313/AGENTS.md#L82-L97) — the gno-repo `AGENTS.md` requires an ADR in `tm2/adr/pr5313_*.md` for every non-trivial AI-assisted PR; this PR is non-trivial and lands without one.
  <details><summary>details</summary>

  The 9-commit history (multiple refactor passes, the back-and-forth around `GroupMinDiskSpaceLimit`, the Windows/wasm stub merge, the `Halted()`/`Resume()` add-then-remove) is the kind of design churn an ADR exists to capture so a future maintainer doesn't re-tread the same arguments. The tm2/adr/ directory is empty (`.gitkeep` only), so this would be the first one — that doesn't excuse skipping it, it's a chance to seed the convention. Fix: add `tm2/adr/pr5313_autofile_disk_space_halt.md` covering: why 16 MB, why a fixed constant (no operator override), why the warning multiplier is 4×, why the 100-write throttle (vs. time-based or buffer-fill-based), why `Halted()`/`Resume()` were removed, and why the design intentionally leaves `FlushAndSync` unghaltable.
  </details>

- **[warning log fires on every check below threshold; noisy under sustained pressure]** [`tm2/pkg/autofile/group.go:338-346`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L338-L346) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L338-L346) — once `avail < 64 MiB` and the disk stays there, `g.Logger.Warn` is emitted every 100 writes; for a consensus WAL that's ~5-30 warnings/second.
  <details><summary>details</summary>

  No rate-limiting on the warning. The 100-write throttle on `statfs` does not throttle the log emission — each check that finds low space logs. Under a slow disk fill, this floods the operator's log without adding signal beyond the first warning. Fix: gate the warning on a state transition (only log when crossing into/out of the warning band), or on a coarse time interval (e.g. once per minute).
  </details>

- **[fail-open on `availableDiskSpace` error masks real problems]** [`tm2/pkg/autofile/group.go:306-312`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L306-L312) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L306-L312) — if `syscall.Statfs` fails (permissions changed, fs unmounted, NFS hang) the code logs at ERROR and returns nil; combined with the "real ENOSPC at FlushAndSync isn't gated" warning above, the operator gets two unrelated errors and no halt.
  <details><summary>details</summary>

  The comment justifies the fail-open as "unsupported platform" handling, but the supported-platform `availableDiskSpace` errors are distinct: actual `Statfs` failures (EACCES, EIO, ENOTDIR) on a Linux node mean something genuinely wrong, not "this platform doesn't support it." The unsupported-platform case is already handled separately via the `diskSpaceUnsupported` sentinel ([`group.go:313-316`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L313-L316) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L313-L316)) — the `err != nil` branch above it is exclusively for real syscall failures on supported platforms. Logging once and continuing turns a hard signal into a soft one. Fix: count consecutive `Statfs` failures; halt after N (e.g. 10) so the operator gets a clear halt-on-fs-error signal instead of a recurring log line.
  </details>

- **[`Halted()`/`Resume()` removal in commit `e4d2c97` is justified by "no production callers" — but the change is intended to enable operator response]** ([commit e4d2c97](https://github.com/gnolang/gno/pull/5313/commits/e4d2c97b3b95a715e1cfc78651dcf7c777bd2eaf)) — removing the public introspection/manual-resume API leaves operators no way to observe halt state from outside, except by attempting a write and parsing `ErrDiskSpaceUnavailable`.
  <details><summary>details</summary>

  The commit message argues: (a) auto-recovery makes `Resume()` redundant; (b) consensus WAL panics on disk-full before `Halted()` could be polled. (a) is fine — auto-recovery is the right default. (b) is misleading: the file event store ([`bft/state/eventstore/file/file.go:82-89`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/state/eventstore/file/file.go#L82-L89) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/state/eventstore/file/file.go#L82-L89)) returns the error to its caller without panicking, and the mempool WAL ([`bft/mempool/clist_mempool.go:270-277`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277)) logs and continues. Both of those paths could benefit from a `Halted() bool` getter to surface state in a `/status` endpoint or metric. Fix: keep `Halted() bool` as a read-only getter (the cost is one mutex-protected field read), drop only `Resume()`.
  </details>

## Nits

- [`tm2/pkg/autofile/group.go:32`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L32) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L32) — `diskSpaceWarningThreshold = 4` reads as "the threshold is 4" until you reach the doc-comment; name it `diskSpaceWarningMultiplier` to match its arithmetic role.
- [`tm2/pkg/autofile/group.go:45`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L45) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L45) — `diskSpaceUnsupported = ^uint64(0)` ties the sentinel to the return type; a constant `math.MaxUint64` (already a Go builtin) reads clearer.
- [`tm2/pkg/autofile/group.go:86`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L86) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L86) — `writesSinceLastCheck int` is an unbounded counter incremented on every Write; on a long-running validator at sustained low TPS this could overflow in theory but practically never. Cosmetic.
- [`tm2/pkg/autofile/diskspace_unix.go:18-20`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/diskspace_unix.go#L18-L20) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/diskspace_unix.go#L18-L20) — `if stat.Bsize <= 0` returns a formatted error; the test path is unreachable on any real filesystem and the defensive guard noise outweighs its value. Drop or comment why.
- [`tm2/pkg/autofile/group.go:354-362`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L354-L362) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L354-L362) — `halt()` checks `if !g.halted` before logging to avoid log spam, but the caller in `writeBytes` ([`group.go:253`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L253) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L253)) only reaches `halt()` once per write — the idempotency guard is unnecessary unless `ensureDiskSpace`'s halt path can re-fire after `halted=true` (it can: every halted write rechecks). Keep the guard, but the comment should say so.

## Missing Tests

- **[no test exercises the actual halt path]** [`tm2/pkg/autofile/group_test.go:217-265`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group_test.go#L217-L265) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group_test.go#L217-L265) — the two new tests cover auto-recovery (which requires a *succeeding* statfs) and throttling (which never exhausts space), so the codepath from `avail < limit` to `ErrDiskSpaceUnavailable` is untested.
  <details><summary>details</summary>

  This is the core promise of the PR and codecov reports 43 % patch coverage. A reasonable test would inject a stub `availableDiskSpace` via an unexported package-level function variable, then verify: (a) returning `1` halts and returns `ErrDiskSpaceUnavailable`; (b) the halted state surfaces through subsequent writes; (c) raising the return value above the limit auto-resumes; (d) `errors.Is(err, ErrDiskSpaceUnavailable)` works. Without these, the only thing the test suite guarantees is "the code compiles and doesn't break on a healthy disk."
  </details>

- **[no test for the boundary case in mempool's two-Write pattern]** [`tm2/pkg/bft/mempool/clist_mempool.go:270-277`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) — the failure mode flagged in the second Critical above has no regression test.
  <details><summary>details</summary>

  With the injected-stub harness from the prior bullet, a test could write 99 successful messages, then have the stub start returning a low value, then call `mem.wal.Write(tx)` and `mem.wal.Write("\n")` separately and verify the WAL stream is not torn.
  </details>

- **[`FlushAndSync` ENOSPC behavior is not tested]** [`tm2/pkg/autofile/group_test.go`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group_test.go) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group_test.go) — no test verifies what happens when the underlying file Flush returns ENOSPC.
  <details><summary>details</summary>

  A test substituting `g.headBuf` for a writer that returns `syscall.ENOSPC` from `Write` would document the current (un-halt-aware) behavior and surface the gap raised in the first Warning above.
  </details>

## Suggestions

- [`tm2/pkg/autofile/group.go:96-125`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L96-L125) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L96-L125) — consider passing a `diskSpaceProvider func(string) (uint64, error)` option to `OpenGroup` so tests can inject without unsafe global state. The functional-options pattern is already in use (`GroupHeadSizeLimit`, `GroupTotalSizeLimit`).
  <details><summary>details</summary>

  This is the natural seam for the missing tests above and the reinstatement of `GroupMinDiskSpaceLimit` (currently a constant). Production callers default to the real `availableDiskSpace`; tests inject a stub. Keeps the package free of build-tag-gated test hooks.
  </details>

- [`tm2/pkg/autofile/group.go:289-348`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/autofile/group.go#L289-L348) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L289-L348) — consider folding the gate into `FlushAndSync` and `rotateFile` so all three load-bearing I/O paths share one disk-space check.
  <details><summary>details</summary>

  Today the gate runs only on `writeBytes`. Lifting it into a `func (g *Group) preflightDiskCheck() error` called from `writeBytes`, `FlushAndSync`, and `rotateFile` would close the Critical (reactive ENOSPC) and the Warning (FlushAndSync) with one change.
  </details>

## Questions for Author

- Why a fixed 16 MB constant instead of either (a) a config option, or (b) a percentage of headSize + totalSize budget? The PR removed `GroupMinDiskSpaceLimit` in commit `da444579` — what's the operational scenario where a fixed value across all deployments (CI, devnet, mainnet, embedded) is preferable?
- Was the asymmetry of the change considered: `WriteLine` becomes atomic-under-gate but `Write` still allows tearing in the mempool's `tx + \n` pattern. Is the mempool intended to migrate to a single `WriteLine`?
- The PR description references issue #5061 which asks for "Halt cleanly when space is unavailable" and "Emit warnings before critical threshold." Is the panic in `cs.wal.WriteSync` ([`bft/consensus/state.go:643-646`](https://github.com/gnolang/gno/blob/17ff49b/tm2/pkg/bft/consensus/state.go#L643-L646) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/consensus/state.go#L643-L646)) the intended "clean halt," or is a follow-up planned to drain consensus instead of panicking?
- Should this land with `tm2/adr/pr5313_*.md` per the gno `AGENTS.md` ADR rule? The 9-commit history (multiple refactor rounds, two reversed decisions on the `Halted()`/`Resume()` API and the configurable limit) is exactly the design churn an ADR is meant to record.
