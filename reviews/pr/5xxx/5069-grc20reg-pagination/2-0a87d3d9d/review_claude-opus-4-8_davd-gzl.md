# PR #5069: feat(grc20reg): implement pagination

URL: https://github.com/gnolang/gno/pull/5069
Author: davd-gzl | Base: master | Files: 2 | +70 -37
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `0a87d3d9d` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5069 0a87d3d9d`

**TL;DR:** The grc20 token registry's home page printed every registered token in one block. This adds page-by-page browsing (10 per page, `?page=N`/`?size=N`), splits the per-token detail view into its own handler, and adopts the new threaded-realm calling convention.

**Verdict: APPROVE** — pagination works, all three approvals stand, the only carried concern is the still-missing markdown escape on the token name; symbol is now safe via the new `grc20.NewToken` validation, so the finding narrowed to name-only since the last round.

## Summary

Replaces the single-block home `Render` with `p/nt/avl/v0/pager` (page size 10) routed through `p/nt/mux/v0`, and moves the token detail view into `renderToken`. The same diff fixes the info link path (was `/r/demo/grc20reg:`, now `/r/demo/defi/grc20reg:`). Since round 1 the branch merged master, which changed `grc20.NewToken` to `(_ int, rlm realm, name, symbol, decimals)` with name/symbol validation and switched `Register` from the stack-walking `runtime.PreviousRealm()` to the threaded `cur.Previous()`. One pagination test added; existing tests adapted to the new signatures.

## Examples

| Render path | Result |
| --- | --- |
| `Render("")` | page 1, 10 tokens, `**Page 1 of N**`, picker |
| `Render("?page=2")` | page 2 |
| `Render("?size=3")` | 3 per page |
| `Render("gno.land/r/x.slug")` | token detail page |
| `Render("?\x7f")` | panics `invalid path` (see Nit) |

## Glossary

- crossing function — a realm function declared `func F(cur realm, ...)`, called as `F(cross(cur), ...)`; only across such a call does `cur.Previous()` shift to the immediate caller.
- threaded realm — passing the caller's `cur realm` value down instead of stack-walking via `runtime.PreviousRealm()`, so caller identity is unforgeable.

## Fix

Before: home built one string over `registry.Iterate`; `Render` used a `switch`; `Register` read the caller via `runtime.PreviousRealm().PkgPath()`. After: a package-level `pagerInstance = pager.NewPager(rotree.Wrap(registry, nil), 10, false)` is shared across calls; `Render` builds a `mux.Router` per call mapping `""` → [`renderHome`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L65) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L65) and `"*"` → [`renderToken`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L87) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L87); home reads `?page`/`?size` from `req.RawPath`. The load-bearing constraint is that `rotree.Wrap` holds a pointer to the underlying `*avl.Tree`, so the singleton pager sees every later `Register` write without re-wrapping. The delta since round 1 also rethreads the caller: [`grc20reg.gno:32`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32) now reads `cur.Previous().PkgPath()`, the correct form for the crossing `Register(cur realm, ...)` that all call sites invoke as `Register(cross(cur), ...)`.

## Critical (must fix)

None.

## Warnings (should fix)

- **[malicious token name breaks the page layout]** [`grc20reg.gno:78,92`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L78) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L78) — the token `name` reaches the rendered markdown unescaped.
  <details><summary>details</summary>

  Pre-PR both the home line and the detail title wrapped the name in `md.EscapeText` ([baseline `grc20reg.gno`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58)). The PR dropped the `gno.land/p/moul/md` import and prints the name raw: home as `- **%s**` ([line 78](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L78) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L78)), detail as `# %s` ([line 92](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L92) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L92)). The master merge added `validName`, which rejects only control chars and over-length ([`token.gno:60-74`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/p/demo/tokens/grc20/token.gno#L60-L74) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/p/demo/tokens/grc20/token.gno#L60-L74)); markdown-meaningful printables like `[ ] ( )` and backticks pass, so a registering realm can pick a name like `Evil](https://x.com) ` + "`code`" that injects a link or code span into the listing. Symbol is now safe: `validSymbol` restricts it to `[A-Za-z0-9_-]` ([`token.gno:76-86`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/p/demo/tokens/grc20/token.gno#L76-L86) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/p/demo/tokens/grc20/token.gno#L76-L86)), narrowing this from the round-1 name+symbol finding to name only. Verified behaviorally on 0a87d3d9d: a token named `Evil](https://x.com) ` + "`code`" renders that text verbatim into both `Render("")` and the detail page, and `md.EscapeText` escapes those bytes while leaving plain names like `TestToken` untouched ([repro](comment_claude-opus-4-8.md)). Fix: re-add the `gno.land/p/moul/md` import and wrap `token.GetName()` in `md.EscapeText` at both sites.
  </details>

