# Findings in posting order, from round assemble: 10 to post, 5 SKIP, 0 refuted kept out

## misc/deployments/home-alias/home.testnet.md:61 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md#L61) · Warning
State: CONFIRMED, band: Warning, angle: page
TL;DR: a visitor who follows the prose and adds and funds alice loses that key when the deploy block's first line re-adds it, and the deploy then fails
Check: bash tests/b3-lines-reach-rollout-page-catalog-deploy-block-readds-key.sh from the head tree: builds gnokey from the tree, runs `gnokey add alice` twice on a pty with Enter presses, compares `gnokey list` addresses
Details: The prose at line 58 has the visitor add a key and fund it before writing the realm, and the block under it opens on `gnokey add alice`. With a same-name, different-address key, add.go:553 asks `Override the existing key "alice"` through GetConfirmation, which returns true on an empty line (utils.go:65-66 `return true, nil // Default to yes when Enter is pressed`), so Enter replaces the funded key with a new unfunded one; addpkg then signs from an empty account whose address is not the <your-g1-addr> the visitor substituted. The keybase is shared across chains, so an existing alice holding value elsewhere is replaced too, and whatsnew_test.sh:222 stubs gnokey as `exit 0`, so its deploy-block check cannot see this. Fix: drop `gnokey add alice` from the block and keep key creation in the prose step (or open the block on `gnokey add alice` and move the funding sentence after the block), so the funded key is never re-added.
Evidence: Reran the finder's artifact in a judge worktree at 37b7dd1f1 (gnokey built from that tree): `block line 1: gnokey add alice` / `funded: g1njegdnv5cu29untnw9v62fjjmadfcppql0h6h5` / `after:  g1w0ejfwkfkmuy4h35tj3dsqpp0zud8lqg65j7ux` / `FUNDED KEY REPLACED`, rc=1. Read tm2/pkg/commands/utils.go:65-66 and tm2/pkg/crypto/keys/client/add.go:497-553: same name with a different address falls through to the Override prompt, default yes. Merge base: grep for `gnokey add` in misc/deployments/home-alias/ returns nothing; the base page has no deploy block.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b3-lines-reach-rollout-page-catalog-deploy-block-readds-key.sh

## misc/deployments/home-alias/home.testnet.md:82 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md#L82) · Warning
State: CONFIRMED, band: Warning, angle: page
TL;DR: the onyx home's only troubleshooting step for a package that is not enabled points at a host that does not resolve
Check: bash projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b1-removed-claims-page-gpao-status-host.sh misc/deployments/home-alias/home.testnet.md from the head tree: expect DoH Status 0 and an HTTP answer, observe Status 3 and curl_exit=6
Details: The diff adds line 82, which tells a builder whose addpkg did not enable to open https://gpao.onyx.testnets.gno.land/status/<pkg>. That name answers NXDOMAIN, so the builder gets a connection error and no reason for the parked package. Nothing in the tree deploys that host, and gpao serves /status only with --status-listen, which contribs/gpao/README.md:95 lists as off by default. The base page has no such line. Fix: link the status endpoint only once it is deployed under that name, or drop the sentence and keep the Discord pointer.
Evidence: Reran the finder's artifact from the head tree. Output: 'DoH gpao.onyx.testnets.gno.land: {"Status":3,"A":[]}', rpc.onyx resolves (Status 0), 'GET .../status/gno.land/r/gnoland/home: http=000 curl_exit=6', 'tree mentions of the host outside the page: 0'. Bundle line 567 shows the sentence as an added line. grep for gpao in the base home-alias directory: no hit.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b1-removed-claims-page-gpao-status-host.sh

## misc/deployments/home-alias/whatsnew.sh:246 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L246) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: a week with no matching realm call aborts the activity refresh, so the page keeps showing last week's users, realms and ranking beside a fresh zero
Check: WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash tests/b2-lines-reach-rollout-page-catalog-zero-call-week.sh: expect rc 0 and activity 0 0 for the empty week, observe rc=1 and yesterday's 1 1 rendered
Details: The script runs under set -euo pipefail (line 5). When scan_week returns no MsgCall, `grep -E "$realms"` at line 246 exits 1, so the calls=$(...) assignment fails. Errexit then ends source_used before line 247 rewrites the activity file. is_recent at line 355 keeps accepting the stale file for USED_MAX_AGE, so render_builders prints the old 'active users' and 'realms used' figures and the old Top realms list next to a fresh '0 deployments', because source_deployed tolerates an empty week with `grep -c . || true`. Fix: end the pipeline with `|| true`, or swap the filtering greps for an awk that exits 0 on empty input, the way source_deployed does.
Evidence: Reran the finder's artifact against the head script, which sources whatsnew.sh with graphql stubbed: 'week with one call: rc=0 activity=[1 1]', 'week with no call:  rc=1 out=[] activity=[1 1]', rendered '**0** deployments this week / **1** active users this week / **1** realms used this week / 1. [demo/foo](/r/demo/foo) · 1'. Read lines 241-249. whatsnew.sh does not exist at the merge base.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b2-lines-reach-rollout-page-catalog-zero-call-week.sh

