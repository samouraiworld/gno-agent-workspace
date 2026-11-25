# Review: [#6515](https://github.com/gnolang/gno/pull/6515)
Posted: https://github.com/gnolang/gno/pull/6515#pullrequestreview-6055499648
Event: COMMENT
Verdict: NEEDS DISCUSSION, `sliceSwap` moves stored elements where Go swaps their contents, so a pointer into an element no longer behaves as it does in Go.
Model: claude-opus-5-5, high effort, targeted check of the Go divergence, no review round
Commit: 5870ba35e (latest)
Overview: none, targeted check
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6515 5870ba35e`

## Body

## gnovm/stdlibs/sort/slice.go:33 [gh](https://github.com/gnolang/gno/blob/5870ba35e/gnovm/stdlibs/sort/slice.go#L33) · [↗](../../../../../.worktrees/gno-review-6515/gnovm/stdlibs/sort/slice.go#L33) · Warning [posted](https://github.com/gnolang/gno/pull/6515#discussion_r4693271188)
This line moves the stored elements, so a pointer to a field of `data[0]` follows its element to another index after sorting, where Go's [`Swapper`](https://github.com/golang/go/blob/go1.25.9/src/internal/reflectlite/swapper.go#L74-L76) swaps the contents and the pointer stays at index 0. Could this swap the contents the way Go does, so code ported from Go keeps the pointers it took?

<details><summary>repro</summary>

The same program in Go and in Gno at this head:

```go
package main

import "sort"

type pair struct{ key, val int }

func main() {
	data := []pair{{3, 0}, {1, 1}}
	field := &data[0].key
	elem := &data[0]
	sort.Slice(data, func(i, j int) bool { return data[i].key < data[j].key })
	println("*field", *field, "elem.key", elem.key)
	*field = 9
	println(data[0].key, data[0].val, data[1].key, data[1].val)
}
```

Go writes through `field` into `data[0]`; Gno writes into `data[1]`:

```
# go run main.go (go1.25.9)
*field 1 elem.key 1
9 1 3 0

# gno run main.gno
*field 3 elem.key 1
1 1 9 0
```
</details>
