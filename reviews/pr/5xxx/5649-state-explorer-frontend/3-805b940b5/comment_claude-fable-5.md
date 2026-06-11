# Review: PR #5649
Event: REQUEST_CHANGES

## Body
Repros run on the current head (805b940b5). The red CI checks are just stale generated bundles, not a code problem; regenerating also revives the state page's Copy-JSON button, which currently ships without its handler.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5649-state-explorer-frontend/3-805b940b5/review_claude-fable-5_davd-gzl.md · [↗](./review_claude-fable-5_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/page.go:198 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page.go#L198)
With more than one page of search matches, the pagination links drop `search=` and land on the unfiltered realm: `statePageHref` stamps only offset/limit/view, so "Page links carry &offset=N and survive search" from the PR body does not hold. Fix: pass the active search query through `buildPagination`/`statePageHref` (mirroring [`canonicalStateURL`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/helpers.go#L141-L154) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L141)) and extend the search test fixture past 500 matches.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5649 -R gnolang/gno
curl -fsSL -o gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5649-state-explorer-frontend/3-805b940b5/tests/search_pagination_repro_test.go
go test -run 'TestSearchPaginationKeepsFilter' ./gno.land/pkg/gnoweb/feature/state/
rm gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go
```

```
--- FAIL: TestSearchPaginationKeepsFilter (0.04s)
    search_pagination_repro_test.go:94: pagination hrefs dropped the active search filter:
          <a class="b-btn b-btn--secondary c-with-icon" href="/r/demo$offset=500&amp;state" rel="next" aria-label="Next page">
          <a class="b-btn b-btn--secondary c-with-icon" href="/r/demo$offset=500&amp;state" aria-label="Last page">
FAIL
```
</details>

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/json.go:93 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json.go#L93)
Asking for state at a past block (`?height=N`) silently returns the latest state with HTTP 200, on every state surface and for any height value. A consumer who pins a height reads current data with no way to tell. Fix: reject `height` on the state surfaces with a 400 until pinning is implemented ([`ValidateHeightFromURL`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/validate.go#L87) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/validate.go#L87) already exists for this), and update the PR description.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5649 -R gnolang/gno
(cd contribs/gnodev && go run . local ../../examples/gno.land/r/demo/closuretest) &
until curl -sf -o /dev/null http://localhost:8888/; do sleep 2; done
# pin far beyond the dev chain's 2-block tip; a pinned query should not return latest data with 200:
curl -s -o /tmp/pinned.json -w 'HTTP %{http_code}, Cache-Control: %header{cache-control}\n' \
  'http://localhost:8888/r/demo/closuretest$state&json?height=999999'
curl -s 'http://localhost:8888/r/demo/closuretest$state&json' | python3 -c \
  'import json,sys; print("pinned == latest:", json.load(sys.stdin)==json.load(open("/tmp/pinned.json")))'
kill %1
```

```
HTTP 200, Cache-Control: public, max-age=1
pinned == latest: True
```
</details>

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1714 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1714)
`resolveBlock` uses panicking `GetObject` where [`keeper.go:1545-1551`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1545) uses `GetObjectSafe` for the same degrade-gracefully reason; the panic is contained by `doRecoverQueryNoMachine` but surfaces as an opaque "VM panic ... Stacktrace" 500 instead of a clean error. Fix: switch to `GetObjectSafe`; the existing `if b, ok := obj.(*gno.Block)` guard already handles the nil return.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1627 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1627)
`maxTypeDepth=8` trips on moderately nested composites (each `PointerType.Elt` / `StructType.Field.Type` / `DeclaredType.Base` level costs a slot) well before any real cycle, and `marshalTypeJSON` emits bare `null`, indistinguishable from a genuinely nil type; `GetTypeSafe`/`fillType` already cycle-guard the graph upstream. Fix: raise to ~32 and emit a sentinel like `{"@type":"/gno.Truncated"}` instead of silent `null`.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1538 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1538)
`QueryPkg` filters only `""`/`"_"` names, so internal init slots surface as top-level declarations: `closuretest` renders an `init.4` card and TOC entry alongside real decls. Fix: skip `init.*` slots (or label them deliberately if exposing them is intended).

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1556-1567 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1556)
`ExportValues` + `amino.MarshalJSON` run after the gas-metered loads and are themselves unmetered, which is fine (memory is transitively gas-bounded) but looks like an oversight. Fix: a one-line doc comment stating serialization is intentionally outside the gas meter.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper_test.go:3344-3368 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper_test.go#L3344)
No test exercises `marshalTypeJSON`'s defenses: a self-referential `DeclaredType.Base` (depth-bound termination), `time.Time` end-to-end through `QueryType` (the `recover` at [`handler.go:323-331`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/sdk/vm/handler.go#L323-L331) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/handler.go#L323) never fires in tests), or `FuncType`/`InterfaceType` output. Fix: add a self-referential-Base test asserting bounded valid JSON plus a `QueryType` test on a type containing `time.Time`.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/errors.go:50 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/errors.go#L50)
`writeJSONError` sets only `Content-Type`, omitting `X-Content-Type-Options: nosniff`, while every sibling JSON/fragment writer sets it ([`json.go:138`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/json.go#L138) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json.go#L138), [`fragments.go:248`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/fragments.go#L248) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/fragments.go#L248)). Fix: add the header to match.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/handler.go:13 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/handler.go#L13)
The comment justifying `pageTimeout` says "The page path fans out up to ~17 RPC calls", but since the orchestrator removal the page path does 2 (`StatePkg`+`Doc`, or `StateObject`+`StateType`); previews hydrate via separate rate-limited fragment GETs. Fix: update the number.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/preview_test.go:19 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/preview_test.go#L19)
Dead file: it contains zero `Test*` functions and both helpers (`previewStructBody`, `encodeInt64LE`) have no callers anywhere under `pkg/gnoweb`, a leftover from the removed preview-orchestrator tests. Fix: delete the file.

*(AI Agent)*

## gno.land/pkg/gnoweb/frontend/js/controller-search.ts:9 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-search.ts#L9)
`SearchController` is mounted nowhere: no `data-controller="search"` exists in any template (the state search is pure htmx), so this file and the compiled `public/js/controller-search.js` ship as dead code. Fix: delete both, or mount the controller where it was intended.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/walker.go:893 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/walker.go#L893)
`funcSignature` hides any param whose name merely starts with `cur` when its type is a RefType, so a legitimate `cursor mypkg.Cursor` param silently vanishes from rendered signatures. Fix: match the exact crossing-param shape (name `cur`, `realm` type) instead of the prefix.

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:974 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L974)
`clientErrorMessage` with `height > 0` maps every non-NotFound error to 400 "block height N is not available", including genuine timeouts, so a slow node on a pinned source view misreports as a bad height. Fix: check `ErrClientTimeout` before the height short-circuit.

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:700 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L700)
`GnoURL.Height()` coerces garbage to 0, so `$source&height=abc` renders latest with HTTP 200 and no signal, while out-of-range heights get the friendly 400; the strict `ValidateHeightFromURL` is left with only test callers. Fix: run strict height validation in `GetSourceView`/`GetDirectoryView`, or fold it into the planned height-cleanup follow-up.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/json_test.go:121-124 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json_test.go#L121)
The comment describes the envelope as `{pkg_path, height, total, offset, limit}`, but `pkgJSONWrapper` has no `height` field (leftover from the dropped pinning feature). Fix: drop `height` from the comment.

