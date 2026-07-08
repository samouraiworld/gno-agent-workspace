# PR [#5258](https://github.com/gnolang/gno/pull/5258): fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config

URL: https://github.com/gnolang/gno/pull/5258
Author: davd-gzl | Base: master | Files: 10 | +122 -11
Reviewed by: davd-gzl (self-review, scheduled sweep) | Model: claude-opus-4-8 | Commit: `c8b2138a8` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5258 c8b2138a8`

Round 2. Head advanced `554a7546` → `c8b2138a8`; patch-ids differ, so real PR content changed. The change since round 1 is a master-merge conflict resolution that left the WS-origin code identical but accidentally committed a 40 MB compiled binary. Findings on the origin logic carry forward verbatim (re-verified on the new head); the binary is a new blocking finding. Verdict moves APPROVE → REQUEST CHANGES because of the binary.

## Summary

The WS upgrader at [`handlers.go:970-988`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers.go#L970-L988) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L970-L988) and gnodev's emitter at [`server.go:24-33`](https://github.com/gnolang/gno/blob/c8b2138a8/contribs/gnodev/pkg/emitter/server.go#L24-L33) · [↗](../../../../../.worktrees/gno-review-5258/contribs/gnodev/pkg/emitter/server.go#L24-L33) both returned `true` from `CheckOrigin` unconditionally, so any web page could open a cross-site WS to a user's node and call the full RPC route map (CSWSH). The fix plumbs `CORSAllowedOrigins` into `NewWebsocketManager`, builds an `rs/cors` validator when the list is non-empty, and leaves `CheckOrigin` nil otherwise so gorilla's same-origin check runs; gnodev's emitter drops to the same nil default. The origin logic is unchanged from round 1 and re-verified correct here. The blocker this round is unrelated: the June 30 conflict-resolution merge committed a 40 MB compiled `gnokeykc` binary at [`contribs/gnokeykc/gnokeykc`](https://github.com/gnolang/gno/blob/c8b2138a8/contribs/gnokeykc/gnokeykc) · [↗](../../../../../.worktrees/gno-review-5258/contribs/gnokeykc/gnokeykc).

## Examples

Behavior of `NewWebsocketManager`'s upgrade check, verified on `c8b2138a8` (full matrix run in the Warning repro):

| `CORSAllowedOrigins` | Origin header | Result |
|----------------------|---------------|--------|
| `["*"]` (default) | any / absent | 101 upgrade |
| `["http://good.com"]` | `http://good.com` | 101 upgrade |
| `["http://good.com"]` | absent | 403 reject |
| `[]` (CORS off → gorilla) | absent | 101 upgrade |
| `[]` (CORS off → gorilla) | `http://evil.com` | 403 reject |

## Glossary

