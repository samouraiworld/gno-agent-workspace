# Claims: gnolang/gno#5585 round 1, 2eff8be62, claude-opus-5-5, solo review

Round shape: solo round on the reviewer's own pull request, 1 finder and one judge, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/ext_heading.go:34 | sed -n '/-- output.html --/,$p' gno.land/pkg/gnoweb/markdown/golden/ext_frame/heading_inline_tag.md.txtar: expect the h2 to hold only the omitted-HTML comment and the heading-anchor | head: "## </gno-frame>" html=<h2 id="gno-frame"><!-- raw HTML omitted --><a href="#gno-frame" class="heading-anchor" ...></a></h2> toc=[]; "## <gno-icon name=\"nope\" />" html=<h2 id="heading"><!-- gno-icon: unknown name "nope" --><a href="#heading" class="heading-anchor" ...></a></h2> toc=[]; "## <span></span>" same shape, toc=[]; "##" html=<h2 id="heading"></h2>; "## Title" toc=["Title"]. nodeText guard: judge test PASS, TestGnoExtension/ext_icons/valid_setext_heading.md FAIL (anchor dropped after an unlabelled icon). | tests/judge-1-lone-anchor_test.go | warm |
| 2 | REFUTED | Missing test | gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go:472 | render through renderMarkdown at head and check the output still contains `<h1>Welcome to MyRealm</h1>` | sanitize_integration_test.go:342 `m := goldmark.New(goldmark.WithExtensions(...))` with no parser options | tests/judge-1-lone-anchor_test.go |  |
| 3 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/ext_heading.go:49 | read ext_heading.go:46-55 | ext_heading.go:46 `type headingExtension struct{}` |  | warm |
| 4 | REFUTED | Nit | gno.land/pkg/gnoweb/markdown/ext_heading.go:36 | goldmark renderer.go Flush return | goldmark renderer/renderer.go:173 `return writer.Flush()` |  | warm |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 0 files, warm 1/3 confirmed over 38 files, cold 0/0 confirmed over 5 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| reach | gno.land/pkg/gnoweb/frontend/js/controller-observer.ts:43 | scroll-spy querySelectorAll('a[href^="#"]') could pick up every new heading anchor | grep -rn 'data-controller="[^"]*observer' components: only components/views/overview.html:101 <nav class="b-pkg-subheader" ... data-controller="observer">, which holds no rendered markdown |
| lines | gno.land/pkg/gnoweb/frontend/css/05-composition.css:670 | focus outline token undefined would drop the focus ring (the maintainer's earlier --s-color-border-brand finding) | grep -- '--s-color-text-link:' 02-tools.css:88 and :219 defined; --g-space-1 at 01-settings.css:173; --g-border-radius-sm at 01-settings.css:251; compiled main.css carries outline:2px solid var(--s-color-text-link) |
| reach | gno.land/pkg/gnoweb/markdown/ext.go:76 | anchors could land in a container the new CSS does not scope, showing always-visible full-size icons | every RenderRealm caller (handler_http.go:471, 550, 832, 992) renders into c-realm-view (view_realm.go:54, user.html:56) or c-readme-view (overview.html:151, directory.html:50, view_source.go:41); doc view uses documentationGM without GnoExtension (render.go:101-108) |
| reach | gno.land/pkg/gnoweb/markdown/ext_heading.go:38 | #ico-link sprite could be absent on some pages, leaving an empty svg | components/layouts/index.html:14 {{ template "ui/icons" -}}; components/ui/icons.html:123 <symbol id="ico-link"> |
| lines | gno.land/pkg/gnoweb/markdown/ext_heading.go:23 | "0123456"[n.Level] could index out of range for a synthetic heading | grep 'NewHeading' over gno.land/ non-test Go: no hit; headings come only from goldmark ATX/setext parsers (levels 1-6), same index goldmark's own renderHeading uses |
| removed | gno.land/pkg/gnoweb/markdown/ext_foreign.go:438 | sandboxed <gno-foreign> headings could miss or duplicate anchors | ext_foreign.go:435-454 inner instance enables no auto heading IDs and does not load ExtHeading, so foreign headings carry neither id nor anchor |
| removed | gno.land/pkg/gnoweb/public/main.css:1 | maintainer's review flagged a main.css conflict with master | gh pr view 5585 --json mergeable: MERGEABLE; gh api compare 02606904e...master: ahead_by 0 |

## Completeness

- Catalog: Determinism, Gas, Realm state, Caller, Coin, Storage deposit, VM-fault, VM semantics and Type-check do not apply to a gnoweb renderer and CSS change. Global mutable state is row 3 and Error handling row 4, both refuted.
- Sibling sweep by shape: `min(len(svg)` and `<use href=\"#ico-` over every `*_test.go` hit only `ext_icons_test.go:171`, the edited line. The bare `<hN>text</hN>` shape over `*_test.go` and `*.txtar` outside the markdown goldens hit only `sanitize_integration_test.go`, row 2, refuted by a run: its renderer enables no auto heading IDs.
- Thin angle: lines returned one candidate; the judge's run widened row 1 to `## <span></span>` and an unknown `gno-icon` name, and tried the ToC's `nodeText` as the guard, which breaks `ext_icons/valid_setext_heading.md`.

## Retro

- What failed: nothing died and no stage moved model. The solo judge ran neither `round check` nor `prose-check` over the overview, since the own-PR prompt never asked; the parent ran both, clean.
- What worked: one finder at 35 turns produced one candidate. The reflector's three candidates were all refuted, one by a run and two by reading. The judge widened the one CONFIRMED Nit from one heading shape to three and measured the obvious fix against the goldens.
- Hit rate per tier: warm 1/3 confirmed over 38 files; hot and cold none run.
- Cost: 2 agents, 57 turns, 73k output, 7.6M cache read, 0.28M cache write, about $5.21 per `./scripts/review-retro.py`.
- Upgrade: the own-PR solo prompt in `scripts/workflows/review-pipeline.js` now ends on `round check` and `prose-check` over the overview, the same gates the writer path runs. Estimate: +2 judge turns at ~113k context, about +0.23M cache read; the parent runs 2 fewer calls. No change to the finding rate.
