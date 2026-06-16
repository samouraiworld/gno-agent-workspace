# Review: PR #5829
Event: APPROVE

## Body
Looks good. Verified on de74ab7: the compile-time threshold panics at exactly the length where the runtime allocator's `overflow.Addp`/`overflow.Mulp` would overflow `int64`, with the per-element and byte-array paths checked separately. The `Uint8Kind` branch's 1-vs-40 bytes/elem matches `defaultArrayValue`'s routing to `NewDataArray`/`NewListArray`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5829-reject-oversized-array-len/1-de74ab7/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/preprocess.go:2441 [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L2441)
`[...]T{MaxInt64: 1}` isn't rejected here: the check runs after `idx = k + 1`, which overflows to a negative length, so the `length <= 0` guard skips it and the array reaches a runtime `len out of range` panic instead of a compile-time error. Validate the largest index before the `+1` so this form is rejected at preprocess time too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5829 -R gnolang/gno
cat > gnovm/tests/files/zz_ellipsis_maxidx.gno <<'EOF'
package main

func main() {
	x := [...]int{9223372036854775807: 1}
	_ = x
}
EOF
go test -run 'TestFiles/zz_ellipsis_maxidx.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zz_ellipsis_maxidx.gno
```

```
--- FAIL: TestFiles/zz_ellipsis_maxidx.gno (0.00s)
    files_test.go:111: unexpected panic: len out of range
        panic: len out of range
        main/zz_ellipsis_maxidx.gno:4
```

Go on the same source rejects at compile time: `array index 9223372036854775807 out of bounds [0:0]`.
</details>

## gnovm/pkg/gnolang/preprocess.go:4879 [↗](../../../../../.worktrees/gno-review-5829/gnovm/pkg/gnolang/preprocess.go#L4879)
The comment names an `allocMustFit` guard that doesn't exist in the tree (a grep matches only this line); the real guard is the `overflow.Addp`/`overflow.Mulp` calls inside `AllocateListArray`/`AllocateDataArray`. Drop the name or point at those functions.
