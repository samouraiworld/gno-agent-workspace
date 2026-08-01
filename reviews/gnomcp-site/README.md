# gnomcp site reviews

Reviews of [mcp.gno.dev](https://mcp.gno.dev/), the promo page for
[gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp). Source lives at
[`site/`](https://github.com/gnoverse/gno-mcp/tree/main/site) and deploys to
Netlify with no build step, so the served HTML is the committed HTML.

One pass per model: `review_<model>.md` plus its `comment_<model>.md` draft.
The site has no PR, so the draft targets a new issue on `gnoverse/gno-mcp`.

| Review | Draft | Date | Verdict |
|:-------|:------|:-----|:--------|
| [review_claude-opus-5.md](review_claude-opus-5.md) | [comment_claude-opus-5.md](comment_claude-opus-5.md) | 2026-07-30, rechecked 2026-08-01 | REQUEST CHANGES |

The recheck re-derived every finding from a fresh clone at `ef19be4` and from
the live deploy. All 19 originals stand; four line-number or count details were
corrected and three findings added. See the review's *Recheck* section.
