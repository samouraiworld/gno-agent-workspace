# Findings in posting order, from round assemble: 9 to post, 5 SKIP, 0 refuted kept out

## Body

- Suggestion: gno.land/pkg/gnoweb/components/view_action.go:68 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/view_action.go#L68), outside the diff, so no anchor
  State: CONFIRMED, band: Suggestion, angle: sweep
  TL;DR: The $run command still prints a fixed 100000ugnot fee, so it does not follow the chain's gas price like the call commands now do
  Check: grep -rn 'RunGas' gno.land/pkg/gnoweb --include='*.go' | grep -v _test finds only view_action.go:68 and feature/run/handler.go:21. Render the run page with a stub GasPrice of {Gas:1000, Price:5ugnot}: the command still carries 100000ugnot.
  Details: The diff replaces CallGas with the chain-priced TxGas but leaves RunGas = 50_000_000 / 100000ugnot, used by feature/run/handler.go:21. That fee is 2ugnot per 1000 gas; if the chain's price rises above it, the copied $run script is rejected at CheckTx while the call commands beside it adapt. The constant predates the diff, but the diff swept its sibling. Fix: price RunGas through NewTxGas-style pricing from the same GasPrice read, or note why the run page keeps a fixed fee.
  Evidence: Read head view_action.go:68 `RunGas = Gas{Wanted: "50_000_000", Fee: "100000ugnot"}`; grep finds only feature/run/handler.go:21 `Gas: components.RunGas` as a non-test user; base view_action.go:62 had the swept `CallGas` beside it.

## gno.land/pkg/gnoweb/gas_use.go:122 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L122) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: a slow indexer's empty ErrPartial read replaces the last gas measure with nothing for 5 minutes
Check: cp reviews/pr/6xxx/6425-gnokey-gas-price/1-6b7e7bdfc/tests/b2-lines-reach-catalog-partial-drops-measure_test.go gno.land/pkg/gnoweb/zz_b2_partial_drops_measure_test.go && go test ./gno.land/pkg/gnoweb/ -run TestB2PartialEmptyDropsMeasure -v -count=1: expect the last measure kept, observe {Max:0 Calls:0} fresh for 5m0s
Details: read() treats ErrPartial as success (gas_use.go:122) even with zero rows, and indexer.recent returns exactly that when the 10 s gasUseTimeout leaves under minWindowBudget (900 ms) after three 3.2 s bands (queries.go:364 break, then partial() at 356-357). A package whose recent calls sit only in the fourth band (146k to 1.17M blocks back) is measured by a fast read and then wiped by a slow one, so for gasUseTTL the page shows no gas use and NewTxGas falls back to TxGasWanted: the fast-mode fee is lost and a function measured above the default gets a command that fails its simulation. The comment at 125-127 promises the opposite, and a hard error on the same read keeps the measure; the trigger is narrow (slow-but-answering indexer, calls only in the deepest band) and self-heals after 5 min, and gas_use.go is new in this PR. Fix: on ErrPartial with no rows for the package, keep last.uses as the error branch does (or store with gasUseRetry), and store the new map only when it has rows or err == nil.
Evidence: Head, finder artifact rerun: 'RecentByPackage: 0 rows, err=searched the most recent blocks only, took 9.6s' / 'after a slow read: Deposit = {Max:0 Calls:0}, entry fresh=true, expires in 5m0s' / FAIL. Head, judge artifact with a truthful indexer (rows only in the band gt: 8830000, every band 3.2 s on the second read): 'fast read: Deposit = {Max:1400000 Calls:1}' then 'slow read: Deposit = {Max:0 Calls:0}, fresh=true' / FAIL (9.60s). Base: gas_use.go does not exist at 5e9668dab, no gas-use cache to compare.
Artifact: projects/gno-agent-workspace/checkout/reviews/pr/6xxx/6425-gnokey-gas-price/1-6b7e7bdfc/tests/judge-4-partial-band-drops-measure_test.go

