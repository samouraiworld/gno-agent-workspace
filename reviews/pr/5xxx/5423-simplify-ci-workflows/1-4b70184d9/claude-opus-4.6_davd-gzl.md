# PR #5423: chore(ci): simplify workflows

**URL:** https://github.com/gnolang/gno/pull/5423
**Author:** moul | **Base:** master | **Files:** 44 | **+470 -782**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR consolidates 37 GitHub Actions workflow files down to 29 by eliminating unnecessary abstraction layers, duplication, and establishing consistent naming conventions with prefixes (`ci-`, `_ci-`, `ci-dir-`, `deploy-`, `meta-`, `meta-gh-`, `release-`).

Key changes:
1. **Template flattening (7 files -> 2):** The 4-level template chain (`template_main.yml` -> `template_lint.yml` -> `template_build.yml` -> `template_test.yml`) is replaced by a single `_ci-go.yml`. Three separate gno template files (`template_gnofmt/gnolint/gnotest`) are replaced by a single `_ci-gno.yml` with toggle inputs.
2. **Release consolidation (2 -> 1):** `releaser-master.yml` and `releaser-nightly.yml` (90% identical) merged into `release-goreleaser.yml` with conditional args based on trigger event.
3. **Verification consolidation (3 -> 1):** `mod-tidy.yml`, `genproto.yml`, and `docs-generate.yml` merged into `ci-codegen-verify.yml`.
4. **examples.yml rewrite:** The old workflow called `_ci-go.yml` (Go lint+build+test) but `examples/` has no `go.mod` for Go lint/build. Now correctly uses `_ci-gno.yml` for gno checks and inlines generate/mod-tidy checks directly.
5. **Simplifications:** Removed single-value matrices, removed `strategy: fail-fast: false` from non-matrix jobs, switched to `go-version-file: go.mod` where possible, cleaned up CodeQL boilerplate (93 -> 31 lines), fixed checkout ordering for `go-version-file`.
6. **github-bot config update:** Updated regex patterns from `releaser.*\.yml` to `release.*\.yml` and `staging\.yml` to `release-staging.yml`.

All callers are updated to reference new filenames. Internal references (self-path triggers, `workflow_run` triggers, `gh workflow run` commands) are updated correctly.

## Test Results
- **Existing tests:** PASS - All github-bot tests pass. CI checks all pass (except "Merge Requirements" which is a bot comment issue, not a CI failure).
- **Edge-case tests:** Skipped - This is a CI workflow refactor with no runtime Go/Gno logic changes beyond regex pattern updates.

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `_ci-gno.yml:1-2` — Duplicate comment line. Line 1 and line 2 are identical: `# Reusable workflow for Gno packages/realms: fmt + lint + test.`
- [ ] `_ci-go.yml:1-2` — Duplicate comment line. Line 1 and line 2 are identical but slightly different text: `# Reusable workflow for Go modules: lint + build check + test with coverage.` vs `# Reusable workflow for Go modules: lint + build + test.` One should be removed.
- [ ] `_ci-gno.yml:3` — Missing `name:` field. Unlike other workflows, this reusable workflow has no top-level `name:`. While not strictly required, having one would improve consistency and make workflow run references clearer.
- [ ] `_ci-go.yml:3` — Same issue: missing `name:` field. Both reusable workflow templates lack `name:` declarations, which means GitHub Actions will use the filename as the display name.
- [ ] `release-goreleaser.yml:37-38` — The dummy tag step uses `if: github.event_name == 'push'` but this means it won't create the tag for `workflow_dispatch` events. The old `releaser-master.yml` always created the tag. If goreleaser is invoked via `workflow_dispatch` without `snapshot: true`, it will attempt a real release without the dummy tag, which may fail.

## Nits
- [ ] `ci-dir-contribs.yml:20` — Empty line added after `setup:` job key (cosmetic inconsistency with other files).
- [ ] `ci-dir-gnoland.yml:37` — Comment on `node-version: lts/Jod` lost the inline version hint `# (22.x) https://github.com/nodejs/Release` that the old `gnoland.yml` had. Minor documentation loss.
- [ ] `ci-dir-examples.yml:40` — The `debug` input is passed as `${{ inputs.debug || false }}` but the `_ci-gno.yml` workflow declares `debug` as a boolean input. The `|| false` fallback is unnecessary for boolean inputs (GitHub defaults to `false` for unset booleans). Harmless but noisy.
- [ ] `meta-bot.yml:92` — Comment still references `bot-proxy.yml` ("See bot-proxy.yml for more info") instead of `meta-bot-proxy.yml`. Should be updated for consistency with the rename.

## Missing Tests
- [ ] No automated test validates that `workflow_run.workflows` name in `meta-bot-proxy.yml` (`meta / bot`) matches the actual `name:` in `meta-bot.yml`. A mismatch would silently break the bot proxy flow. Consider a CI check or at least a comment noting the coupling.
- [ ] The `release-goreleaser.yml` merge is untested for `schedule` and `workflow_dispatch` triggers (as noted in the PR's own test plan). These should be tested before merge if possible.

## Suggestions
- Consider adding a `name:` field to `_ci-gno.yml` and `_ci-go.yml` for consistent display in GitHub Actions UI (e.g., `name: _ci / gno` and `name: _ci / go`). This would align with the naming convention used for all other workflows.
- The `ci-codegen-verify.yml` path triggers include `docs/**` which means doc-only changes will trigger all three jobs (mod-tidy, genproto, docs). The old `docs-generate.yml` only triggered on `docs/**`. Consider adding path-based `if` conditions to individual jobs so that `mod-tidy` and `genproto` don't run unnecessarily on doc-only changes, saving CI minutes.
- In `release-goreleaser.yml:60`, using `${{ github.event_name }}` and `${{ inputs.snapshot }}` directly in a shell `if` statement is fine but consider using GitHub Actions `if:` conditions or environment variables to avoid potential expression injection risks (though the values here are safe).

## Questions for Author
- For `release-goreleaser.yml`, is there a reason the dummy tag is only created on `push` events and not on `workflow_dispatch`? If someone triggers a non-snapshot manual release, will goreleaser work without it?
- The old `gnoland.yml` had a longer comment explaining why `tests-ts-seq` is needed. The new `ci-dir-gnoland.yml:22` has a shorter FIXME. Is this intentional simplification or was the context accidentally lost?

## Verdict
APPROVE — Well-structured consolidation that significantly reduces CI maintenance burden (37 -> 29 files, 4-level nesting -> 1 level). The duplicate comment lines and the `workflow_dispatch` tag issue are minor and can be addressed in a follow-up. All CI checks pass and the changes are semantically correct.
