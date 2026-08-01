# Site review: mcp.gno.dev

Target: new issue on [gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp)
Event: ISSUE
Pinned: `ef19be4`

## Title
Site review: mcp.gno.dev (1 Critical, 7 Warnings, 14 Nits)

## Body
Review of [mcp.gno.dev](https://mcp.gno.dev/) against
[`site/`](https://github.com/gnoverse/gno-mcp/tree/ef19be4/site),
[`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md),
[`docs/tools.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/tools.md),
and [`scripts/install.sh`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh).
Line numbers are pinned at `ef19be4`. The deployed HTML and CSS are
byte-identical to `site/`, so they resolve in the repo.

Clean on the checks that matter most: the 26 tool names in the strip match
`docs/tools.md` exactly, the realm paths in the example prompts exist in
`examples/`, all 12 external URLs resolve, the heading outline has no skips,
cold load is ~71 KB, and the CSP is `default-src 'none'` with no
`unsafe-inline`. Dark mode passes all 20 contrast pairs measured.

Full review:
https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnomcp-site/review_claude-opus-5.md
[↗](review_claude-opus-5.md)

### Critical: site/index.html:212 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L212)

"Everything read from the chain is wrapped as untrusted content, so a malicious
realm can't steer the agent."

[`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md)
§4 names three channels the envelope does not cover. `gno_read` with
`full=true` or `symbols` is deliberately not wrapped, and §4 says that path
"relies on the client honoring the resource boundary". Error text is
mixed-trust: neutralized, not enveloped. `structuredContent` "carries raw
values", and §4 tells clients that surface those fields to a model to apply
their own marking. §4 then instructs the reader to treat anything from any
channel as data, which is an instruction that exists because the envelope
alone does not finish the job.

The heading directly above reads "Safe by architecture, not by promise", and
the other five invariants hold against §1, §3, §5, §6, and §7. This is the one
claim a skeptical reader can use to discount the rest.

### site/index.html:46 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L46)

The "Works with" row names Claude Desktop and Cursor. Neither appears again
anywhere on the page.
[`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L52-L57)
accepts `claude|gemini|codex|opencode` plus the `none` opt-out, and
[`site/index.html:236`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L236)
says so accurately. A Cursor user is recruited by the hero, runs the one
command, and receives a binary and silence.
[`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md)
carries a Cursor section and an "Other MCP clients" section, and the page links
to neither.

### site/style.css:108-109 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L108-L109)

`outline: 2px solid var(--mint)` measures 1.95:1 on `--bg` and 1.79:1 on
`--surface`. WCAG 2.2 SC 1.4.11 requires 3:1. Every focusable element on the
page is affected in light mode: the section nav, both hero buttons, the copy
button, every documentation link, the footer. The same ring measures 9.92:1 on
the dark background. `--accent-ink` is already defined at `#3b7a64` in the
light block and reaches 5.06:1 on white.

### site/style.css:18-19 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L18-L19)

Four light-mode text pairs fall under 4.5:1 at sizes with no large-text
exemption.

| Element | Token | Size | Ratio |
|:--------|:------|-----:|------:|
| `.tool-strip-note`, the link to `docs/tools.md` | `--fg-faint` | 13px | 3.74:1 |
| `.eyebrow`, once per section | `--label` | 11px | 4.38:1 |
| `.tool-row .needs`, five instances | `--label` | 11px | 4.38:1 |
| `.step-n` | `--label` | 12px bold | 4.38:1 |

`#0a0a0a99` clears 4.5:1 for both tokens.

### site/index.html:235 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L235)

"Get started" scrolls to `#install`, where step 01 repeats the curl command as
`.mini-cmd` with no copy button. Only the hero copy at
[`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38)
has one, so the deliberate scroll-to-convert path ends on the worse version of
the thing the visitor came for.

### site/index.html:3-12 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L3-L12)

No `og:*`, no `twitter:*`, no canonical. The footer links X, Discord, and
Telegram, so the page is built to be shared into the three surfaces that unfurl
link previews and will unfurl as a bare URL in all three. `robots.txt` and
`sitemap.xml` both return 404.

### site/index.html:15 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L15)

No skip link, so there is no way to bypass the header. A keyboard or
screen-reader user crosses the wordmark, five section links, and the GitHub
link before reaching the `h1`. `main` already carries `id="top"` at
[`site/index.html:29`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L29),
so one anchor at the top of `body` closes it. WCAG 2.4.1 is Level A.

### site/style.css:158 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L158)

`.site-nav { display: none }` under 860px with no replacement. On a
five-section page the mobile header keeps the wordmark and the GitHub link, and
navigation becomes scrolling.

### Nit: site/index.html:140 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L140)

The showcased session prompt sets a two-week expiry. Ten lines down,
[`site/index.html:150`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L150)
says keep scopes tight and spend limits low, dropping the third item from
`docs/security.md` §2, which is to keep `expires_in` short. The panel labels
the feature Pre-release.

### Nit: site/index.html:229 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L229)

"It verifies the checksum" is unconditional.
[`scripts/install.sh:97-99`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L97-L99)
warns and continues when neither `sha256sum` nor `shasum` is present.

### Nit: site/index.html:187 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L187)

"Connect and configure" is chipped "A gnoweb URL", but the row opens on
`gno_profile_list`, which `docs/tools.md` describes as plain config, never
dialed. Same shape at
[`site/index.html:192`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L192):
"Your signature for sessions" chips a row where six of nine tools are key
management.

### Nit: site/index.html:84 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L84)

The stat reads 2 chains to start on.
[`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md)
ships `testnet`, its sunset predecessor, and `local`. The predecessor is
labeled sunset, so reading "to start on" as excluding it is defensible and the
stat may be deliberate; raised only because the two numbers sit one click
apart.

### Nit: site/style.css:61-87 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L61-L87)

Complete `:root[data-theme="light"]` and `:root[data-theme="dark"]` palettes.
Nothing in the HTML or the JS sets `data-theme`, and there is no toggle, so the
OS preference cannot be overridden.

### Nit: site/app.js:1 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L1)

No enclosing function. The leading two-space indent is left over from an inline
`<script>`, and `saEvent` and `io` land on `window`.

### Nit: site/app.js:15 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L15)

`flash` swaps `btn.textContent` on an element with no live region, so a screen
reader user gets no copy confirmation.

### Nit: site/app.js:27 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L27)

`flash("Press ⌘C")` shows a Mac-only shortcut. That branch runs on non-secure
contexts and older browsers, which skew away from macOS.

### Nit: site/index.html:11 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L11)

Neither self-hosted woff2 is preloaded, so both are discovered only after
`style.css` parses and the hero takes a round trip of swapped text on first
visit.

### Nit: site/index.html:38 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38)

`id="install-cmd"` is referenced nowhere.
[`site/app.js:21`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L21)
resolves the target through `btn.parentElement.querySelector("code")`.

### Nit: site/index.html:280 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L280)

The `noscript` pixel sets `referrerpolicy="no-referrer-when-downgrade"`, which
sends the full URL to `queue.simpleanalyticscdn.com`. Every other request is
covered by `strict-origin-when-cross-origin` from
[`netlify.toml:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/netlify.toml#L15),
so the attribute is the one place the page opts out of its own header.

### Nit: site/index.html:279 [↗](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L279)

`scripts.simpleanalyticscdn.com/latest.js` runs with no `integrity` and no
`crossorigin`, and the CSP admits the whole host. A rolling `latest.js` cannot
be pinned by hash, so the choice is a versioned URL or an accepted risk stated
as such — worth naming, since the CSP is otherwise `default-src 'none'`.

### Nit: repository metadata

`homepage` and `description` on [gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp)
are both null, so GitHub does not link back to the site.

### Nit: 404 handling

An unknown path returns Netlify's default page.
