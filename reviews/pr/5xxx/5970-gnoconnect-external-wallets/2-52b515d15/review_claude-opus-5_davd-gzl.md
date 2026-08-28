# PR [#5970](https://github.com/gnolang/gno/pull/5970): feat(gnoweb): gnoconnect (1/2) — external-wallet transport: registry, chooser, launch links

URL: https://github.com/gnolang/gno/pull/5970
Author: D4ryl00 | Base: master | Files: 12 | +1181 -11
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 52b515d15 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5970 52b515d15`
Overview: [visual overview](../overview.html)

**TL;DR:** Five commits since round 1 rewrote `docs/resources/gnoconnect.md` and nothing else. The rewrite restored the client-list heading the earlier round flagged, and it renamed the transaction verb from `tx` to `sendtx` without touching the controller or the ADR that still emit `tx`.

**Verdict: REQUEST CHANGES** — the branch now ships a spec and an implementation of that spec that disagree on the verb and on whether `chainid` is optional (2 Warnings, 2 Nits, 1 Suggestion).

## What moved since 1-f620d1c5c

`git diff f620d1c5c..52b515d15` touches one file, `docs/resources/gnoconnect.md`, +468 -26, across five commits: `647baa298`, `46d41129c`, `5f546d30f`, `da53e3c17`, `52b515d15`. Every Go, TypeScript, HTML and CSS file in the PR is byte-identical to round 1, so the four code findings carried over with their line numbers unchanged and each was re-read at this head.

| Round 1 finding | State at 52b515d15 |
|---|---|
| `gnoconnect.md:96-99` — `tcp://` outside the scheme rule | open, re-anchored to `345-346` |
| `gnoconnect.md:74` — `## Supported Clients` heading lost | fixed: the list is back under its own [`## Known Implementations`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L540) heading |
| `wallet_registry.go:74-77` — `Wallets()` returns the shared slice | open, same lines |
| `action.html:182-184` — dialog has no `aria-labelledby` | open, same lines |
| `controller-wallet-launch.ts:168-171` — fail-open opens wallet zero | open, same lines |
| `controller-wallet-launch.ts:105-109` — vendor-list extension probe | open, same lines, still not posted |

## Summary

gnoweb advertises the chain through `gnoconnect:*` meta tags, which a browser extension reads from inside the page and a mobile app cannot. This PR adds the missing transport: a registry of wallets ships in-repo as [`components/wallets.json`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/wallets.json) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallets.json), is validated and marshaled once at package init, and is embedded into the `$help` page as a JSON script tag. A controller intercepts the Execute submit on coarse-pointer devices, shows a chooser dialog, and on selection assigns `window.location.href` to a custom-scheme link. This round's commits turn the accompanying document from a sketch into a specification: network resolution, argument forms, callback rules, and three named verbs.

## Glossary

- GnoConnect: the wallet-integration standard gnoweb implements; a provider page advertises the chain through `gnoconnect:*` meta tags and expresses a transaction intent as a TxLink.
- TxLink: the transaction-intent link a provider emits, naming the realm, the function, and named `&param=value` arguments; a wallet fills the rest from `vm/qdoc`.
- Launch link: the custom-scheme URL the page hands to the operating system so an installed wallet app opens it, `land.gno.gnokey://sendtx?...` in the spec's spelling.

## Critical (must fix)

None.

## Warnings (should fix)

- **[the spec and the controller name different verbs]** unanchored, in the Body — the spec's transaction verb is `sendtx`, the controller and the ADR emit `tx`.
  <details><summary>details</summary>

  At round 1 the document and the code agreed: `git show f620d1c5c:docs/resources/gnoconnect.md` carries `<scheme>://tx?path=...` at line 82. Commit `da53e3c17` and its neighbours replace that with three verbs, [`sendtx`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L329), [`signtx`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L413) and [`connect`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L477), and nothing else in the branch moved. [`_buildLink`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L141) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L141) still returns `${wallet.scheme}://tx?`, the interface comment at [line 9](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L9) still documents `"://tx?..."`, and the ADR shipped in the same branch states the wire format as [`<scheme>://tx?...`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/adr/pr5970_gnoconnect_external_wallets.md#L23) and repeats it as the [wallet-author obligation](https://github.com/gnolang/gno/blob/52b515d15/gno.land/adr/pr5970_gnoconnect_external_wallets.md#L126). One decision settles all four sites, which is why this goes as one Body bullet rather than four anchors.
  </details>

- **[`chainid` is required by the spec and conditional in the code]** unanchored, in the Body — a page with no `gnoconnect:chainid` meta emits a link the spec defines as `invalid_request`.
  <details><summary>details</summary>

  The spec makes `chainid` the field that selects the network and says [a link without one is `invalid_request`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L341-L342), because "whichever chain the wallet happens to be on" is how a signature valid on one chain is produced for another. The controller reads it from the page and [appends it only if non-empty](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L138) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L138), and [`_meta`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L94-L97) returns the empty string when the tag is absent. A gnoweb instance started without a chain id therefore renders Execute buttons whose links every conforming wallet must reject, with no error visible on the page. Fix: refuse to build the link with no chain id, so the failure lands where it can be seen.
  </details>

