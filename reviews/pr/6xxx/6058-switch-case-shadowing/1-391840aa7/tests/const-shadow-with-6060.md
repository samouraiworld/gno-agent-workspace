# The `const` divergence against [#6060](https://github.com/gnolang/gno/pull/6060)

[#6060](https://github.com/gnolang/gno/pull/6060) replaces the name-keyed const
checks with `GetIsConstAt` on the NameExpr's already-resolved path. Its body
names this branch: "found while reviewing #6058; they compose with that PR but
do not depend on it."

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6058 -R gnolang/gno
git merge $(gh api repos/gnolang/gno/pulls/6060 --jq '.head.sha')
cp <this-directory>/switch58.gno gnovm/tests/files/
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/(switch5[238]|if9).gno$' -count=1 -v
rm gnovm/tests/files/switch58.gno
```

The merge is clean. `switch58.gno` asserts `go run`'s output for a `const`
shadow used before its declaration:

```
--- PASS: TestFiles/if9.gno (0.00s)
--- PASS: TestFiles/switch53.gno (0.01s)
--- PASS: TestFiles/switch52.gno (0.00s)
--- PASS: TestFiles/switch58.gno (0.00s)
```

So 6058 alone prints the copy slot's zero value, and 6058 with 6060 prints the
outer value. Merge order is what decides whether master carries the divergence.

## The fix shape that does not work

Refusing the append in `Reserve` when `isConst` is set restores the merge base's
rejection and breaks the branch's own `switch52.gno`, which asserts that a
`const` shadow with no earlier use works:

```go
		if isConst {
			return
		}
```

```
files_test.go:135: unexpected panic: main/switch58.gno:18:9-16: StaticBlock.Define2(v) cannot change const status
files_test.go:135: unexpected panic: main/switch52.gno:22:9-16: StaticBlock.Define2(v) cannot change const status
--- FAIL: TestFiles/switch52.gno (0.04s)
--- FAIL: TestFiles/switch58.gno (0.04s)
```

Position-sensitive const resolution is the only shape that satisfies both, which
is what 6060 implements.
