# PR #5069: feat(grc20reg): implement pagination

URL: https://github.com/gnolang/gno/pull/5069
Author: davd-gzl | Base: master | Files: 2 | +70 -36
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `c05d7b175` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5069 c05d7b175`

Verdict: APPROVE — pagination works, CI green, three approvals; the only loose end is the silent loss of `md.EscapeText` on token names/symbols, which is worth fixing as a follow-up but not blocking.

## Summary

Replaces the `// TODO: add pagination` in the home `Render` with `p/nt/avl/v0/pager` + `p/nt/mux/v0` routing. Same diff also fixes a wrong info link path (was `/r/demo/grc20reg:`, now `/r/demo/defi/grc20reg:` matching the actual realm location at [`examples/gno.land/r/demo/defi/grc20reg`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L72) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L72)). Token detail page is moved into its own handler. One pagination test added.

## Fix

Before: home iterates the full `registry` via `registry.Iterate("", "", ...)` building a single string regardless of size; `Render` uses a `switch` to distinguish home vs token; info link points to `/r/demo/grc20reg:...` (wrong realm path). After: a package-level `pagerInstance = pager.NewPager(rotree.Wrap(registry, nil), 10, false)` is shared across calls; `Render` instantiates a `mux.Router` per call mapping `""` → `renderHome` and `"*"` → `renderToken`; home reads `?page=N` / `?size=N` from `req.RawPath`, slices via `pager.MustGetPageByPath`, appends `**Page X of Y**` and `page.Picker`. The load-bearing constraint is that `rotree.Wrap` holds a pointer to the underlying `*avl.Tree` ([`rotree.gno:72-77`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/p/nt/avl/v0/rotree/rotree.gno#L72-L77) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/p/nt/avl/v0/rotree/rotree.gno#L72-L77)) so the singleton pager sees every later `Register` write without re-wrapping.

## Critical (must fix)

None.

## Warnings (should fix)

- **[lost XSS-style defense]** [`grc20reg.gno:72,86-87`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L72) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L72) — token `name` / `symbol` are no longer markdown-escaped on render.
  <details><summary>details</summary>

  Pre-PR the home line was `md.Bold(md.EscapeText(token.GetName()))` and the info link was `md.Link("info", infoLink)`; post-PR both are raw `ufmt.Sprintf("- **%s** - %s - [info](/r/demo/defi/grc20reg:%s)\n", token.GetName(), rlmLink, item.Key)`. The token detail page has the same regression on name (in `# %s`) and symbol (in `**%s**`). `grc20.NewToken` only rejects empty name/symbol ([`token.gno:16-22`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/p/demo/tokens/grc20/token.gno#L16-L22) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/p/demo/tokens/grc20/token.gno#L16-L22)) — any registering realm can pick e.g. `"](https://evil.com) [pwned"` or markdown-meaningful characters. `validateSlug` covers the slug only ([`grc20reg.gno:102-108`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L102-L108) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L102-L108)) and the existing `TestValidateSlugPanicsOnInjection` ([`grc20reg_test.gno:114-118`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L114-L118) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L114-L118)) shows the author knew the threat model. The `item.Key` substituted into the link URL is safe (rlmPath is trusted, slug is `validateSlug`'d), so this is narrow: escape `token.GetName()` and `token.GetSymbol()` only. Fix: wrap both with `md.EscapeText` (re-add the `gno.land/p/moul/md` import) — keeps the new bullet shape, restores the prior guarantee.
  </details>

## Nits

- [`grc20reg.gno:60`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L60) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L60) — `pagerInstance.MustGetPageByPath(req.RawPath)` panics on `url.Parse` failure ([`pager.gno:111-117`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L111-L117) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L111-L117)). The mux router already gives you a parsed `req.Query`; you could call `pagerInstance.GetPageWithSize(...)` directly and skip re-parsing the raw URL, removing a panic surface for malformed query strings (`?page=%ZZ` style).
- [`grc20reg.gno:53-56`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L53-L56) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L53-L56) — `mux.NewRouter()` + two `HandleFunc` calls run on every `Render` call. Cheap, but trivially hoistable to a package-level `var router = newRouter()` next to `pagerInstance` for consistency.
- [`grc20reg.gno:76`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L76) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L76) — mixed concatenation: `s += "\n"; s += "**Page " + strconv.Itoa(...) + " of " + strconv.Itoa(...) + "**\n\n"`. A single `ufmt.Sprintf("\n**Page %d of %d**\n\n", page.PageNumber, page.TotalPages)` matches the style of the surrounding lines and lets you drop the `strconv` import.

## Missing Tests

- **[edge-case coverage]** [`grc20reg_test.gno:56-74`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L56-L74) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L56-L74) — `TestPagination` only asserts the `"Page X of Y"` substring on pages 1 and 2.
  <details><summary>details</summary>

  Worth adding at least: (a) `?page=999` (out-of-range, asserts no panic and that the page object reports the right `TotalPages`); (b) `?size=3` (custom size, asserts 4 pages for 11 tokens); (c) item membership (page 1 contains `token00`, page 2 contains `token10`) — the current assertion would still pass if `page.Items` were silently empty. Each is a 2-3 line urequire.
  </details>

## Suggestions

- [`grc20reg.gno:17-20`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L17-L20) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L17-L20) — `pagerInstance` exports nothing but is in the same `var ( … )` block as the exported-by-`GetRegistry` `registry`. Consider keeping `registry` in its own line and `pagerInstance` private next to it — minor, but it makes the API surface obvious.
- [`grc20reg.gno:81-92`](https://github.com/gnolang/gno/blob/c05d7b175/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L81-L92) · [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L81-L92) — `renderToken` calls `MustGet` which panics on unknown key. With the new wildcard route, any malformed path past `:` triggers a Gno panic visible to the user. Old behaviour was the same, so not a regression — but a `Get`-then-404 (using `r.NotFoundHandler` or a hand-rolled message) is friendlier and idiomatic for a `mux`-based handler.

## Questions for Author

- Any reason to keep the per-call `mux.NewRouter()` rather than hoisting it to package scope next to `pagerInstance`? (Reads as a leftover from translating the old `switch`.)
- Was the `md.EscapeText` removal intentional (e.g. you wanted the raw name) or an accidental side-effect of replacing `md.Bold(md.EscapeText(...))` with `**%s**`?
