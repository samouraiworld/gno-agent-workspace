# PR #5385: feat(gnovm): add errors.Unwrap, errors.Is, and errors.Join to stdlib

**URL:** https://github.com/gnolang/gno/pull/5385
**Author:** davd-gzl | **Base:** master | **Files:** 16 | **+334 -61**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR implements `errors.Unwrap`, `errors.Is`, and `errors.Join` for the Gno stdlib, bringing the `errors` package closer to full Go parity (only `errors.As` remains absent, blocked on `reflect`).

**New files:**
- `gnovm/stdlibs/errors/wrap.gno` — `Unwrap`, `Is`, and the internal `is`/`comparable` helpers
- `gnovm/stdlibs/errors/join.gno` — `Join` and its private `joinError` type
- `gnovm/stdlibs/errors/wrap_test.gno` — table-driven tests for `Is`/`Unwrap`
- `gnovm/stdlibs/errors/join_test.gno` — tests for `Join`
- `gnovm/tests/files/errors_is_filetest.gno` — VM-level filetest for all three functions

**Modified files (consumers):**
- `gnovm/stdlibs/errors/errors.gno` — doc comment cleanup; removes stale `As` text
- `gnovm/stdlibs/errors/README.md` — rewritten to document the full API and the Gno-specific `comparable` approach
- `gnovm/stdlibs/encoding/csv/reader_test.gno` — replaces hand-rolled `isErr()` with `errors.Is`
- `gnovm/tests/stdlibs/fmt/errors_test.gno` — replaces hand-rolled `errorUnwrap()` with `errors.Unwrap`
- `gnovm/stdlibs/strconv/atoi_test.gno` — un-gates `TestNumErrorUnwrap` (was behind `/* XXX */` comment)
- `examples/gno.land/p/nt/uassert/v0/uassert.gno` — rewrites `ErrorIs` to use `errors.Is` for real chain traversal
- Various test files updated to use `errors.Is` / `uassert.ErrorIs`
- `examples/gno.land/p/demo/tokens/grc721/grc721_metadata_test.gno` — corrects previously wrong assertions (function never validated ownership/existence)
- `examples/gno.land/p/demo/tokens/grc721/grc721_royalty_test.gno` — fixes a variable capture bug (`derr` → `iderr`)
- `examples/gno.land/r/devrels/events/events_test.gno` — reorders test assertions so they run after state is populated

**Design decision for comparability:** Because Gno lacks `reflect`, the `comparable(v error) bool` helper uses `defer/recover` around `_ = v == v` to detect non-comparable types at runtime. This is a reasonable workaround documented in the README.

**No ADR was included.** Per `gno/AGENTS.md`, every non-trivial AI-assisted PR requires one. The PR description does not indicate AI assistance, but the ADR policy applies regardless for non-trivial stdlib additions.

## Test Results

