# PR #5628: feat: add arkhe_pnt transpiled gno library

**URL:** https://github.com/gnolang/gno/pull/5628
**Author:** @google-labs-jules[bot] | **Base:** master | **Files:** 9 | **+1041 -0**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Adds a new `examples/gno.land/p/demo/arkhe_pnt` package — an AI-generated (Google Labs Jules bot) transpilation of the Python `arkhe_pnt` library. The package provides language detection, file-type classification, and a stub compiler/code-analysis frontend. The root `arkhe_pnt.gno` file is explicitly marked in comments as a failed transpilation stub.

## Test Results

- **Existing tests:** Not run — the package does not compile cleanly due to the empty `ComplianceValidator` struct and zero-body `ValidateCompliance()` function; `gno test` was not executed.
- **New tests:** Both `language_detector_test.gno` and `compiler_test.gno` exist but contain zero assertions — every test function body is empty.

## Critical (must fix)

- [ ] `language_catalog.gno:40` — **`.c` extension collision overwrites C entry with C++.** The `cpp` entry lists `Extensions: []string{".cpp", ".cxx", ".cc", ".C", ".h", ".hpp", ".hxx"}`. After `strings.ToLower(".C")` → `".c"`, the C++ entry's `.c` mapping silently replaces the earlier `c` entry's `.c` mapping in the `extIndex` map. Any `.c` file is then mis-detected as C++ instead of C. Fix: remove `.C` from the cpp entry (it is already covered by the `c` entry).

- [ ] `arkhe_pnt.gno` — **Root package is a failed-transpilation stub.** The file's own comment says: "Note: The original Python code structure was complex to directly transpile." `ComplianceValidator` has no fields and `ValidateCompliance()` has no body (returns zero values unconditionally). Merging a stub with a public-facing API surface into `examples/gno.land/p/demo/` sets a bad precedent and will confuse users. This file should either be fully implemented or removed from the PR scope.

- [ ] **Missing ADR.** `gno/AGENTS.md` states: "For non-trivial AI-generated or AI-assisted PRs, include an ADR." This PR is entirely AI-generated (author: `google-labs-jules[bot]`) and adds ~1 000 lines with no ADR. An ADR explaining the transpilation approach, known gaps, and intended use is required.

## Warnings (should fix)

- [ ] `compiler/compiler.gno:18–30` — **`TreeSitterFrontend` can never be initialized.** There is no constructor, and `initialized` is always `false`. `Parse()` unconditionally returns `ErrNotInitialized`. This means the entire compiler frontend is permanently dead code in this PR.

- [ ] `compiler/gnomod.toml` — **Missing `gno` version field.** The standard `gnomod.toml` format requires `gno = "0.9"` (or similar). Without it, `gno mod verify` will reject the package.

- [ ] `language_catalog.gno:120` — **`Procfile` classified as `"yaml"`.** A Heroku `Procfile` is a simple `key: command` DSL, not YAML. Mis-classification will mislead callers.

- [ ] `language_detector.gno:55` — **Nil-catalog panic.** `LanguageDetector.Detect()` calls `d.catalog.GetLanguageByExtension(ext)` without a nil check on `d.catalog`. If a `LanguageDetector` is constructed with `catalog: nil` (the zero value), this panics. Add a nil guard or make the zero value safe.

- [ ] `language_catalog.gno` — **Non-extension strings in `Extensions` arrays are unreachable.** Entries like `"Dockerfile"`, `"Makefile"`, `"Procfile"`, `"Gemfile"` appear in `Extensions` fields but `buildExtIndex` uses `filepath.Ext()` for lookup, which returns `""` for files with no dot. These entries are dead — they will never match via the ext lookup path. A separate `filenames` index is needed for basename-matched files.

## Nits

- [ ] All `.gno` files have header comments saying `// <name>.go` (Go extension) instead of `// <name>.gno`.
- [ ] `compiler/compiler.gno` has `mu int` as a stub field comment for `sync.Mutex` — this is misleading; just omit the field entirely since Gno has no `sync` package anyway.
- [ ] Portuguese-language comments appear throughout (e.g. `// Detecta o idioma`, `// Linguagem de programação`). The rest of the gnolang/gno repo uses English only.
- [ ] Excessive blank lines inside `Parse()` body (8+ consecutive blank lines between statements).
- [ ] `language_detector_test.gno` and `compiler_test.gno` import `"testing"` but call no `t.*` methods — the test files are effectively empty.

## Missing Tests

- [ ] `LanguageDetector.Detect()` with a `.c` file (would expose the collision bug).
- [ ] `LanguageDetector.Detect()` with an unknown extension (should return empty/false).
- [ ] `LanguageCatalog.GetLanguageByExtension()` for all major entries.
- [ ] `TreeSitterFrontend.Parse()` error path.
- [ ] `ComplianceValidator.ValidateCompliance()` when implemented.

## Suggestions

- Consider scoping this PR to only the parts that are actually complete: `language_catalog.gno` + `language_detector.gno` (after the collision fix) with real tests. The stub compiler and the failed-transpilation root file add noise without value.
- For `Dockerfile`/`Makefile`-style basename detection, add a separate `filenameIndex map[string]string` built in `NewLanguageCatalog` alongside `extIndex`.

## Questions for Author

1. What is the intended use case for this library in the gno.land ecosystem? Language detection is typically a developer-tooling concern, not a smart-contract concern — is this intended for gnoweb or gnodev rather than on-chain use?
2. Is the `arkhe_pnt` Python library under a license compatible with gno.land's Apache-2.0? The PR adds no LICENSE file or attribution.
3. Is the `TreeSitterFrontend` planned to be implemented in a follow-up, or was it included by accident?

## Verdict

REQUEST CHANGES — The package contains a self-admitted failed transpilation stub, a silent `.c`/`.C` collision bug that mis-classifies C files as C++, entirely empty test bodies, and a missing ADR for an AI-generated PR; these must be resolved before merge.
