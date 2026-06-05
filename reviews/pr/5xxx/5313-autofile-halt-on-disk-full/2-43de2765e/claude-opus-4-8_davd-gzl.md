# PR #5313: fix(autofile): halt writes on disk space exhaustion with auto-recovery

URL: https://github.com/gnolang/gno/pull/5313
Author: davd-gzl | Base: master | Files: 5 | +420 -72
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `43de2765e` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5313 43de2765e`

Round 2 re-review. Round 1 ([`1-17ff49b/claude-opus-4-7_davd-gzl.md`](../1-17ff49b/claude-opus-4-7_davd-gzl.md)) returned REQUEST CHANGES on commit `17ff49b`. Two commits since: `ac93761` (the substantive fix) and `43de2765` (drop unused const). This round focuses on whether the prior findings were resolved.

**Verdict: APPROVE** — the two Criticals and all six Warnings from round 1 are addressed or consciously deferred to the ADR. The reactive ENOSPC path is now wired through `handleIOErr` from `FlushAndSync` and `rotateFile`, rotation halts instead of panicking on ENOSPC, the warning log is one-shot, the ADR exists, and `Halted()` is back. Two soft gaps remain — the reactive ENOSPC path (now load-bearing) has no test, and the mempool two-`Write` tear is still possible since the mempool code was not touched — but both are documented or low-probability and don't block merge. Recommend the author add an ENOSPC-injection test before or shortly after merge.

## Round 1 → Round 2 status

| # | Round 1 finding | Status | Where |
|---|---|---|---|
| C1 | Reactive `isErrNoSpace` branch unreachable through `bufio.Writer` | Resolved | `handleIOErr` now called from `FlushAndSync` and `rotateFile`, the real flush sites |
| C2 | Mempool split-write can tear line on halt boundary | Not addressed (code), low-prob | mempool untouched; window narrowed by throttle |
| W1 | `FlushAndSync` not gated / doesn't trip halt | Resolved | `group.go:290-292` routes through `handleIOErr` |
| W2 | `rotateFile` panics on ENOSPC | Resolved | `group.go:449-461` halts + returns wrapped err on ENOSPC |
| W3 | Per-validator non-uniform halt | Deferred (by design) | ADR "Consequences" documents it |
| W4 | Missing ADR | Resolved | `tm2/adr/pr5313_autofile_disk_space_halt.md` added |
| W5 | Warning log noisy under sustained pressure | Resolved | `warnedLowSpace` one-shot edge-trigger |
| W6 | Fail-open on `Statfs` error masks problems | Not addressed | still logs + returns nil; not in ADR |
| W7 | `Halted()`/`Resume()` removal | Resolved (split) | `Halted()` getter re-added; `Resume()` stays gone (correct) |

## Summary

Round 1's core objection was that the protection was theater: the only place that set `halted` was the proactive `statfs` gate in `writeBytes`, while the real ENOSPC always surfaced from `bufio.Writer.Flush()` inside `FlushAndSync`/`rotateFile`, which neither halted nor wrapped the error. Commit `ac93761` closes that by extracting [`handleIOErr`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L296-L306) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L296-L306) and calling it from all three I/O sites (buffered write, flush/sync, rotate). `rotateFile` now returns `error` and halts on ENOSPC instead of panicking. The warning is edge-triggered via `warnedLowSpace`. An ADR documents the design and the known consequences (per-node halt, consensus still panics via `WriteSync`). `Halted()` is restored as a read-only getter. Net: the PR now does what its title claims for the flush-driven WAL and event-store paths, which is the realistic ENOSPC site.

## Fix

`writeBytes`, `FlushAndSync`, and `rotateFile` now funnel I/O errors through [`handleIOErr`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L296-L306) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L296-L306), which on ENOSPC sets `halted` and wraps in `ErrDiskSpaceUnavailable`; non-ENOSPC errors pass through (and `rotateFile` still panics on them, preserving prior behavior). The proactive `statfs` gate is unchanged in shape but now reachable via the `availableDiskSpaceFn` package-var indirection so tests stub it. The warning is one-shot per low-space episode. See [`group.go:251-272`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L251-L272) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L251-L272), [`group.go:283-294`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L283-L294) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L283-L294), [`group.go:443-481`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L443-L481) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L443-L481).

## Critical (must fix)

None.

## Warnings (should fix)

- **[mempool two-`Write` tear still possible — code unchanged since round 1]** [`tm2/pkg/bft/mempool/clist_mempool.go:270-277`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) — mempool still issues `Write([]byte(tx))` then `Write([]byte("\n"))` as two separate gated calls; a halt that lands between them buffers an unterminated tx and rejects the `\n`. Downgraded from Critical to Warning: the throttle narrows the window and the failure path is recoverable, not a crash.
  <details><summary>details</summary>

  The PR did not touch the mempool (the only mempool diff vs master is an unrelated Debug→Info log change from master drift). The tear requires the disk to cross the 16 MB threshold inside a 100-write window between the tx half and the `\n` half: tx-`Write` succeeds without a check (counter < 100), then `\n`-`Write` trips the 100th-write `statfs`, finds `< 16 MB`, halts, and returns `ErrDiskSpaceUnavailable` while the tx bytes are already in `headBuf`. The mempool logs and continues ([`clist_mempool.go:267`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/bft/mempool/clist_mempool.go#L267) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L267), `// TODO: Notify administrators when WAL fails`), so on next successful tx the bytes concatenate with no separator, corrupting the WAL stream on replay. Low probability, but the fix is cheap and removes the hazard entirely. Fix: convert the mempool to a single `mem.wal.WriteLine(string(tx))` call so the tx + newline pass through one atomic gate; or leave as-is and note in the ADR that the mempool WAL accepts this tear risk. The round-1 review raised this as a Question for Author; it remains unanswered.
  </details>

