# Review: PR [#6020](https://github.com/gnolang/gno/pull/6020)
Event: APPROVE

## Body
The merge-base number for [`compute_map_key_concrete_key.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/gas/compute_map_key_concrete_key.gno#L22) is 135249. The summary table gives 125449, which is the value after the write-dedup commit, so the golden's real delta against d1a33f574 is 7.7%. Of the 10400, 9800 is the dropped second [`ComputeMapKey`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1881) per write and 600 is the prefix.

Checked on 764ac4d84: deleting the [`mv.ensureVmap` line](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1057), the [`fillMapKeyRefs` line before the copy](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L2461), or [the same line in the map-literal path](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/op_expressions.go#L587) each reddens exactly one of the new realm filetests, a different one each time.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6020-compute-map-keys-once/1-764ac4d84/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gnovm/pkg/gnolang/values.go:1031 [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1031)
Missing test: an interface-keyed map read back from the store, where the lazy build has to keep the TypeID prefix. [`map52.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/map52.gno) covers interface keys inside one execution and [`zrealm_map6.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/zrealm_map6.gno) covers the round trip for a concrete key, but no `// PKGPATH:` filetest has an interface-kinded key type. A build that dropped the prefix there would merge `int(1)` and `int64(1)` into one entry and miss every probe with nothing red.

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

## gnovm/tests/files/map51.gno:3-6 [↗](../../../../../.worktrees/gno-review-6020/gnovm/tests/files/map51.gno#L3-L6)
Missing test: a stored key that is actually observed to change. Deleting `mli.Key = key` from [`GetPointerForKey`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1066) leaves this file green, because every assertion here reads the stored value rather than the stored key. The untyped constant `-0.0` is `+0`, as [`float9.gno`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/tests/files/float9.gno#L1-L3) documents, so the map never receives a negative zero either.

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

## gnovm/pkg/gnolang/values_test.go:433 [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values_test.go#L433)
Nit: the `baseOf` unwrap in [the predicate](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1871-L1872) decides nothing. [`(*DeclaredType).Kind`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/types.go#L1515-L1517) returns `dt.Base.Kind()`, so `baseOf(mt.Key).Kind()` and `mt.Key.Kind()` agree for every input. Dropping it changes no result across [`TestMapKeyOmitType`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values_test.go#L401) and every map and gas filetest this PR adds.

## gnovm/pkg/gnolang/uverse.go:1249 [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/uverse.go#L1249)
Suggestion: `delete` still computes the map key twice, once here and once in [`DeleteForKey`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1104-L1117). On a `[1<<18]byte` key that second call is 104938 gas, a fifth of the whole program. `DeleteForKey` already returns nil when the key is absent, which is the only thing this probe decides.

## gnovm/pkg/gnolang/values.go:1037 [↗](../../../../../.worktrees/gno-review-6020/gnovm/pkg/gnolang/values.go#L1037)
Suggestion: [#5710](https://github.com/gnolang/gno/pull/5710) drops the `*Machine` parameter from [`ComputeMapKey`](https://github.com/gnolang/gno/blob/764ac4d84/gnovm/pkg/gnolang/values.go#L1881) and reads the meter off the `Store` instead. The nil `*Machine` here is what keeps the whole-map build free. Merged after this, #5710 turns that build into a metered first-touch spike sized by the whole map, and it lands silently because the nil argument still compiles.
