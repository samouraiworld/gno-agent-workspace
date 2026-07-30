/* Run: from a gno checkout:
gh pr checkout 5991 -R gnolang/gno && git checkout 65ae435d4
curl -fsSL -o gnovm/pkg/gnolang/declaredtype_typeid_format_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5991-drop-redundant-typeid-recompute/1-65ae435d4/tests/declaredtype_typeid_format_test.go
go test -v -run 'TestDeclaredTypeIDWrittenForm' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/declaredtype_typeid_format_test.go
*/

// The type id of a named type keys stored types in the backend and decides
// typed equality, so both of its written forms are consensus relevant. Nothing
// in the tree pins them: the only in-tree check was the recompute the PR
// deletes. Passes at the pinned hash; fails if either written form changes, if
// the memo stops agreeing with a fresh computation, or if String() drifts from
// TypeID().
package gnolang

import "testing"

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