*(AI Agent)*

## gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:25 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L25)
`OID_PATTERN` (`^[a-f0-9]*:\d+$`) accepts empty or short hashlets that the server's `ValidateOID` (exactly 40 hex chars) rejects, so pasting `:1` into the searchbar redirects to a 400 "invalid object id" page. Fix: align the client pattern with the server's 40-hex rule.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/ratelimit_test.go:1 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go#L1)
Every limiter test and the bench are single-goroutine, so `-race` never observes the mutex contended (behavior itself verified live). Fix: add an N-goroutine fan-out (or `b.RunParallel`) hammering one shared limiter under `-race`.

*(AI Agent)*

## misc/gnojs/src/decode.ts:113 [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.ts#L113)
`decodeTypedValue` recurses through inline `StructValue`/`ArrayValue`/`MapValue`/`PointerValue` trees with no depth cap, so deep hostile input throws an uncaught `RangeError`, and the `atob` calls abort the whole decode on malformed base64. gnoweb no longer uses this path, but in-browser decode of untrusted chain bytes is the library's stated purpose. Fix: add a depth cap, wrap `atob` defensively, and add hostile-input cases to [`decode.test.ts`](https://github.com/gnolang/gno/blob/805b940b5/misc/gnojs/src/decode.test.ts) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.test.ts).

*(AI Agent)*
