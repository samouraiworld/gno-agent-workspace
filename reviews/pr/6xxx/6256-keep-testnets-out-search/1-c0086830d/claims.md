# Claims: gnolang/gno#6256 round 1, c0086830d, claude-opus-5-5, solo review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 2 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/crawl.go:31 | read crawl.go:27-34; go test ./gno.land/pkg/gnoweb/ -run TestHandlerRobotsTXT | crawl.go:28 'body := "User-agent: *\nDisallow: /search.json\nDisallow: /status.json\n"', :31 'body = "# Not indexed: ...\nUser-agent: *\nAllow: /\n"'; TestHandlerRobotsTXT/noindex_still_allows_crawling PASS pins that body (crawl_test.go:47) |  | warm |
| 1001 | CONFIRMED | Suggestion | misc/loop/docker-compose.yml:55 | grep -n 'no-index' misc/loop/docker-compose.yml returns nothing at c0086830d | grep -n 'no-index' misc/loop/docker-compose.yml: no output; sed -n 51,67p shows the command block and the traefik Host rule |  |  |
| 1002 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/realm_directory.go:30 | read client.go:230-234 and gno.land/pkg/sdk/vm/handler.go:199-215; search open PRs for the fix | handler.go:200 'const defaultLimit = 1_000', :209 'limit := defaultLimit'; gh pr diff 6231: '+ if limit > 0 {' '+ qpath += "?limit=" + strconv.Itoa(limit)' |  |  |
| 2001 | PLAUSIBLE | Suggestion | misc/deployments/home-alias/docker-compose.yml:8 | grep -n 'no-index' over the three files; whether they still serve a public host | grep: gnoweb services at test2:35, test3:38, home-alias:4, none carrying no-index |  |  |
| 1 | REFUTED | Suggestion | contribs/gnodev/setup_web.go:30 | grep -rn 'NoIndex\\|no-index' contribs/gnodev; read misc/loop/docker-compose.yml:51-67 for what serves staging.gno.land | misc/loop/docker-compose.yml:52 image: ghcr.io/gnolang/gno/gnoweb:master serves staging.gno.land (:67), and cmd/gnoweb carries -no-index |  |  |
| 1003 | REFUTED | Suggestion | gno.land/pkg/gnoweb/crawl.go:120 | grep sitemap in the merge base's app.go; read aeddi's review section 2 | aeddi-review.md:37 names the uncached per-request queries; merge base app.go mounts no /sitemap.xml |  | warm |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 3 files, warm 1/2 confirmed over 9 files, cold 0/0 confirmed over 8 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | gno.land/pkg/gnoweb/crawl.go:125 | rpcClient.ListFiles has no 'not available' string fallback (File() at client.go:195 has one), so on a node returning an untyped not-found for qfile a missing alias realm would 502 the whole sitemap, against the comment's 'every node version answers it' | git log -S'InvalidPackageError{}, "package %q is not available"' -- gno.land/pkg/sdk/vm/keeper.go -> 59595d429 2025-07-16; git tag --contains 59595d429 -> chain/gnoland1.0, chain/gnoland1.1, chain/mainnet, chain/pearl, chain/sapphire: every chain tag answers the typed InvalidPackageError that rpcClient.query maps to ErrClientPackageNotFound |
| lines | gno.land/pkg/gnoweb/crawl.go:82 | sitemap lists the alias key while the page's canonical might name the resolved realm path, so the sitemap would publish non-canonical URLs | handler_http.go:336 setHeadMetadata(&indexData, headURL, ...) builds the canonical from headURL; handler_http_test.go:2443 asserts `<link rel="canonical" href="https://gno.land/about" />` for the GnowebPath alias /about -> /r/gnoland/pages:p/about |
| reach | gno.land/pkg/gnoweb/app.go:254 | gnolang/gno#6229's head might register its own /robots.txt, so this PR's mux.Handle("/robots.txt") would panic on duplicate pattern after the rebase | git grep -n 'robots.txt\\|sitemap\\|mux.Handle(' ea1c620f7 -- gno.land/pkg/gnoweb/app.go -> routes /, /faucet, chroma, assets, /status.json, /liveness, /ready, /search.json only; no /robots.txt or /sitemap.xml |
| lines | gno.land/pkg/gnoweb/handler_http.go:258 | under -no-index a later HeadData reassignment or NoIndex=false write could leave the meta at 'index, follow' while the header says noindex | grep -rn 'HeadData = \\|\.NoIndex\b' gno.land/pkg/gnoweb (non-test) -> only handler_http.go:250 struct init, :258 seed, :411 and :1469 '= true'; no reassignment or false write; every IndexLayout render (handler_http.go:298, 364, 423) uses Get's indexData |
| reach | gno.land/pkg/gnoweb/crawl.go:120 | ListFiles gets '/r/...' without the chain domain, so every alias realm would read as absent on a real node. | sed -n 213-217p client.go: 'pkgPath := strings.Trim(path, "/")' 'fullPath := fmt.Sprintf("%s/%s", c.domain, pkgPath)'. The domain is prepended, the same way the GetSourceView and GetDirectoryView callers pass gnourl.Path. |
| reach | gno.land/pkg/gnoweb/crawl.go:122 | A parked (inert) alias realm answers vm/qfile on a real node, so the sitemap would list a page that shows 'pending approval'. The test relies on the mock alone. | sdk/vm/keeper.go:1942 QueryFile reads 'store.GetMemPackageAll(dirpath)', the live key space, while inert packages sit in GetInertPackage. keeper_inert.go:592-595: 'vm/qpkg_json and vm/qfile both answer "package not found"'. That maps to ErrClientPackageNotFound (client.go:380-386), so the mock matches the chain. |
| reach | gno.land/pkg/gnoweb/app.go:257 | A response written outside NewRouter's mux would miss the -no-index X-Robots-Tag. | main.go:438-465: SecureHeadersMiddleware only calls w.Header().Set(...) and then next.ServeHTTP(w, r), with no early return. main.go:348-367 mounts nothing beside 'app', and grep for gnoweb.NewRouter across the tree finds only main.go:348. |
| removed | gno.land/cmd/gnoweb/main.go:189 | A doc or comment still names the old -noindex flag after the f448ac406 rename. | grep -rn -- '-noindex\b\|-no-index' over every .md and over the code finds only the '-no-index' spellings at app.go:249, handler_http.go:257 and handler_http_test.go:2466. |
| reach | gno.land/pkg/gnoweb/app.go:254 | Rebasing onto gnolang/gno#6229's head would register /robots.txt or /sitemap.xml twice and panic in ServeMux at startup. | git grep -n 'robots.txt\|sitemap' ea1c620f7 -- gno.land/pkg/gnoweb gno.land/cmd/gnoweb (non-test) finds only a comment at page_kind.go:46 ('anything that lists pages, such as a sitemap'), with no mux registration. |

