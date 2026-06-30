# Review: PR #5872
Event: REQUEST_CHANGES

## Body
The permissions [README.md](https://github.com/gnolang/gno/blob/6945d7542/examples/gno.land/p/gnoland/boards/exts/permissions/README.md?plain=1#L13) [↗](../../../../../.worktrees/gno-review-5872/examples/gno.land/p/gnoland/boards/exts/permissions/README.md#L13) still embeds the deleted `filetests/readme_filetest.gno` through its `[embedmd]` directive. Embedmd regeneration fails on the missing file. Repoint the directive at the new `example_test.gno` and regenerate.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5872-convert-filetests-example-tests/1-6945d7542/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/p/moul/md/example_test.gno:68-72 [↗](../../../../../.worktrees/gno-review-5872/examples/gno.land/p/moul/md/example_test.gno#L68)
`ExampleBlocks` never runs: `// This is a paragraph.` sits above `// Output:`, so the runner finds no output directive and skips execution, leaving `Blockquote` untested. Make `// Output:` the first line of the final comment group and assert the full output.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5872 -R gnolang/gno
cd examples/gno.land/p/moul/md
echo "defined:"; grep -c '^func Example' example_test.gno
echo "executed:"; go run ../../../../../gnovm/cmd/gno test -v . 2>&1 | grep -c '=== RUN   Example'
go run ../../../../../gnovm/cmd/gno test -v . 2>&1 | grep '=== RUN   Example'
```
```
defined:
7
executed:
6
=== RUN   ExampleHeaders
=== RUN   ExampleStyles
=== RUN   ExampleLists
=== RUN   ExampleCode
=== RUN   ExampleReferences
=== RUN   ExampleColumns
```
</details>
