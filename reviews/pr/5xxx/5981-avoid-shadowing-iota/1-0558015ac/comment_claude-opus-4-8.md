# Review: PR [#5981](https://github.com/gnolang/gno/pull/5981)
Event: REQUEST_CHANGES

## Body
Verified on 0558015ac: [`gno lint`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/cmd/gno/lint.go#L39) rejects a package whose exported function takes an `iota` parameter, [as `gnoPreprocessError`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/cmd/gno/common.go#L125-L129), matching the VM even though `go/types` accepts the source. Nested `const` groups still evaluate `iota` exactly as the Go compiler does.

`main / build` is red only on formatting: four of the new filetests end with a trailing blank line.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5981-avoid-shadowing-iota/1-0558015ac/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/nodes.go:2318-2321 [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2318-L2321)
A three-clause `for` init never reaches this check. [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L284-L300) renames those names, and [their body references](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L350-L366), to `<name>.loopvar`, and it [runs before](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L181-L183) [`Reserve`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/nodes.go#L2317) sees them. So `for iota := 0; iota < 2; iota++ { println(iota) }` prints 0 and 1, while [`for iota := range s`](https://github.com/gnolang/gno/blob/0558015ac/gnovm/tests/files/iota_identifier_range.gno#L5) is rejected.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5981 -R gnolang/gno
cat > gnovm/tests/files/iota_forinit.gno <<'EOF'
package main

func main() {
	for iota := 0; iota < 2; iota++ {
		println(iota)
	}
}

// Error:
// main/iota_forinit.gno:4:6-14: builtin identifiers cannot be shadowed: iota
EOF
go test -run 'TestFiles/iota_forinit.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/iota_forinit.gno
```

```
--- FAIL: TestFiles (0.03s)
    --- FAIL: TestFiles/iota_forinit.gno (0.00s)
        files_test.go:135: unexpected output:
            0
            1

FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.044s
```
</details>

## gnovm/pkg/gnolang/nodes.go:2322 [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2322)
`func f(iota int) { println("hi") }`, `func f() (iota int)` and `func (iota T) M()` all run on master, because the name is bound but never [referenced](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/preprocess.go#L1298). Node startup [re-preprocesses](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/machine.go#L328-L364) every stored package at [`VMKeeper.Initialize`](https://github.com/gnolang/gno/blob/0558015ac/gno.land/pkg/sdk/vm/keeper.go#L168), with no per-package recover. A package already on chain that uses one of those forms would fail at boot rather than at its next call.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5981 -R gnolang/gno
cat > iota_param.gno <<'EOF'
package main

func f(iota int) { println("hi") }

func main() { f(3) }
EOF
echo "== at PR head:"; go run ./gnovm/cmd/gno run iota_param.gno
git checkout $(git merge-base origin/master HEAD) -- gnovm/pkg/gnolang/nodes.go
echo "== with the check removed:"; go run ./gnovm/cmd/gno run iota_param.gno
git checkout HEAD -- gnovm/pkg/gnolang/nodes.go
rm iota_param.gno
```

```
== at PR head:
panic: builtin identifiers cannot be shadowed: iota [recovered]
	panic: main/iota_param.gno:3:1-35: builtin identifiers cannot be shadowed: iota:
	--- preprocess stack ---
== with the check removed:
hi
```
</details>

## gnovm/pkg/gnolang/nodes.go:2323 [↗](../../../../../.worktrees/gno-review-5981/gnovm/pkg/gnolang/nodes.go#L2323)
Nit: `func f(len int) int { return len }` compiles here, and so does every [uverse name](https://github.com/gnolang/gno/blob/0558015ac/gnovm/pkg/gnolang/misc.go#L175-L178) but `iota` in that position. The message says builtin identifiers cannot be shadowed, so an author who hits it on a parameter reads a rule the compiler does not enforce.
