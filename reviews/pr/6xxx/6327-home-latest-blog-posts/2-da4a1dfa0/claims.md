# Claims: #6327, round 2 at da4a1dfa0

A hand-run pass over `latest_blogposts`, the failed-query path and `staticFile`; no finders, judges or writer stage ran, so this file holds the three rows and their runs.

| Line | # | Band | State | Base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`gnoalias.sh:37`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L37) | 1 | Nit | CONFIRMED | new, the base script has no blog query | not posted |
| [`alias_static.go:169-177`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/gno.land/pkg/gnoweb/alias_static.go#L169-L177) | 2 | Suggestion | CONFIRMED | new, the base reads the file once at startup | not posted |
| [`gnoalias.sh:67`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L67) | 3 | Nit | the reviewer's own note | new | not posted |

## 1. A multi-line blog title reaches `home.md` outside the bold title

- Claim: `{ print }` at line 37 copies every line between the first `<gno-columns>` and `</gno-columns>` that is not a `### ` line, so a title's later lines land in `home.md` unchanged, and a `<!--` among them hides every section below the blog.
- Check, the realm: `ModAddPost` as admin with a title holding `\n`, `gno test -v -run TestNewlineTitle` in a copy of `examples/` at da4a1dfa0, go1.25.9. It passed and the render carries the title's lines verbatim: [`tests/render_newline_title.md`](tests/render_newline_title.md), [`tests/render_comment_title.md`](tests/render_comment_title.md). `prepareAndSetPost` runs only `strings.TrimSpace` on the title, `blog.gno:176`.
- Check, live: [`tests/comment_title_hides_sections.sh`](tests/comment_title_hides_sections.sh), run verbatim in a clean export of da4a1dfa0 with this branch's `gnoweb` built from it. The served page's `<h2>` list stops at `Latest from the blog`; the same script with the title `Upgrade notice` alone lists all ten sections, ending at `Stay connected`. The generated page is [`tests/home_comment_title.md`](tests/home_comment_title.md).
- Posted repro: a hand-written blog render in place of the realm query, run verbatim on a copy of `misc/deployments/home-alias` at da4a1dfa0; its output puts the `<!--` line above the five headings the live run lost. The realm side rests on the run above.
- Check, every shape: [`tests/render_probe.go`](tests/render_probe.go) renders through `NewHTMLRenderer` and `RenderRealm` as `GetMarkdownView` does. Second-line `<!--`: table of contents 10 entries to 5, the Boards heading and the Read-the-blog button gone. Second-line ```` ``` ````: 10 to 5, the five sections inside a code block. `## `: 11 entries. `<details>`, `<gno-columns>`, `<gno-columns-sep>`: 10, unchanged. Inputs and outputs under [`tests/cases/`](tests/cases/).
- Fix, run both ways: [`tests/fix.diff`](tests/fix.diff). Every case above, plus a first title line of `<!--` or ```` ``` ````, renders the full page with the Boards heading and `Stay connected` last, `tests/cases/fixed_*.md`. On the live mainnet index, `vm/qrender gno.land/r/gnoland/blog:` read 2026-11-19, [`tests/render_mainnet.md`](tests/render_mainnet.md), its output is byte for byte the unfixed script's. The gnokey image the compose file runs is built `FROM alpine:3`, `Dockerfile:120`, so its awk is busybox: in `alpine:3.24.4`, BusyBox v1.37.0, `render_override` with and without the fix gives output byte for byte the GNU Awk 5.4.1 output, on the `<!--` title, the `## ` title and the mainnet index.
- Band: Nit, on the reviewer's call; the case for Warning was this: only the blog admin, its moderators or a passed govDAO proposal write titles, and the same text on the blog page hides only the rest of the blog page; on the homepage it removes half of gno.land's front page, against the script's own comment at lines 29-30 that a title becomes one bold paragraph.
- Not raised before: round 1's two script threads, `discussion_r4297641063` and `discussion_r4297641085`, cover the failed-query path and the interval.

## 2. `staticFile` caches a read that costs 2% of the render beside it

- Claim: reading the file on every request removes `mu`, `modTime`, `size` and `staticFileSettle`, and costs 7.6 µs against 373 µs for rendering the same page.
- Patch: [`tests/suggestion-read-per-request.diff`](tests/suggestion-read-per-request.diff), applied to a separate worktree at da4a1dfa0: `last atomic.Pointer[AliasTarget]`, `failing atomic.Bool`, the FIFO check kept, the doc comment of `NewStaticFileAlias` updated, `TestNewStaticFileAliasTrustsFutureModTime` deleted.
- Run, head: `go test -count=1 -race -timeout 2m ./gno.land/pkg/gnoweb/ -run 'TestNewStaticFileAlias|TestNewStaticAlias|TestResolve|Alias' -v`, ok, 13 PASS lines. Patched before the test deletion: `TestNewStaticFileAliasTrustsFutureModTime` fails, `expected "# One"`, actual `# Two`, since it pins the cache serving stale content. Patched with it deleted: ok, 12 PASS lines; `./gno.land/cmd/gnoweb/ -run Alias` ok; `go vet` and `gofmt -l` clean.
- Timing: [`tests/bench_read_vs_render.go`](tests/bench_read_vs_render.go), 2000 rounds each over `home.mainnet.md`, 5383 bytes, on this 16-core laptop: stat, read and `NewStaticAlias` 7.58 µs; `RenderRealm` 373.493 µs; ratio 49. The posted commands, run verbatim from the pushed raw URLs in a clean export of da4a1dfa0: the patch applies, both packages ok under `-race`, and the timing reads 9.093 µs against 407.178 µs, ratio 45. `GetMarkdownView` calls `RenderRealm` on every request, `handler_http.go:745`.

## 3. The reviewer's own note on a failed first query

- The reviewer's pending comment on `home.mainnet.md`, moved to the line that writes the page without posts and its grammar repaired: "fail" to "fails", "rpc" to "RPC", the second "default" dropped. When the query fails with no `home.md` yet, or after an edit to `home-override.md`, line 67 drops the marker and the section shows its heading and the `Read the blog` button until the next round, at most 300 s later.
- The original pending comment, id 4628715196 in pending review 5975027578, still sits on `home.mainnet.md`; `post-pr-review.py` folds a draft into that pending review, so it posts beside this section unless deleted first.
