# Review: [#6020](https://github.com/gnolang/gno/pull/6020)
Posted: https://github.com/gnolang/gno/pull/6020#pullrequestreview-5002874060
Event: COMMENT

## Body
LGTM, but this collides with [#5710](https://github.com/gnolang/gno/pull/5710), now merged: it meters `ComputeMapKey` on the realm-restore path by keeping the eager `vmap` build.
- The merge-base number for [`compute_map_key_concrete_key.gno`](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/tests/files/gas/compute_map_key_concrete_key.gno#L22) is 135249, not the 125449 the summary table reports, which puts that golden's delta at 7.7% rather than the 0.5% shown.

<details><summary>checks that held</summary>

Deleting the [`mv.ensureVmap` line](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values.go#L1057), the [`fillMapKeyRefs` line before the copy](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values.go#L2461), or [the same line in the map-literal path](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/op_expressions.go#L587) each reddens exactly one of the new realm filetests, a different one each time.

`grep -rn '\.vmap\[' --include='*.go' gnovm/` returns five lines, all inside the one build and the three accessors, so nothing indexes the map outside the paths that pass the flag.
</details>


## gnovm/pkg/gnolang/values.go:1037 [gh](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values.go#L1037) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1037) [posted](https://github.com/gnolang/gno/pull/6020#discussion_r3839120967)
Suggestion: the nil meter leaves this build free, where [#5710](https://github.com/gnolang/gno/pull/5710) charges the eager build it replaces. Keep the build lazy, but pass the gas meter into `ensureVmap`.

## gnovm/pkg/gnolang/values.go:1031 [gh](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values.go#L1031) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1031) [posted](https://github.com/gnolang/gno/pull/6020#discussion_r3839121016)
Missing test: no realm filetest has a map keyed by an interface. Dropping the prefix there would merge `int(1)` and `int64(1)` into one entry with nothing red.

<details><summary>test cases</summary>

```go
// PKGPATH: gno.land/r/map_iface_key
package map_iface_key

// An interface-keyed map read back from the store: the lazy index build and
// every later probe have to agree on keeping the TypeID prefix, or int(1) and
// int64(1) merge into one entry during the build and every probe misses.

var m map[any]string

func init() {
	m = map[any]string{}
	m[int(1)] = "int"
	m[int64(1)] = "int64"
	m["k"] = "string"
}

func main(cur realm) {
	println(len(m))
	println(m[int(1)])
	println(m[int64(1)])
	println(m["k"])
	m[int32(1)] = "int32"
	println(len(m))
	delete(m, int64(1))
	_, ok := m[int64(1)]
	println(len(m), ok)
}

// Output:
// 3
// int
// int64
// string
// 4
// 3 false
```
</details>

## gnovm/tests/files/map51.gno:3-6 [gh](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/tests/files/map51.gno#L3-L6) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/map51.gno#L3-L6) [posted](https://github.com/gnolang/gno/pull/6020#discussion_r3839121060)
Missing test: every assertion here reads the stored value rather than the stored key, so deleting `mli.Key = key` from [`GetPointerForKey`](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values.go#L1066) leaves this file green.

<details><summary>test cases</summary>

Green at 764ac4d84, red on that same deletion while `map51.gno` stays green.

```go
package main

import "math"

// The stored key is replaced by the last one assigned, so the second write
// leaves a negative zero behind where the first left a positive one. A
// negative zero has to come from math.Copysign: the constant -0.0 is +0,
// as gnovm/tests/files/float9.gno documents.

func main() {
	negZero := math.Copysign(0, -1)
	m := map[float64]string{}
	m[0] = "positive"
	m[negZero] = "negative"
	for k, v := range m {
		println(len(m), math.Signbit(k), v)
	}
}

// Output:
// 1 true negative
```
</details>

## gnovm/pkg/gnolang/uverse.go:1249 [gh](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/uverse.go#L1249) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/uverse.go#L1249) [posted](https://github.com/gnolang/gno/pull/6020#discussion_r3839121093)
Suggestion: `delete` computes the map key twice, once here and once in [`DeleteForKey`](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values.go#L1104-L1117), which costs 104938 gas on a `[1<<18]byte` key. `DeleteForKey` can return the removed entry's value alongside its key, so this probe goes.

## SKIP gnovm/pkg/gnolang/values_test.go:433 [gh](https://github.com/Villaquiranm/gno/blob/chore/optimize-map-gas/gnovm/pkg/gnolang/values_test.go#L433) · [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values_test.go#L433)
Nit: `baseOf` decides nothing here, since `(*DeclaredType).Kind` returns `dt.Base.Kind()`. Dropped as comment-level.
