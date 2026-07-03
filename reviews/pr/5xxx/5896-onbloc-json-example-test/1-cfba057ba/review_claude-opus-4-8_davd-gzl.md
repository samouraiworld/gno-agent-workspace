# PR [#5896](https://github.com/gnolang/gno/pull/5896): fix: p/onbloc/json/node_test Example test

URL: https://github.com/gnolang/gno/pull/5896
Author: jefft0 | Base: master | Files: 1 | +4 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: cfba057ba (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5896 cfba057ba`

**TL;DR:** A test in `p/onbloc/json` had an `// Output:` comment that was never actually checked, because the function was named like a normal test and printed nothing. This renames it and prints to stdout so the expected output is now enforced.

**Verdict: APPROVE** — test-only fix, does exactly what it claims; no open concerns. Already approved on GitHub by davd-gzl and notJoon.

## Glossary
- example test: a `ExampleXxx()` function in a `_test.gno` file whose trailing `// Output:` comment is compared against what the function prints.

## Summary
`TestNode_ExampleMust` looked like an example test but was not one: the name started with `Test`, so the runner ran it as an ordinary test, and its last statement was `ufmt.Sprintf(...)` whose result was discarded, so nothing reached stdout. The `// Output: // Object has 1 inheritors inside` directive was therefore inert decoration. This PR renames it to `Example_TestNode_Must` and swaps the two `Sprintf` calls for `Printf`, so the function prints and the runner now compares that print against `// Output:`. The `if root.Size() != 1` branch also switches from `t.Errorf` (no `*testing.T` in an example signature) to `Printf` plus `return`.

## Fix
Before: [`node_test.gno:1079`](https://github.com/gnolang/gno/blob/cfba057ba/examples/gno.land/p/onbloc/json/node_test.gno#L1079) · [↗](../../../../../.worktrees/gno-review-5896/examples/gno.land/p/onbloc/json/node_test.gno#L1079) was `func TestNode_ExampleMust(t *testing.T)`, a plain test whose `ufmt.Sprintf` result was thrown away, so the `// Output:` at [`node_test.gno:1102-1103`](https://github.com/gnolang/gno/blob/cfba057ba/examples/gno.land/p/onbloc/json/node_test.gno#L1102-L1103) · [↗](../../../../../.worktrees/gno-review-5896/examples/gno.land/p/onbloc/json/node_test.gno#L1102) was checked against nothing. After: `func Example_TestNode_Must()` with `ufmt.Printf(...)`, so the example runner enforces the `// Output:` line. The load-bearing constraint is the example-test contract: the name must start with `Example` and the function must print to stdout for `// Output:` to run.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`node_test.gno:1096-1099`](https://github.com/gnolang/gno/blob/cfba057ba/examples/gno.land/p/onbloc/json/node_test.gno#L1096-L1099) · [↗](../../../../../.worktrees/gno-review-5896/examples/gno.land/p/onbloc/json/node_test.gno#L1096) — the `if root.Size() != 1` guard is now redundant. In the example form, a wrong size already produces a wrong printed line and fails the `// Output:` check, so the extra `Printf`/`return` branch adds no coverage. Harmless; no change needed.

## Missing Tests
None. The change is itself the fix that makes an existing assertion real.

## Suggestions
None.

## Open questions
None.
