# PR #5761: feat(gnoweb): add omnibar search for realms, packages and users

URL: https://github.com/gnolang/gno/pull/5761
Author: alexiscolin | Base: master | Files: 13 | +766 -21
Reviewed by: davd-gzl | Model: claude-opus-4 | Commit: a54f574a9 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5761 a54f574a9`

**Verdict: APPROVE** — clean, well-tested feature; no correctness or security blockers. Two non-blocking items: the deep-path relevance cutoff silently drops legitimate matches, and a first-load redraw race can render stale page-snippets. Both Warnings at most; neither blocks merge.

## Summary
Turns the gnoweb search bar into an omnibar. A new `GET /search.json` endpoint returns the realm (`/r/`) and package (`/p/`) path lists fetched live from the chain via `vm/qpaths`; the browser fetches that list once, then filters/ranks/renders client-side per keystroke, plus an in-page text scan of the current realm view. gnoweb stays stateless: no indexer, no per-keystroke node query. The security-sensitive surface — rendering chain-sourced paths and the user's query into the dropdown — is handled correctly: every dynamic value goes through `textContent` / `createTextNode` / `setAttribute`, never `innerHTML`, so there is no HTML-injection path. The node-side `qpaths` scan is bounded by a 1000-path default limit and a gas meter, so there is no unbounded scan. The PR's body documents its own scaling ceiling (1000 paths) honestly.

## Glossary
- `qpaths` — node ABCI query (`vm/qpaths`) returning package/realm paths under a prefix.
- `RealmDirectory` — new interface (the "seam") behind which the path source can later swap from live RPC to a search index.
- `relevance` — client-side scoring function ranking a path against the typed query.
- `firstSegment` — extracts the namespace owner from a path (`/r/demo/boards` → `demo`) to derive `/u/` user links.

## Fix
Before: the search bar was a plain path navigator (Enter submits, `resolveTarget` strips a `gno.land` host). After: `controller-searchbar.ts` grows a fetch-once cache, a relevance ranker, grouped DOM rendering, keyboard nav, and an in-page TreeWalker scan; the backend adds [`realm_directory.go`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/realm_directory.go) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/realm_directory.go) (semaphore-bounded parallel `/r` + `/p` fetch) and [`handler_search.go`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/handler_search.go) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/handler_search.go) (GET-only, `Cache-Control: public, max-age=30`). The load-bearing safety gate is that all rendered text uses DOM text APIs and the only `href` assignment is guarded by `path.startsWith("/")`, and the CSP `connect-src` gains `'self'` so the same-origin fetch is allowed.

## Security review (what I checked, all clean)
- HTML injection: result rows render the path and query via [`fillHighlighted`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L293-L304) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L293-L304) using `createTextNode` and `mark.textContent`; the kind tag uses `tag.textContent` / `tag.dataset.type`; ARIA labels use `setAttribute`. No `innerHTML` anywhere. A package author cannot smuggle markup into the dropdown.
- `javascript:` href: the only `href` set is `item.href = path` at [`controller-searchbar.ts:233`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L230-L233) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L230-L233), guarded by `if (!path.startsWith("/")) continue`. A leading-slash path can never form a scheme URL.
- Template: `{{ .RealmPath }}` in header.html renders through `html/template` (`components/template.go` imports `html/template`), so the `value="..."` attribute is auto-escaped; the new ARIA attributes are static literals.
- Backend scan bound: [`queryPaths`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/sdk/vm/handler.go#L159-L191) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/sdk/vm/handler.go#L159-L191) defaults `limit` to 1000 (cap 10000) and `QueryPaths` runs under a `maxGasQuery` gas meter — the prefix scan is doubly bounded. No unbounded scan.
- DoS amplification: `/search.json` triggers exactly two node queries per request, bounded by a 16-slot semaphore ([`handler_search.go:11`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/handler_search.go#L11) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/handler_search.go#L11)), and the response is a single cacheable URL. Reasonable.

## Critical (must fix)
None.

## Warnings (should fix)
- **[deep paths silently unfindable]** [`controller-searchbar.ts:169-181`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L169-L181) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L169-L181) — `relevance` returns `score - segs.length`, and `rank` keeps only `score > 0`, so a substring-only match (base 20) on a path ≥ 20 segments deep scores ≤ 0 and is dropped despite matching.
  <details><summary>details</summary>

  The score tiers are 100 / 80 / 60 / 40 / 20, then `- segs.length` as a shallow-path tie-breaker. For the weakest tier (substring match that isn't a prefix of any segment, base 20), a path with 20+ segments yields `20 - 20 = 0`, and `rank` filters `score > 0` at [`controller-searchbar.ts:157`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L155-L157) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L155-L157), so the path becomes unfindable. Real gno.land paths rarely hit 20 segments today, so impact is low — but the depth penalty conflating "rank lower" with "exclude entirely" is a latent correctness trap. The same subtraction also lets a deeper exact-name match (base 100) outrank... fine, but the boundary at the bottom tier is the bug. Fix: clamp the tie-breaker so it never crosses zero, e.g. `Math.max(1, score - segs.length)` or use a fractional penalty (`score - segs.length * 0.01`) that orders without eliminating.
  </details>

- **[stale snippets on first keystrokes]** [`controller-searchbar.ts:93-98`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L93-L98) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L93-L98) — `run` is async and `await`s the first fetch; two `run(q)` calls racing across that await can let an older call's `draw` win, rendering page-snippets for a stale query.
  <details><summary>details</summary>

  `run` does `scanPage(q)` → `await ensureLoaded()` → `this.pageMatches = pageMatches; draw(...)`. After the list is loaded once, `ensureLoaded` resolves synchronously and the debounce (120ms) serializes calls, so steady-state is fine. The window is only the very first load: if the user types `ab` then quickly `abc` while the single fetch is still in flight, both `run`s are parked at the await; whichever resolves last sets `this.pageMatches` and calls `draw`. There is no per-call sequence guard (no generation counter / no "is this still the latest query" check), so the displayed snippets can briefly belong to the wrong query. Cosmetic, self-correcting on the next keystroke. Fix: capture a monotonically increasing request id before the await and bail in `draw` if a newer `run` has started, or move `scanPage` after the await so the snippet always reflects the query being drawn.
  </details>

## Nits
- [`realm_directory.go:24-28`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/realm_directory.go#L24-L28) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/realm_directory.go#L24-L28) — `searchPathLimit = 1000` is passed to `ListPaths` but [`rpcClient.ListPaths`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/client.go#L136-L152) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/client.go#L136-L152) never forwards it to the `vm/qpaths` ABCI query (it sends only the prefix), so the constant is inert and the node's own 1000 default governs. The doc comment correctly says so; harmless, but a reader may assume it has effect. When limit-forwarding lands as the documented follow-up, wire it through `ListPaths` so this stops being a no-op.
- [`controller-searchbar.ts:135`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L133-L141) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L133-L141) — users are derived only from the already-ranked `apps`/`packages` (capped at 5 each pre-slice? no — `rank` returns all, slice happens later, so user derivation sees the full ranked list, good). But a user whose namespace matches the query while none of their package *names* do will still surface (namespace match scores 40), so this is fine; noting only that there is no independent user index, by design.

## Missing Tests
- **[no test for relevance scoring]** [`controller-searchbar.ts:153-181`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L153-L181) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L153-L181) — `rank` / `relevance` / `firstSegment` / `resolveTarget` / `typeOf` are pure static functions but have no unit tests; the deep-path-drop bug above would have been caught by one.
  <details><summary>details</summary>

  The Go side (`handler_search_test.go`) is well covered: OK, method-not-allowed, upstream-error, and empty-entry filtering. The client-side ranking — the part with the actual edge-case logic — has none. There is no JS test harness exercised in this PR. A few table tests over `relevance` (exact / prefix / segment-prefix / substring / deep path) and `resolveTarget` (`gno.land/r/x`, `https://gno.land/r/x`, `https://evil.com`, relative) would lock the behavior. Low priority given the feature is additive and degrades gracefully, but the scoring is exactly the kind of code that drifts.
  </details>

