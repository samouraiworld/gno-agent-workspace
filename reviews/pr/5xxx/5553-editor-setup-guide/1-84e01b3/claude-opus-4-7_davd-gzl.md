# PR #5553: docs: add editor setup guide

URL: https://github.com/gnolang/gno/pull/5553
Author: davd-gzl | Base: master | Files: 8 | +70 -5
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `84e01b3` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5553 84e01b3`

**Verdict: APPROVE** — small, additive docs PR; new `docs/builders/editor-setup.md` defers to upstream `gnopls` README as the source of truth, hooks correctly into sidebar/README/getting-started/install/local-dev/power-users, and bumps the stale `Go 1.24+` references to `1.25+` (matches `go.mod`). Three reviewers already approved (`notJoon`, `alexiscolin`, `moul`); all inline threads resolved; docs lint clean.

Self-review disclosure: PR author and reviewer are the same GitHub identity (`davd-gzl`). This is a docs-only change with multiple independent maintainer approvals on record, so the review serves as a structured second pass against the rendered files, not as an independent gate.

## Summary

Adds a 53-line `docs/builders/editor-setup.md` that points users at `gnopls` (the Gno LSP fork of `gopls`) and the VS Code extension, with a three-bullet "did it actually wire up?" smoke test (completion, hover, diagnostics) and pointers to format-on-save plus `gnodev` hot-reload. Sidebar, top-level README, `CONTRIBUTING.md`, `install.md`, `getting-started.md`, `local-dev-with-gnodev.md`, and `power-users.md` are all cross-linked. The Go-toolchain version requirement in `CONTRIBUTING.md` and `install.md` is bumped from `1.24+` to `1.25+`, matching the current [`go.mod`](https://github.com/gnolang/gno/blob/84e01b3/go.mod#L3) · [↗](../../../../../.worktrees/gno-review-5553/go.mod#L3) (`go 1.25.9`, set by PR #5441).

## Fix

Earlier revisions of this page duplicated the `gnopls` README (per-editor install instructions, plugin lists). After [@moul's feedback](https://github.com/gnolang/gno/pull/5553#issuecomment-4429403889) the page was rewritten to defer to upstream — current head delegates editor-specific setup to the `gnopls` README ([editor-setup.md:13-15](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/editor-setup.md#L13-L15) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/editor-setup.md#L13-L15)) and only carries the parts that belong in the Gno docs proper: where the LSP lives, the VS Code shortcut, a smoke test, and links onward to format-on-save and `gnodev`. The Go-version bump rides along to keep the "Install from source" prerequisites accurate ([install.md:54](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/install.md#L54) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/install.md#L54), [install.md:131](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/install.md#L131) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/install.md#L131), [CONTRIBUTING.md:16](https://github.com/gnolang/gno/blob/84e01b3/CONTRIBUTING.md#L16) · [↗](../../../../../.worktrees/gno-review-5553/CONTRIBUTING.md#L16)).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`docs/builders/editor-setup.md:8`](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/editor-setup.md#L8) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/editor-setup.md#L8) — em-dash inconsistency: line 8 uses `-` (hyphen) where the rest of the file (lines 3, 10, 17, 22, 25, 27, 29, 32, 39, 45, 51) uses `—` (em-dash). Trivial.
- [`docs/builders/local-dev-with-gnodev.md:14`](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/local-dev-with-gnodev.md#L14) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/local-dev-with-gnodev.md#L14) — the editor-setup link here lands in an "Once installed, verify that `gnodev` is available" preamble, before the user has done anything Gno-specific. Other pages put the link in a Next-steps or follow-on context. Not blocking; the placement still works.
- [`docs/users/power-users.md:33`](https://github.com/gnolang/gno/blob/84e01b3/docs/users/power-users.md#L33) · [↗](../../../../../.worktrees/gno-review-5553/docs/users/power-users.md#L33) — "Editor Setup" is listed under "Tools" alongside web IDEs (Gno Studio, Playground). Editor setup is a configuration guide, not a tool. Minor categorisation drift; not worth blocking a docs PR.

## Missing Tests

N/A — pure documentation change, no executable surface. Docs lint passes (`make -C docs lint` clean, run against the worktree).

## Suggestions

- [`docs/builders/editor-setup.md:25-30`](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/editor-setup.md#L25-L30) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/editor-setup.md#L25-L30) — the smoke-test list is the highest-value novel content on the page (the rest is router/glue). Consider a single screenshot or a one-line "you should see" snippet under each bullet in a follow-up — [@alexiscolin's review](https://github.com/gnolang/gno/pull/5553#pullrequestreview-4178993505) flagged the screenshot gap and the author acknowledged it ([comment](https://github.com/gnolang/gno/pull/5553#issuecomment-4381693304)) as out-of-scope. Tracking it as a follow-up is fine; flagging here so it doesn't fall off the radar.
- [`docs/builders/editor-setup.md:6-18`](https://github.com/gnolang/gno/blob/84e01b3/docs/builders/editor-setup.md#L6-L18) · [↗](../../../../../.worktrees/gno-review-5553/docs/builders/editor-setup.md#L6-L18) — the "Language server" section names `gnopls` and VS Code but doesn't list other supported editors by name (Neovim, Vim, Emacs, Zed, Sublime). A reader scanning the page won't know whether their editor is supported without clicking through to `gnopls`. One-line "Other editors: see the gnopls README — Neovim, Vim, Emacs, Zed, Sublime are covered there" would close the gap without re-duplicating the upstream content moul asked to remove.

## Questions for Author

None.
