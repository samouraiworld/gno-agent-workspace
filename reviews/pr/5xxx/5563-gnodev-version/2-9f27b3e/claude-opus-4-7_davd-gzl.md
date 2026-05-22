# PR #5563: feat(gnodev): add gnodev version command

**URL:** https://github.com/gnolang/gno/pull/5563
**Author:** AmozPay | **Base:** master | **Files:** 5 | **+68 -3**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Adds a `gnodev version` subcommand for parity with `gno version` and `gnokey version` (resolves #5550). Round 2 review at commit `9f27b3e9b` — the author addressed the round 1 feedback by writing a self-contained `version.go` for gnodev rather than reusing the gnokey `client.NewVersionCmd`, fixing both the wrong "gnokey version:" output string and the unnecessary cross-package import.

Files:

1. **`contribs/gnodev/version.go`** (new, 24 lines) — Defines `newVersionCmd(io commands.IO)` that prints `"gnodev version: " + version.Version`. Mirrors the structure of `gnovm/cmd/gno/version.go` and `gno.land/cmd/gnoland/version.go` exactly.
2. **`contribs/gnodev/version_test.go`** (new, 39 lines) — Direct unit test of `newVersionCmd` via `ParseAndRun`. Mutates package-global `version.Version` to verify the output reflects the build-time injected value.
3. **`contribs/gnodev/main.go`** — Registers the new command via `cmd.AddSubCommands(newVersionCmd(stdio))`. Two surrounding blank lines were collapsed.
4. **`contribs/gnodev/Makefile`** — Adds `VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || ...)` and injects it via `-ldflags "-X github.com/gnolang/gno/tm2/pkg/version.Version=$(VERSION)"`. Same recipe as `gnovm/Makefile:36` and `gno.land/Makefile:21`.
5. **`contribs/gnodev/README.md`** — Adds `version` to the `SUBCOMMANDS` block (regenerated via `make generate`).

Verified locally: `make build` succeeds; `./build/gnodev version` prints `gnodev version: HEAD.3045+9f27b3e9b`; `./build/gnodev -h` lists the new subcommand.

## Test Results

- **Existing tests:** PASS — `go test -v -run TestClient_Version ./contribs/gnodev/...` passes (`PASS: TestClient_Version (0.00s)`). All sibling packages compile and test.
- **Edge-case tests:** skipped — change is too small to warrant adversarial tests.

## Critical (must fix)

- None.

## Warnings (should fix)

- [ ] `contribs/gnodev/version_test.go:36-38` — `t.Cleanup(func() { version.Version = originalVersion })` is registered **after** the assertions. If `require.NoError` (line 27) ever calls `FailNow`, the cleanup is never registered and `version.Version` stays mutated to `"chain/test4.2"` for any subsequent test in the package. Move the `t.Cleanup` call up to right after `originalVersion := version.Version` (line 23), before the mutation.
- [ ] `contribs/gnodev/version_test.go:22` — `versionValues := []string{"develop", "chain/test4.2"}` declares two values but only `versionValues[1]` is ever used. Either drop the slice and use a single string, or actually iterate both cases (e.g. test the default `"develop"` value as well).

## Nits

- [ ] `contribs/gnodev/version.go:19` — Closure parameter `args []string` is unused; rename to `_` to match the surrounding style. (Same nit applies to `gnovm/cmd/gno/version.go:19` and `gno.land/cmd/gnoland/version.go:19`, so it's also a copy-paste artifact.)
- [ ] `contribs/gnodev/Makefile:1` — Stray leading blank line introduced before `VERSION ?=`. The sibling Makefiles do not have this.
- [ ] `contribs/gnodev/main.go:67-68` — Blank line between `AddSubCommands(NewStagingCmd(stdio))` and `AddSubCommands(newVersionCmd(stdio))` was removed; the style elsewhere keeps a blank line before the trailing `// XXX` comment block. Restoring it would match the pre-PR formatting.
- [ ] `contribs/gnodev/version_test.go:14` — Test name `TestClient_Version` is misleading (there is no client involved, and the underscore convention is unusual in this package). Suggest `TestNewVersionCmd` or `TestVersionCmd`.

## Missing Tests

- [ ] No integration coverage that exercises `gnodev version` through the root `cmd`. The current test invokes `versionCmd.ParseAndRun` directly, so a regression that drops `cmd.AddSubCommands(newVersionCmd(stdio))` from `main.go` would not be caught.

## Suggestions

- Refactor `newVersionCmd(io commands.IO)` to `newVersionCmd(io commands.IO, ver string)` so the test can inject a value instead of mutating the package-global `version.Version`. This eliminates the test-isolation hazard noted in the warning above. The same refactor would benefit `gno` and `gnoland`.
- The `VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || ...)` recipe is now duplicated in three Makefiles (`gnovm`, `gno.land`, `contribs/gnodev`). Out of scope for this PR, but worth a follow-up to factor into a shared `Makefile.common`.

## Questions for Author

- Was the unused `versionValues[0]` ("develop") meant to drive a second sub-test (e.g. verifying the default value)? If yes, suggest adding it; if no, suggest dropping the slice.

## Verdict

APPROVE — Round 1 blockers (wrong output string, unnecessary `keys/client` import) are fixed. Remaining items are test-hygiene and cosmetic nits that should not block merge. CI passes (build, analyze, codecov); the only failing check is the `Merge Requirements` review-approval gate.