## SKIP misc/deployments/home-alias/whatsnew.sh:103 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L103) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: no check covers to_md, expand_links or html_to_text, the escaping the body's Safety section rests on
Check: grep -nE 'to_md|expand_links|html_to_text' misc/deployments/home-alias/whatsnew_test.sh (no hit); then mutate line 111 to `gsub("(?<c>x)"; "\(.c)")` and run bash whatsnew_test.sh: expect PASS
Evidence: not run: no verifier reached it; the check is still to run

## SKIP misc/deployments/home-alias/whatsnew.sh:580 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L580) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: no check covers the outage, lost-state and keep-previous paths: main's empty-block guard, refresh's failure branch, fill_block, the reload retry
Check: grep -nE 'refresh|fill_block|reload|nothing to show' misc/deployments/home-alias/whatsnew_test.sh (no hit); delete line 580-583 and run bash whatsnew_test.sh: expect PASS
Evidence: not run: no verifier reached it; the check is still to run

## SKIP misc/deployments/home-alias/whatsnew_test.sh:4 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L4) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: no CI job runs whatsnew_test.sh, and ci-dir-misc excludes the .md whose deploy block it tests
Check: grep -rn whatsnew_test .github (no hit); sed -n 10,15p .github/workflows/ci-dir-misc.yml
Evidence: not run: no verifier reached it; the check is still to run

## SKIP misc/deployments/home-alias/whatsnew_test.sh:223 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L223) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: the gno stub accepts what the real `gno mod init` refuses, so the deploy-block checks pin the stub, not the block
Check: go build ./gnovm/cmd/gno from the head tree; in an empty dir run `gno mod init "gno.land/r/<your-g1-addr>/myrealm"`: expect an invalid-char error; run a valid init twice: expect 'file already exists'
Evidence: not run: no verifier reached it; the check is still to run

## misc/deployments/home-alias/home.testnet.md:3 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md#L3) · Nit
State: CONFIRMED, band: Nit, angle: page
TL;DR: the testnet home says Onyx runs mainnet's code, but Onyx runs v1.8.0-rc.1 while mainnet runs v1.8.0
Check: curl -fsSL https://rpc.onyx.testnets.gno.land:443/status | jq .result.build_version; curl -fsSL https://rpc.gno.land:443/status | jq .result.build_version: the copy holds only if they match or onyx is ahead
Details: home.testnet.md:3 opens with 'Onyx runs mainnet's code with free test tokens.' Today the two chains report different builds, and Onyx is the older one, so a visitor who tests gas or VM behaviour on Onyx as a mainnet stand-in can get different results. Fix: word the line so it holds whatever the version gap is, e.g. 'Onyx runs the same or the next release of mainnet's code', or drop the claim.
Evidence: curl rpc.onyx.testnets.gno.land/status: result.build_version "v1.8.0-rc.1"; curl rpc.gno.land/status: result.build_version "v1.8.0" (2026-11-19).

## misc/deployments/home-alias/home.testnet.md:58 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md#L58) · Nit
State: CONFIRMED, band: Nit, angle: page
TL;DR: the page asks for a key to be added and funded before the block that adds it, so a reader can end up signing with a second, unfunded alice
Check: read home.testnet.md:28-82 in order as a fresh reader; on gnodev, add key k1, fund it, then run the two blocks as printed with <your-g1-addr> = k1's address: expect the deploy to land, observe that alice is unfunded
Details: Step 1 at line 18 says 'Install gnokey and add a key' and step 2 says get test GNOT; line 58 again says to add a key and fund its address, 'Then write a realm:'. The block right after opens with `gnokey add alice`, and the deploy block signs with alice. A reader who added and funded a key with another name before reaching the block creates a new, empty alice, and addpkg fails on fees. The gnodev run was not done; the read settles the order. Fix: move `gnokey add alice` into the prose before the faucet step, or start the block after the key is made and sign with the key the reader funded.
Evidence: home.testnet.md:18 `1. [Install gnokey](...) and add a key`; :58 `Add a key (`gnokey list` prints its g1 address), and fund that address at the [faucet]... Then write a realm:`; :61 `gnokey add alice`; :79 `alice` as the addpkg signer.

