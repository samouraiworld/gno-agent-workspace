# PR [#5873](https://github.com/gnolang/gno/pull/5873): docs: rewrite gnokey into guide and reference, rename gnodev doc

URL: https://github.com/gnolang/gno/pull/5873
Author: davd-gzl | Base: master | Files: 19 | +1187 -1533
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: eb829bec3 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5873 eb829bec3`

**TL;DR:** Splits the single `interact-with-gnokey.md` page into a task-oriented user guide (`users/using-gnokey.md`) and a full command reference (`resources/gnokey-reference.md`), renames `resources/gnodev.md` to `gnodev-reference.md`, and repoints inbound links across the docs. Docs-only, no gno-code change.

**Verdict: REQUEST CHANGES** — one rendering regression remains: the testnet-URL table cell in getting-started.md still ships a literal double-backtick that renders as visible backticks, a regression from master. The two prior blocking defects (unregenerated sidebar, broken airgapped inbound link) are fixed in the head commit; the `docs` CI check is now green.

## Summary
Since the previous reviewed head, one commit (`eb829bec3`) regenerated `misc/docs/sidebar.json` and repointed the gnodev-reference airgapped link. Both fixes hold: the `docs` CI check now passes, and `gnokey-reference.md#airgapped-signing` resolves to the `## Airgapped signing` heading. One defect is untouched: the testnet-URL cell at `docs/builders/getting-started.md:275` still uses a literal double-backtick where master used a zero-width space, so the cell renders as one code span with two visible backticks instead of a clean inline URL. The reference itself remains accurate; every flag and query anchor I re-checked resolves.

## Glossary
None.

## Fix
Restore a clean anti-autolink split in the testnet URL cell at `docs/builders/getting-started.md:275`: the master zero-width-space form, or two separate single-backtick code spans. Optionally retarget the four stale "Interact(ing) with gnokey" labels that now sit over reference-page anchors, and give the `Deposit` fragment in using-gnokey.md a main verb.

Verified on eb829bec3: the delta since the prior head is exactly the sidebar two-slug swap (`users/interact-with-gnokey` → `users/using-gnokey`, `resources/gnodev` → `resources/gnodev-reference`) plus the airgapped-link repoint; the `docs` CI job is green at this sha, and all eight `gnokey-reference.md` anchors referenced tree-wide (`addpackage`, `querying-a-gnoland-network`, `run`, `call`, `making-transactions`, `vmqstorage`, `authgasprice`, `airgapped-signing`) resolve against the file's slugified headings. Rendering the testnet-URL cell reproduces the two literal backticks; master's zero-width-space form renders as a clean two-span split.

## Critical (must fix)
None.

## Warnings (should fix)
- **[table cell renders a stray double-backtick]** [`docs/builders/getting-started.md:275`](https://github.com/gnolang/gno/blob/eb829bec3/docs/builders/getting-started.md?plain=1#L275) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/getting-started.md#L275) — the testnet URL cell `` `https://``rpc.<testN>...` `` mishandles the anti-autolink split.
  <details><summary>details</summary>

  master used a zero-width space between `https://` and `rpc` inside one code span to stop GitHub autolinking the URL. This PR replaced it with a double backtick: `` `https://``rpc.<testN>.testnets.gno.land:443` ``. Under CommonMark/GFM the opening single-backtick run closes at the trailing single backtick, so the whole cell is one code span and the two inner backticks render as literal characters; python-markdown renders it `<code>https://``rpc.&lt;testN&gt;.testnets.gno.land:443</code>`. Either way the cell no longer renders as one clean inline URL. Fix: restore the zero-width-space split, or write the two segments as two separate single-backtick code spans (`` `https://` `` `` `rpc.<testN>.testnets.gno.land:443` ``). Verified on eb829bec3: `git diff origin/master -- docs/builders/getting-started.md` shows this PR replaced master's zero-width-space form with the double backtick, and rendering the cell reproduces the two literal backticks. [repro](comment_claude-opus-4-8.md)
  </details>

## Nits
- [`docs/users/using-gnokey.md:141`](https://github.com/gnolang/gno/blob/eb829bec3/docs/users/using-gnokey.md?plain=1#L141) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/users/using-gnokey.md#L141) — "Calling `Deposit` on the `wugnot` realm to wrap `1000ugnot`." is a verbless sentence fragment; fold it into the preceding sentence or make it a full sentence ("This calls `Deposit` ... to wrap `1000ugnot`.").
- [`docs/builders/getting-started.md:369`](https://github.com/gnolang/gno/blob/eb829bec3/docs/builders/getting-started.md?plain=1#L369) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/getting-started.md#L369) — link text reads "`addpkg` in Interact with gnokey" while the target page is now titled "gnokey command reference", so the label reads oddly. The same stale "Interacting with gnokey" label sits over reference-page anchors in [query-state-api.md:6](https://github.com/gnolang/gno/blob/eb829bec3/docs/builders/query-state-api.md?plain=1#L6) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/query-state-api.md#L6) and [:210](https://github.com/gnolang/gno/blob/eb829bec3/docs/builders/query-state-api.md?plain=1#L210) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/query-state-api.md#L210), [rpc-clients.md:29](https://github.com/gnolang/gno/blob/eb829bec3/docs/builders/rpc-clients.md?plain=1#L29) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/builders/rpc-clients.md#L29), and [gno-packages.md:49](https://github.com/gnolang/gno/blob/eb829bec3/docs/resources/gno-packages.md?plain=1#L49) · [↗](../../../../../.worktrees/gno-review-5873-h/docs/resources/gno-packages.md#L49). Cosmetic; the links resolve. The two other "Interacting with gnokey" labels (glossary.md:175, example-boards.md:41) point at the user guide `using-gnokey.md`, where the label is accurate, so both are fine.

## Missing Tests
None. Docs-only.

## Suggestions
None.

## Open questions
- `docs/builders/tutorial-minisocial.md:25` and `:119` link to `getting-started.md#4-before-you-deploy` and `#run-a-local-chain`, but the headings are "### 3. Before you deploy" (anchor `3-before-you-deploy`) and "### 5. Run a local chain" (anchor `5-run-a-local-chain`), so both anchors are dead. Both are identical on master and untouched by this PR (its only edit to that file repoints the `gnodev.md` link to `gnodev-reference.md`), so out of scope; not posted. Worth a one-line follow-up sweep since this PR is already a link-hygiene pass.
