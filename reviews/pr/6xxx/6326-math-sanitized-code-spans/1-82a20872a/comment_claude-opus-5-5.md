# Review: [#6326](https://github.com/gnolang/gno/pull/6326)
Posted: https://github.com/gnolang/gno/pull/6326#pullrequestreview-5551618615
Event: COMMENT
Verdict: REQUEST CHANGES. The `$$`, `\\(` and `\\[` closers inherit a link-destination skip that runs a formula past its own closer and over a later link, which the branch introduces; the label-stop and code-span gaps predate it and do not block.
Model: claude-opus-5-5, standard review; finders xhigh, judge and writer high
Commit: 82a20872a
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/alexiscolin/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639) · [vscode.dev](https://vscode.dev/github/alexiscolin/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6326 82a20872a`
Round: 1. Two finders, then one judge and writer answering the completeness question, eight candidates, each run or read from scratch by an agent that was not its finder; two Nits read and probed in the same batch. The parent moved the Missing test into the Body, since the golden it asks for sits outside the diff, skipped the link-title Suggestion as a finding on a code comment's wording, and linked gfanton's stays-text case from the Warning.

## Body

> AI review, claude-opus-5-5, standard review, reviewed at [82a20872a](https://github.com/gnolang/gno/pull/6326/commits/82a20872a7e4eb89cae377e8ef46c5c578b1b639), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6326-math-sanitized-code-spans/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6326-math-sanitized-code-spans/1-82a20872a/claims.md) · Status: REQUEST CHANGES

- Related: [`ext_math.go:201`](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L201) · [↗](../../../../../.worktrees/gno-review-6326/gno.land/pkg/gnoweb/markdown/ext_math.go#L201) searches the next line after a label stop, so `[Send $10](/r/demo/bank$help&func=Send)` above `Balance: 100$` becomes one formula.
- Related nit: [`ext_math.go:204`](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L204) · [↗](../../../../../.worktrees/gno-review-6326/gno.land/pkg/gnoweb/markdown/ext_math.go#L204) starts the next-line search outside any code span, so a code span crossing the line break is cut by a `\\)` inside it.
- Related nit: [`ext_math.go:491`](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L491) · [↗](../../../../../.worktrees/gno-review-6326/gno.land/pkg/gnoweb/markdown/ext_math.go#L491) counts a `$$` inside a code span, so ``$$ a `$$` b`` opening a display block is taken by neither parser.
- Missing test: no `golden/sanitize` case like [`inline-newline-fold.txtar`](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/golden/sanitize/inline-newline-fold.txtar#L1) renders a `$` passed through `InlineText` or `TableCell`, so a gnoweb change that lets `\$` open math passes.

<details><summary>test cases</summary>

`gno.land/pkg/gnoweb/markdown/golden/sanitize/inline-dollar-not-math.txtar`, outputs filled with `-update-golden-tests`:

```
// MARKDOWNFUNC: InlineText
// CONTEXT: Winner: %s
-- input.md --
$x^{2} + ADMIN_{verified}$
-- output.md --
-- output.html --
```
</details>


## gno.land/pkg/gnoweb/markdown/ext_math.go:359 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gno.land/pkg/gnoweb/markdown/ext_math.go#L359) · [↗](../../../../../.worktrees/gno-review-6326/gno.land/pkg/gnoweb/markdown/ext_math.go#L359) · Warning [posted](https://github.com/gnolang/gno/pull/6326#discussion_r4297639749)

An unclosed `[a,b](` inside a `$$`, `\\(` or `\\[` formula hides its closer until a balanced `)`, so a later link's `)` lets the formula swallow that link rather than only [stay text](https://github.com/gnolang/gno/pull/6326#discussion_r4283542736).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6326 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'GO'
package markdown

import "testing"

func TestZZRepro(t *testing.T) {
	for _, src := range []string{
		`\\( [a,b](x \\) and [l](/r/y) then \\(z\\)` + "\n",
		"$$[0,1](x $$ and [l](/r/y) z\nw $$\n",
		"[Send $10](/r/demo/bank$help&func=Send)\nBalance: 100$\n",
	} {
		t.Logf("%s", renderMathMarkdown(t, src))
	}
}
GO
(cd gno.land && go test ./pkg/gnoweb/markdown -run TestZZRepro -count=1 -v | grep -o 'x-tex">[^<]*\|href="/r/[^"]*"\|^ok.*')
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

Each line is the TeX of the one formula an input becomes; no `href="/r/y"` is printed, so both links to `/r/y` are inside a formula (the third input is the Body's label-stop case):

```
x-tex"> [a,b](x \\) and [l](/r/y) then \\(z
x-tex">[0,1](x $$ and [l](/r/y) z
x-tex">10](/r/demo/bank$help&amp;func=Send)
ok  	github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown	0.022s
```

The first input closes at its first `\\)` on the merge base and renders the link. The `)` of `\\)` is read as the end of the destination `(x \\)`, so the closer's first byte is never tested; in the second, line 1 has no closer outside the destination and line 2's `$$` closes it. A CommonMark destination holds no space and never crosses a line, so neither span is a link goldmark would parse.
</details>

## SKIP gnovm/stdlibs/chain/markdown/markdown.go:92 [gh](https://github.com/gnolang/gno/blob/82a20872a7e4eb89cae377e8ef46c5c578b1b639/gnovm/stdlibs/chain/markdown/markdown.go#L92) · [↗](../../../../../.worktrees/gno-review-6326/gnovm/stdlibs/chain/markdown/markdown.go#L92) · Suggestion

Suggestion: gnoweb applies backslash escapes in a link title, so `\$` renders as `$` and the backslash this exception avoids never shows.

Not posted: a finding on the wording of the code comment at lines 90-91, which ships SKIP by default; the measurement is row 1001 in claims.md.
