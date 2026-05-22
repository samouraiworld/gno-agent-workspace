# PR #4577: docs: add introduction to Blockchain Indexing

**URL:** https://github.com/gnolang/gno/pull/4577
**Author:** davd-gzl | **Base:** master | **Files:** 3 | **+38 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Adds a short conceptual introduction to blockchain indexing in the Gno docs (`docs/resources/indexing-gno.md`), wired into `docs/README.md` and `misc/docs/sidebar.json`. The doc explains why naive on-chain queries are expensive, introduces the indexing concept, and points readers at `gnolang/tx-indexer` for installation, examples, and a step-by-step tutorial.

History context: previous iterations of this PR included the full tutorial inline (database persistence, GraphQL queries, websocket subscriptions, embedmd-extracted code under `_assets/`). After @moul's `CHANGES_REQUESTED` review (2025-11-30) asking for the tutorial to live in the `tx-indexer` README, the inline tutorial was removed (commit a62afa63a, 2025-12-29). What remains here is a 36-line landing page. @MikaelVallenet approved on 2025-12-29 at commit 5d0370588. Only post-approval change is commit 88bcd0959, a trailing-newline fix. @nemanjantic's 2026-04-27 comment (`is this still relevant?`) reflects 4 months of inactivity; the answer is yes — the doc is ready, but it depends on `gnolang/tx-indexer#219`.

## Test Results
- **Existing tests:** Docs-only change. CI: `embed` job **FAILS** at the lint step (https://github.com/gnolang/gno/actions/runs/22403366112). All other checks pass.
- **Edge-case tests:** N/A (docs).

## Critical (must fix)

- [ ] `docs/resources/indexing-gno.md:36` — Broken link `https://github.com/gnolang/tx-indexer/blob/main/docs/how-to-create-an-indexer.md` causes `make lint` failure in the `embed` CI job. The file does not exist on `gnolang/tx-indexer:main` because the companion PR `gnolang/tx-indexer#219` is still **OPEN** (verified via `gh pr view 219 -R gnolang/tx-indexer`, state=`OPEN`, mergedAt=`null`). This PR cannot be merged until either (a) `tx-indexer#219` merges first, (b) the link is repointed to a branch/commit where the file exists, or (c) the link is removed/replaced with a TODO and added back in a follow-up. This is the answer to nemanjantic's "still relevant?" — yes, but blocked on `tx-indexer#219`.

## Warnings (should fix)

- [ ] `docs/README.md:48` — Index entry `Learn how to index TM2 blockchain (As Gno Land) through tx-indexer.` has three issues: `As` should be lowercase `as`; `Gno Land` should be `Gno.land` (canonical spelling used everywhere else in the file); `index TM2 blockchain` reads as a missing article — `index a TM2 blockchain` or `index TM2 chains` reads better.
- [ ] `docs/resources/indexing-gno.md:14` — Parallelism break: `enabling instant queries and unlock complex real-time use cases` — should be `unlocking` to match `enabling`. Also `'addpkg' transaction` → `'addpkg' transactions` (plural matches "all").
- [ ] `docs/resources/indexing-gno.md:34` — `### Use case: ...` is an H3 nested under the H2 `## Installation`, but the use-case section is unrelated to installation. Promote to H2 (`## Use case: ...`) or move outside the Installation section.

## Nits

- [ ] `docs/resources/indexing-gno.md:5` — Trailing whitespace after `(and can be very costly).` (visible after "costly).").
- [ ] `docs/resources/indexing-gno.md:7-8` — Inconsistent bullet punctuation: bullet 1 has no terminal period, bullet 2 ends with `.`. Pick one.
- [ ] `docs/resources/indexing-gno.md:12` — `### The Indexing Solution` is H3 but is the first subsection under the H1 title, with no H2 above it. Promote to H2 for proper outline.
- [ ] `docs/resources/indexing-gno.md:14-16` — Heavy bold (`**Indexers**`, `**processing**`, `**extracting**`, `**maintaining**`, `**structured datasets**`) reads as marketing copy. Trim to the 1–2 terms readers genuinely need to spot.
- [ ] `docs/resources/indexing-gno.md:34` — Trailing whitespace after `dashboard`.
- [ ] `docs/resources/indexing-gno.md:20` — H2 title `## [tx-indexer](...): The official [TM2](...) Indexer` puts two links inside a heading. Functional, but heading anchors with embedded links are awkward; consider plain-text heading + a leading sentence with the links.

## Missing Tests
- [ ] N/A — docs-only PR.

## Suggestions

- Recommended unblock path: get `gnolang/tx-indexer#219` merged (it has only one COMMENTED review from ajnavarro, no APPROVED). Then rebase this PR — the `embed` lint will pass without further changes here. Alternative: temporarily point the line-36 link at the open PR's branch (`https://github.com/gnolang/tx-indexer/blob/david/docs/tutorial/docs/how-to-create-an-indexer.md`) so CI passes now, then swap to `main` in a follow-up after `#219` merges. The first option is cleaner.
- Address @nemanjantic explicitly with the blocker so the PR doesn't get auto-stale-closed: a one-line reply pointing at `tx-indexer#219` is enough.
- Consider folding the warnings/nits into a single cleanup commit before the unblock so the rebase after `tx-indexer#219` is the last touch.

## Questions for Author

- Why was the trailing-newline-only commit 88bcd0959 added on 2026-02-25 *after* MikaelVallenet's 2025-12-29 approval? If it was just to retrigger CI, fine — but a re-approval may be courteous since approvals technically apply to the previous commit.
- Is there a reason the H2 `## [tx-indexer]...` section duplicates information from the linked tx-indexer README (dual-protocol API, transport, storage)? If the README is canonical, a single sentence + link avoids drift.

## Verdict

**REQUEST CHANGES** — blocked by broken link in `indexing-gno.md:36` causing `embed` CI failure; depends on `gnolang/tx-indexer#219` merging first (or a temporary link redirect). Content is otherwise approval-ready modulo minor copy edits.
