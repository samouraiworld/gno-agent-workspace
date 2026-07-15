# PR [#5965](https://github.com/gnolang/gno/pull/5965): feat(examples): pluggable grc20 ledger storage + p/nt/hashmap (flat gas for large ledgers)

URL: https://github.com/gnolang/gno/pull/5965
Author: omarsy | Base: master | Files: 7 | +615 -6
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep) | Commit: cbb636922 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5965 cbb636922`

**TL;DR:** A cold GRC20 transfer gets more expensive as the token gains holders, because the avl tree behind the ledger loads more persisted objects the deeper it grows. This PR lets a token pick a different storage backend, and adds one whose per-operation object-load count does not grow with holder count. The default stays avl, so nothing changes unless a token opts in.

**Verdict: APPROVE** — the seam is backwards-compatible and the O(1)-object claim is real; open items are a doc claim that overstates "flat gas", a couple of sizing-doc inconsistencies, and a missing persistence guard test. No correctness, determinism, or security defect found.

## Summary

grc20's `PrivateLedger` kept balances and allowances in `avl.Tree`. On an N-holder ledger a `Transfer` walks `O(log N)` tree nodes, each a separate persisted object, and a cold object read costs a flat 59k gas, so transfer gas climbs with holder count (the state-scaling half of [#5906](https://github.com/gnolang/gno/issues/5906): avl at ~1M holders reproduces the observed on-chain 18.3M-gas transfer). The PR introduces a `KV` interface satisfied by `*avl.Tree`, `*hashmap.Map`, and `*bptree.BPTree`, plus a `WithStorage` option on `NewToken`; the fields become `KV` and default to avl. The new `p/nt/hashmap/v0` stores entries in a fixed power-of-two array of native Gno maps: a native map persists as one object regardless of entry count, so any op loads a constant number of objects (Map struct, bucket array, one bucket) rather than `O(log N)`.

## Examples

| call | backend | ordered iteration | per-op object loads |
|---|---|---|---|
| `NewToken(0, cur, "Foo", "FOO", 6)` | avl (default) | yes | O(log N) |
| `NewToken(..., WithStorage(func() KV { return hashmap.New() }))` | hashmap | no | O(1) |
| `NewToken(..., WithStorage(func() KV { return bptree.NewBPTree32() }))` | bptree | yes | O(log N), higher fanout |

## Glossary

- cold object read: a load of a persisted realm object not in the VM object cache, charged the flat store-read cost (59k gas + 17/byte); the count of these per op is what a storage structure's cold gas scales with.

## Critical (must fix)
None.

## Warnings (should fix)

- **[headline gas claim overstates what's guaranteed]** `hashmap.gno:14-15` — the package doc says per-op cost is "flat in N"; only the object-load count is flat, per-op gas still grows with the ledger.
  <details><summary>details</summary>

  The doc reasons: a native map is one object, an op loads a constant number of objects, "Cost per operation is therefore flat in N." The object-load count is constant, but gas is not. A store read is priced `ReadCostFlat 59_000 + ReadCostPerByte 17 × len(bytes)` and a write `WriteCostFlat 24_000 + WriteCostPerByte 14 × len(bytes)` ([`gas.go:404-407`](https://github.com/gnolang/gno/blob/cbb636922/tm2/pkg/store/types/gas.go#L404-L407) · [↗](../../../../../.worktrees/gno-review-5965/tm2/pkg/store/types/gas.go#L404-L407)). The one touched bucket is a native map whose serialized size is `O(N/buckets)`, so its per-byte read and its full re-serialization on every Set/Remove grow with entries-per-bucket. This is why the ADR's own numbers climb 4.3M → 5.7M from 20k to 1M entries. Fix: state that the object-load count is constant while per-op gas still rises slowly with entries-per-bucket via the per-byte read/write cost. Verified behaviorally: read gas is flat-plus-per-byte per the cited config; the ADR table shows the resulting slope.
  </details>

## Nits

- **[doc contradicts the same file]** `hashmap.gno:47-48` — `DefaultBuckets` says 1024 is "Good for maps up to roughly 100k entries", but the sizing table at [`hashmap.gno:29-32`](https://github.com/gnolang/gno/blob/cbb636922/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L29-L32) · [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L29-L32) and the ADR both say 1024 stays flat to ~1,000,000. Align the 100k figure to 1M.
- **[documented footgun, no code change needed]** `token.gno:113` — a `WithStorage` closure that returns the same map for both maps (violating the documented "fresh, empty store" contract) makes `balances` and `allowances` share one store; lookups stay correct because address keys and `owner:spender` keys never collide, but `KnownAccounts()` counts allowance entries too. Reachable only by ignoring the doc; the default path is unaffected. Flagging for the record, not posting.
- **[documented footgun, no code change needed]** `hashmap.gno:84-86` — a zero-value `Map` (`var m Map`) panics with a bare index-out-of-range on first `Set` (`len(buckets)-1` = -1 masks to all-ones, indexes an empty slice). Documented ("zero value is not usable") and it surfaces as a normal gno-recoverable VM Exception, so it is a doc-matching footgun, not a fault escape. Not posting.

## Missing Tests

- **[core property has no committed guard]** `examples/gno.land/p/nt/hashmap/v0/` — the package's whole justification (a native map is one object; O(1) object loads; correct across a cold store reload) has no committed test. `hashmap_test.gno` is entirely in-memory, and [`storage_option_test.gno:15`](https://github.com/gnolang/gno/blob/cbb636922/examples/gno.land/p/demo/tokens/grc20/storage_option_test.gno#L15) · [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/demo/tokens/grc20/storage_option_test.gno#L15) runs the token flow inside one machine, never crossing a store boundary.
  <details><summary>details</summary>

  A refactor of `buckets []map[string]any` that accidentally split buckets into per-entry objects would regress the gas property with every unit test still green. Two guards are worth adding: a persistence filetest that finalizes a realm holding a `Map` and reads it back after the init→main boundary (ready and passing, [hashmap_persist_filetest.gno](tests/hashmap_persist_filetest.gno)), and an object-count assertion for the O(1) claim itself, which is not observable from `.gno` and needs a Go-level store-object-count `_test.go` harness (the same harness the ADR used). Fix: add the persistence filetest now; add the object-count harness so the ADR's "77 objects at every ledger size" is guarded in-repo.
  </details>

## Suggestions

- **[doc precision on value types]** `hashmap.gno:11-13` — "A native map persists as a single object regardless of its entry count" holds for the map structure and for inline/primitive values (grc20 stores `int64`, so O(1)-objects holds there), but hashmap is generic `any`-valued: storing pointer or struct values makes each value its own persisted object, reintroducing O(entries) object loads. One sentence noting the property assumes inline-serializing value types would prevent misuse.
- **[missing API caveat]** `hashmap.gno:153-162` — mutating the map inside the `Iterate` callback visits newly inserted entries and can loop far past the original entry count (deterministic, since gno maps iterate in insertion order, so no consensus split, but a gas/termination footgun). avl documents "do not mutate during iteration"; hashmap does not. Add that sentence.
- **[disclosed default-path cost]** `types.gno:141-143`, `token.gno:45` — balances/allowances change from an inline `avl.Tree` value to a `KV` interface holding a `*avl.Tree`, so the default path loads the tree as a separate persisted object behind the interface, adding cold object loads (+1.7% at 20k holders per the ADR) to every token that never opts in. Disclosed and owned in the ADR; only newly deployed tokens are affected since an already-deployed package is immutable. Noted, not posting.
- **[defensive, caller-supplied factory]** `token.gno:45-52` — `NewToken` calls `cfg.newKV()` without checking the result; a closure returning nil (or a typed-nil `*avl.Tree`) yields a ledger that panics on first access. Optional guard; inherent to a caller-supplied factory. Not posting.

## Verified

- **Iteration determinism (the load-bearing consensus property):** map `range` in the GnoVM walks `mv.List.Head`, a doubly-linked insertion-ordered list ([`op_exec.go:391`](https://github.com/gnolang/gno/blob/cbb636922/gnovm/pkg/gnolang/op_exec.go#L391) · [↗](../../../../../.worktrees/gno-review-5965/gnovm/pkg/gnolang/op_exec.go#L391)), not the Go map, so `hashmap.Iterate` is deterministic across nodes. `TestIterateDeterministic` (two builds, identical order) passes.
- **Single-object persistence (the O(1)-object claim):** `MapValue` embeds `ObjectInfo` and is the one persisted object; `MapListItem` ([`values.go:763-768`](https://github.com/gnolang/gno/blob/cbb636922/gnovm/pkg/gnolang/values.go#L763-L768) · [↗](../../../../../.worktrees/gno-review-5965/gnovm/pkg/gnolang/values.go#L763-L768)) carries only Prev/Next/Key/Value, and `MapList.MarshalAmino` serializes the whole chain into that object, so entry count adds no objects.
- **KV satisfaction:** assigning `avl.NewTree()`, `bptree.NewBPTree32()`, and `hashmap.New()` each to a `KV` variable and running Set/Get compiles and prints `1 2 3`; signatures match exactly across [avl](https://github.com/gnolang/gno/blob/cbb636922/examples/gno.land/p/nt/avl/v0/tree.gno#L58) · [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/avl/v0/tree.gno#L58), [bptree](https://github.com/gnolang/gno/blob/cbb636922/examples/gno.land/p/nt/bptree/v0/tree.gno#L70) · [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/bptree/v0/tree.gno#L70), and hashmap.
- **Cold store round-trip:** a `Map` written in `init()` reloads correctly in `main()` across the persist boundary ([hashmap_persist_filetest.gno](tests/hashmap_persist_filetest.gno), passes; the `storage: ...:-80b` line confirms the realm was persisted).
- **Backwards-compat:** all 16 existing `NewToken` callers use the fixed 5-arg form, so the variadic add breaks none; the field-type change is invisible (fields unexported, `NewToken` the only construction site). Downstream suites (foo20, wugnot, grc20factory, grc20reg, treasury) pass unchanged.
- **FNV-1a:** the gno hash matches Go `hash/fnv` New64a for `""`, `"a"`, `"hello"`, `"key:val"`, `"\x00"` (offset basis, prime, and xor-then-multiply order all correct).
- Tests green at cbb636922: `hashmap` (5 tests) and `grc20` including `TestWithStorageBackends` over avl/hashmap/bptree; `gno lint` clean for both packages.

## Open questions

- Persisted type-layout of `PrivateLedger` changes (inline `avl.Tree` → `KV` interface), so its TypeID changes. On gno.land a package is immutable by path, so this cannot re-decode existing on-chain state under new code; it only affects fresh deployments, where it is fine. Not posted because it is not actionable and not a break on immutable-package chains; worth an author sanity-check only if a target chain already carries grc20 ledgers predating the change.
- The default-path +1.7% is a real cost imposed on the un-opted-in majority; the ADR judged it acceptable. Left as a disclosed tradeoff, not re-raised.
