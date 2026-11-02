# Findings in posting order, from round assemble: 2 to post, 1 SKIP, 1 refuted kept out

## gno.land/pkg/gnoweb/markdown/ext_math.go:362 [gh](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/gno.land/pkg/gnoweb/markdown/ext_math.go#L362) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: a cache hit returns sc.open, the backtick run an earlier opener's search left open, so a later $ opener whose own search sees no open run is told it starts in a code span, and its formula on the next line stays text
Check: go test ./gno.land/pkg/gnoweb/markdown -run TestZZCachedOpenRun -v at head and base, then again at head with findCloseCached bypassed (return find(b, ticks, nil) after its len check)
Details: In `$a \\(`\\) $b c⏎d$ `e`, the $a search counts the backtick inside \\(`\\) as an unclosed run and caches open=1. goldmark then parses \\(`\\) as AMS math and asks for $b; findCloseCached answers from the $a entry at ext_math.go:362 with sc.open, so the next-line search starts inside a 1-backtick code span, skips d$ up to `e, and $b c⏎d$ stays text. A fresh search for $b starts after \\(`\\) and closes at d$. Bypassing the cache flips withA from 1 <math> to 2, so the cache alone changes the output, against its doc comment's claim that the reused answer is the same. A raw HTML attribute holding a backtick (`$a <x y="`"> $b c⏎d$ `e`) fails the same way. At the merge base withA also counts 1 <math>, but that is $a pairing with d$, as base carries no run state. The failure only leaves math as text.
Evidence: head: 'alone: 2 math, withA: 1 math', --- FAIL on withA and raw HTML. head with the cache bypassed: 'alone: 2 math, withA: 2 math', --- PASS. base: 'alone: 2 math, withA: 1 math', the math being `a \\(`\\) $b c⏎d`.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6326-math-sanitized-code-spans/2-f6a568ed8/tests/solo-finder-2-cached-open-run_test.go

## gno.land/pkg/gnoweb/markdown/ext_math.go:542 [gh](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/gno.land/pkg/gnoweb/markdown/ext_math.go#L542) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: parenEnd counts the parentheses inside a quoted link title or a <...> destination, so a $$, \\( or \\[ closer after such a link is never found and a formula the merge base rendered stays text, or pairs with a later closer
Check: cp tests/solo-finder-1-dest-hides-closer_test.go gno.land/pkg/gnoweb/markdown/zz_dest_hides_closer_test.go && go test ./gno.land/pkg/gnoweb/markdown -run TestZZDestHidesCloser -v at f6a568ed8 and at 1cae57ed7
Details: linkEnd hands a title opening with " or ' to parenEnd(b, j, depth) at ext_math.go:542, and a <...> destination to parenEnd at ext_math.go:517 and :526; parenEnd counts every unescaped ( and ) to the line end without knowing about quotes or <>. In `$$ [a](x "(") b $$` the ( in the title raises depth to 2, the ) after the title brings it back to 1, and the search runs to the line end, so the $$ after b is skipped. goldmark itself reads the link as closed (title "("). An unclosed < (`\\( [a](<x b \\)`, no link for goldmark) is skipped the same way. On a two-line input the hidden closer pairs with the next line's $$, turning the text between into TeX, and a $$ line further down opens a display block that swallows a paragraph. The merge base found the closer in all six single-line shapes.
Evidence: head f6a568ed8: math=0 for `$$ [a](x "(") b $$`, `\\( [a](x "(") b \\)`, `\\[ [a](x "(") b \\]`, `\\( [a](<x(>) b \\)`, `\\( [a](<x b \\)`, `$$ [a](x 'it(s') b $$`; base 1cae57ed7: math=1 for each. PairsLater at head: annotation ` [a](x &#34;(&#34;) b $$ then\nsee `; at base ` [a](x &#34;(&#34;) b `. Third shape at head: <math display="block"> outside <p>, annotation ending `some text\n`; at base inline math then `some text $$` as paragraph text.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6326-math-sanitized-code-spans/2-f6a568ed8/tests/solo-finder-1-dest-hides-closer_test.go
State: CONFIRMED, band: Nit, angle: claims
TL;DR: a quoted title holding ) ends parenEnd inside the title, so a $$ in that title closes the expression and the link is cut, at base and head alike
Check: TestZZDestHidesCloserPairsLater first case in tests/solo-finder-1-dest-hides-closer_test.go at head and base
Details: `$$ [a](x "t) $$ q") c $$`: goldmark reads <a href="x" title="t) $$ q">; gnoweb renders math ` [a](x "t) ` and no link at both commits. Pre-existing, in scope because the diff adds 'Still whole: a link with a title' to TestMathClosesInBracketsThatAreNoLink, which holds only for titles without parentheses. One edit to the title branch at ext_math.go:542 closes this and candidate 1's title shapes.
Evidence: head and base: math=1 links=0, annotation ` [a](x &#34;t) `; plain goldmark: <a href="x" title="t) $$ q">a</a>
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6326-math-sanitized-code-spans/2-f6a568ed8/tests/solo-finder-1-dest-hides-closer_test.go

## SKIP gnovm/stdlibs/chain/markdown/markdown.go:86 [gh](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/gnovm/stdlibs/chain/markdown/markdown.go#L86) · Suggestion
State: PLAUSIBLE, band: Suggestion, angle: catalog
TL;DR: text a realm sanitized and stored before the upgrade keeps a bare $, so the fix reaches only values written after it
Check: store a treasury request reason holding $ with the merge-base binary, upgrade the node in place to head, render r/gov/dao: expect the stored $ to still open math
Details: Catalog class 'Data an older binary wrote'. prop_requests.gno:248 escapes the reason once at creation and render.gno emits it raw, so a proposal stored by an older binary still opens math on render. Whether any chain carries such state across an in-place upgrade, rather than a genesis replay that re-runs the escape, is not settled here.
Evidence: read: prop_requests.gno:244-249, the comment 'render.gno emits raw'
