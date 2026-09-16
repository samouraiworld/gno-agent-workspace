# from a local clone of gnolang/gno, on the PR branch (head 3aa06ef757b87d731a3e554e96c7540a209fd9a3):
#   cd examples && gno test ./gno.land/r/nt/grc20routes/v0

## Claim

`validate` (called at Register, grc20routes.gno:309, before `sortEntries` at
line 310) detects a duplicate `Op` with a nested rescan of every earlier
entry (5 lines, O(n^2) with n <= maxEntries=16). Swapping the two calls — sort
first, then validate — turns the duplicate check into a 1-line adjacent-pair
comparison.

## Read that settles it

grc20routes.gno:308-310 (Register, the two calls in order):
```go
    validate(r)
    sortEntries(r.Funcs)
    declared.Set(key, &r)
```

grc20routes.gno:543-554 (validate's duplicate scan, nested):
```go
    for i, e := range r.Funcs {
        validateOpName(e.Op)
        if e.Func == "" {
            panic("grc20routes: empty entry point for " + e.Op)
        }
        validateFuncName(e.Op, e.Func)
        for _, prev := range r.Funcs[:i] {
            if prev.Op == e.Op {
                panic("grc20routes: duplicate operation: " + e.Op)
            }
        }
    }
```
Confirms the claim: `validate` runs on the caller's original order, and its
inner `for _, prev := range r.Funcs[:i]` is the nested rescan, 5 lines
(549-553) doing what an adjacent-pair check after sorting would do in 1.

## Rewrite (swap the calls, replace the nested scan)

```go
    sortEntries(r.Funcs)
    validate(r)
    declared.Set(key, &r)
```
and in `validate`, replacing lines 549-553:
```go
    for i, e := range r.Funcs {
        validateOpName(e.Op)
        if e.Func == "" {
            panic("grc20routes: empty entry point for " + e.Op)
        }
        validateFuncName(e.Op, e.Func)
        if i > 0 && r.Funcs[i-1].Op == e.Op {
            panic("grc20routes: duplicate operation: " + e.Op)
        }
    }
```
Line count: 5 lines (549-553) to 1, same panic message. The comment at line
637 ("maxEntries is 16") that justifies the quadratic scan's cost stops
applying to this loop.

Two ordering consequences the rewrite introduces, both worth naming rather
than silently accepting:
- With sort-first, a duplicate pair's reported `Op` is whichever member of
  the pair sorts later, not whichever the caller listed second — a message
  text change on ties, not a behavior change (`Op` is identical on both, so
  the message string is unaffected in practice).
- Sorting `r.Funcs` in place before `validate` can reject the route mutates
  the caller's slice even on a route that Register goes on to panic on. This
  interacts with the separate aliasing finding on Register's caller-slice
  handling (candidate at line 311 in the parent's notes) — sort-then-validate
  makes that aliasing visible one step earlier in the call than
  validate-then-sort does.

## State

CONFIRMED by reading grc20routes.gno:308-310 and 543-554 above: `validate`
runs before `sortEntries` today, and its duplicate check is the nested scan
described. Band: Nit — correctness is identical (same panics, same
messages), and the population that hits the O(n^2) cost is bounded to 16
entries per the code's own comment, so no case pays a real amortized cost.