## SKIP gno.land/pkg/gnoweb/client.go:251 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L251) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: No test pins queryGasPrice's context.WithoutCancel: gasCaller.SendRequest ignores its ctx, so the stale-while-revalidate refresh and the shared first read stay green when the request context is passed through
Check: apply the mutation and run go test ./gno.land/pkg/gnoweb/ -run 'GasPrice' -race -count=1: expect TestGasPriceFirstReadLeavesOnCancel or the stale-while-revalidate test red, observe green unless the RPC client checks ctx after SendRequest returns
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/gnoweb/components/gas_test.go:138 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas_test.go#L138) · Missing test
State: PLAUSIBLE, band: Missing test, angle: claims, on the finder's read only
TL;DR: The foreign-denom block says 'a measure lowers the fast-mode gas but never raises either mode' and pins only the no-raise half
Check: add `&& g.Price.Price.Denom == "ugnot"` to `case w < TxGasWanted && use.Calls >= minGasUseCalls:` in components/gas.go and run go test ./gno.land/pkg/gnoweb/components -run TestNewTxGas: expect red, observe green
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/gnoweb/gas_use.go:97 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/gas_use.go#L97) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: No test pins the detached context of the background gas-use read: stubRecent.recent ignores its ctx and every test passes context.Background(), so dropping context.WithoutCancel stays green
Check: apply the mutation and run go test ./gno.land/pkg/gnoweb/ -run 'GasUse' -race -count=1: expect a red test, observe green; the test to add hands get a context cancelled right after the call and a stub that returns ctx.Err() when it is done
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/gnoweb/handler_http_test.go:1181 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/handler_http_test.go#L1181) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: the node-down row pins only the fee, so nothing at the handler pins that a failed GasPrice reads as 'the chain's price was unavailable' rather than as the chain's own price
Check: in gno.land/pkg/gnoweb/page_kind.go gasPrice replace `return std.GasPrice{}` with `return components.FallbackGasPrice`, run `go test ./gno.land/pkg/gnoweb -run TestHTTPHandler_CommandGas`: expect red, observe green; grep -n 'unavailable' handler_http_test.go shows no assertion on the gas note
Evidence: not run: no verifier reached it; the check is still to run

## gno.land/pkg/gnoweb/components/ui/command.html:18 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/ui/command.html#L18) · Nit
State: CONFIRMED, band: Nit, angle: claims + page
TL;DR: The note rounds the gas cap up: 'for up to 38 M gas' beside -gas-wanted 37_600_000
Check: go test snippet in components: FormatGas(37_600_000) == "38 M" and FormatGas(33_900_000) == "34 M", against NewTxGas(FallbackGasPrice, GasUse{Max: 18_800_000, Calls: 13}).Secure.WantedFlag() == "37_600_000"
Details: FormatGas(37_600_000) skips the k branch (round 37600 >= 1000) and the one-decimal M branch (round(f/1e5) = 376 >= 100), and formats f/1e6 = 37.6 with 0 decimals, giving '38 M'. NewTxGas with Max 18.8 M yields Wanted 37,600,000, so the note on lines 18 and 20 overstates the cap by up to 0.5 M for any raise between 10 M and 100 M that is not a whole million. Fix: print the cap with one decimal below 100 M, or round down (math.Floor) when formatting a limit.
Evidence: Read gas.go:22-25: `case math.Round(f/1e5) < 100: ... 'f', 1 ...` / `case math.Round(f/1e6) < 1_000: return strconv.FormatFloat(f/1e6, 'f', 0, 64) + " M"`; gas.go:129 `w = ... (2*use.Max+gasWantedStep-1)/gasWantedStep*gasWantedStep` gives 37,600,000 for Max 18,800,000.

