# Claims: gnolang/gno#6297 round 2, 6a68bc69f, claude-opus-5-5, solo review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/utils.go:206 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c1' -v ./gno.land/pkg/gnoweb/markdown | head: `<h2 id="heading"><!-- gno-icon: unknown name "nope" --></h2>` TOC ["heading=Top"], same for missing name and `label="Top">`; 84df2c459's utils.go: TOC [] for all three | tests/judge-solo-toc-label.go | warm |
| 2 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/utils.go:206 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c2' -v ./gno.land/pkg/gnoweb/markdown | head: TOC ["picks=Say &quot;hi&quot; Picks"]; 84df2c459's utils.go: ["picks=Picks"] | tests/judge-solo-toc-label.go | warm |
| 3 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/utils.go:206 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c3' -v ./gno.land/pkg/gnoweb/markdown | head: TOC ["picks=PicksTop"]; 84df2c459's utils.go: ["picks=Picks"] | tests/judge-solo-toc-label.go | warm |
| 4 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/ext_icons.go:312 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c4' -v ./gno.land/pkg/gnoweb/markdown | head and 84df2c459 alike: `<h2 id="ipng"><img src="i.png" alt=""> <svg … aria-hidden="true" …></svg></h2>`, no hint comment; control `## <gno-icon name="star" />` carries it | tests/judge-solo-toc-label.go | hot |
| 5 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/markdown/ext_icons.go:196 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c5' -v ./gno.land/pkg/gnoweb/markdown | head: TOC ["heading=Top"] alone, ["heading-1=Top"] after an unlabeled icon heading; 84df2c459: TOC [] in both cases (no entry, so no link to move) | tests/judge-solo-toc-label.go | hot |
| 1001 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/utils.go:206 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c2' -v ./gno.land/pkg/gnoweb/markdown | head: html `aria-label="Q &amp; A"`, TOC ["heading=Q &amp; A"]; 84df2c459's utils.go: no TOC entry | tests/solo-finder-2-toc-label.go | warm |
| 1002 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/utils.go:206 | cp tests/judge-solo-toc-label.go gno.land/pkg/gnoweb/markdown/zz_judge_toc_test.go && go test -count=1 -run 'TestJudgeSoloToc/c1' -v ./gno.land/pkg/gnoweb/markdown | head: TestSoloFinder2TocLabel/broken x3 FAIL `[Top]`; judge c1 PASS with 84df2c459's utils.go | tests/solo-finder-2-toc-label.go | warm |
| 2001 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/ext_icons.go:196 | go test -count=1 -run TestIconHeadingIDAndToc -v ./gno.land/pkg/gnoweb/markdown | TestIconHeadingIDAndToc PASS at 6a68bc69f with `status=Status done` |  | hot |

Hit rate per tier, from the rows above: hot 3/3 confirmed over 1 files, warm 5/5 confirmed over 5 files, cold 0/0 confirmed over 3 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | gno.land/pkg/gnoweb/markdown/ext_icons.go:192 | dropping the Lines().Len() > 0 guard could renumber the id of a heading another extension builds with no source lines | grep -rn 'NewHeading\\|SetAttributeString("id"\\|SetAttribute([]byte("id")\\|\.IDs()' gno.land/pkg/gnoweb (non-test) returned only ext_icons.go:196: no other code creates a heading or sets an id |
| lines | gno.land/pkg/gnoweb/markdown/ext_icons.go:326 | with the nested counter, `first` could stay nil and `first.hint` panic | aloneInNamedParent climbs `for parent != nil && parent.Type() == ast.TypeInline && !namedByContent(parent)`, so no namedByContent node lies between n and parent and n itself is visited at nested==0; the probe's heading-with-link inputs rendered without panic |
| removed | gno.land/pkg/gnoweb/markdown/icons_gen.go:1 | a regenerated outline glyph paints differently from its source symbol (round-1 section on ext_icons_gen_test.go) | tests/solo-finder-2-paint-parity.sh, resvg render of every source symbol beside its registry glyph: head 6a68bc69f `TOTAL 434 compared, 0 differ at 96px` and `0 differ at 21px`; the round-1 table from 84df2c459 `TOTAL 434 compared, 58 differ at 96px`. The head resolves it. |
| removed | gno.land/pkg/gnoweb/markdown/icons_gen.go:1 | 17 icons regenerated with no render change (box-add, wifi, terminal...) gain a `stroke="none"` group wrapper that a child group overrides | comm of the 75 changed names against the 58 that differed at base: box-add now `<g fill-rule="evenodd" stroke="none"><g stroke="currentColor">`, source and output differ on that group as the ADR says the generator writes; renders identical, not a defect |
| reach | gno.land/pkg/gnoweb/markdown/ext_icons_gen_test.go:357 | TestIconTableMatchesSource skips components/ui/icons.html, so a chrome glyph through the new paint path goes unchecked | symbol roots: `434 <symbol viewBox="0 0 21 21" stroke-width="1.3">`, and `grep -c '<symbol'` gives drawn.svg 6 + vendored.svg 428 = 434, `grep -c 'iconHeadStroke,' icons_gen.go` 434: no icons.html symbol takes the paint path |
| reach | gno.land/pkg/gnoweb/markdown/ext_icons.go:326 | the nested counter could leave `first` nil and panic on `first.hint` | the parent loop at ext_icons.go climbs only inlines that are not namedByContent, so the walk reaches n itself at nested 0; `go test -run 'TestIcon\|TestBuildIcon\|TestParseIcon\|TestToc'` ok, TestIconHintPerParent included |
| removed | gno.land/pkg/gnoweb/markdown/ext_icons.go:192 | the empty-heading ID rewrite drops goldmark's numbering | TestIconHeadingIDEmpty passes at head (`ok ... 4.558s`); the empty heading now enters the fresh linearIDs as goldmark's own generateAutoHeadingID does with a nil line. Round-1 section resolved. |
| removed | gno.land/adr/prxxxx_gnoweb_inline_icons.md:183 | ADR: the generator `removes 48 KB of repeats` | body-only measure (allowlisted source verbatim against the generated body, 434 outline icons): `generated bodies 142285 bytes, verbatim bodies 176278 bytes, removed 33993 bytes`; the ADR does not say what it counts and the only fix is its wording, so no candidate |
| removed | gno.land/adr/prxxxx_gnoweb_inline_icons.md:85 | ADR: each icon writes up to 2.3 KB of glyph, about 5 KB with a label filling the tag bound | render of every registry name: `largest glyph "pure": 2300 bytes unlabeled; tag of 512 bytes with 479 quotes: 5162 bytes`; the claim holds |
| removed | gno.land/pkg/gnoweb/markdown/utils.go:54 | a doc sentence still says <gno-button> exists in this tree | `grep -rn 'gno-button'` over gno.land and docs: utils.go:54 `later ones such as <gno-button />` and the ADR's three `the <gno-button /> PR` lines; none claims it is registered here |

