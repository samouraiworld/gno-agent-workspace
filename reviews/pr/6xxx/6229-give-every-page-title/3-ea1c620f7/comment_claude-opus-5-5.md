# Review: [#6229](https://github.com/gnolang/gno/pull/6229)

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Six Warnings come from code the branch adds or makes reachable: staging trust defaults in gnodev and in the loop deployment, a lowercase key line eating a static page's opening, a doubled-slash canonical for dotted realm alias keys, self-canonical args variants of fallback views, and view aliases taking the bare realm's canonical; an off-site /u/ link followed on trusted pages sits outside the diff.
Model: claude-opus-5-5, standard review: finders at xhigh, judges and reflector at medium, writer and text pass at low
Commit: ea1c620f7f5413f9a6db21883476c51b14ba8b9a
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6229 ea1c620f7`
Overview: [overview](../overview.md)
Open the code: [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a)
Round: 3. The head moved from fedfdde13 to ea1c620f7 through seven commits and two merges of master; of the six round-2 Warnings, block-scalar descriptions, registered indexing of file views, the 4 KiB lead buffer and followed scheme-relative links are fixed, while the dotted alias canonical and the args self-canonical are fixed for static aliases and pure packages and carried for realm aliases and fallback views. 5 finders, one reflector, 19 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted, at Nit.

## Body

- Warning: an off-site link with a `/u/` path, `https://casino.example/u/promo`, gets no `rel` on a trusted page and shows the user icon. [`detectLinkType`](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/markdown/ext_links.go#L298) returns the user type before comparing hosts.

  <details><summary>observed output</summary>

  ```
  /r/gnoland/forum  https://casino.example/u/promo -> <a href="https://casino.example/u/promo">
  /r/gnoland/forum  //casino.example/u/promo -> <a href="//casino.example/u/promo">
  /r/gnoland/forum  https://casino.example/promo -> <a href="https://casino.example/promo" rel="noopener nofollow ugc">
  --- FAIL: TestZZOffsiteUserPath
  ```
  </details>

## contribs/gnodev/setup_web.go:46 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/contribs/gnodev/setup_web.go#L46) · [↗](../../../../../.worktrees/gno-review-6229/contribs/gnodev/setup_web.go#L46) · Warning

Staging mode keeps gnoweb's default trusted list and `registered` indexing, and gnodev never enables `r/sys/names`. Any deployer under `gnoland` then sets an official page's title and description, with no gnodev flag to turn either off.

<details><summary>observed output</summary>

```
staging TrustedPaths=[gnoland sys gov ... gnoswap] IndexCommunity=registered
/r/gnoland/airdrop status=200 title="Official GNOT airdrop - gno.land" description="Claim your free GNOT airdrop at evil.example before it closes tonight." robots="index, follow"
PASS
```
</details>

## gno.land/cmd/gnoweb/main.go:84 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/cmd/gnoweb/main.go#L84) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/cmd/gnoweb/main.go#L84) · Warning

The staging gnoweb in [`misc/loop/docker-compose.yml`](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/misc/loop/docker-compose.yml) runs these defaults on a chain whose genesis never enables `r/sys/names`. A stranger's heading and first paragraph then become a `/r/gnoland` page's `<title>` and description.

<details><summary>observed output</summary>

```
TrustedPaths=[gnoland sys gov ... gnoswap] IndexCommunity=registered
status=200 title="Official GNOT airdrop - gno.land" description="Claim your free GNOT airdrop at evil.example before it closes tonight." robots="index, follow" X-Robots-Tag=""
PASS
```
</details>

## gno.land/pkg/gnoweb/alias_static.go:49 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/alias_static.go#L49) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L49) · Warning

`isFrontMatterKey` accepts any lowercase `word: text` line between two `---` rules. Such a page then loses everything up to the second rule, its heading included.

<details><summary>observed output</summary>

```
TestZZR3FrontMatter/lowercase  title="" description="" body="More.\n"   --- FAIL
TestZZR3LowercaseProseServed   <title>about - gno.land</title>  status=200 welcome=false note=false more=true   --- FAIL
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:329 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L329) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/handler_http.go#L329) · Warning

`weburl.ParseFromURL(&requested)` reads a realm alias keyed like a file, `/license.md` or `/Terms`, as a file under `/`, so its canonical and `og:url` become `https://gno.land//license.md`, which answers 400.

<details><summary>observed output</summary>

