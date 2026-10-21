# Math conversion: comment runs, `\color` runs, malformed input
claude-opus-5-5
PR: [gnolang/gno#6377](https://github.com/gnolang/gno/pull/6377)
Review files: [reviews/pr/6xxx/6377-math-comment-runs](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6377-math-comment-runs)

## TLDR

Some math on a realm page was slow to show, and some broken math made the page show the raw formula text instead of the formula.
The change makes long runs of comment lines and of colour commands quick to convert, and stops the broken input from crashing the converter.
Math that was already correct looks the same as before.

## Blank runs and `last`

A formula is a list of tokens, and a `%` comment or a space is a token that draws nothing. `TokenBuffer.Empty` answers whether only such tokens remain, and before the change it walked the remaining blank run on every call, so a formula made of many comment lines cost its length squared.

`NewTokenBuffer` now finds `last` once, the index just past the last token that is neither a space nor a comment. `Empty` is then one comparison, `idx >= last`, and moves the position to the end when it holds. This works because a buffer's tokens never change after it is built, which the field's comment states.

## The blank-run skip in `fixFences`

`fixFences` is the tokenizer pass that joins `\left`, `\middle`, `\right` and the `\big` family to the delimiter after them, skipping the spaces and comments in between, as TeX does. Before the change it ran that skip after every token. It now checks first whether the token is one of those commands, so every other token goes straight to the output and a comment run is walked once.

## The `cellEnd` table

A style switch such as `\bf` or `\color{red}` applies up to the end of the cell it sits in: the first `&` or line break outside a group. Before the change each switch walked forward to find that point, so a run of switches walked the same tokens once per switch.

`cellEnd` now reads the answer from a table, `cellEnds`, filled in one backward pass on first use. Each entry is a distance, how far the cell end lies past that position: a cell end is 0 from itself, an ordinary token is one more than the token after it, and a group opener jumps past its closer. The after version:

```go
case opensGroup(t):
	j := min(i+t.MatchOffset+1, n)
	b.cellEnds[i] = j - i + b.cellEnds[j]
case isCellEnd(t):
	b.cellEnds[i] = 0
default:
	b.cellEnds[i] = 1 + b.cellEnds[i+1]
```

Because the entries are distances and not indexes, a sub-buffer can reuse a slice of its parent's table unchanged. `sub` cuts every sub-buffer and hands it that slice, so the table is built once per formula, at the root. Where the parent's answer lies past the sub-buffer's end, `cellEnd` clamps it to the sub-buffer's length, which is the right answer there too: no cell end lies inside.

## `colorArgMissing`

`\color` is a switch that also takes an argument, `\color{red}`, with an optional colour model first, `\color[RGB]{255,0,0}`. Before the change a bare `\color` cut its cell, looked for the argument, found none and stepped back, so a run of bare ones cost a cell scan each. `colorArgMissing` answers first: it skips blanks and an optional `[model]` that closes inside the cell, and reports whether a `{` group follows. The missing-argument error is returned before any cell is cut. The read of the argument after the cut then has no error branch of its own: `colorArgMissing` has already found the `{` inside the cell, and the cut cell holds it.

## `matched`

A `{` or `[` counts as the start of a group only when its closer lies inside the buffer being read. Before the change a sub-buffer cut at a cell end could start a group whose closer lay past the cut, and reading that group sliced past the buffer and crashed, as `\bf\sqrt[a&b]x` did. `GetNextToken`, `GetNextExpr` and `GetOptions` now ask `matched`, so such a bracket is read as a plain token.

## Commands that draw nothing

`\class{a}{}`, or `\mathrm` with nothing after it as the delimiter of `\left`, `\middle` or `\bigl`, produce no node. `ProcessCommand` and the three fence paths in `ParseTex` dereferenced that missing node and crashed. They now return or skip it, and the converter carries on with the rest of the formula.

A missing node still has to fill a script slot. `ParseTex` keeps `promotedProperties`, the script or `\over` role the next atom will take, until an atom arrives. Had it skipped the missing node and kept that role, `x^\class{a}{} y` would put `y` in the superscript. So when a pending role meets a missing node, `ParseTex` puts in an empty `mrow` carrying the role, as `x^{} y` does, and `y` stays on the line. The after version:

```go
if child == nil {
	if promotedProperties != 0 {
		empty := NewMMLNode("mrow")
		empty.Properties |= promotedProperties
		promotedProperties = 0
		siblings = append(siblings, empty)
	}
	continue
}
```

This covers every node that comes back missing, not only commands: an empty `matrix` in a script slot fills it the same way. On `x^\mathrm{} y`, which LaTeX accepts, the result is what LaTeX draws, an empty superscript with `y` on the line, where before `y` went into the superscript.

## Before and after

**Before:** a run of comment lines, or of bare `\color`, converted in time growing with the square of its length, through `TokenBuffer.Empty`, `fixFences` and the per-switch cell scan in `ProcessCommand`; malformed input such as `\bf\sqrt[a&b]x` or `x\bigr\mathrm` crashed, and gnoweb showed the source.

**After:** each run costs time in proportion to its length, through `last`, the fence-command check in `fixFences`, `colorArgMissing` and the `cellEnds` table shared from the root; those inputs convert without a crash, and a command drawing nothing in a script slot leaves the next atom on the line.

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test.

### 1. `hot` · The fence check in `fixFences`, [`gno.land/pkg/gnoweb/markdown/mathml/tokenize.go`](https://github.com/gnolang/gno/pull/6377/files#diff-0a86b2ef7f4c0cd16b59b787292e98ac793071187c792e328b5b450e5f360f86R647-R695)
Sends every token that is not a fence or `\big` command to the output before the blank-run skip.
**If wrong:** a `\left` or `\big` loses its delimiter, or a comment run is walked once per comment again.

### 2. `hot` · `last`, `cellEnd`, the table and `sub`, [`gno.land/pkg/gnoweb/markdown/mathml/tokenize.go`](https://github.com/gnolang/gno/pull/6377/files#diff-0a86b2ef7f4c0cd16b59b787292e98ac793071187c792e328b5b450e5f360f86R312-R353)
`Empty` is true once the position reaches `last`; `cellEnd` reads a distance from a table built once and sliced into every sub-buffer.
**If wrong:** a switch spans the wrong cell, or `Empty` ends a buffer early.

### 3. `hot` · `matched`, [`gno.land/pkg/gnoweb/markdown/mathml/tokenize.go`](https://github.com/gnolang/gno/pull/6377/files#diff-0a86b2ef7f4c0cd16b59b787292e98ac793071187c792e328b5b450e5f360f86R355-R360)
A `[` or `{` counts as a group only when its closer lies within the buffer.
**If wrong:** a sub-buffer cut through a group slices past its end and crashes.

### 4. `hot` · The bare `\color` shortcut and `colorArgMissing`, [`gno.land/pkg/gnoweb/markdown/mathml/commands.go`](https://github.com/gnolang/gno/pull/6377/files#diff-e00d7e18532389cb0ca1c98a580bda6dfa410ede42130ce86f1b2a455787f085R584-R598)
Returns the missing-argument error before the switch cuts its cell.
**If wrong:** a `\color` with an argument renders as an error, or a run of bare ones is quadratic again.

### 5. `warm` · The nil guards, [`gno.land/pkg/gnoweb/markdown/mathml/parse.go`](https://github.com/gnolang/gno/pull/6377/files#diff-68347990c0498360ca09607c0fd3a3d1b55d14229add4b92002742c27ae265d6R229-R232)
A fence whose delimiter is a command drawing nothing is skipped, as is `\class{a}{}`.
**If wrong:** that input crashes the converter again, and the page shows the formula as source.

### 6. `warm` · The empty script filler, [`gno.land/pkg/gnoweb/markdown/mathml/parse.go`](https://github.com/gnolang/gno/pull/6377/files#diff-68347990c0498360ca09607c0fd3a3d1b55d14229add4b92002742c27ae265d6R298-R308)
Gives a pending script or `\over` role to an empty `mrow` when the node that should take it is missing.
**If wrong:** in `x^\class{a}{} y`, the `y` is drawn as the superscript.

### 7. `cold` · The tests, [`gno.land/pkg/gnoweb/markdown/mathml/edgecases_test.go`](https://github.com/gnolang/gno/pull/6377/files#diff-bad9ba16d355ebdd42acccc074b55514171fb90517664d7730a3fb2d1f042cd3R114-R207)
Fence-and-comment cases, `TestCommentRunIsLinear`, `TestBareColorRunIsLinear`, `TestSwitchRunAllocations`, the two cell tests, `TestEmptyCommandFillsScript` and `TestMalformedInputDoesNotCrash`.
**If wrong:** a quadratic walk or a crash comes back and the suite stays green.

</details>

## Words used here

| Name | What it is |
| --- | --- |
| `fixFences` | The tokenizer pass that folds `\left`, `\middle`, `\right` and the `\big` family into the delimiter token after them. |
| `TokenBuffer.Empty` | True when every token from the buffer's position on is a space or a comment; it then moves the position to the end. |
| `last` | The index just past a buffer's last token that is neither a space nor a comment, set by `NewTokenBuffer`. |
| `cellEnd` | The index of the first `&` or line break from the buffer's position on, outside a group, or the buffer's length. |
| `cellEnds` | The table `cellEnd` reads: per position, the distance to that index, built once at the root and sliced by `sub`. |
| `colorArgMissing` | True when a `\color` has no `{argument}` in its cell, after an optional `[model]` closing there. |
| `matched` | True when a `[` or `{` opens a group whose closer lies within the buffer being read. |
| `promotedProperties` | In `ParseTex`, the superscript, subscript or `\over` role a `^`, `_` or `\over` hands to the next atom; reset once an atom takes it. |
