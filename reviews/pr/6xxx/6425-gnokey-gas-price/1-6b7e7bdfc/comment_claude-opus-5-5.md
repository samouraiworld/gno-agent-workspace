# Review: [#6425](https://github.com/gnolang/gno/pull/6425)
Posted: https://github.com/gnolang/gno/pull/6425#pullrequestreview-5974584286
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The gas-use cache in [gas_use.go](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L122) drops a good measure on a slow indexer read, a Warning the branch introduces; the rest are Suggestions and Nits.
Model: claude-opus-5-5, standard review
Commit: 6b7e7bdfc61f4db559693bfdc521da564f364f41
Overview: [overview.md](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6425-gnokey-gas-price/overview.md)
Open the code: [gno.land/pkg/gnoweb at the head](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go)
Round: 1. 3 finders, one reflector, 9 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## Body

- Related suggestion: [RunGas](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/view_action.go#L68) keeps a fixed 100000ugnot fee for the `$run` command that [the run handler](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/feature/run/handler.go#L21) prints. The chain rejects that script at CheckTx once its price passes 2ugnot per 1000 gas.

  <details><summary>check</summary>

  ```
  grep -rn 'RunGas' gno.land/pkg/gnoweb --include='*.go' | grep -v _test
  ```
  </details>

## gno.land/pkg/gnoweb/gas_use.go:122 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L122) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/gas_use.go#L122) · Warning [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628381964)

A partial indexer read with zero rows counts as success. One slow read then wipes a package's last gas measure for five minutes, and its commands fall back to the 10 M default.

<details><summary>repro</summary>

The test keeps rows only in the deepest band and makes every band take 3.2 s on the second read; the first read measures `Deposit`, the second wipes it.

```
fast read: Deposit = {Max:1400000 Calls:1}
slow read: Deposit = {Max:0 Calls:0}, fresh=true
--- FAIL (9.60s)
```

Test file: [judge-4-partial-band-drops-measure_test.go](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6425-gnokey-gas-price/1-6b7e7bdfc/tests/judge-4-partial-band-drops-measure_test.go), copied into `gno.land/pkg/gnoweb` and run with `go test -run TestJudge4PartialBandDropsMeasure -count=1`.
</details>

## SKIP gno.land/pkg/gnoweb/client.go:251 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L251) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/client.go#L251) · Missing test

Missing test: no test fails when [queryGasPrice](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L251) drops `context.WithoutCancel`, since the stub caller ignores its context.

Not posted: a PLAUSIBLE Missing test that no verifier ran.

## SKIP gno.land/pkg/gnoweb/components/gas_test.go:138 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas_test.go#L138) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/components/gas_test.go#L138) · Missing test

Missing test: the foreign-denom block in [gas_test.go](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas_test.go#L138) asserts that a measure never raises either mode and leaves the lowering of the fast mode unpinned.

Not posted: a PLAUSIBLE Missing test that no verifier ran.

## SKIP gno.land/pkg/gnoweb/gas_use.go:97 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L97) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/gas_use.go#L97) · Missing test

Missing test: no test fails when the background read in [gas_use.go](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L97) drops `context.WithoutCancel`, since the stub indexer ignores its context.

Not posted: a PLAUSIBLE Missing test that no verifier ran.

## SKIP gno.land/pkg/gnoweb/handler_http_test.go:1181 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/handler_http_test.go#L1181) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/handler_http_test.go#L1181) · Missing test

Missing test: the node-down row in [handler_http_test.go](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/handler_http_test.go#L1181) pins the fee and never the note saying the chain's price was unavailable.

Not posted: a PLAUSIBLE Missing test that no verifier ran.

## gno.land/pkg/gnoweb/components/ui/command.html:18 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/ui/command.html#L18) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/components/ui/command.html#L18) · Nit [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628381976)

Nit: the note rounds the gas cap up, printing 38 M beside `-gas-wanted 37_600_000`, because [FormatGas](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas.go#L25) drops the decimal from 10 M upward.

<details><summary>check</summary>

```go
FormatGas(37_600_000) // "38 M"
NewTxGas(FallbackGasPrice, GasUse{Max: 18_800_000, Calls: 13}).Secure.WantedFlag() // "37_600_000"
```
</details>

## gno.land/pkg/gnoweb/markdown/context.go:60 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/markdown/context.go#L60) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/markdown/context.go#L60) · Nit [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628381981)

Nit: no reader takes the gas lookup copied into each `<gno-foreign>` context, since [the foreign parser](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L431) loads no forms.

## SKIP gno.land/pkg/gnoweb/client.go:235 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L235) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/client.go#L235) · Suggestion

Suggestion: [GasPrice](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L235) serves the last good price with no bound on its age.

Not posted: carried by the section on client.go:261, and no judge ran it.

## gno.land/pkg/gnoweb/client.go:261 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L261) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/client.go#L261) · Suggestion [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628381996)

Suggestion: a failed refresh resets `gasAt` and keeps the old price, so the page labels a price of any age as the chain's own. I think GasPrice should return an error past a few TTLs, so the page shows the estimated label.

## gno.land/pkg/gnoweb/components/gas.go:35 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas.go#L35) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/components/gas.go#L35) · Suggestion [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628382004)

Suggestion: a call with no indexer data gets `-gas-wanted 10_000_000`, and [-indexer-url defaults to empty](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/cmd/gnoweb/main.go#L152). A call above 10 M then fails its simulation with no figure in the note to raise to; I think the note should name the ceiling and the fee per 1000 gas.

<details><summary>check</summary>

```
grep -n 'TxGasWanted = ' gno.land/pkg/gnoweb/components/gas.go
grep -n 'indexer-url' -A3 gno.land/cmd/gnoweb/main.go
```
</details>

## gno.land/pkg/gnoweb/components/gas.go:135 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas.go#L135) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/components/gas.go#L135) · Suggestion [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628382012)

Suggestion: the raise case has no `minGasUseCalls` check, and [gasUses](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L61) keeps the maximum. Any single 400 M call raises that function's command to the 1 GNOT cap for every viewer.

<details><summary>check</summary>

```go
uses := gasUses([]indexer.Tx{callTx(400_000_000, true, call("F")), callTx(1_000_000, true, call("F")), callTx(1_000_000, true, call("F"))}, testPkg)
g := components.NewTxGas(components.FallbackGasPrice, uses["F"])
// g.Fast.Fee.Amount == 1_000_000, g.Fast.Capped == true
```
</details>

## SKIP gno.land/pkg/gnoweb/components/ui/command.html:21 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/ui/command.html#L21) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/components/ui/command.html#L21) · Suggestion

Suggestion: the dry run and the broadcast sit on [separate lines](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/ui/command.html#L15-L16) with no `&&`. A pasted block broadcasts anyway, and the chain charges the fee.

Not posted: gfanton's [open thread](https://github.com/gnolang/gno/pull/6425#discussion_r4626768204) asks for the same `&&` and blocks on it.

## gno.land/pkg/gnoweb/markdown/ext_forms.go:254 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/markdown/ext_forms.go#L254) · [↗](../../../../../.worktrees/gno-review-6425/gno.land/pkg/gnoweb/markdown/ext_forms.go#L254) · Suggestion [posted](https://github.com/gnolang/gno/pull/6425#discussion_r4628382029)

Suggestion: a form on a `/u/` page looks its gas up under `gno.land/u/<name>`, a path no call targets. [GetUserView](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/handler_http.go#L1138) renders `/r/<name>/home` with the `/u/` URL, so the measure never applies and each page spends an indexer read.
