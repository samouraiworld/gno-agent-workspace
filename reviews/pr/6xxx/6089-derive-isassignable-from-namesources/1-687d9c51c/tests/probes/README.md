# Behaviour probes, merge base against PR head

Nine single-file programs, each run with `gno run` from source on both sides:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6089 -R gnolang/gno
for t in t*.gno; do GNOROOT=$PWD go run ./gnovm/cmd/gno run "$t" 2>&1 | head -2; done
git checkout $(git merge-base origin/master HEAD)
for t in t*.gno; do GNOROOT=$PWD go run ./gnovm/cmd/gno run "$t" 2>&1 | head -2; done
```

| File | Program | Output, identical on both sides |
| --- | --- | --- |
| `t1.gno` | `func f(){}` then `f = nil` | `not assignable` |
| `t2.gno` | package-level `var f = 1` beside an unrelated `func f2()` | `1` |
| `t3.gno` | local `f := 1; f = 2` shadowing a package-level `func f()` | `2` |
| `t4.gno` | `println = nil` | `cannot assign to uverse println` |
| `t5.gno` | `const c = 1` then `c = 2` | `cannot assign to const c` |
| `t6.gno` | `type T int` then `T = 1` | `cannot assign to const T` |
| `t7.gno` | `func init(){}` beside `main` | `ok` |
| `t8.gno` | `var f = 1` declared before `func f()` | `f redeclared in this block` |
| `t9.gno` | `func f()` declared before `var f = 1` | `f redeclared in this block` |

`t8` and `t9` are the one shape where the two representations could have diverged: `Reserve` is a
no-op on a name the block already holds, while the deleted `UnassignableNames` append was not. The
go/types pass refuses both programs before the preprocessor sees them.
