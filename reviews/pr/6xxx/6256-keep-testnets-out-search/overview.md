# `-no-index`: keep a gnoweb deployment out of search engines

Written by claude-opus-5-5.

PR: [gnolang/gno#6256](https://github.com/gnolang/gno/pull/6256)

Review files: [6256-keep-testnets-out-search](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6256-keep-testnets-out-search)

## TLDR

Before, gnoweb had no switch to keep a deployment out of search, and the PR
body reports staging and old testnets indexed beside gno.land; a gnoweb built
from master answers `GET /robots.txt` with 400. Now `gnoweb -no-index` sends
`X-Robots-Tag: noindex, nofollow` on every response, writes the noindex robots
meta, names no canonical and serves no sitemap. Every deployment now serves
`/robots.txt`, marks action forms noindex, and with `-canonical-origin` lists
its alias pages in `/sitemap.xml`. It builds on
[#6229](https://github.com/gnolang/gno/pull/6229), which adds
`-canonical-origin` and the per-page noindex robots meta.

## What `-no-index` is

An operator starting gnoweb for a testnet, a staging chain or a preview adds
one flag, [`-no-index`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/cmd/gnoweb/main.go#L187-L192),
and every response asks search engines to leave it out of their results. The
flag is off by [default](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L66-L68),
so a deployment whose operator forgets every flag stays indexable. Its help
text says never to set it on mainnet.

Three things change for every deployment, with or without the flag:

- `/robots.txt` is a route of its own and answers 200.
- An action form, a realm URL carrying both `$help` and `func=`, gets the
  noindex robots meta.
- With `-canonical-origin` set, `/sitemap.xml` lists the operator's alias
  pages, and `robots.txt` points to it.

## How it works, in 5 steps

The after state, following one deployment from its flags to the bytes a
crawler reads.

1. **The flag reaches the router.** `setupWeb` copies it into
   [`AppConfig.NoIndex`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/cmd/gnoweb/main.go#L292),
   and `NewRouter`
   [logs the policy](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L128)
   at startup: `crawl policy {"noindex": true, "canonical_origin": "https://gno.land"}`.
2. **The canonical origin is dropped for every page.** `NewRouter`
   [empties `staticMeta.CanonicalOrigin`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L166-L170)
   under `-no-index`. `canonicalURL` then
   [returns an empty string](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L1430-L1432),
   and the [share image](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L1452-L1454)
   is never set, even where the operator copied mainnet's `-canonical-origin`.
3. **The crawl routes are registered.** `NewRouter`
   [mounts `/robots.txt` and `/sitemap.xml`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L249-L255)
   with an empty sitemap origin under `-no-index`, so `robots.txt`
   [allows everything](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L30-L31)
   and the sitemap [answers 404](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L64-L67).
4. **Every response gets the header.** `NewRouter`
   [wraps the whole mux](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L256-L258)
   in `noIndexMiddleware`, which
   [sets `X-Robots-Tag: noindex, nofollow`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L21)
   before any handler runs. A build from the head returns it on a page, on
   `/public/favicon.ico`, on `/status.json` and on `/robots.txt` alike.
5. **Every page head says noindex.** `HTTPHandler.Get`
   [copies `Static.NoIndex` into the head data](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L257-L258),
   and `head.html`
   [writes it as the robots meta](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/components/layouts/head.html#L54).

One request, after: a gnoweb built from the head with `-no-index
-canonical-origin https://gno.land -no-default-aliases` and one static alias,
`/terms`. The output is trimmed to the crawl-relevant lines.

```text
GET /robots.txt
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
X-Robots-Tag: noindex, nofollow

# Not indexed: every response carries X-Robots-Tag: noindex.
User-agent: *
Allow: /

GET /sitemap.xml
HTTP/1.1 404 Not Found
X-Robots-Tag: noindex, nofollow

GET /terms
HTTP/1.1 200 OK
X-Robots-Tag: noindex, nofollow
<meta name="robots" content="noindex, nofollow" />
```

The same three requests, after, on the same build without `-no-index`:

```text
GET /robots.txt
HTTP/1.1 200 OK
User-agent: *
Disallow: /search.json
Disallow: /status.json

Sitemap: https://gno.land/sitemap.xml

GET /sitemap.xml
HTTP/1.1 200 OK
Content-Type: application/xml; charset=utf-8
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://gno.land/terms</loc></url></urlset>

GET /terms
HTTP/1.1 200 OK
<link rel="canonical" href="https://gno.land/terms" />
<meta property="og:url" content="https://gno.land/terms" />
<meta name="robots" content="index, follow" />
```

## The parts, at a glance

The after state.

| Part | File | Job |
| --- | --- | --- |
| [`-no-index`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/cmd/gnoweb/main.go#L187-L192) | `gno.land/cmd/gnoweb/main.go` | The operator's switch, off by default. |
| [`AppConfig.NoIndex`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L66-L68) | `gno.land/pkg/gnoweb/app.go` | Carries the switch into `NewRouter`. |
| [canonical clearing](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L166-L170) | `gno.land/pkg/gnoweb/app.go` | Empties the origin every page head builds its canonical from. |
| [crawl routes](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L249-L258) | `gno.land/pkg/gnoweb/app.go` | Mounts `/robots.txt` and `/sitemap.xml`, and wraps the mux under `-no-index`. |
| [`noIndexMiddleware`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L17-L24) | `gno.land/pkg/gnoweb/crawl.go` | Sets `X-Robots-Tag` on every response. |
| [`handlerRobotsTXT`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L26-L44) | `gno.land/pkg/gnoweb/crawl.go` | Builds the `robots.txt` body once at startup and serves it to `GET` and `HEAD`. |
| [`handlerSitemapXML`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L60-L92) | `gno.land/pkg/gnoweb/crawl.go` | Answers 404 without an origin, 502 when a realm check fails, else the XML. |
| [`sitemapAliases`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L94-L131) | `gno.land/pkg/gnoweb/crawl.go` | Picks which aliases to list and checks each target realm on the chain. |
| [`isCleanWebPath`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L133-L137) | `gno.land/pkg/gnoweb/crawl.go` | Keeps an alias only when gnoweb serves it at that exact URL. |
| [`StaticMetadata.NoIndex`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L51) | `gno.land/pkg/gnoweb/handler_http.go` | Hands the switch to the page handler. |
| [action-form noindex](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L409-L412) | `gno.land/pkg/gnoweb/handler_http.go` | Marks `$help&func=` pages noindex on every deployment. |
| [noindex canonical clearing](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L414-L419) | `gno.land/pkg/gnoweb/handler_http.go` | Drops canonical, `og:url` and share image from any noindex page. |
| [robots meta](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/components/layouts/head.html#L54) | `gno.land/pkg/gnoweb/components/layouts/head.html` | Writes `noindex, nofollow` or `index, follow`; unchanged by this PR. |

## Read the code in this order

1. [`main.go:187-192`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/cmd/gnoweb/main.go#L187-L192)
   declares the flag. Its default is `false`, since
   [`defaultWebOptions`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/cmd/gnoweb/main.go#L81-L88) sets no
   `noIndex`; a default of `true` would take mainnet out of search.

   ```go
   	fs.BoolVar(
   		&c.noIndex,
   		"no-index",
   		defaultWebOptions.noIndex,
   		"keep this deployment out of search engines (testnets, staging); never on mainnet",
   	)
   ```

2. [`app.go:166-170`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L166-L170)
   empties the canonical origin before any handler copies it. If it stayed, a
   testnet run with mainnet's flags would name gno.land as the canonical of
   every page, and the code comment records that Google may carry the noindex
   over to that target.

   ```go
   	if cfg.NoIndex {
   		staticMeta.CanonicalOrigin = ""
   	}
   ```

3. [`app.go:249-258`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L249-L258)
   mounts the two crawl routes and wraps the mux itself, so the middleware
   covers every route on it.

   ```go
   	sitemapOrigin := canonicalOrigin
   	if cfg.NoIndex {
   		sitemapOrigin = ""
   	}
   	mux.Handle("/robots.txt", handlerRobotsTXT(cfg.NoIndex, sitemapOrigin))
   	mux.Handle("/sitemap.xml", handlerSitemapXML(logger, sitemapOrigin, cfg.Aliases, adpcli))
   	if cfg.NoIndex {
   		return noIndexMiddleware(mux), nil
   	}
   ```

4. [`crawl.go:27-34`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L27-L34)
   picks the `robots.txt` body. Under `-no-index` it allows every path,
   because a crawler a `Disallow` keeps off a page never fetches it, and so
   [never sees its noindex header](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L44).

   ```go
   	body := "User-agent: *\nDisallow: /search.json\nDisallow: /status.json\n"
   	switch {
   	case noindex:
   		body = "# Not indexed: every response carries X-Robots-Tag: noindex.\nUser-agent: *\nAllow: /\n"
   	case sitemapOrigin != "":
   		body += "\nSitemap: " + sitemapOrigin + "/sitemap.xml\n"
   	}
   ```

5. [`crawl.go:99-131`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L99-L131)
   is `sitemapAliases`. It groups path aliases by target realm and sends one
   `ListFiles` call per realm, which the RPC client answers with a
   [`vm/qfile` query](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/client.go#L213-L220).
   A not-found error drops that realm's aliases; any other error fails the
   whole sitemap, so a node outage never publishes an empty list.

   ```go
   	for realm, realmAliases := range byRealm {
   		_, err := client.ListFiles(ctx, realm, 0)
   		switch {
   		case err == nil:
   			paths = append(paths, realmAliases...)
   		// A target with a file-like last segment is a bad alias, not an outage.
   		case !errors.Is(err, ErrClientPackageNotFound) && !errors.Is(err, ErrClientFileNotFound):
   			return nil, fmt.Errorf("%s: %w", realm, err)
   		}
   	}
   ```

6. [`handler_http.go:409-419`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L409-L419)
   marks action forms noindex and then strips the canonical from every
   noindex page, whatever set the flag: `-no-index`, an action form, or an
   error status.

   ```go
   	if gnourl.WebQuery.Has("help") && gnourl.WebQuery.Has("func") {
   		indexData.HeadData.NoIndex = true
   	}

   	// Google may carry a noindex over to the canonical target.
   	if indexData.HeadData.NoIndex {
   		indexData.HeadData.Canonical = ""
   		indexData.HeadData.URL = ""
   		indexData.HeadData.Image = ""
   	}
   ```

## What a user notices

Every effect below is decided per deployment, by the flags that gnoweb
process started with, except the action-form rule, which is decided per URL.
The package's own tests, `go test -count=1 ./gno.land/pkg/gnoweb/ -run
'Crawl|Robots|Sitemap|NoIndex'`, pass 5 tests and 13 subtests at the head and
assert every crawler and visitor row below.

| Who | Under `-no-index` | Default, with `-canonical-origin` |
| --- | --- | --- |
| A crawler fetching any URL | the header [`X-Robots-Tag: noindex, nofollow`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L189-L191) | [no `X-Robots-Tag`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L170-L172) |
| A crawler reading `/robots.txt` | [`Allow: /`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L47) | [two `Disallow` lines and a `Sitemap` line](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L32) |
| A crawler reading `/sitemap.xml` | [404](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L193) | [the alias pages](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L108-L115) |
| A crawler on `$help&func=Render` | [noindex, no canonical](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http_test.go#L2499) | the same: [`$help` alone stays indexable](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http_test.go#L2498) |
| A visitor sharing a link | a card with [no canonical and no `og:url`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L196-L200) | the canonical and `og:url` [under the origin](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L1446-L1449) |
| An operator | the [`crawl policy` line](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L128) in the startup log | the same line, plus [one `vm/qfile` query per target realm](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L119-L120) on each `/sitemap.xml` request |

Without `-canonical-origin`, a default deployment
[serves no `Sitemap` line](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L39)
and [answers 404 on the sitemap](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L120).
A visitor reading a page sees the same body under every flag.

The sitemap lists an alias only when gnoweb would serve a page at that exact
URL. The table below is the test's alias set at
[`crawl_test.go:61-74`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl_test.go#L61-L74),
after, with each outcome the test asserts.

| Alias | Target | In the sitemap |
| --- | --- | --- |
| `/`, `/about`, `/start` | realms on the chain | yes |
| `/src` | `/r/demo/boards$source` | yes, its realm exists |
| `/terms`, `/r/demo/boards` | static markdown | yes |
| `/events` | `/r/devrels/events`, absent from the chain | no |
| `/docs` | `/u/docs`, not a realm | no |
| `/blog` | a key in `Redirects` | no |
| `/terms.md`, `/foo bar`, `relative` | URLs gnoweb rewrites or rejects | no |

With the default aliases, the
[targets](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/app.go#L19-L31)
are `/r/gnoland/home`, `/r/gnoland/pages`, `/r/devrels/events` and `/u/docs`,
so a sitemap request costs three `vm/qfile` queries; the response carries
[`Cache-Control: public, max-age=3600`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L86).

## Words used here

| Name | What it is |
| --- | --- |
| `X-Robots-Tag` | An HTTP response header carrying the same directive as the robots meta, so it also covers a response that is not a page. |
| robots meta | The `<meta name="robots">` tag in a page head, [written by `head.html`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/components/layouts/head.html#L54) on every page gnoweb renders. |
| `noindex, nofollow` | Do not list this URL in results, and do not follow its links. |
| canonical | The `<link rel="canonical">` URL a page names as its own preferred address, [present only when the head has one](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/components/layouts/head.html#L23-L25). |
| [`-canonical-origin`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/cmd/gnoweb/main.go#L180-L185) | The flag from #6229 naming the public `scheme://host` a deployment is reached at, empty by default; every [canonical](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/handler_http.go#L1433) and [sitemap URL](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/crawl.go#L82) starts with it. |
| alias | An entry of `AppConfig.Aliases`: a short URL such as `/about` mapped to a realm path, kind `GnowebPath`, or to inline markdown, kind `StaticMarkdown`. |
| [`Redirects`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/redirect.go#L12) | Paths gnoweb redirects before it looks at aliases, so an alias under one of them is never the page served. |
| action form | The page at `<realm>$help&func=<name>` that fills a call to one realm function; a URL is one when its query has both `help` and `func`. |
| `vm/qfile` | The node query listing a package's files; the client [turns its unknown-package answer](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/client.go#L379-L386) into `ErrClientPackageNotFound`. |
| [`ErrClientPackageNotFound`](https://github.com/gnolang/gno/blob/c0086830daedaec5d152853345711e35e303be9a/gno.land/pkg/gnoweb/client.go#L21) | The client error for a path with no package; the sitemap treats it as "leave this alias out", never as an outage. |
