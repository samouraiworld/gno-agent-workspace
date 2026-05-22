# PR #5407: fix(gnovm): skip closure frames in `AssertOriginCall` origin check

**URL:** https://github.com/gnolang/gno/pull/5407
**Author:** notJoon | **Base:** master | **Files:** 5 | **+81 -6**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary
This PR fixes issue #3918 where `runtime.AssertOriginCall()` incorrectly panicked with "invalid non-origin call" when invoked through an anonymous function (closure). The root cause was that `isOriginCall` in `gnovm/stdlibs/chain/runtime/native.go` determined origin status by comparing `Machine.NumFrames()` against a hardcoded threshold (`n <= 2`). Closures add a frame when invoked, but they are implementation details of the enclosing function, not separate method calls — causing false negatives.

The fix adds `Machine.NumNonClosureFrames()` to `gnovm/pkg/gnolang/machine.go`, which iterates all frames and counts only those where `fr.Func != nil` (call frames, not basic for/switch/range frames) and `!fr.Func.IsClosure`. Both the production `isOriginCall` (`stdlibs/chain/runtime/native.go`) and the testing version (`tests/stdlibs/chain/runtime/testing_runtime.go`) are updated to use this new method.

Two new file tests are added:
- `std13.gno` — verifies `AssertOriginCall` succeeds when called through a single closure wrapper (`main → closure → Register → AssertOriginCall`)
- `std14.gno` — verifies `AssertOriginCall` correctly panics when called through a named function wrapper (`main → notPanics → closure → Register → AssertOriginCall`), since `notPanics` adds a non-closure frame

Security analysis confirms the fix is safe: closures exist within the same realm, and cross-realm calls always use named functions. An attacker cannot inject closures between realms to reduce the non-closure frame count — e.g., `MsgCall → Attack() → closure → victim.Register(cross) → AssertOriginCall()` still has 3 non-closure frames (Attack, Register, AssertOriginCall) and correctly rejects.

## Test Results
- **Existing tests:** PASS (std9.gno, std13.gno, std14.gno all pass)
- **Edge-case tests:** skipped

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `gnovm/pkg/gnolang/machine.go:2112-2122` — `NumNonClosureFrames()` skips frames where `fr.Func == nil` (basic frames from for/switch/range statements), but the method name only says "NonClosure". This silently changes behavior for code like `func Register() { for i := 0; i < 1; i++ { runtime.AssertOriginCall() } }` — previously this would have failed (basic frame counted), now it passes (basic frame skipped). While arguably the correct behavior, the method name should reflect that it counts "non-closure call frames" not just "non-closure frames". Consider renaming to `NumCallFrames` or `NumNonClosureCallFrames`, or at minimum documenting that `Func == nil` frames (basic blocks) are excluded.
- [ ] `gnovm/tests/files/std14.gno:28` — The label `"direct call panicked: true"` is misleading. Both calls go through the `notPanics` wrapper (a non-closure named function), which adds an extra non-closure frame. Neither call is "direct". The second call is identical in structure to the first. A clearer label would be `"second anon call panicked: true"` or simply `"call through wrapper panicked: true"`.

## Nits
- [ ] `gnovm/tests/files/std13.gno` — This test only covers a single level of closure wrapping. Adding a test with nested closures (`func() { func() { Register() }() }()`) would strengthen confidence that the fix handles arbitrary closure depth.

## Missing Tests
- [ ] No test for `AssertOriginCall` invoked inside a for/switch/range block (basic frame scenario). This is a behavioral change from the old code.
- [ ] No test for deeply nested closures (3+ levels) calling `AssertOriginCall`.
- [ ] No Go-level unit test for `NumNonClosureFrames()` itself with various frame configurations.

## Suggestions
- Consider adding a test case where a closure is used across a realm boundary to confirm the security property holds: `MsgCall → RealmA.Attack() → closure → RealmB.Register(cross) → AssertOriginCall()` should still panic (3 non-closure frames > 2).
- The PR also implicitly fixes a pre-existing issue where `AssertOriginCall` inside a for-loop body would fail due to the basic frame being counted. If intentional, this should be mentioned in the PR description; if unintentional, verify it's the desired behavior.

## Questions for Author
- Was the behavioral change for basic frames (for/switch/range) intentional? The old code would count these frames; the new code skips them.
- Should `std13.gno` also test nested closures (e.g., `func() { func() { Register() }() }()`) to cover deeper nesting?

## Verdict
APPROVE — Fix is correct and security analysis holds; minor naming and test coverage suggestions.
