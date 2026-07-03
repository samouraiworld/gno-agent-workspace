# Review: PR [#5882](https://github.com/gnolang/gno/pull/5882)
Event: APPROVE

## Body
Looks good. Verified on e98021315: pointing the delete builtin's `DidUpdate` at the argument key's object instead of the stored key's removes the `d[...:8](-213)` reclaim from the zrealm_map5 finalize golden, so this change is what deletes the stored key. Delete reclaims only the map's own copy of an array or struct key: an escaped copy of the same key value stays readable after the delete, and a pointer key's shared pointee is updated rather than deleted.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5882-reclaim-stored-map-key/1-e98021315/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:828-840 [↗](../../../../../.worktrees/gno-review-5882/gnovm/pkg/gnolang/values.go#L828)
DeleteForKey already takes the machine, so it could mark the removed key deleted itself instead of returning it for the builtin to mark. Was keeping it a pure container op and consolidating the realm bookkeeping in the builtin a deliberate split?

## gnovm/tests/files/zrealm_map5.gno:1 [↗](../../../../../.worktrees/gno-review-5882/gnovm/tests/files/zrealm_map5.gno#L1)
The regression golden covers only an array key. A struct key runs the same value-composite reclaim path, so a companion golden locks the general case.

<details><summary>test</summary>

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
