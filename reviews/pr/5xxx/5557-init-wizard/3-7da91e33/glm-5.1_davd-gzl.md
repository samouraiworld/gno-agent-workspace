# PR #5557: feat(gnovm): add interactive gno init wizard with template scaffolding

**URL:** https://github.com/gnolang/gno/pull/5557
**Author:** davd-gzl | **Base:** master | **Files:** 23 | **+1986 -81**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

Round-3 review at HEAD `7da91e33`. Since the last review (`4265d140`, round 2), seven commits reshaped the PR significantly:

1. **`53bb0988`** — Fixed orphan gnomod.toml on template conflict (extracted `renderModuleFiles` pre-check) and added `init` to README for `make generate`.
2. **`b5652367`** — Addressed round-2 nits: `gno mod init` kept as legacy alias (not deprecated), `validateGnoPath` tightened with `filepath.IsLocal`, `insertPathLetter` made idempotent, `scaffoldModule`/`scaffoldModuleWith` consolidated, namespace prompt wording partially updated, ADR refreshed.
3. **`f33b0557`** — Uppercased default `[P]ackage` in module-kind prompt.
4. **`feed85aa`** — **Major refactor**: replaced cargo-new-style subdirectory scaffolding with CWD-based model. Removed `prepareTargetDir`, `inPlace`, `rollbackCreatedDir`, `displayDir`, and all related helpers. `gno init <path>` now always writes to CWD; only `.gno` run scripts create directories (`run/`). Simplified early-exit checks and next-steps hints.
5. **`afd7cecf`** — Simplified helpers: `filepath.IsLocal` for `validateGnoPath`, `slices.Sorted(maps.Keys(...))` replacing `sortedKeys`, `strings.Cut` in `insertPathLetter`, `scaffoldModule` absorbs `writeModule`, `promptModuleKind` uses single slice, legacy cmd renamed from `newModInitDeprecatedCmd` → `newModInitLegacyCmd`, testdata renamed `deprecated` → `legacy`.
6. **`fc142340`** — Further control flow simplification: merged `scaffoldModule` + `scaffoldModuleWith` into one, extracted `writeFiles` helper (conflict-check + sorted write), added `kindLabel` and `runScriptDir` constant, `validateGnoPath` now returns script name for reuse.
7. **`7da91e33`** — Comment simplification.

The codebase is now significantly cleaner than in round 2: `mod.go` is more linear, the CWD-only model eliminates an entire class of directory-management bugs, and `writeFiles` provides a single conflict-check + write loop reused by both module and run-script paths.

## Test Results

- **Existing tests:** PASS — all `TestModInit*`, `TestPrompt*`, `TestKindFromPath`, `TestRenderTemplateDir`, `TestValidateGnoPath`, `TestModApp`, and `tm2/pkg/commands` tests green.
- **CI:** All checks passing (lint, build, generate, e2e). `main / test` was still pending at review time but expected green.
- **Edge-case tests:** Skipped (all round-1/2 critical bugs are fixed and have regression tests).

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `gnovm/adr/pr5557_mod_init_template.md:217` — **ADR Key Files table is stale.** References `scaffoldModuleWith`, `writeModule`, `newModInitDeprecatedCmd` which were removed/renamed in the refactoring. Actual code has `scaffoldModule` (single function), `writeFiles`, `newModInitLegacyCmd`. The table should be updated to match the current code so future contributors can navigate the codebase from the ADR.
- [ ] `gnovm/cmd/gno/mod.go:498` — **`promptModulePath` still says "Namespace or address".** `validateName` only accepts `[a-z0-9_]+`; no bech32 address can pass this validator. The round-2 review flagged this and the ADR claims it was fixed, but the code still includes "or address". Drop "or address" — it misleads users into thinking they can type an address.
- [ ] `gnovm/cmd/gno/mod.go:424-428` — **`writeRunScript` creates parent directory before template rendering.** If `renderTemplateDir` fails (e.g. corrupt template FS), `os.MkdirAll` has already created an empty `run/` directory that is never cleaned up. Reorder: render template first (no side effects), then create directory and write files.

## Nits

- [ ] `gnovm/adr/pr5557_mod_init_template.md:67` — ADR prompt text uses a colon (`:`) but the actual code at `mod.go:482` uses an em-dash (`—`). Minor doc drift.
- [ ] `gnovm/cmd/gno/mod.go:337-342` — `writeFiles` performs a conflict check even when called from `scaffoldModule`, which already pre-checks via `renderModuleFiles`. The redundant check is harmless (TOCTOU safety) but could be noted in a comment or split into `writeFilesUnchecked` for the pre-validated path.

## Missing Tests

- [ ] **`gno mod init` (legacy alias) without arguments.** The old `execModInit` accepted no args (creating gnomod.toml with empty module). The new bare path errors: `"module path is required with --bare"`. A test asserting this new behavior would lock it in. Currently `TestModInitLegacyAlias` only tests with a path argument.
- [ ] **`writeRunScript` orphan directory on render failure.** No test verifies that a render error doesn't leave a `run/` directory behind. A test with a deliberately broken template would catch this.
- [ ] **Interactive wizard with `--template` flag.** The `--template` flag is accepted but ignored in the full wizard path because `resolveOrPickTemplate` is called with `cfg.template` — but when `kind == kindRun`, `execInitRun` is called before `resolveOrPickTemplate`, so `cfg.template` is resolved via `resolveTemplate` directly. A test asserting `--template dao` works correctly in the wizard's run-script path would be valuable.

## Suggestions

- Update the ADR Key Files table to reflect the current code structure (remove `scaffoldModuleWith`, `writeModule`, `newModInitDeprecatedCmd`; add `writeFiles`, `kindLabel`, `printNextSteps`, `newModInitLegacyCmd`).
- Move the `MkdirAll` call in `writeRunScript` after the `renderTemplateDir` call to avoid orphan directories on render errors. The render is pure computation; only the write phase should have side effects.
- Consider extracting all init-related code (~500 lines) from `mod.go` into `init.go`. `mod.go` would revert to purely `gno mod` subcommands, and `init.go` would own the top-level `gno init` command and all its helpers. This was suggested in round 2 and is more compelling now that the init code is self-contained.

## Questions for Author

- Was the "Namespace or address" wording intentionally kept? The ADR says it was renamed to "Namespace" but the code says otherwise.
- Is the behavior change for `gno mod init` (no args) intentional? Old code created empty-module gnomod.toml; new code errors.

## Verdict

APPROVE — All round-1 and round-2 critical findings are verified fixed. The major CWD-based refactoring is a significant improvement that eliminates an entire class of directory-management bugs. The code is clean, well-documented, and well-tested. Remaining items (ADR staleness, "or address" wording, orphan directory on render failure) are non-blocking and can land in a follow-up.
