# Review: [#6231](https://github.com/gnolang/gno/pull/6231)

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The branch ships 14 Warnings to gnoweb readers, among them `-indexer-url` credentials printed on every indexer-backed results page, a `$search` limiter at 100 requests a minute, and scoped searches answering from the whole chain.
Model: claude-opus-5-5, standard review
Commit: dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6231 dd40ef3dc`
Round: 2. 10 finders, one reflector, 53 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted, a Warning.

## Body

- Warning: the gnoweb binary runs both limiters, the `$search` one included, at 100 requests a minute instead of 1200. [`NewDefaultAppConfig`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/app.go#L116) sets `StateRateLimitPerMinute: 100`, and [`NewHTTPHandler`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_http.go#L184-L186) applies the 1200 default only when the rate is 0 or less. [That rate feeds the search limiter](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_http.go#L216-L219), so behind a proxy with no `--trusted-proxies` every visitor shares one bucket and omnibar typeahead answers 429 after 100 requests a minute site-wide.

  <details><summary>repro</summary>

  The 101st request from one address gets 429.

  ```sh
  # from a local clone of gnolang/gno:
  gh pr checkout 6231 -R gnolang/gno
  curl -fsSL -o gno.land/pkg/gnoweb/zz_default_ratelimit_test.go \
    https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b1-lines-removed-claims-reach-catalog-default-ratelimit-100.go
  go test ./gno.land/pkg/gnoweb -run TestZZDefaultAppConfigRateLimit -count=1 -v
  curl -fsSL -o gno.land/pkg/gnoweb/zz_router_rate_default_test.go \
    https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b2-lines-removed-reach-catalog-router-rate-default_test.go
  go test ./gno.land/pkg/gnoweb -run TestRouterSearchRateLimitDefault -v -count=1
  ```
  </details>

## gno.land/pkg/gnoweb/client.go:238 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/client.go#L238) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/client.go#L238) · Warning
Forwarding `?limit=` caps `/u/<user>` at [`MaxUserContributions`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_http.go#L673), 200 packages per namespace, where the bare `vm/qpaths` returned the [node's default of 1000](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/sdk/vm/handler.go#L200). Namespaces of 201 to 1000 packages lose their tail, and the page shows the capped count as the total with no truncation mark.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/zz_user_contrib_cap_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b2-lines-removed-reach-catalog-user-contrib-cap_test.go
go test ./gno.land/pkg/gnoweb -run TestUserContributionsListingNotLowered -v -count=1
```
</details>

## gno.land/pkg/gnoweb/feature/omnisearch/discover.go:17 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L17) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L17) · Warning
`discover` builds its filter from `q.Text` alone, so a search scoped to `/r/alice/blog`, by the page or by `in:`, lists `/r/bob/blog` from the whole chain. [The page hint](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L96) advertises `in:` as "Narrow to a package", and the JSON response echoes [`pkg_path`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/json.go#L48) as the scope.

<details><summary>repro</summary>

The first test finds an out-of-scope result for both `/$search&q=blog+in:/r/alice/blog` and `/r/alice/blog$search&q=blog`; the second reads `pkg_path="/r/alice/blog"` beside `result=/r/bob/blog` in the JSON response.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b6-lines-removed-reach-catalog-discover-ignores-scope_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b6-lines-removed-reach-catalog-discover-ignores-scope_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB6DiscoverIgnoresScope -v
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_inqual_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b3-lines-reach-catalog-in-qualifier-ignored_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3InQualifier' -count=1 -v
```

[`Handler.Search`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L130) sends every query naming no selector to `discover`.

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/feature.go:96 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/feature.go#L96) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/feature.go#L96) · Warning
`Client.Doc` runs inside `singleflight.DoChan` with no recover, so a panic in `Doc` during a `func:`, `type:` or `imports` search exits the whole gnoweb process. `DoChan` re-panics on a fresh goroutine that the per-request recover of net/http cannot reach, where [`state/page.go`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/state/page.go#L75) wraps the same call in `recoverFetcher`.

<details><summary>repro</summary>

The panic is injected through a fake client; no real node reply was shown to panic.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_docpanic_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b3-lines-reach-catalog-doc-panic-crash_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3DocPanic' -count=1 -v
```
</details>

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/handler.go:23 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L23) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L23) · Warning
`jsonTimeout` of 3s expires before the 4s timeout of the indexer client, and [`Client.Query`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L135) records the caller's `context.DeadlineExceeded` as an indexer failure. Three concurrent omnibar lookups against an indexer answering in 3.3s open the breaker, and every results page reports "indexer unavailable" for 30s.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_breaker_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b3-lines-reach-catalog-omnibar-trips-breaker_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3OmnibarTripsBreaker' -count=1 -v
```
</details>

Not posted: same defect as `gno.land/pkg/gnoweb/indexer/client.go:135`, which one edit to the breaker condition in `Client.Query` closes; its repro is folded there.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/handler.go:130 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L130) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L130) · Warning
`discover` answers every query with no selector and never reads `q.PkgPath`, so `in:/r/alice/blog` returns the unscoped listing. [The page hint](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L96) advertises `in:` as "Narrow to a package", and the JSON echoes `pkg_path` as the scope.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_inqual_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b3-lines-reach-catalog-in-qualifier-ignored_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3InQualifier' -count=1 -v
```
</details>

