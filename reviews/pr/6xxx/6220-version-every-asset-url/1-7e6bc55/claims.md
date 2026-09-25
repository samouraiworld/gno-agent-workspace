# Claims — PR 6220, round 1 at 7e6bc551447b80122b90e3b66f172e2c84bf0414

Round: 1. No workflow and no finder stage. One parent pass found, ran and judged, with `gnoweb` built from source at the head and at the merge base `877379432e0b3c9e084bdd19dac3a636ffb43d80` and both booted. The shape was chosen against the target rather than taken from the preset table: the diff is 42 lines over two files, which one read holds. Projections for the shapes not run, from `./scripts/review-plan.py` over this diff's bundles: `quick` 9 agents and about $17, `review` 13 agents and about $43, `trivial` 2 agents and about $4.

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | gno.land/pkg/gnoweb/components/layouts/head.html:9-10 | boot gnoweb from source at the head, read the preload href off `/` and the `@font-face` url off `/public/main.css` | preload `Intervar.woff2?v=20260920235923`, stylesheet `Intervar.woff2`; the same script passes at the merge base | [tests/preload-url-mismatch.sh](tests/preload-url-mismatch.sh) | warm |
| 2 | CONFIRMED | Warning | gno.land/pkg/gnoweb/components/layouts/head.html:9-10 | load the page in Chromium at both servers and read the font requests | head makes 3 font requests and the merge base 2; the extra is `Intervar.woff2` with no query, fetched after the versioned preload went unclaimed | browser network log, quoted in the draft | warm |
| 3 | REFUTED | Nit | gno.land/pkg/gnoweb/components/layouts/head.html:35 | does the asset handler reject a query string on the favicon and the chroma stylesheet | `favicon.ico?v=…` 200 `image/vnd.microsoft.icon` 7406 bytes, `_chroma/style.css?v=…` 200 `text/css` 13846 bytes | | warm |
| 4 | REFUTED | Nit | gno.land/pkg/gnoweb/components/layouts/head.html:35 | does dropping the leading slash break the favicon URL | `assetsBase` is built as `"/" + strings.Trim(cfg.AssetsPath, "/") + "/"` at `app.go:104`, so it always ends in a slash and `/public/favicon.ico` resolves | | warm |
| 5 | REFUTED | Warning | gno.land/pkg/gnoweb/components/layouts/ | an asset URL in another template the change missed | every `AssetsPath` and `ChromaPath` reference in `components/**/*.html` carries `?v=`: `head.html` 5, `footer.html` 2, `analytics.html` 2 | | warm |
| 6 | REFUTED | Nit | gno.land/pkg/gnoweb/components/layout_test.go:606 | does the suite go red | `go test ./gno.land/pkg/gnoweb/...` rc 0 over five packages, `components` 0.028s | | cold |

Candidates 1 and 2 are two measurements of one defect and ship as one section.

## Completeness

- **Every changed line read?** Yes. Two files, 38 added and 4 deleted; `head.html` read whole at both ends, `layout_test.go`'s added block read whole.
- **Blast radius mapped?** The head template's five asset URLs, their handlers, and the one other place the same files are named. `AssetsPath`, `ChromaPath` and `BuildTime` are set at `app.go:104`, `:126` and `:139` and reach the head through `handler_http.go:213-217`. The static handler serves the embedded `public/` tree and ignores the query, measured on the favicon and the chroma stylesheet. The second naming of the fonts is `frontend/css/04-elements.css:12-23`, built into `public/main.css`, which is candidate 1.
- **The catalog?** `skills/invariant-catalog.md` was not walked: its classes cover the GnoVM, stdlibs and realm code, and this diff touches an HTML template and a Go test in gnoweb. The delta's own rule skips the catalog for docs- and tooling-only PRs; this is the same case one step over.
- **CI?** Every check run at the head is success or skipped. The combined commit status reads `failure` only because the `Gno2D2` bot holds out for a gnoweb codeowner, `alexiscolin` or `gfanton`. No failing job to reproduce.
- **Danger pass?** Not triggered. `author_association` is `MEMBER`.
- **What was not run.** The edge cache the description names is not reachable from here, so the claim that a TTL past an hour is unsafe while a URL stays unversioned rests on the description alone. Nothing in the finding depends on it.

## Retro

No workflow ran, so `./scripts/review-retro.py` has no directory to read and there are no agent rows to add.

- What failed: the plan's own sizing. `./scripts/review-plan.py --preset quick` projected 4 verifier agents for a diff whose two finders can return 6 candidates at most, because `project()` sizes candidates off the angle count rather than the finder-job count. And `round dispatch` cut this 42-line change into 2 bundles, putting `head.html` in one and the test asserting it in the other, since its merge predicate matches siblings only and never a child against its own parent. Both are lines of the workspace `TODO.md`.
- What worked: booting both ends from source. The template diff reads as obviously correct, and only the pair of running servers showed that the preload and the stylesheet had stopped naming the same URL. The browser's own network log settled it in one load.
- Hit rate per tier: warm 2 of 5 confirmed, cold 0 of 1. One tier above the other, which is the direction `round risk` intends, on a sample too small to weigh.
- One upgrade: size the agent count from what one agent can hold before the preset's table is read. Landed this round as a rule in `skills/review.md` *Launch*, with the estimate beside it: on this target 2 agents against 9 and about $4 against $17, output 64k against 234k, cache read 2M against 12M, 8 minutes against 32, finding rate unchanged at one Warning. Estimate until an outcome table measures it.

## Outcomes

| Section | Band | Outcome | Replies | Author replied |
| --- | --- | --- | --- | --- |
| gno.land/pkg/gnoweb/components/layouts/head.html:9-10 | Warning | fixed | 1 | yes |
