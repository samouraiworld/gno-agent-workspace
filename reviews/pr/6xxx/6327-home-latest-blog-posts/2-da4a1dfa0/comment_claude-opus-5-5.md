# PR [#6327](https://github.com/gnolang/gno/pull/6327): feat(home): show the latest blog posts, and let gnoweb re-read static pages
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. A blog title whose second line is `<!--` hides every homepage section below the blog, and the branch is what copies that line into `home.md`; the rest of the head since 4ee1b62ff was read for the cache type and the failed-query path only.
Model: claude-opus-5-5, hand-run pass at high effort, no finders or judges
Commit: da4a1dfa0 (latest)
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6327 da4a1dfa0`
Open the code: [github.dev](https://github.dev/alexiscolin/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c) · [vscode.dev](https://vscode.dev/github/alexiscolin/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c)
Round: 2. The head moved from 4ee1b62ff to da4a1dfa0. Two findings run here, on `latest_blogposts` and `staticFile`, and the reviewer's own note on the failed-query path; the author answered both round 1 threads on the script.

## Body

## misc/deployments/home-alias/gnoalias.sh:37 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L37) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L37) · Warning

This `{ print }` copies a blog title's later lines into `home.md`, so a title whose second line is `<!--` hides every homepage section below the blog.

```suggestion
    /^(<gno-columns>|<gno-columns-sep>|<\/gno-columns>|[0-9][0-9] [A-Z][a-z][a-z] [0-9][0-9][0-9][0-9])$/ { print }
```

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6327 -R gnolang/gno
cd misc/deployments/home-alias
printf '<gno-columns>\n### [Upgrade notice\n<!--](/r/gnoland/blog:p/x)\n30 Oct 2026\n</gno-columns>\n' > render.md
sed -n '/^BLOG_MARKER=/p;/^latest_blogposts() {/,/^}/p;/^render_override() {/,/^}/p' gnoalias.sh > funcs.sh
cp home.mainnet.md home-override.md
qrender() { cat render.md; }; . ./funcs.sh; render_override | grep -E '^<!--|^## ' | sed 's/<gno-icon[^>]*> //; s/](.*//'
rm render.md funcs.sh home-override.md
```

The `<!--` from the title opens a comment no later line closes, so gnoweb serves only the five sections above it:

```text
## Use apps
## Build with Gno
## Community
## Live on mainnet
## Latest from the blog
<!--
## Boards: on-chain forum
## Explore packages and realms
## About Gno.land
## Developer resources
## Stay connected
```

[`prepareAndSetPost`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/examples/gno.land/p/gnoland/blog/v0/blog.gno#L176) trims only the ends of a title, so `ModAddPost` stores this one with its newline.
</details>

## gno.land/pkg/gnoweb/alias_static.go:169-177 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/gno.land/pkg/gnoweb/alias_static.go#L169-L177) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static.go#L169) · Suggestion

Suggestion: read `home.md` on every request and drop this cache, its lock and its 2 s settle window, since [`GetMarkdownView`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/gno.land/pkg/gnoweb/handler_http.go#L745) already renders the whole page on every request.

## misc/deployments/home-alias/gnoalias.sh:67 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L67) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L67) · Nit

nit: by default, you should add a sentence if the script fails to fetch from the RPC.
