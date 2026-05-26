# PR #5258: fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config

URL: https://github.com/gnolang/gno/pull/5258
Author: davd-gzl | Base: master | Files: 10 | +123 -10
Reviewed by: davd-gzl (self-review, scheduled sweep) | Model: claude-opus-4-7

Verdict: APPROVE (with one compat note) — wires `CORSAllowedOrigins` into WS upgrade via `rs/cors`, mirrors HTTP CORS, gorilla default same-origin check kicks in when CORS is disabled; one behavior change worth flagging: a restricted (non-`["*"]`, non-empty) `CORSAllowedOrigins` list now rejects no-Origin requests with 403, which blocks the in-tree Go WS client that sends no Origin header.

## Summary

The WS upgrader at [`handlers.go:833`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L833) and gnodev's emitter at [`server.go:24`](../../../../../.worktrees/gno-review-5258/contribs/gnodev/pkg/emitter/server.go#L24) both returned `true` from `CheckOrigin` unconditionally, so any web page could open a cross-site WS to a user's node and call the full RPC route map (CSWSH). The fix plumbs `CORSAllowedOrigins` into `NewWebsocketManager`, builds an `rs/cors` validator when the list is non-empty (matching the existing HTTP CORS middleware at [`node.go:768-774`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/node/node.go#L768-L774)), and leaves `CheckOrigin` nil otherwise so gorilla's `checkSameOrigin` runs. In gnodev's standalone emitter, the unrestricted callback is replaced with the same nil default.

## Glossary

- **CSWSH** — Cross-Site WebSocket Hijacking; a malicious origin opens a WS to a victim's authenticated endpoint.
- **`CheckOrigin`** — gorilla/websocket hook called during upgrade; nil falls back to same-origin check.
- **`IsCorsEnabled()`** — `len(CORSAllowedOrigins) != 0` ([`config.go:143-144`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/config/config.go#L143-L144)).
- **`rs/cors.OriginAllowed`** — returns true if `Origin` matches the configured list; empty list defaults to "all allowed" inside the lib, but the PR guards against that with `len(allowedOrigins) > 0`.

## Fix

Before: WS upgrade allowed every Origin; HTTP CORS already validated against `CORSAllowedOrigins`. After: WS upgrade uses the same `CORSAllowedOrigins` config when CORS is enabled, and falls back to `gorilla/websocket`'s same-origin check when not. The plumbing in [`node.go:740-746`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/node/node.go#L740-L746) only forwards the list when `IsCorsEnabled()` is true, so the disabled-CORS path produces a nil `CheckOrigin` rather than passing an empty slice that `rs/cors` would silently treat as "all allowed". `NewWebsocketManager` gained a required `allowedOrigins []string` parameter; the single in-tree caller ([`node.go:745`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/node/node.go#L745)), the test helper, and the package doc example are all updated.

## Critical (must fix)

None.

## Warnings (should fix)

- **[restricted CORS list blocks non-browser WS clients]** [`handlers.go:833-846`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L833-L846) — operator sets `CORSAllowedOrigins: ["http://my-frontend.com"]` to lock CSWSH down; the in-tree WS client ([`client.go:42`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/client/ws/client.go#L42)) dials with no Origin header, gets 403.
  <details><summary>details</summary>

  Probe (build a local test): with `allowedOrigins=["http://good.com"]` and a dial that sets no `Origin` header, `rs/cors.OriginAllowed` returns false because `""` does not match any entry, and the upgrade is rejected with 403. The bundled WS client `websocket.DefaultDialer.Dial(rpcURL, nil)` does exactly that — sends no Origin. Same applies to any other non-browser client (curl, gnoclient extensions, monitoring scripts) that connects to the WS endpoint when an operator has narrowed CORS for browsers. Fix options: (a) treat empty Origin as allowed in the WS path (mirrors gorilla's default and the CSWSH model — browsers always send Origin, attackers cannot strip it from a victim browser); (b) document this in the `cors_allowed_origins` config comment so operators know to keep `*` or add their own client identifier; (c) add a `CORSAllowedOriginsWS` config split. (a) is the cheapest and aligns with the gorilla intent — no browser CSWSH attacker can omit Origin. Code: wrap `corsValidator.OriginAllowed` in a closure that returns true when `r.Header.Get("Origin") == ""`.
  </details>

## Nits

- [`handlers.go:832`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L832) — doc comment says `Use []string{"*"} to allow all origins`; could also note `nil`/`[]` falls back to gorilla same-origin and mention the no-Origin reject behavior of restricted lists (ties to the warning above).
- [`doc.go:69`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/doc.go#L69) — example uses `[]string{"*"}`; harmless but it teaches the most permissive setting; consider showing a restricted example or a short comment.
- [`handlers.go:837-840`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L837-L840) — `cors.New` is heavy for what is effectively an origin matcher; a small in-file matcher would avoid pulling `rs/cors` into the WS lib's tidied modules. Trade-off is parity with HTTP CORS (wildcard semantics, case folding) — keeping `rs/cors` is reasonable, just noting the surface.
- ADR not included; the change is small and surgical so it likely falls under the "trivial bug fix" exception in `gno/AGENTS.md`. Maintainer's call.

## Missing Tests

- **[no-Origin reject under restricted list is undocumented]** [`handlers_test.go:296-300`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L296-L300) — the existing `no_origin_header_is_allowed` case uses `allowedOrigins: []string{}`, exercising the gorilla fallback. A case with non-empty `allowedOrigins` and missing Origin header would lock in the behavior change called out in the warning (today: 403). Without it, a future "be nice to native clients" patch could quietly flip behavior again.
  <details><summary>details</summary>

  Add a table row: `{name: "no origin header rejected when allowedOrigins set", allowedOrigins: []string{"http://good.com"}, origin: "", expectAllowed: false}`. Verified empirically with a throwaway test — current behavior is 403.
  </details>

## Suggestions

- Consider deprecating the `XXX review.` comment at [`config.go:142`](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/config/config.go#L142) in a follow-up; the semantics of `IsCorsEnabled` are now load-bearing for two call sites, not just HTTP.

## Questions for Author

- Is the no-Origin → 403 behavior under a restricted list intentional? If yes, worth documenting near `cors_allowed_origins`. If not, the one-line fix is to short-circuit `OriginAllowed` when the header is absent (matches gorilla's default rationale: browser attackers cannot omit Origin).
- PR #4954 (RPC server overhaul, still open) rewrites this code path entirely. Is the plan to land this fix first as a backportable security patch and let #4954 absorb it on rebase? The diff is small enough that double-landing is cheap.
