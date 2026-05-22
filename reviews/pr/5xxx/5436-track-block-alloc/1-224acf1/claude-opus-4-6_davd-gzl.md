# PR #5436: fix(gnovm): track block item allocations in PrepareNewValues

**URL:** https://github.com/gnolang/gno/pull/5436
**Author:** omarsy | **Base:** master | **Files:** 9 | **+87 -11**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a GC allocation/recount mismatch in GnoVM that caused infinite recursion to trigger the wrong panic: `"should not happen, allocation limit exceeded while gc."` instead of the expected `"allocation limit exceeded"`.

The root cause is in `PackageNode.PrepareNewValues()` (`nodes.go:1437`). When new package-level declarations are added after initial package creation, their block items are appended to the package block's `Values` slice via `block.Values = append(block.Values, nvs...)` without calling `AllocateBlockItems`. During normal execution, `alloc.bytes` doesn't account for these items. But during GC recount, `Block.GetShallowSize()` computes the block size using `len(b.Values)`, which includes the untracked items. The mismatch is small (~80 bytes for 2 functions at 40 bytes each), but during infinite recursion the allocator fills to near-max, and the untracked bytes cause the GC recount to exceed `maxBytes`, triggering the "should not happen" panic path in `alloc.go:142`.

The fix adds a single line: `alloc.AllocateBlockItems(int64(len(nvs)))` before appending to `block.Values`. This charges the allocator (and gas meter) for the block items so GC recount stays consistent. Gas values in 6 test files are updated accordingly (+40 per additional block item tracked at 40 bytes each).

The PR also includes a new filetest `alloc_12.gno` that verifies infinite recursion produces the correct "allocation limit exceeded" error, and an ADR documenting the analysis.

## Test Results
- **Existing tests:** PASS — all alloc file tests (including alloc_10_long, alloc_10a_long, alloc_10b_long), gas file tests (const.gno, nested_alloc.gno, slice_alloc.gno), and TestAddPkgDeliverTx all pass.
- **Edge-case tests:** skipped — the new `alloc_12.gno` filetest adequately covers the fix.

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `gnovm/adr/pr5436_fix_gc_alloc_mismatch.md:1` — The ADR title uses a generic format `# Fix GC allocation/recount mismatch` instead of the convention `# PR5436: Fix GC allocation/recount mismatch`. Minor, but should match the established ADR naming pattern.

## Nits
- [ ] PR body mentions "Fix pre-existing nil pointer panic in `GetLocalIndex` debug logging" but no such change is present in the diff. Either remove this claim from the PR body or include the fix.

## Missing Tests
None — The `alloc_12.gno` filetest directly validates the fix by asserting infinite recursion produces "allocation limit exceeded" and not the GC panic.

## Suggestions
- The gas value changes are all arithmetically consistent with `allocBlockItem = 40` bytes per item. The `const.gno` test increases by +80 (2 declarations: `s` constant + `main` function), `nested_alloc.gno` by +40 (1 `main` function), and the integration tests by +40 or +80 per `addpkg` call. This is a good sign — no unexpected side effects.
- Consider adding a comment at `nodes.go:1437` explaining why `AllocateBlockItems` is needed here (to keep allocator and GC recount consistent), since the `block.Values = append(...)` pattern without allocation tracking was a subtle enough bug that it went undetected.

## Questions for Author
- Are there other places in the codebase where block items are appended to an existing block without going through `AllocateBlockItems`? A grep for `block.Values = append` or `.Values = append` might reveal other instances with the same mismatch pattern.

## Verdict
APPROVE — Precise, well-analyzed single-line fix for a real allocator/GC inconsistency. The root cause analysis is thorough, the ADR clearly documents the decision, the gas value changes are arithmetically consistent, and the new filetest directly validates the fix. The only nit is the PR body claiming a `GetLocalIndex` fix that isn't present in the diff.
