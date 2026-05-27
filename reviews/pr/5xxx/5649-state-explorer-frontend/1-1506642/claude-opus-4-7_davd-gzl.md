# PR #5649: refactor: state explorer frontend

URL: https://github.com/gnolang/gno/pull/5649
Author: alexiscolin | Base: master | Files: 63 | +11135 -188
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `1506642` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5649 1506642`

**Verdict: APPROVE** — well-tested feature work with sensible defense-in-depth; warnings are polish, none blocking. Closest to a real concern is `maxTypeDepth=8` mislabeling fields on deep generic-flavored types.

## Summary

Moves the State Explorer rendering pipeline from client-side TypeScript (`@gnojs/amino`) to server-side Go templates, while preserving and slightly extending the VM JSON query surface. Every object/realm now lives at a bookmarkable URL (`/r/<path>$state[&oid=&tid=&height=N]`); `?height=N` propagates through all fetches via the new `ABCIQueryWithOptions` path so a pinned view is fully consistent at that block. Two new VM endpoints (`vm/qpkg_json`, `vm/qtype_json`) emit gas-bounded JSON; an orchestrator walks the decoded tree with bounded fan-out (8 concurrent fetches, ≤30 previews/render, 2 rounds) and a custom `marshalTypeJSON` defends the type-graph emitter against `time.Time`-style self-loops.

```
old: client                    chain
     ─── ABCIQuery ───────────► vm/qrender, vm/qfile
     ◄── amino-encoded ─────────
     │
     ├─ amino-js decode in browser
     ├─ render tree
     └─ no URL state, no time-travel

