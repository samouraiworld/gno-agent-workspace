# PR #5656: docs(builders): consolidate and clean up builder documentation

URL: https://github.com/gnolang/gno/pull/5656
Author: davd-gzl | Base: master | Files: 32 | +816 -808
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5656 b00a4b2f7` (then `gh -R gnolang/gno pr checkout 5656` inside it)

**Verdict: REQUEST CHANGES** — solid IA cleanup, but `devtest` is presented as the gnodev default key while the source still ships `test1`; the `gnokey ... devtest` example will fail for any reader.

## Summary

Collapses `docs/builders/` from 11 to 8 pages, renames a few for clarity (`become-a-gnome` → `contributor-guide`, `what-is-gnolang` → `what-is-gno`, `connect-clients-and-apps` → `rpc-clients`, `example-minisocial-dapp` → `tutorial-minisocial`), rewrites `local-dev-with-gnodev` as a reference at [`docs/resources/gnodev.md`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gnodev.md) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md), and tightens the minisocial tutorial (full files instead of embedmd-sliced fragments, fewer redundant gnodev startup walkthroughs). Sidebar, cross-links, README, and whitepaper all updated to match. The headline pages — `getting-started`, `cheatsheet`, `editor-setup`, `gnodev`, `tutorial-minisocial` — read well and are clearly the path the project wants to put new builders on.

## Glossary

- `devtest`: name the docs use for the gnodev default account (the publicly-known `test1` mnemonic, addr `g1jg8mtutu9...`). Source still calls it `test1`.
- embedmd: docs preprocessor that pulls labeled fragments out of files in `docs/_assets/` into fenced blocks. Slice patterns (`/regex/ $`) often break silently when the source file shifts.

## Critical (must fix)

- **[docs claim a key name the binary doesn't use]** [`docs/resources/gnodev.md:19`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gnodev.md#L19) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L19) — gnodev still prints `name=test1`, not `name=devtest`; the example call on [line 86](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gnodev.md#L86) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L86) will fail with "key not found".
  <details><summary>details</summary>

  The PR body asserts the rename "matches the cheatsheet and gnodev's actual default" — but [`gno.land/pkg/integration/node_testing.go:25`](https://github.com/gnolang/gno/blob/b00a4b2f7/gno.land/pkg/integration/node_testing.go#L25) · [↗](../../../../../.worktrees/gno-review-5656/gno.land/pkg/integration/node_testing.go#L25) still defines `DefaultAccount_Name = "test1"`, and [`contribs/gnodev/app.go:29`](https://github.com/gnolang/gno/blob/b00a4b2f7/contribs/gnodev/app.go#L29) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/app.go#L29) wires that through as `DefaultDeployerName`. Nothing in this PR or its declared dependencies (#5563 is `gnodev version`, unrelated) flips the constant.

  Concrete consequences for a reader following the docs:
  - `gnodev .` will log `name=test1`, not `name=devtest` as [`docs/resources/gnodev.md:19`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gnodev.md#L19) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L19) shows.
  - The `gnokey maketx call ... devtest` example at [`docs/resources/gnodev.md:86`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gnodev.md#L86) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L86) will error — there is no `devtest` entry in the keybase. The reader has to know to substitute `test1`.
  - [`docs/cheatsheet.md:69`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/cheatsheet.md#L69) · [↗](../../../../../.worktrees/gno-review-5656/docs/cheatsheet.md#L69) tells the reader to recover a key under the name `devtest` using the test1 mnemonic — works, but now the reader has both `test1` (the gnodev default) and `devtest` (their own import) in the keybase, mapped to the same address. Confusing without explanation.

  Fix: pick one of two paths. (a) Land the upstream rename of `DefaultAccount_Name` to `devtest` in `gno.land/pkg/integration/node_testing.go` first (or in this PR), then the docs match. (b) Revert the docs to `test1` everywhere — `docs/resources/gnodev.md` (8 occurrences), `docs/cheatsheet.md:69`. Either is fine, but the docs and the binary have to agree.
  </details>

## Warnings (should fix)

- **[stale link to deleted file]** [`contribs/gnodev/README.md:7`](https://github.com/gnolang/gno/blob/b00a4b2f7/contribs/gnodev/README.md#L7) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/README.md#L7) — points at `docs/builders/local-dev-with-gnodev.md`, which this PR deletes.
  <details><summary>details</summary>

  The PR updated [`contribs/gnodev/app.go:463`](https://github.com/gnolang/gno/blob/b00a4b2f7/contribs/gnodev/app.go#L463) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/app.go#L463) (the in-terminal help string) to the new `https://docs.gno.land/resources/gnodev` URL but missed the README sitting two lines into the same directory. One-line fix:

  ```diff
  -[docs/builders/local-dev-with-gnodev.md](../../docs/builders/local-dev-with-gnodev.md).
  +[docs/resources/gnodev.md](../../docs/resources/gnodev.md).
  ```
  </details>

