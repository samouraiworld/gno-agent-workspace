# Review: PR #5808
Event: REQUEST_CHANGES

> Not posted. A prior posting of this review was removed on 2026-06-11; re-post only on explicit request (`./scripts/post-pr-review.py 5808 <this file>`).

## Body
All five related filetests pass on 17b76f841, but the master merge inside this PR ([bf1467158](https://github.com/gnolang/gno/commit/bf1467158)) staled two of the rationales being pinned (inline comments).

- `cannot delete from readonly tainted map` ([`uverse.go:983-985`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L983-L985) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L983-L985)) is asserted by no test on this branch: since [#5747](https://github.com/gnolang/gno/commit/310dc2a04), [`zrealm_map1.gno`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/zrealm_map1.gno#L32-L33) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_map1.gno#L32-L33) and [`zrealm_map3.gno`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/zrealm_map3.gno#L46-L49) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_map3.gno#L46-L49) pin successful same-realm deletes. Fix: add a test pinning the readonly-delete panic (external-realm map fixture plus a filetest or txtar), or state explicitly that it is intentionally unpinned.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/review_claude-fable-5_davd-gzl.md · [↗](./review_claude-fable-5_davd-gzl.md)

*(AI Agent)*

## gnovm/tests/files/delete1.gno:20-24 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno#L20)
The "unrecoverable VM abort" justification for this pin is stale on this head: since [#5501](https://github.com/gnolang/gno/commit/326832e56) (merged in via [bf1467158](https://github.com/gnolang/gno/commit/bf1467158)) a slice-key delete panics recoverably, so gc parity for the exact pinned case is now implementable. Fix: extend this comment so the no-op reads as a current choice, not a hard constraint — it stays defensible, since func and map keys still abort via the plain-panic default at [`values.go:1683-1686`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/values.go#L1683-L1686) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/values.go#L1683-L1686) and nil-map reads no-op the same way.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
cat > gnovm/tests/files/zz5808_recover.gno <<'EOF'
package main

func main() {
	mi := map[interface{}]int{"a": 1}
	func() {
		defer func() {
			println("recovered:", recover())
		}()
		delete(mi, []int{1})
	}()
	println("after, len:", len(mi))
}

// Output:
// recovered: runtime error: slice type cannot be used as map key
// after, len: 1
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/^zz5808_recover\.gno$' -v
rm gnovm/tests/files/zz5808_recover.gno
```

```
--- PASS: TestFiles/zz5808_recover.gno (0.00s)
```
</details>

*(AI Agent)*

## gnovm/tests/files/delete1.gno:13-19 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno#L13)
Two behaviors this PR's verification relies on are pinned nowhere: the key expression being evaluated exactly once before the nil no-op, and the nil-map read with an unhashable key. Fix: extend this file with both — paste-ready version below keeps every existing case.

<details><summary>extended delete1.gno (passes on 17b76f841)</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
cat > gnovm/tests/files/delete1.gno <<'EOF'
package main

type S struct {
	M map[string]int
}

var pkgM map[string]int

func ret() map[string]int { return nil }

func key() string {
	println("key evaluated")
	return "k"
}

// Per the Go spec, delete on a nil map is a no-op.
func main() {
	var m map[string]int
	delete(m, "k")
	delete(pkgM, "k")
	var s S
	delete(s.M, "k")
	delete(ret(), "k")
	delete(map[string]int(nil), "k")
	// The key expression is still evaluated, exactly once, before the no-op.
	delete(m, key())
	// An unhashable interface key on a nil map also no-ops: the Go spec
	// says delete on a nil map is a no-op (the gc runtime panics here;
	// gno follows the spec text and its own nil-map read behavior).
	var mi map[interface{}]int
	delete(mi, []int{1})
	// The matching nil-map read with an unhashable key no-ops the same way.
	v, ok := mi[[]int{2}]
	println("ok", len(m), len(pkgM), len(s.M), len(mi), v, ok)
}

// Output:
// key evaluated
// ok 0 0 0 0 0 false
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/^delete1\.gno$' -v
git checkout HEAD -- gnovm/tests/files/delete1.gno
```

```
--- PASS: TestFiles/delete1.gno (0.00s)
```
</details>

*(AI Agent)*

## gnovm/tests/files/zrealm_mapnil.gno:9-11 [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_mapnil.gno#L9)
The comment cites `TypedValue.IsReadonly`, removed by [#5747](https://github.com/gnolang/gno/commit/310dc2a04); the claim itself still holds via [`IsReadonlyBy`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/ownership.go#L526-L527) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/ownership.go#L526-L527), which returns false for non-object values. Fix: reword to the current mechanism.

*(AI Agent)*
