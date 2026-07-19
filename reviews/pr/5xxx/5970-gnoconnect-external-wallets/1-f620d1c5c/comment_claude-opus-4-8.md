# Review: PR [#5970](https://github.com/gnolang/gno/pull/5970)
Event: REQUEST_CHANGES

## Body
Verified on f620d1c5c against a local gnodev: of the four Execute paths, only a coarse pointer with a registered wallet cancels the submit, and Continue in browser lands on the same URL the plain Execute button reaches.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5970-gnoconnect-external-wallets/1-f620d1c5c/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/resources/gnoconnect.md:96-99 [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L96-L99)
The rule covers a missing scheme, not `tcp://`, which is what gnodev serves as `gnoconnect:rpc` and [`_buildLink`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L135-L138) forwards verbatim. A wallet that implements only what is written here leaves `tcp://` alone and cannot reach a local node, the setup this transport was validated on.

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

```
<meta name="gnoconnect:rpc" content="tcp://127.0.0.1:26699" />
```

gnodev assigns the raw remote address in [`setup_web.go`](https://github.com/gnolang/gno/blob/f620d1c5c/contribs/gnodev/setup_web.go#L22-L27) instead of passing it through [`normalizeRemoteURL`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/cmd/gnoweb/main.go#L387-L408).
</details>

## docs/resources/gnoconnect.md:74 [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L74)
The [Gnoweb, Adena and Gnobro list](https://github.com/gnolang/gno/blob/f620d1c5c/docs/resources/gnoconnect.md?plain=1#L104-L107) lost its own `## Supported Clients` heading, so it now sits inside the launch-link spec and reads as a list of launch-link implementers.

## gno.land/pkg/gnoweb/components/wallet_registry.go:74-77 [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallet_registry.go#L74-L77)
Nit: `Wallets()` hands out the package-level slice, so a caller can mutate the shared registry. Only [`wallet_registry_test.go`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/components/wallet_registry_test.go#L16) calls it today.

## gno.land/pkg/gnoweb/components/views/action.html:182-184 [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/views/action.html#L182-L184)
Nit: the dialog has no `aria-labelledby` pointing at its own "Open with a wallet" title, so a screen reader announces it unnamed.

## gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts:168-171 [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168-L171)
Suggestion: with the dialog absent this opens the first wallet directly, so a user without that app installed gets the silent dead end the chooser exists to prevent. Every page renders the dialog today, so it is latent.
