# PR #5563: feat(gnodev): add gnodev version command

**URL:** https://github.com/gnolang/gno/pull/5563
**Author:** AmozPay | **Base:** master | **Files:** 8 | **+138 -3**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Round 4 review at HEAD `d82b70505` (`style(gnodev): remove trailing remove unused args`). Single style commit since round 3 (`0a6ed8b22`). Diff vs round 3 is 2 lines:

1. **`contribs/gnodev/main.go:68`** — Adds a blank line after `cmd.AddSubCommands(newVersionCmd(stdio))` so the `// XXX:` block keeps its pre-PR separation. Resolves the round 2/3 nit.
2. **`contribs/gnodev/version.go:19`** — Renames closure parameter `args []string` → `_ []string`. Resolves the round 2/3 nit; aligns with `main.go:60` which already uses `_ []string`.

No code paths or behavior changed. The functional surface (version subcommand, txtar suite, `RegisterExecCommand` / `RunTestscript` helpers in `gnovm/pkg/integration/testscript.go`, the `tm2/pkg/version.Version=$(VERSION)` ldflag wiring in the gnodev `Makefile`, and the `README.md` blurb) is identical to round 3.

Re-verified locally: `go test -run 'TestVersionCmd|TestScripts' ./contribs/gnodev/...` passes (`TestScripts/version` 0.03s under a 3.5s build, `TestVersionCmd` 0.00s).

## Test Results

- **Existing tests:** PASS
  - `TestVersionCmd` — `PASS` (0.00s)
  - `TestScripts/version` — `PASS` (0.03s)
- **CI:** all green except the `Merge Requirements` review-approval gate (manual). Codecov patch coverage 41% — unchanged since round 3, dominated by `RegisterExecCommand` having no negative-path test.
- **Edge-case tests:** skipped — no functional change since round 3.

## Critical (must fix)

- None.

## Warnings (should fix)

- None.

## Nits

- [ ] `contribs/gnodev/Makefile:1` — Stray leading blank line still present (sibling Makefiles `gnovm/Makefile`, `gno.land/Makefile` start with `VERSION ?=` on line 1). Carried forward from rounds 2 and 3.
- [ ] `contribs/gnodev/testdata_test.go:26` — `buildCmd.Dir = "."` is redundant; `go test` already runs in the package directory. Carried forward from round 3.
- [ ] `gnovm/pkg/integration/testscript.go:46` — `panic` on duplicate command name is inconsistent with sibling `SetupTestscriptsCoverage` which returns an error. Carried forward from round 3.
- [ ] `gnovm/pkg/integration/testscript.go:46` — Typo: `"unable register %q"` → `"unable to register %q"`. Carried forward from round 3.
- [ ] `gnovm/pkg/integration/testscript.go:64-67` — `RunTestscript` is a one-line passthrough used only by `contribs/gnodev/testdata_test.go`. `gnovm/cmd/gno/testdata_test.go` still imports `go-internal/testscript` directly, so the "shared integration pattern" rationale does not hold repo-wide. Carried forward from round 3.

## Missing Tests

- [ ] No negative-case coverage of `RegisterExecCommand` (duplicate-registration panic, `! gnodev nonsense-subcommand` to exercise the `successExpected != commandSucceeded` branch). Carried forward from round 3.
- [ ] No `gnodev version --help` or `gnodev local version` (should fail) txtar lines locking in the routing decision in `main.go:65-67`. Carried forward from round 3.

## Suggestions

- Consider renaming `RegisterExecCommand` → `RegisterBinary` or `RegisterExternalCommand`. "Exec" reads as the `os/exec` package; the helper actually wires `ts.Exec` into the testscript command map. Carried forward from round 3.
- Out of scope, but the `VERSION ?= $(shell git describe ...)` recipe is now duplicated across `gnovm/Makefile`, `gno.land/Makefile`, and `contribs/gnodev/Makefile` — a shared `Makefile.common` include would be a worthwhile follow-up.

## Questions for Author

- None new.

## Verdict

APPROVE — Round 3 nits on `main.go` blank line and `version.go` unused parameter are both addressed. No functional change since the previously-approved round 3; remaining items are stylistic carries unchanged across the last two rounds. CI green except the manual approval gate.
