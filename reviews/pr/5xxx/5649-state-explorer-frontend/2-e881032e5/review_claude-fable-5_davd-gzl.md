# PR #5649: refactor: state explorer frontend

URL: https://github.com/gnolang/gno/pull/5649
Author: alexiscolin | Base: master | Files: 104 | +16219 -250
Reviewed by: davd-gzl | Model: claude-fable-5 | Commit: `e881032e5` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5649 e881032e5`

**TL;DR:** gnoweb gets a "State" tab: every realm's stored variables and objects become browsable web pages, rendered server-side with shareable URLs, lazy expansion of nested objects, live search, and a raw JSON API for external tools. Same architecture as before (chain query endpoints feed a Go decoder), but the rendering moved from browser TypeScript into Go templates.

**Verdict: REQUEST CHANGES** — same commit as the [opus round](claude-opus-4-8_davd-gzl.md), independent fable-5 cross-check. Two new must-fix items: pagination links silently exit an active search filter (behavioral repro below), and CI is now red because the committed frontend bundles are stale against current master (`gnoweb_generate` + `main / build` fail on the merge ref). jaekwon's `$state`-routing hold remains the design gate regardless.

## Summary

Independent re-review of the exact commit the opus round approved; the goal was cross-checking its findings and hunting for what it missed, with a live gnodev boot. The hardened surfaces all verify clean in practice: input validation, error envelopes, the per-IP rate limiter, panic recovery, and the bare-colon-OID parse fix behave as documented when exercised against a running node. The opus round's two code warnings (`resolveBlock` panic path, `maxTypeDepth=8`) re-verify and are carried. New this round: the search feature's pagination footer drops the filter (its URL builder never learned about `search=`, while the address-bar builder did), CI went red since the opus pass because master moved under the PR's committed generated assets, and a handful of leftovers from the removed preview-orchestrator era (dead test helper file, unmounted JS controller, stale fan-out comment).

## Glossary

- `qpkg_json` / `qtype_json`: new VM ABCI query paths returning package state / type definition as JSON; gas-bounded, throwaway tx store.
- webargs / `$` grammar: gnoweb-only URL params (`/r/foo$state&oid=…`), split from the path before `?query`; invisible to realm `Render()`.
- OOB swap: htmx out-of-band swap; one response fragment also replaces `#state-sidebar` and `#state-kind-tabs` so badges stay coherent with filtered cards.
- merge ref: the synthetic PR+master merge commit CI builds (`refs/pull/N/merge`); drifts from the PR head as master moves.

## Critical (must fix)

None.

## Warnings (should fix)

