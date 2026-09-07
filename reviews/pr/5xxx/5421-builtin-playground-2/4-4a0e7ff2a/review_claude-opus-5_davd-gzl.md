# PR [#5421](https://github.com/gnolang/gno/pull/5421): feat(gnoweb): built-in playground(2)

URL: https://github.com/gnolang/gno/pull/5421
Author: moul | Base: master | Files: 68 | +4481 -85
Reviewed by: davd-gzl | Model: claude-opus-5, effort high | Commit: `4a0e7ff2a` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5421 4a0e7ff2a`
Overview: [overview](../overview.md)

Round 4. Head advanced `15613c21a` to `4a0e7ff2a`; the patch-ids differ, and the whole
difference is [one commit regenerating the bundled assets](https://github.com/gnolang/gno/commit/bd2c982418a04371ac6ea5b8b30f41079e5fc50d)
plus two merges of master. Every Go file under `feature/playground` is byte-identical to the
round-3 head. What moved is the base: master's [#6088](https://github.com/gnolang/gno/pull/6088)
made the node refuse an unsigned `MsgRun` on the simulate path, and that lands the branch's Dry
Run button on a signature it does not produce. The four findings round 3 left open all still
reproduce here.

## Overview

The branch puts an editor and an expression evaluator inside gnoweb, so gno.land serves the job
that play.gno.land does today from a separate application. Three JSON endpoints stand between
the page and the node: `/_/api/eval` runs one expression, `/_/api/funcs` lists a package's
functions, and `/_/api/dryrun` simulates a whole `maketx run` script. All three are open to
anyone who can reach the site, and everything on screen comes from a query the node already
answers.

**Verdict: REQUEST CHANGES** — the Dry Run button cannot succeed at this head, because
[`Simulate`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L285-L286) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L285-L286)
puts a public key where the node now wants a signature; alongside it the Run view's one key
field is read as a key name by the command it prints and as a bech32 address by the request it
posts, and `/_/api/funcs` still forwards a `vm/qdoc` per call with nothing throttling it.
(1 Critical, 5 Warnings, 1 Missing test, 3 Nits.)

## Verify first

- [`client.go:285-286`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L285-L286) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L285-L286) — boot gnodev from this head and press Dry Run on any realm's `?run` page with a funded address that has already signed. Expect the rendered output; the head returns `unauthorized error`.
- [`controller-run.ts:134`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L134) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L134) — type the key field's own placeholder, `mykey`, and press Dry Run. The generated command below accepts it; the request rejects it.
- [`feature/playground/handler.go:299`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L299) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L299) — `curl` `/_/api/funcs?path=…` 60 times from one address and count the 429s. There are none.

## Summary

Nothing in the playground's own Go code changed since round 3, so this round is a re-measurement
plus the first pass over `serveDryRun`, which arrived with [#6035](https://github.com/gnolang/gno/pull/6035)
at the round-3 head and no round has covered. That endpoint is now broken: master's
[`txCarriesCode`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoland/app.go#L1298) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoland/app.go#L1298)
predicate, [wired into the auth ante](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoland/app.go#L161) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoland/app.go#L161)
by #6088, makes `.app/simulate` verify the signature on any transaction carrying Gno source, and
gnoweb sends a pubkey with no signature bytes. The same button worked at `15613c21a`, measured
on a gnodev built from that head. Beside it, the Run view's single key field is consumed two
incompatible ways, and the three amplification gaps round 3 recorded are unchanged.

Reading order: [`client.go`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L265-L307) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L265-L307)
first, then [`feature/playground/handler.go`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L346-L426) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L346-L426),
then the two frontend controllers.

## Fix

`Simulate` queries the account, reads its public key, and
[assigns a signature list holding that key alone](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L285-L287) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L285-L287),
with the comment "No need to sign to get the actual signature bytes". That held while simulate
skipped verification for every message. It no longer does for `MsgRun`, which is the only
message `serveDryRun` ever builds. The constraint is that gnoweb holds no key, so the fix is
either a signature the caller supplies or dropping the button rather than a change inside
`Simulate`.

## Numbers

Measured on gnodev built from each head, one gno.land node per head, address
`g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5` funded and holding a public key on chain.

| Call | 15613c21a | 4a0e7ff2a |
| --- | --- | --- |
| `POST /_/api/dryrun`, bech32 address | rendered output, 200 | `error encountered during simulation: unauthorized error`, 200 |
| `POST /_/api/dryrun`, key name `mykey` | `address must be a bech32 address`, 400 | `address must be a bech32 address`, 400 |
| `gnokey maketx run --simulate only`, same script | gas estimate | gas estimate, 144124346 |
| `POST /_/api/eval` x30, one address, no header | 10 pass then 429 | 10 pass then 429 |
| `POST /_/api/eval` x30, rotating `X-Forwarded-For` | 30 pass | 30 pass |
| `GET /_/api/funcs` x60, one address | 60 pass | 60 pass |

## Critical (must fix)

- **[the Dry Run button cannot succeed against a node built from this branch]** [`client.go:285-286`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L285-L286) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L285-L286) — `Simulate` sends a signature list carrying a public key and no signature bytes, and the merged base makes `.app/simulate` verify signatures on every message that carries Gno source.
  <details><summary>details</summary>

  [`serveDryRun`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L397-L411) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L397-L411) builds a `vm.MsgRun` and hands it to `Simulate`, which fills `tx.Signatures` from the account's public key. Master's [`txCarriesCode`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoland/app.go#L1298) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoland/app.go#L1298) selects `MsgRun`, and the auth ante [verifies the signature whenever that predicate fires](https://github.com/gnolang/gno/blob/4a0e7ff2a/tm2/pkg/sdk/auth/ante.go#L309-L313) · [↗](../../../../../.worktrees/gno-review-5421/tm2/pkg/sdk/auth/ante.go#L309-L313), simulate or not. `AnteOptions` says so directly: a transaction selected by the predicate "must carry a real signature, not a pubkey-only placeholder". Every dry run therefore ends in `unauthorized error`, and it arrives at HTTP 200, so the browser renders it as an ordinary script failure.

  The same button returned the rendered output on a gnodev built from `15613c21a`, whose base predates #6088. `gnokey maketx run --simulate only` succeeds against a node from this head, signing a second transaction for the estimate, which puts the gap in gnoweb's client rather than in the node. The repro is [`tests/dryrun-unauthorized.sh`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5421-builtin-playground-2/4-4a0e7ff2a/tests/dryrun-unauthorized.sh) and it prints both halves.

  Fix: take a signed transaction from the caller, or drop Dry Run from the branch until a wallet can sign one.
  </details>

## Warnings (should fix)

- **[one key field, two readers, incompatible formats]** [`controller-run.ts:134`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L134) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L134) — the dry run posts the key field as `address` and the endpoint requires bech32, while the field is labelled "Key name or address" and placeholdered `mykey`.
  <details><summary>details</summary>

  [`_buildCmd`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L76-L101) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L76-L101) drops the same string into `gnokey maketx run … <key> script.gno`, where a key name is the ordinary value and the [label offers it first](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/templates/page.html#L53-L56) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/templates/page.html#L53-L56). `serveDryRun` [rejects anything that is not bech32](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L372-L376) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L372-L376) with a 400. Typing the placeholder gives `Error: address must be a bech32 address`; gnokey accepts an address in place of a key name, so the address is the only value both readers take, and nothing on the page says so.

  Measured against gnodev at this head: field `mykey` returns 400 with that message, field `g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5` reaches the node. The clip is [`media/dry-run-never-succeeds.gif`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5421-builtin-playground-2/4-4a0e7ff2a/media/dry-run-never-succeeds.gif), produced by [`tests/film-5421-dryrun.mjs`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5421-builtin-playground-2/4-4a0e7ff2a/tests/film-5421-dryrun.mjs), and it carries the Critical above in the same ledger.

  Fix: label the field for the address the dry run needs, or resolve a key name before posting it.
  </details>

- **[`/_/api/funcs` has no rate limiter]** raised by [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256256566) [`feature/playground/handler.go:299`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L299) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L299) — `serveFuncs` never calls `h.limiter.allow`, though `serveEval` and `serveDryRun` on the same `Handler` both do.
  <details><summary>details</summary>

  `GET /_/api/funcs?path=…` forwards a `vm/qdoc` on every request, which walks the package AST and serializes a JSON entry per exported symbol. [`serveEval`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L252-L255) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L252-L255) and [`serveDryRun`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L353-L356) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L353-L356) share one bucket; funcs consults none.

  Measured at this head: 60 back-to-back calls from one address return 60 times 200.

  Fix: run `serveFuncs` through the same limiter the other two use.
  </details>

- **[the rate-limit key is a header the caller writes]** raised by [@moul](https://github.com/gnolang/gno/pull/5421#discussion_r3512098587) and [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256226373) [`feature/playground/ratelimit.go:88`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L88) — `clientIP` returns the first `X-Forwarded-For` entry with no check on `RemoteAddr`, so a caller rotating the header lands every request in a fresh bucket.
  <details><summary>details</summary>

  Measured at this head from one TCP peer: 30 eval calls without the header give 10 passes then 20 rejections, and 30 with a rotating `X-Forwarded-For` give 30 passes. Since #6035 the same bucket also guards `serveDryRun`, so the bypass now reaches the simulate path, whose script the caller writes.

  [jefft0 recorded the maintainers' decision](https://github.com/gnolang/gno/pull/5421#discussion_r3718851602) on this line: the infrastructure handles rate limiting, not gnoweb. Kept here as a measurement, not as a proposal.
  </details>

- **[`pruneLoop` has no shutdown path]** raised by [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256267671) [`feature/playground/ratelimit.go:40`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L40) — `newRateLimiter` starts the goroutine with no context, no `Stop`, and no reachable exit.
  <details><summary>details</summary>

  [`pruneLoop`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L70-L83) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/ratelimit.go#L70-L83) ranges over a ticker nothing stops. Bounded in production because `playground.New` runs once per gnoweb instance; every `playground.New` in a test leaks one goroutine.

  Fix: select on a context in `pruneLoop`, or expose a `Stop` that closes a done channel.
  </details>

- **[a node failure and a Gno failure arrive as the same status]** raised by [@alexiscolin](https://github.com/gnolang/gno/pull/5421#discussion_r3256269388) [`feature/playground/handler.go:292`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L292) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L292) — eval, [funcs](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L313) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L313) and [dry run](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L416-L418) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L416-L418) all answer `200 {"error":…}` when the query fails at the node.

  <details><summary>details</summary>

  A 200 on an upstream failure leaves nothing for a `grep '" 5'` over access logs to find, defeats reverse-proxy retry, and keeps `response.ok` true on the frontend. The Critical above is the live example: `unauthorized error` comes back at 200 and the Result pane styles it exactly like a script that returned an error of its own.

  Fix: answer 502 when the query fails at the transport or node level, and keep 200 for a query that succeeded carrying a Gno-level error.
  </details>

## Missing Tests

- **[nothing exercises `serveDryRun`]** [`feature/playground/handler_test.go:65-67`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L65-L67) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler_test.go#L65-L67) — `stubClient` carries a `Simulate` method and `simulateResult` and `simulateErr` fields, and no test in the package sets or reads any of them.
  <details><summary>details</summary>

  The package covers eval, funcs, the fork view, the deflate cap and the limiter. The one endpoint that hands the node a whole transaction has no case at all, which is why the branch reaches this head with the button dead and CI green. A table case posting a valid body and asserting 200 with the stub's result, one posting a non-bech32 `address` and asserting 400, and one setting `simulateErr` and asserting what the caller sees, would each have caught something in this round.

  Fix: mirror `TestHandlerPlaygroundEval` for `DryRunHandler`.
  </details>

## Nits

- [`feature/playground/handler.go:120`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L120) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L120) — `fileanme` is misspelled in four places in `GetForkView`, at [L120](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L120), [L126](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L126), [L159](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L159) and [L169](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L169). No enabled linter in [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/4a0e7ff2a/.github/golangci.yml) catches it.
- [`controller-action-function.ts:266`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/frontend/js/controller-action-function.ts#L266) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/frontend/js/controller-action-function.ts#L266) — the Eval button's placeholder builder gives `""` to a `string` parameter and `0` to every other type, so `bool` and `std.Address` both arrive as `0`, and the split on `,` reads `f(a, b string)` as one parameter named `a` and one typed `string`. Not posted: the value is a placeholder the caller edits before evaluating.
- [`controller-run.ts:87`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L87) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L87) — `send !== "0ugnot"` skips the exact string only, so `0`, `0 ugnot` and `00ugnot` each add a `-send` line to the printed command. Carried from round 3, unchanged. Not posted: it affects a command string the reader copies, and the reader sees the value they typed.

## Verified

- Dry Run at `15613c21a` returns the rendered output of `gno.land/r/gnoland/home`, and at `4a0e7ff2a` returns `unauthorized error`. Two gnodev binaries, one per head, each with the address funded by one broadcast transaction so it carries a public key.
- `gnokey maketx run --simulate only` against a node from `4a0e7ff2a` reports 144124346 gas used for the same script, which is the control keeping the failure attributable to gnoweb's client.
- The rate-limit rows in Numbers are single runs against gnodev at this head, not read from the tests.
- `go test ./gno.land/pkg/gnoweb/feature/playground/... ./gno.land/pkg/gnoweb/feature/run/...` is green at this head.

## Existing threads

| Reviewer | Gist | State | Link |
| --- | --- | --- | --- |
| moul | eval body unbounded | fixed at this head, `maxEvalBodyBytes` and the two length caps | [r3512098582](https://github.com/gnolang/gno/pull/5421#discussion_r3512098582) |
| moul, alexiscolin | XFF trusted-proxy gate | settled: infrastructure handles it | [r3718851602](https://github.com/gnolang/gno/pull/5421#discussion_r3718851602) |
| alexiscolin | funcs limiter | open, reproduces here | [r3256256566](https://github.com/gnolang/gno/pull/5421#discussion_r3256256566) |
| alexiscolin | `pruneLoop` shutdown | open | [r3256267671](https://github.com/gnolang/gno/pull/5421#discussion_r3256267671) |
| alexiscolin | 200 on backend failure | open | [r3256269388](https://github.com/gnolang/gno/pull/5421#discussion_r3256269388) |
| alexiscolin | `prompt()` for a new file | deferred to the design review | [r3256275783](https://github.com/gnolang/gno/pull/5421#discussion_r3256275783) |
| moul, jefft0 | the playground Run button evaluates a fixed package | answered by Dry Run, which this round finds broken | [r3512098590](https://github.com/gnolang/gno/pull/5421#discussion_r3512098590) |
| davd-gzl | the `?from=` case asserts no fork | open, unchanged at [`handler_http_test.go:1626`](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/handler_http_test.go#L1626) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/handler_http_test.go#L1626) | [r3889878022](https://github.com/gnolang/gno/pull/5421#discussion_r3889878022) |

## Open questions

- `?fork` runs no limiter and fetches [8 files at a time](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/feature/playground/handler.go#L112) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/feature/playground/handler.go#L112), each bounded only by the client's own [8 MiB response cap](https://github.com/gnolang/gno/blob/4a0e7ff2a/gno.land/pkg/gnoweb/client.go#L38) · [↗](../../../../../.worktrees/gno-review-5421/gno.land/pkg/gnoweb/client.go#L38), and the 1 MiB fork ceiling is checked after each body is already in memory. The peak that implies was not measured here. Not posted: no run behind it.
- The Fork header button is commented out rather than deleted, and `?fork` still serves. Whether a query with no way to reach it should ship is the author's call. Not posted: no defect, and the branch says the button waits on publishing.
