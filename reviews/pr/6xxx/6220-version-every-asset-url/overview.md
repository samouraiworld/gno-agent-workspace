# Cache-busting the assets gnoweb's page head asks for
claude-opus-5, xhigh

## TLDR

gnoweb puts a build stamp on some of the URLs in its HTML `<head>` and not
others, so a cache sitting in front of the server keeps handing out last
release's favicon, code-highlighting colours and fonts. This change stamps the
four that were missing it.

## What it is for

[gnoweb](https://github.com/gnolang/gno/tree/7e6bc5514/gno.land/pkg/gnoweb) is
the web server behind gno.land: it renders realms and packages as pages. Like
any site it ships static files beside the HTML, a stylesheet, two fonts, an
icon, and it sits behind an edge cache that stores copies of those files close
to readers.

An edge cache stores a copy under the URL it was asked for. Ask for
`/public/main.css` today and again next week and you get the stored copy both
times, until the time-to-live runs out. That is the point: the file travels
once and is served from nearby afterwards.

It also means a release that changes the file changes nothing readers see. The
URL did not move, so the cache answers from the copy it already holds, and the
site keeps the old stylesheet until the TTL expires. Raising the TTL makes the
site faster and the staleness longer, so an operator who cannot version the
URLs is stuck with a short one.

The standard answer is to put something in the URL that changes per release.
`/public/main.css?v=20260920120000` is a different URL from
`/public/main.css?v=20260101000000`, so a new build gets a fresh entry and
readers get the new file on the first request. The file on disk keeps its name;
only what the page asks for moves.

## How it works today

The page head is one Go template,
[`layouts/head.html`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/components/layouts/head.html),
filled from three values the server computes once at startup:

| Value | What it holds | Set at |
| --- | --- | --- |
| `AssetsPath` | the prefix the static files are served under, `/public/` | [`app.go:104`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/app.go#L104) |
| `ChromaPath` | the syntax-highlighting stylesheet's own path | [`app.go:126`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/app.go#L126) |
| `BuildTime` | the stamp that changes per build | [`app.go:139`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/app.go#L139) |

Before this change the stamp reached the main stylesheet and the scripts in the
footer, and not the other four. The head as the merge base emits it, with the
one versioned line last:

```html
<!-- before -->
<link rel="preload" href="/public/fonts/intervar/Intervar.woff2" as="font" type="font/woff2" crossorigin />
<link rel="preload" href="/public/fonts/roboto/roboto-mono-normal.woff2" as="font" type="font/woff2" crossorigin />
<link rel="icon" href="/public//favicon.ico" type="image/x-icon" />
<link rel="stylesheet" href="/public/_chroma/style.css" />
<link rel="stylesheet" href="/public/main.css?v=20260920235729" />
```

The icon line also carries a doubled slash, `/public//favicon.ico`. `AssetsPath`
already ends in a slash, [enforced at `app.go:104`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/app.go#L104),
and the template added a second one. Servers resolve it, and a cache keyed on
the exact string stores it under a path no other link uses.

## What the change does

Four lines gain `?v={{ .BuildTime }}` and the icon line loses its extra slash.
The head as the branch emits it:

```html
<!-- after -->
<link rel="preload" href="/public/fonts/intervar/Intervar.woff2?v=20260920235649" as="font" type="font/woff2" crossorigin />
<link rel="preload" href="/public/fonts/roboto/roboto-mono-normal.woff2?v=20260920235649" as="font" type="font/woff2" crossorigin />
<link rel="icon" href="/public/favicon.ico?v=20260920235649" type="image/x-icon" />
<link rel="stylesheet" href="/public/_chroma/style.css?v=20260920235649" />
<link rel="stylesheet" href="/public/main.css?v=20260920235649" />
```

A test comes with it,
[`TestIndexLayoutVersionsEveryAssetURL`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/components/layout_test.go#L606-L635),
which renders the layout and asserts each of the five carries the stamp and
that no URL holds a doubled slash.

## Concepts

**Preload.** A `<link rel="preload">` tells the browser to start fetching a file
before the stylesheet that needs it has been parsed. The browser keeps the
result in a small holding area and hands it to whichever later request asks for
the same thing. Matching is by URL: a preload of `font.woff2?v=1` answers a
request for `font.woff2?v=1` and not one for `font.woff2`. A preload nothing
claims is downloaded and thrown away.

**Where a font URL is written.** The two fonts are named twice. The page head
preloads them, and the stylesheet declares them in its
[`@font-face` rules](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/frontend/css/04-elements.css#L12-L23)
as `url("./fonts/intervar/Intervar.woff2")`, relative to the stylesheet's own
location. The second is the one the browser actually fetches the font from; the
first only tries to get there first.

**The built stylesheet is checked in.**
[`public/main.css`](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/public/main.css)
is a build artifact committed to the repository, produced from
[`frontend/css/`](https://github.com/gnolang/gno/tree/7e6bc5514/gno.land/pkg/gnoweb/frontend/css)
by PostCSS, and the Go binary embeds and serves that file rather than the
sources. Editing a source stylesheet changes nothing a reader sees until the
bundle is rebuilt.
