Target: https://github.com/gnolang/gno/issues/new
Status: draft, not posted. Searched every title and the full text of gnolang/gno issues and pull requests for faux block, frame per function, flatten blocks, stack frame, escape analysis, closure captures heap, block allocation per scope, redesign blocks and loopvar rename on 2026-11-25: no issue proposes this. Comments searched too for faux block, stack frame, flatten, escape analysis and heap item closure: no written proposal found; the user recalls the idea from an earlier conversation, place unknown. Nearest misses: #4588, closed, a gas charge by frame size; #4098, where a comment proposes a frame depth beside the block depth for `goto`. The loop-variable rename, #6209 and #6506, is named as left open rather than as solved.

## Title

gnovm: refactor variable layout to one block per function, each scope after its parent's

## Body

## Description

Opening this to start a discussion.

The VM gives each scope of a function its own block of variable slots, and the preprocessor turns every name into a block depth and a slot. A `switch` or an `if` is the exception: it shares one block with its cases. Three pieces of code make that work: [`copyFromFauxBlock`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L4272) copies the names the init statement declares into each case, the depth count [skips the case](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L6253), and `fallthrough` [cuts the block back](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/op_exec.go#L767) before the next case. #6058 fixes a bug that the copy causes: a case redeclaring an init name panics, or overwrites the outer variable when the types match.

A variable that a closure captures or whose address is taken already lives in a heap item. [`addHeapCapture`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L3853) records each capture on the function literal, and at run time the closure holds the variable's heap item in [`FuncValue.Captures`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/values.go#L668). [`&v` marks `v`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L3737-L3746) for the same treatment. A closure or a pointer therefore reads the heap item, not the block slot.

Proposal, as a refactor: once #6058 lands nothing here is broken, so this is not needed for correctness, but I think it is a better design. Give each function call one block, each scope's slots starting where its parent's end and sibling scopes reusing the same slots, so a name compiles to a slot number and no case needs a copy. A scope that reuses a slot after another has left it cannot overwrite a variable a closure or a pointer still holds, since those read the heap item.

- [ ] A `switch` or `if` case holds only its own names, starting at the slot right after the statement's, with no copies.
- [ ] Every scope of a function shares the function's block.

Left open: how this layout handles the `.loopvar` rename, which gives a `for` loop's variables fresh storage each iteration; #6209 and #6506 are about that rename.
