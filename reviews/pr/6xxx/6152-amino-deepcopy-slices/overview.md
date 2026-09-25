# amino.DeepCopy handing back the source's slices instead of copies
claude-opus-5-5

## TLDR

`amino.DeepCopy` built a fresh copy of every slice it met and then returned the
original slice anyway, so a "deep" copy shared its slices with the value it was
copied from. The change returns the copy, and keeps a nil slice nil.

## What it is for

`amino.DeepCopy` in `tm2/pkg/amino/deep_copy.go` copies any value by walking it
with reflection. Tendermint2 uses it wherever a caller must not be able to change
shared state through what it was handed: `Header.Copy()`, `sdk.Context.BlockHeader()`
and `ConsensusParams()`, `ConsensusParams.Update`, and the consensus state's
`GetRoundStateDeepCopy` and `GetConfigDeepCopy`, which the RPC serves.

## How it works today

Before the change, both slice branches of `_deepCopy` allocate `cpy`, fill it,
then run `dst.Set(src)`: the filled copy is dropped. A header copy therefore shares
`LastCommitHash`, `AppHash` and every other byte slice with the live header.

## What the change does

After the change, each branch runs `dst.Set(cpy)`, and a nil slice is set back as
nil before any allocation, since a nil slice reached through a pointer skips the nil
check `deepCopy` does at its top.

| Case, after the change | Result |
| --- | --- |
| `[]int{1,2,3}`, then the source is written | the copy keeps its values |
| `[]byte(nil)` behind a pointer | the copy is nil, as before |
| `[]byte{}` | the copy is empty and non-nil, as before |

The cost is unchanged: the merge base already allocated and filled the same copy,
then discarded it. Amino encodes a nil and an empty slice the same way, so a copied
header hashes like its source either way.

## Concepts

- **Aliasing**: two slice headers pointing at one backing array, so a write through
  either shows in both.
- **`deepCopy` and `_deepCopy`**: the first checks for nil and for a `DeepCopy` or
  `MarshalAmino` method on the type, the second walks the value by its kind.
