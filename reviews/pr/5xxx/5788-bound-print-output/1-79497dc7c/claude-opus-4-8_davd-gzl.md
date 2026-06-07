# PR #5788: fix(gnovm): bound print/println output to prevent native memory exhaustion

URL: https://github.com/gnolang/gno/pull/5788
Author: thehowl | Base: master | Files: 5 | +629 -192
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `79497dc7c` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5788 79497dc7c`

**Verdict: APPROVE** — closes the exact gap left open by #5155: the 64 KB cap now lives on the actual `print`/`println` (`Sprint`) path, not just the Go-side `.String()` entry points, and every renderer halts on `done()` so traversal cost is bounded too. CI green; only the two pre-existing Go-version `TypeCheckError` wording filetests fail. Open items are minor: the user-`String()`/`Error()` branch is (correctly) not capped by the builder, and no filetest drives the nested-explosion case through `println`.

## Summary
`print`/`println` of an attacker-crafted value could exhaust a validator's native Go memory: the renderer built `[]string` + `strings.Join` and `fmt.Sprintf`ed *before* gas was charged on the output, so the allocation was unbounded and untracked. #5155 added a per-collection `printLimit = 256` cap and a 64 KB `printOutputLimit`, but the 64 KB net only wrapped the Go-side `.String()` methods, never `Sprint` (the path `println` uses) — so the nested "cheap to build, O(breadth^depth) to print" shape was still open. This PR routes every value→string render through one size-capped `boundedBuilder` (`io.Writer`, 64 KB hard ceiling), and every renderer checks `done()`/`writeSep` before descending, so both the flat and nested-explosion cases are bounded to `O(printOutputLimit)` work regardless of depth or breadth. Folded in: the duplicated `String`/`Sprint`/`ProtectedString`/`ProtectedSprint` families collapse into `writeWrapped`/`writeSprint`, and `ArrayValue`/`SliceValue` share `writeArrayContents`.

```
println(deeply-shared tree)         render loop, each level:
  level 0: [c, c, c, ... x256]        for i := range elems {
  level 1: [c, c, c, ... x256]   -->    if w.writeSep(i) { break }  // <- done()==true after 64KB
  ...      shared child c               elem.writeWrapped(w, seen)  // <- returns immediately if done()
  256^depth expansions               }
                                     => productive work O(64KB), then O(depth) unwind. No blow-up.
```

## Glossary
- `boundedBuilder` — new `strings.Builder` wrapper; every write checked against `printOutputLimit`, appends `...(truncated)` once and drops further writes. Implements `io.Writer`.
- `printOutputLimit` — 64_000 bytes; global cap now enforced incrementally on every write, including the `Sprint` path.
- `printLimit` — 256; per-collection element cap producing the readable `slice[...(N elements)]` summaries.
- `done()` — reports the cap was hit; renderers consult it to stop iterating.
- `writeSprint` / `writeWrapped` — the unified raw and `(value type)` renderers; mutually recursive, both write into the builder.
- `boundedBuf` (`bounded_strings.go`) — separate, structural-only renderer for the untrusted validator panic-recovery path; never invokes user `String()`/`Error()`.

## Fix
Before: `Sprint` (via the removed `ProtectedSprint`/`ProtectedString`) built strings with no global cap, and the 64 KB net only wrapped `SliceValue.String()` and siblings — the Go-side debug path, not `println`. After: [`Sprint`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_string.go#L467-L487) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_string.go#L467-L487) and every `.String()` allocate a `boundedBuilder` and render into it; each renderer opens with `if w.done() { return }` and every element loop guards with [`writeSep`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_string.go#L151-L159) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_string.go#L151-L159). Gas is charged on `len(output)` *after* formatting in [`uversePrint`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/uverse.go#L1580-L1587) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/uverse.go#L1580-L1587), so capping the formatter's native allocation to 64 KB is the load-bearing change. `ConvertTo`'s overflow panic switches from the removed `ProtectedSprint` to a local `boundedBuilder` ([`values_conversions.go:59-61`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_conversions.go#L59-L61) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_conversions.go#L59-L61)).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`values_string.go:474-482`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_string.go#L474-L482) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_string.go#L474-L482) — the user-`String()`/`Error()` branch in `Sprint` returns `res[0].GetString()` directly, bypassing `boundedBuilder`. Correct by design (that output is produced by gas-metered Gno code, so it's allocator-bounded, not a native-exhaustion vector), but it means the PR-body claim that `print`/`println` output "is now capped at 64 KB" is not literally true for `Stringer`/`error` values. Worth one line in the doc/comment: the 64 KB cap covers the structural renderer; user `String()`/`Error()` output is bounded by gas instead.
- [`values_string.go:96-101`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_string.go#L96-L101) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_string.go#L96-L101) — `writeString`/`Write`/`writeByte` slice on a byte boundary (`s[:avail]`, `p[:avail]`), so a multi-byte UTF-8 rune can be split right before `...(truncated)`, yielding a stray invalid byte in the output. Cosmetic only (display string), but a `utf8`-aware backtrack would keep the tail valid.
- [`values_string.go:23-34`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_string.go#L23-L34) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_string.go#L23-L34) — `printOutputLimit` is described as "the maximum length of any string produced", but the truncation marker is appended *after* the cap, so the real ceiling is `printOutputLimit + len(truncatedSuffix)` (= 64_014). `TestBoundedBuilder` asserts exactly that length, so the code is self-consistent; only the prose overstates the bound. Same imprecision carried over from the #5155 review.

## Missing Tests
- **[explosion case not driven through `println`]** [`values_test.go:608-630`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_test.go#L608-L630) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_test.go#L608-L630) — the nested-explosion regression test calls `cur.String()` (`writeWrapped`), not the `Sprint` runtime path that `println` actually invokes.
  <details><summary>details</summary>

  The two entries share the same `boundedBuilder`/`done()` machinery, so coverage is real, but the security fix is specifically about the `println` → `Sprint` path, and `print4.gno` only exercises *flat* collections. A `.gno` filetest that `println`s a shared nested slice (e.g. `a := []int{...}; b := [][]int{a, a, ...}` deepened a few levels) would assert the production entry point renders bounded and instantly. This is the same gap flagged in the #5155 review ([`reviews/pr/5xxx/5155-print-truncation/1-9ae784f3/claude-opus-4-7_davd-gzl.md:52`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5155-print-truncation/1-9ae784f3/claude-opus-4-7_davd-gzl.md)); the Go-level test added here covers the mechanism but not the filetest path. Low severity.
  </details>

## Suggestions
- [`values_string.go:188-211`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/values_string.go#L188-L211) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/values_string.go#L188-L211) — `writeArrayContents` cleanly unifies the array/slice bodies; the `kind` prefix being the only difference is exactly the right seam. No change requested, noting it reads well.

## Questions for Author
- Was 64 KB re-confirmed against the original HackenProof PoC on this (now-reachable) `println` path, or carried over from #5155? An inline pointer to the ceiling's rationale would help future maintainers tune it.
- `formatUverseOutput` is called twice when native metering is enabled ([`uverse.go:1200`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/uverse.go#L1200) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/uverse.go#L1200), [`:1222`](https://github.com/gnolang/gno/blob/79497dc7c/gnovm/pkg/gnolang/uverse.go#L1222) · [↗](../../../../../.worktrees/gno-review-5788/gnovm/pkg/gnolang/uverse.go#L1222)), so `Sprint` runs twice per `println` (pre-existing, not this PR). Each pass is now bounded, so no exhaustion concern, but is the double render worth caching the formatted bytes?
