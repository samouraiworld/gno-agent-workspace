# Site review: mcp.gno.dev

Target: [gnoverse/gno-mcp#69](https://github.com/gnoverse/gno-mcp/issues/69)
Event: ISSUE
Pinned: `ef19be4`
Reverified: 2026-08-03, `ef19be4` still the tip of `main`
Posted: 2026-08-03, [gnoverse/gno-mcp#69](https://github.com/gnoverse/gno-mcp/issues/69). A manual pass by the user follows.

The body below is unwrapped, one line per paragraph, per `skills/writing-style.md`.

## Title
Site review: mcp.gno.dev (1 Critical, 7 Warnings, 14 Nits)

## Body
> Written by an agent. Every finding was re-derived against `ef19be4` before filing. No human has checked it yet.

Review of [`site/`](https://github.com/gnoverse/gno-mcp/tree/ef19be4/site) at `ef19be4` against [mcp.gno.dev](https://mcp.gno.dev/). The three deployed files are byte-identical to the repo, so every line number below resolves. [Full review.](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnomcp-site/review_claude-opus-5.md)

| Check | Result |
|:------|:-------|
| 26 tool names in the strip vs [`docs/tools.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/tools.md) | identical |
| Realm paths in the example prompts vs gnolang/gno [`examples/`](https://github.com/gnolang/gno/tree/master/examples/gno.land/r/gnoland/wugnot) | all three exist |
| Every external URL | 12 of 12 return 200 |
| WCAG contrast, 20 pairs, dark mode | 20 pass |
| Response headers vs [`netlify.toml`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/netlify.toml#L10-L18) | CSP `default-src 'none'`, HSTS, nosniff, frame-deny all served |
| Heading outline | one `h1`, no skipped levels |
| Cold-load transfer weight | 71.1 KB |

## Critical

### [`site/index.html:212`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L212)

> Everything read from the chain is wrapped as untrusted content, so a malicious realm can't steer the agent.

[`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md) §4 leaves three channels outside the envelope. `gno_read` with `full=true` or `symbols` is not wrapped, and §4 says that path "relies on the client honoring the resource boundary". Error text is neutralized, not enveloped. `structuredContent` "carries raw values", and §4 tells clients that surface those fields to a model to apply their own marking. §4 closes by telling the reader to treat anything from any channel as data.

## Warnings

### [`site/index.html:46-48`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L46-L48)

The "Works with" row names Claude Desktop and Cursor. [`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L52-L57) takes `claude|gemini|codex|opencode` and the `none` opt-out. Neither client appears again on the page, and nothing links to the Cursor section in [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md).

### [`site/style.css:108-109`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L108-L109)

`outline: 2px solid var(--mint)` measures 1.95:1 on `--bg` and 1.79:1 on `--surface`, against the 3:1 WCAG 2.2 SC 1.4.11 requires. The same ring measures 9.94:1 on the dark background, so every focusable element fails in light mode only. [`--accent-ink`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L32) is already in the palette at 5.06:1.

### [`site/style.css:18-19`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L18-L19)

Four light-mode pairs fall under 4.5:1 at sizes with no large-text exemption. `#0a0a0a99` clears it for both tokens.

| Element | Token | Size | Ratio |
|:--------|:------|-----:|------:|
| [`.tool-strip-note`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L618-L621), the link to `docs/tools.md` | `--fg-faint` | 13px | 3.74:1 |
| [`.eyebrow`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L177-L183), once per section | `--label` | 11px bold | 4.38:1 |
| [`.tool-row .needs`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L591-L597), five instances | `--label` | 11px bold | 4.38:1 |
| [`.step-n`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L674-L678) | `--label` | 12px bold | 4.38:1 |

### [`site/index.html:235`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L235)

"Get started" scrolls to `#install`, where the curl command repeats as `.mini-cmd` with no copy button. The one copy button on the page is the hero's, at [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38).

### [`site/index.html:3-12`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L3-L12)

No `og:*`, no `twitter:*`, no canonical, and `robots.txt` and `sitemap.xml` both return 404. The footer links X, Discord and Telegram, the three surfaces that unfurl link previews.

### [`site/index.html:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L15)

No skip link, so a keyboard or screen-reader user crosses the wordmark, five section links and the GitHub link before the `h1`. `main` already carries `id="top"` at [`site/index.html:29`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L29). WCAG 2.4.1 is Level A.

### [`site/style.css:158`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L158)

`.site-nav { display: none }` under 860px with no replacement, on a five-section page. Desktop has no scroll-spy or `aria-current` either.

## Nits

### [`site/index.html:140`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L140)

The showcased session prompt sets a two-week expiry. [`site/index.html:150`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L150) says keep scopes tight and spend limits low, dropping the third item from [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md#L25) §2, keep `expires_in` short.

### [`site/index.html:229`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L229)

"It verifies the checksum" carries no condition. [`scripts/install.sh:97-99`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L97-L99) warns and continues when neither `sha256sum` nor `shasum` is present.

### [`site/index.html:187`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L187)

"Connect and configure" is chipped "A gnoweb URL", but the row opens on [`gno_profile_list`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/tools.md#L47), plain config that is never dialed. [`site/index.html:192`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L192) chips "Your signature for sessions" on a row where six of nine tools are key management.

### [`site/index.html:85`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L85)

The stat reads 2 chains to start on. [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md#L128) ships `testnet`, its sunset predecessor and `local`.

### [`site/style.css:61-88`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L61-L88)

`:root[data-theme="light"]` and `:root[data-theme="dark"]` are complete palettes that nothing sets. What ships is the live [`:root`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L14) and its [`prefers-color-scheme: dark`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L44) override.

### [`site/app.js:1`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L1)

No enclosing function. `saEvent` and `io` land on `window`.

### [`site/app.js:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L15)

`flash` swaps `btn.textContent` on an element with no live region.

### [`site/app.js:27`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L27)

`flash("Press ⌘C")` names a Mac-only shortcut on a branch that runs on non-secure contexts and older browsers.

### [`site/index.html:11`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L11)

Neither self-hosted woff2 is preloaded, so both are discovered only after `style.css` parses.

### [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38)

`id="install-cmd"` is referenced nowhere. [`site/app.js:21`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L21) resolves the target through `btn.parentElement.querySelector("code")`.

### [`site/index.html:280`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L280)

The `noscript` pixel sets `referrerpolicy="no-referrer-when-downgrade"`, which sends the full URL to `queue.simpleanalyticscdn.com`. [`netlify.toml:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/netlify.toml#L15) serves `strict-origin-when-cross-origin` for everything else.

### [`site/index.html:279`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L279)

`scripts.simpleanalyticscdn.com/latest.js` runs with no `integrity` and no `crossorigin`, and the CSP admits the whole host. A rolling `latest.js` cannot be pinned by hash, so this is a risk to state rather than close.

### Repository metadata

`homepage` and `description` on [gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp) are both empty.

### 404 handling

An unknown path returns Netlify's default page.
