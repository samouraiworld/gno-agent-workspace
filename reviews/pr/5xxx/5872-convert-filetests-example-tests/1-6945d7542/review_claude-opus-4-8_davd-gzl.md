# PR #5872: chore: Convert some filetests to Example tests

URL: https://github.com/gnolang/gno/pull/5872
Author: jefft0 | Base: master | Files: 6 | +386 -224
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `6945d7542` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5872 6945d7542`

**TL;DR:** Converts five package filetests (`*_filetest.gno`, run as standalone `main` programs with a golden `// Output:`) into Go-style `ExampleXxx()` tests that live in the package and print to compare against `// Output:`, and adds new examples for `bptree`. One converted example in `p/moul/md` is malformed so it silently never runs.

**Verdict: REQUEST CHANGES** — `ExampleBlocks` in the md conversion never executes (misplaced `// Output:`), dropping all `Blockquote` coverage; the permissions `README.md` still embeds the old filetest by a now-deleted path.

## Summary
The PR moves filetests into their packages as example tests for five packages and adds three new bptree examples. Four of the five conversions are faithful: the same calls, the same golden output, and they run and pass. The md conversion splits one `main` filetest into seven `Example` functions, but `ExampleBlocks` places a stray `// This is a paragraph.` comment line above its `// Output:` marker. The example runner (Go `go/doc` semantics) only recognizes an output directive when `// Output:` begins the function's final comment group, so it treats `ExampleBlocks` as outputless and skips running it entirely. `Blockquote` is exercised nowhere else, so its coverage goes from tested (in the deleted filetest) to zero. Separately, the permissions `README.md` embeds the converted file via `[embedmd]:# (filetests/readme_filetest.gno go)`, a path this PR deleted, so the README now shows stale `package main` code and `embedmd -w` would fail.

## Glossary
- filetest: a `*_filetest.gno` file the VM runs and asserts against golden directives (`// Output:`, `// Error:`, ...).
- example test: a `ExampleXxx()` function whose trailing `// Output:` comment is compared to what it prints; `// Output:` must begin the function's last comment group or the runner skips execution.

## Warnings (should fix)
- **[converted example silently never runs]** [`examples/gno.land/p/moul/md/example_test.gno:68-72`](https://github.com/gnolang/gno/blob/6945d7542/examples/gno.land/p/moul/md/example_test.gno#L68-L72) · [↗](../../../../../.worktrees/gno-review-5872/examples/gno.land/p/moul/md/example_test.gno#L68-L72) — `ExampleBlocks` is skipped because `// Output:` is not the first line of its final comment group, so `Blockquote` is left untested.
  <details><summary>details</summary>

  Lines 68-69 (`// This is a paragraph.` then a blank `//`) sit above `// Output:` at line 70. The runner reads the function's last comment group, and recognizes an output directive only when that group begins with `Output:`; here it begins with `This is a paragraph.`, so the group is not an output directive and the example compiles but does not run. A `gno test -v` of the package runs the other six examples (`ExampleHeaders`, `ExampleStyles`, `ExampleLists`, `ExampleCode`, `ExampleReferences`, `ExampleColumns`) and never lists `ExampleBlocks`. `Blockquote` appears in no other test in the package ([`md_test.gno:33`](https://github.com/gnolang/gno/blob/6945d7542/examples/gno.land/p/moul/md/md_test.gno#L33) · [↗](../../../../../.worktrees/gno-review-5872/examples/gno.land/p/moul/md/md_test.gno#L33) covers `Paragraph` but not `Blockquote`), and the deleted `filetests/z1_filetest.gno` did assert it, so this conversion drops `Blockquote` coverage. The embedded text is also wrong on its own terms: the real output of the two `println` calls is `This is a paragraph.` followed by blank lines and then the blockquote, not the blockquote alone. Fix: make `// Output:` the first line of the final comment group and assert the full printed output, including the paragraph line and the blank lines between it and the blockquote, so the example actually runs.

  **Repro:**
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

- **[README embeds a file the PR deleted]** [`examples/gno.land/p/gnoland/boards/exts/permissions/README.md:13`](https://github.com/gnolang/gno/blob/6945d7542/examples/gno.land/p/gnoland/boards/exts/permissions/README.md?plain=1#L13) · [↗](../../../../../.worktrees/gno-review-5872/examples/gno.land/p/gnoland/boards/exts/permissions/README.md#L13) — the embedmd directive points at the renamed `filetests/readme_filetest.gno`, so the README shows stale `package main` code and regeneration breaks.
  <details><summary>details</summary>

  `README.md:13` carries `[embedmd]:# (filetests/readme_filetest.gno go)`, and the fenced block below it is the pre-PR `package main` / `func main()` source. This PR renames that file to `example_test.gno` (now `package permissions` / `func ExamplePermission()`) and removes the `filetests/` copy, so the embed path no longer exists and the displayed code contradicts the actual example. `embedmd -w` over this README would fail on the missing file. No CI job catches it: embedmd runs only over `docs/` ([`docs/Makefile:5`](https://github.com/gnolang/gno/blob/6945d7542/docs/Makefile#L5) · [↗](../../../../../.worktrees/gno-review-5872/docs/Makefile#L5)), and the docs linter ([`misc/docs/tools/linter/links.go:28`](https://github.com/gnolang/gno/blob/6945d7542/misc/docs/tools/linter/links.go#L28) · [↗](../../../../../.worktrees/gno-review-5872/misc/docs/tools/linter/links.go#L28)) checks embedmd links only in `docs/`, not example READMEs. Fix: repoint the directive at `example_test.gno` (an `ExamplePermission` selector) and regenerate, or update the block by hand. README.md is not in the PR diff, so this is reported in the review body, not as an inline comment.
  </details>

## Critical (must fix)
None.

## Nits
None.

## Missing Tests
- **[coverage regression]** [`examples/gno.land/p/moul/md/example_test.gno:64-73`](https://github.com/gnolang/gno/blob/6945d7542/examples/gno.land/p/moul/md/example_test.gno#L64-L73) · [↗](../../../../../.worktrees/gno-review-5872/examples/gno.land/p/moul/md/example_test.gno#L64-L73) — `Blockquote` has no executing test once `ExampleBlocks` is skipped; fixing the example above restores it.
  <details><summary>details</summary>

  Covered by the first Warning. Flagged separately so the coverage loss is explicit: the deleted filetest asserted `Blockquote`, `md_test.gno` does not, and the replacement example does not run.
  </details>

## Suggestions
None.

## Open questions
- The avlhelpers conversion switches from an external test package (`// PKGPATH: gno.land/p/test`, `package test`) to an in-package white-box example (`package avlhelpers`). Fine for an example test; not posted, no behavior or coverage change.
