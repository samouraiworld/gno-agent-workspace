# PR #5326: fix(gnovm): add recursion guard to ComputeMapKey; make isEql iterative

URL: https://github.com/gnolang/gno/pull/5326
Author: thehowl | Base: master | Files: 4 | +337 -125
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `6c867397a` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5326 6c867397a`

**Verdict: REQUEST CHANGES** — fix is sound but ships with a broken txtar test (Part 1 OOGs at the configured `-gas-wanted` because allocator gas now lives in master post-#5127), branch needs a rebase that resolves conflicts with the already-merged #5127, and isEql's pair-stack growth is uncharged Go-heap memory.

## Summary

Two unbounded recursion paths through composite values can crash the node with `fatal error: stack overflow` (unrecoverable, every validator dies on the same block → chain halt). `ComputeMapKey` recurses once per nesting level of `[1]any{...}`; `isEql` does the same for `==`. The fix caps `ComputeMapKey` recursion at `maxComputeMapKeyDepth = 10000` (string panic → caught by `keeper.doRecover` → normal tx error) and converts `isEql` to an explicit `[]pair` LIFO stack so equality has zero Go-stack depth regardless of nesting. The companion PR #5127 (consume-gas-on-ComputeMapKey) has since merged on master, making half of this PR's protection redundant via gas — but the depth guard still defends the unrecoverable stack-overflow scenario, and the iterative `isEql` is strictly better than recursion.

## Glossary

- `ComputeMapKey` — serializes a `TypedValue` into a `MapKey` (string) for use as a Go-map key inside the VM. Recurses on arrays/structs.
- `isEql` — VM-level equality for `TypedValue` (used by `==`, `!=`, type-switch case match).
- `maxComputeMapKeyDepth` — new constant, 10000. Recursion guard ceiling.
- `MsgRun` — submit-and-run arbitrary Gno code in one tx; the attack vector.
- `maxAllocTx` — per-tx allocator cap (~500 MB) that already bounds memory used by deeply nested operand construction.

## Fix

`ComputeMapKey` keeps its public signature but delegates to a new private `computeMapKey(store, omitType, depth)` that panics with `"map key nesting depth limit exceeded"` when `depth > 10000`, propagating `depth+1` into each array-element and struct-field recursion ([`gnovm/pkg/gnolang/values.go:1574-1577`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/values.go#L1574-L1577) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/values.go#L1574-L1577), [`values.go:1639`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/values.go#L1639) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/values.go#L1639), [`values.go:1662`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/values.go#L1662) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/values.go#L1662)). The string panic bubbles past `runOnce`'s `*Exception`-only filter ([`machine.go:1499-1508`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/machine.go#L1499-L1508) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/machine.go#L1499-L1508)) and lands in `doRecover` ([`gno.land/pkg/sdk/vm/keeper.go:871-915`](https://github.com/gnolang/gno/blob/6c867397a/gno.land/pkg/sdk/vm/keeper.go#L871-L915) · [↗](../../../../../.worktrees/gno-review-5326/gno.land/pkg/sdk/vm/keeper.go#L871-L915)) which formats it as `VM panic: map key nesting depth limit exceeded`. `isEql` is rewritten as a `for len(stack) > 0` loop over a `[]pair{l, r TypedValue}` with array/struct fields pushed in reverse so element 0 is consumed first ([`op_binary.go:432-626`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go#L432-L626) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go#L432-L626)); the early-return-on-both-undefined trap was caught mid-PR (commit 63071e77) and properly switched to `continue`.

## Critical (must fix)

- **[txtar test fails on its own happy-path assertion]** [`gno.land/pkg/integration/testdata/mapkey_recursion_overflow.txtar:30-31`](https://github.com/gnolang/gno/blob/6c867397a/gno.land/pkg/integration/testdata/mapkey_recursion_overflow.txtar#L30-L31) · [↗](../../../../../.worktrees/gno-review-5326/gno.land/pkg/integration/testdata/mapkey_recursion_overflow.txtar#L30-L31) — Part 1 (`9999`-level shallow.gno) OOGs at `-gas-wanted 2000000000` (memory allocation), so `stdout 'OK!'` never matches and the test fails. CI `main / test` for this PR records the same failure.
  <details><summary>details</summary>

  Local run on PR head `6c867397a`:

  ```
  > gnokey maketx run -max-deposit 2000000000ugnot -gas-fee 10000000ugnot -gas-wanted 2000000000 ...
  Data: out of gas error
  Msg Traces:
      0  gno/tm2/pkg/errors/errors.go:103 - deliver transaction failed:
         log:out of gas, gasWanted: 2000000000, gasUsed: 2000000060
         location: memory allocation
  FAIL: testdata/mapkey_recursion_overflow.txtar:30: unexpected "gnokey" command failure: out of gas error
  ```

  And on CI run [24795958989](https://github.com/gnolang/gno/actions/runs/24795958989) (main / test, status: fail):

  ```
  --- FAIL: TestTestdata/mapkey_recursion_overflow (8.14s)
      FAIL: testdata/mapkey_recursion_overflow.txtar:30: unexpected "gnokey" command failure: out of gas error
  ```

  The author's own PR comment ([2026-04-22](https://github.com/gnolang/gno/pull/5326#issuecomment-4299084626)) measures a 9999-deep mapkey at ~1.54 M gas in `main()` only — but that experiment was on a branch with gas calibration adjustments; on master+#5127 with the allocation gas table, building a 9999-deep `[1]any{...}` operand alone consumes ~2 B gas. The fix: either raise `-gas-wanted` to `3000000000` (the block max — and that's still tight), or shrink Part 1 to a depth far below the allocation budget (e.g. 100–500 levels — still exercises the recursive code path, no risk of OOG). The current configuration leaves zero headroom for the actual `ComputeMapKey` work.

  Fix: lower shallow.gno depth to ~500 (well above 1–10 typical, well below allocation budget), keep `stdout 'OK!'`.

  **Repro:**

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5326 -R gnolang/gno
  go test -timeout 120s -v -run 'TestTestdata/mapkey_recursion_overflow' ./gno.land/pkg/integration/
  # observe: --- FAIL ... out of gas error ... gasUsed: 2000000060 ... memory allocation
  ```
  </details>

