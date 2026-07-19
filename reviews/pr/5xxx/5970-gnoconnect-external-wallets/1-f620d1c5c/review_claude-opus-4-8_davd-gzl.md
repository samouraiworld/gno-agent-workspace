# PR [#5970](https://github.com/gnolang/gno/pull/5970): feat(gnoweb): GnoConnect external-wallet transport — registry + chooser + launch-link

URL: https://github.com/gnolang/gno/pull/5970
Author: D4ryl00 | Base: master | Files: 12 | +733 -5
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: f620d1c5c (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5970 f620d1c5c`

**TL;DR:** On a phone, gnoweb's Execute button currently just prints a `gnokey` command you cannot run. This PR makes it open a wallet app instead, through a link the phone hands to the app, and adds a small chooser so you can also stay in the browser.

**Verdict: REQUEST CHANGES** — the code path works end to end, but the wire format published for wallet authors omits the `rpc` value gnodev actually emits, and the edit drops the standard's Supported Clients heading (2 Warnings, 2 Nits, 2 Suggestions).

## Summary

gnoweb already advertises the chain through `gnoconnect:*` meta tags, and browser extensions read them from inside the page. A mobile app cannot: it has no way to announce itself on `window`, and a same-domain form submit cannot reach it. This PR adds the missing transport. A registry of wallets ships in-repo as [`components/wallets.json`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/components/wallets.json) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallets.json), is validated and marshaled once at package init, and is embedded into the `$help` page as a JSON script tag. A new controller intercepts the Execute submit on coarse-pointer devices, shows a chooser dialog, and on selection assigns `window.location.href = "<scheme>://tx?path=…&func=…&arg.<name>=…&rpc=…&chainid=…&callback=…"`. Every other case, desktop, an in-page extension, or an empty registry, falls through to the existing submit untouched.

The one entry today is gnokey-mobile, whose icon is 3346 bytes of base64 inlined into every `$help` page.

## Examples

| Situation | What Execute does |
|---|---|
| Desktop, wallet registered | native submit, unchanged |
| Mobile, browser extension present | native submit, unchanged |
| Mobile, empty or malformed registry | native submit, unchanged |
| Mobile, wallet registered | chooser opens; picking the wallet opens `land.gno.gnokey://tx?…`, picking Continue in browser does the native submit |

Composed link observed live for `Render` with the input `  hello world&x=1 `:

```
http://tx/?path=gno.land%2Fr%2Fdemo%2Fcounter&func=Render&arg._=hello%20world%26x%3D1
  &rpc=tcp%3A%2F%2F127.0.0.1%3A26701&chainid=dev&callback=http%3A%2F%2Flocalhost%3A8901%2Fr%2Fdemo%2Fcounter%24help
```

(`http` substituted for the wallet scheme so the navigation was observable.)

## Glossary

- GnoConnect: the wallet-integration standard gnoweb implements; a provider page advertises the chain through `gnoconnect:*` meta tags and expresses a transaction intent as a TxLink.
- TxLink: the transaction-intent link a provider emits, naming the realm, the function, and named `&param=value` arguments; a wallet fills the rest from `vm/qdoc`.
- gnoweb: the web frontend serving chain content, server-rendered Go templates plus per-view controllers compiled from TypeScript.

## Fix

Before, the only Execute outcome on a page with no in-page provider was a copy-paste `gnokey` command. After, [`WalletLaunchController._onSubmit`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L148-L159) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L148-L159) calls `preventDefault()` only after three guards pass, and the chooser always carries a Continue in browser action that reaches the native submit. The load-bearing constraint is that a failed custom-scheme launch is silent, so every interception must keep a visible path back to the browser.

## Critical (must fix)

None.

## Warnings (should fix)

