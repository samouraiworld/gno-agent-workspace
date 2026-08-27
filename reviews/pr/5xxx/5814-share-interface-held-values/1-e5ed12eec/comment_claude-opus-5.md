# Review: [#5814](https://github.com/gnolang/gno/pull/5814)
Posted: https://github.com/gnolang/gno/pull/5814#pullrequestreview-5042804178
Event: COMMENT
Status: posting as an AI. Verdict COMMENT.

## Body
[AI review, opus 5]
Status: COMMENT

The merge-base VM fails 4 of 2454 filetests against this branch's goldens, three of them the ones the branch adds.

- The description misses a second behavior change: the merge base persisted one escaped object or two owned ones for the same realm code, depending on whether a read had resolved the element from a `RefValue` before the copy, and this branch persists the same graph either way.
- `copy` and `append` over `[]any` still deep-copy every element, so `dst = src` shares where `copy(dst, src)` does not, while Go shares in both, and the slice is the shape realm code has: 322 lines under `examples/gno.land` and `gnovm/stdlibs` mention `[]any` or `[]error` and none declares a fixed-size interface array.

Repros run at e5ed12eec.

## gnovm/pkg/gnolang/values.go:472 [gh](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L472) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L472) [posted](https://github.com/gnolang/gno/pull/5814#discussion_r3873394362)
`go/types` enforces the addressability premise and the VM does not, so `arr2[0].(S).F = 9` now writes into the source array too. `panicIllegalPointerLHS` already refuses a pointer-receiver call on an interface-held value, so rejecting the assignment there would put the guarantee in the VM.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5814 -R gnolang/gno
cat > zz_ifshare.gno <<'EOF'
package main

type S struct{ F int }

func main() {
	var arr [1]any
	arr[0] = S{1}
	arr2 := arr
	arr2[0].(S).F = 9
	println("orig:", arr[0].(S).F, "copy:", arr2[0].(S).F)
}
EOF
go run ./gnovm/cmd/gno run zz_ifshare.gno
rm zz_ifshare.gno
```

`gno run` prints no diagnostic and the write lands on both arrays; the same command on the merge base prints `orig: 1 copy: 9`.

```
orig: 9 copy: 9
```

`gno lint` reports `cannot assign to arr2[0].(S).F (neither addressable nor a map index expression)` on the same source, and `TypeCheckMemPackage` runs in both `AddPackage` and `Run`, so nothing here reaches a transaction.
</details>

## gnovm/tests/files/zrealm_iface_array_share.gno:19-20 [gh](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/zrealm_iface_array_share.gno#L19-L20) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/zrealm_iface_array_share.gno#L19-L20) [posted](https://github.com/gnolang/gno/pull/5814#discussion_r3873394373)
Missing test: all three added tests create the element in the transaction that copies it, so nothing covers an array already in the store and read before the copy, or one handed to another realm by value, where the merge base and this branch persist different graphs.

<details><summary>test cases</summary>

Two filetests, goldens filled and both run at e5ed12eec and at the merge base 754780601. Each carries a `Run:` header that fetches it into a gno checkout.

- [`zrealm_iface_array_share_stored.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5814-share-interface-held-values/1-e5ed12eec/tests/zrealm_iface_array_share_stored.gno) covers an array already in the store, read before the copy. It passes at e5ed12eec; at 754780601 the golden differs, where the merge base writes a second object rather than escaping the first.
- [`zrealm_iface_array_share_crossrealm.gno`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5814-share-interface-held-values/1-e5ed12eec/tests/zrealm_iface_array_share_crossrealm.gno) covers an array handed to another realm by value through `crossrealm_b.SetObject`. Same result: passes at e5ed12eec, golden differs at 754780601.

</details>

## gnovm/tests/files/gas/nested_alloc.gno:12 [gh](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/gas/nested_alloc.gno#L12) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/gas/nested_alloc.gno#L12) [posted](https://github.com/gnolang/gno/pull/5814#discussion_r3873394383)
Nit: the description's callout quotes 8,559,690,088 dropping to 17,013,825; the golden and the merge base give 8,559,690,224 and 17,013,961 instead.
