# PR #5672: fix(examples/urequire): delegate `NotAborts` to `uassert.NotAborts`

**URL:** https://github.com/gnolang/gno/pull/5672
**Author:** davd-gzl | **Base:** master | **Files:** 1 | **+1 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

One-line fix to [examples/gno.land/p/nt/urequire/v0/urequire.gno:75](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/nt/urequire/v0/urequire.gno#L75): the wrapper `urequire.NotAborts` previously delegated to `uassert.NotPanics`, which is a superset (catches both same-realm panics via `defer recover` and cross-realm aborts via `revive`). This PR re-points the delegation to `uassert.NotAborts`, which only catches cross-realm aborts via `revive`.

This brings the wrapper's runtime behavior in line with both its name and its docstring at [urequire.gno:69-72](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/nt/urequire/v0/urequire.gno#L69-L72):
> NotAborts requires that the code inside the specified func does NOT abort when crossing an execution boundary (e.g., VM call). Use NotPanics for requiring the absence of local panics within the same realm.

The sibling wrapper `urequire.NotPanics` already delegates correctly to `uassert.NotPanics`, so this PR restores the symmetric `NotAborts → NotAborts` / `NotPanics → NotPanics` mapping that the rest of the file follows for all other wrappers.

### Behavioral implication

This is a behavioral change, not a pure refactor:

- **Before:** if user code passed to `urequire.NotAborts(t, f)` panicked locally (same realm), the wrapper caught it via `uassert.NotPanics`' inner `defer recover()` and reported a clean assertion failure.
- **After:** local same-realm panics in `f` are no longer caught by `uassert.NotAborts` (which only wraps in `revive`). They will propagate as unhandled test panics rather than producing the wrapper's normal "should not abort" assertion failure.

Blast-radius check: `grep -rn "urequire\.NotAborts" examples/ gnovm/` returns zero callers in the entire repository. The only matches for `NotAborts` inside the urequire package are the wrapper's own definition and its docstring cross-references. So no test or realm in-tree breaks from the narrower semantics.

## Test Results

- **Existing tests (gno test ./examples/gno.land/p/nt/urequire/v0/):** PASS (`TestPackage`, the only test; +GAS 410102).
- **Existing tests (gno test ./examples/gno.land/p/nt/uassert/v0/):** PASS (full suite including `TestNil`, `TestNotNil`, `TestTypedNil`, `TestNotTypedNil`).
- **CI status:** `build`, `e2e-test`, `gno-checks/{fmt,lint,test}`, `gno2go`, `mod-tidy`, `stdlibs/{fmt,lint}` all green. Validator scenarios pending (orthogonal to this change). The "Merge Requirements" failure is the `review/triage-pending` gate, not a real CI failure.
- **Edge-case tests:** skipped — the change is a one-token delegation swap with zero in-tree callers, and the upstream `uassert.NotAborts` already has its own tests.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [ ] [urequire.gno:71](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/nt/urequire/v0/urequire.gno#L71) — Pre-existing. `uassert.NotAborts` itself carries the comment "Consider using NotPanics which checks for both panics and aborts." ([uassert.gno:157](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/nt/uassert/v0/uassert.gno#L157)). After this PR, the analogous nudge could be added to `urequire.NotAborts` ("Use NotPanics for requiring the absence of local panics") — but the docstring already says exactly that, so it's a wash.

## Missing Tests

- [ ] No direct test exercises the new (now narrower) behavior of `urequire.NotAborts`. The package's lone `TestPackage` only asserts `Equal(t, 42, 42)` ([urequire_test.gno:6](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/nt/urequire/v0/urequire_test.gno#L6)). The file header `// XXX: codegen the package.` and the inline `// XXX: find a way to unit test this package thoroughly` already acknowledge this gap, and the gap is package-wide, not specific to this PR.

## Suggestions

- A follow-up PR could add a mock-`TestingT`-based test file mirroring [uassert/v0/mock_test.gno](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/nt/uassert/v0/mock_test.gno), exercising both "success path returns" and "failure path triggers FailNow" for each wrapper, including a regression case proving that `urequire.NotAborts` now lets local panics escape (as intended).
- Worth opening a tracking issue for the `XXX: codegen the package.` note: with 20 wrappers all in the same shape (`t.Helper(); if uassert.X(...) { return }; t.FailNow()`), `go generate` would prevent this exact class of mis-delegation bug from recurring.

## Questions for Author

- Is there a known caller of `urequire.NotAborts` outside the `gnolang/gno` monorepo (community realms, demo realms) that relied on the broader "catches local panics too" behavior? If yes, this is worth a one-line entry in the PR description to flag the semantic change.

## Verdict

**APPROVE** — Correct, minimal, well-scoped fix. The wrapper now matches its name and its docstring. Behavioral change is real but narrowly so (only affects callers that pass functions panicking locally), and no in-tree caller is affected. CI is green where it matters; pending checks are orthogonal scenarios and the bot triage gate.
