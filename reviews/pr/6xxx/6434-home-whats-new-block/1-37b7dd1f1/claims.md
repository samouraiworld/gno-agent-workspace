# Claims: gnolang/gno#6434 round 1, 37b7dd1f1, claude-opus-5-5, quick review

Round shape: 3 finders, no reflector, 11 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`home.testnet.md:61`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L61) | 10 | Warning | CONFIRMED | differs | posts |
| [`home.testnet.md:82`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L82) | 1 | Warning | CONFIRMED | differs | posts |
| [`whatsnew.sh:246`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L246) | 6 | Warning | CONFIRMED | differs | posts |
| [`whatsnew.sh:103`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L103) | 12 | Missing test | PLAUSIBLE |  | SKIP |
| [`whatsnew.sh:580`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L580) | 13 | Missing test | PLAUSIBLE |  | SKIP |
| [`whatsnew_test.sh:4`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew_test.sh#L4) | 14 | Missing test | PLAUSIBLE |  | SKIP |
| [`whatsnew_test.sh:223`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew_test.sh#L223) | 15 | Missing test | PLAUSIBLE |  | SKIP |
| [`home.testnet.md:3`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L3) | 2 | Nit | CONFIRMED | differs | posts |
| [`home.testnet.md:58`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L58) | 3 | Nit | CONFIRMED | differs | posts |
| [`whatsnew.sh:73`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L73) | 7 | Nit | CONFIRMED | differs | posts |
| [`whatsnew.sh:281`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L281) | 5 | Nit | CONFIRMED | differs | posts |
| [`whatsnew.sh:376`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L376) | 4 | Nit | PLAUSIBLE | differs | SKIP |
| [`whatsnew.sh:447`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L447) | 8 | Nit | CONFIRMED | differs | posts |
| [`whatsnew_test.sh:233`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew_test.sh#L233) | 11 | Nit | CONFIRMED | differs | posts |
| [`README.md:91`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/README.md#L91) | 9 | Suggestion | CONFIRMED | differs | posts |

### 10 · Warning · CONFIRMED · `misc/deployments/home-alias/home.testnet.md:61`

A visitor who follows the prose and adds and funds alice loses that key when the deploy block's first line re-adds it, and the deploy then fails

- Check: bash tests/b3-lines-reach-rollout-page-catalog-deploy-block-readds-key.sh from the head tree: builds gnokey from the tree, runs `gnokey add alice` twice on a pty with Enter presses, compares `gnokey list` addresses
- Observed: Reran the finder's artifact in a judge worktree at 37b7dd1f1 (gnokey built from that tree): `block line 1: gnokey add alice` / `funded: g1njegdnv5cu29untnw9v62fjjmadfcppql0h6h5` / `after: g1w0ejfwkfkmuy4h35tj3dsqpp0zud8lqg65j7ux` / `FUNDED KEY REPLACED`, rc=1. Read tm2/pkg/commands/utils.go:65-66 and tm2/pkg/crypto/keys/client/add.go:497-553: same name with a different address falls through to the Override prompt, default yes. Merge base: grep for `gnokey add` in misc/deployments/home-alias/ returns nothing; the base page has no deploy block.
- Test: [`tests/b3-lines-reach-rollout-page-catalog-deploy-block-readds-key.sh`](tests/b3-lines-reach-rollout-page-catalog-deploy-block-readds-key.sh)

### 1 · Warning · CONFIRMED · `misc/deployments/home-alias/home.testnet.md:82`

The onyx home's only troubleshooting step for a package that is not enabled points at a host that does not resolve

- Check: bash projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b1-removed-claims-page-gpao-status-host.sh misc/deployments/home-alias/home.testnet.md from the head tree: expect DoH Status 0 and an HTTP answer, observe Status 3 and curl_exit=6
- Observed: Reran the finder's artifact from the head tree. Output: 'DoH gpao.onyx.testnets.gno.land: {"Status":3,"A":[]}', rpc.onyx resolves (Status 0), 'GET .../status/gno.land/r/gnoland/home: http=000 curl_exit=6', 'tree mentions of the host outside the page: 0'. Bundle line 567 shows the sentence as an added line. grep for gpao in the base home-alias directory: no hit.
- Test: [`tests/b1-removed-claims-page-gpao-status-host.sh`](tests/b1-removed-claims-page-gpao-status-host.sh)

### 6 · Warning · CONFIRMED · `misc/deployments/home-alias/whatsnew.sh:246`

A week with no matching realm call aborts the activity refresh, so the page keeps showing last week's users, realms and ranking beside a fresh zero

- Check: WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash tests/b2-lines-reach-rollout-page-catalog-zero-call-week.sh: expect rc 0 and activity 0 0 for the empty week, observe rc=1 and yesterday's 1 1 rendered
- Observed: Reran the finder's artifact against the head script, which sources whatsnew.sh with graphql stubbed: 'week with one call: rc=0 activity=[1 1]', 'week with no call: rc=1 out=[] activity=[1 1]', rendered '**0** deployments this week / **1** active users this week / **1** realms used this week / 1. [demo/foo](/r/demo/foo) · 1'. Read lines 241-249. whatsnew.sh does not exist at the merge base.
- Test: [`tests/b2-lines-reach-rollout-page-catalog-zero-call-week.sh`](tests/b2-lines-reach-rollout-page-catalog-zero-call-week.sh)

### 12 · Missing test · PLAUSIBLE · `misc/deployments/home-alias/whatsnew.sh:103`

No check covers to_md, expand_links or html_to_text, the escaping the body's Safety section rests on

- Check: grep -nE 'to_md\|expand_links\|html_to_text' misc/deployments/home-alias/whatsnew_test.sh (no hit); then mutate line 111 to `gsub("(?<c>x)"; "\(.c)")` and run bash whatsnew_test.sh: expect PASS
- Observed: not run: no verifier reached it; the check is still to run

### 13 · Missing test · PLAUSIBLE · `misc/deployments/home-alias/whatsnew.sh:580`

No check covers the outage, lost-state and keep-previous paths: main's empty-block guard, refresh's failure branch, fill_block, the reload retry

- Check: grep -nE 'refresh\|fill_block\|reload\|nothing to show' misc/deployments/home-alias/whatsnew_test.sh (no hit); delete line 580-583 and run bash whatsnew_test.sh: expect PASS
- Observed: not run: no verifier reached it; the check is still to run

### 14 · Missing test · PLAUSIBLE · `misc/deployments/home-alias/whatsnew_test.sh:4`

No CI job runs whatsnew_test.sh, and ci-dir-misc excludes the .md whose deploy block it tests

- Check: grep -rn whatsnew_test .github (no hit); sed -n 10,15p .github/workflows/ci-dir-misc.yml
- Observed: not run: no verifier reached it; the check is still to run

### 15 · Missing test · PLAUSIBLE · `misc/deployments/home-alias/whatsnew_test.sh:223`

The gno stub accepts what the real `gno mod init` refuses, so the deploy-block checks pin the stub, not the block

- Check: go build ./gnovm/cmd/gno from the head tree; in an empty dir run `gno mod init "gno.land/r/<your-g1-addr>/myrealm"`: expect an invalid-char error; run a valid init twice: expect 'file already exists'
- Observed: not run: no verifier reached it; the check is still to run

### 2 · Nit · CONFIRMED · `misc/deployments/home-alias/home.testnet.md:3`

The testnet home says Onyx runs mainnet's code, but Onyx runs v1.8.0-rc.1 while mainnet runs v1.8.0

- Check: curl -fsSL https://rpc.onyx.testnets.gno.land:443/status \| jq .result.build_version; curl -fsSL https://rpc.gno.land:443/status \| jq .result.build_version: the copy holds only if they match or onyx is ahead
- Observed: curl rpc.onyx.testnets.gno.land/status: result.build_version "v1.8.0-rc.1"; curl rpc.gno.land/status: result.build_version "v1.8.0" (2026-11-19).

### 3 · Nit · CONFIRMED · `misc/deployments/home-alias/home.testnet.md:58`

The page asks for a key to be added and funded before the block that adds it, so a reader can end up signing with a second, unfunded alice

- Check: read home.testnet.md:28-82 in order as a fresh reader; on gnodev, add key k1, fund it, then run the two blocks as printed with <your-g1-addr> = k1's address: expect the deploy to land, observe that alice is unfunded
- Observed: home.testnet.md:18 `1. [Install gnokey](...) and add a key`; :58 `Add a key (`gnokey list` prints its g1 address), and fund that address at the [faucet]... Then write a realm:`; :61 `gnokey add alice`; :79 `alice` as the addpkg signer.

### 7 · Nit · CONFIRMED · `misc/deployments/home-alias/whatsnew.sh:73`

The mainnet home's X post reads 'CompleteGno.land' because tag stripping glues the word before a link to the link's replacement

- Check: STATE_DIR=$(mktemp -d) bash misc/deployments/home-alias/whatsnew.sh --source x \| grep -c 'CompleteGno'; and curl publish.x.com/oembed for that status to see the missing break
- Observed: curl publish.x.com/oembed?url=.../2123072056153759774 \| jq -r .html: `Upgrade: Complete<a href="https://t.co/UPaevcA79z">https://t.co/UPaevcA79z</a> mainnet`; that string through the html_to_text gsubs: `Upgrade: Completehttps://t.co/UPaevcA79z mainnet`; curl -w '%{redirect_url}' https://t.co/UPaevcA79z: `https://gno.land/`, which expand_links:86-87 rewrites to `Gno.land`.

### 5 · Nit · CONFIRMED · `misc/deployments/home-alias/whatsnew.sh:281`

The weekly deployments figure counts parked addpkg transactions that never went online, while the list beside it shows only live packages

- Check: WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash tests/b2-lines-reach-rollout-page-catalog-deployed-count-parked.sh (live, read-only, onyx RPC and indexer): expect the count to equal the live deployments, observe 31 counted and 26 online
- Observed: Reran the finder's artifact live and read-only against onyx: 'code_submission_policy: "inert"', 'deployments this week (page): 31', 'counted deployments online now: 26 of 31', not-live rows for four nym-mnsrehearsal001/*/v3 packages and sinopia4/safe/v2. Read lines 275-282: the grep -c count comes before live_rows. whatsnew.sh does not exist at the merge base.
- Test: [`tests/b2-lines-reach-rollout-page-catalog-deployed-count-parked.sh`](tests/b2-lines-reach-rollout-page-catalog-deployed-count-parked.sh)

### 4 · Nit · PLAUSIBLE · `misc/deployments/home-alias/whatsnew.sh:376`

The X card header may wrap onto three lines at 1280 wide; only a render can show it

- Check: build gnoweb from head, serve a copy of home.mainnet.md filled by whatsnew with -aliases /x=static:<file>, take a screenshot with chromium --headless --window-size=1280,5200 --screenshot, and compare it with one at 1440
- Observed: whatsnew.sh:376 `printf '%s **[%s · %s](%s)**\n\n%s' "$1" "$2" "$(short_date "$date")" "$url" "$text"`. No screenshot taken.

### 8 · Nit · CONFIRMED · `misc/deployments/home-alias/whatsnew.sh:447`

Each Explore column links its heading and its button to the same page, so a visitor sees two links to one place

- Check: sed -n '427p;447p;454p;461p;470p;421,423p' misc/deployments/home-alias/whatsnew.sh: each heading link equals its column's button href
- Observed: whatsnew.sh:447 `**[Top realms](/r/)**` with :427 `with_button "$used" /r/ 'Explore realms'`; :454 `**[Top packages](/p/)**` with :421 `/p/ 'Explore packages'`; :461 `**[Gno.land releases](https://github.com/gnolang/gno/releases)**` with :422 `https://github.com/gnolang/gno/releases 'All releases'`; :470 `pulls?q=is%%3Apr+is%%3Amerged` (printf gives %3A) with :423 `"$pulls" 'All merged PRs'`, $pulls = `...pulls?q=is%3Apr+is%3Amerged`.

### 11 · Nit · CONFIRMED · `misc/deployments/home-alias/whatsnew_test.sh:233`

The 'second run' test re-runs the block from a fresh shell, while a reader who re-pastes it in the same terminal is already inside myrealm/ and gets myrealm/myrealm/

- Check: extract the block as line 225 does, then `bash -c 'source block.sh; source block.sh'` with gnokey and gno stubbed; ls myrealm/myrealm
- Observed: whatsnew_test.sh:233 `(cd "$work/deploy" && PATH=... bash -e block.sh ...)` then :234 `check "a second run still writes into myrealm/" '[ ! -e "$work/deploy/myrealm.gno" ] ...'`; home.testnet.md:62 `mkdir -p myrealm && cd myrealm`.

### 9 · Suggestion · CONFIRMED · `misc/deployments/home-alias/README.md:91`

The gnoalias.sh route the README recommends for several hosts restarts gnoweb on every host each time the block changes, the request drop the README warns against

- Check: sed -n '26,31p;51,55p' misc/deployments/home-alias/gnoalias.sh and gh pr view 6327 -R gnolang/gno --json state; count the page rewrites a day from STATE_DIR mtimes on a running host
- Observed: gnoalias.sh:28-31 `if cmp -s home-override.md home.md; then return 1 else cp home-override.md home.md; return 0`; :53-55 `if check_gnoland_home; then ... eval "$RELOAD_CMD"`; :12-13 DEFAULT_RELOAD_CMD `docker ps -a --filter ancestor="ghcr.io/gnolang/gno/gnoweb:master" ... \| xargs -r docker restart`; :6 SLEEPING_TIME=60. gh pr view 6327: {"state":"OPEN"}. Restart count per day not measured: no running host here.

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`pr6434_home_whats_new.md:61`](https://github.com/gnolang/gno/blob/37b7dd1f1/gno.land/adr/pr6434_home_whats_new.md#L61) | claims | ADR: the scan copies gnoweb's chain map with the same rate probe, band width and window | whatsnew.sh:38-43 ACTIVITY_WINDOW=604800 RATE_PROBE=20000 BAND_WIDTH=25000 MIN_BAND_WIDTH=250; activity.go:18 activityWindow = 7*24h, :34 rateProbe = 20_000, :41 bandWidth = 25_000, :45 minBandWidth = 250 |
| [`pr6434_home_whats_new.md:75`](https://github.com/gnolang/gno/blob/37b7dd1f1/gno.land/adr/pr6434_home_whats_new.md#L75) | claims | ADR: only a release tag goes into the version link, an untagged build leaves the line out | whatsnew.sh:293 jq test("^v[0-9.]+(-rc\\.[0-9]+)?$") else ""; :488 [ -z "$tag" ] \|\| printf 'Runs [...]'; a live --builders run rendered 'Runs v1.8.0-rc.1' |
| [`README.md:26`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/README.md#L26) | claims | README: without the script the marker lines render as nothing | head gnoweb serving the unfilled home.testnet.md: the markers come out as HTML comments only (grep -c 'whats-new\|raw HTML omitted' = 2), nothing visible in the screenshot |
| [`README.md:91`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/README.md#L91) | claims | README: with gnoalias.sh, run whatsnew on home-override.md with ON_CHANGE empty | gnoalias.sh:26-31 copies home-override.md over home.md when they differ, inside a 60 s loop that then runs the reload command |
| [`home.testnet.md:78`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L78) | page | Deploy command's -chainid and -gas-fee might be wrong for onyx | onyx /status node_info.network = onyx-1; abci auth/gasprice = {gas 1000, price 1ugnot}, so 20000000 gas needs 20000ugnot and 40000ugnot covers it |
| [`home.testnet.md:38`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L38) | page | New headings use icons lightbulb and rocket, and the filled block uses signal and branch, which may be unknown to gnoweb | gno.land/pkg/gnoweb/markdown/icons/icons.txt:196 rocket, :199 lightbulb, :177 'signal signal_full', :155 branch; head render shows no 'gno-icon: unknown name' comment |
| [`home.testnet.md:46`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L46) | page | Build with AI column says gnomcp creates its own key and funds itself | mcp.gno.dev page: 'Funding a key and deploying to testnet. gno_faucet_fund agent key funded', and tool list includes gno_key_generate and gno_faucet_fund |
| [`home.testnet.md:111`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L111) | page | Get a grant links to github.com/gnolang/grants, which may be gone | gh api repos/gnolang/grants: archived false; README opens 'Gno.land Grants Program' |
| [`home.testnet.md:34`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L34) | page | 'the list' link to /r/ may lead nowhere | head gnoweb GET /r/ = 200, page title 'Realms - gno.land' |
| [`home.testnet.md:1`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L1) | removed | Removed testnet items (On this testnet frames, Boards, faucet as first use link, the testnet notice) may have been added at a reviewer's request | git log -S at d0cbe58c3: 'On this testnet', 'Boards is deployed on this testnet', 'Get free testnet GNOTs' added in b884b4a56 (gnolang/gno#6293) and '**This is a testnet.**' in 21136dcae (gnolang/gno#6223), both by the PR's own author; the notice's content survives in the new NOTE ('package names here are not reserved for production') |
| [`home.mainnet.md:33`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.mainnet.md#L33) | page | PR body: shorter descriptions make the three column buttons line up | head gnoweb screenshots of the filled mainnet home at 1280 and 1440: Explore the ecosystem, Getting started guide and Join gno.world sit on one baseline at both widths |
| [`home.testnet.md:53`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/home.testnet.md#L53) | page | Gnockpit and Gnoscan links in the filled Live on onyx block may not serve onyx | curl -L https://gnockpit.onyx.testnets.gno.land = 200; https://gnoscan.io/?chainId=onyx-1 = 200 |
| [`whatsnew.sh:310`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L310) | reach | vm/qpaths?limit=10000 sits at the handler's maxLimit, so a chain past 10000 packages would be ranked on a silent prefix | live qpaths on rpc.gno.land: 248 paths under gno.land/p/, 463 under gno.land/r/, far from the 10_000 cap in handler.go:255 |
| [`whatsnew.sh:265`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L265) | reach | vm/qfile might answer for a parked package, so live_rows would list parked code as deployed | keeper.go:2065 QueryFile reads store.GetMemPackageAll (active store; parked code sits under inert_pkg:<path>), and the onyx run printed not-live for 5 parked paths |
| [`whatsnew.sh:314`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L314) | reach | the imports ranking has no namespace guard, so throwaway importers could push a package up | params/vm:p:code_submission_policy answers "inert" on rpc.gno.land: an importer is online only once enabled, so throwaway deploys do not reach qpaths |
| [`whatsnew.sh:136`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L136) | page | X oEmbed author_url or the date link might be twitter.com, emptying the X row | live `--source x` rc=0 printed 3 rows, the oEmbed html dates link https://x.com/_gnoland/status/...">November 18, 2026</a> |
| [`whatsnew.sh:161`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L161) | page | the blog capture might not match the blog realm's list format | blog.gno:309-312 writes "\n### [%s](%s)\n" then Format("02 Jan 2006"); live `--source blog` rc=0 printed 3 rows |
| [`whatsnew.sh:293`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L293) | reach | /status might carry no build_version, so the Runs line never shows | tm2 responses.go:164 BuildVersion `json:"build_version"`; live `--source version` printed v1.8.0 |
| [`whatsnew.sh:489`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L489) | page | Gnoscan and Gnockpit links for onyx might lead nowhere | gnockpit.onyx.testnets.gno.land answers 200; one gnoscan.io JS chunk names onyx, and onyx's upgrades.json chain_id is onyx-1 |
| [`whatsnew.sh:403`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L403) | page | the Read the blog button's ${LINK_BASE}/blog might 404 | redirect.go:11 "/blog": "/r/gnoland/blog"; curl https://gno.land/blog answers 302 to /r/gnoland/blog; /r/ and /p/ answer 200 |
| [`whatsnew.sh:402`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L402) | lines | ${X_ACCOUNT/_/\\_} might drop the backslash under bash 5.2+ patsub_replacement | bash 5.3.20: x=_gnoland; echo "${x/_/\\_}" printed \_gnoland |
| [`whatsnew.sh:244`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L244) | reach | tx-indexer's like might not be a regular expression, emptying the namespace-filtered scan | live `--source used` on indexer.gno.land rc=0 printed gnoland/wugnot 71, gnoswap/gns 68 ...; onyx with USED_NAMESPACES= rc=0, activity 27 18 |
| [`whatsnew.sh:331`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L331) | rollout | a timer period equal to a TTL would skip every other refresh, the stamp being a few seconds younger than the TTL | README timer is OnUnitActiveSec=10min with RandomizedDelaySec=2min against FEED_TTL 30: no period equals a TTL |
| [`whatsnew.sh:401`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L401) | page | an icon name the script uses might be missing from gnoweb's set | ICONS.md lists book:75, box:80, branch:86, cube:178, flame:233, newspaper:352, rocket:420, signal:440 |
| [`whatsnew.sh:1`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L1) | catalog | invariant catalog classes against a deployment shell script | skills/invariant-catalog.md classes (determinism, gas, realm state, caller, coin, storage deposit, VM faults, VM semantics, type-check, older-binary data) all concern gno code; the script touches none |
| [`whatsnew.sh:293`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew.sh#L293) | reach | source_version's -rc.N-only regex (pinned by the version test) might miss the live chains' build_version shape | curl rpc.onyx.testnets.gno.land/status and rpc.gno.land/status: {"b":"v1.8.0-rc.1",...} and {"b":"v1.8.0",...}, both match ^v[0-9.]+(-rc\.[0-9]+)?$ |
| [`whatsnew_test.sh:194`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew_test.sh#L194) | lines | the root-only read-only check might fail when it does run | unshare -r bash whatsnew_test.sh: 'ok a read-only page stays read-only under root' ... PASS |
| [`whatsnew_test.sh:45`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew_test.sh#L45) | lines | the stub case answering null for negative getBlocks heights looked unreachable | whatsnew.sh:217 `from=$((tip > RATE_PROBE ? tip - RATE_PROBE : 1))`: the case makes a regression dropping that guard fail scan_week, so it pins the guard rather than dead code |
| [`whatsnew_test.sh:1`](https://github.com/gnolang/gno/blob/37b7dd1f1/misc/deployments/home-alias/whatsnew_test.sh#L1) | lines | the suite might not pass at head, or the body's '34 checks' might not match | bash whatsnew_test.sh at 37b7dd1f1: 33 'ok', 1 'skip ... (not root)', PASS rc=0; grep -c 'check ' = 35 including the definition, so 34 checks |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | misc/deployments/home-alias/home.testnet.md:82 | bash projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b1-removed-claims-page-gpao-status-host.sh misc/deployments/home-alias/home.testnet.md from the head tree: expect DoH Status 0 and an HTTP answer, observe Status 3 and curl_exit=6 | hot | dead-troubleshooting-link |
| 2 | CONFIRMED | Nit | misc/deployments/home-alias/home.testnet.md:3 | curl -fsSL https://rpc.onyx.testnets.gno.land:443/status \| jq .result.build_version; curl -fsSL https://rpc.gno.land:443/status \| jq .result.build_version: the copy holds only if they match or onyx is ahead | hot |  |
| 3 | CONFIRMED | Nit | misc/deployments/home-alias/home.testnet.md:58 | read home.testnet.md:28-82 in order as a fresh reader; on gnodev, add key k1, fund it, then run the two blocks as printed with <your-g1-addr> = k1's address: expect the deploy to land, observe that alice is unfunded | hot |  |
| 4 | PLAUSIBLE | Nit | misc/deployments/home-alias/whatsnew.sh:376 | build gnoweb from head, serve a copy of home.mainnet.md filled by whatsnew with -aliases /x=static:<file>, take a screenshot with chromium --headless --window-size=1280,5200 --screenshot, and compare it with one at 1440 | hot |  |
| 5 | CONFIRMED | Nit | misc/deployments/home-alias/whatsnew.sh:281 | WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash tests/b2-lines-reach-rollout-page-catalog-deployed-count-parked.sh (live, read-only, onyx RPC and indexer): expect the count to equal the live deployments, observe 31 counted and 26 online | hot |  |
| 6 | CONFIRMED | Warning | misc/deployments/home-alias/whatsnew.sh:246 | WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash tests/b2-lines-reach-rollout-page-catalog-zero-call-week.sh: expect rc 0 and activity 0 0 for the empty week, observe rc=1 and yesterday's 1 1 rendered | hot | pipefail-empty-grep |
| 7 | CONFIRMED | Nit | misc/deployments/home-alias/whatsnew.sh:73 | STATE_DIR=$(mktemp -d) bash misc/deployments/home-alias/whatsnew.sh --source x \| grep -c 'CompleteGno'; and curl publish.x.com/oembed for that status to see the missing break | hot |  |
| 8 | CONFIRMED | Nit | misc/deployments/home-alias/whatsnew.sh:447 | sed -n '427p;447p;454p;461p;470p;421,423p' misc/deployments/home-alias/whatsnew.sh: each heading link equals its column's button href | hot |  |
| 9 | CONFIRMED | Suggestion | misc/deployments/home-alias/README.md:91 | sed -n '26,31p;51,55p' misc/deployments/home-alias/gnoalias.sh and gh pr view 6327 -R gnolang/gno --json state; count the page rewrites a day from STATE_DIR mtimes on a running host | cold |  |
| 10 | CONFIRMED | Warning | misc/deployments/home-alias/home.testnet.md:61 | bash tests/b3-lines-reach-rollout-page-catalog-deploy-block-readds-key.sh from the head tree: builds gnokey from the tree, runs `gnokey add alice` twice on a pty with Enter presses, compares `gnokey list` addresses | hot | tutorial-step-reorders-key-creation |
| 11 | CONFIRMED | Nit | misc/deployments/home-alias/whatsnew_test.sh:233 | extract the block as line 225 does, then `bash -c 'source block.sh; source block.sh'` with gnokey and gno stubbed; ls myrealm/myrealm | hot |  |
| 12 | PLAUSIBLE | Missing test | misc/deployments/home-alias/whatsnew.sh:103 | grep -nE 'to_md\|expand_links\|html_to_text' misc/deployments/home-alias/whatsnew_test.sh (no hit); then mutate line 111 to `gsub("(?<c>x)"; "\(.c)")` and run bash whatsnew_test.sh: expect PASS | hot |  |
| 13 | PLAUSIBLE | Missing test | misc/deployments/home-alias/whatsnew.sh:580 | grep -nE 'refresh\|fill_block\|reload\|nothing to show' misc/deployments/home-alias/whatsnew_test.sh (no hit); delete line 580-583 and run bash whatsnew_test.sh: expect PASS | hot |  |
| 14 | PLAUSIBLE | Missing test | misc/deployments/home-alias/whatsnew_test.sh:4 | grep -rn whatsnew_test .github (no hit); sed -n 10,15p .github/workflows/ci-dir-misc.yml | hot |  |
| 15 | PLAUSIBLE | Missing test | misc/deployments/home-alias/whatsnew_test.sh:223 | go build ./gnovm/cmd/gno from the head tree; in an empty dir run `gno mod init "gno.land/r/<your-g1-addr>/myrealm"`: expect an invalid-char error; run a valid init twice: expect 'file already exists' | hot |  |

</details>

Hit rate per tier, from the record: hot 9/14 confirmed over 3 files, warm 0/0 confirmed over 0 files, cold 1/1 confirmed over 3 files.

## Completeness

- Diff read: every hunk in `diff.md`, hot files first per `risk.md`; three finders covered the page, lines, reach and rollout angles.
- Gate: every Warning judged by an agent that was not its finder; 15 rows, 10 CONFIRMED, 5 PLAUSIBLE, 0 refuted.
- Merged lines: `home.testnet.md:58` ships SKIP, since the edit to `home.testnet.md:61` closes it.
- Unrun: the four Missing test rows and the `whatsnew.sh:376` layout Nit stay PLAUSIBLE; their checks are named in their blocks.
- Not measured: restarts a day on the `gnoalias.sh` route, which needs a running host.

## Retro

- What failed: the first writer run took a message the user typed mid-run as its request, read it as declining the round, and wrote nothing. The resume raised `disk_retry` to change the writer's prompt, so the run cost a second writer pass. Measured cost: about $17.84 over 13 agents, 277k output, 17.0M cache read.
- What worked: 16 candidates from 3 finders, and 0 refuted. Every Warning was run by a judge that was not its finder.
- Upgrade: give the runner a resume counter that changes the writer's prompt for any cause, not only a disk stop. It costs one line in `review-pipeline.js` and saves a cached refusal replaying on resume. Recorded, not applied: a script change waits for the corpus review loop.
