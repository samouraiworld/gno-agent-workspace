# Switch and if clauses may shadow their init names
Written by claude-opus-5-5.
PR: [gnolang/gno#6058](https://github.com/gnolang/gno/pull/6058)

## TLDR

In Go, the body of a `switch` or `if` clause can declare a variable with the same name as one declared in the statement's init, `switch v := 1; v { case 1: v := "s" }`. Gno either panicked on that code or let the inner declaration overwrite the outer one. This change gives the inner declaration a slot of its own, so both variables live side by side and every use reads the right one, as Go does.

## Faux blocks: why a clause holds copies of the init names

A clause of an `if` or a `switch` does not get a runtime block of its own. It shares one block with its statement, so the clause's static block starts with copies of the statement's init names, in the same order, and its own declarations follow. `numFauxCopiedNames()` returns how many leading slots are those copies ([`nodes.go`](https://github.com/gnolang/gno/pull/6058/files#diff-de40464561985f3fa10d53253bc3b01cdaa01b0e77ce52a87e2ed169f9359065R2345)); every slot at or past that index belongs to the clause. Sharing one flat block keeps `fallthrough`, `break` and the block pool simple, and it is why a shadow cannot simply live in a child block.

## A second slot for the same name

Before the change, a clause declaring `v` when `v` was already a copied init name found the existing slot and reused it, so `v := "s"` retyped the outer `v`. `StaticBlock.Reserve` ([`nodes.go`](https://github.com/gnolang/gno/pull/6058/files#diff-de40464561985f3fa10d53253bc3b01cdaa01b0e77ce52a87e2ed169f9359065R2309-R2332)) now appends a new slot when the only existing slot for the name is a copy (`idx < numFauxCopiedNames()`). A name the clause already owns returns early, and that early return is also what keeps a clause from redeclaring a type switch variable, since that variable sits at exactly `numFauxCopiedNames()`. The append is skipped when the declaration recorded on the copy's `NameSource` is the one reserving again, because the copy loop re-runs on a block node that was already set up.

## Last slot wins

With two slots for one name, a lookup has to pick one. `GetLocalIndex` scans from the end and `buildNameIndex`, the map used past 32 names, keeps the later index, so a name resolves to the shadow. Outside a faux clause a block never holds the same name twice, so nothing else changes.

## Uses before the shadow read the outer name

In `case 1: println(v); v := "s"`, the `println(v)` must read the init `v`. The shadow's slot is reserved with no type until preprocessing reaches its declaration, and name resolution already treats a reserved but untyped slot as "not declared here yet" and walks up to the parent block. That is the same rule that makes `println(x); x := 2` work in an ordinary nested block.

## Copies are written by index

The copy step and the type switch variable used to write by name, which would now find the shadow. `defineFauxCopy` ([`nodes.go`](https://github.com/gnolang/gno/pull/6058/files#diff-de40464561985f3fa10d53253bc3b01cdaa01b0e77ce52a87e2ed169f9359065R2355)) writes the slot by its index and panics, in every build, when that slot does not hold the expected name, so a misplaced boundary fails loudly instead of giving a variable the wrong type. Its two callers are the copy loop and `defineSwitchVar` in [`preprocess.go`](https://github.com/gnolang/gno/pull/6058/files#diff-ebdb0494324ba82339ca88219105656407e3f3a49240cf32c24c9a266d31e0f4R1054-R1057), which writes the type switch variable at `numFauxCopiedNames()`.

## Words used here

| Name | What it is |
| --- | --- |
| faux block | A clause's static block that shares its runtime block with the parent `if` or `switch`; `fauxChildBlockNode` is true for it. |
| `numFauxCopiedNames()` | The number of leading names a faux clause copied from its parent; 0 for every other block. |
| `NameSource` | The declaration a slot came from: origin node, kind (`NSDefine`, `NSTypeSwitch`, ...) and index. |
| `nameIndex` | The name-to-slot map a block builds once it holds more than 32 names (`nameIndexThreshold`). |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `cold` a test.

### 1. `hot` · The append in Reserve, [`gnovm/pkg/gnolang/nodes.go`](https://github.com/gnolang/gno/pull/6058/files#diff-de40464561985f3fa10d53253bc3b01cdaa01b0e77ce52a87e2ed169f9359065R2309-R2332)
Decides when a clause declaration gets a second slot instead of reusing the copy.
**If wrong:** a shadow retypes the outer variable, or the copy loop appends duplicates on every pass.

### 2. `hot` · Writing a copy by index, [`gnovm/pkg/gnolang/nodes.go`](https://github.com/gnolang/gno/pull/6058/files#diff-de40464561985f3fa10d53253bc3b01cdaa01b0e77ce52a87e2ed169f9359065R2355)
Writes a copied name, or the type switch variable, into its slot and checks the name.
**If wrong:** a copied name lands in the shadow's slot with the wrong type.

### 3. `cold` · The shadowing tests, [`gnovm/tests/files/switch52.gno`](https://github.com/gnolang/gno/pull/6058/files#diff-51e9357051c64cb80478777b4f743a8187dfa37223eba6b1d71251ec91d2696eR1)
`:=`, `var` and `const` shadows, a use and an assignment before the shadow, and a nested switch; `switch53.gno`, `if9.gno` and `typeswitch11.gno` cover closures, `fallthrough`, `if` and type switches.
**If wrong:** a regression in any of the shapes above goes unnoticed.

</details>
