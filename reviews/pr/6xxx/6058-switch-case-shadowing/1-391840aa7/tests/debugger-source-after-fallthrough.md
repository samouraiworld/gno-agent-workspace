# `print` after a fallthrough, across three commits

The branch's `ExpandWith` hunk moves `b.Source = source` above the equal-size early
return. This records what the debugger's `print` does without it, at the merge base
and at current master.

The fixture is the branch's own `gnovm/tests/integ/debugger/fallthrough.gno` and the
branch's own `TestDebug` case, both applied unchanged to each tree:

```bash
# from a local clone of gnolang/gno:
git checkout <commit>
cat > gnovm/tests/integ/debugger/fallthrough.gno <<'GNO'
// Sample target for the debugger fallthrough test.
package main

func main() {
	switch 1 {
	case 1:
		y := 5
		_ = y
		fallthrough
	case 2:
		println("two")
	}
}
GNO
# append to TestDebug in gnovm/pkg/gnolang/debugger_test.go:
#   runDebugTest(t, "../../tests/integ/debugger/fallthrough.gno", []dtest{
#     {in: "s\nb 11\nc\np y\n", out: "Command failed: name y not declared"},
#   })
go test ./gnovm/pkg/gnolang/ -run 'TestDebug$' -count=1
```

Toolchain: go1.25.9, the version `go.mod` pins.

| Commit | What it is | `p y` after the fallthrough |
|---|---|---|
| `754780601` | merge base, before [#6056](https://github.com/gnolang/gno/pull/6056) | `err: unexpected block size shrinkage: 1 vs 0`, the program never runs |
| `0cf310707` | current master, [#6056](https://github.com/gnolang/gno/pull/6056) merged | `Command failed: runtime error: index out of range [0] with length 0` |
| `391840aa7` | this branch | `Command failed: name y not declared` |

Observed at `754780601`:

```
--- FAIL: TestDebug/#62 (0.00s)
    debugger_test.go:81: in: s
        b 11
        c
        p y
         out:  err: unexpected block size shrinkage: 1 vs 0
```

Observed at `0cf310707`:

```
            dbg> Breakpoint 0 at main main/../../tests/integ/debugger/fallthrough.gno:11:1
            =>   11: 		println("two")
            dbg> Command failed: runtime error: index out of range [0] with length 0
```

`0cf310707` is the squash of #6056, whose truncation lowers `len(b.Values)` to the
switch's own name count. `ExpandWith` then finds the next clause the same size and
returns before assigning `b.Source`, so the block keeps the fallen-from clause as its
source and the debugger resolves `y` against a slot the truncation removed.
