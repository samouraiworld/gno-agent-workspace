# Gno glossary

Shared vocabulary for gnolang/gno review and reporting, loaded by `skills/review.md`. A living doc: when a review needs a project-internal term, add a one-line entry here so the next review names it instead of re-explaining it. One terse line per term, alphabetical. Each term names a reusable concept the next review will meet again — a component, mechanism, or recurring pattern — not a function or symbol scoped to one PR; cite a specific symbol inside the definition when it pins the concept, never as the term itself. Prefer a plain, readable term over an opaque abbreviation (write "copy-on-write", not "COW"); keep an abbreviation as the term only when it is the codebase's own name (IAVL, ICS23, ABCI).

- addpkg: the transaction (`maketx addpkg`) that uploads a package or realm to the chain.
- Allocator: VM component tracking memory allocation and charging allocation gas; `fallbackAllocator` is the global, effectively-unbounded (`MaxInt64` budget) instance for pure-function or no-Machine paths.
- app hash: the per-block commitment to application state (the multistore's Merkle root), agreed in consensus; two honest nodes computing different app hashes for the same block halts the chain.
- banker: stdlib API (package `chain/banker`, interface `banker.Banker`, constructed via `banker.NewBanker`) for issuing, sending, and burning coins from a realm.
- chain: the on-chain stdlib root package (`gnovm/stdlibs/chain`); holds `Emit`, `Coins`, `PackageAddress`, with realm context in `chain/runtime` and the spoofable stack-walkers quarantined in `chain/runtime/unsafe`. Replaces the former `std` gno package.
- copy-on-write (COW): a persistent-tree update that, instead of changing a node in place, writes a new node and rewrites the path up to the root, leaving older versions untouched; how IAVL and the bptree share unchanged subtrees across versions.
- crossing / `cross`: a call into a crossing function (`func F(cur realm, ...)`), invoked as `cross(cur)`; the callee identifies its caller through `cur.Previous()` (see realm), not the stack-walking `unsafe.PreviousRealm()`.
- Exception: the GnoVM's Go-level panic value (`gnovm/pkg/gnolang`), wrapping the gno panic value in `Exception.Value`; `runOnce` (`machine.go`) catches `*Exception` via Go `recover()` and re-raises anything else, so a bare Go panic escapes the VM and gno `recover()` (which returns `Exception.Value`) can't catch it.
- filetest: a `*_filetest.gno` file executed by the VM and asserted against golden directives (`// Output:`, `// Error:`, `// Realm:`, `// Events:`, ...).
- gas: metered cost of CPU and memory during execution; consensus-relevant, so any change to it is a behavior change.
- GnoVM: the Gno virtual machine (`gnovm/pkg/gnolang`) that preprocesses and interprets gno code.
- gnobuiltins: synthetic packages (e.g. `gnobuiltins/gno0p9`) injected only for type-checking, never run on chain.
- gnomod.toml: per-package manifest declaring the module path (pkgpath) and gno language version.
- `gno test -p`: parallel mode of `gno test` (`gnovm/cmd/gno/test.go`); `-p N` tests N packages concurrently (`N <= 0` means GOMAXPROCS), each worker with its own Store, output buffered and printed in completion order. Surfaces global-state races that sequential `-p 1` hides.
- IAVL: the incumbent versioned, self-balancing Merkle binary tree backing tm2 state storage (`tm2/pkg/iavl`, store wrapper `tm2/pkg/store/iavl`); the bptree is a drop-in alternative.
- ICS23: the IBC vector-commitment proof standard (`cosmos/ics23/go`); a `ProofSpec` plus existence/non-existence `CommitmentProof`s let a light client verify membership/absence against a root. The bptree maps each node's mini-merkle to a uniform chain of binary `InnerOp`s.
- Machine: a GnoVM execution instance (`gno.Machine`) bound to a Store, Allocator, and Context.
- MemPackage: in-memory set of a package's source files (`std.MemPackage`, Go package `tm2/pkg/std`), the unit loaded, type-checked, and run.
- preprocess: the static pass (`PredefineFileSet`/`initStaticBlocks`) that resolves names, types, and blocks before execution.
- pure package: an importable, stateless package under `p/`; contrast realm.
- realm: a stateful on-chain package under `r/` whose objects persist across transactions; also the VM builtin type threaded as a `cur realm` parameter, where `cur.Previous()` returns the caller (unforgeable caller identity is `cur.Previous().Address()`, `cur.Previous().PkgPath()`), while bare `cur.Address()` / `cur.PkgPath()` are the current realm's own.
- state sync: bootstrapping a node from a snapshot of committed state at a height (rather than replaying all blocks); the snapshot is streamed and rebuilt, then checked against the trusted app hash. The bptree's Exporter/Importer are its (not-yet-wired) state-sync surface.
- stdlib: gno standard libraries: prod under `gnovm/stdlibs/`, test-only overrides under `gnovm/tests/stdlibs/`.
- storage deposit: per-realm refundable charge for on-chain storage, locked on positive byte delta and refunded in proportion to the realm's original deposit ratio; `processStorageDeposit` (`gno.land/pkg/sdk/vm`), tracked as `rlm.Deposit`/`rlm.Storage`, governed by the `storage_price` and `default_deposit` VM params.
- Store: the backing store for packages and objects (`gno.Store`/`defaultStore`), layered over a tm2 CommitStore/IAVL.
- transactionStore: the per-transaction Store wrapper (struct `{*defaultStore}`) returned by `BeginTransaction`, carrying per-tx caches and the tx-scoped gas meter; its methods promote to the embedded `*defaultStore`, so it is NOT matched by a `store.(*defaultStore)` type assertion.
- txtar: testscript-based integration tests under `gno.land/pkg/integration/testdata/`.
- type-check: go/types-based validation of gno source (`TypeCheckMemPackage`), distinct from preprocessing.
- TypeCheckCache: per-run map of already-type-checked imported packages (`gno.TypeCheckCache`, `gotypecheck.go`), passed via `TypeCheckOptions.Cache` to skip re-checking; an unlocked map, so parallel `gno test` workers each hold their own.
- unsafe: package `chain/runtime/unsafe` holding the stack-walking tx-origin primitives (`PreviousRealm`, `CurrentRealm`, `OriginCaller`, `OriginSend`); footgun-prone for auth (prefer `cur.Previous()`; see the catalog's Caller & access control).
