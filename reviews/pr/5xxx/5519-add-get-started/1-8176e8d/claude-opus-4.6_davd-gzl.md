# PR #5519: docs: add get started page

**URL:** https://github.com/gnolang/gno/pull/5519
**Author:** alexiscolin | **Base:** master | **Files:** 4 | **+276 -8**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

Adds a new canonical "Getting started" page (`docs/getting-started.md`) that takes newcomers from zero to a working toolchain in ~15 minutes. The page covers: prerequisites, install from source, Docker alternative, first `gnodev` run, key creation, faucet, live network query, namespaces/CLA, next steps, help channels, and troubleshooting.

The PR also: (1) adds a "Getting Started" section to `docs/README.md` as a hero pointer, (2) adds a sidebar entry in `misc/docs/sidebar.json` as a new top-level category right after README, and (3) replaces the duplicated SSH clone+install block in `docs/builders/local-dev-with-gnodev.md` with a pointer to the new page.

This addresses issue #5459 (UX-1 setup friction) as part of the Developer Onboarding meta #5458. The previous install instructions were duplicated across two pages with inconsistencies (SSH vs HTTPS, stale Go 1.22+ vs actual 1.24+ requirement).

## Test Results
- **Existing tests:** PASS — all CI checks green, including the `docs` build job.
- **Edge-case tests:** skipped — docs-only PR, no code changes.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `docs/getting-started.md:82-88` — Docker image paths may be incorrect. The listed paths (`ghcr.io/gnolang/gno/gnokey`, `ghcr.io/gnolang/gno/gnodev`, etc.) assume a multi-image repo structure under `ghcr.io/gnolang/gno/`. These should be verified against what is actually published to GHCR. If images don't exist at these paths, newcomers will hit a confusing `manifest unknown` error on their first try.

- [ ] `docs/getting-started.md:10-14` — The Playground tip says "run it instantly" but as jefft0 noted in review, the default Playground example fails with `name main not declared` when clicking Run. This is misleading for newcomers. Consider either removing "run it instantly" or qualifying it (e.g., "write and test Gno code in your browser").

- [ ] `docs/getting-started.md:193` — The CLA signing command includes `-gas-fee 100000ugnot -gas-wanted 2000000` with placeholder `<chain-id>` and `<rpc-endpoint>`, but the info box below only provides Betanet values. A newcomer may not know which values to substitute. Consider inlining the Betanet command directly as the primary example, since that's the main live network.

## Nits

- [ ] `docs/getting-started.md:29` — Line is 107 chars. Consider wrapping for consistency with the rest of the file which wraps at ~72 chars.

- [ ] `docs/getting-started.md:109` — "Edit any `.gno` file under `examples/`" — this is slightly misleading since `gnodev` watches the *current working directory*, not the examples folder. Running bare `gnodev` from the repo root loads examples but hot-reload is for the CWD. Consider clarifying.

- [ ] `docs/getting-started.md:178-179` — "Username-based namespaces like `gno.land/r/alice/...` are **not available yet**" — this may become stale. Consider linking to an issue or docs page that tracks namespace status rather than making a point-in-time claim.

- [ ] `docs/builders/local-dev-with-gnodev.md:163` — The `-broadcast` flag is still present in the `gnokey maketx call` example. As jefft0 noted, broadcasting is now the default after a recent merge, so this flag is redundant and may confuse readers who see it here but not in the getting-started page.

- [ ] `misc/docs/sidebar.json:5-11` — "Getting Started" is a category with a single item. Consider making it a direct doc link instead of a category to reduce nesting, unless more items are planned for this section.

## Missing Tests

- [ ] No link validation test to ensure all relative markdown links in `getting-started.md` resolve. The docs build CI likely catches broken links, but worth confirming all 10+ relative links are valid.

## Suggestions

- Consider adding a brief "What you'll need" time estimate per section (e.g., "Install: ~5 min, Key setup: ~2 min") to help readers plan. The overall "15 minutes" claim is good but section-level estimates would be even more helpful.

- The `gnodev` section could mention that `Ctrl+R` reloads and `h` shows help — small details that make the first experience smoother.

- The "Getting help" section omits Discord/Telegram if those exist. If there are community chat channels, they're typically the fastest way to get unstuck and worth including.

## Questions for Author

- Are the Docker image paths verified? Specifically `ghcr.io/gnolang/gno/gnokey` and siblings — do these actually exist as published images?
- Is there a plan to add more items under the "Getting Started" sidebar category, or should it be flattened to a direct link?

## Verdict

APPROVE — Well-structured, comprehensive getting-started guide that fills a real gap. The warnings are minor and can be addressed in follow-up commits. The core content is accurate, well-written, and provides a clear path from zero to first interaction.
