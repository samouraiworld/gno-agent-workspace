# PR [#5593](https://github.com/gnolang/gno/pull/5593): feat(gnoweb): add `:::details` collapsible block

URL: https://github.com/gnolang/gno/pull/5593
Author: davd-gzl | Base: master | Files: 16 | +577 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep, multi-lens) | Commit: 76ccef0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5593 76ccef0`
Fix shipped: the in-branch upgrade below was committed and pushed to the PR as 1a10b2c9a.

**TL;DR:** Adds a `:::details Summary … :::` markdown block to gnoweb that renders a plain collapsible `<details>`/`<summary>`, without the icon and colored border the alert syntax carries. Deep multi-lens review of the fenced-block parser found four correctness leaks around the closing fence, all fixed in-branch by matching goldmark's own fenced-block advance discipline and adopting pandoc variable-length fences.

**Verdict: APPROVE (post-fix)** — the reviewed commit 76ccef0 had four parser leaks (empty body, indented close, nested block, `:::` inside a code sample); all are fixed in the working tree with new golden fixtures locking each case. No XSS vector; escaping and the happy paths were already correct.

## Summary

The extension parses a pandoc-style fenced container. The reviewed commit's block parser had one structural flaw with two faces: `Open` and `Continue` both call `reader.Advance(len(line))`, where `line` includes the trailing newline. Advancing across the newline moves the reader onto the *next* line, then goldmark's block loop advances again, so one line is skipped per fence. That skip is why the documented minimal form `:::details X` / `:::` never closed on its fence and leaked `<p>:::</p>` into the body. A second, independent limitation: the close fence was a fixed exact-`:::` match, so any `:::` line inside the body (a nested block, or a code sample showing the syntax) closed the block early. The fix advances to end-of-line only (the loop consumes the newline, matching goldmark's `fencedCodeBlockParser`) and stores the opening fence length so a block closes only on a fence of at least its own colon count.

## Examples

| Input | Reviewed 76ccef0 | After fix |
|-------|------------------|-----------|
| `:::details X` / `:::` (empty body) | `<div><p>:::</p></div>`, block never closed by fence | `<div></div>`, closed cleanly |
| `:::details X` / body / `   :::` (3-space close) | close not recognized, `:::` leaks as text | closed |
| `::::details` / `:::details` inner / `:::` / `::::` | inner fence prefix not even recognized | clean nested `<details>` |
| `::::details` / ` ``` ` / `:::` / ` ``` ` / `::::` | code block split, stray top-level `<pre>:::</pre>` | code block preserves `:::` |
| `:::details[open]NoSpace` | `<p>…</p>` (unchanged, documented) | `<p>…</p>` (locked by fixture) |

## Glossary

- Fenced container / fenced div: a block delimited by fence lines (here `:::…`) that wraps other markdown, pandoc's `:::` convention.
- Goldmark block loop: goldmark's per-line pass that calls each open block's `Continue`, parent before child, then tries to open new child blocks.

## Fix

`parseOpenFence` counts the leading colon run (minimum three) and returns it as `Fence`, stored on `DetailsBlock`; `isCloseFence(line, min)` matches a run of at least `min` colons and now tolerates up to three leading spaces, symmetric with the opening fence. `Open` and `Continue` use [`reader.AdvanceToEOL()`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/markdown/ext_details.go#L166) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/markdown/ext_details.go#L204) instead of `reader.Advance(len(line))`, leaving the newline for the block loop so the following line reaches `Continue`. Verified on the fix: the ten pre-existing golden fixtures are byte-identical, and seven new fixtures lock the repaired cases.

## Critical (must fix)

None.

## Warnings (should fix)

All four found in 76ccef0 and fixed in-branch; kept here as the record of what the deep review surfaced.

- **[documented minimal form never closed]** [`ext_details.go:172-179`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/markdown/ext_details.go#L172-L179) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/markdown/ext_details.go#L210-L216) — an empty-body block (`:::details X` immediately followed by `:::`) did not close on the fence; the `:::` leaked as `<p>:::</p>` and the block ran to end-of-document.
  <details><summary>details</summary>

  Root cause is the newline over-advance in `Open`. `reader.Advance(len(line))` crossed the opening line's `\n`, landing the reader on line 2; goldmark's block loop then advanced again, so `Continue` never saw the `:::` line and the paragraph parser claimed it. Confirmed by instrumenting `Open`/`Continue`: no `Continue` call fired for the empty-body input. Fix: `AdvanceToEOL()` in `Open`, mirroring `fencedCodeBlockParser`. Locked by `golden/ext_details/valid_empty_body.md.txtar`.
  </details>

- **[open/close indentation asymmetry]** [`ext_details.go:127-129`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/markdown/ext_details.go#L127-L129) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/markdown/ext_details.go#L150-L165) — the opening fence allowed up to three leading spaces, the closing fence allowed none, so `   :::` failed to close and leaked as body text.
  <details><summary>details</summary>

  `isCloseFence` compared the trailing-trimmed line to exactly `:::`, so any leading space broke the match. Fix: `isCloseFence` now skips up to three leading spaces before counting colons, symmetric with `Open`. Locked by `valid_indented_close.md.txtar`.
  </details>

- **[nested blocks leaked a fence]** [`ext_details.go:172-179`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/markdown/ext_details.go#L172-L179) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/markdown/ext_details.go#L210-L216) — with the fixed exact-`:::` fence, a nested `:::details` could not be closed independently of its parent; one closing fence leaked into body text.
  <details><summary>details</summary>

  Adopted pandoc variable-length fences: open the outer block with more colons than anything inside (`::::details … ::::`). A block closes on the first fence of at least its own colon count, so a `:::` inner fence no longer closes a `::::` outer. Locked by `valid_nested.md.txtar`. Same-length nesting still leaves a stray `<p>:::</p>`; that is the pandoc contract and is documented in the ADR and the docs realm.
  </details>

- **[`:::` inside a code sample split the block]** [`ext_details.go:172-179`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/markdown/ext_details.go#L172-L179) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/markdown/ext_details.go#L210-L216) — a `:::`-only line inside a fenced code block within a details body closed the details block early and emitted a stray top-level `<pre><code>:::</code></pre>`.
  <details><summary>details</summary>

  goldmark calls the container's `Continue` before the inner code fence's, so the container grabbed the `:::` line. This is the headline doc use case (showing the `:::details` syntax itself). Fix is the same variable-length fence: wrap the sample with `::::details … ::::`. Locked by `valid_codefence_colons.md.txtar`, whose golden output preserves the inner `:::details Example … :::` verbatim inside the code block.
  </details>

## Nits

- [`ext_details.go:86`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/markdown/ext_details.go#L86) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/markdown/ext_details.go#L89) — `NewDetailsParser` allocated a fresh struct per call; siblings `ext_alert.go` / goldmark keep a package `default…Parser` singleton. Aligned to a `defaultDetailsParser` singleton.
- [`pr5593_gnoweb_details_block.md:35`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/adr/pr5593_gnoweb_details_block.md?plain=1#L35) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/adr/pr5593_gnoweb_details_block.md#L41) — ADR showed `<details class="gno-details" [open]>` with literal brackets (not valid HTML). Reworded to prose and updated the fence contract.

## Missing Tests

Closed in this upgrade. Seven fixtures added under `golden/ext_details/`: `valid_empty_body`, `valid_indented_close`, `valid_nested`, `valid_codefence_colons`, `valid_consecutive`, `invalid_no_space_flag`, `valid_summary_escaped` (the last locks summary HTML escaping: `<script>` becomes `<!-- raw HTML omitted -->`, `&`/`"` are entity-escaped).