- **[fail-open on `availableDiskSpace` error — unchanged, not in ADR]** [`tm2/pkg/autofile/group.go:325-331`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L325-L331) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L325-L331) — when `availableDiskSpaceFn` returns a real error (EACCES, EIO, fs unmounted) on a supported platform, the code logs at ERROR and returns nil, so writes continue and the group never halts on a genuinely broken filesystem.
  <details><summary>details</summary>

  This is the one round-1 finding with no movement in either code or ADR. The unsupported-platform case is already handled by the `diskSpaceUnsupported` sentinel ([`group.go:332-335`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L332-L335) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L332-L335)), so the `err != nil` branch is exclusively real syscall failure. Logging once and continuing turns a hard signal into a soft one; if the fs is unmounted, the subsequent flush will ENOSPC-or-other and `handleIOErr` only halts on ENOSPC specifically, so a non-ENOSPC fs failure still slips through. Fix: count consecutive `Statfs` failures and halt after N, or at minimum add a one-line note to the ADR "Consequences" that supported-platform `statfs` errors are intentionally fail-open. Acceptable to defer, but it should be a conscious decision on record, not silence.
  </details>

## Nits

- [`tm2/pkg/autofile/group.go:366`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L366) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L366) — `"time", time.Now()` on the warning log adds a wall-clock timestamp inside the structured log, but the logger already stamps each line; the field is redundant unless it's meant to record the *first* low-space moment (in which case it should be captured once, not re-evaluated).
- [`tm2/pkg/autofile/group.go:46`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L46) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L46) — `const diskSpaceUnsupported = uint64(math.MaxUint64)` — the `uint64(...)` cast on an already-untyped-but-`MaxUint64` constant is harmless; `math.MaxUint64` alone in a `uint64` const context suffices. Cosmetic, addresses the round-1 nit.
- [`tm2/pkg/autofile/diskspace_unix.go:18-20`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/diskspace_unix.go#L18-L20) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/diskspace_unix.go#L18-L20) — `if stat.Bsize <= 0` guard is still unreachable on any real filesystem (round-1 nit, unchanged). Keep or drop, not blocking.

## Missing Tests

- **[reactive ENOSPC path — now load-bearing — has no test]** [`tm2/pkg/autofile/group_test.go`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group_test.go) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group_test.go) — the two new tests (`TestHaltOnLowDiskSpace`, `TestAutoRecoveryAfterSpaceFreed`) stub `availableDiskSpaceFn` to exercise the proactive `statfs` gate, but nothing injects an ENOSPC from the underlying file `Write`/`Flush`/`Sync` to exercise `handleIOErr` — the very path round 1's Critical was about and that `ac93761` added.
  <details><summary>details</summary>

  `handleIOErr`, the ENOSPC branch in `FlushAndSync` ([`group.go:290-292`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L290-L292) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L290-L292)), and the ENOSPC-halt-instead-of-panic in `rotateFile` ([`group.go:449-461`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L449-L461) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L449-L461)) are uncovered. codecov reports the patch failing its threshold. A test substituting `g.headBuf`'s underlying `Head.Write` (or wrapping the file) with a writer that returns `syscall.ENOSPC` would assert: (a) `FlushAndSync` returns `errors.Is(err, ErrDiskSpaceUnavailable)` and `g.Halted()` is true; (b) `rotateFile` halts and returns the wrapped error rather than panicking. Without it, the fix for the round-1 Critical is asserted only by reading the code. Recommend adding before or right after merge.
  </details>

- **[mempool tear has no regression test]** [`tm2/pkg/bft/mempool/clist_mempool.go:270-277`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/bft/mempool/clist_mempool.go#L270-L277) — the Warning above has no test pinning the torn-line behavior. Only worth adding if the author chooses to keep the two-`Write` pattern.

## Suggestions

- [`tm2/pkg/autofile/group.go:283-294`](https://github.com/gnolang/gno/blob/43de2765e/tm2/pkg/autofile/group.go#L283-L294) · [↗](../../../../../.worktrees/gno-review-5313/tm2/pkg/autofile/group.go#L283-L294) — `FlushAndSync` attempts the flush even when `g.halted` is already true, so a halted group keeps trying to push buffered bytes onto a full disk on every flush. Harmless (the flush just re-errors and re-halts) but a one-line `if g.halted { return ErrDiskSpaceUnavailable }` early-return would avoid the redundant syscall and make the contract explicit. Optional.

## Questions for Author

- Mempool two-`Write` tear (round-1 C2, now W1): keep the pattern and document the accepted tear risk in the ADR, or switch to a single `WriteLine`? This is the last unaddressed round-1 finding with a code dimension.
- Fail-open on supported-platform `statfs` error: intentional? If so, a one-line ADR note closes it.
