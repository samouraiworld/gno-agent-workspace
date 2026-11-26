# Review: [#6304](https://github.com/gnolang/gno/pull/6304)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES, a dedented fence turns the rest of a Rich post into code, a list-item fence indented 4-6 spaces or by a tab lets the user's `[home]` link to the realm's definition, and past the scan budget a definition line with trailing text still puts the realm's footer in a code block.
Model: claude-opus-5-5, standard review
Commit: a8bdd53b0 (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6304 a8bdd53b0`
Round: 1. 6 finders, one reflector, 12 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## Body

Automatic review, not manually checked, here to help the reviewer team.

- This can't go in yet: three inputs the merge base handled now render wrongly, and the new check on text after a link definition stops at the scan's work cap. The blockers are the comments on the closing fence [`closeFencesAtDedent`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1012) writes, the indent loop in `parseFenceOpener`, the tab guard in `isNonBlankIndentedLess` and the whitespace guard in `scanLRDTail`. The HTML-block, table and link-text comments, the Nits and the calibration bullet are optional.

- Suggestion: [`gen_native_table.py:209-210`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/cmd/calibrate/gen_native_table.py#L209-L210) fits `EscapeBlockHazards` only from `BenchmarkNative_Markdown_EscapeBlockHazards_(\d+)-`, which matches neither [`BenchmarkNative_Markdown_EscapeBlockHazards_Slow_1000`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/cmd/calibrate/markdown_bench_test.go#L379) nor any `EscapeBlockHazardsRich` name. A re-fit pasted into [`native_gas.go`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/native_gas.go#L184-L185) therefore drops the ×2.45 strict and ×2.95 Rich gas multipliers this pull request applies. Fix: for instance, add script rows for the `_Slow_` benchmarks and fit the shipped slopes from them, or carry the factors in the script's table.

## gnovm/stdlibs/chain/markdown/markdown.go:505 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L505) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L505) · Warning

Related: [`mayOpenHTMLBlock`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L567-L570) misses a `<div>` after a list marker, so the sanitizer's fence tracker counts a fence goldmark ignores, and the item's `[home]` links to the realm's URL.

**Fix:** for instance, pass `mayOpenHTMLBlock` the line with its list and quote markers stripped, as the fence tracker reads it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	for _, in := range []string{
		"- a\n\n  see [home]\n",
		"- <div>\n  ```\n\n  see [home]\n",
	} {
		out := callSanitizeGno(t, "BlockRich", in, "", "")
		html := renderMarkdown(t, "[home]: https://realm.example/home\n\nREALM BEFORE"+out+"REALM AFTER\n")
		t.Logf("%q realm-link=%v", in, strings.Contains(html, `href="https://realm.example/home"`))
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep realm-link
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

The `<div>` item links to the realm's URL, and the merge base prints the same:

```
zz_repro_test.go:15: "- a\n\n  see [home]\n" realm-link=false
zz_repro_test.go:15: "- <div>\n  ```\n\n  see [home]\n" realm-link=true
```

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:882 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L882) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L882) · Warning

`i < 3` counts indent from column 0, so the sanitizer's fence tracker misses a fence 4-6 spaces into a list item, and the item's `[home]` links to the realm's URL.

**Fix:** for instance, measure the indent from the enclosing list item's content column, and add both inputs below to `TestEscapeBlockHazardsRich`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	for _, in := range []string{
		"- a\n\n  see [home]\n",
		"- a\n\n     ```\n  ```\n  see [home]\n",
		"1. a\n\n    ```\n   ```\n   see [home]\n",
	} {
		out := callSanitizeGno(t, "BlockRich", in, "", "")
		html := renderMarkdown(t, "[home]: https://realm.example/home\n\nREALM BEFORE"+out+"REALM AFTER\n")
		t.Logf("%q realm-link=%v", in, strings.Contains(html, `href="https://realm.example/home"`))
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep realm-link
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

Both fenced items link to the realm's URL:

```
zz_repro_test.go:16: "- a\n\n  see [home]\n" realm-link=false
zz_repro_test.go:16: "- a\n\n     ```\n  ```\n  see [home]\n" realm-link=true
zz_repro_test.go:16: "1. a\n\n    ```\n   ```\n   see [home]\n" realm-link=true
```

The merge base escapes both:

```
zz_repro_test.go:16: "- a\n\n  see [home]\n" realm-link=false
zz_repro_test.go:16: "- a\n\n     ```\n  ```\n  see [home]\n" realm-link=false
zz_repro_test.go:16: "1. a\n\n    ```\n   ```\n   see [home]\n" realm-link=false
```

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:917 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L917) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L917) · Warning

`isNonBlankIndentedLess` returns false on a leading tab, so the sanitizer's fence tracker misses a tab-indented fence line in a list item, and the item's `[home]` links to the realm's URL.

**Fix:** for instance, expand a leading tab to the next multiple of 4 columns in `parseFenceOpener`, [`isCloseFence`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1155) and `isNonBlankIndentedLess` before comparing indents.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	for _, in := range []string{
		"- a\n\n  see [home]\n",
		"- a\n\n\t```\n  ```\n  see [home]\n",
		"- a\n\n  ```\n\t```\n  see [home]\n",
	} {
		out := callSanitizeGno(t, "BlockRich", in, "", "")
		html := renderMarkdown(t, "[home]: https://realm.example/home\n\nREALM BEFORE"+out+"REALM AFTER\n")
		t.Logf("%q realm-link=%v", in, strings.Contains(html, `href="https://realm.example/home"`))
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep realm-link
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

The tab-indented opener and closer both link to the realm's URL:

```
zz_repro_test.go:16: "- a\n\n  see [home]\n" realm-link=false
zz_repro_test.go:16: "- a\n\n\t```\n  ```\n  see [home]\n" realm-link=true
zz_repro_test.go:16: "- a\n\n  ```\n\t```\n  see [home]\n" realm-link=true
```

The merge base escapes the opener form and fails the closer form the same way:

```
zz_repro_test.go:16: "- a\n\n  see [home]\n" realm-link=false
zz_repro_test.go:16: "- a\n\n\t```\n  ```\n  see [home]\n" realm-link=false
zz_repro_test.go:16: "- a\n\n  ```\n\t```\n  see [home]\n" realm-link=true
```

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:1057 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1057) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L1057) · Warning

`closeFencesAtDedent` writes a closer before column-0 code in a fence indented 1-3 spaces, so the user's own column-0 closer then opens a new fence and every later paragraph renders as code.

**Fix:** write the dedented line at the opener's column instead of closing the fence before it; only the golden files expecting the fence split in two change.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	in := " ```\ncode\n```\n\nprose after\n\n## heading after\n"
	for _, fn := range []string{"Block", "BlockRich", "BlockquoteRich"} {
		html := renderMarkdown(t, callSanitizeGno(t, fn, in, "", ""))
		t.Logf("%s prose-is-paragraph=%v", fn, strings.Contains(html, "<p>prose after</p>"))
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep prose-is
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

Neither Rich helper renders the prose as a paragraph:

```
zz_repro_test.go:12: Block prose-is-paragraph=true
zz_repro_test.go:12: BlockRich prose-is-paragraph=false
zz_repro_test.go:12: BlockquoteRich prose-is-paragraph=false
```

The merge base keeps it a paragraph in every helper:

```
zz_repro_test.go:12: Block prose-is-paragraph=true
zz_repro_test.go:12: BlockRich prose-is-paragraph=true
zz_repro_test.go:12: BlockquoteRich prose-is-paragraph=true
```

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:1123 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1123) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L1123) · Warning

Related: only a line-leading `|` is escaped, so `Block` passes a GFM table with no leading pipe, against the [package doc](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L161-L162) promising no `<table>`.

**Fix:** for instance, escape the first `|` of a GFM delimiter row in strict mode, or track it in an issue if it stays out of this pull request.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	out := callSanitizeGno(t, "Block", "a | b\n--- | ---\nc | d\n", "", "")
	html := renderMarkdown(t, "REALM BEFORE\n"+out+"\nREALM AFTER\n")
	t.Logf("table=%v", strings.Contains(html, "<table>"))
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep table=
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

`Block` lets the table through, and the merge base prints the same:

```
zz_repro_test.go:11: table=true
```

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:1316 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1316) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L1316) · Warning

Related: `findBracketSpans` refuses a link span only across a fence or definition line, so a `> [!WARNING]` line inside link text renders a gnoweb alert.

**Fix:** for instance, reject a link span crossing any line [`isBlockInterrupt`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1736) accepts, or split the span at that line as [`appendFenceSpans`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1927) does.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	for _, in := range []string{
		"> [!WARNING] spoof\n",
		"[a\n> [!WARNING] spoof](/u)\n",
		"[a\n> see [home]](/u)\n",
	} {
		out := callSanitizeGno(t, "BlockRich", in, "", "")
		html := renderMarkdown(t, "[home]: https://realm.example/home\n\nREALM BEFORE"+out+"REALM AFTER\n")
		t.Logf("%q alert=%v realm-link=%v", in, strings.Contains(html, "gno-alert"),
			strings.Contains(html, `href="https://realm.example/home"`))
	}
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep realm-link
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

The alert line on its own is escaped, and inside a link span it renders; the merge base prints the same:

```
zz_repro_test.go:16: "> [!WARNING] spoof\n" alert=false realm-link=false
zz_repro_test.go:16: "[a\n> [!WARNING] spoof](/u)\n" alert=true realm-link=false
zz_repro_test.go:16: "[a\n> see [home]](/u)\n" alert=false realm-link=true
```

</details>

## gnovm/stdlibs/chain/markdown/markdown.go:1712 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1712) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L1712) · Warning

Past the scan budget, `scanLRDTail` skips its trailing-text check, so a definition line ending in a fence is deleted and the next fence puts the realm's footer in code.

**Fix:** make the check fail closed; `TestEscapeBlockHazards` and `TestEscapeBlockHazardsRich` stay green with it.

```suggestion
	if !ok || end < len(s) && s[end] != '\n' {
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6304 -R gnolang/gno
cat > gnovm/stdlibs/chain/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestRepro(t *testing.T) {
	const target = "[a]: u ```\n```\n- y\n  ```\n~~~\n"
	for k := 8; k <= 10; k++ {
		for tail := 0; tail <= 1500; tail++ {
			in := strings.Repeat("[a]: u\n(x\n", k) + strings.Repeat("`\n", tail) + "\n" + target + "\n\n"
			out := EscapeBlockHazardsRich(in)
			if strings.Contains(out, "u ```") {
				continue
			}
			var buf bytes.Buffer
			if err := goldmark.New().Convert([]byte("HEADER\n\n\n"+out+"\n\nREALM-FOOTER\n"), &buf); err != nil {
				t.Fatal(err)
			}
			html := buf.String()
			f := strings.Index(html, "REALM-FOOTER")
			open := strings.LastIndex(html[:f], "<pre><code")
			inCode := open >= 0 && !strings.Contains(html[open:f], "</code></pre>")
			t.Logf("k=%d tail=%d: definition line deleted, footer in code block=%v", k, tail, inCode)
			return
		}
	}
	t.Log("definition line kept at every size")
}
EOF
go test ./gnovm/stdlibs/chain/markdown -run TestRepro -v -count=1 | grep definition
rm gnovm/stdlibs/chain/markdown/zz_repro_test.go
```

Past the budget the line is deleted and the footer lands in the code block:

```
zz_repro_test.go:28: k=9 tail=269: definition line deleted, footer in code block=true
```

The merge base, which has no such check, deletes the line at the smallest size:

```
zz_repro_test.go:28: k=8 tail=0: definition line deleted, footer in code block=true
```

</details>

## gno.land/adr/pr6304_sanitize_gno_tags.md:41 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gno.land/adr/pr6304_sanitize_gno_tags.md?plain=1#L41) · [↗](../../../../../.worktrees/gno-review-6304/gno.land/adr/pr6304_sanitize_gno_tags.md#L41) · Nit

Nit: the code span holding the gas calibration input breaks across lines, so it renders as `[a]: u (x `, and the unclosed `` `\n `` spans on [line 43](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gno.land/adr/pr6304_sanitize_gno_tags.md?plain=1#L43) turn prose into code.

**Fix:** write the shape as `[a]: u\n(x\n`, as line 43 does, and close each `` `\n `` span on line 43.

## gno.land/adr/pr6304_sanitize_gno_tags.md:61 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gno.land/adr/pr6304_sanitize_gno_tags.md?plain=1#L61) · [↗](../../../../../.worktrees/gno-review-6304/gno.land/adr/pr6304_sanitize_gno_tags.md#L61) · Nit

Nit: the ADR promises a backslash on a `[label]: url` code line, while Rich mode's fence split at a less-indented line deletes it and renders `# install` as a heading.

**Fix:** say that lines a dedent split moves out of the fence render as markdown, and that a definition-shaped line among them is deleted.

## gnovm/stdlibs/chain/markdown/markdown.go:476 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L476) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/chain/markdown/markdown.go#L476) · Nit

Related nit: the `<gno-…>` escape runs after the link scan, so it breaks a `<gno-a b>` destination on the next line, and the user's `home` links to the realm's URL.

**Fix:** for instance, run the block-start `<gno-…>` escape before [`escapeBracketsOutsideLinks`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L412), as [`escapeGnoButtonTags`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L391) already is, and add a `[home](\n<gno-a b>)` row beside [`markdown_test.go:258`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown_test.go#L258).

## SKIP gnovm/stdlibs/native_gas.go:185 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/native_gas.go#L185) · [↗](../../../../../.worktrees/gno-review-6304/gnovm/stdlibs/native_gas.go#L185) · Nit

Nit: the slowest Rich shape times at a median 3.01× the old calibration per byte on a loaded host, above the ×2.95 factor this row charges.

**Fix:** for instance, round the Rich factor up with a stated margin, or re-time it on the reference machine at p75.

Not posted: the gas factors sit in alexiscolin's thread on `native_gas.go`, and the timing comes from a loaded host that is not the reference machine.

## SKIP examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:173 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L173) · [↗](../../../../../.worktrees/gno-review-6304/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L173) · Nit

Nit: the package doc exempts HTML block types 6 and 7 from any escape, against [lines 362-365](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L362-L365), which escape one followed by a fence line.

**Fix:** for instance, say that a type 6 or 7 opener followed by a fence line before the next blank line is escaped.

Not posted: it rests on the finder's read alone, and no agent ran it.

## SKIP examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:578 [gh](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L578) · [↗](../../../../../.worktrees/gno-review-6304/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L578) · Nit

Nit: the behaviour table promises an autoclose for every unclosed fence, while the tracker never opens or closes a fence on a list-marker or quote line.

**Fix:** for instance, limit the row to fences opened at document level.

Not posted: it rests on the finder's read alone, and no agent ran it.
