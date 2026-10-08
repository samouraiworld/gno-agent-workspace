# Review: [#6297](https://github.com/gnolang/gno/pull/6297)

Event: APPROVE
Verdict: APPROVE. Both round-1 Warnings are fixed at this head. Two Nits and a Suggestion remain on how icon labels reach the TOC, and none of them blocks.
Model: claude-opus-5-5, solo review (two finders, one judge and writer)
Commit: 6a68bc69fc638c389d2308743e2fb1d11cc326bc
Overview: [overview.md](../overview.md)
Open the code: [6a68bc6](https://github.com/gnolang/gno/tree/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown)
Round: 2, covering the fix commits 84df2c459..6a68bc69f only. 2 finders, the judge acting as reflector, 8 candidates, each one rerun from scratch by the judge, which was not its finder; 8 confirmed, none above Suggestion. Round 1 at this head: the stroke Warning is resolved (`TestIconTableMatchesSource` passes, and fails with the old `icons_gen.go`), the duplicate-id Warning is resolved (round 1's test passes), the link-hint Nit is resolved (round 1's test passes), the fuzz Test is resolved (the seeds pass), the nested_emphasis Test is resolved (the file is deleted), the three ADR Nits are resolved (the ADR text matches the code), and the TOC Suggestion is resolved, with the two new Nits below. Carried open, the author declining them for a follow-up: `ext_icons.go:105`, `ext_icons.go:236`, `iconset/main.go:122`, `ext_icons.go:182`.

## Body

## gno.land/pkg/gnoweb/markdown/utils.go:206 [gh](https://github.com/gnolang/gno/blob/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown/utils.go#L206) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/utils.go#L206) · Nit

Nit: this case writes the raw `n.Label` even for an icon that renders nothing, so the TOC title can disagree with its heading.

<details><summary>cases</summary>

- An unknown name, a missing name or a tag that is not self-closing renders `<h2 id="heading"><!-- gno-icon: unknown name "nope" --></h2>`, and the TOC lists that empty heading as `Top`.
- `label="Q &amp; A"` names the svg `Q & A` but titles the TOC entry `Q &amp; A`, which the TOC template escapes a second time. `&quot;` does the same.
- `## Picks<gno-icon name="star" label="Top" />` is listed as `PicksTop`.

</details>

## gno.land/pkg/gnoweb/markdown/ext_icons.go:312 [gh](https://github.com/gnolang/gno/blob/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown/ext_icons.go#L312) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L312) · Nit

Nit: `nested == 0` keeps icons in an image's alt text, so in `## ![<gno-icon name="star" />](i.png) <gno-icon name="star" />` the unrendered alt icon takes the hint and the nameless heading gets none.

## gno.land/pkg/gnoweb/markdown/ext_icons.go:196 [gh](https://github.com/gnolang/gno/blob/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown/ext_icons.go#L196) · [↗](../../../../../.worktrees/gno-review-6297/gno.land/pkg/gnoweb/markdown/ext_icons.go#L196) · Suggestion

Suggestion: the ID set here drops the labels the TOC title keeps, so a heading listed as `Top` gets a positional `#heading-N` anchor that any icon-only heading above it shifts. I would build the ID from the text the TOC shows, as the `iconHeadingIDTransformer` doc comment says.
