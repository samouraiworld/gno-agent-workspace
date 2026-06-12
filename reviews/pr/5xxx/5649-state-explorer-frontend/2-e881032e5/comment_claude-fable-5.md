# Review: PR #5649
Event: REQUEST_CHANGES

## Body
Repros run on e881032e5. The red CI checks are just stale bundles vs master, not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5649-state-explorer-frontend/2-e881032e5/review_claude-fable-5_davd-gzl.md · [↗](./review_claude-fable-5_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/page.go:198 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page.go#L198)
When a search has more than one page of results, the Next/Prev links lose the `search=` filter: clicking them shows the unfiltered realm. Fix: thread the search query through `buildPagination`/`statePageHref` (like [`canonicalStateURL`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/helpers.go#L141-L154) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L141) does) and add a search test with more than 500 matches.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5649 -R gnolang/gno
curl -fsSL -o gno.land/pkg/gnoweb/feature/state/search_pagination_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5649-state-explorer-frontend/2-e881032e5/tests/search_pagination_repro_test.go
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

## gno.land/pkg/sdk/vm/keeper.go:1714 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1714)
`GetObject` panics when the object is missing, so the user gets an opaque "VM panic ... Stacktrace" 500 instead of a clean error. Fix: use `GetObjectSafe` like [`keeper.go:1545-1551`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1545) does; the `*gno.Block` check below already handles nil.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1627 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1627)
Ordinary nested types reach `maxTypeDepth=8` long before any real cycle, and the serializer then writes `null`, which looks like a missing type. Fix: raise the bound to ~32 and write an explicit truncation marker instead of `null`.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1538 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1538)
Internal init slots are not filtered out, so the explorer shows an `init.4` card next to real declarations (visible on `closuretest`). Fix: skip `init.*` names, or keep them on purpose with a label.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1556-1567 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1556)
Serialization here runs outside the gas meter, which is fine but looks like an oversight. Fix: one doc comment saying it is intentional.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper_test.go:3344-3368 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper_test.go#L3344)
The tricky cases of `marshalTypeJSON` have no tests: a type that references itself, `time.Time` through `QueryType` (its `recover` never fires in tests), func/interface types. Fix: add a self-referencing-type test and a `QueryType` test on a type containing `time.Time`.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/errors.go:50 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/errors.go#L50)
`writeJSONError` is the only JSON writer that doesn't set `X-Content-Type-Options: nosniff`. Fix: add the header to match the others.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/handler.go:13 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/handler.go#L13)
This comment says the page path fans out to ~17 RPC calls, but it now does 2. Fix: update the number.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/preview_test.go:19 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/preview_test.go#L19)
Dead file: no tests, and its two helpers are used nowhere. Fix: delete it.

*(AI Agent)*

## gno.land/pkg/gnoweb/frontend/js/controller-search.ts:9 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-search.ts#L9)
`SearchController` is never mounted (no `data-controller="search"` in any template), so this file and its compiled bundle are dead code. Fix: delete both, or mount it where intended.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/walker.go:893 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/walker.go#L893)
Any parameter whose name starts with `cur` is hidden from the displayed signature, so a real param like `cursor mypkg.Cursor` disappears. Fix: match the exact crossing param (`cur realm`) instead of the prefix.

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:974 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L974)
With a pinned height, every error becomes "block height N is not available", including timeouts. Fix: check the timeout error before the height branch.

*(AI Agent)*

## gno.land/pkg/gnoweb/handler_http.go:700 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L700)
A garbage height like `$source&height=abc` silently renders as latest (HTTP 200, no signal), while an out-of-range height correctly gets a 400. Fix: validate the height strictly here too; `ValidateHeightFromURL` exists but only tests call it.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/json_test.go:121-124 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json_test.go#L121)
The comment still lists a `height` field that the envelope no longer has. Fix: drop it.

*(AI Agent)*

## gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:25 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L25)
The client OID pattern accepts short IDs that the server rejects, so pasting `:1` lands on a 400 page. Fix: use the same 40-hex rule as the server's `ValidateOID`.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/ratelimit_test.go:1 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go#L1)
All limiter tests are single-goroutine, so `-race` never sees the lock under contention (behavior itself verified live). Fix: add a multi-goroutine test hammering one shared limiter.

*(AI Agent)*

## misc/gnojs/src/decode.ts:113 [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.ts#L113)
Deeply nested input crashes `decodeTypedValue` (no recursion cap), and one malformed base64 string aborts the whole decode; this library's purpose is decoding untrusted chain bytes in the browser. Fix: add a depth cap, wrap `atob`, and add bad-input tests.

*(AI Agent)*
