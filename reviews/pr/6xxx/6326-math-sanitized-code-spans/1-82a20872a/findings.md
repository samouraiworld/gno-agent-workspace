# Findings in posting order, from round assemble: 6 to post, 0 SKIP, 2 refuted kept out

## Body

- Warning: gno.land/pkg/gnoweb/markdown/ext_math.go:201 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L201), outside the diff, so no anchor
  State: CONFIRMED, band: Warning, angle: claims
  TL;DR: After findClose stops a single-$ expression at its label's '](', Parse still searches the next line, so '[Send $10](...)' followed by a line holding '100$' becomes one formula and the link is lost; predates the diff
  Check: cd gno.land && go test ./pkg/gnoweb/markdown -run TestZZLabelStopNextLine -count=1 -v at head and base
  Details: findCloseCached returns only -1, the same for 'no closer on this line' and 'stopped at a point no expression crosses', so Parse's else branch runs the line-2 search from fresh state. The diff rewrites this scan and documents stopAtLabel as ending the expression. Same output at the merge base.
  Evidence: head and base: '--- PASS: TestZZLabelStopNextLine/one_line', '--- FAIL: TestZZLabelStopNextLine/two_lines'; probe TeX '10](/r/demo/bank$help&amp;func=Send)\nBalance: 100' at both
  Artifact: tests/solo-finder-1-label-stop-next-line_test.go
- Nit: gno.land/pkg/gnoweb/markdown/ext_math.go:204 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L204), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: claims
  TL;DR: The next-line search starts findClose outside any code span, so a code span crossing the paragraph's line break is still cut by a closer inside it
  Check: render '\\(a `b\nc\\)d` e\\)' at head and base
  Details: Same output at the merge base; the ADR's 'keeps a code span whole' holds on one line only.
  Evidence: judge probe head and base: TeX 'a `b\nc'
  Artifact: tests/judge-1-probe_test.go
- Nit: gno.land/pkg/gnoweb/markdown/ext_math.go:491 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L491), outside the diff, so no anchor
  State: CONFIRMED, band: Nit, angle: lines
  TL;DR: The block opener's bytes.Contains sees a $$ inside a code span and declines, while the inline scan skips that code span, so the display block is opened by neither
  Check: render '$$ a `$$` b\nc\n$$' at head and base
  Details: Head renders '<p>$$ a <code>$$</code> b\nc\n$$</p>'. Base rendered a formula ' a `' that cut the code span. Not worse than the base; a sibling of the closer-search change left on bytes.Contains.
  Evidence: judge probe head 'OUT <p>$$ a <code>$$</code> b'; base 'TEX: a `'
  Artifact: tests/judge-1-probe_test.go

## gno.land/pkg/gnoweb/markdown/ext_math.go:359 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L359) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: findDelimClose hands $$, \\( and \\[ a destination skip that ends only at a balanced ')', so a formula holding an unclosed '[..](' runs past its closer and swallows the links after it
Check: cp tests/solo-finder-1-unclosed-dest-swallows-closer_test.go gno.land/pkg/gnoweb/markdown/ && cd gno.land && go test ./pkg/gnoweb/markdown -run TestZZUnclosedDest -count=1 -v; base via go test -overlay
Details: At the merge base bytes.Index closed each at its first closer. At head, after '](' with a label open, findClose sets depth=1 and tests no closer until a ')' balances it: in '\\( [a,b](x \\) and [l](/r/y) then \\(z\\)' the ')' of the closer \\) ends the destination, so the formula runs to the last \\( and the link /r/y is gone; in '$$[0,1](x $$ and [l](/r/y) z' + newline + 'w $$' line 1 has no closer and the formula takes line 2's. A destination with a space or a line break is not a CommonMark destination, so the skip covers spans goldmark never parses as a link. The single-$ closer shares the skip at both trees.
Evidence: head: '--- FAIL: TestZZUnclosedDest' all three subtests; base (-overlay): '--- PASS: TestZZUnclosedDest' all three. Judge probe head TeX ' [a,b](x \\) and [l](/r/y) then \\(z'; base renders two formulas and <a href="/r/y">
Artifact: tests/solo-finder-1-unclosed-dest-swallows-closer_test.go

## gnovm/stdlibs/chain/markdown/markdown_test.go:43 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gnovm/stdlibs/chain/markdown/markdown_test.go#L43) · Missing test
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: No golden/sanitize case sends a '$' through InlineText, TableCell or LinkTitle and renders it with gnoweb, so only the native's bytes pin the escape
Check: grep -l 'MARKDOWNFUNC: \(InlineText\|TableCell\|LinkTitle\)' gno.land/pkg/gnoweb/markdown/golden/sanitize/*.txtar | xargs grep -l '\$'
Details: TestEscapeInline asserts '\$x\$'; nothing asserts that gnoweb renders the escaped dollar as text. The grep over 258 golden/sanitize files returns no file.
Evidence: grep returned no file

## gnovm/stdlibs/chain/markdown/markdown.go:92 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gnovm/stdlibs/chain/markdown/markdown.go#L92) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: titleEscapeSet['$'] = false rests on gnoweb showing a title's backslashes, but gnoweb unescapes '\$' in a title to '$'
Check: render [a](/r/x "Price $5 v1\.0 \$6") with gnoweb at head
Details: The comment at markdown.go:90-91 says 'Price \$5' would read as such. goldmark applies backslash escapes in a link title, as it does for '\.' already in the set.
Evidence: head: SRC '[a](/r/x "Price $5 v1\.0 \$6")' HTML 'title="Price $5 v1.0 $6"' (TestProbeEscapedDollar)
Artifact: tests/solo-finder-2-title-unescape_test.go
