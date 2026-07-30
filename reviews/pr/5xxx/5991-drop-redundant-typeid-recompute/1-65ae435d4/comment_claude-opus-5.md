# Review: PR [#5991](https://github.com/gnolang/gno/pull/5991)
Event: APPROVE

## Body
Ran the two new benchmarks against merge base d1a33f574 and head 65ae435d4. The cached call goes from 3 allocations to zero for a package-level type, and from 13 to zero for a function-level one.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5991-drop-redundant-typeid-recompute/1-65ae435d4/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go:5-11 [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L5-L11)
Missing test: nothing in the tree pins either written form of a declared type's identity, and the recompute this PR deletes was the only in-tree check tying `dt.typeid` to [`DeclaredTypeID`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/types.go#L1957-L1963). That identity keys stored type definitions through [`backendTypeKey`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/store.go#L1451-L1453) and decides typed equality. Filetest goldens pin only the `pkgPath[loc].Name` form, and they reach it through [`DeclaredType.String`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/types.go#L1965-L1971), which rebuilds the string in separate code.

<details><summary>test cases</summary>

```go
func TestDeclaredTypeIDWrittenForm(t *testing.T) {
	t.Parallel()
	funcLoc := Location{
		PkgPath: "gno.land/r/demo/boards",
		File:    "boards.gno",
		Span:    Span{Pos: Pos{Line: 42, Column: 3}, End: Pos{Line: 50, Column: 1}},
	}
	tests := []struct {
		name string
		dt   *DeclaredType
		want TypeID
	}{
		{
			name: "package level",
			dt:   &DeclaredType{PkgPath: "gno.land/r/demo/boards", Name: "BoardID"},
			want: "gno.land/r/demo/boards.BoardID",
		},
		{
			name: "function level",
			dt: &DeclaredType{
				PkgPath:   "gno.land/r/demo/boards",
				Name:      "localType",
				ParentLoc: funcLoc,
			},
			want: "gno.land/r/demo/boards[gno.land/r/demo/boards/boards.gno:42:3-50:1].localType",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.dt.TypeID(); got != tt.want {
				t.Fatalf("first TypeID() = %q, want %q", got, tt.want)
			}
			// Second call reads the memo; it has to agree with a fresh
			// computation from the same three fields.
			fresh := DeclaredTypeID(tt.dt.PkgPath, tt.dt.ParentLoc, tt.dt.Name)
			if got := tt.dt.TypeID(); got != fresh {
				t.Fatalf("memoized TypeID() = %q, recomputed %q", got, fresh)
			}
			// String() builds the same written form through separate code and
			// is the form the filetest goldens pin; keep the two together.
			if got := TypeID(tt.dt.String()); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
```
</details>

## gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go:30-32 [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L30-L32)
Nit: the comment says three `fmt.Sprintf` for the function-level benchmark, but [the fixture's span](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L40-L43) runs from line 42 to line 50, which takes [`Span.String`'s multi-line branch](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/nodes_location.go#L192-L196) and costs five. A `ParentLoc` is a function location, so multi-line is the ordinary case. The [same comments](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/declaredtype_typeid_bench_test.go#L5-L16) also frame both benchmarks against a baseline that stops existing at merge.

<details><summary>counts</summary>

Instrumenting `typeidf`, [`Location.String`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/nodes_location.go#L304-L317), `Span.String` and [`Pos.String`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/nodes_location.go#L83-L85) gives 1 for a zero `ParentLoc`, 3 for a same-line span and 5 for this fixture.
</details>

## gnovm/pkg/gnolang/types.go:1931-1936 [↗](../../../../../.worktrees/gno-review-5991/gnovm/pkg/gnolang/types.go#L1931-L1936)
Suggestion: nothing else compares `dt.typeid` against the three fields it was built from, so a future rewrite of `PkgPath`, `ParentLoc` or `Name` on a live [`DeclaredType`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/types.go#L1429-L1437) silently keeps a stale identity instead of aborting. The store keeps an identity check of the same class rather than dropping it, [gated on `debugAssert` where a decoded type is compared against the key it was loaded under](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/store.go#L826-L831). [`debug.go`](https://github.com/gnolang/gno/blob/65ae435d4/gnovm/pkg/gnolang/debug.go#L24-L28) names the remaining `if debug { panic }` sites elsewhere as candidates for that tag rather than removal.
