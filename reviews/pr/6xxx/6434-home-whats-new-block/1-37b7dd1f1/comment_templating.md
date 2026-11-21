# Comment: [#6434](https://github.com/gnolang/gno/pull/6434)
Event: ISSUE_COMMENT, posted in the conversation under the user's own name, no AI marker
Commit: 37b7dd1f1 (latest)
Status: draft, waits for `post`

## Body

I think it works and I like this new part of the home, but `whatsnew.sh` mixes two jobs, fetching and rendering. I'd keep the script for fetching and move the rendering to a Go [`text/template`](https://pkg.go.dev/text/template):

- `home.mainnet.md` becomes the template, replacing the [13 `render_*` functions](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L359-L497) and the [marker handling](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L500-L523).
- [`to_md`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L103-L113) becomes a template function, so the escaping stays in one place.
- #6437 already renders the devnet home [this way](https://github.com/gnolang/gno/blob/1f8f0b14a/contribs/gnodev/devnet_home.go#L26).