- **[local wallets get an rpc value the spec does not cover]** `docs/resources/gnoconnect.md:345-346` — the rule normalizes a missing scheme and says nothing about `tcp://`, which is what gnodev publishes.
  <details><summary>details</summary>

  [The `rpc` bullet](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L345-L346) · [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L345) covers two shapes, a full URL and a bare `127.0.0.1:26657`. gnodev produces a third: [`setup_web.go` assigns the raw remote address](https://github.com/gnolang/gno/blob/52b515d15/contribs/gnodev/setup_web.go#L22) · [↗](../../../../../.worktrees/gno-review-5970/contribs/gnodev/setup_web.go#L22) rather than passing it through [`normalizeRemoteURL`, whose `tcp` case rewrites it to `http://`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/cmd/gnoweb/main.go#L402-L403) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/cmd/gnoweb/main.go#L402). Booting gnodev from this worktree at 52b515d15 and reading the rendered `$help` page returned `<meta name="gnoconnect:rpc" content="tcp://127.0.0.1:26699" />`; see [repro](comment_claude-opus-5.md). Under the round's new resolution rules `rpc` no longer selects the network, so the cost has narrowed: the value now only prefills an add-network proposal, and a wallet that leaves `tcp://` alone prefills an endpoint it cannot reach. Fix: name `tcp://` in the bullet as a scheme the wallet reads as `http://`.
  </details>

## Nits

- **[exported surface nothing calls]** `gno.land/pkg/gnoweb/components/wallet_registry.go:74-77` — [`Wallets()`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/wallet_registry.go#L74-L77) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallet_registry.go#L74) has no caller outside [`wallet_registry_test.go`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/wallet_registry_test.go#L16) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/wallet_registry_test.go#L16), and it returns the package-level slice, so a future caller can mutate the shared registry.

- **[screen readers announce an unnamed dialog]** `gno.land/pkg/gnoweb/components/views/action.html:182-184` — the [chooser `<dialog>`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/components/views/action.html#L182-L184) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/components/views/action.html#L182) has no `aria-labelledby` pointing at its own "Open with a wallet" title.

## Missing Tests

None. The Go side covers registry validation, the JSON round trip and the rendered script tag; the frontend has no JavaScript test runner in [`frontend/package.json`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/package.json#L1-L22) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/package.json#L1-L22), so a controller test would have no home. The two Body findings are the shape a controller test would have caught, which is the argument for adding that runner in part 2.

## Suggestions

- **[a missing dialog turns fail-safe into fail-shut]** `gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts:168-171` — [the fail-open branch of `_openChooser`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168-L171) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L168) runs after `preventDefault()`, so with no dialog it opens `wallets[0]` with no chooser and no Continue in browser. If that wallet is not installed the launch fails silently and Execute is dead. Every page renders the dialog today, so it is latent.

- **[extension detection is a vendor list]** `gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts:105-109` — [`_hasInPageProvider`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L105-L109) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L105) tests `window.adena` and `window.gnoconnect` by name, so an extension registering itself another way is shadowed by the chooser on a coarse-pointer device. Not posted: Continue in browser recovers, so the cost is one extra tap, and the round's Body already spends the author's attention on the spec split.

## Verified

- Re-read every carried anchor at 52b515d15 from the worktree. `Wallets()` is at 74-77, the chooser `<dialog>` at 182-184, and `_openChooser`'s fail-open branch at 168-171, unchanged from round 1 because the diff touches only the document.
- Booted gnodev from this worktree at 52b515d15 against `r/demo/counter` and read the rendered `$help` page: `gnoconnect:rpc` is `tcp://127.0.0.1:26699` and `gnoconnect:chainid` is `dev`.
- Walked the rewritten document against `_buildLink` field by field. `path`, `func`, `send`, `arg.<name>`, `rpc` and `callback` match the spec's spelling; `state` and `signer` are optional and absent, which conforms; the verb and `chainid` do not match, and are the two Body findings.
- The client list is back: [`## Known Implementations`](https://github.com/gnolang/gno/blob/52b515d15/docs/resources/gnoconnect.md?plain=1#L540) · [↗](../../../../../.worktrees/gno-review-5970/docs/resources/gnoconnect.md#L540) carries Gnoweb, Adena and Gnobro under a heading of their own, so round 1's second Warning is closed.
- No `state` finding: the spec recommends it for a producer that consumes callbacks, and gnoweb consumes none. [`_callbackURL`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L113-L118) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/frontend/js/controller-wallet-launch.ts#L113) strips `status` and `hash` from the next link and nothing reads them.
- The invariant catalog walk does not apply: the diff touches no gno code, only gnoweb Go, TypeScript, CSS and docs.

## Open questions

- The document now specifies `signtx` and `connect`, which part 1 implements nowhere. Whether the spec should ship ahead of the transport or land with part 2 is the author's call, and nothing in the diff is wrong either way. Not posted: it is a sequencing question, not a change to this diff.
- Realm-rendered markdown exec forms ([`ext_forms.go:468`](https://github.com/gnolang/gno/blob/52b515d15/gno.land/pkg/gnoweb/markdown/ext_forms.go#L468) · [↗](../../../../../.worktrees/gno-review-5970/gno.land/pkg/gnoweb/markdown/ext_forms.go#L468)) are a second Execute surface and still dead-end on mobile. Not posted: a follow-up scoping call, unchanged from round 1.
- Each registry entry inlines its icon into every `$help` response, 3346 bytes for gnokey today. Not posted: one entry is negligible, and the offline requirement drives the choice.