## Nits

- [`grc20reg.gno:66`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L66) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L66) — `renderHome` calls `pagerInstance.MustGetPageByPath(req.RawPath)`, re-parsing the URL the mux router already parsed into `req.Query`; the re-parse panics `invalid path` on inputs `url.Parse` rejects. Confirmed behaviorally on 0a87d3d9d: `Render("?\x7f")` routes to home and panics, while the same input renders fine through the router's own tolerant `url.ParseQuery`. Calling `pagerInstance.GetPageWithSize(...)` from `req.Query` drops the second parse and the panic surface. (Percent-encoding like `?page=%ZZ` is tolerated and does not panic; only raw chars `url.Parse` rejects do.)
- [`grc20reg.gno:59-62`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L59-L62) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L59-L62) — `mux.NewRouter()` plus two `HandleFunc` calls run on every `Render`. Cheap, but trivially hoistable to a package-level `var router = ...` next to `pagerInstance` for consistency.
- [`grc20reg.gno:81-82`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L81-L82) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L81-L82) — mixed concatenation `s += "\n"; s += "**Page " + strconv.Itoa(...) + " of " + strconv.Itoa(...) + "**\n\n"`. A single `ufmt.Sprintf("\n**Page %d of %d**\n\n", page.PageNumber, page.TotalPages)` matches the surrounding lines and lets the `strconv` import go.

## Missing Tests

- **[edge-case coverage]** [`grc20reg_test.gno:55-73`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L55-L73) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L55-L73) — `TestPagination` asserts only the `"Page X of Y"` substring on pages 1 and 2.
  <details><summary>details</summary>

  Worth adding: (a) `?page=999` out-of-range, asserting no panic and the right `TotalPages`; (b) `?size=3` custom size, asserting 4 pages for 11 tokens; (c) item membership (page 1 contains `Token00`, page 2 contains `Token10`) — the current assertion still passes if `page.Items` were silently empty. Each is a 2-3 line urequire.
  </details>

## Suggestions

- [`grc20reg.gno:16-19`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L16-L19) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L16-L19) — `pagerInstance` exports nothing but shares the `var ( … )` block with `registry` (exposed by `GetRegistry`). Splitting them makes the API surface obvious.
- [`grc20reg.gno:87-98`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L87-L98) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L87-L98) — `renderToken` calls `MustGet`, which panics on an unknown key. Any malformed path past `:` triggers a Gno panic the user sees. Old behavior was the same, so not a regression, but a `Get`-then-404 (via `r.NotFoundHandler` or a hand-rolled message) is friendlier for a `mux`-based handler.

## Open questions

- The new doc comment at [`grc20reg.gno:21-26`](https://github.com/gnolang/gno/blob/0a87d3d9d/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L21-L26) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L21-L26) accurately describes the new `grc20.NewToken` signature and its `IsCurrent` binding; verified against `token.gno`. No action. Not posted — purely confirmatory.
- Was the `md.EscapeText` removal intentional, or a side effect of replacing `md.Bold(md.EscapeText(...))` with `**%s**`? Folded into the Warning; not posted separately.
