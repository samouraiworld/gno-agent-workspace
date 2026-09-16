# from a local clone of gnolang/gno, on the PR branch (head 3aa06ef757b87d731a3e554e96c7540a209fd9a3):
#   cd examples && gno test ./gno.land/r/nt/grc20routes/v0

## Claim

`validateFuncName` (grc20routes.gno:615) is documented "requires an exported
identifier" but its body is a bare `for i, c := range name`, which iterates
zero times on `""` and returns without panicking. The emptiness check instead
lives at the one call site, `validate` (grc20routes.gno:545-548), while the
sibling `validateOpName` (line 600-603) checks `op == ""` itself.

## Read that settles it

grc20routes.gno:543-548 (validate):
```go
for i, e := range r.Funcs {
    validateOpName(e.Op)
    if e.Func == "" {
        panic("grc20routes: empty entry point for " + e.Op)
    }
    validateFuncName(e.Op, e.Func)
```

grc20routes.gno:614-630 (validateFuncName, full body):
```go
// validateFuncName requires an exported identifier.
func validateFuncName(field, name string) {
    if len(name) > maxIdentLen {
        panic("grc20routes: " + field + " name too long")
    }
    for i, c := range name {
        if i == 0 {
            if c < 'A' || c > 'Z' {
                panic("grc20routes: " + field + " must be an exported function name")
            }
            continue
        }
        if !isAlphanumeric(c) && c != '_' && c != '-' {
            panic("grc20routes: invalid character in " + field + " name: " + string(c))
        }
    }
}
```
No branch tests `name == ""`; called standalone with `""` it returns nil.
Confirms the claim: the helper's doc comment is false of the helper alone,
true only in combination with its one caller's separate guard.

## Rewrite (fold the check in, drop it from the call site)

```go
// validateFuncName requires an exported identifier.
func validateFuncName(field, name string) {
    if name == "" {
        panic("grc20routes: empty entry point for " + field)
    }
    if len(name) > maxIdentLen {
        panic("grc20routes: " + field + " name too long")
    }
    for i, c := range name {
        if i == 0 {
            if c < 'A' || c > 'Z' {
                panic("grc20routes: " + field + " must be an exported function name")
            }
            continue
        }
        if !isAlphanumeric(c) && c != '_' && c != '-' {
            panic("grc20routes: invalid character in " + field + " name: " + string(c))
        }
    }
}
```
and in `validate`:
```go
for i, e := range r.Funcs {
    validateOpName(e.Op)
    validateFuncName(e.Op, e.Func)
```
Line count: `validate` loses 3 lines (545-548 minus the `validateFuncName` call
that stays), `validateFuncName` gains 3 (the new `if`). Net zero lines, but
the emptiness check moves to live beside its sibling in the function that
owns the contract, and `validateFuncName` becomes safe to call standalone —
what a future second declare path or op-set extension would otherwise miss.

## State

CONFIRMED by reading grc20routes.gno:543-630 above: the loop body has no
`len(name) == 0` guard, and the guard the message text implies exists only in
the caller. Band: Nit — `validate` already calls the check before
`validateFuncName` on every declared route today, so no route in the current
call graph reaches the empty case unchecked; the defect is that the helper's
own contract is false standalone, for whichever caller reads its doc comment
next.