## misc/deployments/home-alias/whatsnew.sh:73 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L73) · Nit
State: CONFIRMED, band: Nit, angle: page
TL;DR: the mainnet home's X post reads 'CompleteGno.land' because tag stripping glues the word before a link to the link's replacement
Check: STATE_DIR=$(mktemp -d) bash misc/deployments/home-alias/whatsnew.sh --source x | grep -c 'CompleteGno'; and curl publish.x.com/oembed for that status to see the missing break
Details: X's oEmbed html for status 2123072056153759774 has no space or <br> between 'Complete' and the <a> tag, where the post had a line break. html_to_text removes the tag, leaving 'Completehttps://t.co/UPaevcA79z', and expand_links swaps that t.co URL for 'Gno.land' since it redirects to https://gno.land/, so visitors read 'Upgrade: CompleteGno.land mainnet is now running v1.8.0'. Fix: replace each <a ...> opening tag with a space before stripping tags, e.g. gsub("<a [^>]*>"; " "), so the later space squeeze keeps one blank.
Evidence: curl publish.x.com/oembed?url=.../2123072056153759774 | jq -r .html: `Upgrade: Complete<a href="https://t.co/UPaevcA79z">https://t.co/UPaevcA79z</a> mainnet`; that string through the html_to_text gsubs: `Upgrade: Completehttps://t.co/UPaevcA79z mainnet`; curl -w '%{redirect_url}' https://t.co/UPaevcA79z: `https://gno.land/`, which expand_links:86-87 rewrites to `Gno.land`.

## misc/deployments/home-alias/whatsnew.sh:281 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L281) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: the weekly deployments figure counts parked addpkg transactions that never went online, while the list beside it shows only live packages
Check: WHATSNEW=$PWD/misc/deployments/home-alias/whatsnew.sh bash tests/b2-lines-reach-rollout-page-catalog-deployed-count-parked.sh (live, read-only, onyx RPC and indexer): expect the count to equal the live deployments, observe 31 counted and 26 online
Details: source_deployed writes deployed-count at line 281 from every successful MsgAddPackage row. The vm/qfile liveness filter in live_rows runs only on the listed rows after that. On onyx, which runs the inert policy, the page shows 31 deployments this week, and 5 of those 31 do not answer vm/qfile. The number reproduces. The label 'deployments' still matches what a successful addpkg is, though, and the list beside it is deduplicated by project, so the two never had to match. That makes this a question of definition rather than a clear defect, so it is banded Nit. Fix: count only rows that answer vm/qfile, or relabel the figure as submissions.
Evidence: Reran the finder's artifact live and read-only against onyx: 'code_submission_policy: "inert"', 'deployments this week (page): 31', 'counted deployments online now: 26 of 31', not-live rows for four nym-mnsrehearsal001/*/v3 packages and sinopia4/safe/v2. Read lines 275-282: the grep -c count comes before live_rows. whatsnew.sh does not exist at the merge base.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6434-home-whats-new-block/1-37b7dd1f1/tests/b2-lines-reach-rollout-page-catalog-deployed-count-parked.sh

## SKIP misc/deployments/home-alias/whatsnew.sh:376 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L376) · Nit
State: PLAUSIBLE, band: Nit, angle: page
TL;DR: the X card header may wrap onto three lines at 1280 wide; only a render can show it
Check: build gnoweb from head, serve a copy of home.mainnet.md filled by whatsnew with -aliases /x=static:<file>, take a screenshot with chromium --headless --window-size=1280,5200 --screenshot, and compare it with one at 1440
Details: render_post prints `<icon> **[Gno.land @_gnoland · Nov 19](url)**`, so the header is the icon followed by a bold link and a date, and whether it wraps depends on the column width gnoweb gives it. A read cannot settle the layout and no render was taken here, as this judge runs nothing that builds. Confirm with a headless screenshot of the filled page at 1280 and 1440. Fix: none until a screenshot shows the wrap; SKIP.
Evidence: whatsnew.sh:376 `printf '%s **[%s · %s](%s)**\n\n%s' "$1" "$2" "$(short_date "$date")" "$url" "$text"`. No screenshot taken.

