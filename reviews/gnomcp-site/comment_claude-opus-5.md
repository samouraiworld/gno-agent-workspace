# Site review: mcp.gno.dev

Target: [gnoverse/gno-mcp#69](https://github.com/gnoverse/gno-mcp/issues/69)
Event: ISSUE
Pinned: `ef19be4`
Reverified: 2026-08-03, `ef19be4` still the tip of `main`
Posted: 2026-08-03, [gnoverse/gno-mcp#69](https://github.com/gnoverse/gno-mcp/issues/69). A manual pass by the user follows.

The body below is deliberately unwrapped: one line per paragraph, no 80-column limit. GitHub reflows it, and the raw markdown stays readable in the issue editor.

## Title
Site review: mcp.gno.dev (1 Critical, 7 Warnings, 14 Nits)

## Body
> Written by an agent. Every finding was re-derived against `ef19be4` before filing. No human has checked it yet.

Review of [mcp.gno.dev](https://mcp.gno.dev/) against [`site/`](https://github.com/gnoverse/gno-mcp/tree/ef19be4/site), [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md), [`docs/tools.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/tools.md), and [`scripts/install.sh`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh). Line numbers are pinned at `ef19be4`. The deployed HTML and CSS are byte-identical to `site/`, so every one of them resolves in the repo.

Clean on the checks that matter most: the 26 tool names in the strip match `docs/tools.md` exactly, the realm paths in the example prompts exist in `examples/`, all 12 external URLs resolve, the heading outline has no skips, cold load is ~71 KB, and the CSP is `default-src 'none'` with no `unsafe-inline`. Dark mode passes all 20 contrast pairs measured.

[Full review, with the evidence behind every line below.](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnomcp-site/review_claude-opus-5.md)

## Critical

### [`site/index.html:212`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L212) — the security section states an absolute the docs decline to state

> Everything read from the chain is wrapped as untrusted content, so a malicious realm can't steer the agent.

[`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md) §4 names three channels the envelope does not cover. `gno_read` with `full=true` or `symbols` is deliberately not wrapped, and §4 says that path "relies on the client honoring the resource boundary". Error text is mixed-trust: neutralized, not enveloped. `structuredContent` "carries raw values", and §4 tells clients that surface those fields to a model to apply their own marking. §4 then instructs the reader to treat anything from any channel as data, which is an instruction that exists because the envelope alone does not finish the job.

The heading directly above reads "Safe by architecture, not by promise", and the other five invariants hold against §1, §3, §5, §6, and §7. This is the one claim a skeptical reader can use to discount the rest. The fix is to state the mechanism rather than the outcome: wrapping plus envelope-tag neutralization is a strong claim on its own, and it is true.

## Warnings

### [`site/index.html:46-48`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L46-L48) — two advertised clients have no install path

The "Works with" row names Claude Desktop and Cursor. Neither appears again anywhere on the page. [`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L52-L57) accepts `claude|gemini|codex|opencode` plus the `none` opt-out, and [`site/index.html:236`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L236) says so accurately. A Cursor user is recruited by the hero, runs the one command, and receives a binary and silence. [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md) carries a Cursor section and an "Other MCP clients" section, and the page links to neither.

### [`site/style.css:108-109`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L108-L109) — the focus ring is invisible in light mode

`outline: 2px solid var(--mint)` measures 1.95:1 on `--bg` and 1.79:1 on `--surface`. WCAG 2.2 SC 1.4.11 requires 3:1. Every focusable element on the page is affected in light mode: the section nav, both hero buttons, the copy button, every documentation link, the footer. The same ring measures 9.94:1 on the dark background, so the failure is confined to the light theme. [`--accent-ink`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L32) is already defined at `#3b7a64` in the live `:root` and reaches 5.06:1 on white.

### [`site/style.css:18-19`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L18-L19) — four light-mode text contrast failures

Four pairs fall under 4.5:1 at sizes with no large-text exemption.

| Element | Token | Size | Ratio |
|:--------|:------|-----:|------:|
| `.tool-strip-note`, the link to `docs/tools.md` | `--fg-faint` | 13px | 3.74:1 |
| `.eyebrow`, once per section | `--label` | 11px bold | 4.38:1 |
| `.tool-row .needs`, five instances | `--label` | 11px bold | 4.38:1 |
| `.step-n` | `--label` | 12px bold | 4.38:1 |

`#0a0a0a99` clears 4.5:1 for both tokens.

### [`site/index.html:235`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L235) — the primary call to action lands on a command that cannot be copied

"Get started" scrolls to `#install`, where step 01 repeats the curl command as `.mini-cmd` with no copy button. Only the hero copy at [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38) has one, so the deliberate scroll-to-convert path ends on the worse version of the thing the visitor came for.

### [`site/index.html:3-12`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L3-L12) — no Open Graph or Twitter Card tags

No `og:*`, no `twitter:*`, no canonical. The footer links X, Discord, and Telegram, so the page is built to be shared into the three surfaces that unfurl link previews and will unfurl as a bare URL in all three. `robots.txt` and `sitemap.xml` both return 404.

### [`site/index.html:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L15) — no way to bypass the header, WCAG 2.4.1

No skip link. A keyboard or screen-reader user crosses the wordmark, five section links, and the GitHub link before reaching the `h1`. `main` already carries `id="top"` at [`site/index.html:29`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L29), so one anchor at the top of `body` closes it. SC 2.4.1 is Level A, the same tier as the focus-ring failure above.

### [`site/style.css:158`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L158) — mobile loses all section navigation

`.site-nav { display: none }` under 860px with no replacement. On a five-section page the mobile header keeps the wordmark and the GitHub link, and navigation becomes scrolling. Desktop has no scroll-spy or `aria-current` either.

## Nits

### [`site/index.html:140`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L140) — the session example demos the opposite of the guidance beside it

The showcased prompt sets a two-week expiry. Ten lines down, [`site/index.html:150`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L150) says keep scopes tight and spend limits low, dropping the third item from `docs/security.md` §2, which is to keep `expires_in` short. The panel labels the feature Pre-release.

### [`site/index.html:229`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L229) — checksum verification is stated unconditionally

"It verifies the checksum" carries no condition. [`scripts/install.sh:97-99`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L97-L99) warns and continues when neither `sha256sum` nor `shasum` is present.

### [`site/index.html:187`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L187) — two "needs" chips describe one tool out of the row

"Connect and configure" is chipped "A gnoweb URL", but the row opens on `gno_profile_list`, which `docs/tools.md` describes as plain config, never dialed. Same shape at [`site/index.html:192`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L192): "Your signature for sessions" chips a row where six of nine tools are key management.

### [`site/index.html:85`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L85) — the chain count is 2 on the page and 3 in the docs

The stat reads 2 chains to start on. [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md) ships `testnet`, its sunset predecessor, and `local`. The predecessor is labeled sunset, so reading "to start on" as excluding it is defensible and the stat may be deliberate; raised only because the two numbers sit one click apart.

### [`site/style.css:61-88`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L61-L88) — dead theme system

Complete `:root[data-theme="light"]` and `:root[data-theme="dark"]` palettes. Nothing in the HTML or the JS sets `data-theme`, and there is no toggle, so the OS preference cannot be overridden. What ships instead is the live [`:root`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L14) plus the [`prefers-color-scheme: dark`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L44) override, which duplicates these two blocks.

### [`site/app.js:1`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L1) — no enclosing function

The leading two-space indent is left over from an inline `<script>`, and `saEvent` and `io` land on `window`.

### [`site/app.js:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L15) — copy confirmation is silent to screen readers

`flash` swaps `btn.textContent` on an element with no live region.

### [`site/app.js:27`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L27) — the clipboard fallback shows a Mac-only shortcut

`flash("Press ⌘C")`. That branch runs on non-secure contexts and older browsers, which skew away from macOS.

### [`site/index.html:11`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L11) — fonts are not preloaded

Neither self-hosted woff2 is preloaded, so both are discovered only after `style.css` parses and the hero takes a round trip of swapped text on first visit.

### [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38) — dead id

`id="install-cmd"` is referenced nowhere. [`site/app.js:21`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L21) resolves the target through `btn.parentElement.querySelector("code")`.

### [`site/index.html:280`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L280) — the analytics pixel loosens the site's own referrer policy

The `noscript` pixel sets `referrerpolicy="no-referrer-when-downgrade"`, which sends the full URL to `queue.simpleanalyticscdn.com`. Every other request is covered by `strict-origin-when-cross-origin` from [`netlify.toml:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/netlify.toml#L15), so the attribute is the one place the page opts out of its own header.

### [`site/index.html:279`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L279) — the third-party script carries no integrity

`scripts.simpleanalyticscdn.com/latest.js` runs with no `integrity` and no `crossorigin`, and the CSP admits the whole host. A rolling `latest.js` cannot be pinned by hash, so the choice is a versioned URL or an accepted risk stated as such — worth naming, since the CSP is otherwise `default-src 'none'`.

### Repository metadata

`homepage` and `description` on [gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp) are both empty, so GitHub does not link back to the site.

### 404 handling

An unknown path returns Netlify's default page.
