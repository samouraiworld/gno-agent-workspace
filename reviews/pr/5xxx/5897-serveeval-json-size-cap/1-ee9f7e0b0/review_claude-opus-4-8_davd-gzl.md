# PR [#5897](https://github.com/gnolang/gno/pull/5897): fix(gnoweb): In serveEval, use MaxBytesReader to cap the JSON size

URL: https://github.com/gnolang/gno/pull/5897
Author: jefft0 | Base: playground2 | Files: 1 | +17 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: ee9f7e0b0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5897 ee9f7e0b0`

**TL;DR:** The playground eval endpoint accepts a POST JSON body and forwards its `pkg_path` and `expression` fields to an RPC node. This PR caps the request body at 1 MiB and rejects a `pkg_path` over 1024 bytes or an `expression` over 64 KiB, so an oversized payload is rejected up front instead of being buffered and forwarded.

**Verdict: APPROVE** — correct hardening, verified live; the only gap is that none of the three new caps has a test, though the adjacent table tests every other eval case.

## Summary
`serveEval` previously read the full request body with no size bound and forwarded arbitrary-length `pkg_path`/`expression` to `Client.Eval`. The fix wraps `r.Body` in `http.MaxBytesReader` at 1 MiB before decoding, then rejects a decoded `pkg_path` over `maxEvalPkgPathLen` (1024) or `expression` over `maxEvalExpressionLen` (64 KiB). The two field caps sit well below the body cap, so they catch oversized fields that still fit inside a 1 MiB body. New constants are grouped with the existing `maxForkCodeSize`/`maxDecompressedCodeSize` guards.

## Fix
Before: unbounded read at [`handler.go:227-228`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler.go#L227-L228) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler.go#L227). After: [`handler.go:225`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler.go#L225) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler.go#L225) caps the body, and [`handler.go:238-241`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler.go#L238-L241) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler.go#L238) caps the fields. The load-bearing constraint is that the field caps are strictly below the body cap, so a request that passes the body cap can still be rejected on an oversized field before it reaches the RPC node.

## Behavior verified

Booted the eval handler from the worktree against a stub client and drove the three caps:

| Input | Result |
|-------|--------|
| body 2 MiB | 400 `invalid request body` (MaxBytesReader trips mid-decode) |
| expression 100 KiB (body < 1 MiB) | 400 `pkg_path or expression is too long` |
| pkg_path 2000 bytes | 400 `pkg_path or expression is too long` |
| expression exactly 64 KiB | 200 (boundary inclusive) |

The 100 KiB-expression row confirms the field cap is reachable and meaningful: it rejects fields that pass the body cap. None of these paths is exercised by the existing suite.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[a guard with no test can be silently dropped or inverted later]** `gno.land/pkg/gnoweb/feature/playground/handler.go:225` — the body cap, the `pkg_path` cap, and the `expression` cap add no test case.
  <details><summary>details</summary>

  [`TestHandlerPlaygroundEval`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L100-L141) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L100) tests every other eval case, and the same file tests the fork cap ([`handler_test.go:350`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L350) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L350)) and the decompression bomb ([`handler_test.go:278`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L278) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L278)), but none of the three new caps. Add three rows to the table, keyed off the constants:

  ```go
  {
  	name:       "oversized body",
  	method:     http.MethodPost,
  	body:       `{"pkg_path":"r/x","expression":"` + strings.Repeat("a", maxEvalBodyBytes+1) + `"}`,
  	wantStatus: http.StatusBadRequest,
  	wantError:  "invalid request body",
  },
  {
  	name:       "oversized pkg_path",
  	method:     http.MethodPost,
  	body:       `{"pkg_path":"` + strings.Repeat("a", maxEvalPkgPathLen+1) + `","expression":"Render(\"\")"}`,
  	wantStatus: http.StatusBadRequest,
  	wantError:  "too long",
  },
  {
  	name:       "oversized expression",
  	method:     http.MethodPost,
  	body:       `{"pkg_path":"r/x","expression":"` + strings.Repeat("a", maxEvalExpressionLen+1) + `"}`,
  	wantStatus: http.StatusBadRequest,
  	wantError:  "too long",
  },
  ```

  All three subtests pass on ee9f7e0b0 when added to the table.
  </details>

## Suggestions
- `gno.land/pkg/gnoweb/feature/playground/handler.go:238-241` — the field-cap rejection returns 400, while the sibling fork guard returns 413.
  <details><summary>details</summary>

  The fork handler returns [`http.StatusRequestEntityTooLarge`](https://github.com/gnolang/gno/blob/ee9f7e0b0/gno.land/pkg/gnoweb/feature/playground/handler.go#L133) · [↗](../../../../../.worktrees/gno-review-5897/gno.land/pkg/gnoweb/feature/playground/handler.go#L133) when its cap trips; the eval field cap returns 400. A 413 would be more consistent and clearer to a client, but 400 is defensible for a bad field and this is optional. Not posted.
  </details>

## Open questions
- The PR body asks whether 1024 (`pkg_path`) and 64 KiB (`expression`) are good sizes. They are safely generous: real pkg paths run to a few dozen bytes and expressions are short function calls, so both caps leave large headroom while still bounding the forward to the RPC node. Answered in the comment Body rather than posted as a finding.
