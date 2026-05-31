# PR #5649: refactor: state explorer frontend

URL: https://github.com/gnolang/gno/pull/5649
Author: alexiscolin | Base: master | Files: 107 | +16225 -251
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `e881032e5` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5649 e881032e5`

**Verdict: APPROVE (blocked by maintainer hold)** — round-2 re-review after a substantial refactor. Both of gfanton's JSON-path findings are fixed and tested; round-1's cache-control and timeout concerns are fixed. The PR is mergeable on code grounds, but [jaekwon placed an explicit "do not merge yet" hold](https://github.com/gnolang/gno/pull/5649) on the `$state` routing-consistency question (unknown `$XYZ` should be rejected, not silently passed) — that design thread, not code quality, is the gate. Remaining code items are non-blocking: `maxTypeDepth=8` still silently truncates (carried from round 1), `resolveBlock` panics instead of degrading, and a few header/test nits. The failing `main / lint` CI check is a transient go-proxy fetch flake, not a real issue (see Notes).

## Summary

Round 1 reviewed commit `1506642` (APPROVE, 63 files). Since then the explorer was extracted into a self-contained `gno.land/pkg/gnoweb/feature/state` module (commit `0175e995`), the **time-travel / `?height=` UI was removed** from the state feature (`ea14dcd`), `state_orchestrator.go` was deleted (previews now hydrate client-side via htmx `hx-trigger="revealed"`), and the JSON error paths gfanton flagged were rewired. Now 107 files, +16225. This re-review focuses on the delta and on verifying the open reviewer threads — not on re-litigating the architecture, which round 1 already covered.

Net assessment: the untrusted-input surface (walker DoS bounds, cycle handling, rate limiter, ID/file/height validation) is carefully hardened and well-tested — no Critical or Warning issues there. The keeper-side JSON query endpoints are determinism-safe (ABCI query path, not consensus state) and gas-bounded. The actionable findings are a panic-path degrade, the unchanged `maxTypeDepth`, and missing cycle/FuncType tests.

## Glossary

- `feature/state`: new self-contained gnoweb module holding the explorer (walker, page, fragments, validate, ratelimit).
- `qpkg_json` / `qtype_json`: VM ABCI query paths returning package state / type definition as JSON; gas-bounded, run in a throwaway tx store.
- `marshalTypeJSON`: hand-rolled keeper-side type-graph serializer with `maxTypeDepth` cycle defense.
- `recoverToErr` / `recoverFetcher`: panic-recovery helpers — the former surfaces a 500 via a named return (fatal fetch), the latter logs and swallows (best-effort sibling fetch).
- `ExportValues`: gnovm serializer that breaks ephemeral cycles into `ExportRefValue{":N"}` markers before JSON emission.

## Resolved since round 1

Credited so the author sees what's closed:

- **gfanton — JSON path drops friendly height-error message** → moot. Height pinning was removed from the state feature; every state RPC call passes height `0` (latest), so the out-of-range path no longer exists. If `?height=` is ever re-enabled, the friendly-message logic must be re-added to [`feature/state/errors.go`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/errors.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/errors.go) (`mapClientError` has no `height` param).
- **gfanton — JSON request gets HTML on URL parse failure** → fixed. `isStateJSONRequest` is now checked before the HTML fallback at [`handler_http.go:223-226`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/handler_http.go#L223-L226) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L223-L226); covered by `TestHTTPHandler_StateJSONErrorOnBadURL`.
- **Round-1 — `max-age=86400, immutable` on cancelled context** → fixed. Now `stateCacheControl = "public, max-age=1"` ([`helpers.go:257`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/helpers.go#L257) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/helpers.go#L257)); a stale body expires in 1s and error paths fail fast before any cache header is stamped.
- **Round-1 — GET with no configured timeout hangs** → fixed. `defaultRequestTimeout = 30s` applied at [`handler_http.go:178-184`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/handler_http.go#L178-L184) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L178-L184); tested by `TestHTTPHandler_GetAlwaysBoundsContext`.
- **Round-1 — silent fetcher-panic recovery** → fixed. The deleted orchestrator's `_ = recover()` is gone; `recoverFetcher`/`recoverToErr` now log or surface every recovered panic.

## Critical (must fix)

None.

## Warnings (should fix)

- **[opaque 500 where a clean error was intended]** [`gno.land/pkg/sdk/vm/keeper.go:1713`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1713) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1713) — `resolveBlock` uses panicking `GetObject`, not the `GetObjectSafe` the author used 160 lines up for the same degrade-gracefully reason.
  <details><summary>details</summary>

  `resolveBlock` does `obj := store.GetObject(cv.ObjectID)`, which panics with `"unexpected object with id ..."` when the RefValue's target is absent (stale/missing block). The panic is caught by `doRecoverQueryNoMachine`, so the node survives — but the user gets an opaque `"VM panic: ... Stacktrace"` 500 instead of a clean "block not found". The author already knows the right pattern: [`keeper.go:1545-1551`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1545-L1551) uses `GetObjectSafe` with a comment ("a stale ref must degrade to ... rather than 500 the whole page"). Fix: switch `resolveBlock` to `GetObjectSafe`; the existing `if b, ok := obj.(*gno.Block); ok` guard already handles the nil return.
  </details>

- **[`maxTypeDepth=8` still truncates common types to `null`]** [`gno.land/pkg/sdk/vm/keeper.go:1626`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1626) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1626) — round-1 feedback unaddressed; value unchanged.
  <details><summary>details</summary>

  `const maxTypeDepth = 8`; when `depth > maxTypeDepth` (or `t == nil`), `marshalTypeJSON` writes `"null"` with no marker. Each nesting level (`PointerType.Elt`, `StructType.Field.Type`, `DeclaredType.Base`, `MapType.Key/Value`) costs one level, so a moderately nested composite (e.g. `[]*Foo` inside a struct inside a declared type) reaches 8 well before any real cycle and renders inner types as bare `null` — indistinguishable from a genuinely-nil type. Lower severity than it looks: `GetTypeSafe` → `fillType` already fully resolves the type graph with a `sealed`-flag cycle guard *before* `marshalTypeJSON` runs, so true infinite recursion is prevented upstream and depth-8 mainly just clips legitimately deep types. Fix: raise to ~32 and emit a sentinel (`{"@type":"/gno.Truncated"}`) instead of silent `null` so consumers can tell truncation from nil.
  </details>

- **[`$state` breaks the routing pattern without rejecting unknown `$XYZ`]** [@jaekwon](https://github.com/gnolang/gno/pull/5649) [`gno.land/pkg/gnoweb/weburl/url.go:285-286`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/weburl/url.go#L285-L286) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/weburl/url.go#L285-L286) — maintainer hold; surfaced here as the merge gate.
  <details><summary>details</summary>

  `$state` short-circuits routing the same way `$download`/`$help`/`$source` do, but jaekwon's point is consistency: if `$state` is special and bypasses `Render`, then any unhandled `$XYZ` should be rejected rather than silently swallowed. Author acknowledged and offered a strict-allowlist follow-up, calling it broader than this PR's scope. This is a design decision for the maintainers, not a code defect — but it is the active "do not merge yet" hold, so the verdict is gated on it regardless of code quality.
  </details>

## Nits

- [`gno.land/pkg/gnoweb/feature/state/errors.go:49-55`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/errors.go#L49-L55) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/errors.go#L49-L55) — `writeJSONError` (the state JSON *error* writer) sets only `Content-Type`, omitting `X-Content-Type-Options: nosniff`. Every sibling JSON writer sets it — [`json.go:138`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/json.go#L138) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json.go#L138), [`fragments.go:281`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/fragments.go#L281) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/fragments.go#L281), [`handler_http.go:909`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/handler_http.go#L909) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L909). Low risk, but inconsistent — add `nosniff` to match.
- [`gno.land/pkg/gnoweb/weburl/url.go:162-165`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/weburl/url.go#L162-L165) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/weburl/url.go#L162-L165) — `Height()` still reads from both `WebQuery` then `Query` as fallback; jeronimoalbi asked to drop the `Query` fallback. It's now live on the realm/source path ([`handler_http.go:679`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/handler_http.go#L679) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L679), `:797`), not dead — so it's a decision to close, not a defect. The author folded this into a "height-cleanup follow-up"; confirm the thread is signed off either way.
- [`gno.land/pkg/sdk/vm/keeper.go:1669-1672`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1669-L1672) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1669-L1672) — `marshalTypeJSON` emits `{"@type":"/gno.FuncType"}` and `{"@type":"/gno.InterfaceType"}` with no Params/Results/Methods, while `StructType.Fields` is expanded. Safe (valid JSON, no panic) and fine for the walker, but document it's deliberate so external `qtype_json` consumers don't file a phantom bug.
- [`gno.land/pkg/gnoweb/feature/state/json_test.go:123`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/json_test.go#L123) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/json_test.go#L123) — comment describes the envelope as `{pkg_path, height, total, offset, limit}`, but `pkgJSONWrapper` has no `height` field (leftover from the dropped pinning feature). Drop `height` from the comment.
- [`gno.land/pkg/gnoweb/feature/state/page.go:97`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/page.go#L97) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/page.go#L97) — `realmTotal := min(len(resp.Names), len(resp.Values))` silently tolerates a `Names`/`Values` length mismatch from the RPC. Defensively correct (no panic), but a mismatch signals a malformed payload and goes unlogged; a debug log would help diagnose a regressed upstream.

## Missing Tests

- **[no cycle / depth / FuncType test for `marshalTypeJSON`]** [`gno.land/pkg/sdk/vm/keeper_test.go:3105-3160`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper_test.go#L3105-L3160) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper_test.go#L3105-L3160) — round-1's ask is still open.
  <details><summary>details</summary>

  The only tests are `TestMarshalTypeJSON_ProducesValidJSONForControlCharNames` (flat StructType) and `TestQueryType_EnvelopeValidJSON` (flat IntType). Nothing exercises: (a) a self-referential `DeclaredType.Base` to prove depth-8 truncation terminates with valid JSON; (b) `time.Time` end-to-end through `QueryType` — the `recover` added to [`handler.go:312-319`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/handler.go#L312-L319) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/handler.go#L312-L319) for exactly this case is never triggered in tests; (c) `FuncType`/`InterfaceType` output. Fix: add a self-referential-`Base` test asserting bounded valid JSON, plus a `QueryType` test on a stdlib type containing `time.Time`.
  </details>

- **[no concurrent rate-limiter test]** [`gno.land/pkg/gnoweb/feature/state/ratelimit_test.go`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/ratelimit_test.go) — the limiter's lock is correct by inspection, but every test and the bench are single-goroutine, so `-race` has nothing concurrent to check. A `b.RunParallel` or N-goroutine fan-out hammering a shared limiter would actually exercise the lock under the detector.

- **[gnojs decode.ts: no hostile-input cases]** [`misc/gnojs/src/decode.test.ts`](https://github.com/gnolang/gno/blob/e881032e5/misc/gnojs/src/decode.test.ts) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.test.ts) — all 14 cases are well-formed fixtures. For a published browser decoder of untrusted chain bytes, add deep-nesting (recursion cap), malformed-base64, and oversized-`N` cases. See the decode.ts depth-cap suggestion below.

## Suggestions

- [`misc/gnojs/src/decode.ts:113-225`](https://github.com/gnolang/gno/blob/e881032e5/misc/gnojs/src/decode.ts#L113-L225) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.ts#L113-L225) — `decodeTypedValue` recurses through inline `StructValue.Fields`/`ArrayValue.List`/`MapValue`/`PointerValue.TV`/`HeapItemValue.Value` with no depth cap; deeply-nested input → uncaught `RangeError` (stack overflow). True cycles are safe (they serialize as `RefValue`/`ExportRefValue`, which return without recursing), so this is bounded-depth-but-unbounded only for genuinely deep inline trees. Not in the live gnoweb path (the explorer decodes server-side in Go — `decodePkg`/`decodeObject` have zero importers under `gno.land/`), so browser exposure is theoretical, but the library's stated purpose is in-browser decode. Add a `depth` param with a sane cap.
- [`misc/gnojs/src/primitives.ts:83`](https://github.com/gnolang/gno/blob/e881032e5/misc/gnojs/src/primitives.ts#L83) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/primitives.ts#L83) and [`decode.ts:168`](https://github.com/gnolang/gno/blob/e881032e5/misc/gnojs/src/decode.ts#L168) · [↗](../../../../../.worktrees/gno-review-5649/misc/gnojs/src/decode.ts#L168) — `atob` on chain-provided base64 with no try/catch aborts the whole decode on malformed input; `decodeURIComponent` in [`controller-state.ts:274`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/gnoweb/feature/state/frontend/controller-state.ts#L274) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/feature/state/frontend/controller-state.ts#L274) similarly throws on a hand-edited `%`. The controller wraps its calls in try/catch already; the decoder doesn't. Defensive wrapping degrades more gracefully.
- [`gno.land/pkg/sdk/vm/keeper.go:1556-1567`](https://github.com/gnolang/gno/blob/e881032e5/gno.land/pkg/sdk/vm/keeper.go#L1556-L1567) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1556-L1567) — `ExportValues` + `amino.MarshalJSON` run *after* the gas-metered store loads and are themselves unmetered. Not a practical OOM lever (loading the values is gas-bounded, which transitively bounds what's in memory; `ExportValues` breaks cycles and uses insertion-ordered `MapList`), but a one-line doc-comment noting serialization is intentionally outside the gas meter would prevent a future reader assuming it's a bug.

## Questions for Author

- jeronimoalbi noted the Link/copy-link buttons copy only the realm path, dropping scheme+domain (he says it predates this PR, also affects the Actions help view). Since this PR's whole pitch is shareable URLs, is fixing the copied link in scope here, or tracked separately?
- jaekwon's original hold also questioned "Gno objects aren't stored in the iavl store so I don't see how this is possible" re: historical state. Now that `?height=` pinning is removed from the state feature, is that concern fully retired, or does the planned re-enablement still need to answer it?

## Notes

- Tests pass in the worktree at `e881032e5`: `gno.land/pkg/sdk/vm/...`, `gno.land/pkg/gnoweb/feature/state/...` (with `-race`, no races), and `gno.land/pkg/gnoweb/...`. The lone failure (`gnoweb/markdown` sanitize-integration) is a pre-existing gnovm-preprocessor panic in an untouched package, unrelated to this PR.
- The failing `main / lint` CI check is a transient infra flake, not a code issue: `proxy.golang.org` returned a stream `INTERNAL_ERROR` fetching `cockroachdb/pebble@v1.1.5`. Re-run the job.
