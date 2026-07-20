# Review: PR [#5969](https://github.com/gnolang/gno/pull/5969)
Posted: https://github.com/gnolang/gno/pull/5969#pullrequestreview-4733409931
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Both shapes flagged inline reach the same `default:` branch in [`applyPointerDeref`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L731-L732), so one guard in the phase that decides what promotes closes both.

Verified on 7e0728bd5: compiled each shape with the Go compiler and compared its verdict against GnoVM's. They agree on every shape the diff covers.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5969-defined-pointer-type-semantics/1-7e0728bd5/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/types.go:2564-2566 [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L2564-L2566) [posted](https://github.com/gnolang/gno/pull/5969#discussion_r3613027671)
`type S struct{ *I }` with `I` an interface passes this guard, and selecting `s.M` through it panics `should not happen`. Go rejects it at declaration with `embedded field type cannot be a pointer to an interface`, the sibling of the rule ported here. It reproduces on master too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5969 -R gnolang/gno
cat > gnovm/tests/files/struct65.gno <<'EOF'
package main

type I interface{ M() string }

type S struct{ *I }

func main() {
	var s S
	_ = s.M
}

// Error:
// main/struct65.gno:5:8-18: embedded field type cannot be a pointer to an interface: *main.I

// TypeCheckError:
// main/struct65.gno:5:16: embedded field type cannot be a pointer to an interface
EOF
go test -run 'TestFiles/struct65.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/struct65.gno
```

```
--- FAIL: TestFiles/struct65.gno (0.00s)
    files_test.go:135: Error diff:
        --- Expected
        +++ Actual
        @@ -1 +1 @@
        -main/struct65.gno:5:8-18: embedded field type cannot be a pointer to an interface: *main.I
        +main/struct65.gno:9:6-9: should not happen
        # …
        gnolang.applyPointerDeref(…) types.go:732
        gnolang.buildEmbeddedTrail(…) types.go:3226
```
</details>

## gnovm/pkg/gnolang/types.go:3129-3133 [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3129-L3133) [posted](https://github.com/gnolang/gno/pull/5969#discussion_r3613027674)
`p.A` for `p` of type `*D1` panics `should not happen` instead of reporting a missing field, here and on master. Go rejects it because the `(*x).f` shorthand does not apply when the operand is itself a pointer type. The root strip in [`findEmbeddedFieldType`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L2927-L2932) turns `*D1` into `D1` before the walk starts, so the crossing is entered one level too late and the field is reported found.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5969 -R gnolang/gno
cat > gnovm/tests/files/ptr14.gno <<'EOF'
package main

type D2 struct{ A int }

type D1 *D2

func main() {
	var p *D1
	_ = p.A
}

// Error:
// main/ptr14.gno:9:6-9: missing field A in *main.D1

// TypeCheckError:
// main/ptr14.gno:9:8: p.A undefined (type *D1 has no field or method A)
EOF
go test -run 'TestFiles/ptr14.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/ptr14.gno
```

```
--- FAIL: TestFiles/ptr14.gno (0.02s)
    files_test.go:135: Error diff:
        --- Expected
        +++ Actual
        @@ -1 +1 @@
        -main/ptr14.gno:9:6-9: missing field A in *main.D1
        +main/ptr14.gno:9:6-9: should not happen
        # …
        gnolang.applyPointerDeref(…) types.go:732
        gnolang.buildEmbeddedTrail(…) types.go:3226
```
</details>

## gnovm/pkg/gnolang/types.go:3153-3160 [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3153-L3160) [posted](https://github.com/gnolang/gno/pull/5969#discussion_r3613027677)
Missing test: nothing asserts that a defined pointer type no longer satisfies an interface through its base's methods. `VerifyImplementedBy` resolves through this same lookup, so suppressing methods past the crossing changes assignability, and [`method47`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/method47.gno#L1) through [`method50`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/method50.gno#L1), [`ptr12`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/ptr12.gno#L1), [`ptr13`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/ptr13.gno#L1) and [`struct64`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/struct64.gno#L1) cover selectors and declarations only.

<details><summary>test cases</summary>

`gnovm/tests/files/method51.gno`, panics `should not happen` at the merge base 959cefd91, green at 7e0728bd5:

```go
package main

type D2 struct{}

func (D2) Foo() string { return "m" }

type D1 *D2

type Fooer interface{ Foo() string }

func main() {
	d2 := D2{}
	var x D1 = &d2
	var f Fooer = x
	println(f.Foo())
}

// Error:
// main/method51.gno:14:6-17: main.D1 does not implement main.Fooer (missing method Foo)

// TypeCheckError:
// main/method51.gno:14:16: cannot use x (variable of pointer type D1) as Fooer value in variable declaration: D1 does not implement Fooer (missing method Foo)
```
</details>

## gnovm/tests/files/struct64b.gno:10 [↗](../../../../../.worktrees/gno-review-5969/gnovm/tests/files/struct64b.gno#L10) [posted](https://github.com/gnolang/gno/pull/5969#discussion_r3613027686)
Missing test: nothing covers the legal spelling the new embedded-pointer rejection must not catch, an alias of a pointer type. `type P = *D2; type S struct{ P }` is legal Go and reaches the same [guard](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L2564-L2566), which runs on every embedded field of every struct in every package.

<details><summary>test cases</summary>

`gnovm/tests/files/struct64c.gno`, green at 7e0728bd5:

```go
package main

type D2 struct{ A int }

type P = *D2 // alias of a pointer type: legal as an embedded field

type S struct{ P }

func main() {
	var s S
	s.P = &D2{A: 3}
	println(s.A)
}

// Output:
// 3
```
</details>
