# PR #13: Retry rate-limited Sheets requests; simplify approve message

Repo: samouraiworld/gno-onboarding-bot
URL: https://github.com/samouraiworld/gno-onboarding-bot/pull/13
Author: D4ryl00 (Rémi BARBERO) | Base: master | Head: update-approve-template | Files: 7 | +497 -136
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: f0bbdb7 (latest)
Local checkout: `git clone https://github.com/samouraiworld/gno-onboarding-bot .worktrees/onboarding-bot-review-13 && cd .worktrees/onboarding-bot-review-13 && gh pr checkout 13`

**TL;DR:** Routes every Google Sheets API call through a retry-with-backoff helper that retries 429 (quota) and 5xx (transient) responses with exponential backoff, jitter, Retry-After support, and context cancellation. Policy is config-driven (`sheet_max_retries`/`sheet_retry_base`/`sheet_retry_max`). Also simplifies the approve-message wording.

**Verdict: APPROVE.** The retry helper wraps every Sheets call, retries 429/5xx with bounded exponential backoff, and respects ctx cancellation. The wrapped calls reproduce the prior behavior, and the only non-idempotent Sheets op, `Values.Append`, is not on a retried path, so a retry cannot double-write. `retry_test.go` covers classification, Retry-After parsing, exhaustion, and ctx-cancel. Three advisory notes below, none blocking.

## Summary

The diff wraps each `c.svc.Spreadsheets...Do()` call in a new `c.do(ctx, op)` helper ([retry.go:42](internal/sheet/retry.go:42)) that re-runs `op` on retryable errors with exponential backoff plus `[0.5,1.0]` jitter, honoring a `Retry-After` header and respecting `ctx` cancellation. `isRetryable` classifies `*googleapi.Error` 429 and >=500 as retryable, everything else not ([retry.go:97](internal/sheet/retry.go:97)). The policy comes from config (`SheetMaxRetries`/`SheetRetryBaseEvery`/`SheetRetryMaxEvery`), with `normalized()` filling unset fields from `DefaultRetryPolicy` (8 attempts / 2s base / 60s cap) and clamping `MaxDelay >= BaseDelay`.

Verified the retry can't corrupt the tracker: the candidate write path uses `UpdateRow` (overwrite of a computed target range) under `appendMu`, not the Sheets `Append` API ([sheet.go:367-388](internal/sheet/sheet.go:367)). The `Append` client method exists in the interface but has no business caller, so the one non-idempotent Sheets op isn't on a retried path. Every retried write (`Update`, `BatchUpdate` cell/format/validation, `Clear`, `WriteRows`) targets a fixed range or is a set-operation, so re-execution after a lost-response 5xx lands the same result.

## Good

- **Retried writes are idempotent.** `Update`/`UpdateRow`/`WriteRows` overwrite a fixed A1 range and `BatchUpdate` set-requests are declarative, so a retry after a 5xx that already applied server-side reproduces the same cell state. The one append-style op (`AppendCandidateRow`) is built on `UpdateRow` to a pre-scanned empty row, not `Values.Append`, so retry can't duplicate a row.
- **Context cancellation is correct and non-retried.** The backoff `select`s on `ctx.Done()` ([retry.go:73-77](internal/sheet/retry.go:73)), and a ctx error from `Do()` is not a `*googleapi.Error`, so `isRetryable` returns false and the call returns immediately instead of looping. Covered by `TestDoStopsOnContextCancel`.
- **Solid unit coverage.** `retry_test.go` exercises retryable classification (429/5xx vs 4xx, wrapped errors), Retry-After parsing (int seconds, zero, negative, garbage, future/past HTTP date, absent), policy normalization, retry-then-succeed, attempt exhaustion, no-retry on 4xx, and ctx-cancel.
- **Config parse errors fail fast.** `sheet_retry_base`/`sheet_retry_max` are parsed at load with a clear `"is not a valid Go duration"` error ([config.go:84-95](internal/config/config.go:84)); unset fields fall through to the client default via `normalized()`.

## Warnings (consider)

- **[transient network errors aren't retried, only HTTP 429/5xx]** `internal/sheet/retry.go:97` — `isRetryable` matches only `*googleapi.Error`. A connection reset, TLS error, or client `i/o timeout` mid-harvest surfaces as a raw `url.Error`/`net` error, not a `googleapi.Error`, so it skips retry and still aborts the whole pass.
  <details><summary>details</summary>

  The PR's goal is to keep a long `/harvest` pass alive across transient failures. HTTP 429/5xx is the common case, but a dropped TCP connection or a transport timeout is exactly the same class of transient failure and won't be a `*googleapi.Error` (those are produced by `googleapi.CheckResponse` only when the server returned a structured error response). So a single network blip during a multi-minute pass defeats the retry.

  The PR body scopes the change to "429 and 5xx", so this may be deliberate. If so, fine. If transport-level resilience is wanted, also treat `net.Error` with `Timeout()` true and `errors.Is(err, io.ErrUnexpectedEOF)` / `syscall.ECONNRESET` as retryable. Note these are NOT idempotency-safe for `Append`, but `Append` isn't on a retried path here.
  </details>

## Nits

- `internal/sheet/retry.go:62-65` — a `Retry-After` longer than `MaxDelay` is clamped down to `MaxDelay`, so the retry fires before the server said it would accept traffic and burns an attempt on a near-certain repeat 429. Documented as intentional ("keep the whole backoff bounded by MaxDelay"), and the 60s default cap matches the per-minute quota window, so it rarely bites. If a `/harvest` keeps exhausting attempts, raising `sheet_retry_max` is the lever; worth a one-line mention in the config comment.
- `internal/sheet/client.go:435` (`EnsureTab`) — the `AddSheet` `BatchUpdate` is wrapped in `c.do`. If a 5xx is returned after the tab was created server-side, the retried `AddSheet` gets a 400 "already exists" (non-retryable) and `EnsureTab` returns an error despite the tab existing. Very low probability (5xx-after-apply on a create) and `EnsureTab` runs once per tab, not on the hot harvest path; flagging for awareness, not a fix.

## Open questions

- `sheet_max_retries` is "total attempts including the first" (so `8` = 7 retries), clarified in both the struct comment and config.example. Naming reads as a retry count; not worth renaming, but confirm the config.example wording is enough to avoid an off-by-one in operator expectations.
