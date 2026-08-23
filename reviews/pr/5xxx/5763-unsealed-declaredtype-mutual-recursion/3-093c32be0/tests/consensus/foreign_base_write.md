# `fillTypeInPlace` writes bases the declaring package does not own

`preprocess.go:3077` runs for every non-alias `*DeclaredType` declaration, not
only the mutual-recursion ones. When the declaration is `type Local Foreign`,
`declareWith` set `dstT.Base = baseOf(Foreign)` at predefine time and sets
`tmp2.Base = baseOf(Foreign)` again at `TRANS_LEAVE`, so the two are the *same
pointer* and `fillTypeInPlace` executes `*dst = *dst`. The value does not
change; the write does. Before this PR that pointer was never written.

Two shapes reach a pointer outside the declaring package:

| Source | `dst` | Owner |
| --- | --- | --- |
| `type MyTree avl.Tree` | `*StructType` | `gno.land/p/nt/avl/v0`, shared by every package in the store that imported avl |
| `type MyErr error` | `*InterfaceType` | the process-global uverse singleton (`uverseValue`, `uverse.go:483`) |

## Measurement

Probe applied at the fill site, printing `dst == src`, whether `dst` is a
uverse base, and the base's own `PkgPath`:

```go
if dstT.Base != nil {
	zzsame := dstT.Base == tmp2.Base
	zzuv := zzIsUverseBase(dstT.Base)
	if fillTypeInPlace(dstT.Base, tmp2.Base) {
		zzLog(ctxpn.PkgPath, string(n.Name), zzsame, zzuv, dstT.Base)
		tmp2.Base = dstT.Base
	}
}
```

Over 137 packages under `examples/gno.land` the fill site fires 10063 times.
All 10063 write a base whose `PkgPath` equals the declaring package, because no
examples package declares a type derived from another package's named type. The
two rows below come from a package written for this measurement:

```
ZZFILL	same=true	uverse=""		declPkg=gno.land/p/zz/derived	name=MyTree	baseKind=StructKind	basePkg=gno.land/p/nt/avl/v0
ZZFILL	same=true	uverse="uverse:error"	declPkg=gno.land/p/zz/derived	name=MyErr	baseKind=InterfaceKind	basePkg=.uverse
```

`same=true` on both: the helper copies a struct onto itself.

## Consequence

No value changes and no state diverges: the corpus differential
(`corpus_result.txt`) is byte-identical across 343 types. What changes is that a
pointer shared between packages, and in the `error` case between every store and
every goroutine in the process, is now written during preprocessing.

Two goroutines preprocessing `type E error` at the same time race on it.
Measured with `uverse_base_race_test.go`:

| Tree | `WARNING: DATA RACE` | naming `fillTypeInPlace` |
| --- | --- | --- |
| merge-base `0397fc87f` | 2 | 0 |
| head `093c32be0` | 3 | 2 |
| head + the guard below | 2 | 0 |

The two on the merge-base are a pre-existing `(*DeclaredType).TypeID()`
memoization race reached through `InitStoreCaches`. The two new ones are
`types.go:1577` (`*dst = *src` for `*InterfaceType`) writing the uverse
singleton, once against a concurrent `embedDepth()` read of the same
`*InterfaceType` and once against another `fillTypeInPlace` write. Reports in
`race_base.txt`, `race_head.txt`, `race_guard.txt`.

gno.land executes transactions serially, so no consensus path reaches this. The
in-repo parallel surface is `TestFiles`, whose long filetests run with
`t.Parallel()` and their own stores but the same process-global uverse
(`files_test.go:106-108`); none of them declares a type derived from a uverse or
cross-package named type, so `go test -race ./gnovm/pkg/gnolang/` on the head is
clean (0 races, 2636s). The reproduction has to construct the concurrency. gno
CI runs no `-race` at all, so no job would report it either way.

## Fix

Add the pointer-inequality test the helper's own doc comment already assumes:

```go
if dstT.Base != nil && dstT.Base != tmp2.Base && fillTypeInPlace(dstT.Base, tmp2.Base) {
	tmp2.Base = dstT.Base
}
```

When the two are the same pointer both statements are no-ops, so the guard is
behaviour-preserving. Verified: the 13-shape base matrix is unchanged, the
corpus differential against the merge-base is identical with the guard applied
(`corpus_result.txt`, second diff), and the two new races disappear.
