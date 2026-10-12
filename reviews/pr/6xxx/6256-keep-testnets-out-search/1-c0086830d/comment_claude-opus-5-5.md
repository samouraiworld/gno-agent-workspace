# Review: [#6256](https://github.com/gnolang/gno/pull/6256)
Posted: https://github.com/gnolang/gno/pull/6256#pullrequestreview-5487214035

Event: COMMENT
Verdict: APPROVE. No Warning survived in this branch's own five commits; the one posted finding is a Suggestion, the robots.txt body that drops its two JSON `Disallow` lines under the flag. aeddi's open point, that the branch must be reworked onto [#6229](https://github.com/gnolang/gno/pull/6229)'s current robots policy, stands and is not repeated here.
Model: claude-opus-5-5, solo review: one judge and writer over two finders' candidates
Overview: [overview](../overview.md)
Commit: c0086830daedaec5d152853345711e35e303be9a
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6256 c0086830d`
Open the code: [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a)
Round: 1. 2 finders, one reflector pass, 6 candidates (5 from the finders, 1 from the reflector), each judged by read or a named test run by an agent that was not its finder; 2 refuted, 1 plausible and skipped, 9 settled by the finders.

## Body

> AI review, claude-opus-5-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6256-keep-testnets-out-search/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6256-keep-testnets-out-search/1-c0086830d/claims.md) · Status: APPROVE

## gno.land/pkg/gnoweb/crawl.go:31 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L31) · [↗](../../../../../.worktrees/gno-review-6256/gno.land/pkg/gnoweb/crawl.go#L31) · Suggestion [posted](https://github.com/gnolang/gno/pull/6256#discussion_r4245250447)

Suggestion: under `-no-index` this assignment drops the `Disallow` lines for `/search.json` and `/status.json`, which query the node and are never indexed. Keeping both beside `Allow: /` leaves every page crawlable.

<details><summary>the test pinning the body</summary>

`TestHandlerRobotsTXT/noindex_still_allows_crawling` asserts the body without either `Disallow` line ([`crawl_test.go:47`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L47)):

```text
=== RUN   TestHandlerRobotsTXT/noindex_still_allows_crawling
--- PASS: TestHandlerRobotsTXT (0.00s)
ok  	github.com/gnolang/gno/gno.land/pkg/gnoweb	0.041s
```

</details>

## SKIP gno.land/pkg/gnoweb/client.go:233 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/client.go#L233) · [↗](../../../../../.worktrees/gno-review-6256/gno.land/pkg/gnoweb/client.go#L233) · Suggestion

Related suggestion: this call drops its `limit`, so `/search.json` gets [the node's default cap](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/sdk/vm/handler.go#L200) · [↗](../../../../../.worktrees/gno-review-6256/gno.land/pkg/sdk/vm/handler.go#L200) and loses `r/gnoland/*` once user realms fill it.

Not posted: the cap predates this branch, which leaves `/search.json` untouched, and gnolang/gno#6231 already forwards the limit.

## SKIP misc/loop/docker-compose.yml:55-61 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/misc/loop/docker-compose.yml#L55-L61) · [↗](../../../../../.worktrees/gno-review-6256/misc/loop/docker-compose.yml#L55) · Suggestion

Suggestion: the staging gnoweb command carries no `--no-index`, so staging.gno.land stays indexable once its image picks up this branch.

Not posted: the description already hands the flag on testnets and staging to aeddi.

## SKIP misc/deployments/home-alias/docker-compose.yml:4-10 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/misc/deployments/home-alias/docker-compose.yml#L4-L10) · [↗](../../../../../.worktrees/gno-review-6256/misc/deployments/home-alias/docker-compose.yml#L4) · Suggestion

Suggestion: three more compose files run gnoweb without `--no-index`: home-alias, `test2.gno.land` and `test3.gno.land`.

Not posted: whether any of the three still serves a public host is not in the repository.
