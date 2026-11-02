# Review: [#6326](https://github.com/gnolang/gno/pull/6326)
Posted: https://github.com/gnolang/gno/pull/6326#pullrequestreview-5738623515
Event: COMMENT
Verdict: REQUEST CHANGES. The `$$`, `\\(` and `\\[` closers are missed after a link whose title or `<…>` destination holds a parenthesis, and a cached search hands a `$` opener a backtick run from another construct; the branch introduces both, so both block.
Model: claude-opus-5-5, standard review; finders and judge high
Commit: f6a568ed8
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/alexiscolin/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233) · [vscode.dev](https://vscode.dev/github/alexiscolin/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6326 f6a568ed8`
Round: 2. The head moved from 82a20872a to f6a568ed8: cf44ce7f0, 25fd73c13 and 797a73858 rewrite the closer search, 21f72443c the ADR, f6a568ed8 a test, plus two master merges with no conflict hunk. The author answered round 1's Warning and its three Body bullets in cf44ce7f0 and 25fd73c13; its link-title `SKIP` stays skipped. Two finders, then one judge and writer answering the completeness question, five candidates, one of them its own, each run or read from scratch by an agent that was not its finder; the Nit sharing the Warning's line was probed in the same run. thehowl's three findings at this head were not candidates.

## Body

> AI review, claude-opus-5-5, standard review, reviewed at [f6a568ed8](https://github.com/gnolang/gno/pull/6326/commits/f6a568ed830794fafe1ca4840786c30d4dc0c233), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6326-math-sanitized-code-spans/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6326-math-sanitized-code-spans/2-f6a568ed8/claims.md) · Status: REQUEST CHANGES

## gno.land/pkg/gnoweb/markdown/ext_math.go:362 [gh](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/gno.land/pkg/gnoweb/markdown/ext_math.go#L362) · [↗](../../../../../.worktrees/gno-review-6326/gno.land/pkg/gnoweb/markdown/ext_math.go#L362) · Warning [posted](https://github.com/gnolang/gno/pull/6326#discussion_r4443978295)

The cache hit returns the earlier search's `sc.open` to a later `$` opener, so `$b c` closed by `d$` on the next line stays text once `` $a \\(`\\) `` precedes it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6326 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_cache_run_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestZZCacheRun(t *testing.T) {
	alone := renderMathMarkdown(t, "x \\\\(`\\\\) $b c\nd$ `e\n")
	withA := renderMathMarkdown(t, "$a \\\\(`\\\\) $b c\nd$ `e\n")
	t.Logf("alone: %d math, withA: %d math", strings.Count(alone, "<math"), strings.Count(withA, "<math"))
	if !strings.Contains(withA, ">b c\nd</annotation>") {
		t.Errorf("$b c⏎d$ is text once $a precedes it:\n%s", withA)
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestZZCacheRun -v -count=1
rm gno.land/pkg/gnoweb/markdown/zz_cache_run_test.go
```

The test fails: the same `$b` opener gets a different answer depending on an unrelated `$a` before it.

```
    zz_cache_run_test.go:11: alone: 2 math, withA: 1 math
    zz_cache_run_test.go:13: $b c⏎d$ is text once $a precedes it:
        <p>$a <math class="math-textstyle" display="inline" xmlns="http://www.w3.org/1998/Math/MathML"><semantics><mrow><mo>`</mo></mrow><annotation encoding="application/x-tex">`</annotation></semantics></math> $b c
        d$ `e</p>
--- FAIL: TestZZCacheRun (0.00s)
```

The `$a` search counts the backtick inside `` \\(`\\) `` as an unclosed run and caches it. goldmark then parses `` \\(`\\) `` as math and asks for `$b`, and the cache answers with that run, so the next-line search starts inside a code span and skips `d$`. With `findCloseCached` bypassed (`return find(b, ticks, nil)` after its length check), the same input renders 2 `<math>`. The merge base fails the test too, on another path: with no run state, `$a` pairs with `d$` and takes `` \\(`\\) $b c `` into one formula. The doc comment on `findCloseCached` says a run before the opener that closes on the next line makes goldmark read the opener as code and never ask; that holds for a code span, not for a run inside `\\(…\\)` math or a raw HTML attribute (`` $a <x y="`"> $b c `` fails the same way).

</details>

## gno.land/pkg/gnoweb/markdown/ext_math.go:542 [gh](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/gno.land/pkg/gnoweb/markdown/ext_math.go#L542) · [↗](../../../../../.worktrees/gno-review-6326/gno.land/pkg/gnoweb/markdown/ext_math.go#L542) · Warning [posted](https://github.com/gnolang/gno/pull/6326#discussion_r4443978345)

`parenEnd(b, j, depth)` counts parentheses inside a quoted title, so the closing `$$` of `$$ [a](x "(") b $$` is never found and the formula renders as text.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6326 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_title_paren_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestZZTitleParen(t *testing.T) {
	for _, src := range []string{
		`$$ [a](x "(") b $$` + "\n",
		`\\( [a](x "(") b \\)` + "\n",
		`\\( [a](<x(>) b \\)` + "\n",
		`\\( [a](<x b \\)` + "\n",
		`Pay $$ [a](x "(") b $$ then` + "\nsee $$ below.\n",
	} {
		out := renderMathMarkdown(t, src)
		n := strings.Count(out, "<math")
		t.Logf("%q: math=%d", src, n)
		if n == 0 || strings.Contains(out, "see </annotation>") {
			t.Errorf("%q: closer after the link missed", src)
		}
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestZZTitleParen -v -count=1
rm gno.land/pkg/gnoweb/markdown/zz_title_paren_test.go
```

The test fails on every shape: each renders one `<math>` at the merge base, and the last one pairs with the next line's `$$`, turning ` then⏎see` into TeX.

```
    zz_title_paren_test.go:18: "$$ [a](x \"(\") b $$\n": math=0
    zz_title_paren_test.go:20: "$$ [a](x \"(\") b $$\n": closer after the link missed
    zz_title_paren_test.go:18: "\\\\( [a](x \"(\") b \\\\)\n": math=0
    # …
    zz_title_paren_test.go:18: "\\\\( [a](<x(>) b \\\\)\n": math=0
    # …
    zz_title_paren_test.go:18: "\\\\( [a](<x b \\\\)\n": math=0
    # …
    zz_title_paren_test.go:18: "Pay $$ [a](x \"(\") b $$ then\nsee $$ below.\n": math=1
    zz_title_paren_test.go:20: "Pay $$ [a](x \"(\") b $$ then\nsee $$ below.\n": closer after the link missed
--- FAIL: TestZZTitleParen (0.00s)
```

In `$$ [a](x "(") b $$` the `(` in the title raises the depth to 2, the `)` after the title brings it back to 1, and the scan runs to the line end. The `<…>` branches of `linkEnd` count the same way inside a destination, and an unclosed `<` (`\\( [a](<x b \\)`), which goldmark reads as no link, is skipped the same way. Below a `$$` line, `$$ [a](x "(") b $$` opens a display block that swallows the paragraph up to it. A title holding `)` cuts the other way, at the merge base too: in `$$ [a](x "t) $$ q") c $$` the `$$` inside the title closes the formula and the link is lost, which the added "Still whole: a link with a title" case of `TestMathClosesInBracketsThatAreNoLink` does not cover.

</details>

## SKIP gnovm/stdlibs/chain/markdown/markdown.go:86 [gh](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/gnovm/stdlibs/chain/markdown/markdown.go#L86) · [↗](../../../../../.worktrees/gno-review-6326/gnovm/stdlibs/chain/markdown/markdown.go#L86) · Suggestion

Suggestion: `'$'` in `inlineEscapeSet` misses stored text, since [`prop_requests.gno:248`](https://github.com/gnolang/gno/blob/f6a568ed830794fafe1ca4840786c30d4dc0c233/examples/gno.land/r/gov/dao/impl/v0/prop_requests.gno#L248) escapes a treasury reason once, so one stored before the upgrade keeps a bare `$`. I think the ADR should name that limit.

Not posted: gnoland-1 escapes and stores the reason this way, but none of its 16 proposals holds a `$` outside its vote links, so nothing renders wrong today.