## Suggestions

- [`06-blocks.css:2569`](https://github.com/gnolang/gno/blob/76ccef0/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L2569) · [↗](../../../../../.worktrees/gno-review-5593/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L2569) — the chevron `transition: transform 0.15s ease` is the only animated summary chevron in the stylesheet and there is no `prefers-reduced-motion` handling anywhere in `frontend/css/`; the duration is a literal rather than a `--g-duration-*` token. Left as-is to avoid introducing a lone reduced-motion guard and a `main.css` rebuild in this PR; worth a follow-up that adds the guard package-wide.

## Open questions

- `:::details` inside a blockquote (`> :::details …`) renders the body wrapped in a nested `<blockquote>`. Works and is locked implicitly, but it is a surprising shape; not posted, no author action needed.

## What changed in this upgrade

- `ext_details.go`: variable-length fence (`Fence` field, `minDetailsFence`), `AdvanceToEOL` in `Open`/`Continue`, indent-tolerant `isCloseFence`, parser singleton, `Fence` in `Dump`.
- `pr5593_gnoweb_details_block.md`: fence contract, no-space rejection, bracket-notation fix.
- `markdown.gno`: accurate close-fence wording, a nesting example using `::::`.
- Seven new golden fixtures; the ten existing ones unchanged.

Verified on the fix: `go test ./gno.land/pkg/gnoweb/...` green (59s), `gno lint ./gno.land/r/docs/markdown` clean, `gofmt`/`go vet` clean, every fixture case HTML-balanced.
