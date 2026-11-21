# Comment: [#6434](https://github.com/gnolang/gno/pull/6434)
Event: ISSUE_COMMENT, posted in the conversation under the user's own name, no AI marker
Commit: 37b7dd1f1 (latest)
Status: draft, waits for `post`

## Body

I think it works and I like this new part of the home, but `whatsnew.sh` mixes two jobs, fetching the data and rendering the page, and the rendering half is hard to maintain in shell. A Go [`text/template`](https://pkg.go.dev/text/template) could take the rendering, leaving the script, or a small Go command, to fetch.

- The [13 `render_*` functions](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L359-L497) build the markdown with `printf` and `awk`, and [`markers_ok`, `page_ok` and `fill_block`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L500-L523) paste it between marker lines. With `home.mainnet.md` as the template, `{{range .Posts}}` replaces both, and a layout change becomes an edit to the page instead of to a shell function.
- The escaping stays in one place: [`to_md`](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/misc/deployments/home-alias/whatsnew.sh#L103-L113) becomes a template function, called as `{{.Text | md}}`. `html/template` cannot do this part, since its automatic escaping knows HTML and not markdown.
- In Go, the week's scan can call gnoweb's [`chainmap`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/gnoweb/feature/chainmap/activity.go) instead of copying it, which removes the drift the [ADR describes](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/gno.land/adr/pr6434_home_whats_new.md#L61-L64).
- #6437 already renders the devnet home this way, from [`devnet_home.md.tmpl`](https://github.com/gnolang/gno/blob/1f8f0b14a/contribs/gnodev/devnet_home.md.tmpl) through [`text/template`](https://github.com/gnolang/gno/blob/1f8f0b14a/contribs/gnodev/devnet_home.go#L26), so the three homes would share one approach.

Could the ADR's [alternatives](https://github.com/gnolang/gno/blob/37b7dd1f15a79144153303fe49a98e71f6ff9846/gno.land/adr/pr6434_home_whats_new.md#L78) at least weigh this against bash, `curl` and `jq`?
