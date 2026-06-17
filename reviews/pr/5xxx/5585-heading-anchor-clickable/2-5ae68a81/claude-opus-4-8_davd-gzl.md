# PR #5585: feat(gnoweb): make heading text clickable to set URL hash

URL: https://github.com/gnolang/gno/pull/5585
Author: davd-gzl | Base: master | Files: 24 | +285 -34
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `5ae68a81` (stale — +84 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5585 5ae68a81`

> Round-2 re-review. Prior round-1 verdict (commit `5c82b44a`) was REQUEST CHANGES on three correctness/a11y bugs. This round checks whether the redesign fixed them and looks for new regressions. Self-authored PR, reviewed adversarially.

**Verdict: APPROVE** — all three round-1 critical findings are resolved by the per-non-link-run redesign, and the round-1 CSS/test warnings are closed. One non-code blocker remains: the commit history carries `Co-Authored-By: Claude` trailers, which the current `AGENTS.md` forbids (it mandates `Assisted-By`). That is a rebase, not a code change. Code, tests, and ADR are merge-ready.

## Summary

Closes #5579: headings get auto IDs but the text is not clickable, so there is no quick way to copy a section permalink. The PR adds a goldmark AST transformer (`headingAnchorTransformer`) that wraps each contiguous run of non-link inline children of a heading in `<a class="heading-anchor" href="#id">…</a>`. Inline links inside a heading stay untouched and render with their own `<a>`, so nested anchors are impossible by construction. The default goldmark heading renderer still emits `<hN id="…">`.

This is the round-3 design from the PR's own history. Round-1 reviewed the round-1 design (wrap the whole heading in one `<a>`, with `aria-label`), which had three independently-critical bugs. The current head is a different implementation.

```
## Title with [link](/x)
round-1 design:  <h2><a heading-anchor>Title with <a href=/x>link</a></a></h2>   nested <a> — invalid
current design:  <h2><a heading-anchor>Title with </a><a href=/x>link</a></h2>   link is a run boundary — valid
```

## Round-1 findings: status

| # | Round-1 finding (commit `5c82b44a`) | Status at `5ae68a81` |
|---|---|---|
| Critical 1 | Nested `<a>` when heading contains an inline link | Resolved. Links are run boundaries; `isLinkLike` covers `KindLink`/`KindAutoLink`/`KindGnoLink`. Golden `all_cases.md.txtar` asserts the split. |
| Critical 2 | Unbalanced `</a>` when heading has no `id` | Resolved. `wrapHeadingChildren` returns early when `id` is absent or empty; `</a>` is emitted only by the `headingAnchorNode` renderer, which only exists when a wrap was created. |
| Critical 3 | `aria-label` overrides heading accessible name | Resolved. No `aria-label` anywhere; ADR documents the rejection and the fallback to wrapped-text accessible name. |
| Warning | Stale ADR (described rejected design) + `prxxxx` filename | Resolved. ADR rewritten to the transformer design; file renamed `pr5585_heading_anchor_links.md`. |
| Warning | Dead `::after` rule (no `content:`) in `c-realm-view` | Resolved. The dead rule is gone; CSS is the hoisted `.heading-anchor` + `:focus-visible` block. |
| Warning | CI `main.css` out of sync | Resolved (for this cause). `main.css` regenerated. Current CI red is infra noise, see below. |

Round-1.5 self-review (commit `32106472b`) raised: `Co-Authored-By` trailers (still present, see Warnings), PR body out of sync (resolved — body now describes the wrap-content design and is no longer "WIP"), `isLinkLike` maintainability (addressed — `MAINTAINERS:` comment added), idempotency (addressed — `KindHeadingAnchor` in `isLinkLike` makes a second pass a no-op), priority dependency (addressed — `PriorityLinkTransformer = 500` constant, `priorityHeadingAnchor = +499`), 12x `:focus-visible` CSS duplication (resolved — hoisted to one rule).

## Glossary

- `headingAnchorTransformer` — AST transformer that regroups heading children into anchor-wrapped runs.
- `headingAnchorNode` — synthetic inline node (`KindHeadingAnchor`) the transformer inserts; renders as `<a class="heading-anchor">`.
- `isLinkLike` — predicate that flags nodes rendering as `<a>` (so they are run boundaries, never wrapped).
- `GnoLink` — gnoweb's rewritten link node (`linkTransformer` turns `Link`/`AutoLink` into `GnoLink` at priority 500).

## CI

`main / test` and `gnobro / build` are red, but both are infrastructure failures unrelated to the diff: `main / test` fails in the Go-toolchain cache extraction step (`/usr/bin/tar: …toolchain@v0.0.1-go1.24.4…: Cannot open: File exists`), `gnobro / build` fails on `Command failed: go env GOPATH`. No test assertion or build of the changed package fails. `Merge Requirements` is red only on codeowner approval (alexiscolin / gfanton), expected. Locally `go test ./gno.land/pkg/gnoweb/markdown/...` passes.

## Critical (must fix)

None.

## Warnings (should fix)

- **[AGENTS.md violation — commit trailers]** commit history — four commits carry `Co-Authored-By: Claude Opus 4.7`, but `AGENTS.md` now mandates `Assisted-By` (NOT `Co-Authored-By`).
  <details><summary>details</summary>

  Commits `8e73c308a`, `1239d781d`, `325388ce5`, `32106472b` end with `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>`. [`AGENTS.md:74-75`](https://github.com/gnolang/gno/blob/5ae68a81/AGENTS.md#L74-L75) · [↗](../../../../../.worktrees/gno-review-5585/AGENTS.md#L74-L75) reads: "Add `Assisted-By` (NOT Co-Authored-By) lines or AI tool credits". The rule changed since round-1 (which quoted a blanket "Do NOT add Co-Authored-By"), but the trailer is still non-conformant either way. This was flagged at round-1 and not addressed. Fix: rebase to rewrite the four trailers to `Assisted-By:` (or drop them and disclose AI usage in the PR body). Not a code change, does not block on correctness, but it is a repo-policy gate a maintainer can reasonably hold the merge on.
  </details>

- **[noisy history]** commit history — 8 feature commits including two dead-end refactors (round-1 wrap-everything, round-2 sibling-anchor) plus a reverted `anchorMode` enum, none squashed.
  <details><summary>details</summary>

  The final code is the round-3 per-run transformer; commits `5c82b44aa` (wrap-everything), `8e73c308a`/`1239d781d` (sibling-anchor + the `anchorMode` enum that `325388ce5` then deleted) are superseded intermediate work. Squashing to one or two clean commits before merge shrinks the review surface and removes the obsolete trailers in one pass. Flagged at round-1; still open. Optional, but it pairs naturally with the trailer rebase above.
  </details>

## Nits

- [`05-composition.css:537-539`](https://github.com/gnolang/gno/blob/5ae68a81/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L537-L539) · [↗](../../../../../.worktrees/gno-review-5585/gno.land/pkg/gnoweb/frontend/css/05-composition.css#L537-L539) — `scroll-margin-top: var(--cr-space-24)` reverted per alexiscolin's review (sticky header would otherwise hide the linked heading). Confirmed sensible; left as a pointer for the codeowner thread, not an action item.
- [`ext_heading.go:117`](https://github.com/gnolang/gno/blob/5ae68a81/gno.land/pkg/gnoweb/markdown/ext_heading.go#L117-L127) · [↗](../../../../../.worktrees/gno-review-5585/gno.land/pkg/gnoweb/markdown/ext_heading.go#L117-L127) — renderer drops `WriteString` errors (`_, _ =`). Consistent with every other renderer in the package, so leave as-is.

## Missing Tests

- **[coverage gap, low risk]** [`ext_heading.go`](https://github.com/gnolang/gno/blob/5ae68a81/gno.land/pkg/gnoweb/markdown/ext_heading.go) · [↗](../../../../../.worktrees/gno-review-5585/gno.land/pkg/gnoweb/markdown/ext_heading.go) — codecov reports 90% patch coverage (6 lines missing). The consolidated [`golden/ext_heading/all_cases.md.txtar`](https://github.com/gnolang/gno/blob/5ae68a81/gno.land/pkg/gnoweb/markdown/golden/ext_heading/all_cases.md.txtar) · [↗](../../../../../.worktrees/gno-review-5585/gno.land/pkg/gnoweb/markdown/golden/ext_heading/all_cases.md.txtar) covers plain, bold/italic, single-link, multi-link, link-only, mention, empty, and after-empty headings — strong coverage of the run-splitting logic. The uncovered lines are likely the early-return `id`-absent guards (`wrapHeadingChildren:64-71`) and the `Dump` helper, none of which run on the goldmark path. The round-1.5 adversarial suite (image-in-heading hijack, footnote-not-enabled, bech32) verified these paths manually but is not in-tree. Acceptable to merge; a one-line raw-HTML-heading or image-in-heading fixture would close the gap if desired.

## Suggestions

- ADR [`pr5585_heading_anchor_links.md:32`](https://github.com/gnolang/gno/blob/5ae68a81/gno.land/adr/pr5585_heading_anchor_links.md#L32) · [↗](../../../../../.worktrees/gno-review-5585/gno.land/adr/pr5585_heading_anchor_links.md#L32) already documents the image-in-heading click-hijack consequence — good. No further action.

## Questions for Author

- None outstanding from a code standpoint. The only open item is the trailer/squash rebase before a codeowner merges.
