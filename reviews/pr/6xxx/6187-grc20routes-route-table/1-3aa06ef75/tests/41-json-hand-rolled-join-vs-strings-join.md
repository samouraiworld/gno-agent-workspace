# from a local clone of gnolang/gno, on the PR branch (head 3aa06ef757b87d731a3e554e96c7540a209fd9a3):
#   cd examples && gno test ./gno.land/r/nt/grc20routes/v0

## Claim

`JSON` (grc20routes.gno:466-490) builds its `prefix` and `funcs` fragments
with two near-identical index-guarded concatenation loops (20 lines total)
where `strings.Join` over two prebuilt slices would take 10, and `strings` is
already imported and already used the same way in `Register`.

## Read that settles it

grc20routes.gno:470-484 (the two hand-rolled loops):
```go
    prefix := ""
    for i, p := range r.Prefix {
        if i > 0 {
            prefix += ","
        }
        prefix += `"` + p + `"`
    }

    funcs := ""
    for i, e := range r.Funcs {
        if i > 0 {
            funcs += ","
        }
        funcs += `"` + e.Op + `":"` + e.Func + `"`
    }
```
That is lines 470-484, 15 lines of loop body (plus the surrounding blank
lines and the `Sprintf` call bring the finder's cited range to 470-489, 20
lines including whitespace).

grc20routes.gno:325-326 (Register, same job on the same field, already via
`strings.Join`):
```go
        "prefix", strings.Join(r.Prefix, ","),
```
Confirms `strings` is imported and `strings.Join` is the package's own
established idiom for this exact concatenation.

## Rewrite

```go
    prefixParts := make([]string, len(r.Prefix))
    for i, p := range r.Prefix {
        prefixParts[i] = `"` + p + `"`
    }

    funcParts := make([]string, len(r.Funcs))
    for i, e := range r.Funcs {
        funcParts[i] = `"` + e.Op + `":"` + e.Func + `"`
    }

    return ufmt.Sprintf(
        `{"token_key":"%s","pkg_path":"%s","prefix":[%s],"source":"%s","funcs":{%s}}`,
        tokenKey, pkgPath, strings.Join(prefixParts, ","), source, strings.Join(funcParts, ","),
    )
```
Line count: 20 lines (470-489) to 10. Output is byte-identical: the `i > 0`
guard's only effect was to place a comma between elements and none before
the first or after the last, which is exactly what `strings.Join(x, ",")`
does over the same elements in the same order.

## State

CONFIRMED by reading grc20routes.gno:470-489 and 325-326 above: the two loops
are structurally identical index-guarded concatenations, and `strings.Join`
is already the package's idiom for the same data one function away. Band:
Nit — `TestJSONShape` (grc20routes_test.gno:299, per the diff's own test
file) and the integration txtar
`gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar` both assert
the exact JSON string, which the rewrite reproduces byte-for-byte, so no
behavior changes.
