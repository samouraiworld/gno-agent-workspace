# Review: PR [#5967](https://github.com/gnolang/gno/pull/5967)
Posted: https://github.com/gnolang/gno/pull/5967#pullrequestreview-4728324243
Event: COMMENT

## Body
[AI bot]

Verified on aeb88c9eb: booted gnodev from this branch and ran `gnokey maketx call -profile` end to end. Once the package is loaded the profile is good, naming `foo20.Faucet` with the `grc20` and `avl` frames beneath it.

On a fresh node it fails. `.app/profiletx` is missing from the `handleQuery` switch in [`contribs/gnodev/pkg/proxy/path_interceptor.go`](https://github.com/gnolang/gno/blob/aeb88c9eb/contribs/gnodev/pkg/proxy/path_interceptor.go#L314-L327), so the package never lazy-loads. gnodev enables both the profiler and lazy loading by default, so this hits the first `-profile` of any package: it reports `partial profile: tx did not complete: internal error` and still writes a profile holding only `(ante)` and `(root)`. Add `.app/profiletx` to the `.app/simulate` case, which already routes the same tx bytes through `handleTx`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5967 -R gnolang/gno
mkdir -p /tmp/gnoprof/kh
go build -o /tmp/gnoprof/gnokey ./gno.land/cmd/gnokey
# gnodev is its own Go module, so build it from its own directory
(cd contribs/gnodev && go build -o /tmp/gnoprof/gnodev .)

GNOROOT=$(pwd) /tmp/gnoprof/gnodev --no-watch -v \
  --node-rpc-listener 127.0.0.1:26657 > /tmp/gnoprof/gnodev.log 2>&1 &
sleep 25

printf 'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast\n\n\n' \
  | /tmp/gnoprof/gnokey add devtest --home /tmp/gnoprof/kh --recover --insecure-password-stdin

# A: profile a package the node has not loaded yet
/tmp/gnoprof/gnokey maketx call -pkgpath gno.land/r/demo/defi/foo20 -func Faucet \
  -gas-fee 1000000ugnot -gas-wanted 20000000 -profile /tmp/gnoprof/a.pprof \
  -insecure-password-stdin -home /tmp/gnoprof/kh -remote 127.0.0.1:26657 -chainid dev devtest <<< ""
grep -o 'unhandled: [^ ]*profiletx[^ ]*' /tmp/gnoprof/gnodev.log | head -1

# B: same tx through .app/simulate, which does trigger the load
/tmp/gnoprof/gnokey maketx call -pkgpath gno.land/r/demo/defi/foo20 -func Faucet \
  -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -simulate only \
  -insecure-password-stdin -home /tmp/gnoprof/kh -remote 127.0.0.1:26657 -chainid dev devtest <<< ""

# C: the identical -profile command from A, now that the package is loaded
/tmp/gnoprof/gnokey maketx call -pkgpath gno.land/r/demo/defi/foo20 -func Faucet \
  -gas-fee 1000000ugnot -gas-wanted 20000000 -profile /tmp/gnoprof/c.pprof \
  -insecure-password-stdin -home /tmp/gnoprof/kh -remote 127.0.0.1:26657 -chainid dev devtest <<< ""
go tool pprof -sample_index=total_gas -top -nodecount=3 /tmp/gnoprof/c.pprof

pkill -f /tmp/gnoprof/gnodev
```

```
A: gas profile written to /tmp/gnoprof/a.pprof (partial profile: tx did not complete: internal error)
   unhandled: \".app/profiletx\""
B: estimated gas usage: 5073498 (suggested, with 5% margin: 5327173)
C: gas profile written to /tmp/gnoprof/c.pprof (ok)
         flat  flat%   sum%        cum   cum%
   2581927gas 48.28% 48.28% 3730856gas 69.76%  gno.land/r/demo/defi/foo20.Faucet
    994511gas 18.60% 66.87% 5348238gas   100%  (root)
    622871gas 11.65% 78.52%  622871gas 11.65%  (ante)
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5967-source-level-gas-profiler/1-aeb88c9eb/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/crypto/keys/client/maketx.go:148 [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx.go#L148) [posted](https://github.com/gnolang/gno/pull/5967#discussion_r3609097950)
Nit: this is `-gasprofile` on `gno test` but `-profile` here, so learning one gives "flag provided but not defined" on the other. Worth aligning on `-gasprofile` before merge, since flag names are API.

## tm2/pkg/crypto/keys/client/maketx.go:385 [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx.go#L385) [posted](https://github.com/gnolang/gno/pull/5967#discussion_r3609097972)
Suggestion: `-profile` with `-broadcast` writes the profile and returns before broadcasting, exiting 0, so the intended transaction is silently dropped. Reject the combination, or note on stderr that `-broadcast` is ignored.
