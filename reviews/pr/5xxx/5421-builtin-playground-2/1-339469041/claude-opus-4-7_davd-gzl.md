# PR #5421: feat(gnoweb): built-in playground(2)

URL: https://github.com/gnolang/gno/pull/5421
Author: moul | Base: master | Files: 46 | +3854 -1593
Reviewed by: davd-gzl | Model: claude-opus-4 | Commit: `339469041` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5421 339469041`

**Verdict: REQUEST CHANGES** — load-bearing security gaps on public, unauthenticated endpoints: deflate bomb on `/_/play?code=…&z`, no body cap on `/_/api/eval`, XFF-spoofable rate limiter, no limiter at all on `/_/api/funcs`, unbounded serial RPC fan-out on `?fork`, and a goroutine-leaking `pruneLoop`. All six are concrete amplification or OOM vectors and most were already flagged by @alexiscolin in the round-1 thread without being addressed. Approve once those land; the feature scope and UX are solid.

## Summary

Ships a Go-native, in-gnoweb playground replacing the separate `gnostudio/studio` app. Adds three views (`/_/play` scratch pad, `?fork` source-fork, `?run` `maketx run` scratchpad), one inline evaluator embedded in the Actions view (`?eval` per the description, but now part of `action.html`), and two JSON APIs (`POST /_/api/eval`, `GET /_/api/funcs`). Backend is ~480 LoC Go ([handler_playground.go](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L1-L256) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L1-L256), additions to [handler_http.go](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http.go#L782-L863) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L782-L863)), frontend is ~860 LoC TS, plus 611 lines of CSS bolted onto `06-blocks.css`. The amplification surface is the worrying part: every new endpoint forwards to the node, and four of them have no body cap, no concurrency cap, or no rate limit.

## Glossary

- **`vm/qeval`** — read-only ABCI query that evaluates a Gno expression against a deployed package. Backend for `/_/api/eval`.
- **`vm/qdoc`** — read-only ABCI query returning package + function documentation. Backend for `/_/api/funcs`.
- **`ListFiles` / `File`** — `ClientAdapter` methods backed by `vm/qfile`. One RPC call per file.
- **`ForkView`** — `?fork` handler: loads every `.gno` file of a package over RPC and concatenates them into the playground textarea.
- **`pruneLoop`** — background goroutine on the per-IP rate limiter that GCs stale buckets every minute.
- **XFF** — `X-Forwarded-For` HTTP header. Honored without any trusted-proxy gate.

## Fix

Three new GET views (`/_/play`, `?fork`, `?run`) and two JSON APIs (`/_/api/eval`, `/_/api/funcs`) are wired into the root mux in [`app.go:181-182`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/app.go#L181-L182) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/app.go#L181-L182). The eval handler adds a per-IP token bucket (10 burst, ~20 req/min) keyed on `clientIP()`. The playground UI is a CodeMirror editor with multi-file tabs, share-via-URL (base64 + optional deflate), and TAR export. A 125-line ADR ([`adr/pr5421_integrated_playground.md`](https://github.com/gnolang/gno/blob/339469041/gno.land/adr/pr5421_integrated_playground.md) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/adr/pr5421_integrated_playground.md)) documents the architecture but is stale on five counts (see Nits). CI is green; merge is blocked only on codeowner approvals.

## Critical (must fix)

- **[deflate bomb — server OOM via shareable URL]** [`handler_http.go:792-802`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http.go#L792-L802) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L792-L802) — `io.ReadAll(flate.NewReader(...))` with no size cap; ~1029x amplification confirmed.
  <details><summary>details</summary>

  **Shape:** anyone can craft `/_/play?code=<base64>&z` where the base64 decodes to a flate-compressed stream. The server pipes that into `flate.NewReader` and `io.ReadAll` with no `io.LimitReader` wrapper. The client-side `MAX_SHARE_URL_LENGTH = 8_000` in [`controller-playground.ts:14`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L14) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L14) only guards the share button; an attacker hand-crafts the URL.

  **Mechanism:** flate of repeated zeros compresses ~1000x. I measured 100 KB → ~100 MB and 1 MB → 1 GB in a standalone Go program (`compress/flate` + `BestCompression`). One concurrent request per gnoweb worker at 1 MB-in / 1 GB-out is enough to OOM a default-sized node; cached at the edge proxy a single crafted URL serves the payload to every requesting worker.

  **What you see:** `gnoweb` heap climbs proportionally to compressed-payload-size × concurrent-requests; OOM-kill on small instances. No log entry — the existing code swallows both `io.ReadAll` errors at L794 and `r.Close()` errors at L798.

  **Fix:** wrap with `io.LimitReader(r, maxPlaygroundCodeBytes+1)` and reject when the readback exceeds the cap; @alexiscolin's [round-1 suggestion](https://github.com/gnolang/gno/pull/5421#discussion_r…) at this exact line proposes `256 KiB` which matches the share-URL ceiling intent. Log the swallowed errors at Warn.
  </details>

- **[/_/api/eval body unbounded — RPC amplification]** [`handler_playground.go:168-172`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L168-L172) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L168-L172) — `json.NewDecoder(r.Body).Decode(&req)` with no `http.MaxBytesReader`; no length cap on `req.Expression`.
  <details><summary>details</summary>

  **Shape:** `POST /_/api/eval` accepts arbitrarily large JSON bodies. The handler validates only that `PkgPath` and `Expression` are non-empty (L174), then sends `fmt.Sprintf("%s/%s.%s", domain, pkgPath, expression)` straight to `vm/qeval` in [L185-190](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L185-L190) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L185-L190).

  **Mechanism:** the rate limiter caps requests/sec but not bytes/req. A 10 MB expression per request × 10 burst = 100 MB forwarded to the node per IP per 3s window, and an attacker who spoofs XFF (see next finding) gets unlimited concurrency. Backend `vm/qeval` then has to parse a 10 MB Gno expression. Same vector @alexiscolin called out from their State Explorer refactor.

  **Fix:** wrap with `r.Body = http.MaxBytesReader(w, r.Body, 64<<10)` before decoding; add an explicit `len(req.Expression) > someCap` check.
  </details>

- **[XFF trusted blindly — rate limiter bypass]** [`handler_playground.go:142-155`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L142-L155) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L142-L155) — any client can set `X-Forwarded-For: 1.2.3.4` rotated per request and skip the limiter entirely.
  <details><summary>details</summary>

  **Shape:** `clientIP(r)` returns the first XFF entry unconditionally, with no check that `r.RemoteAddr` is in a trusted-proxy CIDR. The limiter buckets by IP. Rotate XFF, get a fresh bucket every request.

  **Mechanism:** the 10-burst / 20-rpm budget is effectively infinite for any motivated attacker, while collapsing onto a single bucket for legitimate users behind shared NAT (corp, mobile carrier). The rate limiter does nothing security-wise; it only inconveniences honest users.

  **Fix:** gate XFF on a `trustedProxies []*net.IPNet` allowlist (empty default = trust nothing → use `RemoteAddr`). @alexiscolin pasted a one-function fix on the same thread; it's load-bearing for the rate limiter to mean anything.
  </details>

- **[/_/api/funcs has no rate limiter — sibling amplification]** [`handler_playground.go:134-140`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L134-L140) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L134-L140) — `handlerPlaygroundFuncs` builds `playgroundAPIHandler{limiter: nil}` and calls `Client.Doc()` (full `vm/qdoc` RPC) on every request.
  <details><summary>details</summary>

  **Shape:** same `playgroundAPIHandler` struct, same RPC backend, same JS controllers calling it on every package switch — but no limiter. The eval handler's `if h.limiter != nil` guard at [L163](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L163) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L163) silently noops here.

  **Mechanism:** `vm/qdoc` walks the package AST and serializes JSON for every exported symbol. Spamming `GET /_/api/funcs?path=…` with rotating package paths saturates the node with no upstream limit. The amplification vector the eval limiter was meant to close, on a sibling endpoint with no fix.

  **Fix:** share the same limiter (or a slightly larger one for read-only Doc) between both handlers; pass `limiter` into `handlerPlaygroundFuncs` and drop the `!= nil` ambiguity.
  </details>

- **[`?fork` serial RPC fan-out — no caps, no timeout]** [`handler_http.go:817-853`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http.go#L817-L853) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L817-L853) — `ListFiles` + per-file `Client.File()` loop with no file-count cap, no byte cap, no request timeout.
  <details><summary>details</summary>

  **Shape:** any `?fork` request triggers `N+1` serial RPCs to the node (one `ListFiles`, then one `File` per `.gno` file). A package with 100 files = 100 sequential RPC round-trips per request, multi-MB of concatenated text into a single in-memory `strings.Builder`. No rate limit anywhere on this endpoint.

  **Mechanism:** unauthenticated GET, no limiter, no cap → straightforward amplification vector and a latency cliff for legitimate forks of large packages. `r/sys/users` or `/u/...` packages are realistic candidates; nothing stops a package from growing to 1000+ files.

  **Fix:** cap file count (e.g. `len(files) > 50` → return error), cap total bytes (`allCode.Len() > 1<<20` → break), add a per-request deadline, and route this through a rate limiter the same way `/_/api/eval` does.
  </details>

- **[`pruneLoop` leaks — no shutdown, no context]** [`handler_playground.go:65-78`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L65-L78) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L65-L78) — goroutine started in `newRateLimiter` runs forever; no `ctx`, no reachable `Stop()`, no way to drain.
  <details><summary>details</summary>

  **Shape:** `go rl.pruneLoop()` fires once per limiter creation. The ticker stops only when the goroutine exits — which it never does, because the only exit path is a closed `ticker.C` and nobody closes it.

  **What you see:** tests that construct a fresh limiter leak one goroutine each (the test file constructs one in `TestRateLimiter`); production gnoweb has no clean shutdown path for it. Currently bounded because `newRateLimiter` is called twice at startup (once for eval), but the pattern is wrong and will multiply if anyone calls it per-request.

  **Fix:** accept a `context.Context` in `newRateLimiter`, select on `ctx.Done()` in `pruneLoop`, plumb gnoweb's lifecycle context down. Or take a `done chan struct{}` and document the cleanup contract.
  </details>

## Warnings (should fix)

- **[200 OK on backend RPC failure breaks monitoring]** [`handler_playground.go:196,217`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L196) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L196) — eval and funcs return `200 {"error":"..."}` for upstream node errors.
  <details><summary>details</summary>

  Already flagged by @alexiscolin on both lines. HTTP-semantics-wise these are 5xx (or 502/504 specifically); operationally, returning 200 breaks `fetch().ok` on the frontend (controller-action-eval.ts:78 checks `response.ok` and never trips), defeats reverse-proxy retry logic, and makes `grep '" 5'` on access logs useless. The frontend has to dig into the JSON body to find out things failed.

  **Fix:** return `502 Bad Gateway` (or `504` for context-deadline-exceeded) when the upstream RPC errors; keep `200` only for the "evaluation succeeded but produced a Gno error" path, if the two can be told apart from `err`.
  </details>

- **[deflate errors swallowed]** [`handler_http.go:794,798`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http.go#L794-L798) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L794-L798) — `io.ReadAll` error discarded; `r.Close()` error discarded.
  <details><summary>details</summary>

  Two `err` values dropped on the floor. If decompression fails mid-stream (truncated payload, malformed deflate, oversize after the cap is added), the user sees an empty editor and the operator sees nothing. Log at Warn with the request URL hash so abuse becomes visible.
  </details>

- **[ADR is stale on five counts]** [`adr/pr5421_integrated_playground.md`](https://github.com/gnolang/gno/blob/339469041/gno.land/adr/pr5421_integrated_playground.md) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/adr/pr5421_integrated_playground.md) — describes the round-1 state, not what shipped.
  <details><summary>details</summary>

  The ADR says: (a) editor is `<textarea>` (L28: CodeMirror landed via #5610), (b) `?eval` adds a tab (L41: it's now embedded in `action.html`, not a separate route), (c) "No rate-limiting or sandboxing on /_/api/eval" (L111-113: rate limiter was added), (d) "Rate limiting / abuse prevention on eval API" listed under "Not Yet Implemented" (L124: contradicts what's actually shipped), (e) the PR description claims "Eval and Fork nav links added to header" (only Fork and Run are in `layout_header.go:91-95`).

  **Fix:** sweep the ADR before merge — it's the file future readers will trust to explain why the feature looks the way it does. The doc-debt is small but mounts up.
  </details>

- **[`?from=` test passes by accident]** [`handler_http_test.go:1500`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http_test.go#L1500) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1500) — claims to test fork-via-`?from=` but `GetPlaygroundView` never reads `from`; the test passes because the URL itself echoes into the page header/breadcrumb.
  <details><summary>details</summary>

  `GetPlaygroundView` ([`handler_http.go:782-815`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http.go#L782-L815) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L782-L815)) reads `code` and `z` only. The test asserts the response body contains `gno.land/r/demo/foo` — true, because the request URL renders into the page somewhere, not because any fork logic ran. Either implement the documented behavior or delete the test; right now it's a false-positive that will hide a regression when someone adds real `?from=` handling.
  </details>

- **[Eval rate limiter is the worst of both worlds]** [`handler_playground.go:128`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L128) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L128) — 10 burst / +1 every 3s ≈ 20 req/min. Too low for legitimate dev iteration, trivially bypassed via XFF.
  <details><summary>details</summary>

  Without the XFF gate (above), the limiter discriminates against honest users behind NAT while letting attackers walk around it. A developer hammering `?eval` against their own realm will hit the cap inside a minute. Either gate XFF and raise the budget to something developer-friendly (e.g. 60 burst / +1/s) or drop the rate limiter entirely and rely on the body cap + RPC-side limits.
  </details>

## Nits

- [`controller-run.ts:80`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/frontend/js/controller-run.ts#L80) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-run.ts#L80) — `if (send && send !== "0ugnot")` is fragile (does not skip `00ugnot`, `0 ugnot`, etc.). Use a coin parser or skip the line when amount is `0`.
- [`controller-playground.ts:14`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L14) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L14) — `MAX_SHARE_URL_LENGTH = 8_000` is client-only. Document that it's a UX guardrail, not a security limit.
- [`controller-playground.ts:233`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L233) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L233) — `prompt()` for "add file" is fine as a stopgap (@alexiscolin and @jeronimoalbi agreed in thread); leaving the marker so it doesn't ship to mainnet.
- [`layout_header.go:96`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/components/layout_header.go#L96) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/components/layout_header.go#L96) — `default:` returns all five Dev links; the Realm view shows Fork + Run even on realms that have no `.gno` source to fork. Consider hiding Fork on `?fork`-empty packages.
- [`handler_playground.go:185`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L185) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L185) — `fmt.Sprintf("%s/%s.%s", domain, pkgPath, expression)` works because `parseQueryEvalData` ([`sdk/vm/handler.go:246-259`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/sdk/vm/handler.go#L246-L259) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/sdk/vm/handler.go#L246-L259)) splits on first `.` after first `/`. If anyone changes the qeval format, this string concat silently rebuilds the wrong shape. A struct + helper in `client.go` would be safer.
- [`view_run.go:9`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/components/view_run.go#L9) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/components/view_run.go#L9) — `RunData.Remote` comment says `"https://rpc.gno.land:443"` but the value passed in is `cfg.RemoteHelp` which is `"127.0.0.1:26657"` by default. Comment misleading.

## Missing Tests

- **[no test exercises the deflate path with oversize input]** [`handler_http_test.go:1499`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http_test.go#L1499) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1499) — the `with compressed encoded code` case uses a tiny payload. Once the size cap lands, add a case with a 1 MB compressed-to-1 GB payload and assert the handler returns the cap message, not OOM.
- **[no test for XFF spoofing]** [`handler_playground_test.go:171`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground_test.go#L171) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground_test.go#L171) — `TestRateLimiter` only sets `RemoteAddr`. Add a case that sets `X-Forwarded-For: 1.2.3.<i>` per request and asserts the limiter still caps (post-fix).
- **[no test for oversize eval body]** — add `TestHandlerPlaygroundEval_BodyTooLarge` once `MaxBytesReader` lands.
- **[no test for `?fork` file-count cap]** [`handler_http_test.go:1516`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_http_test.go#L1516) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1516) — the existing fork test uses two files. Add a stub `ListFiles` returning 100 entries and assert the handler short-circuits.

## Suggestions

- [`app.go:181-182`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/app.go#L181-L182) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/app.go#L181-L182) — @alexiscolin's [feature-module proposal](https://github.com/gnolang/gno/pull/5421#discussion_r…) (`pkg/gnoweb/feature/playground/`) is the right shape for the next iteration. The two `mux.Handle` lines wiring API endpoints directly into the root mux are exactly the pattern that's awkward to extend. Worth landing this PR first, then doing the refactor as a follow-up so playground becomes the framework's flagship user.
- [`handler_playground.go:120-140`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/handler_playground.go#L120-L140) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_playground.go#L120-L140) — the two constructors take `(logger, cli, …)` separately; bundling into a `PlaygroundConfig` struct would scale better when the limiter, body cap, deadline, and trusted-proxies list all need plumbing.
- [`controller-playground.ts:362-377`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L362-L377) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L362-L377) — `downloadFiles()` builds the TAR in the browser. For consistency with `?fork`'s server-side concat, consider a `GET /_/api/pack?path=…` endpoint instead; one source of truth for file enumeration and easier to apply the file-count/byte caps you'll need anyway.

## Questions for Author

- Was the rate limiter intentionally global (per gnoweb instance) rather than per-endpoint? A single bucket across `/_/api/eval` requests from one IP means quick-call buttons in the UI eat the same budget as the manual expression field; a tab-switch sequence on a realm with five buttons could throttle the user out.
- The `playground_preview` pkg_path in [`controller-playground.ts:276`](https://github.com/gnolang/gno/blob/339469041/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L276) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-playground.ts#L276) — does that resolve to a real on-chain package, or is it a placeholder expected to 404 (so the `catch` branch always runs for scratch-pad)? If the latter, every "Run" button click costs one RPC round-trip just to fail; worth documenting.
- Is there a plan to surface the eval rate-limit budget to the user (response header, UI counter)? The current `429 {"error":"rate limit exceeded"}` gives no recovery hint.