- **[orphaned page]** [`docs/builders/quickstart.md`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/builders/quickstart.md) · [↗](../../../../../.worktrees/gno-review-5656/docs/builders/quickstart.md) — dropped from `misc/docs/sidebar.json` but still linked from three places.
  <details><summary>details</summary>

  The diff at [`misc/docs/sidebar.json`](https://github.com/gnolang/gno/blob/b00a4b2f7/misc/docs/sidebar.json) · [↗](../../../../../.worktrees/gno-review-5656/misc/docs/sidebar.json) removes `builders/quickstart` from the sidebar without restoring it. The file still exists and three pages still link to it:

  - [`docs/README.md:13`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/README.md#L13) · [↗](../../../../../.worktrees/gno-review-5656/docs/README.md#L13) — "Need Docker, a … See **[Quick Start](builders/quickstart.md)**"
  - [`docs/builders/what-is-gno.md:5`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/builders/what-is-gno.md#L5) · [↗](../../../../../.worktrees/gno-review-5656/docs/builders/what-is-gno.md#L5) — "[Quick Start](./quickstart.md) if you just want the commands"
  - [`docs/builders/getting-started.md:12`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/builders/getting-started.md#L12) · [↗](../../../../../.worktrees/gno-review-5656/docs/builders/getting-started.md#L12) — "See [Quick Start](./quickstart.md)"

  Either: (a) keep `quickstart` in the sidebar so it's discoverable from the nav, or (b) delete the file and remove the three inline links. The current shape — reachable only by inline link from three top-of-funnel pages but not in the sidebar — is unstable: any future rewrite that drops one of those inline links silently strands the file.
  </details>

## Nits

- [`docs/cheatsheet.md:169`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/cheatsheet.md#L169) · [↗](../../../../../.worktrees/gno-review-5656/docs/cheatsheet.md#L169) — `> [Multisig keys](users/interact-with-gnokey.md)` has no anchor; reader lands at the top of a long page.
- [`docs/builders/tutorial-minisocial.md:170`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/builders/tutorial-minisocial.md#L170) · [↗](../../../../../.worktrees/gno-review-5656/docs/builders/tutorial-minisocial.md#L170) — uses `{MYKEY}` while the rest of the new docs use `MyKey`. One placeholder convention.
- [`docs/cheatsheet.md:316`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/cheatsheet.md#L316) · [↗](../../../../../.worktrees/gno-review-5656/docs/cheatsheet.md#L316) — "Create a Run Script" `maketx run` example omits `-broadcast`; every other `maketx` example in the cheatsheet (call, send, addpkg) includes it. Either drop `-broadcast` throughout or add it here.
- [`docs/builders/tutorial-minisocial.md:175-205`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/builders/tutorial-minisocial.md#L175-L205) · [↗](../../../../../.worktrees/gno-review-5656/docs/builders/tutorial-minisocial.md#L175-L205) — the bonus & main `String()` sections show the full `types-2.gno` / `types-2-bonus.gno` files. Clear, but a one-line "Replace `types.gno` with:" intro would prevent readers parsing it as a new file alongside the previous `types-1.gno` block.

## Missing Tests

None — docs-only change.

## Suggestions

- [`docs/resources/gnodev.md:107-117`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gnodev.md#L107-L117) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L107-L117) — the new key-binding table is correct against [`contribs/gnodev/app.go:476-545`](https://github.com/gnolang/gno/blob/b00a4b2f7/contribs/gnodev/app.go#L476-L545) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/app.go#L476-L545) (`N`/`P` are next/previous tx, `R` reloads, `Ctrl+R` resets). [`contribs/gnodev/README.md:27-30`](https://github.com/gnolang/gno/blob/b00a4b2f7/contribs/gnodev/README.md#L27-L30) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/README.md#L27-L30) still describes them as "cancel last action / redo cancelled action" — wrong, and now in two places. Worth a follow-up to delete that stale block from the contrib README and just cross-link to `docs/resources/gnodev.md`.
- [`docs/resources/gno-packages.md:51-60`](https://github.com/gnolang/gno/blob/b00a4b2f7/docs/resources/gno-packages.md#L51-L60) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gno-packages.md#L51-L60) — the new "Import rules" section is the canonical place for these rules now that `anatomy-of-a-gno-package.md` is gone. Worth a one-line addition: "Ephemeral packages can import any of the above." Currently the section says `/e/` "cannot be imported by anything" but is silent on what `/e/` itself can import, which the deleted page covered explicitly.

## Questions for Author

- Is `quickstart` intentionally kept out of the sidebar (inline-link-only navigation) or an oversight? If intentional, the PR body should say so — every other rename/removal is called out.
- The PR body lists #5563 as a dep, but #5563 is `feat(gnodev): add gnodev version` — unrelated to the `test1` → `devtest` rename you describe. Is there a missing dependency PR that does the source-side rename?
