# Unique BlockNode Locations for else-if
Written by claude-opus-5-5.

## TLDR
Every block of Gno code gets a `Location`, and the store finds blocks by it. An `else if` gave two blocks the same `Location`, so one silently replaced the other in the store's node cache. The change makes every `Location` unique and adds a check that panics on a duplicate.

## What it is for
The Gno VM turns each source file into a tree of nodes. Nodes that open a scope, a function, an `if`, a `for` body, are `BlockNode`s. Each carries a `Location`: package path, file name, and a span (start, end, and a `Num` tie-breaker). `SetBlockNode` caches block nodes keyed by that `Location`, and a stored closure or block finds its source node again through it. A planned change (gnolang/gno#6060) will also derive local type identities from it.

## How it works today
Before the change, Go's parser represents `else if b {}` as an `Else` field holding a nested `IfStmt`. The Gno converter wraps that nested `IfStmt` in a synthetic else block and copies the nested statement's span onto the wrapper, so both get the same span and, after `setNodeLocations`, the same `Location`. Across the repository's examples, stdlibs and filetests that is 254 duplicate pairs. The checker meant to catch this, `checkNodeLinesLocations`, was an empty stub.

## What the change does
After the change:

- the converter marks the else-if wrapper with `Num = 1`;
- `setNodeLocations` keeps a counter per (start, end) pair and gives each node sharing a span the next free `Num`;
- `checkNodeLinesLocations` walks the tree and panics on a repeated `Location`, and on a path or file name that does not match the caller's.

Before and after, for `if a {} else if b {}` with no final else:

| Node | Num before | Num after |
| --- | --- | --- |
| else wrapper `IfCaseStmt` | 0 | 1 |
| nested `IfStmt` | 0 (same as the wrapper) | 2 |
| empty else of the nested `IfStmt` | 1 | 3 |

Only `IfStmt` and `IfCaseStmt` Locations move. Function and file Locations, the ones stored closures and local types refer to, stay the same.

## Concepts
- **Span**: a node's start and end position in the file, plus `Num`, which tells apart nodes with the same start and end.
- **Location**: package path, file name and span; the key of the block-node cache.
- **else-if wrapper**: the else branch Gno builds around a nested `IfStmt`; it has no source text of its own, so it shares the nested statement's span.