- **[branch is stale; conflicts with merged #5127]** [`gnovm/pkg/gnolang/values.go`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/values.go) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/values.go), [`gnovm/pkg/gnolang/op_binary.go`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go) — #5127 (consume gas on ComputeMapKey) and the related allocator changes have landed on master since this PR was last updated (2026-04-22); rebasing produces merge conflicts in both files. The PR must be rebased before review can certify that the post-merge behavior matches the ADR's claims.
  <details><summary>details</summary>

  Master HEAD `4166be993` already contains `720af8bcd fix: consume gas on ComputeMapKey (#5127)`, which modifies the same `ComputeMapKey` body to charge gas and allocate for `av.Data` copies. The PR's iterative isEql also has overlapping touched lines with subsequent op_binary.go modifications. A rebase trial:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5326 -R gnolang/gno
  git fetch origin master
  git rebase origin/master
  # CONFLICT (content): Merge conflict in gnovm/pkg/gnolang/op_binary.go
  # CONFLICT (content): Merge conflict in gnovm/pkg/gnolang/values.go
  git rebase --abort
  ```

  After rebase, omarsy's [@review comment](https://github.com/gnolang/gno/pull/5326#discussion_r3107485647) ("after #5127, should never happen") needs to be re-validated: does the depth guard still protect against unrecoverable stack overflow at any gas/alloc budget below #5127's metering? The author's [response](https://github.com/gnolang/gno/pull/5326#issuecomment-4299084626) argued yes for cheap defensive insurance; the post-rebase code should make that argument empirically (a calibration test like `TestComputeMapKey_GasVsDepthLimit` from the reverted commit 435c8ae1 was the right artifact — it should come back, not be discarded).

  Fix: rebase onto current master, resolve conflicts, re-run the txtar with whatever gas budget actually permits a 500-level shallow path, and either include the gas-vs-depth calibration test or reference it in the ADR as load-bearing evidence.
  </details>

## Warnings (should fix)

- **[uncharged Go-heap memory in iterative isEql]** [`gnovm/pkg/gnolang/op_binary.go:436-437`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go#L436-L437) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go#L436-L437) — the explicit `stack []pair{l, r TypedValue}` grows on the Go heap with O(depth) memory not tracked by the GnoVM allocator. Two TypedValues per pair ≈ 80 bytes; at depth N this is ~80N bytes of transient memory that bypasses `maxAllocTx`.
  <details><summary>details</summary>

  The ADR claims "`isEql` now uses O(depth) heap memory instead of O(depth) stack frames — bounded by `maxAllocTx`" ([`gnovm/adr/pr5326_map_key_depth_limit.md:54-55`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/adr/pr5326_map_key_depth_limit.md#L54-L55) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/adr/pr5326_map_key_depth_limit.md#L54-L55)). That's almost true: the *operands* on the stack are pre-existing TypedValues whose memory was already tracked when the array/struct was built (and bounded by `maxAllocTx`). But the **slice headers** holding the pairs are fresh Go-heap allocations, plus the value-copy of each TypedValue (T pointer, V interface, [8]byte) is duplicated into the stack — these duplicates are not allocator-tracked. At the documented max equality depth (bounded only by `maxAllocTx ~= 500 MB`/40 bytes/TypedValue ≈ 12.5 M elements), the pair-stack could transiently grow to ~1 GB of un-metered Go heap. In practice block-gas and allocation-gas bound the operand construction long before that, so this is not exploitable, but the ADR claim is too strong and the un-metered slice growth deserves an `Allocate` call per push (or per growth-doubling) for defense-in-depth.

  Fix: either narrow the ADR claim to "bounded transitively by the operand's allocation cost (~2x in transient memory)", or charge an `m.Alloc.Allocate(sizeOfPair)` per push. The former is cheaper and probably sufficient given gas-bounded operand construction is the real ceiling.
  </details>

- **[ADR overstates depth-headroom]** [`gnovm/adr/pr5326_map_key_depth_limit.md:34`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/adr/pr5326_map_key_depth_limit.md#L34) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/adr/pr5326_map_key_depth_limit.md#L34) — "10 000 levels is far above any legitimate use case (typical nesting: 1–10) and well below the Go goroutine stack limit". The first half is fine; the second half is true but irrelevant. The load-bearing bound at this point is not Go's stack — it's the block gas budget for allocating the operand. At master+#5127, a 9999-deep `[1]any{...}` operand alone consumes ~2 B gas (66% of block budget), per the txtar OOG repro above. The ADR should rank the bounds in their real-world order: allocation-gas-on-operand-construction → depth guard → Go stack.
  <details><summary>details</summary>

  This isn't a correctness bug; it's a clarity bug in the ADR's justification for choosing 10 000. The right framing is: "depth guard 10 000 = 100× above any realistic use case, with allocation-gas-on-operand-construction acting as the practical first-line ceiling at the block-gas boundary". The current "well below the Go goroutine stack limit" framing implies the Go-stack ceiling is what matters; it isn't, except as the catastrophic fallback the guard exists to prevent. Tighten the language so a future maintainer doesn't lower the constant under the impression that the Go-stack limit is the binding constraint.

  Fix: replace "well below the Go goroutine stack limit (~1 900 000 frames at ~526 B each)" with a single sentence: "guard exists to prevent the unrecoverable Go stack-overflow that occurs around ~1.9 M levels in the absence of gas metering; the practical ceiling under #5127 is the block gas budget consumed by allocator gas on operand construction (~2 B gas at depth 9999)".
  </details>

- **[user-recover bypass risk if Exception type ever changes]** [`gnovm/pkg/gnolang/values.go:1576`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/values.go#L1576) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/values.go#L1576), [`gnovm/pkg/gnolang/machine.go:1499-1508`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/machine.go#L1499-L1508) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/machine.go#L1499-L1508) — the depth panic uses a plain Go `panic("string")` which bypasses the VM's structured `*Exception` path. This is *currently* correct (string panics are re-raised by `runOnce` and caught only by `keeper.doRecover`, so Gno user code's `recover()` cannot intercept), but the property is fragile.
  <details><summary>details</summary>

  The reason the user can't catch the depth panic is that `runOnce`'s deferred recover only converts `r.(*Exception)` into `caught`; everything else is re-raised ([`machine.go:1505`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/machine.go#L1505) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/machine.go#L1505)) and unwinds out of `Machine.Run` entirely, surfacing to `keeper.doRecover`. If a future refactor ever generalizes that re-raise to wrap string panics into `*Exception` (so user code can see them, or for stacktrace richness), this guard becomes a recover-able panic that user code could swallow and retry from in a tight loop — a slow chain-halt instead of a fast one.

  Fix: either add a regression test in `op_binary_test.go` proving a Gno `defer recover()` does NOT catch the depth panic, or convert the panic to a typed `*Exception`-shaped value with a sentinel that `keeper.doRecover` recognizes. The latter is more invasive; the former is a one-screen test.
  </details>

## Nits

- [`gnovm/pkg/gnolang/op_binary.go:438`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go#L438) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go#L438) — initial `make([]pair, 0, 8)` is fine but consider `cap = 16` to cover small struct comparisons without growing.
- [`gnovm/pkg/gnolang/op_binary.go:546-552`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go#L546-L552) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go#L546-L552) — "Push in reverse order so element 0 is at the top of the stack" reads well, but adding `// equivalent to original recursive left-to-right scan: index 0 popped first` would help future maintainers verify the order matters for behavior (it doesn't, all-or-nothing equality, but the early-mismatch optimization wants index-0-first).
- [`gnovm/adr/pr5326_map_key_depth_limit.md:1`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/adr/pr5326_map_key_depth_limit.md#L1) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/adr/pr5326_map_key_depth_limit.md#L1) — filename was renamed from `prxxxx_` to `pr5326_` mid-PR (commit `3d3ead12`). Confirm the ADR registry/index expects this naming convention (other ADRs in `gnovm/adr/` use a mix of prefixes).

## Missing Tests

- **[no direct unit test for `isEql` semantic parity]** [`gnovm/pkg/gnolang/op_binary.go:432`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go#L432) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go#L432) — the iterative rewrite is non-trivial (every recursive branch becomes a switch-arm-or-continue) and has zero direct Go-level unit tests. Coverage relies entirely on `TestFiles` regression catching divergence.
  <details><summary>details</summary>

  Coverage report from this PR: `op_binary.go` patch coverage is **78.52%** with 28 missing and 7 partial lines. The grep for `TestIsEql` / `TestEql` in `gnovm/pkg/gnolang/*_test.go` returns nothing — the function has no dedicated tests. The PointerKind branch in particular is subtle (DataByteType vs non-DataByteType, both-nil vs one-nil) and its semantic equivalence to the original `return lv.V == rv.V` fall-through depends on Go interface equality rules. A table-driven `TestIsEql` covering: (1) primitive types, (2) array of primitives, (3) deeply nested array-of-any, (4) struct with embedded undefined field, (5) pointer-to-DataByte vs pointer-to-DataByte (equal and unequal), (6) pointer-to-non-DataByte one-side-nil — would catch any future regression and document the semantic contract.

  Fix: add `gnovm/pkg/gnolang/op_binary_isEql_test.go` with table-driven cases mirroring each switch arm.
  </details>

- **[no calibration test for depth-guard necessity post-#5127]** [`gnovm/adr/pr5326_map_key_depth_limit.md:11-13`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/adr/pr5326_map_key_depth_limit.md#L11-L13) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/adr/pr5326_map_key_depth_limit.md#L11-L13) — the ADR's central empirical claim ("attack window approximately [~1.88 M, ~2.02 M] nesting levels, reachable within production limits") needs a runnable test that proves, on current master + #5127 gas calibration, that an attacker can still reach a depth that crashes Go's stack absent the guard.
  <details><summary>details</summary>

  Commit `435c8ae1` (subsequently reverted in `a789e697`) included `TestComputeMapKey_GasVsDepthLimit` precisely to demonstrate that `OpCPUComputeMapKey=10` allows ~300 M depth before OOG while the stack crashes at ~1.9 M. That test is the load-bearing evidence for keeping the depth guard despite gas being the primary defense. Discarding it removes the argument from the PR's own artifacts; a year from now a maintainer trimming "redundant" guards will have no way to verify the guard still earns its place.

  Fix: restore `TestComputeMapKey_GasVsDepthLimit` (or equivalent) as a `go test -short`-runnable artifact in `gnovm/pkg/gnolang/`, and add a one-line reference to it from the ADR's "Decision" section so the reasoning survives the next PR cycle.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/op_binary.go:435`](https://github.com/gnolang/gno/blob/6c867397a/gnovm/pkg/gnolang/op_binary.go#L435) · [↗](../../../../../.worktrees/gno-review-5326/gnovm/pkg/gnolang/op_binary.go#L435) — the local `type pair struct{ l, r TypedValue }` is reasonable, but if hot-path benchmarks (`bench_ops_test.go:3374-3452`) show measurable allocator pressure, consider a `sync.Pool` of `[]pair` or a per-Machine reusable buffer. Worth measuring before optimizing.
  <details><summary>details</summary>

  The current code does `stack := make([]pair, 0, 8)` on every `isEql` call, which is per-`==`, per-switch-case-match, and per-map-key-store. For programs that do many small equality compares, this is one allocation per call. A pool keyed off `*Machine` would amortize that. Not blocking; just a flag for follow-up.
  </details>

## Questions for Author

- After rebasing onto master (with #5127), does `TestComputeMapKey_GasVsDepthLimit` (or an equivalent calibration test) still show that gas alone leaves a stack-overflow window? If yes, surface that test as a permanent artifact; if no, the depth guard becomes purely defensive and the ADR should say so plainly.
- Why does shallow.gno use `9999` levels rather than something well clear of the gas budget (say 500)? The intent reads as "as close to the guard as possible without firing it" but the practical effect is "bumps into the block-gas ceiling first" — moving the depth far below the gas cliff makes the test cheaper, less flaky across gas-calibration changes, and still exercises the recursive code path.
- The string panic `"map key nesting depth limit exceeded"` is matched by stderr substring in Part 2 — is there a more structured error type you'd want to expose so SDK/tooling can detect this case without string-matching? (Compare to `OutOfGasError` which has a typed Go error.)
