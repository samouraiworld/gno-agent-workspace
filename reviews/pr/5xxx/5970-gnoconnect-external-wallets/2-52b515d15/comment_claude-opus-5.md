# Review: [#5970](https://github.com/gnolang/gno/pull/5970)
Event: REQUEST_CHANGES
Status: not posted. Round 2 at 52b515d15. The Supported Clients finding is fixed at this head and was dropped; the spec rewrite in this round opened two divergences against the controller the same branch ships. On `post as an AI` the Body leads with `[AI review, opus 5] (not manually verified)`, then `Status: REQUEST_CHANGES`.

## Body
The spec this branch rewrites and the controller it ships describe different links, so a wallet built from the document does not open what gnoweb emits.

- The transaction verb is [`sendtx`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L329) in the spec, and [`tx`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L141) in the link the controller builds and in the [ADR](https://github.com/gnolang/gno/blob/52b515d15/gno.land/adr/pr5970_gnoconnect_external_wallets.md#L23) beside it.
- [`chainid` is required and a link without one is `invalid_request`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L341-L342), while the controller [appends it only when the page carries the meta tag](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L138), so a deployment that omits it emits a link every conforming wallet rejects.

## docs/resources/gnoconnect.md:345-346 [gh](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L345-L346) · [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L345)
The rule covers a missing scheme, not `tcp://`, which is what gnodev publishes as `gnoconnect:rpc` and what [`_buildLink`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L137) forwards verbatim, so a wallet that implements the rule as written prefills an add-network proposal with an endpoint it cannot reach.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5970 -R gnolang/gno
(cd contribs/gnodev && go build -o /tmp/gnodev-5970 .)
/tmp/gnodev-5970 local -web-listener 127.0.0.1:8899 -node-rpc-listener 127.0.0.1:26699 \
  -no-watch ./examples/gno.land/r/demo/counter > /tmp/gnodev-5970.log 2>&1 &
sleep 30
curl -s 'http://127.0.0.1:8899/r/demo/counter$help' | grep -o '<meta name="gnoconnect:rpc"[^>]*>'
kill %1; rm -f /tmp/gnodev-5970 /tmp/gnodev-5970.log
```

The published endpoint carries a scheme the spec does not name:

```
<meta name="gnoconnect:rpc" content="tcp://127.0.0.1:26699" />
```

gnodev assigns the raw remote address in [`setup_web.go`](https://github.com/gnolang/gno/blob/52b515d15/contribs/gnodev/setup_web.go#L22) rather than passing it through [`normalizeRemoteURL`, which maps `tcp://` to `http://`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/cmd/gnoweb/main.go#L402-L403).
</details>

## gno.land/pkg/gnoweb/components/wallet_registry.go:74-77 [gh](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/wallet_registry.go#L74-L77) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallet_registry.go#L74)
Nit: `Wallets()` hands out the package-level slice, so a caller can mutate the shared registry. Only [`wallet_registry_test.go`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/wallet_registry_test.go#L16) calls it today.

## gno.land/pkg/gnoweb/components/views/action.html:182-184 [gh](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/views/action.html#L182-L184) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/views/action.html#L182)
Nit: the dialog has no `aria-labelledby` pointing at its own "Open with a wallet" title, so a screen reader announces it unnamed.

## gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts:168-171 [gh](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168-L171) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168)
Suggestion: with the dialog absent this opens the first wallet directly, so a user without that app installed gets the silent dead end the chooser exists to prevent; falling through to the native submit keeps the fail-open property the rest of the controller has.
