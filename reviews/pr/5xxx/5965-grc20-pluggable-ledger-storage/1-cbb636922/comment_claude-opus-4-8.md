# Review: PR [#5965](https://github.com/gnolang/gno/pull/5965)
Event: APPROVE

## Body
The storage seam is opt-in: the default avl path and all existing `NewToken` callers are unchanged. Verified on cbb636922: the package's FNV-1a matches Go's hash/fnv for the same keys, so bucket placement is the standard hash and its grindability is exactly what the doc describes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5965-grc20-pluggable-ledger-storage/1-cbb636922/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/p/nt/hashmap/v0/hashmap.gno:14-15 [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L14)
The doc says per-operation cost is flat in N, but only the object-load count is constant. The touched bucket is a native map of size O(N/buckets), and [store reads and writes both charge per serialized byte](https://github.com/gnolang/gno/blob/cbb636922/tm2/pkg/store/types/gas.go#L404-L407), so per-op gas rises with entries-per-bucket, which is the 4.3M to 5.7M slope the ADR measures from 20k to 1M.

## examples/gno.land/p/nt/hashmap/v0/hashmap.gno:47-48 [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L47)
Nit: this says 1024 buckets is good up to roughly 100k entries, but the [sizing table below](https://github.com/gnolang/gno/blob/cbb636922/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L29-L32) and the ADR both put 1024 flat to ~1,000,000.

## examples/gno.land/p/nt/hashmap/v0/hashmap.gno:11-13 [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L11)
Suggestion: the single-object claim holds only when values serialize inline. This `any`-valued map would spawn a persisted object per struct or pointer value and reintroduce O(entries) loads; grc20's int64 keeps the property.

## examples/gno.land/p/nt/hashmap/v0/hashmap.gno:153-162 [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/nt/hashmap/v0/hashmap.gno#L153)
Suggestion: inserting into the map inside the `Iterate` callback visits the newly added entries and can run well past the original count. Order stays deterministic so there is no consensus risk, but it is a gas and termination footgun. Document that the map must not be mutated during iteration.

## examples/gno.land/p/demo/tokens/grc20/storage_option_test.gno:15 [↗](../../../../../.worktrees/gno-review-5965/examples/gno.land/p/demo/tokens/grc20/storage_option_test.gno#L15)
Missing test: no committed test guards the single-object, O(1)-object-load property the package exists to provide; `hashmap_test.gno` is in-memory and this flow runs in one machine without crossing a store boundary. A refactor of the bucket layout could regress the gas property with every test still green. Add a persistence filetest to the hashmap package and an object-count harness.

<details><summary>test cases</summary>

Persistence filetest, passing at cbb636922 with a `storage: ...:-80b` line that confirms the realm was persisted. Place it under `examples/gno.land/p/nt/hashmap/v0/filetests/`:

```gno
// PKGPATH: gno.land/r/demo/hmtest
package hmtest

import "gno.land/p/nt/hashmap/v0"

var m *hashmap.Map

func init() {
	m = hashmap.New()
	m.Set("alice", int64(1000))
	m.Set("bob", int64(500))
}

func main(cur realm) {
	println("alice:", m.Get("alice").(int64))
	println("bob:", m.Get("bob").(int64))
	println("size:", m.Size())
	m.Set("alice", int64(1234))
	m.Remove("bob")
	println("alice-after:", m.Get("alice").(int64))
	println("has-bob-after:", m.Has("bob"))
	println("size-after:", m.Size())
}

// Output:
// alice: 1000
// bob: 500
// size: 2
// alice-after: 1234
// has-bob-after: false
// size-after: 1
```

The O(1)-object claim itself is not observable from `.gno`; it needs a Go store-object-count `_test.go` harness, the one the ADR used, so "77 objects at every ledger size" is guarded in-repo.
</details>