- **[pagination silently exits an active search filter]** [`gno.land/pkg/gnoweb/feature/state/page.go:198`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/page.go#L198) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page.go#L198) — with more than one page of matches, the footer's First/Prev/Next/Last links drop `search=` and land on the unfiltered realm.
  <details><summary>details</summary>

  Two URL builders disagree: the htmx search path pushes [`canonicalStateURL`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/helpers.go#L141-L154) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L141-L154) into the address bar (carries `search=`, live-verified: `HX-Push-Url: /r/demo/closuretest$search=ste&state`), but the footer hrefs come from [`buildPagination` → `statePageHref`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/helpers.go#L219-L244) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L219-L244), which stamps only offset/limit/view. With >`maxTopLevelDecls` (500) matches the footer renders and a Next click serves the unfiltered realm at offset=500; the PR body's "Page links carry &offset=N and survive search" does not hold. Behavioral repro: [search_pagination_repro_test.go](tests/search_pagination_repro_test.go) fails at `e881032e5` (the Last href renders as `/r/demo$offset=500&state`, no `search=`) and passes once the filter is threaded through; see [comment_claude-fable-5.md](comment_claude-fable-5.md) for the run. The existing search tests stop at 12 decls, below one page. Fix: pass the active search query through `buildPagination`/`statePageHref` (mirroring `canonicalStateURL`) and extend the search test fixture past 500 matches.
  </details>

- **[CI red: committed gnoweb bundles stale against current master]** [`gno.land/pkg/gnoweb/public/main.css`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/public/main.css) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/public/main.css) — `gnoweb_generate` and `main / build` fail; regenerating on the merge ref diffs `controller-copy.js`, `controller-theme.js`, `main.css`.
  <details><summary>details</summary>

  Not a defect in the PR's own commit: at `e881032e5` itself `cd gno.land/pkg/gnoweb && npm ci && make generate` leaves the tree clean (verified locally). The diff appears on CI's PR+master merge ref because master moved since this branch last merged it — frontend changes [#5615 (copy origin)](https://github.com/gnolang/gno/commit/3961a0d09), [#5761 (omnibar)](https://github.com/gnolang/gno/commit/bf5b31eda), and npm dep bumps now feed the same bundles this PR commits. Fix: merge current master, run `make gnoweb.generate` under `./gno.land`, commit the regenerated assets. Side benefit: #5615 prefixes copied `/`-rooted paths with `location.origin`, so the merge also fixes the new state Link buttons' path-only copies (jeronimoalbi's shareable-URL comment) with no PR change; the `controller-copy.ts` hunks don't overlap, so the merge should be conflict-free.
  </details>

- **[opaque 500 where a clean error was intended]** [carried: opus round](claude-opus-4-8_davd-gzl.md) [`gno.land/pkg/sdk/vm/keeper.go:1714`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1714) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1714) — `resolveBlock` uses panicking `GetObject` where the same file uses `GetObjectSafe` for the same degrade-gracefully reason.
  <details><summary>details</summary>

  Re-verified, still present. [`keeper.go:1545-1551`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) documents the pattern ("a stale ref must degrade … rather than 500 the whole page") 160 lines up. The panic is contained by `doRecoverQueryNoMachine`, so it costs a confusing `VM panic … Stacktrace` 500 instead of a crash. Fix: switch to `GetObjectSafe`; the existing `if b, ok := obj.(*gno.Block)` guard already handles nil.
  </details>

- **[`maxTypeDepth=8` truncates common types to `null`]** [carried: opus round](claude-opus-4-8_davd-gzl.md) [`gno.land/pkg/sdk/vm/keeper.go:1627`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1627) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1627) — round-1 feedback still unaddressed; nested composites hit the bound long before any real cycle and emit bare `null`, indistinguishable from a nil type.
  <details><summary>details</summary>

  Re-verified, value unchanged across both rounds. `GetTypeSafe`/`fillType` already resolve the type graph with a cycle guard before `marshalTypeJSON` runs, so the depth bound mainly clips legitimate deep types. Fix: raise to ~32 and emit a sentinel (`{"@type":"/gno.Truncated"}`) instead of silent `null`.
  </details>

