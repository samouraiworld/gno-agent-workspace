# Review: PR [#5920](https://github.com/gnolang/gno/pull/5920)
Event: APPROVE

## Body
Looks good. Verified on 3732be8d3: reverting the blank-`_` skip drops the second blank func's type error from the go/types deploy gate, and restoring it reports both.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5920 -R gnolang/gno
# drop the blank-func skip the PR added, back to the buggy version
sed -i -e '/fd\.Name\.Name == "_" {/d' \
       -e 's/fd\.Name\.Name == "init" ||/fd.Name.Name == "init" {/' \
       gnovm/pkg/gnolang/gotypecheck.go
go test -run 'TestTypeCheckMemPackage/BlankFuncsAllChecked' ./gnovm/pkg/gnolang/
git checkout gnovm/pkg/gnolang/gotypecheck.go
```

```
--- FAIL: TestTypeCheckMemPackage/BlankFuncsAllChecked (0.00s)
    gotypecheck_test.go:408: expected 2 errors, got 1
FAIL
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5920-typecheck-blank-func-decls/1-3732be8d3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/pkg/gnolang/gotypecheck.go:544 [↗](../../../../../.worktrees/gno-review-5920/gnovm/pkg/gnolang/gotypecheck.go#L544)
Nit: the comment says the loop ignores methods and init functions, but it now also skips blank funcs.
