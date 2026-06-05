# PR #5656: docs(builders): consolidate and clean up builder documentation

URL: https://github.com/gnolang/gno/pull/5656
Author: davd-gzl | Base: master | Files: 29 | +330 -835
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `3a8f6bf1e` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5656 3a8f6bf1e`

Round 2 re-review. Prior round (`1-b00a4b2f7`, verdict REQUEST CHANGES) raised one Critical (`devtest` vs `test1`), two Warnings (stale gnodev README link, orphaned `quickstart`), four Nits, two Suggestions, two Questions.

**Verdict: NEEDS DISCUSSION** — both Warnings and the actionable Nits are fixed and the IA is clean, but the round-1 Critical is only half-addressed: docs still assert gnodev prints `name=devtest`, while the source default is `test1` and the actual printed name comes from the reader's own keybase. A footnote now documents the address+mnemonic, which removes the "key not found" dead-end, so this is a judgment call on whether the displayed-name mismatch is merge-blocking or a follow-up.

## Summary

Collapses `docs/builders/` from 11 to 8 pages, renames the rest, moves the gnodev guide to [`docs/resources/gnodev.md`](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md) as a reference, and tightens the minisocial tutorial. Since round 1 the branch absorbed a 24-commit master merge, renamed the sidebar section `Resources` → `References`, re-wired `quickstart` into both the sidebar and the README, fixed the stale gnodev README link, and added a footnote to `gnodev.md` documenting the `devtest` address and mnemonic. CI is green (the only red check is the bot merge-requirements gate, which needs a maintainer approval, not a code fix). Docs lint passes.

## Glossary

- `devtest`: the name the docs use for the gnodev default deployer account (publicly-known mnemonic, addr `g1jg8mtutu9...`). The source constant is still `test1`.
- keybase: the local `gnokey` keystore. gnodev's startup log prints whatever name the keybase maps the default deployer address to, not a hardcoded string.

## Round-1 finding status

| # | Round-1 finding | Status |
|---|---|---|
| Critical | `devtest` claimed as gnodev default, source ships `test1` | Partial — see Critical below |
| Warning 1 | stale `local-dev-with-gnodev.md` link in gnodev README | Fixed |
| Warning 2 | orphaned `quickstart` (dropped from sidebar, linked from 3 pages) | Fixed |
| Nit | `{MYKEY}` placeholder in tutorial | Fixed (`MyKey`) |
| Nit | cheatsheet anchor / `-broadcast` consistency | N/A — `docs/cheatsheet.md` no longer in this PR |
| Nit | full-file `types-2*.gno` intro line | Resolved (see tutorial) |
| Suggestion | stale "cancel/redo" key bindings in gnodev README | Not addressed — still wrong, out of PR scope |
| Question | quickstart intentionally sidebar-omitted? | Resolved — now in sidebar |
| Question | missing source-rename dependency PR? | Unresolved — no PR flips the constant |

## Critical (must fix)

- **[docs claim a key name the binary may not print]** [`docs/resources/gnodev.md:19`](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L19) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L19) — the startup-log sample shows `name=devtest`, but gnodev prints the keybase name for the default address, which for a conventional setup is `test1` (or `_default#g1jg8m` if absent), never `devtest`.
  <details><summary>details</summary>

  The display name is not a constant. [`contribs/gnodev/setup_address_book.go:43-49`](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/setup_address_book.go#L43-L49) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/setup_address_book.go#L43-L49) looks up `defaultDeployerAddress` in the keybase and logs `names[0]` — whatever the reader named that key. If the address isn't in the keybase, [line 55](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/setup_address_book.go#L55) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/setup_address_book.go#L55) synthesizes `_default#g1jg8m`. The address itself is wired from [`contribs/gnodev/app.go:33`](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/app.go#L33) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/app.go#L33) → [`gno.land/pkg/integration/node_testing.go:26`](https://github.com/gnolang/gno/blob/3a8f6bf1e/gno.land/pkg/integration/node_testing.go#L26) · [↗](../../../../../.worktrees/gno-review-5656/gno.land/pkg/integration/node_testing.go#L26), and the default name is still [`DefaultAccount_Name = "test1"`](https://github.com/gnolang/gno/blob/3a8f6bf1e/gno.land/pkg/integration/node_testing.go#L25) · [↗](../../../../../.worktrees/gno-review-5656/gno.land/pkg/integration/node_testing.go#L25). Nothing in this PR flips the constant, and the round-1 question about a source-rename dependency PR is still unanswered (#5755, the listed dep, is a docs PR).

  What changed since round 1, and why this is now NEEDS DISCUSSION rather than a hard block:
  - The footnote at [`docs/resources/gnodev.md:137-138`](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L137-L138) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L137-L138) now documents the address and mnemonic. The round-1 "the `gnokey ... devtest` example errors with key not found" failure is no longer a silent dead-end: a reader who follows the footnote can import the key under any name. The cheatsheet `recover devtest` step that round 1 flagged is also gone (the cheatsheet is no longer part of this PR).
  - What remains is a cosmetic-but-real mismatch: every `devtest` reference in the page (lines [19](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L19), [26](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L26), [45](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L45), [55](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L55), [86](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L86), [102](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L102), [124](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L124)) presents `devtest` as the name the reader will see and type, but a clean machine running `gnodev .` with the conventional `test1` key (or no key) will not show `devtest`. The sample log at line 19 is the most concrete: it claims output the binary won't produce.

  Fix: pick one and make the page internally consistent with the binary. (a) Land the `DefaultAccount_Name` → `devtest` rename in `node_testing.go` (this PR or a stacked one) so the binary matches the docs. (b) Revert the page to `test1` and lean on the footnote to explain the publicly-known mnemonic. The current state — name `test1` in source, `devtest` in docs, footnote bridging the two — is defensible for a careful reader but will confuse anyone who diffs the sample log against their terminal.
  </details>

