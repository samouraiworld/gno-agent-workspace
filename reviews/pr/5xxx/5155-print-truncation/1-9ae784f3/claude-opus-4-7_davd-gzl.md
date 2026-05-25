# PR #5155: fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps

URL: https://github.com/gnolang/gno/pull/5155
Author: davd-gzl | Base: master | Files: 3 | +251 -11
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — fix correctly caps the per-level native `make([]string)` / `strings.Join` allocations driving the HackenProof report ([NEWTENDG-59](https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-59)); only nit is the `printOutputLimit` safety net is dead code on the actual `print()`/`println()` path because `Sprint`/`ProtectedSprint` bypass it. Self-review disclaimer: this review is on the author's own PR — flagged for independent confirmation before merge.

## Summary

Native Go allocations inside `ArrayValue/SliceValue/MapValue.ProtectedString` (`make([]string, len(...))` + `strings.Join`) were unbounded by the GnoVM allocator. An attacker submitting an `MsgRun` could create a slice whose backing `make([]string, N)` allocation grew with `N`, bypassing the per-tx allocator budget. The fix caps each renderer at 256 elements (matching the pre-existing byte-slice cap) before the `make` line is reached, emitting `slice[...(N elements)]` / `array[...(N elements)]` / `map{...(N entries)}` summaries. A second `printOutputLimit = 64_000` (~6.4% of `MaxTxBytes = 1MB`) wraps direct `.String()` entry points as a combinatorial-blowup safety net.

## Glossary