```
/license.md status=200 robots="index, follow" canonical="https://gno.land//license.md" og:url="https://gno.land//license.md" title="gno.land"
/Terms      canonical="https://gno.land//Terms"
//license.md status=400 title="Invalid path - gno.land"
--- FAIL: TestB3R3_RealmAliasDottedKey
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:1440 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L1440) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/handler_http.go#L1440) · Warning

This guard keeps the args on two views that ignore them: the [directory view](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L601-L603) of a realm with no `Render` and a namespace's [paths listing](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L607). Each args variant is indexed under a canonical naming itself.

<details><summary>observed output</summary>

```
/r/gnoland/norender:anything-at-all status=200 robots="index, follow" canonical="https://gno.land/r/gnoland/norender:anything-at-all"
/r/gnoland:anything-at-all          status=200 robots="index, follow" canonical="https://gno.land/r/gnoland:anything-at-all"
--- FAIL: TestB3R3_FallbackViewsKeepArgs
```
</details>

## gno.land/pkg/gnoweb/page_kind.go:77 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/page_kind.go#L77) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/page_kind.go#L77) · Warning

`aliasKey` drops the view, so an alias to `/r/gnoland/pages$source` becomes the rendered realm's canonical. An alias to `$help` points the realm at a `noindex` page.

<details><summary>observed output</summary>

```
/r/gnoland/pages status=200 robots="index, follow" canonical="https://gno.land/src" og:url="https://gno.land/src"
/r/gnoland/home  canonical="https://gno.land/actions"
/actions         status=200 robots="noindex, follow" canonical=""
--- FAIL: TestB3R3_ViewAliasTakesRealmCanonical
```
</details>

## SKIP gno.land/pkg/gnoweb/markdown/utils.go:213 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/markdown/utils.go#L213) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/utils.go#L213) · Missing test

Missing test: no test pins the space `nodeText` adds at a line break, which only the table-of-contents title at [toc.go:95](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/markdown/toc.go#L95) reads.

Not posted: no judge ran the deletion that would show the suite staying green.

## gno.land/pkg/gnoweb/alias_static.go:30 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/alias_static.go#L30) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L30) · Nit

Nit: a file holding only front matter with no final newline misses the `"\n---\n"` cut, so the page shows its YAML and gets no title.

## gno.land/pkg/gnoweb/alias_static.go:111 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/alias_static.go#L111) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L111) · Nit

Nit: `parts` gets an empty first entry when the title sits on the next line, so the joined value opens on a space and keeps both quotes.

## gno.land/pkg/gnoweb/alias_static.go:122 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/alias_static.go#L122) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L122) · Nit

Nit: a title keeps a trailing YAML comment and a doubled single quote, so `About # shipped with the node` and `It''s gno` reach the page head as written.

## gno.land/pkg/gnoweb/canonical_origin.go:34 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/canonical_origin.go#L34) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/canonical_origin.go#L34) · Nit

Nit: `host == ""` tests the host and port together, so `-canonical-origin https://:8443` starts cleanly and every canonical names an origin with no host.

## gno.land/pkg/gnoweb/canonical_origin_test.go:73 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/canonical_origin_test.go#L73) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/canonical_origin_test.go#L73) · Nit

Test: `newPagePolicy` here reads `DefaultAliases` in parallel with a write from `TestStaticMarkdownDevLinks` through [NewDefaultAppConfig](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/app.go#L120), so the package fails under `-race`.

<details><summary>observed output</summary>

```
WARNING: DATA RACE   (x5)
  app_test.go:181 write / page_kind.go:69 read <- canonical_origin_test.go:73
FAIL github.com/gnolang/gno/gno.land/pkg/gnoweb 37.359s
```
</details>

## gno.land/pkg/gnoweb/community_index.go:105 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/community_index.go#L105) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/community_index.go#L105) · Nit

Nit: `u.IsDir()` is true only for a path ending in `/`, so under `registered` the listing at `/r/nym` is indexed while `/r/nym/` is not.

## SKIP gno.land/pkg/gnoweb/community_index.go:114 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/community_index.go#L114) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/community_index.go#L114) · Nit

Nit: an official source file is indexed both at its file path and at `$source&file=`.

Not posted: the same defect as the section on handler_http.go:1455.

## gno.land/pkg/gnoweb/handler_http.go:1455 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L1455) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/handler_http.go#L1455) · Nit

Nit: a source file is indexed at `/r/x/render.gno` and at `/r/x$source&file=render.gno`, both [served by GetSourceView](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L569-L571) and each naming itself canonical.

## gno.land/pkg/gnoweb/markdown/description.go:52 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/markdown/description.go#L52) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L52) · Nit

Nit: the walk ends only at a top-level heading or rule, so a paragraph after a list, a quote or a table becomes the description, which the [godoc](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/markdown/description.go#L38-L39) rules out.

## gno.land/pkg/gnoweb/markdown/description.go:55 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/markdown/description.go#L55) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L55) · Nit

Nit: `[]rune(text)` copies the whole first paragraph twice to keep a short summary, so a long plain paragraph allocates more than rendering the page does.

## gno.land/pkg/gnoweb/render.go:151 [gh](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/render.go#L151) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/render.go#L151) · Suggestion

Suggestion: `md.Lead` runs on every render. The user page and README view [discard its result](https://github.com/gnolang/gno/blob/ea1c620f7f5413f9a6db21883476c51b14ba8b9a/gno.land/pkg/gnoweb/handler_http.go#L903) and community pages drop it, so computing it only for callers reading it saves the walk.