## Completeness

- Angles: reach and lines came back with the deployment-side candidates; the reflector swept every in-tree gnoweb compose file for the same command shape and filed home-alias, test2 and test3 as 2001 (PLAUSIBLE, liveness not in the repo).
- Catalog: gnoweb tooling, outside the catalog's gating. Determinism walked anyway: `sitemapAliases` iterates the `byRealm` map, so which realm's error reaches the log varies, but the returned paths go through `slices.Sort` (crawl.go:129). Global mutable state: no package var is written. Error handling: the two `_ =` are on `ResponseWriter.Write`.
- Siblings of edited tests: `TestHandlerRobotsTXT` pins all three bodies with a whole-string `assert.Equal`; the live-header-map shape in crawl_test.go is aeddi's point and not refiled.
- Already named and dropped: 1003 (aeddi section 2, sitemap bullet 3). 1001 extends aeddi section 4 bullet 1 by naming misc/loop/docker-compose.yml:55-61.
- 1002 predates the branch; gnolang/gno#6231 (same author, open) forwards `?limit=` in `rpcClient.ListPaths`, so no issue draft.
- Unread: a custom `GnowebPath` alias whose value carries `?` passes the query into the realm handed to `ListFiles`; none of `DefaultAliases` does.

## Retro

- What failed: nothing died and no cap was hit. Two of six candidates were refuted. One repeated aeddi's sitemap point despite the threads note. The other, the staging flag, came back confirmed as 1001, and the parent moved it to SKIP, since the PR description already hands that flag to aeddi. The suggested pipeline was overridden to solo: only the branch's five own commits, about 450 of the diff's 1613 lines, were in scope, and the rest is gnolang/gno#6229 as of c736a5dbd.
- What worked: 2 finders and one judge-writer, 3 agents, 87 turns, 117k output, about $7.06 measured against about $20 projected. The one posted finding, at `crawl.go:31` in the warm tier, is new beside aeddi's review.
- Hit rate per tier: hot 0 of 0 rows, warm 1 of 1, and cold 0 of 0. The remaining rows sit outside the diff: deployment compose files and `client.go`.
- Upgrade: setup cannot scope a branch stacked on an open pull request whose head moved on. The base would be the parent's state the branch carries, merged with master, and both builds of it conflicted, reverting the own commits and merging c736a5dbd with master. Logged in TODO.md.
