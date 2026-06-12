# Review: PR #5649
Posted: https://github.com/gnolang/gno/pull/5649#pullrequestreview-4487230385
Event: REQUEST_CHANGES

## Body
Repros run on 805b940b5. The red CI checks are just stale generated bundles, not a code problem; regenerating also revives the state page's Copy-JSON button, which currently ships without its handler.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5649-state-explorer-frontend/3-805b940b5/review_claude-fable-5_davd-gzl.md · [↗](./review_claude-fable-5_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/page.go:198 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page.go#L198)
If a search returns more than one page of matches, clicking Next drops the filter and lands on the unfiltered realm. The pagination links ([`statePageHref`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/helpers.go#L122-L139) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L122)) never include `search=`. Fix: carry the search query in the pagination links, the way [`canonicalStateURL`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/helpers.go#L141-L154) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L141) already does for the address bar, and add a search test with more than 500 matches.

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
Asking for state at a past block (`?height=N`) silently returns the latest state with HTTP 200, so anyone pinning a height reads current data with no way to tell. Fix: reject `height` on the state surfaces with a 400 until pinning is implemented ([`ValidateHeightFromURL`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/validate.go#L87) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/validate.go#L87) already exists for this), and update the PR description.

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
A stale reference here panics and reaches the user as an opaque "VM panic ... Stacktrace" 500, while the same situation at [`keeper.go:1545-1551`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1545) degrades cleanly with `GetObjectSafe`. Fix: switch to `GetObjectSafe`; the existing nil guard below already handles the rest.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1627 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1627)
Types nested more than 8 levels deep render as a bare [`null`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/sdk/vm/keeper.go#L1632-L1633) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1632), which looks identical to a missing type. Every layer costs one level (a pointer, a slice element, a struct field's type, a declared type's base, a map key or value), so an ordinary declared struct holding a `map[string][]*Other` already burns 6-8 levels and gets clipped while perfectly valid. The cap is only a backstop: real cycles are broken upstream by [`fillType`'s cycle guard](https://github.com/gnolang/gno/blob/805b940b5/gnovm/pkg/gnolang/realm.go#L1810-L1814) · [↗](../../../../../.worktrees/gno-review-5649/gnovm/pkg/gnolang/realm.go#L1810) before this serializer runs. Fix: raise the cap to ~32 (roughly double the worst legitimate nesting, still a trivially cheap recursion bound) and emit an explicit marker like `{"@type":"/gno.Truncated"}` instead of `null`, so consumers can tell truncation from a nil type.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper.go:1538 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1538)
Internal `init.N` slots show up as top-level declarations next to real ones: closuretest's [`func init()`](https://github.com/gnolang/gno/blob/805b940b5/examples/gno.land/r/demo/closuretest/closuretest.gno#L15) · [↗](../../../../../.worktrees/gno-review-5649/examples/gno.land/r/demo/closuretest/closuretest.gno#L15) renders as an `init.4` card and TOC entry on `/r/demo/closuretest$state` (the preprocessor renames every `init` to `init.N`, [`preprocess.go:468-475`](https://github.com/gnolang/gno/blob/805b940b5/gnovm/pkg/gnolang/preprocess.go#L468-L475) · [↗](../../../../../.worktrees/gno-review-5649/gnovm/pkg/gnolang/preprocess.go#L468)). Fix: filter `init.*` names out, or label them clearly if showing them is intended.

*(AI Agent)*

## SKIP gno.land/pkg/sdk/vm/keeper.go:1556-1567 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1556)
The JSON serialization here runs outside the gas meter, which is fine (everything it touches was already gas-bounded when loaded) but reads like an oversight. Fix: one doc comment saying it is intentional.

*(AI Agent)*

## gno.land/pkg/sdk/vm/keeper_test.go:3344-3368 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper_test.go#L3344)
The type-serializer's defenses are untested: nothing exercises a self-referencing type, a type containing `time.Time` (the recover at [`handler.go:323-331`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/sdk/vm/handler.go#L323-L331) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/handler.go#L323) never fires in tests), or function/interface types. Fix: add a self-referencing-type test asserting bounded valid JSON, plus a `QueryType` test on a type containing `time.Time`.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/errors.go:50 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/errors.go#L50)
JSON error responses miss the `X-Content-Type-Options: nosniff` header that every other JSON and fragment response sets ([`json.go:138`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/json.go#L138) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json.go#L138), [`fragments.go:248`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/fragments.go#L248) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/fragments.go#L248)).

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/handler.go:13 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/handler.go#L13)
This comment justifies the page timeout with "up to ~17 RPC calls", but the page path now makes 2 ([`page.go:71-76`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/page.go#L71-L76) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page.go#L71)).

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/preview_test.go:19 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/preview_test.go#L19)
Dead file: it contains no tests, and its two helpers are called from nowhere; a leftover from the removed preview orchestrator.

*(AI Agent)*

## gno.land/pkg/gnoweb/frontend/js/controller-search.ts:9 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-search.ts#L9)
This controller is never mounted: no template uses `data-controller="search"` (the state search runs on htmx), so the file and its compiled bundle ship as dead code. Fix: delete both, or mount it where it was meant to go.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/walker.go:893 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/walker.go#L893)
Any parameter whose name merely starts with `cur` is hidden from rendered function signatures, so a legitimate `cursor` parameter silently disappears. Fix: match the exact crossing-parameter shape (name `cur`, type `realm`) instead of the prefix.

*(AI Agent)*

## SKIP gno.land/pkg/gnoweb/handler_http.go:974 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L974)
When a height is pinned, every error except not-found becomes 400 "block height N is not available", including real timeouts, so a slow node looks like a bad height. Fix: check the timeout error before this branch.

*(AI Agent)*

## SKIP gno.land/pkg/gnoweb/handler_http.go:700 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L700)
A garbage height like `$source&height=abc` is silently treated as "latest" and renders with 200, while a too-large height gets a clear 400; the strict validator that would catch this ([`ValidateHeightFromURL`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/validate.go#L87) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/validate.go#L87)) exists but is only used by tests. Fix: validate the height strictly in the source and directory views, or fold it into the planned height-cleanup follow-up.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/json_test.go:121-124 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json_test.go#L121)
This comment says the envelope contains `height`, but it doesn't (leftover from the dropped pinning feature). Fix: drop `height` from the comment.

*(AI Agent)*

## gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:25 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L25)
The searchbar accepts object IDs the server rejects (the client pattern allows an empty hash, [`ValidateOID`](https://github.com/gnolang/gno/blob/805b940b5/gno.land/pkg/gnoweb/feature/state/validate.go#L36) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/validate.go#L36) wants exactly 40 hex chars), so pasting `:1` lands on a 400 page. Fix: align the client pattern with the server's rule.

*(AI Agent)*

## gno.land/pkg/gnoweb/feature/state/ratelimit_test.go:1 [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go#L1)
All rate-limiter tests run on one goroutine, so the race detector never sees the lock contended (the behavior itself was verified live). Fix: add a test that hammers one shared limiter from many goroutines under `-race`.

*(AI Agent)*

## misc/gnojs/src/decode.ts:113 [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.ts#L113)
The decoder recurses into nested values with no depth cap, so deeply nested hostile input crashes with a stack overflow, and malformed base64 aborts the whole decode; decoding untrusted chain bytes in the browser is the library's stated purpose. The nested value stays small on the wire (well under the 8 MiB RPC cap), so the cap doesn't help. Fix: add a depth cap, wrap the base64 decode defensively, and add hostile-input cases to [`decode.test.ts`](https://github.com/gnolang/gno/blob/805b940b5/misc/gnojs/src/decode.test.ts) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.test.ts).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5649 -R gnolang/gno
cd misc/gnojs
cat > poc.mjs <<'EOF'
import { decodeTypedValue } from "./src/decode.js";
// A slice whose only element is the next level down, nested 60k deep.
function nest(d) {
  let tv = { T: { "@type": "/gno.PrimitiveType", value: "16" }, N: "AA==" };
  for (let i = 0; i < d; i++)
    tv = { T: { "@type": "/gno.SliceType" }, V: { "@type": "/gno.ArrayValue", List: [tv] } };
  return tv;
}
try { decodeTypedValue("root", nest(60000)); console.log("returned without error"); }
catch (e) { console.log(e.constructor.name + ": " + e.message); }
EOF
npx --yes tsx poc.mjs
rm poc.mjs
```

```
RangeError: Maximum call stack size exceeded
```
</details>

*(AI Agent)*
