# PR #5676: feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers

**URL:** https://github.com/gnolang/gno/pull/5676
**Author:** davd-gzl | **Base:** master | **Files:** 4 | **+263 -37**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary
Ports five free functions (`Cut`, `CutPrefix`, `CutSuffix`, `Clone`, `ContainsFunc`) into `gnovm/stdlibs/bytes/bytes.gno` and three `*Buffer` methods (`Available`, `AvailableBuffer`, `Peek`) into `gnovm/stdlibs/bytes/buffer.gno` from Go 1.26.3, plus matching tests (`TestCut`, `TestCutPrefix`, `TestCutSuffix`, `TestContainsFunc`, `TestClone`, `TestWriteAppend`, `TestPeek`). The function bodies are verbatim copies of upstream Go — verified line-by-line against `/usr/lib/go/src/bytes/{bytes,buffer}.go` (Go 1.26.3 on the reviewer's box).

Three deviations are honest and clearly `// XXX:`-noted in source:
1. `TestClone` drops the upstream `unsafe.SliceData` aliasing check (no `unsafe` in gno).
2. `TestWriteAppend` drops the upstream `AllocsPerRun == 0` assertion (no `AllocsPerRun` in gno's `testing`).
3. `TestPeek` renames upstream's local `bytes` variable to `b` because gno's test file is `package bytes_test` without dot imports, so `bytes` is the package identifier and cannot be shadowed.

The third commit (`63c55f9`) additionally cleans up four pre-existing test sites (`TestEqual`, `runIndexTests` alloc block, `TestIndexRune` post-table block, `TestGrow`) that wrapped useless calls in `testing.AllocsPerRun2(...)`. `AllocsPerRun2` is a gno stub at `gnovm/tests/stdlibs/testing/testing.gno` that always returns 0 (`TODO: actually compute allocations`), making the `if allocs != 0` checks dead code. The cleanup correctly preserves `TestGrow`'s 100-iteration stress via an explicit `for` loop; the other three drop multi-iteration calls down to a single execution since their bodies operate on static input and don't depend on iteration count for correctness coverage. `TestIndexRune`'s removed post-table block (`'s'@2`, `'世'@4`) is genuinely subsumed by the main `tests` table, which already covers both the ASCII `IndexByte` fast path (`{"foo", 'o', 1}`) and the Unicode `Index(bytes-of-rune)` path (`{"a☻☺b", '☺', 4}`).

The PR is consistent with similar gno conventions: `gnovm/stdlibs/path/path_test.gno` already uses identical `// XXX: AllocsPerRun is not defined` deviation handling, and `gnovm/stdlibs/strings/strings.gno` already has `Cut`/`CutPrefix`/`CutSuffix`, so this brings `bytes` to feature parity with `strings`.

The PR description also notes (out of scope) that the upstream `iter.go` family (`Lines`, `SplitSeq`, etc.) is deferred pending generics + range-over-func support.

## Test Results
- **Existing tests:** PASS — `go test ./pkg/gnolang/ -run 'TestStdlibs$/bytes$' -v` completes in ~92s, all assertions including the new ones pass (`TestContainsFunc`, `TestCut`, `TestCutPrefix`, `TestCutSuffix`, `TestClone`, `TestPeek`, `TestWriteAppend`, `TestGrow`, `TestEqual`, `TestIndexRune`).
- **Edge-case tests:** 5 written, all PASS. Tests dropped into `gnovm/stdlibs/bytes/peek_lastread_test.gno` and run through `TestStdlibs/bytes`:
  - `TestPeekPreservesLastRead` — confirms `Peek` does not invalidate `lastRead` (a subtle property: upstream `Peek` deliberately doesn't touch `b.lastRead`, so an `UnreadByte` after `ReadByte` + `Peek` must still succeed). PASS.
  - `TestPeekEOFShortBuffer` — confirms documented behavior: `Peek(n)` with `n > Len()` returns the unread tail + `io.EOF` (not nil), and does **not** advance `b.off`. PASS.
  - `TestPeekZeroEmpty` — `Peek(0)` on an empty buffer returns `[]byte{}, nil` (since `Len() (0) < n (0)` is false). PASS.
  - `TestAvailableBufferAliasing` — confirms `AvailableBuffer` returns a slice aliasing the unused capacity tail; `append + Write` round-trips without corruption. PASS.
  - `TestCloneIndependence` — behavioral substitute for the dropped `unsafe.SliceData` assertion: mutating the input after `Clone` does not affect the clone (and vice versa). PASS.
  - `TestCutAliasing` — confirms `Cut` returns slices that alias `s` (the documented contract). Mutating `s` after `Cut` propagates to both `before` and `after`. PASS.
- CI: All `gh pr checks 5676` automated checks are green (codecov, github-bot merge requirements).

Test file preserved at `reviews/pr/5xxx/5676-bytes-cut-clone-helpers/1-63c55f963/tests/peek_lastread_test.gno`.

## Critical (must fix)
- None.

## Warnings (should fix)
- None.

## Nits
- `gnovm/stdlibs/bytes/buffer.gno:349-354` — `Peek` inherits upstream's behavior of panicking with a slice-bounds error on negative `n` (because `b.Len() < n` is false for any negative `n`, then `b.buf[b.off:b.off+n]` panics). This matches upstream Go 1.26.3 exactly, so it's a faithful port, but worth noting since gno applications may not realize the failure mode. Not a blocker.

## Missing Tests
- None within the scope of this PR. Every new function has table-driven coverage matching upstream, plus the five adversarial tests above passed.
- Note (out-of-scope): the deviation comments faithfully document what was dropped, but the gno `bytes` package has no replacement way to detect allocation regressions. If a future PR adds an allocation counter to gno's `testing`, the `// XXX: AllocsPerRun` blocks here become a ready-to-restore TODO list. No action needed in this PR.

## Suggestions
- (Optional, out-of-scope) `strings.Clone` and `strings.ContainsFunc` are not in gno's `strings` stdlib either. A follow-up PR could bring `strings` to the same parity. Not a blocker for this PR.
- (Optional) The `TestEqual` cleanup at `gnovm/stdlibs/bytes/bytes_test.gno:69` drops a 10x iteration count down to 1. The body iterates the full `compareTests` table on each run, so coverage is identical, but if the goal is to detect non-determinism in `bytes.Equal` (e.g. timing-dependent string interning), preserving a small loop (`for i := 0; i < 10; i++`) like in `TestGrow` would be more defensible. Same point applies to `runIndexTests` at `bytes_test.gno:239`. Low priority — `bytes.Equal` is pure.
- (Optional, nit) The `// XXX:` comment style is consistent with surrounding code, but could mention the upstream Go version (1.26.3) inline for future port-comparison audits. Not blocking.

## Questions for Author
- The `// XXX: AllocsPerRun` deviations explicitly note what was dropped, but none flag the upstream Go version they're tracking. If the upstream `bytes_test.go` block evolves (e.g. Go 1.27 changes the table), a future reviewer has no anchor. Worth adding "(upstream: Go 1.26.3, file/line)" to each deviation? (Style choice, not a request.)
- Is there a tracked issue for porting the deferred `iter.go` family? The PR description mentions it as a follow-up — linking a tracker would help future contributors pick it up.

## Verdict
APPROVE — Verbatim port of well-known upstream Go 1.26 functions, all deviations are honest and well-justified, all tests pass (including five adversarial tests I wrote to probe `Peek`/`AvailableBuffer`/`Clone`/`Cut` aliasing semantics), and the alloc-stub cleanup is a legitimate improvement that removes dead `if allocs != 0` checks that could never fire.
