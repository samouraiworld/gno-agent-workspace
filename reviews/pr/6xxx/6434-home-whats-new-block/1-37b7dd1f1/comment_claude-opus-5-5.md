# Review: [#6434](https://github.com/gnolang/gno/pull/6434)
Posted: https://github.com/gnolang/gno/pull/6434#pullrequestreview-5974577384

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Three Warnings block, all on lines the branch adds: the deploy block replaces a funded `alice` key, the troubleshooting link points at an unresolvable host, and a week with no realm call leaves stale activity figures on the page.
Model: claude-opus-5-5, low effort, quick review
Commit: 37b7dd1f15a79144153303fe49a98e71f6ff9846
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6434 37b7dd1f1`
Open the code: [home-alias](https://github.com/gnolang/gno/tree/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias)
Round: 1. 3 finders, no reflector, 11 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## misc/deployments/home-alias/home.testnet.md:61 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md#L61) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/home.testnet.md#L61) · Warning [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628376982)

The block's first line re-adds the key the [prose above](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md?plain=1#L58) had the visitor fund. An Enter at gnokey's [override prompt](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/tm2/pkg/commands/utils.go#L65-L66) replaces that funded key with an empty one, and the deploy fails on fees.

<details><summary>repro</summary>

The second `gnokey add alice` asks to override the existing key, and an Enter keeps the default, so `alice` ends on a new address with no funds:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6434 -R gnolang/gno
go build -o /tmp/gnokey ./gno.land/cmd/gnokey
home=$(mktemp -d)
/tmp/gnokey add alice -home "$home" -insecure-password-stdin <<<$'\n\n' >/dev/null 2>&1
before=$(/tmp/gnokey list -home "$home" | grep -o 'g1[a-z0-9]*')
/tmp/gnokey add alice -home "$home" -insecure-password-stdin <<<$'\n\n\n' >/dev/null 2>&1
after=$(/tmp/gnokey list -home "$home" | grep -o 'g1[a-z0-9]*')
echo "funded: $before"; echo "after:  $after"
[ "$before" = "$after" ] && echo KEY KEPT || echo FUNDED KEY REPLACED
rm -rf "$home"
```

```
funded: g1njegdnv5cu29untnw9v62fjjmadfcppql0h6h5
after:  g1w0ejfwkfkmuy4h35tj3dsqpp0zud8lqg65j7ux
FUNDED KEY REPLACED
```

The keybase is shared across chains, so an `alice` holding value on another chain is replaced too. The deploy-block check in [`whatsnew_test.sh`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L222) stubs gnokey with `exit 0` and cannot see it.

</details>

## misc/deployments/home-alias/home.testnet.md:82 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md#L82) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/home.testnet.md#L82) · Warning [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377004)