new: client       gnoweb (Go templates)              chain
     ─── HTTP ─► ─── ABCIQueryWithOptions(height) ─► vm/qpkg_json, vm/qtype_json, vm/qrender, vm/qfile
                 ◄── amino-JSON ─────────────────────
                 │
                 ├─ StateNode walker (depth-capped, dedup'd)
                 ├─ orchestrator: bounded preview fan-out
                 └─ stable URL with oid/tid/height
     ◄── server-rendered HTML / JSON envelope ───
```

## Glossary

- `oid` / `tid`: object ID / TypeID query params; capped at `maxStateIDLength=256` bytes.
- `StateNode`: in-memory tree the walker decodes Amino-JSON into for templating.
- `qpkg_json` / `qtype_json`: new VM query paths returning package state / type definition as JSON.
- `marshalTypeJSON`: hand-rolled keeper-side serializer that replaces `amino.MarshalJSON` for type-graph emission, with `maxTypeDepth` cycle defense.
- `ABCIQueryWithOptions`: ABCI query variant that accepts a target `height`, used for time-travel.

## Fix

Server-side rendering replaces the browser-side decoder: a 492-line template at [`views/state.html`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/views/state.html) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/views/state.html) drives a `StateNode` model produced by [`state_walker.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_walker.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_walker.go), enriched by [`state_orchestrator.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go) which runs inline-preview + source-snippet fetches under shared semaphores. Keeper additions at [`gno.land/pkg/sdk/vm/keeper.go +214`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/sdk/vm/keeper.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go) introduce `vm/qpkg_json` (top-level package state) and `vm/qtype_json` (type definition by TypeID), both gas-bounded via `maxGasQuery` and living in a throwaway transaction store. Stable JSON API surface (`?state&json` and friends) passes raw chain bytes through with proper status codes, an `{"error":"..."}` envelope, and `Cache-Control` headers (`max-age=1` for latest, `max-age=86400, immutable` for pinned-height). gnoweb hardening already on the branch: `MaxBytesReader`-capped POST body (64 KiB), `MaxUserContributions=200`, open-redirect guard, request timeout, dedupe; DoS bounds: `maxStateIDLength=256`, `maxChildrenPerNode=500`, `maxDecodeDepth=256`, `maxTypeDepth=8`.

## Critical (must fix)

None.

## Warnings (should fix)

- **[bound too tight; mislabels common types]** [`gno.land/pkg/sdk/vm/keeper.go:1606`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/sdk/vm/keeper.go#L1606) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1606) — `maxTypeDepth=8` trips on any moderately nested composite, emitting `null` and forcing positional field rendering.
  <details><summary>details</summary>

  Any moderately nested generic-flavored composite (e.g. `*[]map[K]struct{...}` declared atop another DeclaredType, AVL trees with `*Node{left, right, value any}`) reaches depth 8 well before any infinite cycle would. When the bound trips, the emitter writes `null` and the walker on the client side then renders fields by positional index instead of by name — users perceive this as "the State Explorer suddenly forgets field names on deep types." The depth-bound is an *acceptable* anti-DoS but doubles as a feature defect for the common case.

  Fix: either (a) raise the bound to ~32 — still far below the cycle length needed to crash a goroutine given the per-frame cost is tiny; or (b) replace recursion with a `seen map[gno.TypeID]struct{}` cycle-detection set, which is the proper fix.
  </details>

- **[endpoint contract under-documented]** [`gno.land/pkg/sdk/vm/keeper.go:1649`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/sdk/vm/keeper.go#L1649) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1649) — `marshalTypeJSON` emits `{"@type":"/gno.FuncType"}` with no `Params`/`Results`; the new docs advertise the endpoint as "type definition (struct fields, etc.)".
  <details><summary>details</summary>

  Fine for the gnoweb walker today (field names come from `*gno.StructType`, not from func signatures), but the new [`docs/builders/query-state-api.md`](https://github.com/gnolang/gno/blob/1506642/docs/builders/query-state-api.md) · [↗](../../../../../.worktrees/gno-review-5649/docs/builders/query-state-api.md) leaves a footgun for any external consumer of `vm/qtype_json` who reads "type definition (struct fields, etc.)" and assumes the absence of `Params`/`Results` is a bug.

  Fix: add an explicit "function-type Params/Results are omitted" note in `docs/builders/query-state-api.md` so callers don't file a phantom bug.
  </details>

- **[cancelled output cached for 24h]** [`gno.land/pkg/gnoweb/handler_http.go:817`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go#L817) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L817) — `ServeStateJSON` writes the body without checking `ctx.Err()` after `wg.Wait()`; on the pinned-height path the bad payload is then cached for a day.
  <details><summary>details</summary>

  If the upstream RPC honors cancellation and returns partial/empty bytes, the handler still sets `Cache-Control: public, max-age=86400, immutable` and ships them. Blast radius is tiny (the only inputs are short fetches) but the immutable cache directive turns a transient cancellation into a 24-hour wrong-answer.

  Fix: bail before `w.WriteHeader` when `ctx.Err() != nil` — mirror the same check that already gates `walkRenderSnippets` at [`state_orchestrator.go:59`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go#L59) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go#L59).
  </details>

- **[silent panic, no operator breadcrumb]** [`gno.land/pkg/gnoweb/components/state_orchestrator.go:95`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go#L95) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go#L95), [`:222`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go#L222) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go#L222), [`:243`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go#L243) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go#L243) — `defer func() { _ = recover() }()` swallows fetcher panics with no logging.
  <details><summary>details</summary>

  The comment justifying it ("fetcher panics must not crash the process") is correct, but the silent path means a regressed fetcher (e.g. rpcClient swallowing a nil deref) shows up only as "the State Explorer is missing some children" with nothing in operator logs.

  Fix: `if r := recover(); r != nil { logFromCtx(ctx).Error("state preview fetcher panic", "err", r) }` with a small logger plumbed in, or at minimum a `runtime/debug.Stack()` print to stderr.
  </details>

- **[silent decay risk]** [`gno.land/pkg/sdk/vm/keeper.go:1517-1532`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/sdk/vm/keeper.go#L1517-L1532) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1517-L1532) — `QueryPkg` pairs `block.Values` with `names := sb.Names` by position; a one-line invariant comment would protect against future name/value swaps.
  <details><summary>details</summary>

  The `if i >= len(names) { break }` guard is correct and the walker tests pass, but the function also unwraps heap items in place and filters `varNames` to skip blanks — the `_` filter runs *after* the index match (correct), but it would be easy to invert in a future refactor. The whole block depends on the order in `block.Values` matching `sb.Names[i]`; if `sb.Names` later interleaves non-decl entries (e.g. an `init` slot), the values silently take on the wrong labels.

  Fix: pin the invariant with a one-line comment, or panic if `names[i]` resolves to an `init` / blank slot.
  </details>

- **[missing config = unbounded waits]** [`gno.land/pkg/gnoweb/handler_http.go:131`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go#L131) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L131) — GET sets `context.WithTimeout` only when `h.Timeout > 0`; an operator who forgets `NodeRequestTimeout` gets the bare request context.
  <details><summary>details</summary>

  Without `NodeRequestTimeout` wired in `AppConfig`, GET inherits no deadline and the bounded fan-out can still hang on a sluggish chain — every fetch is under a shared cap of 8 goroutines but each waits indefinitely.

  Fix: defensive default (say 30s) inside `NewHTTPHandler` when `cfg.Timeout == 0`.
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/keeper.go:1582`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/sdk/vm/keeper.go#L1582) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper.go#L1582) — `tidJSON, _ := json.Marshal(tidStr)` discards the error. `json.Marshal` on `string` cannot fail, so correct, but `must`/`exhaustive` linters will flag it. A `// json.Marshal(string) never errors` comment would help.
- [`gno.land/pkg/gnoweb/components/state_walker.go:218-219`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_walker.go#L218-L219) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_walker.go#L218-L219) — `maxDecodeDepth=256` and `maxChildrenPerNode=500` are good defaults but `const`; a stricter-cap tenant would have to patch + rebuild. Expose as package-level `var`.
- [`gno.land/pkg/gnoweb/weburl/url.go:191-206`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/weburl/url.go#L191-L206) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/weburl/url.go#L191-L206) — hand-rolled height parser with overflow check; `strconv.ParseInt` already returns `errors.Is(err, strconv.ErrRange)` on overflow. Shaves 15 lines and removes a custom code path. Height parsing isn't hot.
- [`gno.land/pkg/gnoweb/components/state_orchestrator.go:301-311`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go#L301-L311) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go#L301-L311) — `stateObjectHref` allocates a fresh `url.Values{}` every call. With `maxInlinePreviewFetches=30` plus per-node Href construction in the post-fetch reassembly, this churns. Not a correctness issue.
- [`gno.land/pkg/gnoweb/components/view_state.go:36`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/view_state.go#L36) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/view_state.go#L36) — `KindCounts` counts only top-level nodes; deeply nested fields don't roll up. Matches the filter-tab visual model but a comment would prevent "is this a bug?" tickets.
- [`gno.land/pkg/gnoweb/components/state_walker.go:880-895`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_walker.go#L880-L895) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_walker.go#L880-L895) — `extractFuncSource` doesn't validate `Span.Pos.Line <= Span.End.Line`. `sliceLines` defends by falling through to EOF, so safe, but `if rn.Location.Span.Pos.Line > rn.Location.Span.End.Line { return nil }` makes the contract explicit.

## Missing Tests

- **[header-injection round-trip]** [`gno.land/pkg/gnoweb/handler_http_test.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http_test.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http_test.go) — lengths on `oid`/`tid` are capped at 256 but contents pass verbatim to `ABCIQueryWithOptions(ctx, qpath, []byte(oid))`.
  <details><summary>details</summary>

  A `\r\n` or `\x00` byte inside `oid` won't reach an HTTP header (the path is ABCI-internal), but a regression test asserting the byte is round-tripped or rejected would prevent future leakage if anyone refactors `query()` into an HTTP path.
  </details>

- **[overflow at one-past-MaxInt64]** [`gno.land/pkg/gnoweb/weburl/url.go:191-206`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/weburl/url.go#L191-L206) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/weburl/url.go#L191-L206) — existing test stops at exactly `MaxInt64`; no case asserts `?height=9223372036854775808` is normalised to 0/latest.

- **[actual cycle in `marshalTypeJSON`]** [`gno.land/pkg/sdk/vm/keeper_test.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/sdk/vm/keeper_test.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/sdk/vm/keeper_test.go) — `TestMarshalTypeJSON_ProducesValidJSONForControlCharNames` covers control-char names but not cycles. A `DeclaredType` whose `Base` points back at itself would pin the depth-bound's termination.

- **[cross-round dedupe under cycles]** [`gno.land/pkg/gnoweb/components/state_orchestrator.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go) — `fetched` map dedupes across rounds, but a hostile state that round-1 fetches A → B and round-2 sees C → A would re-walk A. Current test pins the no-redundant-fetch behavior for a linear graph only.

- **[length-cap at response boundary]** [`gno.land/pkg/gnoweb/handler_http.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go) — `maxPostFormBytes=64KiB` caps the body, but a 64KiB-1 path stuffed into `__gno_path` becomes `gnourl.Args` and flows into `EncodeFormURL()` → `Location` header. A length-cap test at the response-build boundary closes the gap.

## Suggestions

- [`gno.land/pkg/gnoweb/components/state_walker.go:124-158`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_walker.go#L124-L158) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_walker.go#L124-L158) — only `DecodeObjectFull` is used by the live handler; `DecodeObjectJSON` and `DecodeObjectJSONWithType` are public surface with no in-PR caller. Delete or annotate as "kept for external consumers of `components/`".
- [`gno.land/pkg/gnoweb/handler_http.go:758`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go#L758) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L758), [`:784`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go#L784) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L784), [`:791`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go#L791) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L791), [`:925`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http.go#L925) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http.go#L925) — `maxStateIDLength` check duplicated across four call sites. Extract a `validateStateID(name, value string) (int, *components.View, bool)` helper: small DRY win plus a single test surface.
- [`gno.land/pkg/gnoweb/components/state_orchestrator.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/components/state_orchestrator.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/components/state_orchestrator.go) — `maxInlinePreviewRounds=2` has no test asserting it terminates within 2 rounds for the documented "heap→ref→struct" indirection pattern. A targeted 3-level test would document the invariant and catch a future bump to 3 needing a test update.
- [`examples/gno.land/r/demo/closuretest/`](https://github.com/gnolang/gno/blob/1506642/examples/gno.land/r/demo/closuretest/) · [↗](../../../../../.worktrees/gno-review-5649/examples/gno.land/r/demo/closuretest/) — useful fixture for screenshots but lacks tests; doesn't appear in any genesis manifest I could grep. Verify it's loaded by the dev node before relying on it for demos.
- [`gno.land/pkg/gnoweb/handler_http_test.go`](https://github.com/gnolang/gno/blob/1506642/gno.land/pkg/gnoweb/handler_http_test.go) · [↗](../../../../../.worktrees/gno-review-5649/gno.land/pkg/gnoweb/handler_http_test.go) — the PR body claims "saving 1 RTT" (parallel state+doc on package view) and "max(qobject, qtype) instead of their sum" (object view). Both are correct (`wg.Add(2)`), but a microbenchmark measuring p50 wall-clock vs the sequential version would lock the win against future refactors.

## Questions for Author

- ADR-003 historically had `@gnojs/amino` as canonical decoder. With the Go decoder becoming the source of truth, what's the maintenance contract for `misc/gnojs/`? README says "best effort" — does it survive the next breaking Amino-JSON change without an explicit deprecation cycle? A months-not-days deprecation timeline would help external integrations.
- Roadmap mentions PRs N+1 (nginx + ETag + rate limit) and N+2 (in-process cache + streaming + htmx). What ETag strategy is intended for the latest-height path? `Cache-Control: max-age=1` is fine for now but a stale ETag could pin a 1s window to "no fetch needed" indefinitely under heavy traffic.
- The 8-concurrency cap is shared between object and type fetches in `fetchPreviewsConcurrent`. Was a split semaphore (4+4) considered? Type fetches are typically faster (no full traversal), so coupling them might starve previews under hostile timing.
- `extractFuncSource` reads `Location.Span.Pos.Line` / `Span.End.Line` from the gnovm `RefNode`. If a realm is upgraded (different bytecode for the same path), do those line numbers stay valid when `?height=N` pins to a pre-upgrade block? My intuition: `qfile` returns the *current* file but lines were emitted at upgrade time, so they'd silently misalign. Worth confirming.
