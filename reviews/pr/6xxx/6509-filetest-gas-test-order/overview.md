# `runFiletest` shares one allocator with its store
Written by claude-opus-5-5.
PR: [gnolang/gno#6509](https://github.com/gnolang/gno/pull/6509)

## TLDR

A filetest's recorded gas could change depending on which tests ran before it in the same process, so a test passed in the full suite and failed when run alone. The change makes each filetest count its memory in one place, the way the chain does, so the number is the same however the tests are ordered. Nineteen gas goldens take new values, and chain gas is untouched.

## The allocator

`Allocator` (`gnovm/pkg/gnolang/alloc.go`) counts the bytes a run allocates, refuses past `maxBytes`, and, once a machine installs its gas meter on it, charges gas for each allocation. A `Machine` holds one in `m.Alloc`, and a store holds one too, reached through `GetAllocator`. The store's allocator is the one that counts what the store itself allocates: objects loaded from the database, and the `FuncValue` copies made while a generic builtin such as `make` is specialized during preprocessing. Whatever lands on an allocator with no gas meter is counted for `MAXALLOC` and never charged as gas.

## How a store and a machine end up sharing one

`NewMachineWithOptions` puts the machine's allocator on its store only when the store has none (`machine.go`, `store.GetAllocator() == nil`). `BeginTransaction` gives the transaction store `ds.alloc.Fork().Reset()`: a copy of the parent's allocator with the same cap, zero bytes and no gas meter, or nil when the parent has none. In the keeper both sides are built from one allocator on purpose, so a transaction's store-side allocations are charged to the transaction.

## Why the old filetest setup drifted

The root test store from `StoreWithOptions` started with no allocator. A filetest run on a fresh store therefore forked nil, and its machine installed its own metered allocator on the transaction store: preprocessing was charged. The first stdlib load, though, ran a machine on the root store and left that machine's allocator there for good. Every later transaction forked it, got an allocator with no gas meter, and the filetest machine kept a separate one: preprocessing was no longer charged. So `gas/small_string_cmp.gno` read 13130 alone and 13017 after other tests.

## How the change fixes it

**The root store gets its own allocator.** `StoreWithOptions` passes `gno.NewAllocator(math.MaxInt64)` to `gno.NewStore`, so no stdlib load can install one later and every run sees the same root.

**One allocator per filetest.** `runFiletest` builds `gno.NewAllocator(maxAlloc)` once, sets it on the transaction store with `SetAllocator`, and hands the same value to the machine as `Alloc`. Preprocessing allocations are then charged gas and counted against `MAXALLOC`, whatever ran before.

**The goldens follow.** Nineteen `gas/` filetests take the charged values, and `slice_alloc.gno`'s largest `n` under its cap drops from 12499840 to 12499838, since preprocessing now takes part of the budget.

## Before and after

**Before:** a filetest's gas and allocation count depended on whether the pooled store had already loaded a stdlib, through the allocator the first machine left on the root store.

**After:** each filetest's store and machine share one allocator built in `runFiletest`, and `TestFiletestGasIndependentOfStoreHistory` runs one program before and after a `strings` load and requires the same gas.

## What a user notices

- Anyone running one gas filetest alone gets the golden value the full suite gets.
- `gno run` and `gno repl` build their store through `StoreWithOptions` too, so their machines no longer install their allocator on that store.

## Not fixed

- Package tests in `gnovm/pkg/test/test.go` still run the machine on a different allocator from its store, as the description says.
- A realm filetest is preprocessed on a second transaction store that `runTest` opens, whose allocator is a gasless fork, so its preprocessing stays uncharged.
- Filetests do not install the keeper's metered preprocess allocator, so code evaluated statically while preprocessing is still undercharged.

## Words used here

| Name | What it is |
| --- | --- |
| `Allocator` | Byte counter with a cap; charges gas per allocation once a gas meter is installed on it. |
| `Fork` | Copies an allocator's cap and byte count, without its gas meter or GC hook. |
| `BeginTransaction` | Opens a transaction store over a parent store, its allocator a reset fork of the parent's. |
| `StoreWithOptions` | Builds the root test store used by filetests, `gno test`, `gno run` and `gno repl`. |
| `MAXALLOC` | Filetest directive capping allocated bytes; 0 or absent means `math.MaxInt64`. |
| `// Gas:` | Filetest golden holding the gas the run consumed. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file.

### 1. `cold` · One allocator per filetest, [`gnovm/pkg/test/filetest.go`](https://github.com/gnolang/gno/pull/6509/files#diff-3c90bc91debf24adcbfadff251a8a824c2cb6dad01b7551d7b2c89198d929af1R81-R90)
Builds the filetest's allocator once and gives it to both the transaction store and the machine.
**If wrong:** preprocessing gas depends again on what ran earlier on the store.

### 2. `cold` · The root store's allocator, [`gnovm/pkg/test/imports.go`](https://github.com/gnolang/gno/pull/6509/files#diff-86d4dfabace87a9cf547d35ee19d1ff84409b7a536e48bfe161f1b17c8b14bc9R246-R248)
Gives the root test store an allocator at construction, so no machine replaces it.
**If wrong:** the first stdlib load leaves its machine's allocator on the root store.

### 3. `cold` · The order test, [`gnovm/pkg/test/filetest_test.go`](https://github.com/gnolang/gno/pull/6509/files#diff-47712c9a76a4043e29a86c5419a1612364546727d117f889b983d3414d2be62bR13-R39)
Runs one program before and after a stdlib load and requires the same allocator and the same gas.
**If wrong:** a regression of the root allocator passes CI.

</details>