## Completeness

- Sibling of the edited `h.Lines().Len() > 0` guard: `grep -rn 'Lines().At(\|Lines().Len()'` over `gno.land/pkg/gnoweb` non-test code returns only `ext_icons.go:192-193`. No sibling.
- Callers of the changed `writeNodeText`: `grep -rn 'nodeText('` returns only `toc.go:96`, which runs `util.UnescapePunctuations` on it, so the label reaches the TOC title and nothing else.
- Invariant catalog: its classes are realm audit patterns and caller identity predicates for gno code. This diff is Go renderer code in gnoweb, so no class applies.
- Thin angle: the claims angle on the transformer's doc. Filed as reflector candidate 2001 and joined to candidate 5's section, one root: the ID and the TOC title read different text.
- Round-1 sections at 6a68bc69f, each from a run or a read this round:
  - `ext_icons_gen_test.go:203` stroke Warning: resolved. `TestIconTableMatchesSource` passes at head and fails with 84df2c459's `icons_gen.go` swapped in (`fill=currentColor stroke=currentColor width=1.3` against `fill=currentColor stroke=none`). Finder 2's resvg run: 434 compared, 0 differ.
  - `ext_icons.go:190` duplicate-id Warning: resolved. Round 1's `TestB3EmptyHeadingDuplicateID` and the PR's `TestIconHeadingIDEmpty` pass at head.
  - `ext_icons.go:298` link hint Nit: resolved. Round 1's `TestB3LinkHintLostInsideHeading` and `TestIconHintPerParent` pass at head.
  - `ext_icons_test.go:170` fuzz Test: resolved. The `FuzzIconRender` seeds pass, with `label="turn on"` at line 152.
  - `nested_emphasis.md.txtar:7` Test: resolved. The file is deleted.
  - ADR `:83` size Nit: resolved. Finder 2 measured the largest glyph at 2300 bytes and 5162 bytes with a 512-byte label.
  - ADR `:100` and `:104` Nits: resolved. Line 105 reads `(the <gno-button /> PR) can register it`, and lines 225-227 call `sanitize.Block` unchanged.
  - Body TOC Suggestion: resolved. The label reaches the TOC (`TestIconHeadingIDAndToc`, `heading-1=Top`). The new path raises the `utils.go:206` and `ext_icons.go:196` sections.
  - Carried open, declined for a follow-up, with the lines unchanged in the diff: `ext_icons.go:105` (`TrimSpace` on the raw label), `ext_icons.go:236` (`name=star/>`), `iconset/main.go:122` (`http.Get` has no timeout; the file is not in the diff), `ext_icons.go:182` (`newLinearIDs()` replaces a caller's generator).
- Merge base: every TOC-title candidate (1, 1002, 1001, 2, 3) was run against 84df2c459's `utils.go`, where each heading has no TOC entry or drops the label, so the fix commits cause them. Candidate 4 behaves the same at 84df2c459, which is round 1's code. The fix commits rewrote that selection at `:312` and missed the image case.