- `printLimit` — new constant, 256 elements per slice/array/map at render time.
- `printOutputLimit` — new constant, 64_000 bytes; cap applied by `truncateOutput` at top-level `.String()`.
- `ProtectedString` — value-tree walk that builds the string with `seenValues`-based cycle detection.
- `ProtectedSprint` — sibling walker used by `print()`/`println()`; routes through `ProtectedString` for composite types but is itself never wrapped.
- `uversePrint` — runtime impl of Gno `print`/`println` ([`uverse.go:1214`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/uverse.go#L1214)); calls `tv.Sprint(m)`.

## Fix

Before: `SliceValue.ProtectedString` did `make([]string, sv.Length)` for any non-byte slice ([old code](https://github.com/gnolang/gno/blob/master/gnovm/pkg/gnolang/values_string.go#L133)), routing every element through `strings.Join`. After: each composite renderer short-circuits when count exceeds `printLimit = 256`, returning a `[...(N elements)]` summary; the `make` is only reached when count ≤ 256. Byte-slice/byte-array paths now use the same `printLimit` constant instead of the hardcoded `256`. A `truncateOutput` wrapper caps the final string at 64KB and is added to every direct `.String()` entry point ([`values_string.go:100`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L100), [`:134`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L134), [`:173`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L173), [`:195`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L195), [`:246`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L246), [`:486`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L486)).

## Critical (must fix)

None.

## Warnings (should fix)

- **[safety net never fires on `print()`]** [`values_string.go:101,135,196,247,487`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L101) — `truncateOutput` is unreachable on the actual `print`/`println` call path; only the per-level `printLimit` cap protects production.
  <details><summary>details</summary>

  The runtime print path is [`uversePrint`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/uverse.go#L1214) → `tv.Sprint(m)` → [`ProtectedSprint`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L359) → `ps.ProtectedString(seen)` ([line 466](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L466)). `truncateOutput` is wired only on direct `.String()` entry points ([`SliceValue.String`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L134), etc.), which are used for Go-side debugging/test formatting, not by the VM's `print()`/`println()`. So the 64KB safety net guards `fmt.Println(slice)` from Go test code but does nothing for the attacker's actual `MsgRun` with `println(...)`. The mitigation that does ship for production is the per-level `printLimit = 256` cap, which is sufficient on its own — `sv.Fields`, `av.List`, `mv.GetLength()` are all allocator-tracked, so output size is bounded by the per-tx allocator budget. The commit message at [`9ae784f3`](https://github.com/gnolang/gno/pull/5155/commits/9ae784f3719ec8e4b2f6f09dbf3b70ccec314b42) describes `printOutputLimit` as protecting against "combinatorial explosion from nested structures" — that's the wrong path. Fix: either wrap `ProtectedSprint`'s public-facing entry (`Sprint`) with `truncateOutput`, or drop the `printOutputLimit` constant and `truncateOutput` helper entirely since the per-level cap already does the work; either way, the docstring on `printOutputLimit` should match what it actually protects.
  </details>

- **[self-review on security PR]** PR-level — author and reviewer are the same person (`davd-gzl`); merge requires an independent eye on the HackenProof scope.
  <details><summary>details</summary>

  This PR carries a `fix:` link to a HackenProof report and is the disclosed security mitigation. Approvals on file are from `notJoon` ("seems valid to me") and `mvallenet` ("LGTM, but i let some comments and open question"); neither approval explicitly evaluates whether `printLimit = 256` is the right ceiling against the original PoC payload. The HackenProof report itself is private — reviewers should confirm the cap suffices against the disclosed exploit before merging. Fix: request a tech-staff reviewer with HackenProof access to confirm the bound; if they can reproduce the original OOM, attach a Gno filetest reproducing the pre-fix path and verify the post-fix output stays under a documented byte budget.
  </details>

## Nits

- [`values_string.go:74-82`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L74) — `"...(truncated)"` is appended *after* the cap, so the actual returned length is `printOutputLimit + 14`. Not a bug, but the comment "caps the output string at printOutputLimit" reads as a hard ceiling. Either tighten the comment ("caps the prefix at `printOutputLimit`, appending a 14-byte suffix") or truncate to `printOutputLimit - len(suffix)`.
- [`values_string.go:128-129`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L128) — byte-array branch uses `fmt.Sprintf("array[0x%X...(%d)]", ...)` while the non-byte branch uses `array[...(%d elements)]`. Consider unifying the format so log parsers don't need two patterns.
- [`values_test.go:565`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_test.go#L565) — the `[][]int` nested test asserts the literal "slice[...(300 elements)]" twice but does not exercise the actual `print()`/`println()` runtime path; it only exercises `SliceValue.String()`. A filetest mirroring `print4.gno` with a `[][]int` of 2×300 would cover the production path the security fix is for.

## Missing Tests

- **[print path coverage]** [`gnovm/tests/files/print4.gno`](../../../../../.worktrees/gno-review-5155/gnovm/tests/files/print4.gno) — `print4.gno` only covers flat 300-element slice/array/map; no filetest covers nested or struct paths through `println`.
  <details><summary>details</summary>

  The shipped filetest `print4.gno` exercises only single-level containers. The original report mentions `make([]string)` + `strings.Join` being unbounded — that's the slice path, which is covered. Worth adding: (1) a nested filetest like `[][]int{300, 300}` to demonstrate cap-at-each-level behavior in production output; (2) a struct-with-large-slice filetest, since `StructValue.ProtectedString` ([`values_string.go:209`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L209)) has no per-field cap and relies on the slice cap firing inside fields. See [`reviews/pr/5xxx/5155-print-truncation/1-9ae784f3/tests/print_nested_blowup.gno`](tests/print_nested_blowup.gno) for a passing adversarial filetest covering case (1).
  </details>

- **[struct field-count baseline]** [`values_string.go:209`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L209) — no test asserts that a struct with N fields renders within an expected size envelope.
  <details><summary>details</summary>

  `StructValue.ProtectedString` does `make([]string, len(sv.Fields))` with no cap — the only thing keeping native memory bounded for structs is that `sv.Fields` is allocator-tracked via `AllocateStructFields` ([`alloc.go:191`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/alloc.go#L191)). That's a non-obvious invariant; a regression test asserting "struct with 1000 fields renders without exceeding 4× the field count in bytes" would document it. Not required for merge.
  </details>

## Suggestions

- [`values_string.go:30`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L30) — comment on `printOutputLimit` says "safety net to prevent combinatorial explosion from nested structures." Replace with what it actually protects: "applied to direct `.String()` entry points used by Go-side debug/test formatting; the runtime `print()`/`println()` path uses `ProtectedSprint` and is bounded only by `printLimit`."
- [`values_string.go:25`](../../../../../.worktrees/gno-review-5155/gnovm/pkg/gnolang/values_string.go#L25) — `printLimit = 256` matches the prior hardcoded byte cap. Worth a brief one-liner naming the original rationale (matching the byte-slice display cap from before this PR) so future maintainers don't tune one without the other.

## Questions for Author

- Was the choice of `printOutputLimit = 64_000` driven by a specific OOM ceiling from the HackenProof PoC, or is it a round number? If the PoC is private, an inline pointer would help future maintainers reason about the value.
- Should `StructValue.ProtectedString` also gain an explicit cap, or is the implicit allocator bound on `sv.Fields` considered sufficient documentation?