Not posted: same defect as `gno.land/pkg/gnoweb/feature/omnisearch/discover.go:17`, which one edit to `discover` closes; its repro is folded there.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/handler.go:229 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L229) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L229) · Warning
`IndexerStatus.URL` takes [`Indexer.URL()`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L115) verbatim, so a basic-auth password in `-indexer-url` is printed in the results page footer to every anonymous visitor. Go's HTTP client sends that password as working credentials.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b5_userinfo_probe_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b5-lines-reach-catalog-url-userinfo_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB5' -count=1 -v
```
</details>

Not posted: same defect as `gno.land/pkg/gnoweb/indexer/client.go:115`, which one edit to `URL()` closes; its repro is folded there.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/json.go:92 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/json.go#L92) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/json.go#L92) · Warning
`jsonIndexer.URL` copies the configured indexer URL into the public `/$search&json` response, so the same `-indexer-url` userinfo reaches every JSON reader as `"indexer":{"url":...}`.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b5_userinfo_probe_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b5-lines-reach-catalog-url-userinfo_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB5' -count=1 -v
```
</details>

Not posted: same defect as `gno.land/pkg/gnoweb/indexer/client.go:115`, which one edit to `URL()` closes; its repro is folded there.

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go:158 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L158) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L158) · Warning
`capResults` cuts `func:`, `type:`, `file:` and `imports` results at [`MaxResults`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/query.go#L26), here and at lines [179](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L179), [213](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L213) and [234](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L234), with `Group.Truncated` left false, so a partial list reads as complete.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b9-lines-reach-catalog-resolve-chain_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b9-lines-reach-catalog-resolve-chain_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB9FuncResultsCappedReportTruncation -v
```

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go:135 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L135) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L135) · Warning
`SourceContains` gets the bare `q.ChainPath` as an [unanchored pattern](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/queries.go#L200-L201), so "Imported by" on `gno.land/r/demo/foo` also lists importers of `gno.land/r/demo/foobar`. Every `gno.land/r/demo/foo/<sub>` matches through its gnomod module line too, each using up the [`recentLimit`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L16) budget.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/indexer/zz_window_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b4-lines-reach-catalog-recent-window.go
go test ./gno.land/pkg/gnoweb/indexer/ -run TestZZImportersTextMatchesALongerPath -v
```
</details>

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go:175 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L175) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L175) · Warning
This loop takes `Path()` of every message in a matched transaction, and [tx-indexer returns the whole transaction](https://github.com/gnolang/tx-indexer/blob/57b1b385c928a55df1dd71415f54d3e11f44f3b6/serve/graph/model/filter_methods_gen.go#L1365-L1375) when any message matches. A realm the transaction only called, or a sibling deployed alongside, is listed under `content:` and, through the [same loop in `resolveImporters`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L143-L157), under importers.

<details><summary>repro</summary>

Expected only `gno.land/r/alice/a`; `gno.land/r/bob/b`, which the transaction only calls, comes back too.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/zz_multimsg_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b4-lines-reach-catalog-multimsg.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestZZ -v
```
</details>

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go:82 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L82) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L82) · Warning
Every `Realm` failure returns nil here, so a `render:` search answers "Nothing matched." instead of "Could not answer" when the node times out or is down. With one candidate from `in:` or page scope, the page tells the reader the realm lacks the text.

<details><summary>repro</summary>

Copy `tests/b6-lines-removed-reach-catalog-render-errors-swallowed_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test -run TestB6RenderAllCandidatesFailReportsNothingMatched -v`: "Nothing matched." in all three cases, a deadline via `in:`, connection refused via page scope and connection refused via `author:`.

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go:119 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L119) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L119) · Warning
This call drops the truncated flag of the listing for `render:<text> author:<ns>`, and the loop stops at [`maxRenderCandidates`, 8](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L16), without marking the group. A match past either cut reads as "Nothing matched.", while [`discover` marks the same listing `Truncated`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L86-L92).

<details><summary>repro</summary>

Copy `tests/b6-lines-removed-reach-catalog-render-cap-silent_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test -run TestB6RenderCapIsSilent -v`: `results=0 truncated=false err=<nil>`, and `render truncated=false` beside `discover truncated=true` on the same directory.

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/templates/_results.html:22 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/_results.html#L22) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/templates/_results.html#L22) · Warning
The truncation notice lives inside a group, and [`discover` drops empty groups](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L78-L80). A truncated listing with no visible match therefore says "Nothing matched." with no notice, which is exactly the case the flag exists for.

<details><summary>repro</summary>

Copy `tests/b8-lines-reach-catalog-results-page_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB8TruncatedListingWithNoMatchSaysNothingMatched -v`: `groups=0` and "Nothing matched.".

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html:14 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L14) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L14) · Warning
`PkgPath` alone decides the scope header here, so whole-chain discovery results get it too. The reader is told the list covers one realm when it does not.

<details><summary>repro</summary>

Copy `tests/b8-lines-reach-catalog-results-page_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB8ScopedHeaderOverWholeChainResults -v`: the page shows both the scope header and `/r/bob/blog`.

</details>

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html:65 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L65) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L65) · Warning
The provenance footer renders `IndexerStatus.URL` verbatim, and `json.go:92` serialises it, so credentials in `-indexer-url` reach every visitor even when the indexer is down.

<details><summary>repro</summary>

Copy `tests/b8-lines-reach-catalog-results-page_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB8ProvenanceFooterPrintsIndexerURLVerbatim -v`: the page reads `Indexed by http://ops:s3cret@127.0.0.1:1/graphql/query`.

</details>

Not posted: same defect as `gno.land/pkg/gnoweb/indexer/client.go:115`, which one edit to `URL()` closes; its repro is folded there.

## gno.land/pkg/gnoweb/indexer/client.go:115 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L115) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/client.go#L115) · Warning
`URL()` returns `-indexer-url` as configured, so a basic-auth `user:password` or a query-string key reaches every anonymous reader through the [results page footer](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L65) and the [`/$search&json` response](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/json.go#L92). The URL is the only place such a credential fits, since [`GNOWEB_INDEXER_TOKEN`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L195) only sends `Bearer`.

<details><summary>repro</summary>

The first test finds `hunter2` in both the HTML and the JSON response; the second, with the indexer down, reads `Indexed by http://ops:s3cret@127.0.0.1:1/graphql/query`.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b5_userinfo_probe_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b5-lines-reach-catalog-url-userinfo_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB5IndexerURLUserinfoIsPublished -count=1 -v
rm gno.land/pkg/gnoweb/feature/omnisearch/b5_userinfo_probe_test.go
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b8-lines-reach-catalog-results-page_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b8-lines-reach-catalog-results-page_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB8ProvenanceFooterPrintsIndexerURLVerbatim -v
```

[`indexerStatus`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L229) is the only caller of `URL()`, and Go's HTTP client sends the userinfo of the URL as working basic-auth credentials.

</details>

## gno.land/pkg/gnoweb/indexer/client.go:135 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L135) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/client.go#L135) · Warning
This condition counts a caller's own `context.DeadlineExceeded` as an indexer failure, so three concurrent callers whose deadline expires before the answer open the breaker for every reader for 30s. Omnibar lookups trip the breaker through the 3s [`jsonTimeout`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L23) against the [4s client timeout](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L45), and deploys scans through a [`recent()` band](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/queries.go#L301) outliving the caller's deadline.

<details><summary>repro</summary>

The first test reads `indexer queries seen=3, says unavailable=true` on the results page after three omnibar lookups against an indexer answering in 3.3s; the second simulates a 1.5s band, and after three concurrent `Deploys`, `RecentByAddress` returns `ErrUnavailable`. Real tx-indexer latency on a 128k to 1M block band is the measurement still open.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_breaker_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b3-lines-reach-catalog-omnibar-trips-breaker_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3OmnibarTripsBreaker' -count=1 -v
curl -fsSL -o gno.land/pkg/gnoweb/indexer/zz_window_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b4-lines-reach-catalog-recent-window.go
go test ./gno.land/pkg/gnoweb/indexer/ -run TestZZThreeSlowScansOpenTheBreaker -v
```

The breaker opens after [`breakerThreshold`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L53), 3, failures and stays open for [`breakerCooldown`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L54), 30s. One lookup at a time does not open it: the tip fetch of `indexerStatus` succeeds after each search and resets the count.

</details>

## gno.land/pkg/gnoweb/indexer/queries.go:277 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/queries.go#L277) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/queries.go#L277) · Warning
`recent()` stops after [`maxWindowSteps`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/queries.go#L248), 4 windows or about 1.17M blocks, and returns what it found with a nil error. Deploy history then reads empty, with no truncation notice, for any package deployed further below the tip.

<details><summary>repro</summary>

Expected an error or a truncation signal; 0 rows and a nil error come back.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/indexer/zz_window_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b4-lines-reach-catalog-recent-window.go
go test ./gno.land/pkg/gnoweb/indexer/ -run TestZZDeploysMissesAnythingPastTheFourthWindow -v
```
</details>

## SKIP gno.land/pkg/gnoweb/indexer/queries.go:301 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/queries.go#L301) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/queries.go#L301) · Warning
This `c.Query` fails with `context.DeadlineExceeded` when a wide window outlives the caller's deadline, and the shared breaker counts that failure. Three concurrent deploys scans thus block every visitor's indexer searches for 30s, and real tx-indexer latency on a 128k to 1M block window is the measurement still open.

<details><summary>repro</summary>

The test simulates a 1.5s window; after three concurrent `Deploys`, `RecentByAddress` returns `ErrUnavailable`.

```sh
curl -fsSL -o gno.land/pkg/gnoweb/indexer/zz_window_repro_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b4-lines-reach-catalog-recent-window.go
go test ./gno.land/pkg/gnoweb/indexer/ -run TestZZThreeSlowScansOpenTheBreaker -v
```
</details>

Not posted: same defect as `gno.land/pkg/gnoweb/indexer/client.go:135`, which one edit to the breaker condition in `Client.Query` closes; its repro is folded there.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/feature.go:93 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/feature.go#L93) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/feature.go#L93) · Missing test
Missing test: no test reaches the `singleflight` in `Handler.doc`. Sharing one fetch between callers, a deadline expiring while the detached fetch continues, and a panicking `Doc` are all unpinned.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/handler_test.go:155 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler_test.go#L155) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/handler_test.go#L155) · Missing test
Missing test: the rate-limit test covers only the JSON path, so the page branch of `writeRateLimited`, a plain-text 429 with a nil view, is never exercised.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/query_test.go:94 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/query_test.go#L94) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/query_test.go#L94) · Missing test
Missing test: the two bounds in `ParseQuery` are tested only one past the limit, so a `>` to `>=` slip in either stays green.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/search_test.go:380 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/search_test.go#L380) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/search_test.go#L380) · Missing test
Missing test: `TestDiscoveryNarrowsByAuthor` asserts only that nothing leaks, so an author-only query returning no groups at all passes.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/search_test.go:398 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/search_test.go#L398) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/search_test.go#L398) · Missing test
Missing test: `TestDiscoveryNarrowsByKind` passes with the `is:` filter deleted, since no package in its fixture matches "blog" and `discover` drops empty groups.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/handler_http.go:1354 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_http.go#L1354) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/handler_http.go#L1354) · Missing test
Missing test: the `search` arm of `isFeatureJSONRequest` has no test.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/handler_search_test.go:57 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_search_test.go#L57) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/handler_search_test.go#L57) · Missing test
Missing test: no directory test asserts `Truncated` at `searchPathLimit`, or that a cancelled first caller leaves the callers sharing its fetch answered.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## SKIP gno.land/pkg/gnoweb/indexer/client_test.go:152 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client_test.go#L152) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/client_test.go#L152) · Missing test
Missing test: no test pins the breaker exemptions for `ErrTooLarge` and `context.Canceled`, and only `ErrNotFound` has a breaker test.

Not posted: PLAUSIBLE Missing test on the finder's read; the mutation that would show it was not run.

## gno.land/adr/pr6231_gnoweb_indexer.md:133 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/adr/pr6231_gnoweb_indexer.md?plain=1#L133) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/adr/pr6231_gnoweb_indexer.md#L133) · Nit
Nit: this line says `is:` cuts a discovery search to one `ListPaths` call. [`discover`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L25) always calls `Directory.Paths`, and `is:` only [skips a group](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L46), after both kinds are fetched.

<details><summary>repro</summary>

```
sed -n 25,46p gno.land/pkg/gnoweb/feature/omnisearch/discover.go
```
`Paths` is called unconditionally, before the `hasIs` filter.
</details>

## gno.land/adr/pr6231_gnoweb_indexer.md:264 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/adr/pr6231_gnoweb_indexer.md?plain=1#L264) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/adr/pr6231_gnoweb_indexer.md#L264) · Nit
Nit: `RateLimitConfig.TrustedProxies` was already applied at the merge base, where `-trusted-proxies` fed it and [`feature/state/handler.go`](https://github.com/gnolang/gno/blob/87f0357fe2b373476bb92a26d59833402b9916d3/gno.land/pkg/gnoweb/feature/state/handler.go#L24) applied it through `extractIP`. `AllowRequest` moves that rule into the limiter without changing behaviour.

<details><summary>repro</summary>

```
grep -n 'extractIP\|TrustedProxies' <base>/gno.land/pkg/gnoweb/feature/state/handler.go <base>/gno.land/pkg/gnoweb/handler_http.go
grep -n trusted-proxies <base>/gno.land/cmd/gnoweb/main.go
```
</details>

## gno.land/cmd/gnoweb/main.go:311 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/cmd/gnoweb/main.go#L311) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/cmd/gnoweb/main.go#L311) · Nit
Refactor: `splitAndTrim` repeats the trim and the empty-entry skip that [`ParseTrustedProxies`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/state/ratelimit.go#L49-L52) does already, so a plain split gives the same networks and matches [`TrustedPaths`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/cmd/gnoweb/main.go#L340).

```suggestion
		appcfg.StateRateLimitTrustedProxies = strings.Split(cfg.trustedProxies, ",")
```

Then delete `splitAndTrim`.

<details><summary>repro</summary>

```
sed -n 43,52p gno.land/pkg/gnoweb/feature/state/ratelimit.go
```
The output shows `strings.TrimSpace` and the empty-entry skip.
</details>

## gno.land/pkg/gnoweb/components/layout_header.go:177 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/components/layout_header.go#L177) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/components/layout_header.go#L177) · Nit
Nit: `data.RealmURL.Path` is the alias target after the [rewrite in the handler](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_http.go#L309-L310), while the JS controller scopes from `window.location.pathname`. On an alias page such as the home page, a search is scoped to the alias realm without JS and chain-wide with it.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/zz_alias_search_scope_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b10-lines-removed-reach-catalog-alias-search-scope_test.go
go test ./gno.land/pkg/gnoweb/ -run TestAliasHeaderSearchScope -v
```
</details>

## gno.land/pkg/gnoweb/feature/omnisearch/discover.go:21 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L21) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L21) · Nit
Nit: a bare word under [`MinTermLen`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/query.go#L23), or an `is:` value naming no kind such as `is:pkg`, returns no group, so the page says "Nothing matched." while matching paths exist.

<details><summary>repro</summary>

Copy `tests/b8-lines-reach-catalog-results-page_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB8(ShortBareWord|UnknownIsKind)' -v`: "Nothing matched." in both.

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/handler.go:183 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L183) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L183) · Nit
Nit: a `Bare` selector gets an empty term even when typed with a value, so `imports:json` lists every import with no hint that `json` was dropped.

## gno.land/pkg/gnoweb/feature/omnisearch/handler.go:220 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L220) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L220) · Nit
Nit: `g.Source == SourceIndexer` also counts a group carrying only a [pre-flight error](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/handler.go#L141-L154). `content:abc` or an unscoped `activity` then prints "Some results above come from an indexer" over zero results.

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go:150 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L150) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L150) · Nit
Nit: `func:` tags every top-level `/r/` function except `Render` as an action, unexported helpers included. Non-crossing getters that [`MsgCall` panics on](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/sdk/vm/keeper.go#L1256) get the tag too, so the action link targets functions the help page omits.

<details><summary>repro</summary>

Copy `tests/b9-lines-reach-catalog-resolve-chain_test.go` into `gno.land/pkg/gnoweb/feature/omnisearch/` and run `go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB9ActionTagOnlyOnCallableFuncs -v`: `helper` and `GetBoard` carry the action tag.

</details>

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go:224 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L224) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_chain.go#L224) · Nit
Nit: the `matches` call on the term never filters, because `selectorFor` hands the `Bare` selector `imports` an empty term. The call reads as if `imports:x` narrows the list.

## SKIP gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go:214 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L214) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go#L214) · Nit
Nit: `tx.Messages[0].Path()` titles and links every activity row from the first message, so a transaction touching this package in a later message shows another package's title and link.

Not posted: PLAUSIBLE Nit; whether tx-indexer matches the `RecentByPackage` predicate on any message element is unchecked.

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go:91 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L91) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L91) · Nit
Nit: `hits` is appended in goroutine completion order and `capResults` only truncates, so `render:` results, and which ones survive the cap, change between reloads.

## gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go:141 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L141) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/resolve_render.go#L141) · Nit
Nit: `snippetAround` slices at byte offsets, so a multi-byte rune at either edge is split and the snippet shows U+FFFD.

## gno.land/pkg/gnoweb/feature/omnisearch/search_test.go:139 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/search_test.go#L139) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/search_test.go#L139) · Nit
Test: `idx` is already an `Indexer` interface, so a nil `*mockIndexer` arrives non-nil and passes `idx != nil`, and this guard cannot stop the typed-nil trap its comment names.

## gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html:41 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L41) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L41) · Nit
Nit: `HasResults` ignores groups carrying only an `Err`, so a failed backend prints "Could not answer: ..." and "Nothing matched." on the same page.

## gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html:63 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L63) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/templates/page.html#L63) · Nit
Nit: `indexerStatus` marks the indexer used on `g.Source` alone, so with the indexer down this footer claims results above came from it on a page with none.

## SKIP gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:295 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L295) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L295) · Nit
Nit: any 404 from `$search&json` sets `searchAvailable = false`, and the [only JSON 404 on that route](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/handler_http.go#L322) is "invalid path". On an invalid-path page, Enter on free text goes to `/<text>` instead of the results page.

Not posted: PLAUSIBLE Nit; no concrete path both fails `ParseFromURL` and renders the header bar was found.

## gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:298 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L298) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L298) · Nit
Nit: `if (!res.ok) return null` discards the `{error}` body the server writes for a 429 or a 400. A refused qualified search falls back to the local path filter, and the dropdown reads "No results".

<details><summary>repro</summary>

At `/r/m429/x`, `/r/m400/x` and `/r/m200/x`, type `author:alice` and read `#omnisearch-results` after 1.5s.

```sh
python3 tests/b7-lines-removed-reach-catalog-dropdown-silent-fallback.py gno.land/pkg/gnoweb 8765
```
</details>

## gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts:389 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L389) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/frontend/js/controller-searchbar.ts#L389) · Nit
Nit: the full-results item appended here makes `this.items` non-empty, so the `this.items.length === 0` check never fires. A qualified query matching nothing shows only the link, never "No results".

## gno.land/pkg/gnoweb/indexer/client.go:231 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L231) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/client.go#L231) · Nit
Nit: this line returns nil on a 200 JSON body with neither `data` nor `errors`, the answer of an `-indexer-url` pointing at the wrong endpoint. `LatestBlockHeight` then reads 0, `TxByHash` says not found, and nothing is logged.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/indexer/b5-lines-reach-catalog-breaker_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b5-lines-reach-catalog-breaker_test.go
go test ./gno.land/pkg/gnoweb/indexer/ -run TestB5EnvelopeWithoutDataReportsSuccess -count=1 -v
```

</details>

## gno.land/pkg/gnoweb/app.go:195 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/app.go#L195) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/app.go#L195) · Suggestion
Suggestion: parse `cfg.IndexerURL` at startup, since [`indexer.New`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L94) stores it unparsed. A value such as `localhost:8546/graphql` parses with scheme `localhost`, and every indexer search fails behind a startup log calling the indexer enabled.

## gno.land/pkg/gnoweb/feature/omnisearch/templates/_results.html:13 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/templates/_results.html#L13) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/feature/omnisearch/templates/_results.html#L13) · Suggestion
Suggestion: render `10+` or a notice on a discovery group capped at [`maxDiscoverResults`](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/feature/omnisearch/discover.go#L11), which shows count 10 for 25 matches with nothing saying more matched.

<details><summary>repro</summary>

```sh
curl -fsSL -o gno.land/pkg/gnoweb/feature/omnisearch/b8-lines-reach-catalog-results-page_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6231-add-optional-indexer-backend/2-dd40ef3dc/tests/b8-lines-reach-catalog-results-page_test.go
go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run TestB8DiscoveryCountReadsTheCapNotTheMatches -v
```

</details>

## SKIP gno.land/pkg/gnoweb/indexer/client.go:135 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L135) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/client.go#L135) · Suggestion
Suggestion: exempt `ErrResponseTooLarge` from `record()` as `ErrTooLarge` is. Three such answers, or three expired caller deadlines, open the shared breaker for every reader for 30 s.

<details><summary>repro</summary>

`go test ./gno.land/pkg/gnoweb/indexer/ -run 'TestB5(OversizedResponseOpensBreaker|CallerDeadlineOpensBreaker)' -count=1 -v` with `tests/b5-lines-reach-catalog-breaker_test.go` copied in: the 4th call gets `ErrUnavailable`.

</details>

Not posted: PLAUSIBLE Suggestion; the oversized case is documented as deliberate and no real band over 8 MiB was produced, and the Warning on this line carries the caller-deadline case.

## SKIP gno.land/pkg/gnoweb/indexer/client.go:217 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/indexer/client.go#L217) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/indexer/client.go#L217) · Suggestion
Suggestion: log the error messages of a non-200 JSON body. A non-200 answer logs only status, size and content type, so the operator cannot see why the indexer rejected a query.

Not posted: PLAUSIBLE Suggestion; whether tx-indexer answers validation errors with 422 is unverified.

## SKIP gno.land/pkg/gnoweb/realm_directory.go:57 [gh](https://github.com/gnolang/gno/blob/dd40ef3dc9b92a3df888bb9e70dc21871dcc192a/gno.land/pkg/gnoweb/realm_directory.go#L57) · [↗](../../../../../.worktrees/gno-review-6231/gno.land/pkg/gnoweb/realm_directory.go#L57) · Suggestion
Suggestion: measure the `/search.json` body at `searchPathLimit` 10000, since every omnibar downloads it on the first keystroke and it can reach 20000 paths.

Not posted: UNVERIFIED Suggestion; no run sized the body at 10000 paths.
