# Review: [#6229](https://github.com/gnolang/gno/pull/6229)
Posted: https://github.com/gnolang/gno/pull/6229#pullrequestreview-5454204758

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Six Warnings come from code the branch adds: block-scalar front matter descriptions, indexed file views under registered community indexing, a broken canonical for dotted alias keys, self-canonical args variants, the per-node 4 KiB lead allocation and followed scheme-relative links in trusted realms.
Model: claude-opus-5-5, standard review
Commit: fedfdde13b4b289b28f4f291a414ded649aff5f3
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6229 fedfdde13`
Overview: [overview](../overview.md)
Open the code: [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3)
Round: 2. 5 finders, one reflector, 21 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted, six Warnings among the confirmed. The round 1 directory a 2026-09-30 batch created holds nothing, so this is the first review of the branch.

## Body

## SKIP gno.land/pkg/gnoweb/handler_http.go:306 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/handler_http.go#L306) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/handler_http.go#L306) · Nit

Nit: [`ServeHelpJSON`](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/handler_http.go#L306) returns before `classifyPage` runs, so `$help&json` is served with no `X-Robots-Tag` while `$help` is noindex.

Not posted: the line sits outside the diff and no judge ran the claim.

## gno.land/pkg/gnoweb/alias_static.go:48 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/alias_static.go#L48) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L48) · Warning [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004555)

The `continue` on indented lines drops every continuation of a front matter value. A `description: >` or `description: |` then becomes the literal `>` or `|` in the [page's description, og and twitter meta](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/alias_static.go#L58-L59), and a wrapped plain value keeps only its first line.

<details><summary>observed output</summary>

```
folded block scalar   title="About" description=">"
literal block scalar  description="|"
multi-line plain      description="What gno.land is"
<meta name="description" content="&gt;" />
<meta property="og:description" content="&gt;" />
<meta name="twitter:description" content="&gt;" />
```
</details>

## gno.land/pkg/gnoweb/community_index.go:104 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/community_index.go#L104) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/community_index.go#L104) · Warning [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004572)

The registered branch never checks `u.File` or a directory path, so `/r/nym/app/render.gno` and `/r/nym/app/` get indexed. The same source at `$source` stays `noindex, nofollow`.

<details><summary>observed output</summary>

```
/r/nym/app$source       X-Robots-Tag="noindex, nofollow" canonical=""
/r/nym/app/render.gno   X-Robots-Tag="" canonical="https://gno.land/r/nym/app/render.gno"  robots "index, follow"
/r/nym/app/             X-Robots-Tag="" canonical="https://gno.land/r/nym/app/"
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:329 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/handler_http.go#L329) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/handler_http.go#L329) · Warning [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004582)

