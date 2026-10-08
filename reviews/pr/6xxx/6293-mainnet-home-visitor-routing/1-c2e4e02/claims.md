# Claims: gnolang/gno#6293 round 1, c2e4e0207, claude-opus-5-5, solo review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/redirect.go:10 | curl -s -o /dev/null -w '%{http_code}' https://gno.land/newsletter (302) and https://gno.land/r/gnoland/pages:p/newsletter (404); grep -c list-manage misc/deployments/home-alias/home.mainnet.md (2) | curl: gno.land/newsletter 302 then 404 with -L; onyx.testnets.gno.land/newsletter 302 then 404; /r/gnoland/pages:p/newsletter 404 on both; Mailchimp subscribe URL 200; grep -c list-manage: home.mainnet.md 2, home.testnet.md 2; redirect.go:10 `var Redirects = map[string]string{` |  | warm |

Hit rate per tier, from the rows above: hot 0/0 confirmed over 0 files, warm 1/1 confirmed over 2 files, cold 0/0 confirmed over 3 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| claims | misc/deployments/home-alias/home.mainnet.md:1 | body claims every link returns 200 on its network | curl -L each ](...) target, relative ones on https://gno.land and https://onyx.testnets.gno.land: every row 200, including /r/gnoland/blog$source, /r/gov/dao, /r/gnops/valopers, /p/nt, /partners, faucet, gnoswap.io |
| removed | gno.land/pkg/gnoweb/redirect.go:10 | removed redirects might still be linked or their targets might be live | curl gno.land: 404 /r/gnoland/pages:p/newsletter, 404 /r/demo/boards:gnolang/3; git grep -E '/newsletter\b\|boards:gnolang/[36]\b\|pages:p/newsletter' outside examples/quarantined: no hits; gh search code '"gno.land/newsletter"': no hits; only Redirects readers are app.go:179 and contribs/gnobro/pkg/browser/utils.go:21, and no test lists the removed paths |
| lines | misc/deployments/home-alias/home.mainnet.md:30 | in-page anchors #live-on-mainnet, #on-this-testnet, #explore-packages-and-realms and #community might not match the generated heading ids | rendered both files with goldmark WithAutoHeadingID plus md.NewGnoExtension, as render_config.go:36 does: every href="#..." resolves to a heading id (true x3 per file), 5/4 column blocks of 3 columns each, the alert renders, and no raw tag leaks; the live gno.land page shows id="socials" on a heading inside gno-columns |
| reach | misc/deployments/home-alias/home.testnet.md:23 | the testnet's first CTA, faucet.gno.land, might not serve onyx | faucet.gno.land/assets/index-DalgKuER.js contains faucets=[{name:"Onyx Faucet",chain_id:"onyx-1" and url:"https://faucet.onyx.testnets.gno.land |
| claims | misc/deployments/home-alias/home.testnet.md:86 | Boards note says creating boards is not open to the public, while the live realm shows 'Be the first to create a new board!' | examples/gno.land/r/gnoland/boards2/v0/boards.gno:57 perms.AddRole(RoleAdmin, PermissionBoardCreate), and public.gno:145 gates CreateBoard on it; the realm's invite text is its generic empty-state message |
| claims | misc/deployments/home-alias/pages/ecosystem.md:33 | ecosystem page now says GnoSwap is live on gno.land mainnet | curl https://gno.land/r/gnoswap/pool -> 200, https://gno.land/r/gnoswap -> 200 |
| claims | misc/deployments/home-alias/home.mainnet.md:132 | body says r/demo holds only two packages on mainnet and /ecosystem has five entries | curl gno.land/r/demo hrefs: /r/demo/defi/grc20factory, /r/demo/profile (2); grep -c '^### ' pages/ecosystem.md -> 5 |
| claims | gno.land/pkg/gnoweb/redirect.go:16 | body says go test ./gno.land/pkg/gnoweb/ passes after the map edit | gofmt -l redirect.go: clean; git grep 'Redirects' in *.go: no test reads the map, app_test.go:219 lists only /game-of-realms, /getting-started, /blog, /boards |
| removed | examples/gno.land/r/gnoland/home/home.gno:56 | the on-chain r/gnoland/home sibling still opens on Boards and keeps the r/demo card | misc/deployments/home-alias/README.md:15 'Staging is not concerned: it serves r/gnoland/home from the chain, with no alias'; mainnet and onyx serve the alias files |
| catalog | gno.land/pkg/gnoweb/redirect.go:10 | invariant catalog classes | the diff touches no .gno code, GnoVM or stdlib; the Redirects package var is not mutated by the diff, so no catalog class applies |

## Completeness

- Links, rerun by the judge: every relative link of both files on gno.land and onyx.testnets.gno.land answers 200 with `curl -L` (`/blog` through its 302), every external link 200 (docs.gno.land and twitter.com through a 301). The removed targets answer 404 on both networks; `/r/demo/boards` also 404s, so no live boards path was lost.
- In-page anchors: goldmark v1.8.2 `ids.Generate` lowercases, maps spaces to `-` and drops other punctuation, so `Live on mainnet`, `On this testnet`, `Explore packages and realms` and `Community` give the four fragment targets; no other heading in either file yields the same slug, so no `-1` suffix. The judge's own render was not run: its scratch worktree took the disk under 1024 MB and was removed; the finder's render row above stands.
- Catalog: the diff touches no `.gno` code, GnoVM or stdlib, so the project delta skips the catalog; no class applies.
- Siblings swept by shape: the diff edits no test or fixture. The `Redirects` map's test-side sibling, `TestAnalytics` at app_test.go:219-223, lists none of the removed keys. Its `/boards` route under `// Redirects` is a key of the map at neither base nor head and answers 400 on both networks: it predates the diff, sits outside it and is not filed. The remaining map targets, `/r/gnoland/blog`, `/contribute`, `/partners`, `/gnolang`, `/start`, answer 200 on both networks, so no other dead entry is left.
- Removed redirects referenced elsewhere: `git grep` for `pages:p/newsletter`, `"/newsletter`, `gno.land/newsletter` and `boards:gnolang/6` at head: no hit; `boards:gnolang/1` appears only as a permission string in `examples/quarantined/.../acl_test.gno`, not a link.
- Boards note: `/r/gnoland/boards2/v0` answers 200 on both networks, so "deployed" holds.

## Suggestion run

Row 1's Suggestion, applied in a scratch worktree at c2e4e02: a `"/newsletter"` entry in `Redirects` pointing at the Mailchimp subscribe URL, driven through `RedirectMiddleware` with `httptest`. With the entry, `/newsletter` answers 302 to the subscribe URL, `/blog` still 302 to `/r/gnoland/blog`, and `/r/gnoland/home` reaches the next handler. Without it, `/newsletter` reaches the next handler and the other two are unchanged. The scratch worktree was removed after the run.

## Retro

- What failed: nothing; one candidate, one Suggestion.
- What worked: the solo shape, 2 agents, about 260k subagent tokens and 15 minutes from the run notification, every link checked live on both networks.
