# Review: PR [#5978](https://github.com/gnolang/gno/pull/5978)
Event: COMMENT

## Body
Verified on da74644bf: replacing [the line that sets `gof.GoVersion`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L645) [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L645) with a comment makes all four assertions of [the new test](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L36-L78) [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L36-L78) fail, and restoring it turns them green. Without that line, on a binary built with Go 1.26.5, `//go:build go1.26` is accepted while `//go:build go1.27` is rejected as `file requires newer Go version go1.27 (application built with go1.26)`, so the boundary is the compiling toolchain and [`go 1.25.9` in go.mod](https://github.com/gnolang/gno/blob/da74644bf/go.mod#L3) [↗](../../../../../.worktrees/gno-review-5978/go.mod#L3) leaves a Go 1.25 node and a Go 1.26 node disagreeing on `go1.26`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5978-pin-per-file-go-version/1-da74644bf/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/gotypecheck_buildtag_test.go:44-61 [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck_buildtag_test.go#L44-L61)
Missing test: a `//go:build go1.N` line inside an imported package raises that import's own version too, and no case covers it. A package importing a dependency whose file starts with `//go:build go1.22` and contains `for range 10` type-checks clean without the blanking and is rejected with it.

<details><summary>test cases</summary>

```go
type buildTagImportGetter map[string]*std.MemPackage

func (g buildTagImportGetter) GetMemPackage(path string) *std.MemPackage {
	return g[path]
}

func TestTypeCheckMemPackage_BuildTagOnImport(t *testing.T) {
	t.Parallel()

	dep := &std.MemPackage{
		Type: MPUserProd,
		Name: "dep",
		Path: "gno.land/p/demo/dep",
		Files: []*std.MemFile{{Name: "dep.gno", Body: "//go:build go1.22\n\n" +
			"package dep\nfunc G() { for range 10 {} }\n"}},
	}
	root := &std.MemPackage{
		Type: MPUserProd,
		Name: "z",
		Path: "gno.land/p/demo/z",
		Files: []*std.MemFile{{Name: "z.gno", Body: "package z\n" +
			"import \"gno.land/p/demo/dep\"\nfunc F() { dep.G() }\n"}},
	}

	getter := buildTagImportGetter{dep.Path: dep}
	_, err := TypeCheckMemPackage(root, TypeCheckOptions{
		Getter:     getter,
		TestGetter: getter,
		Mode:       TCLatestRelaxed,
	})
	assert.ErrorContains(t, err, "go1.22",
		"a //go:build line in an imported package must not raise the pinned "+
			"GoVersion for that import")
}
```
</details>

## gnovm/pkg/gnolang/gotypecheck.go:645 [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L645)
Nit: [`GoParseMemPackage`](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L587) [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L587) is exported, and its [doc comment](https://github.com/gnolang/gno/blob/da74644bf/gnovm/pkg/gnolang/gotypecheck.go#L580-L586) [↗](../../../../../.worktrees/gno-review-5978/gnovm/pkg/gnolang/gotypecheck.go#L580-L586) lists what comes back without saying every returned file has had its Go version cleared. A caller reading godoc gets ASTs that no longer match the source they were parsed from.
