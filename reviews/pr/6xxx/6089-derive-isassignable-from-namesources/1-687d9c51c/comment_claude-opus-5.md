# Review: [#6089](https://github.com/gnolang/gno/pull/6089)
Posted: https://github.com/gnolang/gno/pull/6089#pullrequestreview-5041171850
Event: COMMENT

## Body
[AI review]

Status: APPROVE

`UnassignableNames` and `NameSources[i].Type == NSFuncDecl` agree on all 56860035 names in the 808021 blocks of the `TestFiles` corpus, and `len(NameSources)` equals `len(Names)` in every one of them.

<details><summary>harness</summary>

Run at the merge base, where both fields still exist. The patch instruments `IsAssignable` to compute both answers, `Preprocess` to compare the two representations over every name in every block it just built, and `Define2` to catch a redefinition overwriting an `NSFuncDecl` entry.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6089 -R gnolang/gno
git checkout $(git merge-base origin/master HEAD)
curl -fsSL -o /tmp/equivalence-harness.patch \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6089-derive-isassignable-from-namesources/1-687d9c51c/tests/equivalence-harness.patch
git apply /tmp/equivalence-harness.patch
go test ./gnovm/pkg/gnolang/ -run TestFiles -timeout 40m -count=1
cat /tmp/zz6089.txt
git checkout -- gnovm/pkg/gnolang && rm gnovm/pkg/gnolang/zz_equiv_report_test.go
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	193.064s
ZZ-REPORT calls=24107 blocks=808021 names=56860035 diffs=0
```
</details>

## gnovm/pkg/gnolang/preprocess.go:2731 [gh](https://github.com/gnolang/gno/blob/687d9c51c/gnovm/pkg/gnolang/preprocess.go#L2731) · [↗](../../../../../.worktrees/gno-review-6089/gnovm/pkg/gnolang/preprocess.go#L2731) [posted](https://github.com/gnolang/gno/pull/6089#discussion_r3872033226)
Missing test: nothing under `gnovm/tests/files/` assigns to a package-level func name, so a change of behaviour in the predicate this line calls would go green.

<details><summary>test cases</summary>

Both pass at this head and on the merge base.

`gnovm/tests/files/assign_func_decl_err.gno`, the refusal itself:

```go
package main

func f() {}

func main() {
	f = nil
}

// Error:
// main/assign_func_decl_err.gno:6:2-9: not assignable

// TypeCheckError:
// main/assign_func_decl_err.gno:6:2: cannot assign to f (neither addressable nor a map index expression)
```

`gnovm/tests/files/assign_func_decl.gno`, the two names the refusal must not swallow, a package-level `var` holding a func value and a local shadowing a package-level func:

```go
package main

func f() { println("f") }

var g = func() { println("g") }

func main() {
	g = f
	g()
	f := 1
	f = 2
	println(f)
}

// Output:
// f
// 2
```

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6089 -R gnolang/gno
go test -run 'TestFiles/assign_func_decl' -count=1 ./gnovm/pkg/gnolang/
```
</details>
