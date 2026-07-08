# PR [#5421](https://github.com/gnolang/gno/pull/5421): feat(gnoweb): built-in playground(2)

URL: https://github.com/gnolang/gno/pull/5421
Author: moul | Base: master | Files: 66 | +4636 -1244
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `13124c534` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5421 13124c534`

Round 2. Head advanced `339469041` → `13124c534`; the PR content changed (patch-ids differ), driven by [#5774](https://github.com/gnolang/gno/pull/5774) which moved the playground and run views into self-contained feature modules under `gno.land/pkg/gnoweb/feature/`. Two round-1 Criticals are now fixed (deflate bomb, `?fork` fan-out). The rest of the round-1 security set is still live at this sha, and each was already raised on the PR by moul and @alexiscolin.

**TL;DR:** Adds an interactive playground to gnoweb: a scratch-pad editor at `/_/play`, an expression evaluator embedded in a realm's Actions view, a one-click `?fork` that loads a package's source into the editor, a `?run` `maketx run` scratchpad, and two JSON APIs (`POST /_/api/eval`, `GET /_/api/funcs`). All Go backend plus vanilla TypeScript, no separate app.

**Verdict: REQUEST CHANGES** — the public unauthenticated eval/funcs endpoints still ship without a request-body cap, with an X-Forwarded-For-spoofable rate limiter, and with no limiter at all on `/_/api/funcs`. All three reproduce on 13124c534 and all three are already open threads (moul, @alexiscolin); a fix is in flight in [#5897](https://github.com/gnolang/gno/pull/5897) but is not in this branch. Approve once the body cap, the trusted-proxy gate, and the funcs limiter land.

## Summary

Round 1's six Criticals were all node-amplification or OOM vectors on new public endpoints. This round closes the two worst: the DEFLATE decompression bomb on `/_/play?code=…&z` now runs through `io.LimitReader` with a 1 MiB ceiling, and `?fork` now bounds itself to 8 parallel file fetches and 1 MiB total with an errgroup that cancels on overflow. The remaining three amplification gaps are unchanged: `POST /_/api/eval` decodes an unbounded JSON body, the eval limiter keys on a spoofable XFF header, and `GET /_/api/funcs` forwards a full `vm/qdoc` RPC with no limiter. The refactor into `feature/playground` and `feature/run` is clean and mirrors the existing `feature/state` module, which is also where the fix belongs: `feature/state` already carries a trusted-proxy-gated limiter (`extractIP` + `ParseTrustedProxies`) that the playground reinvents in a weaker form.

## Glossary

- **`vm/qeval`** — read-only ABCI query evaluating a Gno expression against a deployed package. Backs `/_/api/eval`.
- **`vm/qdoc`** — read-only ABCI query returning package + function documentation. Backs `/_/api/funcs`.
- **XFF** — `X-Forwarded-For` HTTP header. Honored as the rate-limit key with no trusted-proxy gate.
- **token bucket** — the per-IP rate limiter: 10-token burst, +1 token every 3s.
- **feature module** — a self-contained gnoweb sub-package (`feature/state`, `feature/playground`, `feature/run`) owning its handler, view, template, and frontend.

## Fix

The playground and run code moved out of `handler_playground.go` into `feature/playground/` and `feature/run/`. `NewHTTPHandler` constructs both ([`handler_http.go:142-153`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/handler_http.go#L142-L153) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L142-L153)); `NewRouter` mounts the two JSON APIs on the root mux ([`app.go:206-208`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/app.go#L206-L208) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/app.go#L206-L208)); `GetPackageView` dispatches `?fork` and `?run` ([`handler_http.go:455-462`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/handler_http.go#L455-L462) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http.go#L455-L462)). The deflate cap lives in `decodeCompressedCode` ([`feature/playground/handler.go:307-318`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L307-L318) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L307-L318)); the fork bounds live in `GetForkView` ([`feature/playground/handler.go:79-161`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L79-L161) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L79-L161)). The deflate cap and fork bounds are covered by committed tests (`TestDecodeCompressedCode/over-limit_payload_rejected`, `TestGetForkView/oversized_source_is_rejected`). Verified on 13124c534 beyond CI: the three open gaps still reproduce at this sha (5 MiB eval body accepted, funcs never throttled over 200 calls, rotating XFF bypasses the eval limiter; per-finding repros below). CodeMirror also landed (via [#5674](https://github.com/gnolang/gno/pull/5674)); the editor is no longer a textarea.

## Critical (must fix)

- **[public endpoint reads an unbounded request body]** [@moul](https://github.com/gnolang/gno/pull/5421#discussion_r3512098582) [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256247099) [`feature/playground/handler.go:216`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L216) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L216) — `json.NewDecoder(r.Body).Decode(&req)` has no `http.MaxBytesReader`, and neither `pkg_path` nor `expression` has a length cap.
  <details><summary>details</summary>

  `POST /_/api/eval` accepts an arbitrarily large JSON body. The handler validates only that `PkgPath` and `Expression` are non-empty ([L221](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L221) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L221)), then forwards `fmt.Sprintf("%s/%s.%s", domain, pkgPath, expression)` straight to `vm/qeval`. A 5 MiB body is read into memory in full before anything else runs; the rate limiter caps requests per second but not bytes per request. gnoweb's own POST path already wraps `r.Body` in `http.MaxBytesReader` (asserted by `TestHTTPHandler_Post_BodyTooLarge`), so the guard exists in the package and just isn't applied here.

  Verified on 13124c534: a well-formed 5,242,926-byte body returns `200`, not `413`.

  Fix: wrap `r.Body` with `http.MaxBytesReader` before decoding and reject an over-length `expression`.
  </details>

- **[rate limiter keys on a spoofable header, so it can be bypassed per request]** [@moul](https://github.com/gnolang/gno/pull/5421#discussion_r3512098587) [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256226373) [`feature/playground/ratelimit.go:88`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) — `clientIP` returns the first `X-Forwarded-For` entry unconditionally, with no trusted-proxy check on `RemoteAddr`.
  <details><summary>details</summary>

  The eval limiter buckets by `clientIP(r)`. Because `clientIP` trusts the client-supplied XFF header, an attacker rotates `X-Forwarded-For` per request and lands each in a fresh bucket, so the 10-burst / +1-per-3s budget is effectively infinite; meanwhile legitimate users behind shared NAT collapse onto one bucket. The sibling `feature/state` limiter already does this correctly: `extractIP` honors `X-Real-IP` only when `RemoteAddr` is inside a configured trusted-proxy CIDR, wired through `StateRateLimitTrustedProxies` ([`feature/state/ratelimit.go:200-213`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/state/ratelimit.go#L200-L213) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/state/ratelimit.go#L200-L213)). The playground reimplements the key extraction and drops the gate.

  Verified on 13124c534: with `burst=2` and a single TCP peer, 50 requests that rotate `X-Forwarded-For` produce 0 rate-limited responses; without the header they cap after 2.

  Fix: reuse the trusted-proxy IP extraction from `feature/state` (or gate XFF on a trusted-proxy allowlist that is empty by default).
  </details>

## Warnings (should fix)

- **[`/_/api/funcs` has no rate limiter]** [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256256566) [`feature/playground/handler.go:250`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L250) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L250) — `serveFuncs` never calls `h.limiter.allow`, though `serveEval` on the same `Handler` does.
  <details><summary>details</summary>

  `GET /_/api/funcs?path=…` forwards a full `vm/qdoc` RPC on every request with no upstream limit. It shares the `Handler` (which owns a `limiter`), but unlike `serveEval` at [L210](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L210) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L210) it omits the limiter check entirely. `qdoc` walks the package AST and serializes JSON per exported symbol, so spamming rotating package paths saturates the node. This is the same amplification the eval limiter was added to close, on the sibling endpoint with no guard.

  Verified on 13124c534: 200 back-to-back funcs calls from one IP produce 0 rate-limited responses.

  Fix: run `serveFuncs` through the same limiter (or a read-only-Doc limiter) that `serveEval` uses.
  </details>

- **[`pruneLoop` goroutine has no shutdown path]** [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256267671) [`feature/playground/ratelimit.go:40`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40) — `newRateLimiter` starts `go rl.pruneLoop()` with no context, no `Stop`, no reachable exit.
  <details><summary>details</summary>

  `pruneLoop` ([L70-83](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L70-L83) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L70-L83)) ranges over a ticker that never stops, so the goroutine runs for the process lifetime. Bounded in production because `playground.New` runs once per gnoweb instance, but every `playground.New` in a test leaks one goroutine, and the limiter has no clean-shutdown contract if it is ever constructed per-request.

  Fix: accept a `context.Context` and select on `ctx.Done()` in `pruneLoop`, or expose a `Stop()` that closes a done channel.
  </details>

- **[backend RPC failure returns 200]** [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256269388) [`feature/playground/handler.go:243`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L243) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L243) — eval ([L243](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L243) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L243)) and funcs ([L264](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L264) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L264)) return `200 {"error":…}` when the upstream node query errors.
  <details><summary>details</summary>

  A `200` on an upstream RPC failure breaks monitoring (`grep '" 5'` on access logs sees nothing), defeats reverse-proxy retry, and makes `fetch().ok` on the frontend never trip on a node error. The `200`-with-error is correct only for the "query succeeded but the Gno expression itself errored" case; a transport/node failure should be a 5xx.

  Fix: return `502` (or `504` on a context deadline) when the query fails at the transport/node level; keep `200` for a successful query that carries a Gno-level error.
  </details>

- **[a test claims to cover fork-via-`?from=` but exercises no fork path]** [`handler_http_test.go:1621`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/handler_http_test.go#L1621) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1621) — the "with fork param" case hits `/_/play?from=…` and asserts the path appears in the body, but `GetPlaygroundView` never reads `from`.
  <details><summary>details</summary>

  `GetPlaygroundView` ([`feature/playground/handler.go:44-73`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L44-L73) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L44-L73)) reads only `code` and `z`. The assertion passes because the request URL echoes into the surrounding layout, not because any fork ran, and the real fork route is `?fork` on a package page (covered separately by `TestHTTPHandler_ForkView`). The `?from=` name is also stale: no handler consumes it.

  Verified on 13124c534: `/_/play?from=ZZUNIQUEMARKER42` returns 200 with the marker echoed in the body and no `data-playground-fork-from-value` attribute, i.e. no fork occurred.

  Fix: delete the case, or repoint it at `?fork` on a package with a stub `ListFiles`/`File` and assert the concatenated source renders.
  </details>

## Nits

- [`feature/playground/handler.go:232`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L232) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L232) — the qeval string is hand-built with `fmt.Sprintf("%s/%s.%s", domain, pkgPath, expression)`. It matches how `parseQueryEvalData` splits the data string today, but if that format changes this silently builds the wrong shape. A shared builder in the client would keep the two ends in sync. Not posted: no concrete break today.
- [`feature/run/frontend/controller-run.ts:80`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L80) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L80) — `if (send && send !== "0ugnot")` only skips the exact string `0ugnot`; `0`, `0 ugnot`, `00ugnot` all slip through and add a spurious `-send` line. Not posted: cosmetic, affects only a copyable command string.
- [`feature/playground/frontend/controller-playground.ts:14`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/frontend/controller-playground.ts#L14) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/frontend/controller-playground.ts#L14) — `MAX_SHARE_URL_LENGTH = 8_000` guards only the share button. The server-side ceiling that actually protects the node is `maxDecompressedCodeSize` (1 MiB); the two are unrelated and the client value is not a security limit. Not posted: informational.
- [`feature/playground/frontend/controller-playground.ts:240`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/frontend/controller-playground.ts#L240) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/frontend/controller-playground.ts#L240) — `addFile()` uses a native `prompt()`. Already raised by [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256275783); the thread agreed to keep it for now and rework during the design review.
- [`feature/playground/handler.go:54`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler.go#L54) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L54) — a decode failure (malformed or over-limit deflate) is swallowed: `decodeCompressedCode` returns `false` and the view silently falls back to the default code with no log. Now that the size cap makes over-limit a reachable rejection, a Warn log would make abuse visible. Not posted: low value once the cap is in place.

## Missing Tests

- **[no committed test asserts the eval body cap]** [`feature/playground/handler_test.go:88`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L88) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L88) — once `MaxBytesReader` lands, add a case that POSTs an over-cap body and asserts `413`. The equivalent `TestHTTPHandler_Post_BodyTooLarge` exists for the main POST path; mirror it here.
- **[no committed test asserts the funcs limiter]** [`feature/playground/handler_test.go:453`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L453) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L453) — `TestRateLimiter` covers `serveEval` only. After the funcs limiter lands, add a funcs equivalent.
- **[no committed test covers XFF spoofing]** [`feature/playground/handler_test.go:453`](https://github.com/gnolang/gno/blob/13124c534/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L453) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L453) — `TestRateLimiter` sets only `RemoteAddr`. After the trusted-proxy gate lands, add a case that rotates `X-Forwarded-For` from an untrusted peer and asserts the limiter still caps, mirroring `feature/state`'s `TestExtractIPTrustedProxy`.

## Suggestions

- Now that the playground is a feature module, the eval/funcs limiter and the (missing) body cap are the natural place to reuse `feature/state`'s `rateLimiter` + `extractIP` + `ParseTrustedProxies` rather than maintaining a second, weaker limiter. One limiter implementation across features removes the XFF divergence at the source.

## Open questions

- The Run button qevals a fixed `${domain}/r/playground_preview` package rather than the editor contents (confirmed by moul in [the thread on controller-playground.ts:282](https://github.com/gnolang/gno/pull/5421#discussion_r3512098590)). That is a known limitation of this iteration, not a defect; noting it so the eval-vs-editor gap is on record. Not posted: already the author's own open thread.
