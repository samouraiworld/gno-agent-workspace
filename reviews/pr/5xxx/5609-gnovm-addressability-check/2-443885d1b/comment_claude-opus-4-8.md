# Review: PR #5609
Event: REQUEST_CHANGES

## Body
Nice cleanup; moving these from runtime panics to preprocess errors is the right direction. The branch isn't mergeable against master, but the conflict is mechanical (`RefExpr`/`SliceExpr` are unchanged upstream), so a rebase clears it. Two more classes slip past `isAddressable` on top of the ones already noted: `&funcName` for a package-level function builds a usable `*func()`, and `_ = &nil` crashes the preprocessor instead of erroring cleanly.

Verified on 443885d1b: `&funcName` runs to completion under the VM where `go build` rejects it, and `_ = &nil` panics with a nil-pointer dereference in the error path.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## SKIP gnovm/pkg/gnolang/preprocess.go:4395 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4395)
Already raised: https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223 (review body, not reactable)
`&[3]int{1,2,3}[0]`, `&struct{ x int }{}.x`, and `[3]int{1,2,3}[:]` all build pointers or slices at runtime where Go rejects them, because this arm reports any `*CompositeLitExpr` addressable. A composite literal is addressable only as the direct `&T{}` operand, not as an index/slice/selector base. Fix: treat a composite literal reached by recursion as non-addressable.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
func main() {
	a := &[3]int{1, 2, 3}[0]
	b := &struct{ x int }{}.x
	c := [3]int{1, 2, 3}[:]
	println(*a, *b, len(c))
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
1 0 3
```
Go rejects all three: `cannot take address of [3]int{…}[0]`, `cannot take address of struct{x int}{}.x`, `cannot slice unaddressable value [3]int{…}`.
</details>

## gnovm/pkg/gnolang/preprocess.go:4395 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4395)
`&funcName` for a package-level function builds a usable `*func()` at runtime; this arm reports any `*NameExpr` addressable regardless of type. Go rejects taking the address of a function. Fix: reject a name whose static type is a function or type.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
func bar() {}
func main() {
	p := &bar
	_ = p
	println("ok")
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
ok
```
Go: `invalid operation: cannot take address of bar (value of type func())`.
</details>

## SKIP gnovm/pkg/gnolang/preprocess.go:4397-4404 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4397-L4404)
Already raised: https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223 (review body, not reactable)
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

## SKIP gnovm/pkg/gnolang/preprocess.go:4405-4409 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4405-L4409)
Already raised: https://github.com/gnolang/gno/pull/5609#pullrequestreview-4389750223 (review body, not reactable)
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

## gnovm/pkg/gnolang/preprocess.go:2344 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L2344)
This gate runs for an explicit `&x`, but the receiver-address the VM synthesizes for a pointer-method call (preprocess.go:2404/2432) is marked `setPreprocessed`, so it never reaches here. As a result `m["k"].Inc()` and `T{1}.Inc()` are accepted, where Go rejects calling a pointer method on a non-addressable value, and the map case mutates the stored element. Fix: run the same `isAddressable` check before synthesizing the receiver address at 2404 and 2432.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno
tmp=$(mktemp -d)
cat > "$tmp/main.gno" <<'EOF'
package main
type T struct{ n int }
func (t *T) Inc() { t.n++ }
func main() {
	m := map[string]T{"k": {1}}
	m["k"].Inc()
	println(m["k"].n)
}
EOF
go run ./gnovm/cmd/gno run "$tmp/main.gno"
rm -rf "$tmp"
```

```
2
```
Go: `cannot call pointer method Inc on T`.
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

## gnovm/pkg/gnolang/preprocess.go:4393-4413 [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/preprocess.go#L4393-L4413)
`isAddressable` re-walks the node kinds that [`assertValidAssignLhs`](https://github.com/gnolang/gno/blob/443885d1b/gnovm/pkg/gnolang/type_check.go#L1019) · [↗](../../../../../.worktrees/gno-review-5609/gnovm/pkg/gnolang/type_check.go#L1019) already classifies for assignment targets, and that sibling already guards a nil static type and rejects a string index, the two cases this helper gets wrong. Borrowing those two arms and keeping the checks together would fix both and avoid the two classifiers drifting apart.