- **Existing tests:** Go build and `go vet` of `./gnovm/...` pass cleanly. `./gnovm/...` Go tests pass (no relevant tests ran via `go test` since the changed code is `.gno`; VM integration tests require `gno` binary not available). No Go test failures observed.
- **Edge-case tests:** skipped (gno binary not available to run `.gno` tests)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/stdlibs/errors/wrap.gno:28` — Doc comment says **"breadth first traversal"** but the actual `is()` function uses **depth-first traversal** (recursive calls on each element of `Unwrap() []error`, matching Go stdlib). Go stdlib explicitly says "depth-first" (`/usr/local/go/src/errors/wrap.go:31`). This is a misleading doc that will confuse future readers and package users.

- [ ] `gnovm/stdlibs/errors/wrap.gno:41-48` — The Gno `Is()` only short-circuits on `target == nil`, but Go stdlib short-circuits on `err == nil || target == nil`. For `err == nil && target != nil`, the Gno impl unnecessarily calls `comparable(target)` and enters `is()`, where it correctly returns `false` via the `default` branch. Functionally identical, but a micro-inefficiency and a divergence from the reference that makes side-by-side comparison harder. Consider adding `if err == nil { return false }` before the `comparable` call.

- [ ] `examples/gno.land/p/demo/tokens/grc721/grc721_metadata_test.gno:54-89` — The PR removes assertions that `SetTokenMetadata` validates token existence and caller ownership, replacing them with `uassert.NoError`. The comments explain this correctly, but the underlying `metadataNFT.SetTokenMetadata` implementation (which just calls `s.extensions.Set(tid.String(), metadata)` without any validation) appears to be a pre-existing design gap rather than an intentional choice. This PR should not silently accept these weaker guarantees without at least a TODO or a follow-up issue reference. As-is, non-owners can overwrite any token's metadata — this is a correctness gap in the NFT implementation.

- [ ] `docs/resources/go-gno-compatibility.md:164` — The `errors` package is listed as `part`. With `Unwrap`, `Is`, and `Join` now added, the only remaining gap is `As`. The status should be updated (e.g., to reflect that only `errors.As` is missing), or a note added. This PR touches the `errors` package but does not update the compatibility doc.

## Nits

- [ ] `gnovm/stdlibs/errors/wrap.gno:77-88` — The `comparable` helper is unexported but documented only via an inline comment. Since this is a Gno-specific mechanism that differs from Go stdlib, a slightly fuller comment explaining _why_ `v == v` is used (not just `v == someTarget`) would help. The `v == v` comparison probes the type's comparability, not whether `v` equals some other value.

- [ ] `gnovm/stdlibs/errors/join.gno:40-49` — The Go stdlib `joinError.Error()` has a fast path for `len(e.errs) == 1` that avoids building a byte slice. The Gno version skips this optimization. Minor, but worth noting for consistency with the source it claims to port.

- [ ] `gnovm/stdlibs/errors/errors.gno:9` — Package-level comment now lists "Unwrap, Is, and Join" but the package doc comment body still says `errors.Unwrap(fmt.Errorf("... %w ...", ..., err, ...))` as an example. This example is fine but there is no equivalent example for `Is` or `Join` in the doc comment. Minor inconsistency.

## Missing Tests

- [ ] No test for `errors.Is` where `err == nil && target != nil` — the filetest covers `Is(nil, nil)` and `Is(err, nil)` but not `Is(nil, nonNilErr)`. Should return `false`; worth an explicit test case in `wrap_test.gno` or the filetest.
- [ ] No test for deeply nested wrapping with `errors.Join` (e.g., `Join(Join(a, b), c)` and `Is` traversal across join layers). The current `TestJoinIs` only tests a single level of `Join`.
- [ ] No test covering `comparable()` panic recovery path from the perspective of `Is()` when `target` is a non-comparable type _without_ an `Is(error) bool` method (i.e., pure non-comparable sentinel). The test uses `errorUncomparable` which _has_ an `Is` method, so the code path `targetComparable=false, no Is method` on the target is not exercised.

## Suggestions

- The `comparable` function is not exported, but its behavior is part of the semantics of `errors.Is`. Consider a brief mention in the `Is` doc comment: "For types without reflect support, comparability is probed at runtime." This sets the expectation for Gno users who port code from Go.
- The ADR requirement in `gno/AGENTS.md` applies to all non-trivial stdlib additions. Even a short ADR (`gnovm/adr/pr5385_errors_is_join.md`) covering the comparability approach and the decision to omit `errors.As` would satisfy the requirement and provide useful documentation.
- The fix to `grc721_royalty_test.gno` (using `iderr` instead of `derr`) is a correct bug fix but is orthogonal to the errors stdlib feature. It could be a separate commit or at least called out explicitly in the PR description.

## Questions for Author

- Was the `comparable(v error) bool` approach tested under GnoVM's panic/recover semantics? Specifically: does a panic from a map/slice comparison inside a stdlib function correctly get caught by `recover()` in the same stdlib function? If there are any differences in GnoVM's `recover()` scoping, `errors.Is` could panic on uncomparable types instead of returning `false`.
- Is the "breadth first" comment in the `Is` doc an intentional design divergence from Go's depth-first, or a copy/paste error? The actual code is depth-first (matches Go). If it's a typo, fix it; if depth-vs-breadth is intentional, both the comment and the implementation need to match.
- For the `grc721_metadata_test.gno` changes: is the lack of ownership validation in `SetTokenMetadata` a known issue or a deliberate design choice? If deliberate, a comment explaining the rationale (e.g., "any caller can annotate any token") would help. If a bug, a tracking issue reference would be appropriate.

## Verdict

REQUEST CHANGES — The implementation is correct and the approach is sound, but the "breadth first" doc bug in `Is()` is misleading, the grc721 metadata ownership gap deserves a tracking issue reference, the compatibility doc was not updated, and the mandatory ADR is missing.
