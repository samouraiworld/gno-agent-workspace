# Review: [#6272](https://github.com/gnolang/gno/pull/6272)
Posted: https://github.com/gnolang/gno/pull/6272#pullrequestreview-5548547488

Event: APPROVE
Verdict: APPROVE. The open findings are a Suggestion on three package-block shapes where `GetNameSourceForPath` still panics, a defect older than the branch that no caller reaches, and a test Nit; neither blocks.
Model: claude-opus-5-5, standard review, solo shape: 1 finder at xhigh, judge at high
Commit: 03363f296312b625bf4620b8a5fba0927983a1ed
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6272 03363f296`
Round: 2. One finder, a reflector pass by the judge that added no candidate, 3 candidates, each run again or read by an agent that was not its finder: 2 confirmed and posted, 1 plausible and shipped SKIP, 0 refuted. Round 1 at the same sha left finder candidates and no draft.

## Body

## gnovm/pkg/gnolang/getnamesource_test.go:20 [gh](https://github.com/gnolang/gno/blob/03363f296312b625bf4620b8a5fba0927983a1ed/gnovm/pkg/gnolang/getnamesource_test.go#L20) · [↗](../../../../../.worktrees/gno-review-6272/gnovm/pkg/gnolang/getnamesource_test.go#L20) · Nit [posted](https://github.com/gnolang/gno/pull/6272#discussion_r4295256657)

Test: `GetNameSourceForPath` never reads this store, because [`Block.GetSource`](https://github.com/gnolang/gno/blob/03363f296312b625bf4620b8a5fba0927983a1ed/gnovm/pkg/gnolang/values.go#L2971-L2977) consults it only for a `RefNode` source. Passing `nil` pins the same behaviour, so the `TestStore`, `NewPackage`, `SetBlockNode` and `SetCachePackage` calls and their imports can go.

## gnovm/pkg/gnolang/nodes.go:2085 [gh](https://github.com/gnolang/gno/blob/03363f296312b625bf4620b8a5fba0927983a1ed/gnovm/pkg/gnolang/nodes.go#L2085) · [↗](../../../../../.worktrees/gno-review-6272/gnovm/pkg/gnolang/nodes.go#L2085) · Suggestion [posted](https://github.com/gnolang/gno/pull/6272#discussion_r4295256662)

Suggestion: `nsrc.NameExpr` is nil for a name added by `Define`, and the embedded `FileSet` is nil for a package built without one. Uverse names ([`uverse.go`](https://github.com/gnolang/gno/blob/03363f296312b625bf4620b8a5fba0927983a1ed/gnovm/pkg/gnolang/uverse.go#L752-L756)) and the `MsgCall` main package ([`keeper.go`](https://github.com/gnolang/gno/blob/03363f296312b625bf4620b8a5fba0927983a1ed/gno.land/pkg/sdk/vm/keeper.go#L1260-L1261)) therefore panic here where the godoc promises a nil file, though no caller reaches either yet.

```suggestion
		if nsrc.NameExpr != nil && pn.FileSet != nil {
			fn, _, _ = pn.GetDeclForSafe(nsrc.NameExpr.Name)
		}
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6272 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_namesource_test.go <<'EOF'
package gnolang_test

import (
	"fmt"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func probe(t *testing.T, label string, pn *gno.PackageNode, path gno.ValuePath) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s: PANIC %v\n", label, r)
			t.Errorf("%s panicked", label)
		}
	}()
	_, fn, _ := pn.GetNameSourceForPath(nil, path)
	fmt.Printf("%s: ok fnNil=%v\n", label, fn == nil)
}

func blockPath(pn *gno.PackageNode, n gno.Name) gno.ValuePath {
	idx, _ := pn.GetLocalIndex(n)
	p := gno.ValuePath{Type: gno.VPBlock, Name: n}
	p.SetDepth(1)
	p.Index = idx
	return p
}

func TestNameSourceShapes(t *testing.T) {
	a := gno.NewPackageNode("a", "gno.land/r/a", &gno.FileSet{})
	a.Reserve(false, &gno.NameExpr{Name: "v"}, &gno.ValueDecl{}, gno.NSValueDecl, 0)
	probe(t, "empty FileSet, Reserve", a, blockPath(a, "v"))

	b := gno.NewPackageNode("b", "gno.land/r/b", &gno.FileSet{})
	b.Define("pkg", gno.TypedValue{T: &gno.PackageType{}})
	probe(t, "empty FileSet, Define", b, blockPath(b, "pkg"))

	c := gno.NewPackageNode("c", "gno.land/r/c", nil)
	c.Reserve(false, &gno.NameExpr{Name: "w"}, &gno.ValueDecl{}, gno.NSValueDecl, 0)
	probe(t, "nil FileSet, Reserve", c, blockPath(c, "w"))

	u := gno.UverseNode()
	idx, _ := u.GetLocalIndex("true")
	probe(t, "uverse true", u, gno.ValuePath{Type: gno.VPUverse, Name: "true", Index: idx})
}
EOF
go test -count=1 -run TestNameSourceShapes -v ./gnovm/pkg/gnolang | grep -E 'ok fnNil|PANIC|^(ok|FAIL)'
rm gnovm/pkg/gnolang/zz_namesource_test.go
```

Three of the four shapes panic where the godoc says the file comes back nil:

```text
empty FileSet, Reserve: ok fnNil=true
empty FileSet, Define: PANIC runtime error: invalid memory address or nil pointer dereference
nil FileSet, Reserve: PANIC runtime error: invalid memory address or nil pointer dereference
uverse true: PANIC runtime error: invalid memory address or nil pointer dereference
FAIL
```

With the suggestion applied, all four print `ok fnNil=true`, and `TestGetNameSourceForPathNoFilesetDecl` and `go test -short ./gnovm/pkg/repl` pass. The three panics predate the branch: the merge base panics the same way on them.

</details>

## SKIP gnovm/pkg/gnolang/nodes.go:2085 [gh](https://github.com/gnolang/gno/blob/03363f296312b625bf4620b8a5fba0927983a1ed/gnovm/pkg/gnolang/nodes.go#L2085) · [↗](../../../../../.worktrees/gno-review-6272/gnovm/pkg/gnolang/nodes.go#L2085) · Suggestion

Suggestion: discarding `ok` here drops the miss assertion for packages built from files too, returning a nil file on a desync. The assertion survives if the tolerance applies only to an empty `pn.Files`.

Not posted: plausible only; no caller of `GetNameSourceForPath` over a file-built package exists at the head, so no run shows whether a file-built name can legitimately lack a decl.
