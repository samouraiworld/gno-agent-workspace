# Mainnet and testnet home alias files, routed by visitor profile
Written by claude-opus-5-5.
PR: [gnolang/gno#6293](https://github.com/gnolang/gno/pull/6293)

## TLDR

Before, `home.mainnet.md` and `home.testnet.md` opened on a Boards banner linking `/r/gnoland/boards2/v0`, which has no board on either network, and the `Redirects` map in `gnoweb` sent `/newsletter` and `/r/demo/boards:gnolang/6` to pages that answer 404.
After, both home files open on a three-column "Start here" block (use, build, join), list the realms live on their network, move Boards down under a note with no link, and drop the `r/demo` card; the two dead entries leave `Redirects`.
`pages/ecosystem.md` describes GnoSwap as live on mainnet.
The files only reach visitors once the infra copies them as `home-override.md`.

## Before and after

| Place | Before | After |
| --- | --- | --- |
| `/` on gno.land, through `home.mainnet.md` | opens on Boards; "Explore the universe" lists Gnoscan and Akkadia | opens on "Start here"; "Live on mainnet" lists Adena, Gnoscan, GnoSwap, GovDAO, Validators, Blog |
| `/` on onyx, through `home.testnet.md` | opens on Boards; Faucet Hub at the end of a list | faucet is the first "use" link; "On this testnet" lists GovDAO, Validators, Blog |
| `/newsletter`, through `RedirectMiddleware` | 302 to `/r/gnoland/pages:p/newsletter`, a 404 | no entry; the path answers 404 |
| `/r/demo/boards:gnolang/6`, through `RedirectMiddleware` | 302 to `/r/demo/boards:gnolang/3`, a 404 | no entry; the path answers 404 |

## Read the code in this order

1. [`misc/deployments/home-alias/home.mainnet.md`](https://github.com/gnolang/gno/blob/c2e4e0207310376989b2b8d781b002a738fafcb2/misc/deployments/home-alias/home.mainnet.md): the page gno.land serves at `/`. Its in-page links, `#live-on-mainnet`, `#explore-packages-and-realms` and `#community`, rely on goldmark's `WithAutoHeadingID` slugs, so renaming a heading breaks them.
2. [`misc/deployments/home-alias/home.testnet.md`](https://github.com/gnolang/gno/blob/c2e4e0207310376989b2b8d781b002a738fafcb2/misc/deployments/home-alias/home.testnet.md): the same layout for onyx, with `#on-this-testnet` in place of the mainnet section.
3. [`gno.land/pkg/gnoweb/redirect.go`](https://github.com/gnolang/gno/blob/c2e4e0207310376989b2b8d781b002a738fafcb2/gno.land/pkg/gnoweb/redirect.go#L10-L17): the map after the change.
   ```go
   var Redirects = map[string]string{
   	"/blog":            "/r/gnoland/blog",
   	"/gor":             "/contribute",
   	"/game-of-realms":  "/contribute",
   	"/grants":          "/partners",
   	"/language":        "/gnolang",
   	"/getting-started": "/start",
   }
   ```
4. [`misc/deployments/home-alias/README.md`](https://github.com/gnolang/gno/blob/c2e4e0207310376989b2b8d781b002a738fafcb2/misc/deployments/home-alias/README.md#L10-L13): the table saying what differs between the two files.

## What a user notices

Every visitor to `/` on gno.land or onyx sees the new layout once the infra deploys the file; staging serves `r/gnoland/home` from the chain and is unchanged. A saved `gno.land/newsletter` link lands on a 404 page directly rather than after a redirect.

## Words used here

| Name | What it is |
| --- | --- |
| `home-override.md` | the file `gnoweb --aliases=/=static:<path>` serves at `/`; ops copies the matching `home.*.md` to it |
| `Redirects` | the package map `RedirectMiddleware` reads; a matching path gets a 302 to the mapped value |
| `WithAutoHeadingID` | the goldmark parser option in `render_config.go` that gives each heading an id: lowercase, spaces to `-`, other punctuation dropped |
| onyx | the testnet served at `onyx.testnets.gno.land`, chain id `onyx-1` |
