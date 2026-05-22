# PR #5563: feat(gnodev): add gnodev version command

**URL:** https://github.com/gnolang/gno/pull/5563
**Author:** AmozPay | **Base:** master | **Files:** 8 | **+138 -3**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Round 3 review at HEAD `0a6ed8b22`. Adds `gnodev version` for parity with `gno version` / `gnokey version` (resolves #5550). Builds on the round 2 state (`9f27b3e9b`, already APPROVE'd) by addressing test-hygiene feedback and adding integration coverage. New work since round 2:

1. **`contribs/gnodev/version_test.go`** — Renamed `TestClient_Version` → `TestVersionCmd`. `t.Cleanup` is now registered before the package-global `version.Version` mutation, closing the race noted in round 2. Unused `versionValues[0]` slice element dropped; single `versionValue := "chain/test4.2"` literal is used.
2. **`contribs/gnodev/testdata_test.go`** (new, 32 lines) — Functional test that `go build`s the gnodev binary into `t.TempDir()` with `-X tm2/pkg/version.Version=testscript-version`, registers it as a testscript command, and runs `testdata/*.txtar`.
3. **`contribs/gnodev/testdata/version.txtar`** (new, 7 lines) — Single script: invokes `gnodev version`, asserts stdout is `gnodev version: testscript-version` and stderr is empty.
4. **`gnovm/pkg/integration/testscript.go`** — Adds two exported helpers: `RegisterExecCommand(p *testscript.Params, name, bin string)` (registers an external binary as a testscript command, panics on duplicate name) and `RunTestscript(t *testing.T, p testscript.Params)` (one-line wrapper around `testscript.Run`). Stated motivation in the commit message: keep `contribs/gnodev` from importing `go-internal/testscript` directly.

The unmodified-since-round-2 surface (`version.go`, `main.go`, `Makefile`, `README.md`) carries forward the round 2 nits unchanged. Local verification: `go build ./...` and `cd contribs/gnodev && go build ./...` succeed; the txtar test runs the freshly built binary and prints `gnodev version: testscript-version` exactly as asserted.

## Test Results

- **Existing tests:** PASS
  - `go test -run TestVersionCmd ./contribs/gnodev/` → `PASS: TestVersionCmd (0.00s)`
  - `go test -run TestScripts ./contribs/gnodev/` → `PASS: TestScripts/version (0.03s)` (3.04s total, dominated by `go build`)
  - `go test ./gnovm/cmd/gno/...` and `go test ./gnovm/pkg/integration/...` compile cleanly with the new helpers added to `testscript.go`.
- **CI:** all green except the `Merge Requirements` review-approval gate (manual). Codecov flags 41% patch coverage — the uncovered lines are inside `RegisterExecCommand`'s success path (no negative-case test) and the gnodev `main.go` registration line.
- **Edge-case tests:** skipped — change is too small to warrant adversarial tests.

## Critical (must fix)

- None.

## Warnings (should fix)

- None new this round. Round 2 carried no critical/warning items into round 3 either.

## Nits

- [ ] `contribs/gnodev/version.go:19` — Closure parameter `args []string` is still unused; rename to `_ []string` for the same reason flagged in round 2. Sibling `gnovm/cmd/gno/version.go:19` has the same shape, so this is also a copy-paste artifact and out of scope to fully fix here. Carried forward from round 2.
- [ ] `contribs/gnodev/Makefile:1` — Stray leading blank line still present; sibling Makefiles (`gnovm/Makefile`, `gno.land/Makefile`) start with `VERSION ?=` on line 1. Carried forward from round 2.
- [ ] `contribs/gnodev/main.go:67-68` — No blank line between `cmd.AddSubCommands(newVersionCmd(stdio))` and the `// XXX:` block; pre-PR formatting kept one. Carried forward from round 2.
- [ ] `contribs/gnodev/testdata_test.go:26` — `buildCmd.Dir = "."` is redundant; `go test` already runs in the package directory. Removing the line keeps the test minimal.
- [ ] `gnovm/pkg/integration/testscript.go:64-67` — `RunTestscript` is a one-line passthrough (`testscript.Run(t, p)` plus `t.Helper()`). The only caller is `contribs/gnodev/testdata_test.go`. The wrapper exists to keep gnodev from importing `go-internal/testscript`, but `gnovm/cmd/gno/testdata_test.go:9` still imports `go-internal/testscript` directly, so the "shared pattern" claim is not enforced repo-wide. Either drop `RunTestscript` and let the (single) caller do `testscript.Run(t, p)` itself, or migrate `gnovm/cmd/gno` and the gno.land integration suite to the wrapper for consistency.
- [ ] `gnovm/pkg/integration/testscript.go:46` — `panic(fmt.Errorf(...))` on duplicate command name. A returned error would let test setup fail through `require.NoError`, which is the convention elsewhere in this file (e.g. `SetupTestscriptsCoverage` returns an error). Stylistic; not load-bearing.
- [ ] `gnovm/pkg/integration/testscript.go:46` — Typo in panic message: `"unable register %q"` → `"unable to register %q"`.

## Missing Tests

- [ ] No negative-case coverage of `RegisterExecCommand` — duplicate registration (panic path), nor a `neg` invocation in a `.txtar` (`! gnodev nonsense-subcommand` would exercise the `successExpected != commandSucceeded` branch at `testscript.go:57-59`).
- [ ] `gnodev version` is invoked only at the root level. A `.txtar` line for `gnodev version --help` or `gnodev local version` (the latter should fail — `version` is a sibling, not a child of `local`) would lock in the routing decision made in `main.go:65-67`.

## Suggestions

- Consider renaming `RegisterExecCommand` → `RegisterBinary` or `RegisterExternalCommand`. "Exec" reads as the `os/exec` package; the helper actually wires `ts.Exec` into a testscript command map.
- Out of scope for this PR, but the `VERSION ?= $(shell git describe ...)` recipe is now duplicated in `gnovm/Makefile`, `gno.land/Makefile`, and `contribs/gnodev/Makefile`. A shared `Makefile.common` include would be a worthwhile follow-up.
- `testdata_test.go` builds the binary in-process via `os/exec`. If `gnodev` ever grows additional txtar suites, factor the build step into a `t.Helper()` so each suite reuses the same binary path. Not necessary for this PR.

## Questions for Author

- Why panic instead of returning an error from `RegisterExecCommand` on duplicate registration? The sibling helper `SetupTestscriptsCoverage` (in `testscript_coverage.go`) returns an error.
- Was migrating `gnovm/cmd/gno/testdata_test.go` to the new `RunTestscript` wrapper considered? Leaving it on the direct `testscript.Run` call weakens the "shared integration pattern" rationale in the commit message.

## Verdict

APPROVE — Round 2 feedback (test cleanup ordering, dead slice element, test name) is fully addressed. The new txtar suite gives end-to-end coverage of the routing that the round 2 unit test could not. Outstanding items are stylistic nits inherited from round 2 plus minor design observations on the new `integration` helpers; none should block merge. CI is green except the manual review-approval gate.