## misc/deployments/home-alias/whatsnew.sh:447 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L447) · Nit
State: CONFIRMED, band: Nit, angle: page
TL;DR: each Explore column links its heading and its button to the same page, so a visitor sees two links to one place
Check: sed -n '427p;447p;454p;461p;470p;421,423p' misc/deployments/home-alias/whatsnew.sh: each heading link equals its column's button href
Details: The 'Top realms' heading links /r/ and the button under it, 'Explore realms', also goes to /r/; the same holds for /p/ (Top packages, Explore packages), the GitHub releases page (Gno.land releases, All releases) and the merged-PR search (Just merged on GitHub, All merged PRs). A visitor who clicks the heading expecting the ranking gets the full listing the button already offers. Fix: make the headings plain bold text and keep the buttons, or drop the buttons.
Evidence: whatsnew.sh:447 `**[Top realms](/r/)**` with :427 `with_button "$used" /r/ 'Explore realms'`; :454 `**[Top packages](/p/)**` with :421 `/p/ 'Explore packages'`; :461 `**[Gno.land releases](https://github.com/gnolang/gno/releases)**` with :422 `https://github.com/gnolang/gno/releases 'All releases'`; :470 `pulls?q=is%%3Apr+is%%3Amerged` (printf gives %3A) with :423 `"$pulls" 'All merged PRs'`, $pulls = `...pulls?q=is%3Apr+is%3Amerged`.

## misc/deployments/home-alias/whatsnew_test.sh:233 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L233) · Nit
State: CONFIRMED, band: Nit, angle: page
TL;DR: the 'second run' test re-runs the block from a fresh shell, while a reader who re-pastes it in the same terminal is already inside myrealm/ and gets myrealm/myrealm/
Check: extract the block as line 225 does, then `bash -c 'source block.sh; source block.sh'` with gnokey and gno stubbed; ls myrealm/myrealm
Details: Both runs at lines 226 and 233 start with `cd "$work/deploy"` in a new subshell, so the check can only catch a missing -p on mkdir. The block does `mkdir -p myrealm && cd myrealm`, and in a real terminal that cd persists, so a re-paste after fixing <your-g1-addr> writes into myrealm/myrealm/; the check's name says a second run still writes into myrealm/. The double-source run was not done; the shell semantics follow from the lines. Fix: run the second paste from inside myrealm/ in the test, and make the block safe to re-paste, e.g. `[ "${PWD##*/}" = myrealm ] || { mkdir -p myrealm && cd myrealm; }`, or rename the check to what it covers.
Evidence: whatsnew_test.sh:233 `(cd "$work/deploy" && PATH=... bash -e block.sh ...)` then :234 `check "a second run still writes into myrealm/" '[ ! -e "$work/deploy/myrealm.gno" ] ...'`; home.testnet.md:62 `mkdir -p myrealm && cd myrealm`.

## misc/deployments/home-alias/README.md:91 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/README.md#L91) · Suggestion
State: CONFIRMED, band: Suggestion, angle: rollout
TL;DR: the gnoalias.sh route the README recommends for several hosts restarts gnoweb on every host each time the block changes, the request drop the README warns against
Check: sed -n '26,31p;51,55p' misc/deployments/home-alias/gnoalias.sh and gh pr view 6327 -R gnolang/gno --json state; count the page rewrites a day from STATE_DIR mtimes on a running host
Details: README.md:85-87 warns that restarting gnoweb with ON_CHANGE on several hosts makes every host drop requests for the same post, then line 91 sends multi-host setups through gnoalias.sh writing home-override.md. gnoalias.sh copies home-override.md over home.md whenever cmp differs and then runs RELOAD_CMD, which by default is docker restart of every gnoweb:master container, so each block change restarts gnoweb on every host within a 60 s poll. #6327, which would let gnoweb re-read the page, is still OPEN. How many restarts a day this costs was not measured here. Fix: say in the README that the gnoalias route restarts gnoweb on each change until #6327 lands, or hold that route until then.
Evidence: gnoalias.sh:28-31 `if cmp -s home-override.md home.md; then return 1 else cp home-override.md home.md; return 0`; :53-55 `if check_gnoland_home; then ... eval "$RELOAD_CMD"`; :12-13 DEFAULT_RELOAD_CMD `docker ps -a --filter ancestor="ghcr.io/gnolang/gno/gnoweb:master" ... | xargs -r docker restart`; :6 SLEEPING_TIME=60. gh pr view 6327: {"state":"OPEN"}. Restart count per day not measured: no running host here.
