Target: https://github.com/gnolang/gno/issues/new
Status: draft, not posted. Searched every title and the full text of gnolang/gno issues and pull requests for faux block, frame per function, flatten blocks, stack frame, escape analysis, closure captures heap, block allocation per scope, redesign blocks and loopvar rename on 2026-11-25: no issue proposes this. Comments searched too for faux block, stack frame, flatten, escape analysis and heap item closure: no written proposal found; the user recalls the idea from an earlier conversation, place unknown. Nearest misses: #4588, closed, a gas charge by frame size; #4098, where a comment proposes a frame depth beside the block depth for `goto`.

## Title

gnovm: lay out a function's variables in one block, each scope after its parent's

## Body

## Description

Opening this to start a discussion.

The VM gives each scope of a function its own block of variable slots, and the preprocessor turns every name into a block depth and a slot. Two pieces of code exist to reconcile that block chain with Go's scoping, and three open items trace to them:

| Code | What it does | Open items |
|---|---|---|
| [`copyFromFauxBlock`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L4272), the depth count that [skips a case](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L6253) and the [cut on `fallthrough`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/op_exec.go#L767) | let a `switch` or `if` share one block with its cases by copying the names its init statement declares into each case | #6058: a case redeclaring an init name panics, or overwrites the outer variable when the types match |
| [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L256-L271) | renames a `for` loop's variables to `x.loopvar` so each iteration gets its own | #6209: four loop shapes Go accepts are refused. #6506: a func literal's result type inside a loop reads the outer `x`, and the program prints 3 where Go prints 2 |

A closure already copies each outer local variable it uses into a heap item of its own, through [`addHeapCapture`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L3853), held in [`FuncValue.Captures`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/values.go#L668).

Proposal: give each function call one block, each scope's slots starting where its parent's end and sibling scopes reusing the same slots, so a name compiles to a slot number and neither piece of code above has a mapping left to keep.

- [ ] A `switch` or `if` case holds only its own names, numbered after the statement's, with no copies.
- [ ] Every scope of a function shares the function's block.