## Warnings (should fix)

None.

## Nits

- [`contribs/gnodev/README.md:7`](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/README.md#L7) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/README.md#L7) — link now correctly points at `docs/resources/gnodev.md` (round-1 Warning 1 resolved). No action.

## Missing Tests

None — docs-only change.

## Suggestions

- [`contribs/gnodev/README.md:21,30,33,34`](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/README.md#L20-L34) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/README.md#L20-L34) — the contrib README key-binding block is still wrong and now disagrees with the new canonical page. It calls `P` "Cancel the last action" and `N` "Redo the last cancelled action", but [`contribs/gnodev/app.go:465-466,539,533`](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/app.go#L465-L466) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/app.go#L465-L466) bind them to Previous/Next TX, which the new [`docs/resources/gnodev.md:114`](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gnodev.md#L114) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gnodev.md#L114) gets right. It also labels reset/exit `Cmd+R`/`Cmd+C` while the binary uses `Ctrl+R`/`Ctrl+C` ([`app.go:472,452`](https://github.com/gnolang/gno/blob/3a8f6bf1e/contribs/gnodev/app.go#L472) · [↗](../../../../../.worktrees/gno-review-5656/contribs/gnodev/app.go#L472)). This block is outside the PR's edited lines, so it's a fair follow-up rather than a blocker: delete it from the contrib README and cross-link to `docs/resources/gnodev.md` as the single source of truth.
- [`docs/resources/gno-packages.md:55-60`](https://github.com/gnolang/gno/blob/3a8f6bf1e/docs/resources/gno-packages.md#L55-L60) · [↗](../../../../../.worktrees/gno-review-5656/docs/resources/gno-packages.md#L55-L60) — the Import rules section still describes what each type can be imported *by*, but not what `/e/` itself can import. The deleted `anatomy-of-a-gno-package.md` covered this; the nearby line "Can call both crossing and non-crossing functions" partly fills the gap. A one-line "Ephemeral packages can import any of the above" would close it.

## Questions for Author

- Is there a plan to land the `DefaultAccount_Name` → `devtest` source rename, or is the intent to keep `test1` in source and rely on the footnote? The answer decides whether the Critical is a "fix the docs" or "ship a stacked code PR" task.
- alexiscolin's SEO concern (old paths 404 without a Docusaurus redirect) is tracked in gnolang/docs.gno.land#75 per your reply — confirming that follow-up is enough to unblock the deletions here.
