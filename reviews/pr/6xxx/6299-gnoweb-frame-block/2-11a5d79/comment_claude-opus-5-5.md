# Review: [#6299](https://github.com/gnolang/gno/pull/6299)
Posted: https://github.com/gnolang/gno/pull/6299#pullrequestreview-5458897507

Event: COMMENT
Verdict: APPROVE. Nothing above Nit is open: one missing golden for the refused-card flag reset at a columns tag, and two Nits on frame tag lines that either end an HTML comment early or end the outer frame from inside a card.
Model: claude-opus-5-5, solo review (one judge and writer, high)
Commit: 11a5d796a07a92ece6344770ca86e6baac84ebbb
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6299 11a5d796a`
Round: 2, scoped to the fix commits 96fcf8a2b..11a5d796a. 2 finders, a reflector pass by the judge, 6 candidates (5 from the finders, 1 from the reflector), each run from scratch by the judge, which was not its finder: 5 confirmed, 1 refuted. Round 1: `ext_frame.go:270` Warning partly (card-opener half resolved, `html_block_before_card_open` passes; comment half carried as the documented trade-off `html_comment_cut_by_columns` pins); `pr6299_gnoweb_frame.md:157-158` Nit resolved (reworded); `ext_frame.go:231` Nit partly (attribute and depth-cap cards resolved, `invalid_card_attrs_keeps_frame` and `invalid_card_at_depth_cap` pass; a frame refused inside a card is the open half, at line 254 below); `blockrich-gno-frame-indented.txtar:17` Nit carried, untouched by the fix commits; `utils.go:144` Nit carried, declined; `ext_frame.go:167-172` Suggestion resolved (`columns_stray_tags_stay_in_frame` passes); both Body nits resolved (`TestFrameNULInTag` passes, the `trimSentinel` comment names `trimTagLine`).

## Body

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6299-gnoweb-frame-block/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6299-gnoweb-frame-block/2-11a5d79/claims.md) · Status: APPROVE

## gno.land/pkg/gnoweb/markdown/ext_frame.go:226 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L226) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L226) · Missing test [posted](https://github.com/gnolang/gno/pull/6299#discussion_r4220794172)

Missing test: deleting this reset keeps every golden green, since none has a refused card left open before `<gno-columns-sep>`.

<details><summary>test cases</summary>

`golden/ext_frame/invalid_card_unclosed_ends_at_sep.md.txtar`, passing at the head and failing with line 226 deleted:

```
-- input.md --
<gno-frame>
<gno-columns>
<gno-frame x="1">
card
<gno-columns-sep>
b
</gno-columns>
</gno-frame>
after
-- output.html --
<section class="gno-frame">
<div class="gno-columns">
<!-- Column 0 -->
<div class="gno-column">
<!-- unexpected/invalid frame tag omitted -->
<p>card</p>
</div>
<!-- Column 1 -->
<div class="gno-column">
<p>b</p>
</div>
</div> <!-- </gno-columns> -->
</section>
<p>after</p>
```

With the line deleted, the page ends `<!-- unexpected/invalid frame tag omitted -->`, `<p>after</p>`, `</section>`.

</details>

## gno.land/pkg/gnoweb/markdown/ext_frame.go:254 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L254) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L254) · Nit [posted](https://github.com/gnolang/gno/pull/6299#discussion_r4220794185)

Nit: `!frameInner(pc)` remembers only a refused card opener, so in a card holding a refused frame the card's close ends the outer frame and the grid lands outside it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6299 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_frame_in_card_test.go <<'EOF'
package markdown

import (
	"bytes"
	"fmt"
	"testing"
)

func TestFrameInCard(t *testing.T) {
	in := "<gno-frame>\n<gno-columns>\n<gno-frame>\n<gno-frame>\nx\n</gno-frame>\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\nafter\n"
	var buf bytes.Buffer
	if err := convertGno(newGnoMarkdown(), []byte(in), &buf); err != nil {
		t.Fatal(err)
	}
	fmt.Print(buf.String())
}
EOF
(cd gno.land && go test ./pkg/gnoweb/markdown -run TestFrameInCard -count=1 -v)
rm gno.land/pkg/gnoweb/markdown/zz_frame_in_card_test.go
```

