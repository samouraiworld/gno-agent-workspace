# from a local clone of gnolang/gno, on the PR branch (head 3aa06ef757b87d731a3e554e96c7540a209fd9a3):
#   cd examples && gno test ./gno.land/r/nt/grc20routes/v0

## Claim

`Render` (grc20routes.gno:652-697) keeps a local `n` counter across each of
its two `avl.Tree.Iterate` calls purely to decide whether to print
"_None yet._" afterward. `avl.Tree.Size()` answers the same question without
the counter.

## Read that settles it

grc20routes.gno:663-672 (provenKeyed block):
```go
    s += "## Realms proven to route by symbol\n\n"
    n := 0
    provenKeyed.Iterate("", "", func(pkgPath string, _ any) bool {
        n++
        s += "- " + md.InlineCode(pkgPath) + " — `Approve(cur, SYMBOL, spender, amount)`\n"
        return false
    })
    if n == 0 {
        s += "_None yet._\n"
    }
```

grc20routes.gno:674-695 (declared block, same shape):
```go
    s += "\n## Routes declared by their own realm\n\n"
    n = 0
    declared.Iterate("", "", func(key string, v any) bool {
        n++
        r := v.(*Route)
        ...
        return false
    })
    if n == 0 {
        s += "_None yet._\n"
    }
```

examples/gno.land/p/nt/avl/v0/tree.gno:39:
```go
func (tree *Tree) Size() int {
```
`Size()` exists on the same `avl.Tree` type `provenKeyed` and `declared` are
declared as. Confirms the claim: `n` is incremented on every entry and read
exactly once, as a `== 0` boolean, which `provenKeyed.Size() == 0` and
`declared.Size() == 0` answer without the closure-captured counter.

## Rewrite

```go
    s += "## Realms proven to route by symbol\n\n"
    if provenKeyed.Size() == 0 {
        s += "_None yet._\n"
    }
    provenKeyed.Iterate("", "", func(pkgPath string, _ any) bool {
        s += "- " + md.InlineCode(pkgPath) + " — `Approve(cur, SYMBOL, spender, amount)`\n"
        return false
    })

    s += "\n## Routes declared by their own realm\n\n"
    if declared.Size() == 0 {
        s += "_None yet._\n"
    }
    declared.Iterate("", "", func(key string, v any) bool {
        r := v.(*Route)
        rlmPath, symbol := fqname.Parse(key)
        s += "- " + fqname.RenderLink(rlmPath, symbol) + " — "
        for i, e := range r.Funcs {
            if i > 0 {
                s += ", "
            }
            s += md.InlineCode(e.Op + "=" + e.Func)
        }
        if len(r.Prefix) > 0 {
            s += " (prefix " + md.InlineCode(strings.Join(r.Prefix, ", ")) + ")"
        }
        s += "\n"
        return false
    })
    return s
```
Line count: the two `n := 0` / `n++` / `if n == 0` triples (6 lines total:
665, 666, 670-672, and 675, 677 restated, 693-695) drop to two `if ...Size()
== 0` lines placed before each `Iterate`, net 4 lines removed across the two
blocks, matching the finder's count. Output is byte-identical: moving the
"_None yet._" emission before the `Iterate` changes nothing since either
branch (empty tree, so `Iterate` calls the callback zero times; non-empty
tree, so `Size() != 0` and nothing is added) is mutually exclusive with the
loop actually appending rows.

## State

CONFIRMED by reading grc20routes.gno:663-695 and
examples/gno.land/p/nt/avl/v0/tree.gno:39 above: `n` is used only as a
zero/nonzero flag, and `avl.Tree.Size()` is available on the exact type
`provenKeyed` and `declared` are. Band: Nit — no behavior change, no test
exists for `Render` today to run the rewrite through (the parent's separate
missing-render-test candidate covers that gap).
