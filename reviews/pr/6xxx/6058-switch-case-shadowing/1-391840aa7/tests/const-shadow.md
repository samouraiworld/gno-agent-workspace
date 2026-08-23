# A `const` shadow in a case body, across three trees

`switch58.gno` beside this file, run at each commit. Toolchain go1.25.9, the
version `go.mod` pins.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6058 -R gnolang/gno
cp <this-directory>/switch58.gno gnovm/tests/files/
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/switch58.gno$' -count=1
rm gnovm/tests/files/switch58.gno
```

| Tree | First `println(v)` |
|---|---|
| `go run`, go1.25.9 | `1` |
| `391840aa7`, this branch | `0` |
| `754780601`, merge base | rejected: `StaticBlock.Define2(v) cannot change const status` |

Observed at `391840aa7`:

```
--- FAIL: TestFiles/zzconst1.gno (0.00s)
    files_test.go:135: Output diff:
        --- Expected
        +++ Actual
        @@ -1,2 +1,2 @@
        -1
        +0
         c
```

Observed at `754780601`:

```
files_test.go:135: unexpected panic: main/zzconst1.gno:7:9-16: StaticBlock.Define2(v) cannot change const status
```

The same program with an ordinary nested block in place of the switch prints
`0` at the merge base as well:

```go
v := 1
{
	println(v)
	const v = "c"
	println(v)
}
```

```
--- FAIL: TestFiles/zzconst2.gno (0.00s)
    files_test.go:135: Output diff:
        @@ -1,2 +1,2 @@
        -1
        +0
         c
```

So `getLocalIsConst` reading `slices.Contains(sb.Consts, n)` predates the
branch. What the branch changes is which programs reach it: the merge base
rejected a `const` shadow in a case body outright, and the branch accepts it
and prints the copy slot's zero value.
