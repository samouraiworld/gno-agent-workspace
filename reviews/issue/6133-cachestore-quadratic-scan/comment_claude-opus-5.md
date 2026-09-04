# Comment: [issue #6133](https://github.com/gnolang/gno/issues/6133)
Target: https://github.com/gnolang/gno/issues/6133
Event: COMMENT

Post with `gh issue comment 6133 -R gnolang/gno --body-file <this file>`, body only, waiting on the literal word.

## Body

`tm2/pkg/store/cache` is a fork of cosmos-sdk `store/cachekv` from before the three patches that fixed the same quadratic there, and the container this issue proposes to replace is the one they kept.

<details><summary>what upstream did, and what gno is still missing</summary>

- [#10024](https://github.com/cosmos/cosmos-sdk/pull/10024) sorts the pending keys once and binary-searches the `[start, end)` bounds instead of testing every key with `IsKeyInDomain`.
- [#10026](https://github.com/cosmos/cosmos-sdk/pull/10026), "perf: Make CacheKV store interleaved iterator and insertion not O(n^2)", replaced the sorted cache: "Don't use a doubly linked list as sorted data back-end. This doesn't make sense as a data structure, since each insert will take O(N) time to seek, and thus N random insertions requires time O(n^2) overhead."
- [#12885](https://github.com/cosmos/cosmos-sdk/pull/12885) drains at least `minSortSize` entries per call even when the query range is narrow, so a run of non-matching queries still empties the pending set.

The container went on the sorted side, not the unsorted one. [`store/cachekv/store.go`](https://github.com/cosmos/cosmos-sdk/blob/main/store/cachekv/store.go) on `main` today:

```go
type GStore[V any] struct {
	mtx           sync.Mutex
	cache         map[string]*cValue[V]
	unsortedCache map[string]struct{}
	sortedCache   btree.BTree[V] // always ascending sorted
	parent        types.GKVStore[V]
	...
}
```

with `minSortSize = 1024`.

gno has none of the three, and still carries the `*list.List` that #10026 deleted. That list is a second `O(n)` walk on the same path: `dirtyItems` walks it to find each insertion point, and `newMemIterator` walks it again from the front to reach the domain. So a threshold alone does not finish the job here, which is the part of this issue's "no lazy-sort or threshold scheme escapes the quadratic" that holds and the part that does not.
</details>
