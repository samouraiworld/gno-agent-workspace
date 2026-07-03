# Review: PR [#5882](https://github.com/gnolang/gno/pull/5882)
Event: APPROVE

## Body
Looks good. Verified on e98021315: reverting to mark the argument key instead of the stored key drops the `d[...:8](-213)` reclaim from the finalize golden, so this frees the leaked key. Escaped key copies and shared pointees survive the delete, so only the map's own key object is freed.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:828-840 [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/values.go#L828)
DeleteForKey has the machine and could mark the removed key itself instead of returning it for the builtin. Deliberate split to keep it a pure container op?

## gnovm/tests/files/zrealm_map5.gno:1 [↗](../../../../../.worktrees/gno-review-5882/gnovm/tests/files/zrealm_map5.gno#L1)
Missing test: a struct key, which runs the same value-composite reclaim path the array-key golden covers.

<details><summary>test cases</summary>

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

Its `// Realm:` golden ends with `d[...:8](-252)`, the reclaimed struct key. The full file with golden, ready to drop in as `gnovm/tests/files/zrealm_map6.gno`: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/tests/zrealm_map6.gno
</details>
