# PR [#6152](https://github.com/gnolang/gno/pull/6152): fix(amino): deep-copy slices instead of sharing backing array

Verdict: APPROVE. The slice fix is correct and the new guard keeps a nil slice nil; the two Suggestions predate this branch, reach no current caller and block nothing.
Event: COMMENT
Model: claude-opus-5-5 at high effort, quick review, solo shape
Commit: a74212699
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/a742126999443305c6bfcf6fd6364ea57fba51d4) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/a742126999443305c6bfcf6fd6364ea57fba51d4)
Round: 1. One finder, no separate reflector stage, 4 candidates: 3 run from scratch at the head and the merge base by an agent that was not their finder, 1 Nit left on the finder's read.

## Body

> AI review, claude-opus-5-5, quick review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6152-amino-deepcopy-slices/overview.md) · Status: APPROVE · not manually verified, posted to help reviewers

- Related suggestion: the [`Map` case](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L136-L137) hands `SetMapIndex` the source value itself, so a copied map of slices still shares each slice with its source.

<details>
<summary>repro</summary>

```go
// tm2/pkg/amino/zz_map_test.go
// go test ./tm2/pkg/amino -run TestMapValues -v -count=1
package amino_test

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
)

type mapHolder struct{ M map[string][]byte }

func TestMapValues(t *testing.T) {
	src := mapHolder{M: map[string][]byte{"a": {1}}}
	cpy := amino.DeepCopy(src).(mapHolder)
	src.M["a"][0] = 9
	t.Logf("map value aliased: %v", cpy.M["a"][0] == 9)
}
```

A write to the source's slice shows up in the copy:

```
map value aliased: true
```

The merge base prints the same line. With the three lines below in place of `cpy.SetMapIndex(key, val)` it prints `map value aliased: false` and `go test ./tm2/pkg/amino/...` stays green. No type reached from today's callers (`RoundState`, `Header`, `ConsensusParams`, `ConsensusConfig`) holds a map.

```go
			val := reflect.New(src.Type().Elem()).Elem()
			deepCopy(src.MapIndex(key), val)
			cpy.SetMapIndex(key, val)
```
</details>

## tm2/pkg/amino/deep_copy.go:87-90 [gh](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L87-L90) · Suggestion [posted](https://github.com/gnolang/gno/pull/6152#discussion_r4106609544)

Suggestion: this guard covers a nil slice alone, while the [`Pointer` case](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L55-L59) skips the [`isNil` check](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L41-L43) for a nil map, pointer or interface, so `DeepCopy(&p)` panics. One `if isNil(src) { return }` at the top of `_deepCopy`, in place of this guard, copies each of them as nil.

<details>
<summary>repro</summary>

```go
// tm2/pkg/amino/zz_nil_test.go
// go test ./tm2/pkg/amino -run TestNilBehindPointer -v -count=1
package amino_test

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
)

func TestNilBehindPointer(t *testing.T) {
	try := func(name string, f func() bool) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("%s: PANIC %v", name, r)
			}
		}()
		t.Logf("%s: copy nil=%v", name, f())
	}
	var s []int
	try("*[]int", func() bool { return *amino.DeepCopy(&s).(*[]int) == nil })
	var m map[string]int
	try("*map", func() bool { return *amino.DeepCopy(&m).(*map[string]int) == nil })
	var p *int
	try("**int", func() bool { return *amino.DeepCopy(&p).(**int) == nil })
	var i any
	try("*any", func() bool { return *amino.DeepCopy(&i).(*any) == nil })
}
```

Only the slice line copies its nil as nil; the other three either come back non-nil or panic:

```
*[]int: copy nil=true
*map: copy nil=false
**int: PANIC unsupported type invalid
*any: PANIC reflect: call of reflect.Value.Type on zero Value
```

The merge base prints the same four lines. With the guard moved to the top of `_deepCopy` all four print `copy nil=true`, and `go test ./tm2/pkg/amino/... ./tm2/pkg/sdk/... ./tm2/pkg/bft/types/...` stays green:

```diff
 func _deepCopy(src, dst reflect.Value) {
+	if isNil(src) {
+		return
+	}
 	switch src.Kind() {
@@
 	case reflect.Slice:
-		if src.IsNil() {
-			dst.Set(src)
-			return
-		}
```
</details>

## SKIP tm2/pkg/amino/deep_copy.go:111 [gh](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L111) · Nit

Nit: `DeepCopy` zeroes the unexported fields of each slice element, which go through the [`Struct` case](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L122-L124) like every other struct, and no doc comment says so.

Not posted: the only fix is a doc sentence.

## SKIP tm2/pkg/amino/deep_copy_test.go:172 [gh](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy_test.go#L172) · Nit

Test: `DeepCopy(nilInts)` returns at the [`isNil` check](https://github.com/gnolang/gno/blob/a74212699/tm2/pkg/amino/deep_copy.go#L41-L43) before `_deepCopy` runs, so this assertion holds at the merge base too; the `*cpyPtr` one pins the new guard.

Not posted: the `*cpyPtr` assertion already covers the guard, so no test is missing.
