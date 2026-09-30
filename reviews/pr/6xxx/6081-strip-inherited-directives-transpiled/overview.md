# Keeping Gno source directives out of transpiled Go

PR: [gnolang/gno#6081](https://github.com/gnolang/gno/pull/6081). Written by claude-opus-5-5, deep review.

## TLDR

`gno tool transpile` copies every comment of a `.gno` file into the `.go` file it writes. Some comments are instructions to the Go toolchain, so the change rewrites those into an inert marker before printing, keeping one output line per source line.

## What it is for

The transpiled file is Go, and Go tools act on certain comments: `//go:build` sets build constraints, `//go:generate` is a shell command for `go generate`, `//nolint` hides linter findings, and `//line` rewrites the positions the compiler reports. A Gno contract means none of these, but once copied into Go they take effect on whoever builds, generates or lints the output.

## How it works today

Before the change, the transpiler parses the `.gno` file with comments, rewrites imports, then prints the tree after a header it writes itself: a `Code generated` line, `//go:build gno` and `//line <file>:1:1`. Every source comment is printed as written, so a source carrying `//go:build ignore` yields a file `go build` rejects with `multiple //go:build comments`, as the pull request reports, and a `//go:generate` line becomes a command. The `//line <file>:1:1` header maps every output line back to the same line of the `.gno` file, so it only stays true while the body keeps the source's line count.

## What the change does

After import rewriting, every comment in the tree passes through `neutralizeDirective`:

| Comment in the source | After the change |
| --- | --- |
| a Go directive, `//go:build`, `//go:generate`, `//line`, `//export`, `//tool:name` | `//gno:removed-directive` |
| `//nolint`, `// nolint:...`, legacy `// +build` | `// removed directive` |
| a `/*line ...*/` block | `/*` and `*/` with the same number of lines, blank between |
| a line starting `//go:generate` inside a `/* */` block, after the indentation the printer strips | that line's directive replaced by `//gno:removed-directive` |
| any other comment | unchanged |

The replacements keep the comment's line, so the `//line` header stays accurate, and keep whether Go classifies the comment as a directive, because the printer lays out directive and prose lines in a doc comment differently. `IsDirectiveComment` in `gnovm/pkg/gnolang` holds the classification rule, a copy of the unexported `go/ast.isDirective` with a test comparing the two.

## Concepts

- **Directive**: a comment Go tools act on, recognised by `go/ast` as `//` followed by a lowercase `tool:name` with no space, or `//line`, `//extern`, `//export`.
- **Doc position**: a comment directly above a declaration; `go/printer` reformats these, which is where blanking a comment can make its line disappear.
- **Parenthesized import block**: `import ( ... )`; its presence makes `format.Node` re-parse the file, routing doc comments through that reformatting.

## Review files

[Review files for this PR](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6081-strip-inherited-directives-transpiled)