The status host in this sentence answers NXDOMAIN, so a builder with a parked package gets a connection error in place of the reason. [#6504](https://github.com/gnolang/gno/pull/6504) moved the service to `gpas.onyx.testnets.gno.land`, which resolves, so the same rename on this line fixes it.

<details><summary>repro</summary>

The status host does not resolve, while the onyx RPC host beside it does:

```bash
for h in gpao.onyx.testnets.gno.land gpas.onyx.testnets.gno.land rpc.onyx.testnets.gno.land; do
  echo "$h: $(curl -fsS -H 'accept: application/dns-json' "https://cloudflare-dns.com/dns-query?name=$h&type=A" | jq -c '{Status}')"
done
curl -sS -o /dev/null https://gpao.onyx.testnets.gno.land/status/gno.land/r/gnoland/home; echo "curl_exit=$?"
```

```
gpao.onyx.testnets.gno.land: {"Status":3}
gpas.onyx.testnets.gno.land: {"Status":0}
rpc.onyx.testnets.gno.land: {"Status":0}
curl_exit=6
```

</details>

## misc/deployments/home-alias/whatsnew.sh:246 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L246) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L246) · Warning [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377020)

This filter exits 1 on a week with no matching realm call, and the [`set -euo pipefail`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L5) options end the function early. The page then keeps last week's users, realms and ranking beside a fresh zero deployments.

<details><summary>repro</summary>

The same pipeline under the script's shell options fails on empty input, where an awk filter would print nothing and exit 0:

```bash
bash -c 'set -euo pipefail
realms="gno\\.land/r/"
calls=$(printf "" | grep -E "$realms" | sort -u)
echo "activity rewritten"'; echo "rc=$?"
```

```
rc=1
```

With graphql stubbed to return one call and then none, the page keeps the old figures:

```
**0** deployments this week
**1** active users this week
**1** realms used this week
1. [demo/foo](/r/demo/foo) · 1
```

</details>

## SKIP misc/deployments/home-alias/whatsnew.sh:103 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L103) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L103) · Missing test

Missing test: no case in [`whatsnew_test.sh`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh) covers the escaping of untrusted post text into page markdown.

Not posted: PLAUSIBLE on the finder's read; the mutation run against the suite is still to run.

## SKIP misc/deployments/home-alias/whatsnew.sh:580 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L580) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L580) · Missing test

Missing test: no case in [`whatsnew_test.sh`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh) covers an outage or a lost state directory.

Not posted: PLAUSIBLE on the finder's read; deleting the guard and running the suite is still to run.

## SKIP misc/deployments/home-alias/whatsnew_test.sh:4 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L4) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew_test.sh#L4) · Missing test

Missing test: no workflow under `.github` runs `whatsnew_test.sh`, so its checks of the deploy block only run by hand.

Not posted: PLAUSIBLE on the finder's read, and a finding about CI wiring.

## SKIP misc/deployments/home-alias/whatsnew_test.sh:223 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L223) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew_test.sh#L223) · Missing test

Test: the `gno` stub accepts module paths and repeated inits that the real `gno mod init` refuses, so the deploy-block checks pin the stub rather than the block.

Not posted: PLAUSIBLE on the finder's read; the run of the real `gno mod init` is still to run.

## misc/deployments/home-alias/home.testnet.md:3 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md?plain=1#L3) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/home.testnet.md#L3) · Nit [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377032)

Nit: this line says Onyx runs mainnet's code, yet Onyx reports `v1.8.0-rc.1` and mainnet reports `v1.8.0`.

## SKIP misc/deployments/home-alias/home.testnet.md:58 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/home.testnet.md?plain=1#L58) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/home.testnet.md#L58) · Nit

Nit: the prose has the reader fund a key before the block adds `alice`, so a key funded under another name goes unused.

Not posted: the edit to the `gnokey add alice` line closes it, and the `home.testnet.md:61` section carries it.

## misc/deployments/home-alias/whatsnew.sh:73 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L73) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L73) · Nit [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377039)

Nit: stripping the link tag joins the link to the word before it, so the mainnet home reads `Upgrade: CompleteGno.land mainnet is now running v1.8.0`.

## misc/deployments/home-alias/whatsnew.sh:281 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L281) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L281) · Nit [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377046)

Nit: the weekly count includes parked packages that never went online, while the list beside it shows only live ones.

<details><summary>count</summary>

On Onyx, read-only against its RPC and indexer: `deployments this week (page): 31`, `counted deployments online now: 26 of 31`.

</details>

## SKIP misc/deployments/home-alias/whatsnew.sh:376 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L376) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L376) · Nit

Nit: the X card header puts an icon, a bold link and a date on one line, which may wrap onto three lines at 1280 pixels wide.

Not posted: PLAUSIBLE, and only a headless render at 1280 and 1440 wide settles it.

## misc/deployments/home-alias/whatsnew.sh:447 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L447) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew.sh#L447) · Nit [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377054)

Nit: the `Top realms` heading and the [`Explore realms` button](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L427) both open `/r/`. The packages, releases and merged-PR columns repeat that pairing.

## misc/deployments/home-alias/whatsnew_test.sh:233 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew_test.sh#L233) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/whatsnew_test.sh#L233) · Nit [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377062)

Test: re-pasting the block inside `myrealm/` writes into `myrealm/myrealm/`. This check runs the second paste from the parent directory, so it covers only the `-p` flag.

## misc/deployments/home-alias/README.md:91 [gh](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/README.md?plain=1#L91) · [↗](../../../../../.worktrees/gno-review-6434/misc/deployments/home-alias/README.md#L91) · Suggestion [posted](https://github.com/gnolang/gno/pull/6434#discussion_r4628377071)

Suggestion: on each block change, the `gnoalias.sh` route [restarts every gnoweb container](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/gnoalias.sh#L12-L13). That cost is missing from the README, and gnolang/gno#6327 removes it.
