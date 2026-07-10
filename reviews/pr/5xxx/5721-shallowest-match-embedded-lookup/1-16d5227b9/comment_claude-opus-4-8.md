# Review: PR [#5721](https://github.com/gnolang/gno/pull/5721)
Event: APPROVE

## Body
Spec-compliant rewrite. Verified on 16d5227b9 that every ambiguity and shadowing case resolves as the Go compiler does. Reverting the fix reproduces the old deep-rescue on the merge-base. The change only touches the VM layer, behind the go/types type-check that AddPackage already runs. An ambiguous selector is rejected at deploy regardless, so no existing realm regresses. The red pages build is unrelated: pkgsite@latest now needs go 1.26.

<details><summary>parity repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5721 -R gnolang/gno

# gno: shallowest field wins (method40), same-depth siblings are ambiguous (method41)
go test -run 'TestFiles/method40.gno$' ./gnovm/pkg/gnolang/ >/dev/null && echo "gno method40 (shallowest wins): PASS"
go test -run 'TestFiles/method41.gno$' ./gnovm/pkg/gnolang/ >/dev/null && echo "gno method41 (siblings ambiguous): PASS"

# go: same two shapes, built in throwaway modules
amb=$(mktemp -d); ( cd "$amb" && go mod init x 2>/dev/null && cat > amb.go <<'EOF'
package main
type A struct{ Foo int }
type B struct{ Foo int }
type C struct{ A; B }
func main() { var c C; _ = c.Foo }
EOF
echo -n "go siblings: "; go build ./... 2>&1 | grep -o 'ambiguous selector c.Foo' ); rm -rf "$amb"

run=$(mktemp -d); ( cd "$run" && go mod init y 2>/dev/null && cat > main.go <<'EOF'
package main
type embedded string
func (s embedded) val() string { return string(s) }
type A struct{ embedded }
type B struct{ A; embedded }
func main() { b := &B{A: A{embedded: "a"}, embedded: "b"}; println(b.val()) }
EOF
echo -n "go shallowest field: "; go run . ); rm -rf "$run"
```

```
gno method40 (shallowest wins): PASS
gno method41 (siblings ambiguous): PASS
go siblings: ambiguous selector c.Foo
go shallowest field: b
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5721-shallowest-match-embedded-lookup/1-16d5227b9/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/types.go:1026 [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L1026)
Missing test: no case exercises VerifyImplementedBy when the concrete type's method is ambiguous. The rejection is correct, but the VM reports `main.C does not implement main.I (missing method Foo)` where go/types and the direct-selector path both say ambiguous. The status enum is discarded here with `_`.

<details><summary>test cases</summary>

Green today, guarding the correct rejection on this path. The golden line is where `var i I = C{}` lands in the file you drop it in.

```go
package main

type I interface{ Foo() string }
type A struct{}
func (A) Foo() string { return "a" }
type B struct{}
func (B) Foo() string { return "b" }
type C struct {
	A
	B
}
func main() {
	var i I = C{}
	_ = i
}

// Error:
// main/embed_iface_ambiguous.gno:<n>:6-15: main.C does not implement main.I (missing method Foo)
```
</details>

## gnovm/pkg/gnolang/types.go:3194-3204 [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L3194)
Suggestion: `fv.GetType(nil)` is called twice here, for `Params[0]` and for `BoundType`. The two-line NOTE comment is repeated verbatim. The deleted per-type lookup cached it once as `mt := fv.GetType(nil)`.

## gnovm/pkg/gnolang/types.go:2114 [↗](../../../../../.worktrees/gno-review-5721/gnovm/pkg/gnolang/types.go#L2114)
Nit: this doc comment and uverse.go:1658 still name `*DT.FindEmbeddedFieldType()`, the method this PR deletes. Both should name the free `findEmbeddedFieldType`.
