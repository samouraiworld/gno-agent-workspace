# PR #5441: fix(ci): bump Go version to 1.25.0

**URL:** https://github.com/gnolang/gno/pull/5441
**Author:** notJoon | **Base:** master | **Files:** 47 | **+325 -328**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR bumps the Go version from 1.24 to 1.25.0 across the entire monorepo and updates the linting toolchain accordingly. The changes span 47 files across several categories:

**Go version bumps (21 go.mod files + CI + Docker):**
- Root `go.mod` and 20 `contribs/*/go.mod` / `misc/*/go.mod` files updated from `go 1.24` to `go 1.25.0`. The `toolchain` directives are removed (Go 1.25 uses the go line as the minimum version).
- `.github/workflows/*.yml` — Go version updated from `1.24.x` to `1.25.x` in CI matrices and setup steps.
- `Dockerfile` — Base image bumped to `golang:1.25.0-alpine`.

**Linter config updates:**
- `misc/devdeps/go.mod` — golangci-lint bumped from v2.3 to v2.11.
- `.github/golangci.yml` — Disables `prealloc` linter (too noisy with new lint version). Adds gosec excludes: G122, G602, G703, G704, G705. Adds staticcheck exclusion `-QF1012`. Adds `staticcheck` to test exclusion list.
- `gnovm/pkg/doc/json_doc.go:113` — Added `//nolint:staticcheck` for deprecated `ast.MergePackageFiles`.
- `tm2/pkg/iavl/internal/bytes/bytes.go:60,64` — Removed stale `//nolint:staticcheck` directives (the `WriteString`/`Sprintf` pattern is no longer flagged).

**Generated code:**
- `gnovm/pkg/gnolang/string_methods.go` — Regenerated: Go 1.25 stringer produces a different bounds-check pattern (uses subtraction-based bounds instead of comparison-based).

**Filetest error message updates (~10 files):**
- `gnovm/tests/files/*.gno` — Updated expected error messages for Go 1.25 type checker wording changes (e.g., "cannot use X as Y value" → "cannot use X as Y value in argument to F", and similar adjustments to addressability error messages).

## Test Results
- **Existing tests:** PASS — `go test ./gnovm/pkg/gnolang/ -run Files -test.short` passed in the worktree.
- **CI status:** All checks pass except "Merge Requirements" (needs approval).
- **Edge-case tests:** Skipped — version bump; existing test suite provides coverage.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `misc/gnomd/go.mod:3` — The `gnomd` module jumps from `go 1.22` directly to `go 1.25.0`, skipping 1.23 and 1.24. While this works, it's a larger minimum version jump than other modules in the repo (most went from 1.24 to 1.25.0). If `gnomd` was intentionally pinned to 1.22 for compatibility reasons, this jump may break users on older Go versions. Verify this is intentional.

- [ ] `.github/golangci.yml` — **Five new gosec rules excluded (G122, G602, G703, G704, G705) without justification.** These rules detect: G122 (unquoted file paths in shell commands), G602 (slice bounds out of range), G703 (errors not checked with type assertion), G704 (errors not checked), G705 (range over map without ordering). Some of these (particularly G602, G704) catch real bugs. The PR should document why each exclusion is necessary — ideally in a comment in the config file or the PR description. If they're too noisy on the current codebase, a better approach might be to fix the flagged instances rather than globally suppressing them.

## Nits

- [ ] `.github/golangci.yml` — The `prealloc` linter is disabled with no comment explaining why. A brief `# disabled: too noisy with golangci-lint v2.11` comment would help future maintainers understand the reasoning.

- [ ] `gnovm/pkg/gnolang/string_methods.go` — This is generated code, but the PR doesn't mention which command was used to regenerate it. Adding `//go:generate` instructions or noting the command in the PR description helps reviewers verify the output.

## Missing Tests

- [ ] No test verifies that the Go 1.25 stringer output in `string_methods.go` is correct. While it's generated code, a round-trip test (e.g., verifying `SomeType.String()` returns expected values) would catch stale generated files — `gnovm/pkg/gnolang/string_methods.go`.

## Suggestions

- Consider fixing the gosec findings (G602, G703, G704, G705) rather than globally suppressing them, or at minimum adding `//nolint:gosec` directives to specific call sites with explanations. Global suppression hides future violations.

- The `toolchain` directive removal from go.mod files should be mentioned in the PR description, as it's a behavioral change — Go 1.25 uses the `go` line as the minimum version directly, so removing `toolchain` is correct but not obvious to reviewers unfamiliar with the change.

## Questions for Author

- Was the `gnomd` go.mod jump from 1.22 to 1.25.0 intentional, or was it an oversight? All other modules were on 1.24 previously.

- For the gosec exclusions (G122, G602, G703, G704, G705): how many violations does each rule flag? If the count is small, would it be preferable to fix them rather than exclude the rules?

- The `-QF1012` staticcheck exclusion suppresses the "Use `fmt.Fprintf(x, ...)` instead of `x.Write([]byte(fmt.Sprintf(...))))`" quickfix suggestion. Is this because the current pattern is intentional (e.g., for performance or interface compliance), or just to reduce noise?

## Verdict

**APPROVE** — Standard version bump with appropriate adjustments. The gosec exclusions deserve a closer look (warning above) but don't block merging — they can be addressed in a follow-up PR. CI is green.
