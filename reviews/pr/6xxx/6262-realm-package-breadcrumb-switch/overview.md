# Breadcrumb kind switch: `startCounterpart` and the `kind-switch-menu` popover
Written by claude-opus-5-5.
PR: [gnolang/gno#6262](https://github.com/gnolang/gno/pull/6262)

## TLDR

On a `/r/<ns>/…` or `/p/<ns>/…` page, the breadcrumb's first chip was a plain link to `/r/` or `/p/`. It is now a `<button>` opening a native popover, `#kind-switch-menu`, with up to three links: the matching path on the other side when one exists, the namespace page `/u/<ns>`, and the old listing. The match is by path only: `/r/alice/golf/game` looks for packages under `/p/alice/golf`. The lookup runs beside the page's own queries and is cached a minute per project root.

## What the switch is

A menu on the `r`/`p` chip that jumps from a realm to the pure packages of the same project directory, or back. Nothing on chain ties the two sides, so the label says "Matching", never "related".

## How it works, in 4 steps

1. `generateBreadcrumbPaths` sets `Namespace` for realm and pure paths ([handler_http.go](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/handler_http.go#L1395)).
2. `startCounterpart` maps `/r/alice/golf/game` to twin `/p/alice/golf/game` and root `/p/alice/golf` (`counterpartRoots`), then calls `ListPaths(<domain>/p/alice/golf, 100)` in a goroutine with a 5 s timeout, through `counterpartCache` ([counterpart.go](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L148-L186)).
3. `counterpartTarget` picks the link: the twin alone, the twin's directory when it has siblings, or the deepest directory holding several matches, with `n` the count ([counterpart.go](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L75-L117)).
4. After the page's own view is built, the returned closure waits up to 300 ms for the link and sets `Breadcrumb.Counterpart`; the template renders it at the head of the menu ([header.html](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/components/layouts/header.html#L214-L249)).

## The parts, at a glance

| Part | File | Job |
| --- | --- | --- |
| `counterpartRoots`, `counterpartTarget`, `counterpartLink` | [`counterpart.go`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L48) | twin, root, target and label |
| `counterpartCache` | [`counterpart.go`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L191) | one-minute cache, singleflight, 4096 roots |
| `startCounterpart` call sites | [`handler_http.go`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/handler_http.go#L468-L475) | page and `?state` page paths; skipped for markdown, json and fragments |
| `Namespace`, `Counterpart` | [`ui_breadcrumb.go`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/components/ui_breadcrumb.go) | fields the header reads |
| chip and `#kind-switch-menu` | [`header.html`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/components/layouts/header.html#L214) | button with `popovertarget`, the menu |
| `.kind-switch`, `.b-kind-switch` | [`06-blocks.css`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/frontend/css/06-blocks.css#L3979) | anchor positioning, with an `@supports not` fallback |

## Read the code in this order

1. [`counterpartTarget`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L75-L117): decides which page the link opens and the count on its label; a wrong choice sends the user to a page that does not list what the label counts.
2. [`startCounterpart`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L148-L186): runs the lookup detached from the request and bounds how long the page waits.
3. [`counterpartCache.store`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/counterpart.go#L240-L264): what is kept, for how long, and what happens when 4096 roots are cached.
4. [`header.html`](https://github.com/gnolang/gno/blob/c5ae4b13a8883303a382d6b1ae1c61b929fd048c/gno.land/pkg/gnoweb/components/layouts/header.html#L214-L249): the chip and the menu markup.

## What a user notices

- Per page: the `r`/`p` chip shows a chevron and opens the menu; `/r/`, `/p/` and `/u/` pages keep the plain link.
- Per project root, for a minute: the matching line appears or not depending on the cached listing; a package deployed meanwhile shows up after the TTL.
- Per browser: the menu is anchored under the chip with CSS anchor positioning, with a fallback position where that is unsupported.
- Per deployment with analytics on: `breadcrumb_click` no longer counts the first chip or the menu links.

## Words used here

| Name | What it is |
| --- | --- |
| twin | the same path on the other side, `/p/alice/golf/game` for `/r/alice/golf/game` |
| root | the other side's project directory, the first two segments after the kind, `/p/alice/golf` |
| `counterpartGrace` | 300 ms, how long a ready page waits for a lookup still running |
| `counterpartTimeout` | 5 s, the lookup's own bound; it is not cancelled when the page stops waiting |
| `counterpartTTL` | one minute, how long a root's listing, empty included, is reused; errors are not cached |
| `maxCounterpartPaths` | 100, the `ListPaths` limit per root; past it the label reads `100+` |
