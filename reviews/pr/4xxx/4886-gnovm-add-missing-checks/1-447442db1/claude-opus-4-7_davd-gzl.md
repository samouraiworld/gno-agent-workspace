# PR #4886: fix(gnovm): Add missing checks

URL: https://github.com/gnolang/gno/pull/4886
Author: davd-gzl | Base: master | Files: 5 | +61 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7

Note: reviewer is the PR author. Review is unattended-sweep output; treat as self-review surfacing concerns for a human maintainer to arbitrate.

Verdict: APPROVE — fixes two minor audit findings (#4780 exception-chain integrity, #4784 ObjectID unmarshal validation); changes are tightly scoped, additive, covered by unit tests, CI green. One coverage gap noted below (no end-to-end filetest exercising the chained-panic + recover + re-panic path the [`machine.go:2417-2419`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2417-L2419) cleanup unlocks).

## Summary

Two unrelated audit fixes from the PR #4060 security review, bundled. (1) `Exception` chain: `WithPrevious` now panics if the candidate previous already has a forward link ([`frame.go:294-296`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/frame.go#L294-L296)), and `Machine.Recover` severs the back-pointer's `Next` before clearing `m.Exception` ([`machine.go:2417-2419`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2417-L2419)) so a subsequent panic can re-link the older exception via `fr.LastException` without tripping the new invariant. (2) `ObjectID.UnmarshalAmino` rejects `len(parts[0]) != 40` (40 hex chars = 20-byte PkgID), negative `NewTime`, and `NewTime == 0` for non-zero `PkgID` ([`ownership.go:57-73`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L57-L73)). The 40-char length check is load-bearing — `hex.Decode` writes byte-by-byte into the fixed-size 20-byte `Hashlet` and would panic with `index out of range` on overlong input (verified locally).

## Glossary

- `Exception` — gno panic wrapper, doubly-linked via `Previous`/`Next` to track chained panics across defers ([`frame.go:257-262`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/frame.go#L257-L262)).
- `WithPrevious` — links a new exception as head, old exception as tail ([`frame.go:284-300`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/frame.go#L284-L300)).
- `Machine.Recover` — implements gno's `recover()` builtin ([`machine.go:2384-2422`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2384-L2422)).
- `pushPanic` — start a panic from a doOp* handler; chains via `WithPrevious(fr.LastException)` or `WithPrevious(m.Exception)` depending on whether a previous panic is already live ([`machine.go:2360-2379`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2360-L2379)).
- `NewTime` — monotonically increasing nonce inside a realm; starts at 1 for the realm's package object, increments per new object ([`realm.go:1670-1675`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/realm.go#L1670-L1675)).

## Fix

Before: `WithPrevious` accepted any `e2`, silently clobbering its forward link. `Recover` cleared `m.Exception` but left the previous exception's `Next` pointing at the (now-recovered) head; a later panic on the same frame would re-link via `fr.LastException` and find `Next` already set — undetectable silent corruption.

After: `WithPrevious` panics on a non-nil `e2.Next`. `Recover` nils `ex.Previous.Next` before resetting `m.Exception`, so the recovered head detaches from the chain cleanly and subsequent re-linking is sound.

Before (`ownership.go`): malformed amino strings could (a) panic in `hex.Decode` on `len(parts[0]) > 40` (writes past `Hashlet[20]byte`), (b) set `NewTime = 0` for a non-zero PkgID, violating the invariant in [`IsPackageID`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L83-L86) and the `debug` assertion in [`IsZero`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L90-L99), (c) silently wrap negative `int` into a large `uint64` via the conversion at [`ownership.go:74`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L74).

After: explicit length, range, and consistency checks reject all three cases with descriptive errors before mutation.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`ownership.go:64-67`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L64-L67) — `strconv.Atoi` accepts non-canonical forms (`"+1"`, `"01"`, `"007"`). `MarshalAmino` always emits `%d` (canonical), so round-trip is fine, but unmarshal accepts shapes the marshaller never produces. If strict round-trip parity matters here, prefer `strconv.ParseUint(parts[1], 10, 64)` which rejects leading `+`/`0`. Minor.
- [`frame.go:294-296`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/frame.go#L294-L296) — panic message `"exception e2 already has a next link"` references internal parameter name `e2`. From a stack trace the reader has no context for what `e2` is. Consider `"WithPrevious: previous exception already has a forward link"`.
- [`machine.go:2416`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2416) — the new block lands inside a long-running block comment about `m.Exception` re-set semantics, separated only by a blank line. The cleanup logic deserves its own one-line comment explaining the invariant being maintained (the new `WithPrevious` panic condition).

## Missing Tests

- **[no integration test for the cleanup path]** [`machine.go:2417-2419`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2417-L2419) — no `.gno` filetest exercises the exact scenario the cleanup unlocks.
  <details><summary>details</summary>

  The new `WithPrevious` panic at [`frame.go:294-296`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/frame.go#L294-L296) and the cleanup at [`machine.go:2417-2419`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/machine.go#L2417-L2419) are coupled: the cleanup is required precisely because of the new invariant. Without the cleanup, the existing `recover2.gno` test (or similar multi-defer-panic flows) would hit the new `WithPrevious` panic. The unit test `TestException_WithPrevious_E2HasNext` only exercises `WithPrevious` directly; it does not run a `panic → recover → panic` flow end-to-end. Add a `gnovm/tests/files/recover/recoverN.gno` with multi-defer panics, recover, and a follow-up panic that re-uses `fr.LastException`. The existing `recover2.gno` partly exercises this (it passes per local run), but a dedicated test pinned to this PR's invariant makes the regression risk explicit.
  </details>

- **[no test for malformed-amino overlong PkgID]** [`ownership_test.go`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership_test.go) — only undersize case is tested.
  <details><summary>details</summary>

  The PR's `TestObjectID_UnmarshalAmino_InvalidPkgIDLength` tests `"abc:100"` (too short). The load-bearing case — overlong input that would otherwise panic in `hex.Decode` writing past `Hashlet[20]byte` — is not tested. Add a case like `"00112233445566778899aabbccddeeff0011223344:1"` (42 hex chars). Confirms the new length guard prevents the underlying out-of-bounds write rather than just rejecting cosmetic length mismatches.
  </details>

- **[no test for newTime overflow]** [`ownership.go:64-74`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L64-L74) — `int` → `uint64` is checked for negative but not for `int` overflow on 32-bit platforms.
  <details><summary>details</summary>

  `strconv.Atoi` returns `int` (platform-dependent width). On 64-bit hosts the range matches `uint64` once non-negative. On 32-bit hosts a serialized `NewTime > 2^31-1` would fail to parse. Gno's deterministic VM means this matters: a value emitted on 64-bit (`%d` of `uint64`) can fail to round-trip on 32-bit. Either use `strconv.ParseUint(parts[1], 10, 64)` (rejects negatives naturally, parses full uint64 range, platform-independent) — or document the 32-bit constraint. Same call site, same complexity.
  </details>

## Suggestions

- [`ownership.go:64`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/ownership.go#L64) — replace `strconv.Atoi` + negative check + `uint64` conversion with `strconv.ParseUint(parts[1], 10, 64)`. Removes three concerns in one line: negative inputs, non-canonical `+1`/`01` forms, and 32-bit platform overflow. Then keep only the `newTime == 0 && !oid.PkgID.IsZero()` check.
- [`frame_test.go`](../../../../../.worktrees/gno-review-4886/gnovm/pkg/gnolang/frame_test.go) — add a second positive-path test: e1, e2 with `e2.Next == nil`, call `e1.WithPrevious(e2)`, assert `e1.Previous == e2 && e2.Next == e1`. Pins the happy-path contract alongside the panic case.

## Questions for Author

- Was the audit report (PR #4060 review) specific about whether the `WithPrevious` "panic" was the intended remediation, or did it propose "safely detach `e2` from its current chain prior to re-linking"? The issue text says "either reject... or safely detach". The PR chose reject; the chosen path is defensible (catches programmer error early) but a maintainer may prefer the silent-detach fallback for resilience. Worth flagging in the PR body.
- Why did `MikaelVallenet`'s approval (per the Gno2D2 bot) not appear as a formal review on the PR? Asking because the "[bot] MikaelVallenet approved" comment by `jefft0` is the only attestation visible — a real GitHub review would be more durable.
