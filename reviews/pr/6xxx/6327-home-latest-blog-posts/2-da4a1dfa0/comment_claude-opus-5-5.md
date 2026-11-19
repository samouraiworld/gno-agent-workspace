# PR [#6327](https://github.com/gnolang/gno/pull/6327): feat(home): show the latest blog posts, and let gnoweb re-read static pages
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. A blog title whose second line is `<!--` hides every homepage section below the blog, and the branch is what copies that line into `home.md`; the rest of the head since 4ee1b62ff was read for the cache type and the failed-query path only.
Model: claude-opus-5-5, single-finding pass at high effort, no finders or judges
Commit: da4a1dfa0 (latest)
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6327 da4a1dfa0`
Open the code: [github.dev](https://github.dev/alexiscolin/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c) · [vscode.dev](https://vscode.dev/github/alexiscolin/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c)
Round: 2. The head moved from 4ee1b62ff to da4a1dfa0. Three findings from a hand-run read of `latest_blogposts`, the failed-query path and `staticFile`, each run here; the author answered both round 1 threads on the script.

## Body

## misc/deployments/home-alias/gnoalias.sh:37 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L37) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L37) · Warning

This `{ print }` copies a blog title's later lines into `home.md`, so a title whose second line is `<!--` hides every homepage section below the blog.

```suggestion
    /^(<gno-columns>|<gno-columns-sep>|<\/gno-columns>|[0-9][0-9] [A-Z][a-z][a-z] [0-9][0-9][0-9][0-9])$/ { print }
```

<details><summary>repro</summary>

A moderator post titled `Upgrade notice` then `<!--` on a second line, rendered by `r/gnoland/blog`, passed through the script's own `render_override`, and served by this branch's gnoweb.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6327 -R gnolang/gno
cat > examples/gno.land/r/gnoland/blog/newline_title_test.gno <<'EOF'
package blog

import "testing"

func TestNewlineTitle(cur realm, t *testing.T) {
	clearState(t)
	testing.SetOriginCaller(adminAddr)
	testing.SetRealm(testing.NewUserRealm(adminAddr))
	ModAddPost(cross(cur), "b", "B", "body", "2026-10-29T00:00:00Z", "", "")
	ModAddPost(cross(cur), "c", "C", "body", "2026-10-28T00:00:00Z", "", "")
	ModAddPost(cross(cur), "upgrade", "Upgrade notice\n<!--", "body", "2026-10-30T00:00:00Z", "", "")
	println("RENDER-BEGIN")
	println(Render(""))
	println("RENDER-END")
}
EOF
(cd examples && go run ../gnovm/cmd/gno test -v -run TestNewlineTitle ./gno.land/r/gnoland/blog 2>&1) | sed -n '/RENDER-BEGIN/,/RENDER-END/p' | sed '1d;$d' > render.md
sed -n '/^BLOG_MARKER=/p;/^latest_blogposts() {/,/^}/p;/^render_override() {/,/^}/p' misc/deployments/home-alias/gnoalias.sh > funcs.sh
mkdir -p home && cp misc/deployments/home-alias/home.mainnet.md home/home-override.md
(cd home && qrender() { cat ../render.md; } && . ../funcs.sh && BLOG_LINK_BASE= render_override > home.md)
go build -o gnoweb-bin ./gno.land/cmd/gnoweb
./gnoweb-bin --aliases=/=static:$PWD/home/home.md --bind 127.0.0.1:8899 > gnoweb.log 2>&1 &
until curl -sf -o /dev/null http://127.0.0.1:8899/; do sleep 1; done
curl -s http://127.0.0.1:8899/ | grep -o "<h2[^>]*>.*</h2>" | sed "s/<[^>]*>//g"
kill %1
```

The page ends at the blog heading, where the same run with a one-line title lists five more sections, from `Boards: on-chain forum` to `Stay connected`:

```text
Contents
 Use apps
 Build with Gno
 Community
 Live on mainnet
 Latest from the blog
```

[`prepareAndSetPost`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/examples/gno.land/p/gnoland/blog/v0/blog.gno#L176) trims only the ends of a title, so the realm stores the newline, and the markdown parser reads `home.md` whole, so the comment opened inside the posts runs to the end of the page. A second line of ```` ``` ```` turns the same five sections into one code block, and `## ` adds an entry to the table of contents. With the suggested line, all three cases and a title whose first line is `<!--` or ```` ``` ```` render the full page, and the live mainnet blog index comes out byte for byte as it does today.
</details>

## gno.land/pkg/gnoweb/alias_static.go:169-177 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/gno.land/pkg/gnoweb/alias_static.go#L169-L177) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static.go#L169) · Suggestion

Suggestion: read `home.md` on every request and drop the lock, `modTime`, `size` and the settle window, since the read costs 7.6 µs beside a 373 µs render.

<details><summary>patch and timing</summary>

The patch keeps the last good page in an `atomic.Pointer` and deletes `TestNewStaticFileAliasTrustsFutureModTime`, which pins the cache serving stale content; the other alias tests pass with `-race`.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6327 -R gnolang/gno
base=https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6327-home-latest-blog-posts/2-da4a1dfa0/tests
curl -fsSL $base/suggestion-read-per-request.diff | git apply
go test -count=1 -race ./gno.land/pkg/gnoweb/ ./gno.land/cmd/gnoweb/ -run Alias
mkdir -p .bench && curl -fsSL -o .bench/main.go $base/bench_read_vs_render.go && go run ./.bench misc/deployments/home-alias/home.mainnet.md
```

```text
stat+read+NewStaticAlias: 7.58µs per request
RenderRealm:              373.493µs per request
```
</details>

## misc/deployments/home-alias/gnoalias.sh:67 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L67) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L67) · Nit

nit: by default, you should add a sentence if the script fails to fetch from the RPC.