## Suggestions
- [`handler_search.go:33-36`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/handler_search.go#L33-L36) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/handler_search.go#L33-L36) — `json.NewEncoder(...).Encode(...)` error is discarded with `_`. Header + status are already committed by then, so there's nothing to recover, but a `logger.Error` on encode failure would aid debugging a truncated response. Optional.
- [`realm_directory.go:47-61`](https://github.com/gnolang/gno/blob/a54f574a9/gno.land/pkg/gnoweb/realm_directory.go#L47-L61) · [↗](../../../../../.worktrees/gno-review-5761/gno.land/pkg/gnoweb/realm_directory.go#L47-L61) — `Paths` returns the first of `rErr`/`pErr` but lets both goroutines finish (correct). Consider `errors.Join(rErr, pErr)` so a caller debugging a flaky node sees both failures rather than just the realm one. Cosmetic.

## Questions for Author
- The deep-path relevance cutoff at the bottom tier (base 20, `segs.length ≥ 20` → excluded): intentional, or an unintended consequence of reusing the depth penalty as both tie-breaker and filter?
- Server-side: `/search.json` relies entirely on client/edge `Cache-Control: max-age=30`; with no edge cache in front, every cold browser triggers 2 node queries. Is a short server-side TTL cache (single-flight, ~30s) in scope here or deferred to the `RealmDirectory` swap?
