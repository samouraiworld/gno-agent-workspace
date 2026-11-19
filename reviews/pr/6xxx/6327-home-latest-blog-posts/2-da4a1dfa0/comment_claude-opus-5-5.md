# PR [#6327](https://github.com/gnolang/gno/pull/6327): feat(home): show the latest blog posts, and let gnoweb re-read static pages
Event: COMMENT
Verdict: COMMENT, a single-finding pass: one Nit on the blog title filter in `gnoalias.sh`; the rest of the head since 4ee1b62ff was not re-read, so this pass approves nothing.
Model: claude-opus-5-5, single-finding pass at high effort, no finders or judges
Commit: da4a1dfa0 (latest)
Overview: [overview](../overview.md)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6327 da4a1dfa0`
Open the code: [github.dev](https://github.dev/alexiscolin/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c) · [vscode.dev](https://vscode.dev/github/alexiscolin/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c)
Round: 2. The head moved from 4ee1b62ff to da4a1dfa0. One finding from a security read of `latest_blogposts`, run by hand; the author answered both round 1 threads on the script.

## Body

## misc/deployments/home-alias/gnoalias.sh:37 [gh](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L37) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L37) · Nit

Nit: this `{ print }` copies a blog title's later lines into `home.md` as plain markdown, headings and links included, since [`prepareAndSetPost`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/examples/gno.land/p/gnoland/blog/v0/blog.gno#L176) lets a title hold newlines.

```suggestion
    /^(<gno-columns>|<gno-columns-sep>|<\/gno-columns>|[0-9][0-9] [A-Z][a-z][a-z] [0-9][0-9][0-9][0-9])$/ { print }
```

<details><summary>repro</summary>

A moderator post whose title holds a newline, rendered by `r/gnoland/blog`, then passed through the script's own `latest_blogposts`.

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
	ModAddPost(cross(cur), "upgrade", "Upgrade notice\n\n## Action required: move your GNOT\n[Migrate now](/r/attacker/migrate)", "body", "2026-10-30T00:00:00Z", "", "")
	println("RENDER-BEGIN")
	println(Render(""))
	println("RENDER-END")
}
EOF
(cd examples && gno test -v -run TestNewlineTitle ./gno.land/r/gnoland/blog 2>&1) | sed -n '/RENDER-BEGIN/,/RENDER-END/p' | sed '1d;$d' > render.md
sed -n '/^latest_blogposts() {/,/^}/p' misc/deployments/home-alias/gnoalias.sh > latest.sh
qrender() { cat render.md; }; . ./latest.sh; BLOG_LINK_BASE= latest_blogposts
```

The heading and the link from the first title come out as lines of their own, outside the bold title:

```text
<gno-columns>
**[Upgrade notice**


## Action required: move your GNOT
[Migrate now](/r/attacker/migrate)](/r/gnoland/blog:p/upgrade)
30 Oct 2026
<gno-columns-sep>
**[B](/r/gnoland/blog:p/b)**

29 Oct 2026
<gno-columns-sep>
**[C](/r/gnoland/blog:p/c)**

28 Oct 2026
</gno-columns>
```

With the suggested line, the same render keeps only the column tags, each title's first line and the dates, and the live mainnet blog index comes out byte for byte as it does today.
</details>
