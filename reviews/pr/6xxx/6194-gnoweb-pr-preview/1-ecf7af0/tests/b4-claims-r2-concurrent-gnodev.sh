#!/usr/bin/env bash
# Does the before/after pass really only collide on the two fixed locations that
# misc/gnopreview/main.go:161-165 names?
#
#   "The RPC listener and the keybase both default to fixed locations
#    (127.0.0.1:26657 and $GNOHOME), so the before/after passes — which run at
#    the same time — would collide on them. Derive both from the web port instead."
#
# tm2 sets a THIRD fixed address by default, the P2P listener:
#   tm2/pkg/bft/config/config.go:154  cfg.ListenAddress = "tcp://0.0.0.0:26656"
# reached from gnodev through DefaultNodeConfig -> gnoland.NewDefaultTMConfig ->
# tmcfg.TestConfig() -> testP2PConfig(), and gnodev overrides only
# Consensus.* and P2P.PeerExchange. If that listener were bound, the second
# gnodev would fail and every "before" image would be lost.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
#   bash <this file>
#
# Result at ecf7af0: NOT a defect. The in-memory dev node never starts a P2P
# listener, so nothing binds 2665x and both nodes serve concurrently.
set -eu
ROOT="$(git rev-parse --show-toplevel)"
BIN="$(mktemp -d)/gnodev"; T="$(mktemp -d)"
( cd "$ROOT/contribs/gnodev" && go build -o "$BIN" . )
boot() { # $1 web port, $2 rpc port, $3 home
  GNOROOT="$ROOT" "$BIN" local -no-watch \
    -web-listener "127.0.0.1:$1" -node-rpc-listener "tcp://127.0.0.1:$2" \
    -home "$3" -C "$ROOT/examples" "$ROOT/examples/gno.land/r/gnoland/home" \
    > "$3.log" 2>&1 &
  for _ in $(seq 1 60); do curl -sf -o /dev/null "http://127.0.0.1:$1/r/gnoland/home" && return 0; sleep 2; done
  return 1
}
boot 18888 28888 "$T/h1" && echo "node1 http: $(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18888/r/gnoland/home)"
echo "listeners on 2665x while node1 runs:"; ss -ltn | grep -E '2665[0-9]' || echo "  NOTHING on 2665x"
boot 18889 28889 "$T/h2" && echo "node2 http: $(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18889/r/gnoland/home)"
pkill -f "$BIN" || true
