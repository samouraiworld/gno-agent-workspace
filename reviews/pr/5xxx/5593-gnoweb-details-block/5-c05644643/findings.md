# Findings in posting order, from round assemble: 4 to post, 0 SKIP, 3 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_details.go:207 [gh](https://github.com/gnolang/gno/blob/c05644643affeec6d6f0c0f1b76b41f407caba5a/gno.land/pkg/gnoweb/markdown/ext_details.go#L207) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: a realm page with a deep closed blockquote inside a :::details takes 44x the plain parse at 120 KB, growing quadratically
Check: go test ./gno.land/pkg/gnoweb/markdown -run 'TestZZWalkCost|TestZZJudgeWalkGrowth' -v, at c05644643 and with ext_details.go from 8ab095f49
Details: Continue walks node.LastChild() to the bottom of the tree on every line, closed nodes included (ext_details.go:207-211). A blockquote D deep closes on the first blank line, but each of the N following lines still walks D nodes. Three sizes, D and N doubling: details/plain ratio 21.7, 36.6, 44.3; details time multiplies about 4.3x per doubling. At 8ab095f49, without the loop, the ratio is 1.0 to 1.1. Projected from the curve (inference, not run): past the ~8 s the maxMarkdownRenderBytes comment budgets at roughly 140 KB, and minutes at the 1 MiB cap. Fix measured: 'if _, ok := pc.LastOpenedBlock().Node.(*ForeignNode); ok' brings the ratio back to 1.0-1.1 and every golden stays green, since an open ForeignNode opens with NoChildren and so is always the last opened block. Mechanical.
Evidence: c05644643: 'bytes=30001 D=5000 N=25000 plain=12.06ms details=262.17ms ratio=21.7', 'bytes=60001 D=10000 plain=40.06ms details=1.467s ratio=36.6', 'bytes=120001 D=20000 plain=140.9ms details=6.245s ratio=44.3'; finder's TestZZWalkCost 'plain=317.5ms details=6.34s' FAIL. ext_details.go at 8ab095f49: ratios 1.1, 1.1, 1.0, PASS. With tests/judge-1-lastopened.patch: ratios 1.0, 1.1, 1.1, package goldens PASS.
Artifact: tests/judge-1-walk-growth_test.go

## gno.land/pkg/gnoweb/markdown/ext_details.go:208 [gh](https://github.com/gnolang/gno/blob/c05644643affeec6d6f0c0f1b76b41f407caba5a/gno.land/pkg/gnoweb/markdown/ext_details.go#L208) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: an unclosed <gno-foreign> inside a blockquote or list item keeps the :::details from closing, so the rest of the page folds into it
Check: go test ./gno.land/pkg/gnoweb/markdown -run 'TestZZStaleForeign|TestZZJudgeStaleForeign' -v, at c05644643 and with ext_details.go from 8ab095f49
Details: foreignParser.Close (ext_foreign.go:270-279) never sets Closed; only Continue's EOF and close-tag paths and the post-parse transformer do. So once a blockquote or list closes an unclosed foreign, the stale node stays on the chain ext_details.go:207-211 walks, and the ':::' closer is taken as text. In quote_noblank and list_blank the foreign is still open when the details' Continue sees ':::', because goldmark runs the outer Continue before the inner container closes. Three inputs fail at the head and pass at 8ab095f49; a closed foreign in a blockquote passes at both. The LastOpenedBlock form fixes quote_blank only. Needs malformed realm markdown (an unclosed foreign inside a container). Fix is a decision for the author: the exemption has to tell a foreign that will survive this line from one its container is about to close.
Evidence: c05644643: quote_blank, quote_noblank, list_blank all end '<p>footer</p>\n</details>'; closed_control PASS. ext_details.go at 8ab095f49: all four PASS. LastOpenedBlock patch: quote_blank PASS, quote_noblank FAIL, list_blank FAIL.
Artifact: tests/judge-2-stale-foreign_test.go

## gno.land/pkg/gnoweb/markdown/ext_details_test.go:1 [gh](https://github.com/gnolang/gno/blob/c05644643affeec6d6f0c0f1b76b41f407caba5a/gno.land/pkg/gnoweb/markdown/ext_details_test.go#L1) · Missing test
State: CONFIRMED, band: Missing test, angle: catalog
TL;DR: dropping the four-column guard on the closer leaves every test green
Check: replace 'w <= 3 && isCloseFence' with 'true && w >= 0 && isCloseFence' at ext_details.go:212 in a scratch tree, go test ./gno.land/pkg/gnoweb/markdown
Details: At the head ':::details A\nbody\n\n    :::\nafter\n:::\n' renders the indented line as '<pre><code>:::' and 'after' inside the fold, so the guard works; no golden or test pins it. Fix: that input as a golden under golden/ext_details.
Evidence: mutant: 'ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 5.332s'. Head probe output: '<p>body</p>\n<pre><code>:::\n</code></pre>\n<p>after</p>'.

## gno.land/pkg/gnoweb/markdown/ext_details.go:165 [gh](https://github.com/gnolang/gno/blob/c05644643affeec6d6f0c0f1b76b41f407caba5a/gno.land/pkg/gnoweb/markdown/ext_details.go#L165) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the w > 3 branch in Open cannot run
Check: sed -n 960,992p goldmark@v1.8.2/parser/parser.go; plant panic("reached") in the w > 3 branch, go test ./gno.land/pkg/gnoweb/markdown plus seven indented-opener probes
Details: goldmark's openBlocks computes 'w, pos := util.IndentWidth(line, reader.LineOffset())' on the same reader state and skips the parser with 'if w > 3 && !bp.CanAcceptIndentedLine() { continue }' (parser.go:962, 989); detailsParser.CanAcceptIndentedLine returns false and no wrapper in the package calls detailsParser.Open. Remove ext_details.go:164-167's branch and its comment; 'pos' can come from pc.BlockOffset(), which goldmark set to the same value at parser.go:968. Mechanical.
Evidence: planted panic: 'ok github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown 5.950s' with probes '    :::details A', tab, list-nested, blockquote-tab, space-tab, nested-indented and after-paragraph openers.