- **[maintainer hold: `$state` special-cases routing without rejecting unknown `$XYZ`]** [@jaekwon](https://github.com/gnolang/gno/pull/5649#issuecomment-4540827835) [`gno.land/pkg/gnoweb/weburl/url.go:285-296`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/weburl/url.go#L285-L296) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/weburl/url.go#L285-L296) — the active "do not merge yet" gate; design decision, not a code defect.
  <details><summary>details</summary>

  Confirmed live: `GET /r/demo/closuretest$foo` returns HTTP 200 and silently renders the normal realm page, exactly the inconsistency jaekwon flagged. Author acknowledged and offered a strict-allowlist follow-up as out of scope. The verdict stays gated on this thread regardless of the code items above.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/feature/state/errors.go:50`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/errors.go#L50) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/errors.go#L50) — carried from the [opus round](claude-opus-4-8_davd-gzl.md): `writeJSONError` omits `X-Content-Type-Options: nosniff`; every sibling JSON/fragment writer sets it. Add it to match.
- [`gno.land/pkg/gnoweb/feature/state/handler.go:13`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/handler.go#L13) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/handler.go#L13) — comment says "The page path fans out up to ~17 RPC calls"; since the orchestrator removal the page path does 2 (`StatePkg`+`Doc`, or `StateObject`+`StateType`). Stale rationale for `pageTimeout`; update the number.
- [`gno.land/pkg/gnoweb/feature/state/preview_test.go:19`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/preview_test.go#L19) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/preview_test.go#L19) — dead file: zero `Test*` functions, and both helpers (`previewStructBody`, `encodeInt64LE`) have no callers anywhere under `pkg/gnoweb` (leftover from the removed preview-orchestrator tests). Delete the file.
- [`gno.land/pkg/gnoweb/frontend/js/controller-search.ts:9`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/frontend/js/controller-search.ts#L9) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-search.ts#L9) — `SearchController` is mounted nowhere: no `data-controller="search"` in any template (the state search is pure htmx). The compiled `public/js/controller-search.js` ships as dead weight. Delete, or mount it where intended.
- [`gno.land/pkg/gnoweb/feature/state/walker.go:893`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/walker.go#L893) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/walker.go#L893) — `funcSignature` hides any param whose name merely starts with `cur` when its type is a RefType, so a legitimate `cursor mypkg.Cursor` param vanishes from rendered signatures. Match the exact crossing-param shape (name `cur` / type `realm`) instead of the prefix.
- [`gno.land/pkg/sdk/vm/keeper.go:1538`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1538) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1538) — `QueryPkg` filters only `""`/`"_"` names, so internal init slots surface as top-level declarations; confirmed live: closuretest renders an `init.4` card and TOC entry alongside real decls. Filter `init.*` (or label it deliberately).
- [`gno.land/pkg/gnoweb/handler_http.go:974`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/handler_http.go#L974) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L974) — `clientErrorMessage` with `height > 0` maps every non-NotFound error to 400 "block height N is not available", including genuine timeouts (`ErrClientTimeout` loses its 408). Check the timeout sentinel before the height short-circuit.
- [`gno.land/pkg/gnoweb/handler_http.go:700`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/handler_http.go#L700) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L700) — [@gfanton's silent-height-coercion concern](https://github.com/gnolang/gno/pull/5649#discussion_r3265645038) survives on the source/directory views: `GnoURL.Height()` coerces garbage to 0, so `$source&height=abc` renders latest with HTTP 200 and no signal (live-verified), while `$source&height=999999999` gets the friendly 400. The strict `ValidateHeightFromURL` fixed this for state pages, then left with the height-UI removal; its only callers now are tests. Run strict validation on the surviving `Height()` consumers, or fold into the planned height-cleanup follow-up.
- [`gno.land/pkg/gnoweb/feature/state/json_test.go:121-124`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/json_test.go#L121-L124) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json_test.go#L121-L124) — carried from the [opus round](claude-opus-4-8_davd-gzl.md): comment describes the envelope as `{pkg_path, height, total, offset, limit}` but `pkgJSONWrapper` has no `height` field. Drop `height` from the comment.
- [`gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:25`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L25) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L25) — `OID_PATTERN` (`^[a-f0-9]*:\d+$`) accepts empty/short hashlets that the server's `ValidateOID` (exactly 40 hex) rejects, so pasting `:1` redirects to a 400 page. Align the client pattern with the server's.

## Missing Tests

- **[search × pagination beyond one page]** [`gno.land/pkg/gnoweb/feature/state/page_test.go:463`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/page_test.go#L463) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page_test.go#L463) — every search test uses ≤12 decls, so the search+pagination interaction (the Warning above) is unpinned. [search_pagination_repro_test.go](tests/search_pagination_repro_test.go) is a drop-in regression test: fails at this head, passes when the filter is threaded through the footer hrefs.
- **[no cycle / depth / FuncType test for `marshalTypeJSON`]** [carried: opus round](claude-opus-4-8_davd-gzl.md) [`gno.land/pkg/sdk/vm/keeper_test.go:3344-3368`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper_test.go#L3344-L3368) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper_test.go#L3344-L3368) — still open: nothing exercises a self-referential `DeclaredType.Base`, `time.Time` end-to-end through `QueryType` (the `recover` in [`handler.go:312-319`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/handler.go#L312-L319) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/handler.go#L312-L319) is never triggered by tests), or FuncType/InterfaceType output.
- **[no concurrent rate-limiter test]** [carried: opus round](claude-opus-4-8_davd-gzl.md) [`gno.land/pkg/gnoweb/feature/state/ratelimit_test.go`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go) — all tests and the bench are single-goroutine, so `-race` never sees the lock contended. (Live behavior verified this round: 115 rapid requests → 101×200 then 14×429, untrusted `X-Real-IP` ignored.)
- **[gnojs decode.ts: no hostile-input cases]** [carried: opus round](claude-opus-4-8_davd-gzl.md) [`misc/gnojs/src/decode.test.ts`](https://github.com/gnolang/gno/blob/e881032e5/misc/gnojs/src/decode.test.ts) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.test.ts) — all 14 fixtures are well-formed; a published browser decoder of chain bytes should cover deep nesting, malformed base64, oversized values.

## Suggestions

- [`misc/gnojs/src/decode.ts:113-225`](https://github.com/gnolang/gno/blob/e881032e5/misc/gnojs/src/decode.ts#L113-L225) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.ts#L113-L225) — carried from the [opus round](claude-opus-4-8_davd-gzl.md): `decodeTypedValue` recurses inline trees with no depth cap (uncaught `RangeError` on hostile deep input) and `atob`/`decodeURIComponent` calls lack try/catch. Browser exposure is theoretical (gnoweb decodes server-side), but in-browser decode is the library's stated purpose.
- [`gno.land/pkg/sdk/vm/keeper.go:1556-1567`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1556-L1567) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1556-L1567) — carried from the [opus round](claude-opus-4-8_davd-gzl.md): a one-line doc comment noting `ExportValues` + `amino.MarshalJSON` are intentionally outside the gas meter (memory is transitively gas-bounded by the loads) would stop a future reader from filing it as a bug.

## Open questions

- Rate limiter slated to drop in lockstep with the nginx-config PR (gfanton thread, author agreed); fine as-is for this PR — tracked in the PR thread, no action needed here.
- Height-cleanup follow-up bundle (drop `Height()`'s `Query` fallback, `uint64` switch, `Realm()`/`ListPaths()` height threading) is the author's declared scope for the source-view height nits — tracked in the PR threads.
- `feature/state`'s `ClientAdapter` State* methods moving out of the main gnoweb interface — deferred per gfanton, lands with the package-boundary move.

## Notes

- Tests run at `e881032e5`: `gno.land/pkg/gnoweb/feature/state` (2s), `weburl`, `gnoweb` (71s), `components`, and the vm query-endpoint tests — all pass.
- Live verification (gnodev local from the worktree, closuretest realm): state page renders 8 decl cards with correct headers (`Cache-Control: public, max-age=1`, `Vary: HX-Request`, nosniff); `$state&json` returns the `{pkg_path,total,offset,limit,names,values}` envelope; `frag=node` returns chroma-highlighted func bodies; `frag=source` slices the exact span with a "See in code" permalink; htmx search returns the fragment + 2 OOB swaps with `HX-Push-Url` carrying the filter; bad oid → 400 in both HTML and JSON envelopes; missing realm JSON → 404 `{"error":"package not found"}`; bare-colon OID in `&oid=` parses (gfanton's round-1 case); 115-request burst → 101×200 then 429s with the bucket refilling afterward.
- The two failing CI checks (`gnoweb_generate`, `main / build`) share one root cause (stale bundles vs current master, see Warning) and will both clear with a master merge + regenerate commit. All other checks pass; the opus round's `main / lint` go-proxy flake is gone.