An alias key such as `/license.md` or `/Terms` parses as a file under path `/`, so its canonical reads `//license.md`, a URL that answers 400. [`pagePolicy.kind`](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/page_kind.go#L103) then looks up `aliases["/"]` rather than the key, so the page loses its front matter title and description.

<details><summary>observed output</summary>

```
/about        status=200 canonical="https://gno.land/about"       title="Chosen Title - gno.land"
/license.md   status=200 canonical="https://gno.land//license.md" title="gno.land"
/Terms        canonical="https://gno.land//Terms"
//license.md  status=400 title="Invalid path - gno.land"
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:1435 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/handler_http.go#L1435) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/handler_http.go#L1435) · Warning [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004597)

`canonicalURL` keeps args and a trailing slash, so every args variant of a `$source` view or a package listing names itself as canonical and gets indexed. [`StaticHeaderDevLinks`](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/components/layout_header.go#L109-L111) links every args page to such a copy.

<details><summary>observed output</summary>

```
/r/gnoland/pages:p/a$source  robots="index, follow" canonical="https://gno.land/r/gnoland/pages:p/a$source"
/p/gnoland/lib:anything      canonical="https://gno.land/p/gnoland/lib:anything"
/p/gnoland/lib/              canonical="https://gno.land/p/gnoland/lib/"
```
</details>

## gno.land/pkg/gnoweb/markdown/description.go:118 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/markdown/description.go#L118) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L118) · Warning [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004614)

`bufio.NewWriter` allocates a 4 KiB buffer for every text node, on [every render](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/render.go#L151). A 1 MB lead paragraph of emphasis then allocates 2.2 GB, about 11 times what parsing and rendering it costs.

```suggestion
	w := bufio.NewWriterSize(&buf, 16)
```

<details><summary>observed output</summary>

A 16-byte writer keeps the markdown tests green and brings every case under parse and render.

```
emphasis 1000010 B   parse+render 202563320 B | Lead 2210255944 B  alloc 10.9x
code spans 1000010 B parse+render 128628240 B | Lead 1135059600 B  alloc 8.8x
with NewWriterSize(&buf, 16): emphasis 0.8x, links 0.3x, code spans 0.9x
```
</details>

## gno.land/pkg/gnoweb/markdown/ext_links.go:257 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/markdown/ext_links.go#L257) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/ext_links.go#L257) · Warning [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004650)

`detectLinkType` counts `//casino.example` and `https:///casino.example` as in-site, and in-site links are followed here, so a post in a trusted realm plants a followed off-site link with no `rel`.

<details><summary>observed output</summary>

```
//casino.example        -> <a href="//casino.example">
https:///casino.example -> <a href="https:///casino.example">
https://casino.example  -> <a href="https://casino.example" rel="noopener nofollow ugc">
```
</details>

## SKIP gno.land/cmd/gnoweb/main.go:311 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/cmd/gnoweb/main.go#L311) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/cmd/gnoweb/main.go#L311) · Missing test

Missing test: no test covers `setupWeb` passing `-canonical-origin` and `-index-community` to `AppConfig`, so dropping either assignment keeps every test green.

Not posted: no verifier ran the deletion.

## SKIP gno.land/cmd/gnoweb/main_test.go:122 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/cmd/gnoweb/main_test.go#L122) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/cmd/gnoweb/main_test.go#L122) · Missing test

Missing test: `TestSetupTrust` expects the same `DefaultTrustedPaths` that `NewDefaultAppConfig` already fills, so it passes with the trusted paths assignment removed.

Not posted: no verifier ran the revert.

## SKIP gno.land/pkg/gnoweb/alias_static.go:75 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/alias_static.go#L75) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L75) · Missing test

Missing test: no test pins the first-character check in `isFrontMatterKey`, so a key like `2024` is accepted with that check removed.

Not posted: no verifier ran the mutation.

## gno.land/adr/pr6229_gnoweb_page_metadata.md:314 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/adr/pr6229_gnoweb_page_metadata.md?plain=1#L314) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/adr/pr6229_gnoweb_page_metadata.md#L314) · Nit [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004658)

Nit: the ADR names a `HeadData.NoIndex` field that master never had and omits that [`RenderRealm`](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/render.go#L64) returns `md.RealmMeta` in place of `md.Toc`, which every out-of-tree renderer must follow.

## gno.land/pkg/gnoweb/alias_static.go:23 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/alias_static.go#L23) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L23) · Nit [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004666)

Nit: `CutPrefix(content, "---\n")` and the [closing cut](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/alias_static.go#L27) require exact LF bytes. CRLF endings or a BOM show the front matter as a horizontal rule and a heading.

## gno.land/pkg/gnoweb/alias_static.go:54 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/alias_static.go#L54) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/alias_static.go#L54) · Nit [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004677)

Nit: `strings.Trim` strips every outer quote rather than one wrapping pair, so `title: Learn "Gno"` becomes `Learn "Gno`. A `''` escape or a `# comment` also stays in the title.

## gno.land/pkg/gnoweb/markdown/description.go:43 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/markdown/description.go#L43) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L43) · Nit [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004684)

Nit: the leading loop skips only empty paragraphs, so a realm opening on an HTML banner loses its title and description. A markdown image banner keeps both.

## gno.land/pkg/gnoweb/markdown/description.go:73 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/markdown/description.go#L73) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L73) · Nit [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004695)

Nit: dropping every `unicode.Cf` rune also removes the zero-width joiner, the zero-width non-joiner and flag tags. `# 👩‍💻 Dev notes` then shows two glyphs in `<title>`.

## SKIP gno.land/pkg/gnoweb/markdown/description.go:85 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/markdown/description.go#L85) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L85) · Nit

Nit: `visibleText` skips only images, so under `-html` text a page hides with inline HTML such as `<span hidden>` reaches the description.

Not posted: no judge ran the claim.

## gno.land/pkg/gnoweb/markdown/description.go:144 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/markdown/description.go#L144) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/markdown/description.go#L144) · Nit [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004706)

Nit: `truncateRunes` backs off to the last space with no floor, so `# Welcome to gno.land/r/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/home` becomes the title `Welcome to…`.

## contribs/gnodev/setup_web.go:25 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/contribs/gnodev/setup_web.go#L25) · [↗](../../../../../.worktrees/gno-review-6229/contribs/gnodev/setup_web.go#L25) · Suggestion [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004716)

Suggestion: `TrustedPaths = []string{"*"}` applies to `gnodev staging` too, which the [gnodev docs](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/docs/resources/gnodev.md?plain=1#L53) tune for server use. Staging then treats every deployer's realm as official, and I think the wildcard belongs to local mode only.

## SKIP gno.land/cmd/gnoweb/main.go:286 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/cmd/gnoweb/main.go#L286) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/cmd/gnoweb/main.go#L286) · Suggestion

Suggestion: under `-html`, community realms can render a raw `<a>` tag with no `rel`.

Not posted: no judge ran the claim.

## SKIP gno.land/pkg/gnoweb/page_kind.go:143 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/page_kind.go#L143) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/page_kind.go#L143) · Suggestion

Suggestion: `mayRepeat` lets a trusted realm publish a user's paragraph as its official description, and no realm in the tree shows one.

Not posted: no in-tree instance, and the trusted list is the documented control.

## gno.land/pkg/gnoweb/trusted_paths.go:42 [gh](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/trusted_paths.go#L42) · [↗](../../../../../.worktrees/gno-review-6229/gno.land/pkg/gnoweb/trusted_paths.go#L42) · Suggestion [posted](https://github.com/gnolang/gno/pull/6229#discussion_r4217004728)

Suggestion: `trustedPathProblem` accepts `gnoland/*`, which [`contains`](https://github.com/gnolang/gno/blob/fedfdde13b4b289b28f4f291a414ded649aff5f3/gno.land/pkg/gnoweb/trusted_paths.go#L23) never matches, so gno.land's own pages become community pages with no warning. I think any `*` other than the bare one should be reported.
