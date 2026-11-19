# Claims: gnolang/gno#6425 round 1, 6b7e7bdfc, claude-opus-5-5, standard review

Round shape: 3 finders, one reflector, 9 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted.

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`gas_use.go:122`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/gas_use.go#L122) | 4 | Warning | CONFIRMED | differs | posts |
| [`client.go:251`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/client.go#L251) | 12 | Missing test | PLAUSIBLE |  | SKIP |
| [`gas_test.go:138`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/gas_test.go#L138) | 10 | Missing test | PLAUSIBLE |  | SKIP |
| [`gas_use.go:97`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/gas_use.go#L97) | 11 | Missing test | PLAUSIBLE |  | SKIP |
| [`handler_http_test.go:1181`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/handler_http_test.go#L1181) | 14 | Missing test | PLAUSIBLE |  | SKIP |
| [`command.html:18`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/ui/command.html#L18) | 3 | Nit | CONFIRMED | differs | posts |
| [`context.go:60`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/markdown/context.go#L60) | 6 | Nit | CONFIRMED | differs | posts |
| [`client.go:235`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/client.go#L235) | 13 | Suggestion | UNVERIFIED |  | SKIP |
| [`client.go:261`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/client.go#L261) | 5 | Suggestion | CONFIRMED | differs | posts |
| [`gas.go:35`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/gas.go#L35) | 1 | Suggestion | CONFIRMED | differs | posts |
| [`gas.go:135`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/gas.go#L135) | 9 | Suggestion | CONFIRMED | differs | posts |
| [`command.html:21`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/ui/command.html#L21) | 2 | Suggestion | CONFIRMED | differs | posts |
| [`view_action.go:68`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/view_action.go#L68) | 8 | Suggestion | CONFIRMED | same | posts |
| [`ext_forms.go:254`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/markdown/ext_forms.go#L254) | 7 | Suggestion | CONFIRMED | differs | posts |

### 4 · Warning · CONFIRMED · `gno.land/pkg/gnoweb/gas_use.go:122`

A slow indexer's empty ErrPartial read replaces the last gas measure with nothing for 5 minutes

- Check: cp reviews/pr/6xxx/6425-gnokey-gas-price/1-6b7e7bdfc/tests/b2-lines-reach-catalog-partial-drops-measure_test.go gno.land/pkg/gnoweb/zz_b2_partial_drops_measure_test.go && go test ./gno.land/pkg/gnoweb/ -run TestB2PartialEmptyDropsMeasure -v -count=1: expect the last measure kept, observe {Max:0 Calls:0} fresh for 5m0s
- Observed: Head, finder artifact rerun: 'RecentByPackage: 0 rows, err=searched the most recent blocks only, took 9.6s' / 'after a slow read: Deposit = {Max:0 Calls:0}, entry fresh=true, expires in 5m0s' / FAIL. Head, judge artifact with a truthful indexer (rows only in the band gt: 8830000, every band 3.2 s on the second read): 'fast read: Deposit = {Max:1400000 Calls:1}' then 'slow read: Deposit = {Max:0 Calls:0}, fresh=true' / FAIL (9.60s). Base: gas_use.go does not exist at 5e9668dab, no gas-use cache to compare.
- Test: [`tests/judge-4-partial-band-drops-measure_test.go`](tests/judge-4-partial-band-drops-measure_test.go)

### 12 · Missing test · PLAUSIBLE · `gno.land/pkg/gnoweb/client.go:251`

No test pins queryGasPrice's context.WithoutCancel: gasCaller.SendRequest ignores its ctx, so the stale-while-revalidate refresh and the shared first read stay green when the request context is passed through

- Check: apply the mutation and run go test ./gno.land/pkg/gnoweb/ -run 'GasPrice' -race -count=1: expect TestGasPriceFirstReadLeavesOnCancel or the stale-while-revalidate test red, observe green unless the RPC client checks ctx after SendRequest returns
- Observed: not run: no verifier reached it; the check is still to run

### 10 · Missing test · PLAUSIBLE · `gno.land/pkg/gnoweb/components/gas_test.go:138`

The foreign-denom block says 'a measure lowers the fast-mode gas but never raises either mode' and pins only the no-raise half

- Check: add `&& g.Price.Price.Denom == "ugnot"` to `case w < TxGasWanted && use.Calls >= minGasUseCalls:` in components/gas.go and run go test ./gno.land/pkg/gnoweb/components -run TestNewTxGas: expect red, observe green
- Observed: not run: no verifier reached it; the check is still to run

### 11 · Missing test · PLAUSIBLE · `gno.land/pkg/gnoweb/gas_use.go:97`

No test pins the detached context of the background gas-use read: stubRecent.recent ignores its ctx and every test passes context.Background(), so dropping context.WithoutCancel stays green

- Check: apply the mutation and run go test ./gno.land/pkg/gnoweb/ -run 'GasUse' -race -count=1: expect a red test, observe green; the test to add hands get a context cancelled right after the call and a stub that returns ctx.Err() when it is done
- Observed: not run: no verifier reached it; the check is still to run

### 14 · Missing test · PLAUSIBLE · `gno.land/pkg/gnoweb/handler_http_test.go:1181`

The node-down row pins only the fee, so nothing at the handler pins that a failed GasPrice reads as 'the chain's price was unavailable' rather than as the chain's own price

- Check: in gno.land/pkg/gnoweb/page_kind.go gasPrice replace `return std.GasPrice{}` with `return components.FallbackGasPrice`, run `go test ./gno.land/pkg/gnoweb -run TestHTTPHandler_CommandGas`: expect red, observe green; grep -n 'unavailable' handler_http_test.go shows no assertion on the gas note
- Observed: not run: no verifier reached it; the check is still to run

### 3 · Nit · CONFIRMED · `gno.land/pkg/gnoweb/components/ui/command.html:18`

The note rounds the gas cap up: 'for up to 38 M gas' beside -gas-wanted 37_600_000

- Check: go test snippet in components: FormatGas(37_600_000) == "38 M" and FormatGas(33_900_000) == "34 M", against NewTxGas(FallbackGasPrice, GasUse{Max: 18_800_000, Calls: 13}).Secure.WantedFlag() == "37_600_000"
- Observed: Read gas.go:22-25: `case math.Round(f/1e5) < 100: ... 'f', 1 ...` / `case math.Round(f/1e6) < 1_000: return strconv.FormatFloat(f/1e6, 'f', 0, 64) + " M"`; gas.go:129 `w = ... (2*use.Max+gasWantedStep-1)/gasWantedStep*gasWantedStep` gives 37,600,000 for Max 18,800,000.

### 6 · Nit · CONFIRMED · `gno.land/pkg/gnoweb/markdown/context.go:60`

getGnoContext copies the gas lookup into <gno-foreign> content, whose parser never loads forms, so the copy is never read

- Check: Run grep -rn 'gGasContextKey\\|getGasFromContext(' gno.land/pkg/gnoweb/markdown --include='*.go' \| grep -v _test. The only readers are context.go and ext_forms.go:254. Then delete `gas, _ := ...` and `Gas: gas,` from getGnoContext and run go test ./gno.land/pkg/gnoweb/...: it stays green.
- Observed: grep gGasContextKey/getGasFromContext( in markdown non-test: context.go:14,43,60,95-96 and ext_forms.go:254 only; ext_foreign.go:438-460 extension list has no ExtForms.

### 13 · Suggestion · UNVERIFIED · `gno.land/pkg/gnoweb/client.go:235`

GasPrice serves the last good price with no bound on its age: a refresh that keeps failing resets gasAt and keeps gasOK, so an hours-old price is shown as current with Estimated false

- Check: read TestGasPriceStaleWhileRevalidate's last block: a failed refresh keeps 2ugnot; repeat caller.set("", nil); c.expire(); c.GasPrice; waitRefresh N times and observe the old price returned with err nil every time
- Observed: not run: on the finder's read

### 5 · Suggestion · CONFIRMED · `gno.land/pkg/gnoweb/client.go:261`

Once one gas price is read, failed refreshes keep serving it forever as the chain's current price

- Check: read TestGasPriceStaleWhileRevalidate's last block: a failed refresh keeps 2ugnot; repeat caller.set("", nil); c.expire(); c.GasPrice; waitRefresh N times and observe the old price returned with err nil every time
- Observed: Read client.go:259-266: `c.gasAt = time.Now()` / `if err != nil { ... return std.GasPrice{}, ... }` / `c.gasPrice, c.gasOK = gp, true`; GasPrice:241-244 returns gp, nil whenever ok.

### 1 · Suggestion · CONFIRMED · `gno.land/pkg/gnoweb/components/gas.go:35`

Without indexer data, a copied call needing over 10 M gas fails where base's 1,000 M command went through, and the note gives no figure to raise to

- Check: git show edeca7758 (message quoted); grep -n 'indexer-url' -A3 gno.land/cmd/gnoweb/main.go (default empty); in a gnoland integration txtar, call a realm function that burns ~15 M gas with the head command's flags (-gas-fee 30000ugnot -gas-wanted 10_000_000): out of gas at simulation; with the base flags (-gas-fee 1000000ugnot -gas-wanted 1_000_000_000): succeeds
- Observed: Read: base components/view_action.go:62 `CallGas = Gas{Wanted: "1_000_000_000", Fee: "1000000ugnot"}`; head gas.go:35 `const TxGasWanted = 10_000_000`, gas.go:119-121 return the 10 M fee when use.Calls == 0; cmd/gnoweb/main.go:152 'Empty (the default) keeps gnoweb talking only to its RPC node'; git log -1 edeca7758: 'many gnoweb transactions can't simply be copy-pasted as gnokey commands'.

### 9 · Suggestion · CONFIRMED · `gno.land/pkg/gnoweb/components/gas.go:135`

One costly successful call by anyone raises every viewer's copied command for that function to the 1 GNOT cap until 20 newer calls push it out

- Check: go test snippet: uses := gasUses([]indexer.Tx{callTx(400_000_000,true,call("F")), callTx(1_000_000,true,call("F")), callTx(1_000_000,true,call("F"))}, testPkg); g := components.NewTxGas(components.FallbackGasPrice, uses["F"]); expect g.Fast.Fee.Amount == 1_000_000 and g.Fast.Capped
- Observed: Read gas.go:128-137: `if use.Max < most/2 { w = ... 2*use.Max ... }` then `case w > TxGasWanted: g.Secure, g.Fast, g.Used = fee, fee, use`; maxWanted(FallbackGasPrice) = 1e6*1000/1 capped to 1_000_000_000; txFee(fallback, 800_000_000) = 2,400,000 > maxUgnotFee, ceilTimes(1) = 800,000 <= 1,000,000, so Fee 1000000ugnot Capped; gas_use.go:61 `u.Max = max(u.Max, int64(tx.GasUsed))`.

### 2 · Suggestion · CONFIRMED · `gno.land/pkg/gnoweb/components/ui/command.html:21`

Pasting the secure script whole runs the broadcast even when the dry run fails, so the note's 'the dry run stops that first' does not hold and the fee is charged

- Check: copy gnoweb_airgapped.txtar; after `! gnokey broadcast -dry-run $WORK/low.tx` add `! gnokey broadcast $WORK/low.tx` then query auth/accounts: sequence 1 and coins lower by 230001ugnot, the charge the note says the dry run prevents \| Read command.html:15-16 and :20. In a shell, `false` then `echo broadcast` on two lines prints broadcast, while `false && echo broadcast` does not. The goldens under markdown/golden/ext_forms change along with the template.
- Observed: Read command.html:15 `gnokey broadcast -dry-run -remote "{{.Remote}}" call.tx`, :16 `gnokey broadcast -remote "{{.Remote}}" call.tx`, :21 'The broadcast does not simulate, so a call that runs out of gas is still charged: the dry run before it stops that first.' No `&&` or `set -e` in the block.

### 8 · Suggestion · CONFIRMED · `gno.land/pkg/gnoweb/components/view_action.go:68`

The $run command still prints a fixed 100000ugnot fee, so it does not follow the chain's gas price like the call commands now do

- Check: grep -rn 'RunGas' gno.land/pkg/gnoweb --include='*.go' \| grep -v _test finds only view_action.go:68 and feature/run/handler.go:21. Render the run page with a stub GasPrice of {Gas:1000, Price:5ugnot}: the command still carries 100000ugnot.
- Observed: Read head view_action.go:68 `RunGas = Gas{Wanted: "50_000_000", Fee: "100000ugnot"}`; grep finds only feature/run/handler.go:21 `Gas: components.RunGas` as a non-test user; base view_action.go:62 had the swept `CallGas` beside it.

### 7 · Suggestion · CONFIRMED · `gno.land/pkg/gnoweb/markdown/ext_forms.go:254`

On a /u/ page, a form's gas is looked up under gno.land/u/<name>, which never has calls, so the measured gas never applies and an indexer read is spent for nothing

- Check: Write a handler test: the stubClient realmFunc returns "<gno-form exec=\"Bump\">\n</gno-form>\n" for "/r/<addr>/home", and cfg.Indexer = &gasIndexer{}. GET /u/<addr>. Expect askedPaths ["gno.land/r/<addr>/home"]; observe ["gno.land/u/<addr>"], with the body containing -pkgpath "gno.land/u/<addr>".
- Observed: Read handler_http.go:1132-1138 `raw, err := h.Client.Realm(ctx, "/r/"+ns+"/home", "")` then `h.Renderer.RenderRealm(&content, gnourl, raw, ...)`; ext_forms.go:232 `node.RealmName = gnourl.Path`, :254 `getGasFromContext(pc, node.Domain+node.RealmName, exec)`; page_kind.go:259 `components.NewTxGas(price(), h.gasUse.get(ctx, pkgPath)[fn])`; gas_use.go:54 filters `m.PkgPath != pkgPath`.

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`pr6425_gnoweb_gas_fee.md:25`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/adr/pr6425_gnoweb_gas_fee.md#L25) | claims | ADR: tm2 moves the price at least one unit per busy block; a full block raises it ~4.3% (compressor 10, target 70%); x3 covers ~26 full blocks | tm2/pkg/sdk/auth/keeper.go:458 `diff := maxBig(num, bigOne)` in the increase branch; params.go:25-26 DefaultGasPricesChangeCompressor 10, DefaultTargetGasRatio 70; (1-0.7)/0.7/10 = 4.29%, ln3/ln1.0429 = 26.2 |
| [`pr6425_gnoweb_gas_fee.md:105`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/adr/pr6425_gnoweb_gas_fee.md#L105) | claims | ADR: the dry run skips the price check (ante.go, !simulate) | tm2/pkg/sdk/auth/ante.go:84 `if ctx.IsCheckTx() && !simulate {` guards EnsureSufficientMempoolFees |
| [`command.html:15`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/ui/command.html#L15) | rollout | A user's older gnokey lacks broadcast -dry-run ('a flag every gnokey has') | tm2/pkg/crypto/keys/client/broadcast.go:68-71 registers "dry-run"; git log -S DryRun on that file: earliest 5ad12cd2b 2024-01-15 |
| [`command.html:18`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/ui/command.html#L18) | claims | Fast note: 'gnokey simulates first' for the fast command, which carries no -broadcast flag | tm2/pkg/crypto/keys/client/maketx.go:96-110 `"broadcast", true` and `"simulate", "test"` defaults; broadcast.go:133-136 only DryRun or testSimulate simulate, so the standalone broadcast does not (secure note holds too) |
| [`pr6425_gnoweb_gas_fee.md:47`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/adr/pr6425_gnoweb_gas_fee.md#L47) | claims | ADR constants: last 20 calls, last 100 txs, 5 min TTL, 30 s after failure, 4 reads, 1,024 packages, 64-byte func, 30 s price TTL, 2 s wait, single successful MsgCall with positive gas | gas_use.go:20-40 gasUseTxs 100, gasUseCalls 20, gasUseTTL 5m, gasUseRetry 30s, maxGasUseReads 4, maxGasUsePkgs 1024, maxGasUseFunc 64; :50-55 Success, GasUsed > 0, one message, MsgCall; client.go:162 gasPriceTTL 30s, :165 gasPriceTimeout 2s |
| [`controller-action-function.ts:83`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/frontend/js/controller-action-function.ts#L83) | rollout | After a deploy a cached old controller-action-function.js would leave the notes not following the mode | cache.go CacheHandler: unversioned module imports such as controller.js 'are revalidated against the ETag, so a deploy never leaves a stale copy'; only ?v=<this version> or content-hashed files get immutable |
| [`main.css:1`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/public/main.css#L1) | lines | Built assets out of step with frontend sources (.note rule, note toggle) | grep -c 'b-code .note' public/main.css: 1; public/js/controller-action-function.js contains `getTargets("note").forEach(e=>{e.classList.toggle("u-hidden",this.getValue("mode",e)!==t)})` |
| [`view_action.go:23`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/view_action.go#L23) | removed | Removing CallGas and the callGas template func leaves a stale reference or doc | grep -rn 'callGas\|CallGas' over the head tree: no hit; grep of docs/ for gnoweb gas, $help and Actions page fee text: no hit |
| [`view_action.go:134`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/view_action.go#L134) | reach | A CommandData built without Gas prints `-gas-fee -gas-wanted 0` (Coin.String is empty for zero) | only two constructions: view_action.go:134 `Gas: fn.Gas` with handler_http.go:1286 `Gas: components.NewTxGas(price, uses[fn.Name])` for every function, and ext_forms.go:610 `Gas: n.Gas` set at :254 by getGasFromContext, which falls back to NewTxGas(std.GasPrice{}, ...); NewTxGas always yields a fee >= 1 |
| [`command.html:25`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/ui/command.html#L25) | reach | A markdown render path without the Gas lookup would print 'the chain's price was unavailable' while it is available | all RenderRealm calls pass h.renderContext (handler_http.go:739, 856, 1138, 1317), which sets Gas; ext_foreign.go:382 rebuilds from GnoCtx, which context.go now carries Gas into |
| [`gas.go:131`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/gas.go#L131) | catalog | Coin math / swallowed result: `fee, _ = txFee(g.Price, w)` could drop an overflow or an over-cap price | for ugnot w <= maxWanted = floor(1e6*Gas/Amount) so ceil(w*A/G) <= 1e6 and x3 <= 3e6 fits int64; for a foreign denom maxWanted returns TxGasWanted so the fee is at most the 10 M fee txFee already accepted; test cases 'fee overflows' and 'at the old gas' cover both ends |
| [`gas.go:55`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/gas.go#L55) | catalog | Global mutable state: exported package-level var FallbackGasPrice read by parallel tests | grep 'FallbackGasPrice *=' over the tree: only the declaration at gas.go:55; never reassigned at runtime |
| [`client.go:215`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/client.go#L215) | catalog | Global mutable state & concurrency: the new gas price cache, gas-use LRU, atomics and background goroutines race under -race | go test -race -count=1 ./gno.land/pkg/gnoweb/ -> ok github.com/gnolang/gno/gno.land/pkg/gnoweb 45.537s; go test -race -run 'GasPrice\|GasUse' -count=50 -> ok 1.783s |
| [`client_test.go:552`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/client_test.go#L552) | lines | TestGasPriceFirstReadIsShared/LeavesOnCancel flake: a caller whose snapshot predates the cache write but whose DoChan follows singleflight's key removal starts a second query | go test -race -run 'FirstReadIsShared\|FirstReadLeaves\|StaleWhile' -count=1000 -cpu 1,4,16 ./gno.land/pkg/gnoweb/ -> ok 45.608s |
| [`gas_use.go:80`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/gas_use.go#L80) | reach | rpcClient and gasUseCache carry a Mutex, atomics and a singleflight.Group by value inside HTTPHandler: a copy would split the lock | go vet ./gno.land/pkg/gnoweb/ -> vet-exit=0 (copylocks clean) |
| [`gas_use.go:122`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/gas_use.go#L122) | reach | An indexer band slower than the client timeout drops the last measure | repro with 4.6 s bands: RecentByPackage: 0 rows, err=Post ...: context deadline exceeded (Client.Timeout exceeded while awaiting headers); after a slow read: Deposit = {Max:1400000 Calls:9}, expires in 30s -> PASS (error path keeps the measure) |
| [`handler_http.go:233`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/handler_http.go#L233) | lines | a typed-nil Indexer would make gasUse.recent a method value on a nil client and panic in the background read | grep -n Indexer gno.land/pkg/gnoweb/app.go: the only assignment is app.go:247 `handlerCfg.Indexer = indexer.New(cfg.IndexerURL, cfg.IndexerToken)` inside `if cfg.IndexerURL != ""` |
| [`handler_http_test.go:776`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/handler_http_test.go#L776) | catalog | the background gas-use read could race the measured cases or ask the indexer twice (Global mutable state & concurrency class) | go test -race -count=3 -run 'TestHTTPHandler_CommandGas\|TestRenderRealm_FormCommand' ./gno.land/pkg/gnoweb/ in a detached head worktree: `ok github.com/gnolang/gno/gno.land/pkg/gnoweb 1.273s`; go test ./gno.land/pkg/gnoweb/markdown/: `ok ... 13.223s` |
| [`handler_http_test.go:609`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/handler_http_test.go#L609) | reach | ClientAdapter gained GasPrice, breaking implementers outside gnoweb | grep -rn 'func (.*) PublicEval(ctx' --include='*.go' .: only client.go:203, client_mock.go:244, handler_http_test.go:71, all in package gnoweb |
| [`command.html:15`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/components/ui/command.html#L15) | rollout | a reader's older gnokey may lack broadcast -dry-run or maketx simulation, so the copy's free-failure promise would not hold | git log -S'"dry-run"' -- tm2/pkg/crypto/keys/client/broadcast.go: earliest 98cc986cb 2023-04-03, contained in chain/gnoland1.0 onward; git log -S'SimulateTest' maketx.go: 3ea1b47e2 2024-05-13, default "test" at maketx.go:107 |
| [`handler_http.go:486`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/handler_http.go#L486) | lines | a non-crossing function on $help would render a command priced at the fallback and labelled 'unavailable' since only crossing functions trigger the price read | grep -n 'ui/command' components/views/action.html: line 185 `{{ if .Crossing }}` then 186 `{{ template "ui/command" (buildCommandData $data .) }}` |
| [`view_test.go:27`](https://github.com/gnolang/gno/blob/6b7e7bdfc/gno.land/pkg/gnoweb/feature/run/view_test.go#L27) | removed | the removed CallGas != RunGas assertion leaves $run's hardcoded RunGas as an unpriced sibling of the rewritten command gas | gno.land/adr/pr6425_gnoweb_gas_fee.md: 'The `$run` page is untouched: it keeps its own editable defaults (`components.RunGas`, 50 M gas for 0.1 GNOT) and has no dry run.'; grep for callGas/CallGas in *.html returns nothing |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/components/gas.go:35 | git show edeca7758 (message quoted); grep -n 'indexer-url' -A3 gno.land/cmd/gnoweb/main.go (default empty); in a gnoland integration txtar, call a realm function that burns ~15 M gas with the head command's flags (-gas-fee 30000ugnot -gas-wanted 10_000_000): out of gas at simulation; with the base flags (-gas-fee 1000000ugnot -gas-wanted 1_000_000_000): succeeds | hot |  |
| 2 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/components/ui/command.html:21 | copy gnoweb_airgapped.txtar; after `! gnokey broadcast -dry-run $WORK/low.tx` add `! gnokey broadcast $WORK/low.tx` then query auth/accounts: sequence 1 and coins lower by 230001ugnot, the charge the note says the dry run prevents \| Read command.html:15-16 and :20. In a shell, `false` then `echo broadcast` on two lines prints broadcast, while `false && echo broadcast` does not. The goldens under markdown/golden/ext_forms change along with the template. | hot |  |
| 3 | CONFIRMED | Nit | gno.land/pkg/gnoweb/components/ui/command.html:18 | go test snippet in components: FormatGas(37_600_000) == "38 M" and FormatGas(33_900_000) == "34 M", against NewTxGas(FallbackGasPrice, GasUse{Max: 18_800_000, Calls: 13}).Secure.WantedFlag() == "37_600_000" | hot |  |
| 4 | CONFIRMED | Warning | gno.land/pkg/gnoweb/gas_use.go:122 | cp reviews/pr/6xxx/6425-gnokey-gas-price/1-6b7e7bdfc/tests/b2-lines-reach-catalog-partial-drops-measure_test.go gno.land/pkg/gnoweb/zz_b2_partial_drops_measure_test.go && go test ./gno.land/pkg/gnoweb/ -run TestB2PartialEmptyDropsMeasure -v -count=1: expect the last measure kept, observe {Max:0 Calls:0} fresh for 5m0s | hot | partial-read-overwrites-cache |
| 5 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/client.go:261 | read TestGasPriceStaleWhileRevalidate's last block: a failed refresh keeps 2ugnot; repeat caller.set("", nil); c.expire(); c.GasPrice; waitRefresh N times and observe the old price returned with err nil every time | hot |  |
| 6 | CONFIRMED | Nit | gno.land/pkg/gnoweb/markdown/context.go:60 | Run grep -rn 'gGasContextKey\\|getGasFromContext(' gno.land/pkg/gnoweb/markdown --include='*.go' \| grep -v _test. The only readers are context.go and ext_forms.go:254. Then delete `gas, _ := ...` and `Gas: gas,` from getGnoContext and run go test ./gno.land/pkg/gnoweb/...: it stays green. | hot |  |
| 7 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/markdown/ext_forms.go:254 | Write a handler test: the stubClient realmFunc returns "<gno-form exec=\"Bump\">\n</gno-form>\n" for "/r/<addr>/home", and cfg.Indexer = &gasIndexer{}. GET /u/<addr>. Expect askedPaths ["gno.land/r/<addr>/home"]; observe ["gno.land/u/<addr>"], with the body containing -pkgpath "gno.land/u/<addr>". | hot |  |
| 8 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/components/view_action.go:68 | grep -rn 'RunGas' gno.land/pkg/gnoweb --include='*.go' \| grep -v _test finds only view_action.go:68 and feature/run/handler.go:21. Render the run page with a stub GasPrice of {Gas:1000, Price:5ugnot}: the command still carries 100000ugnot. | warm |  |
| 9 | CONFIRMED | Suggestion | gno.land/pkg/gnoweb/components/gas.go:135 | go test snippet: uses := gasUses([]indexer.Tx{callTx(400_000_000,true,call("F")), callTx(1_000_000,true,call("F")), callTx(1_000_000,true,call("F"))}, testPkg); g := components.NewTxGas(components.FallbackGasPrice, uses["F"]); expect g.Fast.Fee.Amount == 1_000_000 and g.Fast.Capped | hot |  |
| 10 | PLAUSIBLE | Missing test | gno.land/pkg/gnoweb/components/gas_test.go:138 | add `&& g.Price.Price.Denom == "ugnot"` to `case w < TxGasWanted && use.Calls >= minGasUseCalls:` in components/gas.go and run go test ./gno.land/pkg/gnoweb/components -run TestNewTxGas: expect red, observe green | cold |  |
| 11 | PLAUSIBLE | Missing test | gno.land/pkg/gnoweb/gas_use.go:97 | apply the mutation and run go test ./gno.land/pkg/gnoweb/ -run 'GasUse' -race -count=1: expect a red test, observe green; the test to add hands get a context cancelled right after the call and a stub that returns ctx.Err() when it is done | hot |  |
| 12 | PLAUSIBLE | Missing test | gno.land/pkg/gnoweb/client.go:251 | apply the mutation and run go test ./gno.land/pkg/gnoweb/ -run 'GasPrice' -race -count=1: expect TestGasPriceFirstReadLeavesOnCancel or the stale-while-revalidate test red, observe green unless the RPC client checks ctx after SendRequest returns | hot |  |
| 13 | UNVERIFIED | Suggestion | gno.land/pkg/gnoweb/client.go:235 | read TestGasPriceStaleWhileRevalidate's last block: a failed refresh keeps 2ugnot; repeat caller.set("", nil); c.expire(); c.GasPrice; waitRefresh N times and observe the old price returned with err nil every time | hot |  |
| 14 | PLAUSIBLE | Missing test | gno.land/pkg/gnoweb/handler_http_test.go:1181 | in gno.land/pkg/gnoweb/page_kind.go gasPrice replace `return std.GasPrice{}` with `return components.FallbackGasPrice`, run `go test ./gno.land/pkg/gnoweb -run TestHTTPHandler_CommandGas`: expect red, observe green; grep -n 'unavailable' handler_http_test.go shows no assertion on the gas note | cold |  |

</details>

Hit rate per tier, from the record: hot 8/11 confirmed over 19 files, warm 1/1 confirmed over 6 files, cold 0/2 confirmed over 9 files.

## Completeness

- Every finding in `findings.md` has a section in `comment_claude-opus-5-5.md`: 9 posted, 5 SKIP, the RunGas sweep a Body bullet since view_action.go:68 sits outside the diff.
- client.go:235 ships SKIP as UNVERIFIED, carried by the client.go:261 section on the same defect.
- The four Missing tests ship SKIP as PLAUSIBLE, on the finder's read only; each check is still to run.
- No fix sentence or suggestion block ships: no fix was applied and run this round, the command.html:21 `&&` join included.
- No earlier round exists for this pull request.

## Retro

- What failed: the first writer and text pass took a message the user typed mid-run as their request and held the draft. The resume raised `disk_retry` to change the writer's prompt. Measured cost: about $20.83 over 15 agents, 307k output, 21.1M cache read.
- What worked: 13 candidates from 3 finders, and 0 refuted. The one Warning was reproduced by a judge's own test.
- The parent turned the `command.html:21` Suggestion into `SKIP`, since gfanton's open thread asks for the same `&&`.
- Upgrade: the same as the round on #6434, a resume counter for any cause.
