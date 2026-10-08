# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 0 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_icons.go:196 [gh](https://github.com/gnolang/gno/blob/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown/ext_icons.go#L196) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: the doc says the ID comes from the text the TOC shows; `## Status <icon label=done/>` is listed as `Status done` and links #status
Check: go test -count=1 -run TestIconHeadingIDAndToc -v ./gno.land/pkg/gnoweb/markdown
Details: The PR's own TestIconHeadingIDAndToc asserts id `status` and TOC title `Status done` for that heading, and passes at head, so the doc at ext_icons.go:170-172 no longer describes the code. One root with candidate 5: the ID and the TOC title read different text.
Evidence: TestIconHeadingIDAndToc PASS at 6a68bc69f with `status=Status done`
State: CONFIRMED, band: Suggestion, angle: claims
TL;DR: a label-only heading is listed by its label but keeps the positional anchor heading-N, which shifts when an unlabeled icon heading is added above it
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c5' -v ./gno.land/pkg/gnoweb/markdown
Details: withoutIcons (ext_icons.go:194) drops every icon tag, labels included, before ids.Generate at :196, while writeNodeText now writes the label into the TOC title. `## <gno-icon name="star" label="Top" />` alone gets heading=Top; with `## <gno-icon name="heart" />` above it, it gets heading-1=Top, so a shared #heading link lands on the heart heading.
Evidence: head: TOC ["heading=Top"] alone, ["heading-1=Top"] after an unlabeled icon heading; 84df2c459: TOC [] in both cases (no entry, so no link to move)
Artifact: tests/judge-solo-toc-label.go

## gno.land/pkg/gnoweb/markdown/ext_icons.go:312 [gh](https://github.com/gnolang/gno/blob/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown/ext_icons.go#L312) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: an icon inside an image's alt text takes `first`, so the heading's add-label hint goes to an icon that never renders
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c4' -v ./gno.land/pkg/gnoweb/markdown
Details: The image is not namedByContent, so nested stays 0 and the alt-text icon becomes first with hint=true; the image renderer writes alt="" and skips its children, and the visible icon returns its memoised hint=false. The heading renders an empty alt and one aria-hidden svg, with no hint. The nested==0 selection at :312 skips icons under a link or heading but not under an image.
Evidence: head and 84df2c459 alike: `<h2 id="ipng"><img src="i.png" alt=""> <svg … aria-hidden="true" …></svg></h2>`, no hint comment; control `## <gno-icon name="star" />` carries it
Artifact: tests/judge-solo-toc-label.go

## gno.land/pkg/gnoweb/markdown/utils.go:206 [gh](https://github.com/gnolang/gno/blob/6a68bc69fc638c389d2308743e2fb1d11cc326bc/gno.land/pkg/gnoweb/markdown/utils.go#L206) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: a labeled icon that renders only a comment titles its empty heading in the TOC
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c1' -v ./gno.land/pkg/gnoweb/markdown
Details: writeNodeText's *Icon case writes n.Label without the checks renderIcon applies (SelfClosing, a name, a registry hit), so an unknown, missing or non-self-closing icon leaves an <h2> holding only an HTML comment while the TOC lists the label.
Evidence: head: `<h2 id="heading"><!-- gno-icon: unknown name "nope" --></h2>` TOC ["heading=Top"], same for missing name and `label="Top">`; 84df2c459's utils.go: TOC [] for all three
Artifact: tests/judge-solo-toc-label.go
State: CONFIRMED, band: Nit, angle: reach
TL;DR: same defect as candidate 1: a labeled icon that renders nothing still titles its heading
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c1' -v ./gno.land/pkg/gnoweb/markdown
Details: Rerun of the finder's tests/solo-finder-2-toc-label.go broken subtests at head: three `Should be empty, but was [Top]`; the judge's c1 subtest reproduces it and passes with the base utils.go.
Evidence: head: TestSoloFinder2TocLabel/broken x3 FAIL `[Top]`; judge c1 PASS with 84df2c459's utils.go
Artifact: tests/solo-finder-2-toc-label.go
State: CONFIRMED, band: Nit, angle: reach
TL;DR: the TOC title takes the label raw, so `label="Q &amp; A"` shows `Q &amp; A` while the svg is named `Q & A`
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c2' -v ./gno.land/pkg/gnoweb/markdown
Details: renderIcon resolves character references through gmhtml.DefaultWriter.Write; writeNodeText writes n.Label raw and toc.go:96 only runs util.UnescapePunctuations, which handles backslash escapes and not references, so the title keeps `&amp;` and the template escapes it again.
Evidence: head: html `aria-label="Q &amp; A"`, TOC ["heading=Q &amp; A"]; 84df2c459's utils.go: no TOC entry
Artifact: tests/solo-finder-2-toc-label.go
State: CONFIRMED, band: Nit, angle: lines
TL;DR: same defect as candidate 1001, with `&quot;`, the only way to put a quote in a double-quoted label
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c2' -v ./gno.land/pkg/gnoweb/markdown
Details: `## <gno-icon name="star" label="Say &quot;hi&quot;" /> Picks` titles the TOC entry with the raw references.
Evidence: head: TOC ["picks=Say &quot;hi&quot; Picks"]; 84df2c459's utils.go: ["picks=Picks"]
Artifact: tests/judge-solo-toc-label.go
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the label is glued to the heading text beside it: `## Picks<gno-icon … label="Top" />` is listed as `PicksTop`
Check: cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c3' -v ./gno.land/pkg/gnoweb/markdown
Details: writeNodeText writes n.Label with no separator, so an icon written against a word, before or after it, joins the label to that word in the TOC title.
Evidence: head: TOC ["picks=PicksTop"]; 84df2c459's utils.go: ["picks=Picks"]
Artifact: tests/judge-solo-toc-label.go
