# gnomcp site reviews

Reviews of [mcp.gno.dev](https://mcp.gno.dev/), the promo page for
[gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp). Source lives at
[`site/`](https://github.com/gnoverse/gno-mcp/tree/main/site) and deploys to
Netlify with no build step, so the served HTML is the committed HTML.

One pass per model: `review_<model>.md` plus its `comment_<model>.md` draft.
The site has no PR, so the draft targets a new issue on `gnoverse/gno-mcp`.

| Review | Draft | Date | Verdict | Filed |
|:-------|:------|:-----|:--------|:------|
| [review_claude-opus-5.md](review_claude-opus-5.md) | [comment_claude-opus-5.md](comment_claude-opus-5.md) | 2026-07-30, rechecked 2026-08-01 and 2026-08-03 | REQUEST CHANGES | [gno-mcp#69](https://github.com/gnoverse/gno-mcp/issues/69) |

Each recheck re-derived every finding from a fresh clone at `ef19be4` and from
the live deploy. The first added three findings and corrected four counts; the
second, run immediately before filing, moved three anchors off a container line
onto the line carrying the claim and corrected one attribution. All 22 findings
stand. See the review's two *Recheck* sections.

The issue opens on a note saying an agent wrote it and that a human pass has not
run yet, so a maintainer reading it knows to treat each item as a claim with
evidence attached.