## gno.land/pkg/gnoweb/markdown/context.go:60 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/markdown/context.go#L60) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: getGnoContext copies the gas lookup into <gno-foreign> content, whose parser never loads forms, so the copy is never read
Check: Run grep -rn 'gGasContextKey\|getGasFromContext(' gno.land/pkg/gnoweb/markdown --include='*.go' | grep -v _test. The only readers are context.go and ext_forms.go:254. Then delete `gas, _ := ...` and `Gas: gas,` from getGnoContext and run go test ./gno.land/pkg/gnoweb/...: it stays green.
Details: getGnoContext reads gGasContextKey and sets Gas on the GnoContext stored per foreign node (ext_foreign.go:228) and rebuilt at :382, but buildInnerForeignMarkdown loads no ExtForms ('gno-form is intentionally NOT loaded', :431). The only reader of the key is getGasFromContext, called only from ext_forms.go:254. Nothing breaks; the line is dead plumbing that holds the request closure on each foreign node. Fix: drop `gas, _ := ...` and `Gas: gas,` from getGnoContext, or comment why it is carried.
Evidence: grep gGasContextKey/getGasFromContext( in markdown non-test: context.go:14,43,60,95-96 and ext_forms.go:254 only; ext_foreign.go:438-460 extension list has no ExtForms.

## SKIP gno.land/pkg/gnoweb/client.go:235 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L235) · Suggestion
State: UNVERIFIED, band: Suggestion, angle: lines, on the finder's read only
TL;DR: GasPrice serves the last good price with no bound on its age: a refresh that keeps failing resets gasAt and keeps gasOK, so an hours-old price is shown as current with Estimated false
Check: read TestGasPriceStaleWhileRevalidate's last block: a failed refresh keeps 2ugnot; repeat caller.set("", nil); c.expire(); c.GasPrice; waitRefresh N times and observe the old price returned with err nil every time
Evidence: not run: on the finder's read

## gno.land/pkg/gnoweb/client.go:261 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/client.go#L261) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: Once one gas price is read, failed refreshes keep serving it forever as the chain's current price
Check: read TestGasPriceStaleWhileRevalidate's last block: a failed refresh keeps 2ugnot; repeat caller.set("", nil); c.expire(); c.GasPrice; waitRefresh N times and observe the old price returned with err nil every time
Details: queryGasPrice sets gasAt = time.Now() before checking err and leaves gasPrice and gasOK untouched on failure, so GasPrice keeps returning the old price with a nil error and the page labels it 'the chain's price' (Estimated stays false). The price has no maximum age. With x3 headroom the worst case is a copied command CheckTx rejects for free after a price rise. Fix: record when the last good read happened and, past a bound (say a few TTLs), return an error so the page falls back to the Estimated label.
Evidence: Read client.go:259-266: `c.gasAt = time.Now()` / `if err != nil { ... return std.GasPrice{}, ... }` / `c.gasPrice, c.gasOK = gp, true`; GasPrice:241-244 returns gp, nil whenever ok.

## gno.land/pkg/gnoweb/components/gas.go:35 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas.go#L35) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: Without indexer data, a copied call needing over 10 M gas fails where base's 1,000 M command went through, and the note gives no figure to raise to
Check: git show edeca7758 (message quoted); grep -n 'indexer-url' -A3 gno.land/cmd/gnoweb/main.go (default empty); in a gnoland integration txtar, call a realm function that burns ~15 M gas with the head command's flags (-gas-fee 30000ugnot -gas-wanted 10_000_000): out of gas at simulation; with the base flags (-gas-fee 1000000ugnot -gas-wanted 1_000_000_000): succeeds
Details: Base printed CallGas = 1_000_000_000 / 1000000ugnot for every function; head prints TxGasWanted = 10_000_000 at 30000ugnot unless the indexer measured the function, and -indexer-url defaults to empty. A call above 10 M therefore fails simulation (fast) or the dry run (secure), which is what #5146 fixed; nothing is charged, but the user must guess a new -gas-wanted, and past 30 M gas the 30000ugnot fee also falls under 1ugnot/1000gas so CheckTx rejects a second time. The per-call gas figures (GovDAO vote, valopers) are the finder's, not measured here. Fix: when Used.Calls is 0, say in the note how far -gas-wanted may go and that -gas-fee scales with it (1ugnot per 1000 gas), or keep a higher default for deployments without an indexer.
Evidence: Read: base components/view_action.go:62 `CallGas = Gas{Wanted: "1_000_000_000", Fee: "1000000ugnot"}`; head gas.go:35 `const TxGasWanted = 10_000_000`, gas.go:119-121 return the 10 M fee when use.Calls == 0; cmd/gnoweb/main.go:152 'Empty (the default) keeps gnoweb talking only to its RPC node'; git log -1 edeca7758: 'many gnoweb transactions can't simply be copy-pasted as gnokey commands'.

## gno.land/pkg/gnoweb/components/gas.go:135 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/gas.go#L135) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: One costly successful call by anyone raises every viewer's copied command for that function to the 1 GNOT cap until 20 newer calls push it out
Check: go test snippet: uses := gasUses([]indexer.Tx{callTx(400_000_000,true,call("F")), callTx(1_000_000,true,call("F")), callTx(1_000_000,true,call("F"))}, testPkg); g := components.NewTxGas(components.FallbackGasPrice, uses["F"]); expect g.Fast.Fee.Amount == 1_000_000 and g.Fast.Capped
Details: gasUses takes the max over the last calls with no outlier filter (gas_use.go:61), and the raise case at gas.go:135 has no minGasUseCalls check. With Max = 400 M at the fallback price, most = 1e9, w = 800 M, and txFee caps the fee at 1,000,000ugnot, set on both Secure and Fast. Base charged 1 GNOT on every call, so nobody pays more than before, but the PR's saving for that function is undone by one call costing its sender about 0.4 GNOT. Fix: require minGasUseCalls for a raise too, or size from a percentile rather than the max.
Evidence: Read gas.go:128-137: `if use.Max < most/2 { w = ... 2*use.Max ... }` then `case w > TxGasWanted: g.Secure, g.Fast, g.Used = fee, fee, use`; maxWanted(FallbackGasPrice) = 1e6*1000/1 capped to 1_000_000_000; txFee(fallback, 800_000_000) = 2,400,000 > maxUgnotFee, ceilTimes(1) = 800,000 <= 1,000,000, so Fee 1000000ugnot Capped; gas_use.go:61 `u.Max = max(u.Max, int64(tx.GasUsed))`.

## gno.land/pkg/gnoweb/components/ui/command.html:21 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/components/ui/command.html#L21) · Suggestion
State: CONFIRMED, band: Suggestion, angle: page
TL;DR: Pasting the secure script whole runs the broadcast even when the dry run fails, so the note's 'the dry run stops that first' does not hold and the fee is charged
Check: copy gnoweb_airgapped.txtar; after `! gnokey broadcast -dry-run $WORK/low.tx` add `! gnokey broadcast $WORK/low.tx` then query auth/accounts: sequence 1 and coins lower by 230001ugnot, the charge the note says the dry run prevents | Read command.html:15-16 and :20. In a shell, `false` then `echo broadcast` on two lines prints broadcast, while `false && echo broadcast` does not. The goldens under markdown/golden/ext_forms change along with the template.
Details: The dry run (line 15) and the broadcast (line 16) are separate lines with no `&&`, so a shell running the pasted block runs line 16 whatever line 15 exits with. The note at line 21 says the dry run 'stops that first', so a user who pastes the block trusts a guard that does not run, and an out-of-gas call is charged its fee. Fix: join the two lines as `gnokey broadcast -dry-run ... call.tx && gnokey broadcast ... call.tx`, or reword the note to say to run the broadcast only after the dry run succeeds; the ext_forms goldens follow the template.
Evidence: Read command.html:15 `gnokey broadcast -dry-run -remote "{{.Remote}}" call.tx`, :16 `gnokey broadcast -remote "{{.Remote}}" call.tx`, :21 'The broadcast does not simulate, so a call that runs out of gas is still charged: the dry run before it stops that first.' No `&&` or `set -e` in the block.

## gno.land/pkg/gnoweb/markdown/ext_forms.go:254 [gh](https://github.com/gnolang/gno/blob/6b7e7bdfc61f4db559693bfdc521da564f364f41/gno.land/pkg/gnoweb/markdown/ext_forms.go#L254) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: On a /u/ page, a form's gas is looked up under gno.land/u/<name>, which never has calls, so the measured gas never applies and an indexer read is spent for nothing
Check: Write a handler test: the stubClient realmFunc returns "<gno-form exec=\"Bump\">\n</gno-form>\n" for "/r/<addr>/home", and cfg.Indexer = &gasIndexer{}. GET /u/<addr>. Expect askedPaths ["gno.land/r/<addr>/home"]; observe ["gno.land/u/<addr>"], with the body containing -pkgpath "gno.land/u/<addr>".
Details: GetUserView renders /r/<ns>/home with the /u/ GnoURL (handler_http.go:1138), FormParser.Open takes RealmName from that URL (ext_forms.go:232), and line 254 asks the gas closure for Domain+RealmName, which page_kind.go:259 passes to gasUse.get. No MsgCall targets gno.land/u/..., so the result is always empty and the fast command stays at 10 M, while the indexer is queried per such page. The printed -pkgpath carries the same wrong path, which predates this diff. Fix: pass the realm actually rendered (the /r/<ns>/home path) as the form's realm on user pages; file the -pkgpath part as its own issue.
Evidence: Read handler_http.go:1132-1138 `raw, err := h.Client.Realm(ctx, "/r/"+ns+"/home", "")` then `h.Renderer.RenderRealm(&content, gnourl, raw, ...)`; ext_forms.go:232 `node.RealmName = gnourl.Path`, :254 `getGasFromContext(pc, node.Domain+node.RealmName, exec)`; page_kind.go:259 `components.NewTxGas(price(), h.gasUse.get(ctx, pkgPath)[fn])`; gas_use.go:54 filters `m.PkgPath != pkgPath`.
