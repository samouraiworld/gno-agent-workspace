# PR #5572: feat(gnoweb): add package overview source doc

**URL:** https://github.com/gnolang/gno/pull/5572
**Author:** alexiscolin | **Base:** master | **Files:** 18 | **+2633 -114**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR replaces the "first file" fallback at `/r/<pkg>$source` with a pkg.go.dev-style package overview page. The new `OverviewView` displays package doc, README, exported constants/variables/functions/types/bugs (with methods), imports, files, and subpackages, together with a metadata sidebar (namespace, path, type, Gno version, license, code stats, quality indicators, table of contents).

Key architectural decisions:
- **`gnovm/pkg/doc`** adds `File` and `Line` fields (both `omitempty`) to `JSONValueDecl`, `JSONFunc`, and `JSONType`, populated via a new `(pkg *pkgData).extractPosition(ast.Node)` helper. This enables deep-links from overview symbols to their exact declaration site in the source view.
- **`gnoweb/components`** adds `OverviewView` data types, a `DocRenderer` interface (injected per-request, avoiding the rejected global `SetRenderer` pattern from PR #4542), and pure metadata derivation helpers in `overview_meta.go`.
- **Handler** (`handler_http.go`) adds `GetOverviewView` which fans out `ListFiles`, `Doc`, README rendering, and `ListPaths` via `errgroup`, then bounded per-file source fetching for import parsing (4 concurrent RPCs, 10 .gno files max).
- **Routing change**: `/r/<pkg>$source` without `file=` now routes to Overview instead of the first file's source view. `/r/<pkg>$source&file=X` is unchanged.
- ADR included at `gno.land/adr/prxxxx_gnoweb-package-overview.md`.
- Comprehensive test coverage: unit tests for all pure helpers, handler tests for success/degraded/404/routing, integration route tests, and updated `gnovm/pkg/doc` fixture.

CI: all checks pass (build, lint, test, e2e). Merge-requirements check fails due to missing codeowner approval (expected — needs gfanton or alexiscolin review per CODEOWNERS).

## Test Results

- **Existing tests:** PASS — CI green (build, lint, test, e2e, gnoweb_front_lint, gnoweb_generate).
- **Edge-case tests:** skipped (coverage from unit tests is extensive at 94.2% patch coverage per Codecov).

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `handler_http.go:843-845` — `renderReadme` is called inside the errgroup but silently swallows errors. The second return value (raw bytes) is discarded with `_`. If `renderReadme` fails, the overview page renders without a README section — acceptable degraded behavior, but the errgroup goroutine returns `nil` even on rendering failure, making it impossible for the outer `g.Wait()` to surface README rendering errors. Consider logging the error at a minimum, or propagating it if README absence should be a visible degradation signal.

- [ ] `handler_http.go:866` — `fetchSourcesForImports` uses `ctx` (the original request context) instead of `gctx` (the errgroup context). After `g.Wait()` returns, `gctx` is done but `ctx` is still alive. This is likely intentional since `fetchSourcesForImports` runs *after* the errgroup, but it means that if the request is cancelled between `g.Wait()` and `fetchSourcesForImports` completing, the goroutines inside `fetchSourcesForImports` will keep running until they notice the cancellation. Using `ctx` here is correct but worth a brief comment for clarity.

- [ ] `overview_meta.go:29` — `BSD-3-Clause` regex uses `[\s\S]*` which could be slow on large license content that doesn't match the "3.\s*neither" pattern. While the 4KB cap in `deriveLicense` bounds the input, the regex still has quadratic worst-case behavior on the capped 4KB slice. Since Go uses RE2 (linear time guarantee), this is safe from ReDoS, but the `[\s\S]*` greedy match will backtrack-linearly across 4KB. Low risk given the cap, but worth noting.

- [ ] `overview_meta.go:29-30` — `BSD-2-Clause` regex will match before `BSD-3-Clause` in the signature list. The list is ordered most-specific-first, so BSD-3-Clause is checked before BSD-2-Clause. This is correct. However, a BSD-3-Clause license text that happens to not contain "3.\s*neither" will be misidentified as BSD-2-Clause. This is a known limitation of regex-based license detection and is acceptable for a UI hint, but the ADR or a comment should note it.

- [ ] `handler_http.go:317` — Stale comment `// Handle Source page` for what is actually the Directory/Package view branch. Minor but misleading.

## Nits

- [ ] `overview_meta.go:291` — `deriveInfo` has an unused second parameter `_ []string` (files). This was likely kept for forward compatibility but adds noise.

- [ ] `overview_meta.go:82-84` — A single `token.FileSet` is shared across all `parser.ParseFile` calls in `parseImports`. While this works for import extraction (only path strings are needed), it means the fset accumulates position data for every file parsed. For packages with many source files, this is wasted memory since the positions are never used. Consider using `token.NewFileSet()` per file, or discarding the fset after use.

- [ ] `overview.html:156` — `{{ range . }}<li>{{ . }}</li>{{ end }}` for Bugs renders user-controlled `BUG(...)` text without HTML escaping. However, since this goes through Go's `html/template`, the `{{ . }}` action auto-escapes, so this is safe. No action needed, just noting for reviewers.

- [ ] `json_doc.go:259` — `extractPosition` is called on `typeSpec` rather than `typ.Decl`. This means the `File`/`Line` for a type refers to the `type` keyword position rather than the `GenDecl` position. This is arguably better for deep-linking (takes you to the type spec, not the `type (` of a group), but differs from how values and functions are extracted (on their `Decl` node). Inconsistent but not a bug.

## Missing Tests

- [ ] `renderSignature` and `renderDocString` in `overview_meta.go:419-443` — the failure-path fallback (renderer returns error → falls back to HTML-escaped string) is not directly tested. The `noopRenderer` never fails. A test with a failing `DocRenderer` would verify the fallback behavior.
- [ ] `parseGnoVersion` in `overview_meta.go:316` — not tested for edge cases (e.g., `gno = ""`, `gno = "0.1" ` with trailing whitespace, malformed gnomod.toml).
- [ ] `filterImportSources` in `handler_http.go:919` — not directly unit-tested; it's only exercised indirectly through `fetchSourcesForImports` in handler tests.
- [ ] `BuildOverview` in `overview_meta.go:503` — the end-to-end integration of all helpers is tested in `view_overview_test.go` but only with a minimal input. No test exercises the full combination of all sections populated simultaneously with real rendered content (the handler test `TestHTTPHandler_GetOverviewView_SuccessRendersAllSections` comes closest but uses the raw renderer).

## Suggestions

- Add a brief comment at `handler_http.go:866` explaining why `ctx` (not `gctx`) is used for `fetchSourcesForImports`.
- Consider extracting the `BSD-3-Clause` / `BSD-2-Clause` ordering rationale into a comment near `licenseSignatures` to help future contributors avoid accidentally reordering them.
- The `noopRenderer` type in `overview_meta_test.go` duplicates the `rawRenderer` in `handler_http_test.go`. Consider sharing a test helper, though this is minor.

## Questions for Author

- The `renderReadme` function uses `RenderRealm` for README.md rendering (which applies realm-specific markdown processing). The overview's `renderDocString` uses `RenderDocumentation`. Is there a reason README.md needs realm-level rendering rather than documentation-level rendering? Could this cause inconsistency in how markdown is rendered between the README section and doc-comment sections?
- For packages with >10 .gno files, only the first 10 are fetched for import parsing. This means `parseImports` may miss imports from later files. Is the 10-file cap sufficient for the vast majority of on-chain packages, or should it be configurable?

## Verdict

APPROVE — Well-structured PR with clean separation of concerns (pure helpers, injected renderer, bounded concurrent fetching), strong test coverage, and an ADR documenting all decisions. The findings are all warnings/nits; no correctness or security issues. The routing change (overview replacing first-file at `$source`) is the main behavioral shift and is well-documented in the ADR with rollback considerations.
