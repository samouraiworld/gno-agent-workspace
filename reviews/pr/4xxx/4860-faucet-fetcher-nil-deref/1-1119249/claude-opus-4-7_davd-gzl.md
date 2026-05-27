# PR #4860: fix: Avoid nil pointer dereferences in faucet Github fetcher

URL: https://github.com/gnolang/gno/pull/4860
Author: ajnavarro | Base: master | Files: 2 | +42 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m] | Commit: `1119249` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4860 1119249`

**Verdict: REQUEST CHANGES** — same nil-deref pattern the PR is fixing still exists one line below the fix at [`fetcher.go:314`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L314) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L314) (`*pe.Action` on `*github.PullRequestEvent`); finish the sweep, then ship.

## Summary

The gnofaucet GitHub fetcher rewards users by counting their issues, merged PRs, and reviews. Several methods on `*github.PullRequest`, `*github.Event`, and event payloads return pointer fields that the upstream `go-github` library populates only when the API returns them; missing fields panic the fetcher loop. This PR guards seven such dereferences (six in `api.go` accessor methods, one for `ev.Type`/`ev.Actor.Login` in the event log line) plus three inside `iterateEvents` payload branches.

## Fix

Before: accessors like `PullRequestGapi.Number()` blindly returned `*p.pr.Number`; `iterateEvents` logged `*ev.Type` and `*ev.Actor.Login` and dereferenced `*pe.Action`, `*pr.Merged`, `*review.State` without guards. After: each accessor returns a zero value (`0` / `""`) when its underlying pointer is nil; the event log line falls back to `"UNKNOWN"`; `IssuesEvent.Action`, `PullRequestEvent.PullRequest.Merged`, and `PullRequestReviewEvent.Review.State` get explicit nil guards before deref. The load-bearing constraint is partial — the parallel `PullRequestEvent.Action` dereference at [`fetcher.go:314`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L314) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L314) is left unguarded.

## Critical (must fix)

- **[same bug, one line below]** [`contribs/gnofaucet/github/fetcher.go:314`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L314) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L314) — `*pe.Action` on `*github.PullRequestEvent` is still dereferenced without a nil check
  <details><summary>details</summary>

  The PR adds `if pe.Action == nil { continue }` to the `*github.IssuesEvent` branch at [`fetcher.go:303-305`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L303-L305) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L303-L305) and `if review.State == nil` to the review branch at [`fetcher.go:338-340`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L338-L340) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L338-L340). The exact same shape — `if *pe.Action != "closed"` on `*github.PullRequestEvent` — is left bare at line 314. Whatever real event with a missing `Action` field motivated the IssuesEvent fix can hit this branch just as well; the panic recovers the loop but loses that page's progress and the Redis pipeline mid-flight. Fix: add `if pe.Action == nil { continue }` before line 314, mirroring the IssuesEvent guard.
  </details>

## Warnings (should fix)

- **[silent miscounting]** [`contribs/gnofaucet/github/api.go:80-128`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/api.go#L80-L128) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/api.go#L80-L128) — zero-value fallbacks let a malformed PR silently count as `Author=""`, `Number=0`, `CommitsCount=0`
  <details><summary>details</summary>

  `processPullRequest` at [`fetcher.go:124-159`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L124-L159) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L124-L159) already short-circuits on `u == ""` (line 126), so a nil `Author` is benign. But `CommitsCount()` returning 0 silently makes the PR contribute nothing to the commit count even if the GraphQL/REST response merely omitted the field. The fetcher emits no warning when this happens, so an upstream regression in the go-github library or a temporary GitHub API quirk would silently drop reward credit with no log breadcrumb. Fix: log an `f.logger.Warn` at the call site (or return `(int, bool)` from accessors) when a required field is nil, so operators can see the rate of malformed responses instead of a silent zero.
  </details>

- **[partial Redis pipeline on panic]** [`contribs/gnofaucet/github/fetcher.go:264-362`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L264-L362) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L264-L362) — `iterateEvents` queues writes into `pipe` (passed in by caller), but if any unguarded deref panics mid-loop the caller `fetchEvents` still calls `pipe.Exec(ctx)` from line 70
  <details><summary>details</summary>

  `fetchEvents` at [`fetcher.go:61-81`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L61-L81) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L61-L81) calls `iterateEvents` then unconditionally calls `pipe.Exec` afterward. A panic inside `iterateEvents` (from any remaining unguarded deref, including line 314 above) would unwind through `iterateEvents` without resetting the pipe, but the deferred `pipe.Exec` is not actually deferred — it runs only if `iterateEvents` returns. A panic kills the whole goroutine, so the pipe is dropped (good) but also the `lastRepoFetchKey` update at line 350 didn't run for the panicking event, leading to re-processing the same page on the next tick. This compounds with rate limits. Not a regression introduced by this PR, but the PR's stated motivation ("we got some unexpected empty fields") suggests the failure mode is real and ongoing — worth a Warn-level log when a deref-guard trips so operators can see how often.
  </details>

## Nits

- [`contribs/gnofaucet/github/fetcher.go:279-291`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L279-L291) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L279-L291) — six-line if/else for two `*string` fallbacks could be one `cmp.Or(deref, "UNKNOWN")` helper or a tiny `strDeref(p *string, fallback string) string` to dedupe the pattern; same shape recurs across the file.
- [`contribs/gnofaucet/github/api.go:88-93`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/api.go#L88-L93) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/api.go#L88-L93) — `Author()` returns `""` if either `p.pr.User` or `p.pr.User.Login` is nil; consider distinguishing "no user object" from "user with nil login" in a debug log, since the former usually means the PR is from a deleted account while the latter is an API anomaly.
- [`contribs/gnofaucet/github/fetcher.go:117`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L117) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L117) — `issue.Number` is logged as a `*int` pointer (no deref), so slog prints the address or `<nil>`. Pre-existing, but worth `issue.GetNumber()` for consistency with the new defensive style.

## Missing Tests

- **[no regression test for the bug being fixed]** [`contribs/gnofaucet/github/fetcher_test.go`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher_test.go) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher_test.go) — the PR adds 42 lines of nil guards but no test that asserts the fetcher survives a `github.Event` with `Type=nil` or a `PullRequestEvent` with `Action=nil`
  <details><summary>details</summary>

  Codecov flags this: 20% patch coverage. The bug is "we saw real responses with empty fields"; a unit test that feeds a synthetic event with the missing field into `iterateEvents` (or directly into `PullRequestGapi` accessors) would lock in the fix. Without it, a future refactor that re-introduces an unguarded `*pe.Action` won't trip CI. Especially relevant given the [`fetcher.go:314`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L314) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L314) miss above — a test for the `IssuesEvent.Action == nil` case would naturally raise the question of whether `PullRequestEvent.Action == nil` is also handled.
  </details>

## Suggestions

- [`contribs/gnofaucet/github/api.go:80-128`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/api.go#L80-L128) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/api.go#L80-L128) — go-github already provides `GetNumber()`, `GetUser().GetLogin()`, `GetCommits()`, `GetState()`, `GetTitle()` helpers on `*github.PullRequest` that do the same nil-safe deref-to-zero-value. The accessor methods on `PullRequestGapi` could be one-liners delegating to those.
  <details><summary>details</summary>

  Example: `func (p *PullRequestGapi) Number() int { return p.pr.GetNumber() }`. Same zero-value semantics, no manual nil checks, no risk of forgetting one when adding a field. The go-github library guarantees these `GetX` accessors return zero on nil receiver or nil field — that's what they exist for. Cuts the PR diff in `api.go` from ~15 lines to ~6, and removes future maintenance burden.
  </details>

## Questions for Author

- What was the actual production failure that motivated this? A stack trace would clarify which field was nil and whether [`fetcher.go:314`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L314) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L314) is hot enough to matter. The PR has been stale for 7 months — is this still a live issue, or has the underlying go-github / GitHub API behavior changed?
- The `Reviews()` method on `PullRequestGapi` at [`api.go:109-112`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/api.go#L109-L112) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/api.go#L109-L112) returns `nil` with a comment "we don't have this information here" — meaning REST-API-sourced PRs (the `iterateEvents` path) never credit reviewers. Is that intentional? If so, the `processEventReview` path at [`fetcher.go:161-186`](https://github.com/gnolang/gno/blob/1119249/contribs/gnofaucet/github/fetcher.go#L161-L186) · [↗](../../../../../.worktrees/gno-review-4860/contribs/gnofaucet/github/fetcher.go#L161-L186) is the only way REST-flow review credit happens, which makes the nil guards there particularly important.