- **CSWSH** — Cross-Site WebSocket Hijacking; a malicious origin opens a WS to a victim's authenticated endpoint.
- **`CheckOrigin`** — gorilla/websocket hook called during upgrade; nil falls back to same-origin check.
- **`IsCorsEnabled()`** — `len(CORSAllowedOrigins) != 0` ([`config.go:143-144`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/config/config.go#L143-L144) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/config/config.go#L143-L144)).
- **`rs/cors.OriginAllowed`** — returns true if `Origin` matches the configured list; an empty list defaults to "all allowed" inside the lib, which the PR avoids by only building the validator when `len(allowedOrigins) > 0`.

## Fix

Before: WS upgrade allowed every Origin; HTTP CORS already validated against `CORSAllowedOrigins`. After: WS upgrade uses the same list when CORS is enabled and falls back to gorilla's same-origin check when not. The plumbing in [`node.go:804-810`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/node/node.go#L804-L810) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/node/node.go#L804-L810) only forwards the list when `IsCorsEnabled()` is true, so the disabled-CORS path produces a nil `CheckOrigin` rather than an empty slice that `rs/cors` would treat as "all allowed". `NewWebsocketManager` gained a required `allowedOrigins []string` parameter; the single in-tree caller, the test helper, and the package doc example are all updated. Verified on `c8b2138a8`: the gnodev emitter's nil default rejects a cross-origin dial (403) and allows same-origin and no-Origin dials (101); no committed test covers the emitter's origin behavior.

## Critical (must fix)

None.

## Warnings (should fix)

- **[compiled binary committed to the tree]** [`contribs/gnokeykc/gnokeykc`](https://github.com/gnolang/gno/blob/c8b2138a8/contribs/gnokeykc/gnokeykc) · [↗](../../../../../.worktrees/gno-review-5258/contribs/gnokeykc/gnokeykc) — the merge commit `c8b2138a8` adds a 40 MB compiled ELF executable to `contribs/gnokeykc/`, unrelated to the WS-origin fix.
  <details><summary>details</summary>

  `git cat-file -s c8b2138a8:contribs/gnokeykc/gnokeykc` is 40367288 bytes; `file` reports `ELF 64-bit LSB executable, x86-64 ... not stripped`. It is absent on `origin/master` and absent at round 1's `554a7546`, so it was introduced by the June 30 conflict-resolution merge, most likely a `make build` artifact staged by accident. Nothing gitignores `contribs/gnokeykc/gnokeykc`. This bloats the repository permanently once merged and is the reason the verdict is REQUEST CHANGES. Fix: `git rm --cached contribs/gnokeykc/gnokeykc`, drop it from the branch, and add the binary name to a gitignore so a future build does not re-stage it.
  </details>

- **[restricted CORS list blocks non-browser WS clients]** [`handlers.go:970-988`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers.go#L970-L988) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L970-L988) — an operator who sets `CORSAllowedOrigins` to a specific list (`["http://my-frontend.com"]`) to lock CSWSH down also makes the in-tree Go WS client, which dials with no Origin header, get a 403.
  <details><summary>details</summary>

  With a non-`*`, non-empty list, `rs/cors.OriginAllowed` returns false for an absent `Origin` because `""` matches no entry, so the upgrade is rejected with 403. The in-tree WS client at [`client.go:42`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/client/ws/client.go#L42) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/client/ws/client.go#L42) dials `websocket.DefaultDialer.Dial(rpcURL, nil)` with no Origin, and it backs the event-subscription client at [`tm2/pkg/bft/rpc/client/client.go:13`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/client/client.go#L13) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/client/client.go#L13), so Go subscribers and any curl/monitoring client break under a narrowed list. The default config ships `CORSAllowedOrigins: ["*"]`, which allows no-Origin, so default deployments are unaffected; the regression is scoped to operators who narrow the list. Fix options: (a) treat empty Origin as allowed in the WS path, which mirrors gorilla's default and the CSWSH model since a victim browser cannot omit Origin; (b) document the no-Origin reject next to `cors_allowed_origins` so operators keep `*` or add their own client identifier; (c) split a `CORSAllowedOriginsWS` config. (a) is cheapest: wrap `corsValidator.OriginAllowed` in a closure returning true when `r.Header.Get("Origin") == ""`.
  </details>

## Nits

- [`handlers.go:966-969`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers.go#L966-L969) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L966-L969) — the doc comment says only `Use []string{"*"} to allow all origins`; it could also note that nil/`[]` falls back to gorilla's same-origin check and that a restricted list rejects no-Origin requests (ties to the second Warning).
- [`doc.go:69`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/doc.go#L69) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/doc.go#L69) — the example uses `[]string{"*"}`; harmless, but it teaches the most permissive setting; a restricted example or a short comment would read better.
- [`handlers.go:973-977`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers.go#L973-L977) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers.go#L973-L977) — `cors.New` is heavy for what is effectively an origin matcher; a small in-file matcher would keep `rs/cors` out of the WS lib's module graph. Parity with the HTTP CORS middleware (wildcard semantics, case folding) is the trade-off, so keeping `rs/cors` is reasonable; noting the surface only. No change requested.

## Missing Tests

- **[no-Origin reject under a restricted list is not pinned]** [`handlers_test.go:471`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L471) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L471) — `TestWebsocketManagerCheckOrigin` has a `no origin header is allowed` case, but it uses `allowedOrigins: []string{}` (the gorilla fallback path at [`handlers_test.go:487`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L487) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/lib/server/handlers_test.go#L487)). No case exercises a non-empty list with a missing Origin header, which is exactly the behavior change in the second Warning (today: 403).
  <details><summary>details</summary>

  Without this row a future "be nice to native clients" patch could flip the no-Origin behavior invisibly. Add a row that pins whichever decision the Warning lands on: `{name: "no origin header rejected when allowedOrigins set", allowedOrigins: []string{"http://good.com"}, origin: "", expectAllowed: false}` if the 403 stays, or `expectAllowed: true` if the fix admits no-Origin. Verified on `c8b2138a8`: the current behavior is 403.
  </details>

## Suggestions

- [`config.go:142`](https://github.com/gnolang/gno/blob/c8b2138a8/tm2/pkg/bft/rpc/config/config.go#L142) · [↗](../../../../../.worktrees/gno-review-5258/tm2/pkg/bft/rpc/config/config.go#L142) — the `XXX review.` comment on `IsCorsEnabled` could be resolved in a follow-up; the predicate is now load-bearing for two call sites (HTTP CORS and WS origin), not just HTTP.

## Open questions

- Is the no-Origin → 403 under a restricted list intentional? If yes, it belongs in the `cors_allowed_origins` doc; if not, the one-line closure fix in the second Warning admits no-Origin without weakening the CSWSH guard (a victim browser cannot omit Origin). Not posted separately; it is the decision the second Warning already asks for.
- The invariant catalog does not apply: the diff is Go-side networking and module plumbing, no GnoVM, stdlib, or `.gno` change.
- PR #4954 (RPC server overhaul), noted in round 1 as the code path this fix overlapped, is now CLOSED, so the double-landing concern is moot. Not posted.
