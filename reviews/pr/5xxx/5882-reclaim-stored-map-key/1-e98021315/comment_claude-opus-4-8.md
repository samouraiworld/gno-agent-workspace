# Review: PR [#5882](https://github.com/gnolang/gno/pull/5882)
Event: APPROVE

## Body
Looks good. Verified on e98021315: reverting to mark the argument key instead of the stored key drops the `d[...:8](-213)` reclaim from the finalize golden, so this frees the leaked key.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:828-840 [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/values.go#L828)
DeleteForKey has the machine and could mark the removed key itself instead of returning it for the builtin. Deliberate split to keep it a pure container op?

## gnovm/tests/files/zrealm_map5.gno:1 [↗](../../../../../.worktrees/gno-review-5882/gnovm/tests/files/zrealm_map5.gno#L1)
Missing test: the struct-key path, and the over-reclaim guard where the key value is also held elsewhere.

<details><summary>test cases</summary>

Struct key, same value-composite reclaim path (golden ends `d[...:8](-252)`), ready to drop in as `gnovm/tests/files/zrealm_map6.gno`: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/tests/zrealm_map6.gno

```go
// PKGPATH: gno.land/r/test
package test

type Key struct{ A, B int }

var m map[Key]int

func init() {
	m = map[Key]int{Key{1, 2}: 10, Key{3, 4}: 20}
}

func main(cur realm) {
	delete(m, Key{1, 2})
	println("len:", len(m))
}

// Output:
// len: 1
```

Over-reclaim guard: the array key value is also held through `keep`, so delete reclaims only the map's copy (golden ends `d[...:9](-213)`) and `keep` stays readable. Reverting the fix drops that reclaim. Ready to drop in as `gnovm/tests/files/zrealm_map7.gno`: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/tests/zrealm_map7.gno

```go
// PKGPATH: gno.land/r/test
package test

var m map[[1]int]int
var keep *[1]int

func init() {
	k := [1]int{1}
	m = map[[1]int]int{k: 10, [1]int{2}: 20}
	keep = &k
}

func main(cur realm) {
	delete(m, [1]int{1})
	println("len:", len(m), "keep:", keep[0])
}

// Output:
// len: 1 keep: 1
```
</details>
