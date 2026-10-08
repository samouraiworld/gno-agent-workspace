# Review: [#6299](https://github.com/gnolang/gno/pull/6299)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. One Warning the branch introduces in the new frame parser: an HTML block inside a frame either swallows a card's opening tag or lets a commented-out columns line render live, and both push the grid out of its frame.
Model: claude-opus-5-5, standard review (finders xhigh, the other stages high)
Commit: 1f9bf51d8e71902053e3bd92986f92d24431efc2
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6299 1f9bf51d8`
Round: 1. 5 finders, one reflector, 7 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted, none of them above Nit.

## Body

- Nit: a raw NUL byte in [`invalid_unicode_lookalikes.md.txtar`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/golden/ext_frame/invalid_unicode_lookalikes.md.txtar#L6) makes git and GitHub show the golden as binary, hiding all three inputs and their output. The sanitize goldens write such bytes as text through [`// INPUT_ESCAPED`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go#L204), which the `ext_frame` golden harness does not read.
- SKIP Nit: the [`trimSentinel` comment](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/examples/gno.land/p/nt/markdown/foreign/v0/foreign.gno#L223) names `trimForeignLine`, which no function in the tree carries, for the trim the parser performs through `trimTagLine`.

  Not posted: the line sits outside the diff, the finding is about a comment's wording, and it rests on the finder's read alone.

## gno.land/pkg/gnoweb/markdown/ext_frame.go:270 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/ext_frame.go#L270) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L270) · Warning
`frameHTMLBlockParser.Continue` lets an HTML block in a frame swallow a card's `<gno-frame>` or end at a commented-out `<gno-columns>`, and either pushes the grid out of the frame.

> Headless Chromium captured this branch's gnoweb rendering the repro's two failing cases from a stub client, as written on the left and without the HTML block on the right.

![div_before_card_open and comment_hides_columns rendered by gnoweb, each beside the same markdown without its HTML block](https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6299-gnoweb-frame-block/1-1f9bf51/media/frame-html-block-escapes-grid.png)

<details><summary>cases</summary>

- `<div></div>` directly above a card's `<gno-frame>` takes the card's opening tag and its text, so the card's `</gno-frame>` ends the outer frame.
- A `<!-- ... -->` comment hiding a `<gno-columns>` line ends at that line, so the grid goes live and `-->` renders as text.
- The same comment outside a frame stays hidden, because [`frameHTMLBlockParser.Open`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/ext_frame.go#L261-L262) defers to goldmark there.
- An HTML line above a card's `</gno-frame>`, or one followed by a blank line, keeps the grid inside the frame.

</details>

<details><summary>repro</summary>

The `div_before_card_open` and `comment_hides_columns` cases fail, and the three controls pass: the outer frame closes before the grid opens.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6299 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_frame_html_block_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"
)

func TestFrameHTMLBlockKeepsGrid(t *testing.T) {
	cases := []struct{ name, src string }{
		{"control_no_html", "<gno-frame>\n<gno-columns>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		{"control_html_before_close", "<gno-frame>\n<gno-columns>\n<gno-frame>\ncard\n<br>\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		{"control_blank_after_html", "<gno-frame>\n<gno-columns>\n<br>\n\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		{"div_before_card_open", "<gno-frame>\n## Apps\n<gno-columns>\n<div></div>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n"},
		{"comment_hides_columns", "<gno-frame>\nintro\n<!--\ndraft\n<gno-columns>\nhidden\n-->\n</gno-frame>\n"},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		if err := convertGno(newGnoMarkdown(), []byte(c.src), &buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		g := strings.Index(out, `<div class="gno-columns">`)
		e := strings.Index(out, `</section>`)
		if g >= 0 && e >= 0 && e < g || strings.Contains(out, "--&gt;") {
			t.Errorf("%s: the frame ends before the grid\n%s", c.name, out)
		}
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown/ -run TestFrameHTMLBlockKeepsGrid -count=1
rm gno.land/pkg/gnoweb/markdown/zz_frame_html_block_test.go
```

```text
--- FAIL: TestFrameHTMLBlockKeepsGrid (0.00s)
    zz_frame_html_block_test.go:26: div_before_card_open: the frame ends before the grid
        <section class="gno-frame">
        <h2>Apps</h2>
        </section>
        <div class="gno-columns">
        <!-- Column 0 -->
        <div class="gno-column">
        <!-- raw HTML omitted -->
        <!-- unexpected/invalid frame tag omitted -->
        # …
    zz_frame_html_block_test.go:26: comment_hides_columns: the frame ends before the grid
        <section class="gno-frame">
        <p>intro</p>
        <!-- raw HTML omitted -->
        </section>
        <div class="gno-columns">
        <!-- Column 0 -->
        <div class="gno-column">
        <p>hidden
        --&gt;</p>
        # …
FAIL
```

</details>

## gno.land/adr/pr6299_gnoweb_frame.md:157-158 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/adr/pr6299_gnoweb_frame.md?plain=1#L157-L158) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/adr/pr6299_gnoweb_frame.md#L157) · Nit
Nit: this branch's [`utils.go`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/utils.go#L219) adds `trimTagLine` and its `utils_test.go` lacks the [`FuzzScanGnoTag`](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/utils_test.go#L48) of #6298, so dropping either copy breaks the build or loses the fuzz test.

## gno.land/pkg/gnoweb/markdown/ext_frame.go:231 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/ext_frame.go#L231) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L231) · Nit
Nit: a refused card's close, at the depth cap or on an attribute, [ends the outer frame](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/golden/ext_frame/invalid_card_at_depth_cap.md.txtar#L18-L20), since only the [inner-frame case](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/ext_frame.go#L225-L227) marks a card open.

## gno.land/pkg/gnoweb/markdown/golden/sanitize/blockrich-gno-frame-indented.txtar:17 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/golden/sanitize/blockrich-gno-frame-indented.txtar#L17) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/golden/sanitize/blockrich-gno-frame-indented.txtar#L17) · Nit
Nit: `blockrich-gno-frame-indented` renders an indented frame tag as a lone backslash and stripped HTML instead of the promised [literal text](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L391), since [`escapeBlockHazardsImpl`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gnovm/stdlibs/chain/markdown/markdown.go#L430-L432) writes the backslash before the indent.

## gno.land/pkg/gnoweb/markdown/utils.go:144 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/utils.go#L144) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/utils.go#L144) · Nit
Nit: `gnoTagLineParser` has no caller on this branch outside [`utils_tagline_test.go`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/utils_tagline_test.go#L15), so merged alone it ships a parser that no extension registers. Its registrations live in [#6297](https://github.com/gnolang/gno/blob/56ff1770722eee73f51f207f97554ea1e2ea2b00/gno.land/pkg/gnoweb/markdown/ext_icons.go#L338) and [#6298](https://github.com/gnolang/gno/blob/065ec369b321c207815f82b0c1dbf2b0f9f10b01/gno.land/pkg/gnoweb/markdown/ext_buttons.go#L190), and the linter's [`unused`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/.github/golangci.yml#L30) check counts the [test call](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/.github/golangci.yml#L8) as a use.

## gno.land/pkg/gnoweb/markdown/ext_frame.go:167-172 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/ext_frame.go#L167-L172) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L167) · Suggestion
Suggestion: the `GnoColumnTagSep` and `GnoColumnTagClose` cases [end the frame](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/golden/ext_frame/columns_sep_outside_grid_in_frame.md.txtar#L8-L12) on a stray `<gno-columns-sep>` or [`</gno-columns>`](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/golden/ext_frame/columns_stray_close_in_frame.md.txtar#L7-L10) with no grid open, and I think such a tag belongs inside the frame.

## SKIP gno.land/pkg/gnoweb/markdown/ext_frame.go:160 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/ext_frame.go#L160) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/ext_frame.go#L160) · Missing test
Missing test: a second `<gno-columns>` inside a frame's own grid, the `return frameGrid(pc)` branch, has no golden pinning its output, and only a depth-balance test runs that input.

Not posted: no verifier ran the mutation that would show the branch unguarded.

## SKIP gno.land/pkg/gnoweb/markdown/golden/ext_frame/valid_indented_three.md.txtar:4 [gh](https://github.com/gnolang/gno/blob/1f9bf51d8e71902053e3bd92986f92d24431efc2/gno.land/pkg/gnoweb/markdown/golden/ext_frame/valid_indented_three.md.txtar#L4) · [↗](../../../../../.worktrees/gno-review-6299/gno.land/pkg/gnoweb/markdown/golden/ext_frame/valid_indented_three.md.txtar#L4) · Missing test
Missing test: no golden pins a `</gno-frame>` indented under a list item, which leaves the frame open to the end of the page.

Not posted: no verifier rendered the input.
