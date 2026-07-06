# PR [#5873](https://github.com/gnolang/gno/pull/5873): docs: rewrite gnokey into guide and reference, rename gnodev doc

URL: https://github.com/gnolang/gno/pull/5873
Author: davd-gzl | Base: master | Files: 19 | +1184 -1530
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 66552ff7a (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5873 66552ff7a`

**TL;DR:** Splits the single `interact-with-gnokey.md` page into a task-oriented user guide (`users/using-gnokey.md`) and a full command reference (`resources/gnokey-reference.md`), renames `resources/gnodev.md` to `gnodev-reference.md`, and repoints inbound links across the docs. Docs-only, no gno-code change.

**Verdict: REQUEST CHANGES** — the sidebar was not regenerated so the `docs` CI check is red and the nav links two deleted pages; one inbound link and one table cell are also broken. All three are mechanical fixes.

## Summary
The rewrite reads well and the reference is factually accurate: every CLI flag and query endpoint I spot-checked against `tm2/pkg/crypto/keys/client` and `gno.land/pkg/sdk/vm/handler.go` matches the source. Three concrete defects block a clean merge. The docs CI job fails because `misc/docs/sidebar.json` was never regenerated after the two renames, so the left-nav still lists `users/interact-with-gnokey` and `resources/gnodev`, both now deleted. One inbound link in the renamed gnodev page still points at the deleted gnokey page. One table cell in getting-started uses a stray double-backtick that renders wrong.

## Glossary
None.

## Fix
Run `make generate` in `docs/` and commit the regenerated `misc/docs/sidebar.json`; that clears the red `docs` check and repoints the two nav entries. Repoint `docs/resources/gnodev-reference.md:118-119` from `../users/interact-with-gnokey.md#making-an-airgapped-transaction` to `./gnokey-reference.md#airgapped-signing`. Restore a clean anti-autolink split in the testnet URL cell at `docs/builders/getting-started.md:275`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[nav links two deleted pages; docs CI red]** `misc/docs/sidebar.json:11,44` — the sidebar still lists `users/interact-with-gnokey` and `resources/gnodev`, both deleted by this PR, and the `docs` CI job fails on the resulting `make generate` diff.
  <details><summary>details</summary>

  The PR renamed `interact-with-gnokey.md` to `using-gnokey.md` plus `gnokey-reference.md`, and `gnodev.md` to `gnodev-reference.md`, but did not run `make generate`. The committed sidebar still points at the old slugs, so the left-nav has two dead entries and the `docs` job fails with "Please run 'make generate' in docs/." Verified: running the target regenerates `misc/docs/sidebar.json` with `users/interact-with-gnokey` → `users/using-gnokey` and `resources/gnodev` → `resources/gnodev-reference`, exactly the two dead links. Fix: run `make generate` in `docs/` and commit the regenerated `misc/docs/sidebar.json`.
  </details>

- **[broken inbound link to a deleted page]** [`docs/resources/gnodev-reference.md:118-119`](https://github.com/gnolang/gno/blob/66552ff7a/docs/resources/gnodev-reference.md?plain=1#L118) · [↗](../../../../../.worktrees/gno-review-5873/docs/resources/gnodev-reference.md#L118) — links to the deleted `interact-with-gnokey.md#making-an-airgapped-transaction`.
  <details><summary>details</summary>

  The `-txs-file` paragraph links "making an airgapped transaction" to `../users/interact-with-gnokey.md#making-an-airgapped-transaction`. That file was deleted in this PR and the anchor exists nowhere in the new docs. The airgapped flow moved to `gnokey-reference.md` under `## Airgapped signing` (line 371), anchor `airgapped-signing`. Since the linking file sits in `resources/`, the correct target is `./gnokey-reference.md#airgapped-signing`. A tree-wide grep confirms this is the only surviving reference to the deleted page. Fix: repoint the link to `./gnokey-reference.md#airgapped-signing`.
  </details>

- **[table cell renders a stray double-backtick]** [`docs/builders/getting-started.md:275`](https://github.com/gnolang/gno/blob/66552ff7a/docs/builders/getting-started.md?plain=1#L275) · [↗](../../../../../.worktrees/gno-review-5873/docs/builders/getting-started.md#L275) — the testnet URL cell `` `https://``rpc.<testN>...` `` mishandles the anti-autolink split.
  <details><summary>details</summary>

  master used a zero-width space between `https://` and `rpc` inside one code span to stop GitHub autolinking the URL. This PR replaced it with a double backtick: `` `https://``rpc.<testN>.testnets.gno.land:443` ``. Under CommonMark/GFM the first single-backtick run closes at the next single backtick, so `` `https://` `` is one code span and the `` `` `` starts a second run that never finds a matching double-backtick closer; the reference python-markdown renders the two backticks as literal characters inside the code span. Either way the cell no longer renders as one clean inline URL. Fix: restore the zero-width-space split, or write the two segments as two separate single-backtick code spans (`` `https://` `` `` `rpc.<testN>.testnets.gno.land:443` ``).
  </details>

## Nits
- [`docs/users/using-gnokey.md:141`](https://github.com/gnolang/gno/blob/66552ff7a/docs/users/using-gnokey.md?plain=1#L141) · [↗](../../../../../.worktrees/gno-review-5873/docs/users/using-gnokey.md#L141) — "Calling `Deposit` on the `wugnot` realm to wrap `1000ugnot`." is a verbless sentence fragment; fold it into the preceding sentence or make it a full sentence ("This calls `Deposit` ... to wrap `1000ugnot`.").
- [`docs/builders/getting-started.md:369`](https://github.com/gnolang/gno/blob/66552ff7a/docs/builders/getting-started.md?plain=1#L369) · [↗](../../../../../.worktrees/gno-review-5873/docs/builders/getting-started.md#L369) — link text still reads "`addpkg` in Interact with gnokey" while the target moved to `gnokey-reference.md`; the page is now titled "gnokey command reference", so the stale "Interact with gnokey" label reads oddly. Same stale-label pattern in the query-state-api, rpc-clients, gas-fees, glossary, gno-packages, storage-deposit, realms, and example-boards repoints, which keep "Interacting with gnokey" text over reference-page targets. Cosmetic; the links resolve.

## Missing Tests
None. Docs-only.

## Suggestions
None.

## Open questions
- `docs/builders/tutorial-minisocial.md:25` and `:119` link to `getting-started.md#4-before-you-deploy` and `#run-a-local-chain`, but the headings are "### 3. Before you deploy" (anchor `3-before-you-deploy`) and "### 5. Run a local chain" (anchor `5-run-a-local-chain`), so both anchors are dead. Both are identical on master and untouched by this PR, so out of scope; not posted. Worth a one-line follow-up sweep since this PR is already a link-hygiene pass.
