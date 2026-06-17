# Review: PR #4885
Event: COMMENT

## Body
Design is sound and both review threads are genuinely closed. Verified on e9199dc9e: with the source string out of GC roots, visiting only the slice still charges the full source backing once, and reverting the range lookup to the old pointer-equality keying drops those bytes; a forked allocator's end-of-cycle cleanup no longer prunes the parent's ranges; and each store-loaded string is charged once (`allocString + len`), since a container's shallow size is independent of its string fields' length, so the load-path `Allocate` adds nothing and `fillTypesOfValue`'s `NewString` is the only charge. That matches the +32 / +485 gas deltas.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4885-correctly-reuse-count-string/1-e9199dc9e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/uverse.go:171 [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/uverse.go#L171)
After this PR a `StringValue` built directly, without `NewString`/`TrackString`, has its backing counted as zero at GC; this site, `GetSlice` ([values.go:2194](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/values.go#L2194)), and `typedString` ([values.go:2720](https://github.com/gnolang/gno/blob/e9199dc9e/gnovm/pkg/gnolang/values.go#L2720)) all do that. It's bounded today since these carry only a realm path and fixed panic messages and every user-controllable string routes through `NewString`, but nothing enforces that invariant and no test fails if a future direct `StringValue(x)` site is added. Route the three through a `safeStringValue(alloc, s)` helper that tracks first, and add a regression test that GC-counts a directly-built `StringValue`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4885 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_untracked_test.go <<'EOF'
package gnolang

import "testing"

func TestUntrackedStringUndercount(t *testing.T) {
	const s = "this-string-is-fifty-something-bytes-long-padded-xx" // 51 bytes
	var vc int64

	a1 := NewAllocator(1_000_000) // direct construction, as in NewConcreteRealm/typedString/GetSlice
	v1 := GCVisitorFn(1, a1, &vc)
	a1.Reset()
	v1(StringValue(s))
	t.Logf("untracked StringValue(s): %d bytes (header only = %d)", a1.bytes, int64(allocString))

	a2 := NewAllocator(1_000_000) // same content via NewString (tracked)
	sv := a2.NewString(s)
	v2 := GCVisitorFn(1, a2, &vc)
	a2.Reset()
	v2(sv)
	t.Logf("tracked   NewString(s):   %d bytes (header + %d backing)", a2.bytes, int64(len(s)))
}
EOF
go test ./gnovm/pkg/gnolang/ -run TestUntrackedStringUndercount -v 2>&1 | grep -E "untracked|tracked"
rm gnovm/pkg/gnolang/zz_untracked_test.go
```

```
untracked StringValue(s): 48 bytes (header only = 48)
tracked   NewString(s):   99 bytes (header + 51 backing)
```
</details>
