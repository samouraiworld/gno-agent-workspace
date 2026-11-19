#!/usr/bin/env bash
# gnolang/gno#6434 at 37b7dd1f1: misc/deployments/home-alias/home.testnet.md:82
# tells a builder whose package is not enabled to open
#   https://gpao.onyx.testnets.gno.land/status/gno.land/r/<your-g1-addr>/myrealm
# This checks that the host exists and answers.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6434/head && git checkout 37b7dd1f15a79144153303fe49a98e71f6ff9846
#   bash b1-removed-claims-page-gpao-status-host.sh misc/deployments/home-alias/home.testnet.md
#
# Expect on a correct page: DNS answers, and /status/<a known path> answers 200.
# Observed 2026-11-19: Cloudflare DNS returns NXDOMAIN (Status 3) for the host,
# curl exits 6 (could not resolve host); rpc.onyx.testnets.gno.land resolves.
# The only mention of this host in the tree is the page itself, and gpao's
# status board is off unless --status-listen is set (contribs/gpao/README.md).
set -u
page=${1:-misc/deployments/home-alias/home.testnet.md}
url=$(grep -oE 'https://gpao\.[a-z0-9.-]+/status/[^` ]+' "$page" | head -1)
host=$(sed -E 's#https://([^/]+)/.*#\1#' <<<"$url")
echo "page cites: $url"
for n in "$host" rpc.onyx.testnets.gno.land; do
  printf 'DoH %s: ' "$n"
  curl -fsS --max-time 15 -H 'accept: application/dns-json' \
    "https://cloudflare-dns.com/dns-query?name=$n&type=A" | jq -c '{Status, A: [.Answer[]?.data]}'
done
printf 'GET https://%s/status/gno.land/r/gnoland/home: ' "$host"
curl -s -o /dev/null -w 'http=%{http_code}' --max-time 20 "https://$host/status/gno.land/r/gnoland/home"
echo " curl_exit=$?"
printf 'tree mentions of the host outside the page: '
git grep -lF "$host" -- . ":!$page" | wc -l
