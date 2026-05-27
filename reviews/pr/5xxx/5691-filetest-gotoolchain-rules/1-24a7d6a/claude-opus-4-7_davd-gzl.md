# PR #5691: docs: add GOTOOLCHAIN filetest version matching rules to AGENTS.md and CLAUDE.md

URL: https://github.com/gnolang/gno/pull/5691
Author: ltzmaxwell | Base: master | Files: 2 | +18 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5691 24a7d6a` (then `gh -R gnolang/gno pr checkout 5691` inside it)

**Verdict: APPROVE** — docs-only change matches the existing Makefile contract; one open question from @jefft0 about whether CLAUDE.md actually pulls AGENTS.md into context.

## Summary

Documents that VM filetests (`gnovm/tests/files/`) compare against TypeCheckError messages whose exact wording depends on the Go toolchain version, so agents must either run via the gnovm Makefile (which exports `GOTOOLCHAIN` from `go.mod`'s `toolchain` directive) or set `GOTOOLCHAIN` themselves before `go test`. Adds a "Filetest version matching" subsection to [`AGENTS.md`](https://github.com/gnolang/gno/blob/24a7d6a/AGENTS.md#L52-L67) · [↗](../../../../../.worktrees/gno-review-5691/AGENTS.md#L52-L67) and swaps the raw-`go test` invocation in [`CLAUDE.md`](https://github.com/gnolang/gno/blob/24a7d6a/CLAUDE.md#L8) · [↗](../../../../../.worktrees/gno-review-5691/CLAUDE.md#L8) for `make -C gnovm _test.filetest`. Purely a documentation change — no code or test logic touched.

## Fix

Before: [`CLAUDE.md:8`](https://github.com/gnolang/gno/blob/24a7d6a/CLAUDE.md#L8) · [↗](../../../../../.worktrees/gno-review-5691/CLAUDE.md#L8) told agents to run `go test ./gnovm/pkg/gnolang/ -run Files -test.short` after gas/alloc changes, which uses the local Go toolchain and can produce spurious filetest diffs vs CI. After: routes through the gnovm Makefile target [`_test.filetest`](https://github.com/gnolang/gno/blob/24a7d6a/gnovm/Makefile#L155-L158) · [↗](../../../../../.worktrees/gno-review-5691/gnovm/Makefile#L155-L158), which inherits the `GOTOOLCHAIN ?= $(shell sed -n 's/^toolchain //p' ../go.mod)` export at [`gnovm/Makefile:21-22`](https://github.com/gnolang/gno/blob/24a7d6a/gnovm/Makefile#L21-L22) · [↗](../../../../../.worktrees/gno-review-5691/gnovm/Makefile#L21-L22). The new [`AGENTS.md` section](https://github.com/gnolang/gno/blob/24a7d6a/AGENTS.md#L52-L67) · [↗](../../../../../.worktrees/gno-review-5691/AGENTS.md#L52-L67) also documents the equivalent direct invocation for callers who can't use Make.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`AGENTS.md:63`](https://github.com/gnolang/gno/blob/24a7d6a/AGENTS.md#L63) · [↗](../../../../../.worktrees/gno-review-5691/AGENTS.md#L63) — example mirrors the Makefile target with `./gnovm/pkg/gnolang/files_test.go` (root-relative) rather than `cd gnovm && go test pkg/gnolang/...`. Both forms work; consider adding a one-line note that the command must run from repo root for `go.mod` to be reachable at `go.mod` (vs `../go.mod` inside `gnovm/`). Otherwise a reader who naively `cd`s into `gnovm/` first will get an empty `GOTOOLCHAIN`.

## Missing Tests

None — docs-only change.

## Suggestions

- [`CLAUDE.md:1`](https://github.com/gnolang/gno/blob/24a7d6a/CLAUDE.md#L1) · [↗](../../../../../.worktrees/gno-review-5691/CLAUDE.md#L1) — picking up on @jefft0's [review comment](https://github.com/gnolang/gno/pull/5691#discussion_r2603830823): consider adding `@AGENTS.md` at the very top of `CLAUDE.md` so Claude Code auto-loads the broader agent guide. Not strictly required for this PR (the new CLAUDE.md line is self-contained — `make -C gnovm _test.filetest` is enough on its own), but the "see AGENTS.md" cross-reference at [`CLAUDE.md:8`](https://github.com/gnolang/gno/blob/24a7d6a/CLAUDE.md#L8) · [↗](../../../../../.worktrees/gno-review-5691/CLAUDE.md#L8) is currently informational only; Claude Code reads `CLAUDE.md` automatically but not `AGENTS.md` unless explicitly imported. Reasonable to defer to a follow-up since this PR is narrowly scoped.

## Questions for Author

- Is the intent for `AGENTS.md` to be auto-loaded into Claude's context (in which case `@AGENTS.md` at the top of `CLAUDE.md` is needed), or just to serve as standalone documentation that any agent (Codex, generic LLMs, humans) can read on demand? Answer determines whether @jefft0's suggestion is in scope here.
