Target: https://github.com/gnolang/gno/issues/6529
Status: posted 2026-11-26. Searched every title and the full text of gnolang/gno issues and pull requests for faux block, frame per function, flatten blocks, stack frame, escape analysis, closure captures heap, block allocation per scope, redesign blocks and loopvar rename on 2026-11-25: no issue proposes this. Comments searched too for faux block, stack frame, flatten, escape analysis and heap item closure: no written proposal found; the user recalls the idea from an earlier conversation, place unknown. Nearest misses: #4588, closed, a gas charge by frame size; #4098, where a comment proposes a frame depth beside the block depth for `goto`. The loop-variable rename, #6209 and #6506, is named as left open rather than as solved.

## Title

gnovm: refactor to one block of variables per function

## Body

## Description

Opening this to start a discussion.

Each scope in a function gets its own block of slots, and every name compiles to a block depth and a slot. A `switch` or `if` instead shares one block with its cases by [copying](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L4272) its init names into each case, and that copy is what #6058 had to work around.

Proposal, a refactor: it is not needed for correctness, but I think it is a better design. Give each function call one block, each scope's slots starting where its parent's end and sibling scopes reusing them. Every name then compiles to a slot, and no case needs a copy.

Reusing a slot is safe because a variable that a closure captures or whose address is taken already lives in a heap item, which closures reach through [`addHeapCapture`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L3853) and pointers through [`&v`](https://github.com/gnolang/gno/blob/ed1a7c859f0f083ccf458ba22c4b053230f67236/gnovm/pkg/gnolang/preprocess.go#L3737-L3746), never through the slot.

- [ ] A `switch` or `if` case keeps only its own names, starting after the statement's.
- [ ] Every scope of a function shares the function's block.

Left open: the `.loopvar` rename that gives each loop iteration fresh variables, which #6209 and #6506 are about.