- **[local wallets get an rpc value the spec does not cover]** `docs/resources/gnoconnect.md:96-99` — the spec tells wallets to assume `http://` only when the scheme is missing, but gnodev emits `tcp://`, so a wallet that follows it cannot reach a local node.
  <details><summary>details</summary>

  [The `rpc` bullet](https://github.com/gnolang/gno/blob/f620d1c5c/docs/resources/gnoconnect.md?plain=1#L96-L99) · [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L96-L99) covers two shapes, a full URL and a bare `127.0.0.1:26657`. gnodev produces a third: [`setup_web.go` assigns the raw remote address](https://github.com/gnolang/gno/blob/f620d1c5c/contribs/gnodev/setup_web.go#L22-L27) · [↗](../../../../../.worktrees/gno-review-5970/contribs/gnodev/setup_web.go#L22-L27) rather than passing it through [`normalizeRemoteURL`, which maps `tcp://` to `http://`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/cmd/gnoweb/main.go#L387-L408) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/cmd/gnoweb/main.go#L387-L408). Booting gnodev from this worktree and driving Execute produced `rpc=tcp%3A%2F%2F127.0.0.1%3A26701` in the launch link; see [repro](comment_claude-opus-4-8.md). A wallet implementing the documented rule leaves that value alone and fails on the localnet flow this transport is built for. Fix: document `tcp://` as an accepted `rpc` scheme that the wallet reads as `http://`.
  </details>

- **[the standard loses a published section]** `docs/resources/gnoconnect.md:74` — the new Launch Links heading replaced `## Supported Clients`, so the client list now reads as part of the launch-link spec.
  <details><summary>details</summary>

  On master the file carries `## Supported Clients` followed by its four bullets. The diff swaps that heading for [`## Launch Links (external wallets)`](https://github.com/gnolang/gno/blob/f620d1c5c/docs/resources/gnoconnect.md?plain=1#L74) · [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L74) and appends the new spec above the old bullets, leaving [Gnoweb, Adena, Gnobro and "Add your clients here"](https://github.com/gnolang/gno/blob/f620d1c5c/docs/resources/gnoconnect.md?plain=1#L104-L107) · [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L104-L107) dangling under Launch Links, where they read as launch-link implementers. Fix: restore the `## Supported Clients` heading above those bullets.
  </details>

## Nits

- **[exported surface nothing calls]** `gno.land/pkg/gnoweb/components/wallet_registry.go:74-77` — [`Wallets()`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/components/wallet_registry.go#L74-L77) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallet_registry.go#L74-L77) has no caller outside [`wallet_registry_test.go`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/components/wallet_registry_test.go#L16) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallet_registry_test.go#L16), and it returns the package-level slice, so a future caller can mutate the shared registry.

- **[screen readers announce an unnamed dialog]** `gno.land/pkg/gnoweb/components/views/action.html:182-184` — the [chooser `<dialog>`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/components/views/action.html#L182-L184) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/views/action.html#L182-L184) has no `aria-labelledby` pointing at its own "Open with a wallet" title; reading `aria-labelledby` off the open dialog in the browser returned null.

## Missing Tests

None. The Go side covers registry validation, the JSON round trip and the rendered script tag; the frontend has no JavaScript test runner in [`frontend/package.json`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/package.json#L1-L22) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/package.json#L1-L22), so a controller test would have no home.

## Suggestions

- **[a missing dialog turns fail-safe into fail-shut]** `gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts:168-171` — when the chooser markup is absent the controller launches the first wallet directly, which is the silent dead end the design argues against.
  <details><summary>details</summary>

  [The fail-open branch of `_openChooser`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168-L171) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168-L171) runs after `preventDefault()`, so with no dialog it opens `wallets[0]` with no chooser and no Continue in browser. If that wallet is not installed the launch fails silently and Execute is dead. Today the dialog always renders, so this is latent. Falling through to the native submit instead would keep the same fail-open property the rest of the controller has.
  </details>

- **[extension detection is a vendor list]** `gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts:105-109` — [`_hasInPageProvider`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L105-L109) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L105-L109) tests `window.adena` and `window.gnoconnect` by name, so an extension using the `registerWallet`/`getWallets` draft the chooser is meant to absorb is not recognised and gets shadowed by the chooser on a coarse-pointer device. Continue in browser recovers, so the cost is one extra tap.

## Verified

- Booted gnodev from this worktree against `r/demo/counter` and drove `$help` Execute in a browser. On a coarse pointer the chooser opens and the submit is cancelled; with `window.adena` defined the submit is not cancelled; with an empty registry the submit is not cancelled; on a fine pointer the submit is not cancelled and the dialog stays closed. Anchors: [`_onSubmit`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L148-L159) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L148-L159).
- Captured the composed launch link through the Navigation API: values are trimmed and percent-encoded, `&` and `=` inside a value survive as `%26`/`%3D`, and args appear in declaration order under `arg.`. Anchor: [`_buildLink`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L123-L142) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L123-L142).
- Continue in browser navigates to the form's own `action` URL, the same target the plain Execute button reaches, so the escape hatch is byte-identical to today's submit. Anchor: [the `chooser-browser` handler](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L199-L206) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L199-L206).
- The wallet round-trip return URL renders: `GET /r/demo/counter$help&func=Render&_=abc?status=success&hash=DEADBEEF` returned 200, and `_callbackURL` strips those two params when composing the next link. Anchor: [`_callbackURL`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L113-L118) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L113-L118).
- The new global [`dialog { margin: auto }`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/css/04-elements.css#L105-L110) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/css/04-elements.css#L105-L110) hits no existing component: the only other dialog in the codebase is [a `<div role="dialog">`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/components/layouts/header.html#L43) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/layouts/header.html#L43), which an element selector cannot match.
- The chooser classes created only in TypeScript survive the production CSS purge: [purgecss scans `./js/**/*.ts`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/frontend/postcss.config.cjs#L33-L38) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/postcss.config.cjs#L33-L38), and the built `public/main.css` served by gnodev contains `b-wallet-chooser__item` and `b-wallet-chooser__icon`.
- The registry survives `html/template` escaping and the gnokey icon paints: `JSON.parse` of the script tag returned one entry in the live page, and the rendered `<img>` reported a natural width of 96.
- `go test ./gno.land/pkg/gnoweb/...` green at f620d1c5c; CI green apart from the bot's codeowner and approval gates.

## Open questions

- Realm-rendered markdown exec forms ([`ext_forms.go:468`](https://github.com/gnolang/gno/blob/f620d1c5c/gno.land/pkg/gnoweb/markdown/ext_forms.go#L468) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/markdown/ext_forms.go#L468)) are a second Execute surface and still dead-end on mobile; not in the PR's out-of-scope list. Not posted: it is a follow-up scoping call, not a change to this diff.
- Each registry entry inlines its icon into every `$help` response (3346 bytes for gnokey today, uncached because it rides the HTML). Not posted: one entry is negligible, and the offline requirement drives the choice.
- The invariant catalog walk does not apply: the diff touches no gno code, only gnoweb Go, TypeScript, CSS and docs.
