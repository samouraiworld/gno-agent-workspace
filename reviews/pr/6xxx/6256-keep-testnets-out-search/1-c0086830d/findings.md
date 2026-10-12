# Findings in posting order, from round assemble: 3 to post, 1 SKIP, 2 refuted kept out

## Body

- Suggestion: gno.land/pkg/gnoweb/realm_directory.go:30 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/realm_directory.go#L30), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: removed
  TL;DR: /search.json reads a qpaths listing capped at 1000, since ListPaths never forwards its limit; this predates the branch and gnolang/gno#6231 already forwards it
  Check: read client.go:230-234 and gno.land/pkg/sdk/vm/handler.go:199-215; search open PRs for the fix
  Details: client.go:233 calls c.query(ctx, qpath, []byte(prefix), 0) and drops limit; vm pathsLimit defaults to 1_000 without ?limit= (handler.go:200,209). realm_directory.go and handler_search.go are unchanged by this branch (/search.json is mounted at the merge base, app.go:223), so the defect predates it. An open fix exists: #6231's client.go appends '?limit=' when limit > 0. Not posted on this PR.
  Evidence: handler.go:200 'const defaultLimit = 1_000', :209 'limit := defaultLimit'; gh pr diff 6231: '+	if limit > 0 {' '+		qpath += "?limit=" + strconv.Itoa(limit)'
- SKIP Suggestion: misc/deployments/home-alias/docker-compose.yml:8 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/misc/deployments/home-alias/docker-compose.yml#L8), outside the diff, so no anchor
  State: PLAUSIBLE, band: Suggestion, angle: reach
  TL;DR: home-alias, test2 and test3 compose files also run gnoweb without --no-index; whether any still serves a public host is not in the repo
  Check: grep -n 'no-index' over the three files; whether they still serve a public host
  Details: The flag is absent from all three by grep. home-alias is marked dev only; test2 and test3 are retired testnet names whose liveness this round cannot settle. Still to run: ask which of them are deployed.
  Evidence: grep: gnoweb services at test2:35, test3:38, home-alias:4, none carrying no-index
- Suggestion: misc/loop/docker-compose.yml:55 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/misc/loop/docker-compose.yml#L55), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: reach
  TL;DR: the staging.gno.land gnoweb command (misc/loop/docker-compose.yml:55-61) has no --no-index, so staging stays indexed after the merge
  Check: grep -n 'no-index' misc/loop/docker-compose.yml returns nothing at c0086830d
  Details: The gnoweb service runs gnoweb:master with --bind, --remote, --chainid=staging, --with-analytics, --faucet-url and --help-remote, routed from Host(`staging.gno.land`). NoIndex defaults to false (app.go:66-68), so the redeployed master image serves staging without X-Robots-Tag. Extends aeddi section 4 bullet 1 by naming the file. The file is outside the diff: a Body bullet.
  Evidence: grep -n 'no-index' misc/loop/docker-compose.yml: no output; sed -n 51,67p shows the command block and the traefik Host rule

## gno.land/pkg/gnoweb/crawl.go:31 [gh](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L31) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: under -no-index robots.txt drops the Disallow for /search.json and /status.json, the two node-querying JSON endpoints, though neither carries a page whose noindex a crawler must see
Check: read crawl.go:27-34; go test ./gno.land/pkg/gnoweb/ -run TestHandlerRobotsTXT
Details: crawl.go:31 assigns the body outright instead of appending Allow to the default, so a -no-index deployment invites crawlers onto /search.json (two vm/qpaths queries) and /status.json (a node status query). The test's rationale 'A Disallow would hide the noindex header' holds for pages; a JSON endpoint is never indexed. Not a regression: the merge base served no robots.txt.
Evidence: crawl.go:28 'body := "User-agent: *\nDisallow: /search.json\nDisallow: /status.json\n"', :31 'body = "# Not indexed: ...\nUser-agent: *\nAllow: /\n"'; TestHandlerRobotsTXT/noindex_still_allows_crawling PASS pins that body (crawl_test.go:47)
