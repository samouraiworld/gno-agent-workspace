Target: https://github.com/gnolang/gno/issues/new
Status: draft, not posted. Searched every title and the full text of gnolang/gno issues and pull requests for faux block, frame per function, flatten blocks, stack frame, escape analysis, closure captures heap, block allocation per scope, redesign blocks and loopvar rename on 2026-11-25: no issue proposes this. Comments searched too for faux block, stack frame, flatten, escape analysis and heap item closure: no written proposal found; the user recalls the idea from an earlier conversation, place unknown. Nearest misses: #4588, closed, a gas charge by frame size; #4098, where a comment proposes a frame depth beside the block depth for `goto`. The loop-variable rename, #6209 and #6506, is left out: per-iteration variables still need their own storage in one block per function, so the proposal does not settle it.

## Title

gnovm: lay out a function's variables in one block, each scope after its parent's

## Body

## Description

Opening this to start a discussion.

The VM gives each scope of a function its own block of variable slots, and the preprocessor turns every name into a block depth and a slot. A `switch` or an `if` is the exception: it shares one block with its cases. Three pieces of code make that work: [`copyFromFauxBlock`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L4272) copies the names the init statement declares into each case, the depth count [skips the case](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L6253), and `fallthrough` [cuts the block back](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/op_exec.go#L767) before the next case. #6058 fixes a bug that the copy causes: a case redeclaring an init name panics, or overwrites the outer variable when the types match.

A closure already reads each outer local variable it uses from a heap item of its own, through [`addHeapCapture`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L3853), held in [`FuncValue.Captures`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/values.go#L668), and not from the enclosing block's slot.

Proposal: give each function call one block, each scope's slots starting where its parent's end and sibling scopes reusing the same slots, so a name compiles to a slot number and no case needs a copy.

- [ ] A `switch` or `if` case holds only its own names, starting at the slot right after the statement's, with no copies.
- [ ] Every scope of a function shares the function's block.