The frame closes before the grid opens, and the outer `</gno-frame>` is left as a stray comment:

```
<section class="gno-frame">
</section>
<div class="gno-columns">
<!-- Column 0 -->
<div class="gno-column">
<section class="gno-frame">
<!-- unexpected/invalid frame tag omitted -->
<p>x</p>
</section>
<!-- unexpected/invalid frame tag omitted -->
</div>
# …
</div> <!-- </gno-columns> -->
<!-- unexpected/invalid frame tag omitted -->
<p>after</p>
```

A frame refused directly inside a frame has the same shape: its close ends the outer frame, which golden `invalid_nested_frame` asserts. Remembering every refused opener while a frame is open, tested ahead of the inner-close case, turns `invalid_frame_in_card`, `invalid_nested_frame` and `invalid_nested_keeps_depth` red, so the goldens record the current reading for both shapes.

</details>

## SKIP gno.land/pkg/gnoweb/markdown/ext_frame.go:254 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L254) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L254) · Nit

Nit: `line[1] != '/'` holds for `<gno-frame/>` too, so a self-closing tag in a framed column eats the next `</gno-frame>` as its close.

Not posted: two readings, and the ADR lists the self-closing form beside attributes as one invalid tag, so treating it as a refused opener is consistent; with the outer close present the grid stays in the frame, and only a page whose one `</gno-frame>` sits in that column has its outer frame wrap the rest of the page.

## gno.land/pkg/gnoweb/markdown/ext_frame.go:292 [gh](https://github.com/gnolang/gno/blob/11a5d796a07a92ece6344770ca86e6baac84ebbb/gno.land/pkg/gnoweb/markdown/ext_frame.go#L292) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L292) · Nit [posted](https://github.com/gnolang/gno/pull/6299#discussion_r4220794205)

Nit: `parseFrameLineTag(tag) != frameTagNone` ends an `<!--` comment at a `<gno-frame>` line in a frame with no grid, where the line opens nothing, so the comment's rest renders as text.

```suggestion
		if tag := trimTagLine(line); parseFrameLineTag(tag) == frameTagClose || columnsLineTag(tag) != GnoColumnTagUndefined || (parseFrameLineTag(tag) != frameTagNone && frameGrid(pc) && gridOpen(pc)) {
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6299 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_comment_opener_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestFrameCommentHidesOpener(t *testing.T) {
	for name, in := range map[string]string{
		"opener":  "<gno-frame>\nintro\n<!--\n<gno-frame>\nsecret\n-->\n</gno-frame>\nafter\n",
		"invalid": "<gno-frame>\nintro\n<!--\n<gno-frame x=\"1\">\nsecret\n-->\n</gno-frame>\nafter\n",
	} {
		var buf bytes.Buffer
		if err := convertGno(newGnoMarkdown(), []byte(in), &buf); err != nil {
			t.Fatal(err)
		}
		if out := buf.String(); strings.Contains(out, "secret") || strings.Contains(out, "--&gt;") {
			t.Errorf("%s: commented-out text rendered:\n%s", name, out)
		}
	}
}
EOF
(cd gno.land && go test ./pkg/gnoweb/markdown -run TestFrameCommentHidesOpener -count=1 -v)
rm gno.land/pkg/gnoweb/markdown/zz_comment_opener_test.go
```

The commented-out line renders inside the frame:

```
    zz_comment_opener_test.go:19: opener: commented-out text rendered:
        <section class="gno-frame">
        <p>intro</p>
        <!-- raw HTML omitted -->
        <!-- unexpected/invalid frame tag omitted -->
        <p>secret
        --&gt;</p>
        </section>
        <p>after</p>
# …
--- FAIL: TestFrameCommentHidesOpener (0.00s)
```

With the suggestion applied the test passes and the whole `gno.land/pkg/gnoweb/markdown` package stays green; cutting on an opener only for HTML block types 6 and 7 also passes it but lets a commented-out card's close end the outer frame again.

</details>
