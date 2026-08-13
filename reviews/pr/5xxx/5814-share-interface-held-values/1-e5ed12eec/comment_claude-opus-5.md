# Review: [#5814](https://github.com/gnolang/gno/pull/5814)
Event: COMMENT

## Body
The merge-base VM fails 4 of 2454 filetests against this branch's goldens, three of them the ones the branch adds.

- The description misses a second behavior change: the merge base persisted one escaped object or two owned ones for the same realm code, depending on whether a read had resolved the element from a `RefValue` before the copy, and this branch persists the same graph either way.
- `copy` and `append` over `[]any` still deep-copy every element, so `dst = src` shares where `copy(dst, src)` does not, while Go shares in both, and the slice is the shape realm code has: 322 lines under `examples/gno.land` and `gnovm/stdlibs` mention `[]any` or `[]error` and none declares a fixed-size interface array.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5814-share-interface-held-values/1-e5ed12eec/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

Repros run at e5ed12eec.

## gnovm/pkg/gnolang/values.go:466 [gh](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L466) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L466)
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

## gnovm/tests/files/zrealm_iface_array_share.gno:19-20 [gh](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/zrealm_iface_array_share.gno#L19-L20) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/zrealm_iface_array_share.gno#L19-L20)
Missing test: all three added tests create the element in the transaction that copies it, so nothing covers an array already in the store and read before the copy, or one handed to another realm by value, where the merge base and this branch persist different graphs.

<details><summary>test cases</summary>

Both need their `Realm:` block filled by `go test -run 'TestFiles/<name>.gno$' -update-golden-tests .` from `gnovm/pkg/gnolang/`, after which they pass at e5ed12eec and fail at 754780601.

For the stored array, the merge base writes a second 216-byte object where this branch escapes the first. Dropping the `pre:` line is the other half of the pair: at 754780601 the two files persist different graphs, at e5ed12eec the same one.

`gnovm/tests/files/zrealm_iface_array_share_stored.gno`:

```go
// PKGPATH: gno.land/r/test
package test

type S struct{ F int }

var (
	arr  [1]any
	arr2 [1]any
)

func init() {
	arr[0] = S{1}
}

func main(cur realm) {
	println("pre:", arr[0].(S).F) // resolves the RefValue before the copy
	arr2 = arr
	println("vals:", arr[0].(S).F, arr2[0].(S).F)
}

// Output:
// pre: 1
// vals: 1 1

// Realm:
// PLACEHOLDER
```

Across the realm boundary the callee's stored array ends up holding a `RefValue` into the caller's object, which loses its `OwnerID` and goes to `RefCount 2` for good; the merge base gives the callee its own 215-byte object instead.

`gnovm/tests/files/zrealm_iface_array_share_crossrealm.gno`:

```go
// PKGPATH: gno.land/r/crossrealm
package crossrealm

import (
	"gno.land/r/tests/vm/crossrealm_b"
)

type S struct{ F int }

var arr [1]any

func init() {
	arr[0] = S{1}
}

func main(cur realm) {
	println("pre:", arr[0].(S).F)
	crossrealm_b.SetObject(cross(cur), arr)
	got := crossrealm_b.GetObject().([1]any)
	println("here:", arr[0].(S).F, "there:", got[0].(S).F)
}

// Output:
// pre: 1
// here: 1 there: 1

// Realm:
// PLACEHOLDER
```
</details>

## gnovm/tests/files/gas/nested_alloc.gno:12 [gh](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/gas/nested_alloc.gno#L12) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/gas/nested_alloc.gno#L12)
Nit: the description's callout quotes 8,559,690,088 dropping to 17,013,825; the golden and the merge base give 8,559,690,224 and 17,013,961 instead.
