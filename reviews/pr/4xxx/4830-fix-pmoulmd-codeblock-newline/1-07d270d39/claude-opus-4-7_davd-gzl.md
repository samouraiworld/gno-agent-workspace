# PR #4830: fix(p/moul/md): add newline to codeblocks

URL: https://github.com/gnolang/gno/pull/4830
Author: vikbez | Base: master | Files: 3 | +8 -6
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — small correctness fix to a markdown helper; appends a trailing `\n` to `CodeBlock` / `LanguageCodeBlock` so concatenated content lands outside the fenced block. Already approved by [@jefft0](https://github.com/gnolang/gno/pull/4830#pullrequestreview-3338841029). Two non-blocking caveats: PR is in `CONFLICTING` state against master (rebase needed after #5418 added `EscapeURL` and #5104 moved filetests into `filetests/`), and the `/r/docs/moul_md` examples flagged in the existing reviewer comment still warrant an update.

## Summary

Without a trailing newline, `md.CodeBlock("x") + "tail"` produces ` ```\nx\n```tail ` — the closing fence shares its line with following content, so GitHub-flavored and CommonMark parsers fail to close the code block, rendering `tail` inside the block. Adding `\n` after the closing fence makes the helpers consistent with every other block-level helper in the file (`H1`...`H6`, `BulletList`, `OrderedList`, `Blockquote`, `HorizontalRule`, `Paragraph`, `CollapsibleSection` — all terminate with `\n`). Tests and the filetest `// Output:` block are updated in lockstep.

## Fix

Before: [`md.gno:168`](../../../../../.worktrees/gno-review-4830/examples/gno.land/p/moul/md/md.gno#L168) returned ` "```\n" + content + "\n```" ` with no trailing newline, leaving the closing fence to merge with whatever the caller appended. After: returns ` "```\n" + content + "\n```\n" ` so the closing fence always sits on its own line. The doc-comment examples and `md_test.gno` golden strings are updated to match; `z1_filetest.gno` gains the two blank `//` lines that the new trailing `\n` produces in `println` output.

## Blast radius

Eight call sites in `examples/` exercise the helpers. None break; the worst case is a single extra blank line between a code block and the next element:

| Caller | Pattern | Before | After |
|---|---|---|---|
| [`p/lou/ascii/ascii.gno:129,173,237`](../../../../../.worktrees/gno-review-4830/examples/gno.land/p/lou/ascii/ascii.gno#L129) | `md.CodeBlock(...) + "\n"` | `...```\n` | `...```\n\n` |
| [`r/docs/moul_md/moul_md.gno:84,102,...`](../../../../../.worktrees/gno-review-4830/examples/gno.land/r/docs/moul_md/moul_md.gno#L84) | `md.LanguageCodeBlock(...) + "\nResult:\n"` | `...```\nResult:` | `...```\n\nResult:` |
| [`r/matijamarjanovic/tokenhub/render.gno:78,93`](../../../../../.worktrees/gno-review-4830/examples/gno.land/r/matijamarjanovic/tokenhub/render.gno#L78) | `out += md.LanguageCodeBlock(...)` then `out += "\n"` | `...```\n` | `...```\n\n` |
| [`r/sacha/home/home.gno:56`](../../../../../.worktrees/gno-review-4830/examples/gno.land/r/sacha/home/home.gno#L56) | `out += md.CodeBlock(art)` | `...```` | `...```\n` |
| [`r/sunspirit/md/md.gno:86,101`](../../../../../.worktrees/gno-review-4830/examples/gno.land/r/sunspirit/md/md.gno#L86) | inside a builder `.Add(...)` chain | `...```` | `...```\n` |

A blank line between blocks is the correct rendering, not a regression.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [merge conflict] PR is `CONFLICTING` against master. Master changed [`CodeBlock`](../../../../../.worktrees/gno-review-4830/examples/gno.land/p/moul/md/md.gno#L167) / [`LanguageCodeBlock`](../../../../../.worktrees/gno-review-4830/examples/gno.land/p/moul/md/md.gno#L173) lines via [#5418](https://github.com/gnolang/gno/pull/5418) (markdown-injection fix; introduced `EscapeURL`, did not touch the closing-fence string), and [#5104](https://github.com/gnolang/gno/pull/5104) moved `z1_filetest.gno` into `filetests/`. Rebase is mechanical — the fix only touches the closing-fence string literal and the test golden, both untouched by #5418.
- [docs example drift] [@davd-gzl](https://github.com/gnolang/gno/pull/4830#issuecomment-3367095170) already flagged: `/r/docs/moul_md#simple-code-block` and other doc realms show the old return shape. After this lands, those examples are wrong. Cheapest path is one follow-up grepping `\\nfoo\\n```"` in doc strings under `examples/gno.land/r/docs/`.

## Missing Tests

None — `md_test.gno` already covers both helpers; the goldens are updated in this diff.

## Suggestions

None.

## Questions for Author

None.
