# Review: PR [#5901](https://github.com/gnolang/gno/pull/5901)
Posted: https://github.com/gnolang/gno/pull/5901#pullrequestreview-5042805140
Event: COMMENT
Status: posting as an AI. Original verdict APPROVE, forced to COMMENT.

## Body
[AI review, opus 4.8]
Status: APPROVE

Reverting the holder to an eager `maps.Clone` at the call site and counting clones puts non-type-checking transactions at zero, [`AddPackage`](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L664) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L664) and [`Run`](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L1057) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L1057) at exactly one per transaction. A write into one holder's clone reaches neither the shared base nor a sibling transaction's clone. The clone stays off the metered path: `TypeCheckMemPackage` runs before `SetPreprocessAllocator` and `SetGasMeter` wire the gas meter to the store.

## gno.land/pkg/sdk/vm/keeper.go:417 [gh](https://github.com/gnolang/gno/blob/cf9cdf9f5/gno.land/pkg/sdk/vm/keeper.go#L417) · [↗](../../../../../.worktrees/gno-review-5901/gno.land/pkg/sdk/vm/keeper.go#L417) [posted](https://github.com/gnolang/gno/pull/5901#discussion_r3873395246)
Missing test: nothing asserts a transaction's clone write stays out of the shared base and out of a sibling transaction's clone. The consumer writes newly type-checked packages back into the passed cache at [`gotypecheck.go:343`](https://github.com/gnolang/gno/blob/cf9cdf9f5/gnovm/pkg/gnolang/gotypecheck.go#L343), so isolation is the property the lazy holder rests on. A `get()` change that shared state across transactions would still pass the whole suite.

<details><summary>test cases</summary>

```go
// in package vm (internal test); exercises typeCheckCacheHolder.get() directly.
func TestHolderLazyAndIsolated(t *testing.T) {
	pkgA := types.NewPackage("a", "a")
	base := gno.TypeCheckCache{"a": pkgA}

	h1 := &typeCheckCacheHolder{base: base}
	if h1.cloned != nil {
		t.Fatal("holder cloned before first get()")
	}
	c1 := h1.get()
	if c1["a"] != pkgA {
		t.Fatal("clone dropped a base entry")
	}

	// single clone per holder: second get() reuses the working copy.
	pkgB := types.NewPackage("b", "b")
	c1["b"] = pkgB
	if h1.get()["b"] != pkgB {
		t.Fatal("second get() re-cloned instead of reusing")
	}

	// isolation from base: the consumer's write must not leak into base.
	if _, ok := base["b"]; ok {
		t.Fatal("clone write leaked into shared base")
	}

	// isolation between transactions: a sibling holder gets its own clone.
	c2 := (&typeCheckCacheHolder{base: base}).get()
	if _, ok := c2["b"]; ok {
		t.Fatal("one tx's clone write leaked into another tx's clone")
	}
}
```

```
# from a local clone of gnolang/gno:
gh pr checkout 5901 -R gnolang/gno
curl -fsSL -o gno.land/pkg/sdk/vm/holder_isolation_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5901-lazy-typecheck-cache-clone/1-cf9cdf9f5/tests/holder_isolation_test.go
go test -run TestHolder ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/holder_isolation_test.go
```

```
ok  	github.com/gnolang/gno/gno.land/pkg/sdk/vm	0.019s
```
</details>
