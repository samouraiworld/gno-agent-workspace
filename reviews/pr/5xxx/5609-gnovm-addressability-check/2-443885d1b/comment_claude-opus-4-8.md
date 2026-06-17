# Review: PR #5609
Event: REQUEST_CHANGES

## Body
Nice cleanup; moving these from runtime panics to preprocess errors is the right direction. Two things before it lands. The branch isn't mergeable against master, but the conflict is mechanical (`RefExpr`/`SliceExpr` are unchanged upstream), so a rebase clears it. And `isAddressable` matches Go only for the classes this PR targeted: several expressions Go rejects still build a usable pointer at runtime, each closable locally in the helper.

Verified on 443885d1b: every flagged expression runs to a printed value under the VM while `go build` of the same source fails with `cannot take address` / `cannot slice unaddressable value`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/preprocess.go:4395 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4395)
`&[3]int{1,2,3}[0]`, `&struct{ x int }{}.x`, `[3]int{1,2,3}[:]`, and `&funcName` all build pointers at runtime where Go rejects them, because this arm accepts any `*CompositeLitExpr` or `*NameExpr`. A composite literal is addressable only as the direct `&T{}` operand, not as an index/slice/selector base, and a function name is never addressable. Fix: treat a composite literal reached by recursion as non-addressable, and reject a name whose static type is a function or type.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
func bar() {}
func main() {
	a := &[3]int{1, 2, 3}[0]
	b := &struct{ x int }{}.x
	c := [3]int{1, 2, 3}[:]
	d := &bar
	_ = d
	println(*a, *b, len(c), "ok")
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
1 0 3 ok
```
Go rejects all four: `cannot take address of [3]int{…}[0]`, `cannot take address of struct{x int}{}.x`, `cannot slice unaddressable value [3]int{…}`, `cannot take address of bar`.
</details>

## gnovm/pkg/gnolang/preprocess.go:4397-4404 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4397-L4404)
`&s[0]` on a string builds a byte pointer: a `StringKind` index isn't handled by the `Kind()` switch and falls through to the recursive call, which returns true for a name base. Go rejects taking the address of a string byte. Fix: reject a string index.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
func main() {
	s := "abc"
	p := &s[0]
	println(*p)
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
97
```
Go: `invalid operation: cannot take address of s[0] (value of type byte)`.
</details>

## gnovm/pkg/gnolang/preprocess.go:4405-4409 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4405-L4409)
`&t.M` on a bound method value builds a pointer: the SelectorExpr arm recurses on the receiver without inspecting the selector, so a method value is treated like a field. Go rejects taking the address of a method value. Fix: reject a selector whose static type is a function.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
type T struct{ x int }
func (t T) M() int { return t.x }
func main() {
	t := T{x: 42}
	p := &t.M
	println((*p)())
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
42
```
Go: `invalid operation: cannot take address of t.M (value of type func() int)`.
</details>

## gnovm/pkg/gnolang/preprocess.go:2347 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2347)
For untyped `nil`, `evalStaticTypeOf` returns nil and `xt.String()` here dereferences it, so `_ = &nil` crashes the preprocessor with a nil-pointer panic instead of a clean error. Fix: guard the nil type and return a `cannot take address of nil` error.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
func main() {
	_ = &nil
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
	panic: runtime error: invalid memory address or nil pointer dereference:
	--- preprocess stack ---
	stack 2: func main() { _<VPInvalid(0)> = &((const (undefined))) }
```
</details>
