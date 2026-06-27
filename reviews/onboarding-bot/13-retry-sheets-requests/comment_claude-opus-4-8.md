# Review: PR #13
Event: COMMENT

## Body
Verified on f0bbdb7. No retried call can double-write the tracker: the candidate write uses `UpdateRow` to a pre-scanned empty row under `appendMu`, not `Values.Append`, so a retry adds no duplicate row, and every other retried op overwrites a fixed range or is a declarative `BatchUpdate`, so re-running after a lost-response 5xx lands the same cell state. Backoff respects ctx cancellation and Retry-After. `retry_test.go` covers classification, parsing, exhaustion, and cancel. Three notes below, none blocking.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/onboarding-bot/13-retry-sheets-requests/review_claude-opus-4-8.md [↗](review_claude-opus-4-8.md)

## internal/sheet/retry.go:97 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/f0bbdb7/internal/sheet/retry.go#L97)
Only `*googleapi.Error` 429/5xx is retried. A connection reset or transport timeout mid-harvest comes back as a raw `url.Error`/`net` error, not a `googleapi.Error`, so it skips retry and still aborts the whole pass, which is the same transient class the retry is meant to absorb. If the 429/5xx-only scope is deliberate, fine; otherwise also retry `net.Error` timeouts and connection resets (those wouldn't be safe for `Append`, but `Append` isn't on a retried path here).

## internal/sheet/retry.go:62 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/f0bbdb7/internal/sheet/retry.go#L62)
A `Retry-After` longer than `MaxDelay` is clamped to `MaxDelay`, so the retry fires before the server said it would accept traffic and spends an attempt on a likely repeat 429. The cap is intentional and the 60s default matches the quota window, so it rarely bites; worth noting in the config comment that `sheet_retry_max` is the lever if a pass keeps exhausting attempts.

## internal/sheet/client.go:435 [↗](https://github.com/samouraiworld/gno-onboarding-bot/blob/f0bbdb7/internal/sheet/client.go#L435)
If a 5xx is returned after `AddSheet` already created the tab server-side, the retry hits a 400 "already exists" and `EnsureTab` errors despite success. Very low probability and off the hot path; flagging for awareness only.
