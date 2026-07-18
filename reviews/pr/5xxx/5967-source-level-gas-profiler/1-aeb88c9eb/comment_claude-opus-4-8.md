# Review: PR [#5967](https://github.com/gnolang/gno/pull/5967)
Event: REQUEST_CHANGES

## Body
Verified on aeb88c9eb. Booted gnodev from this branch and ran `gnokey maketx call -profile` end to end. Once the target package is loaded the profile is correct and useful, naming `foo20.Faucet` and the `grc20` and `avl` frames beneath it. On a fresh node the same command fails, which is the inline comment below.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5967-source-level-gas-profiler/1-aeb88c9eb/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## contribs/gnodev/pkg/proxy/path_interceptor.go:316 [↗](../../../../../.worktrees/gno-review-5967/contribs/gnodev/pkg/proxy/path_interceptor.go#L316)
`.app/profiletx` is missing from this switch, so it falls to `default` and no package is lazy-loaded. gnodev is the only node shipping the profiler enabled and has lazy loading on by default, so a first `gnokey maketx -profile` against any not-yet-loaded package runs on an unloaded package, reports `partial profile: tx did not complete: internal error`, and writes a profile holding only `(ante)` and `(root)`. Add `.app/profiletx` alongside `.app/simulate`, which already routes the identical tx bytes through `handleTx`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5967 -R gnolang/gno
go build -o /tmp/gnodev ./contribs/gnodev && go build -o /tmp/gnokey ./gno.land/cmd/gnokey
GNOROOT=$(pwd) /tmp/gnodev --no-watch -v --node-rpc-listener 127.0.0.1:26657 > /tmp/gnodev.log 2>&1 &
sleep 25

KEY="-insecure-password-stdin -home /tmp/kh -remote 127.0.0.1:26657 -chainid dev"
mkdir -p /tmp/kh
printf 'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast\n\n\n' \
  | /tmp/gnokey add devtest --home /tmp/kh --recover --insecure-password-stdin

TX="-pkgpath gno.land/r/demo/defi/foo20 -func Faucet -gas-fee 1000000ugnot -gas-wanted 20000000"
# A: profile on a fresh node
/tmp/gnokey maketx call $TX -profile /tmp/a.pprof $KEY devtest <<< ""
grep -o 'unhandled: ".app/profiletx"' /tmp/gnodev.log | head -1
# B: same tx via .app/simulate, which does trigger the load
/tmp/gnokey maketx call $TX -broadcast -simulate only $KEY devtest <<< "" > /dev/null
# C: identical profile command now succeeds
/tmp/gnokey maketx call $TX -profile /tmp/c.pprof $KEY devtest <<< ""
go tool pprof -sample_index=total_gas -top -nodecount=3 /tmp/c.pprof

pkill -f /tmp/gnodev
```

```
A: gas profile written to /tmp/a.pprof (partial profile: tx did not complete: internal error)
unhandled: ".app/profiletx"
C: gas profile written to /tmp/c.pprof (ok)
      flat  flat%   sum%        cum   cum%
2581927gas 48.28% 48.28% 3730856gas 69.76%  gno.land/r/demo/defi/foo20.Faucet
 994511gas 18.60% 66.87% 5348238gas   100%  (root)
 622871gas 11.65% 78.52%  622871gas 11.65%  (ante)
```
</details>

## tm2/pkg/crypto/keys/client/maketx.go:148 [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx.go#L148)
Nit: the same profiler is `-gasprofile` on `gno test` and `-profile` here, so someone who learns the first gets "flag provided but not defined" on the second. The `gno test` name is motivated, since the `gno` binary already uses `CPUPROFILE` for Go-level profiling and a bare `-profile` would be ambiguous there, but gnokey has no such conflict to resolve. Naming this one `-gasprofile` too is cheap now and expensive after merge, since flag names are API.

## tm2/pkg/crypto/keys/client/maketx.go:385 [↗](../../../../../.worktrees/gno-review-5967/tm2/pkg/crypto/keys/client/maketx.go#L385)
Suggestion: `-profile` together with `-broadcast` writes the profile and returns before broadcasting, exiting 0, so against a dev node the intended transaction is dropped. `Validate` rejects no such combination. Reject both flags together, or note on stderr that `-broadcast` is ignored under `-profile`.
