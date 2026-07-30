# Site review: [mcp.gno.dev](https://mcp.gno.dev/)

URL: https://mcp.gno.dev/
Source: [`site/`](https://github.com/gnoverse/gno-mcp/tree/main/site) in
[gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp) | Host: Netlify, no
build step
Reviewed by: davd-gzl | Model: claude-opus-5 | Fetched: 2026-07-30

Line numbers cite
[`site/index.html`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html)
and
[`site/style.css`](https://github.com/gnoverse/gno-mcp/blob/main/site/style.css).
Both were byte-compared against the deployed files: identical, so every line
number below resolves in the repo.

**TL;DR:** The one-page promo site for gnomcp, the MCP server that lets an AI
agent read, audit, and write on gno.land. It pitches the server plus the gno
skill, lists the 26 tools, states six security invariants, and ends on the
one-line installer.

**Verdict: REQUEST CHANGES** — the security section states an absolute the
project's own [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/security.md)
does not support, and two of the six advertised clients have no install path
anywhere on the page (1 Critical, 6 Warnings, 12 Nits).

## Verify first

- [`site/index.html:212`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L212)
  against [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/security.md)
  §4: read the four bullets under the untrusted-content envelope and confirm
  whether "Everything read from the chain is wrapped" holds for `gno_read`
  with `full=true`, for error text, and for `structuredContent`.
- [`site/index.html:46`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L46)
  against [`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/main/scripts/install.sh#L52-L57):
  the harness allowlist is `claude|gemini|codex|opencode`. Confirm what a
  Cursor user is meant to run after the curl command.
- [`site/style.css:108`](https://github.com/gnoverse/gno-mcp/blob/main/site/style.css#L108)
  in light mode: tab through the hero and watch the focus ring. Mint `#8ec5b2`
  on white measures 1.95:1.

## Summary

The page is well built. Content accuracy is high where it is checkable, the
infrastructure is close to exemplary, and the tone is more honest than the
category norm. Findings cluster in three places: one security claim stated as
an absolute, a conversion path that abandons two named clients, and light-mode
accessibility. Dark mode passes every contrast check run. Light mode fails
five.

## Checks run

| Check | Result |
|:------|:-------|
| 26 tool names in the strip vs every `gno_*` in `docs/tools.md` | identical, no drift |
| Five tool-row groupings vs the 26 names | sum to 26, no tool orphaned |
| Realm paths in the example prompts vs `examples/` in gnolang/gno | `r/gnoland/wugnot` and `r/gov/dao` both exist |
| Six security invariants vs `docs/security.md` §1, §3, §4, §5, §6, §7 | five hold, one overclaims |
| Every external link | 8 of 8 resolve 200 |
| Deployed HTML and CSS vs `site/` | byte-identical |
| WCAG 2.1 contrast, 20 foreground/background pairs, both themes | dark 20/20, light 16/20 |
| Focus indicator contrast, WCAG 2.2 SC 1.4.11 | dark passes, light fails |
| Heading outline | one `h1`, no skipped levels |
| Cold-load transfer weight | ~71 KB, brotli on HTML, CSS, JS |
| Response headers vs `netlify.toml` | CSP, HSTS, nosniff, frame-deny all served |

## What holds

The Content-Security-Policy is `default-src 'none'` with no `unsafe-inline` on
`style-src`, plus `frame-ancestors`, `base-uri`, and `form-action` all set to
`'none'` and a per-host allowlist for Simple Analytics. Motion handling is
correct in both directions: [`site/app.js`](https://github.com/gnoverse/gno-mcp/blob/main/site/app.js)
adds the `.js` class that arms the reveal-hiding CSS only after confirming both
`IntersectionObserver` support and no reduced-motion preference, so content is
never stranded invisible, and `scroll-behavior` resets to `auto` under reduced
motion. Fonts carry `font-display: swap` and a one-year immutable cache.

## Critical (must fix)

- **[unbackable absolute in the security section]** [`site/index.html:212`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L212)
  — "Everything read from the chain is wrapped as untrusted content, so a
  malicious realm can't steer the agent" states a guarantee that
  [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/security.md)
  §4 declines to state.
  <details><summary>details</summary>

  §4 names three channels the envelope does not cover. `gno_read` with
  `full=true` or `symbols` is deliberately not wrapped, because wrapping would
  corrupt the txtar archive, and §4 says the resource path "relies on the
  client honoring the resource boundary". Error text is called mixed-trust: it
  is neutralized but not enveloped. `structuredContent` "carries raw values",
  and §4 tells clients that surface those fields to a model to apply their own
  marking. §4 closes by telling the reader to treat anything from any channel
  as data, which is an instruction that exists because the envelope alone does
  not finish the job.

  The residual risk is real and lives at the client boundary. "Everything" and
  "can't" convert a layered defense into a promise. The heading directly above
  reads "Safe by architecture, not by promise", and the other five invariants
  check out cleanly against §1, §3, §5, §6, and §7, so this one claim is what
  a skeptical reader will use to discount the rest.

  Fix: state the mechanism rather than the outcome. Wrapping plus envelope-tag
  neutralization is a strong claim on its own and it is true.
  </details>

## Warnings (should fix)

- **[two advertised clients have no install path]** [`site/index.html:46`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L46)
  — the "Works with" row names Claude Desktop and Cursor. Neither appears
  again anywhere on the page.
  <details><summary>details</summary>

  [`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/main/scripts/install.sh#L52-L57)
  accepts `claude|gemini|codex|opencode` and nothing else, and
  [`site/index.html:236`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L236)
  says so accurately: Claude Code, Gemini CLI, and Codex are wired
  automatically, OpenCode gets printed steps. A Cursor user is recruited by
  the hero, runs the one command, and receives a binary and silence.

  [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/gnomcp.md)
  already carries a Cursor section and an "Other MCP clients" section. The
  page never links to either. The only route to those docs is the "Read the
  docs" button in the last block of the page.

  Fix: link the client row to `docs/gnomcp.md#install`.
  </details>

- **[focus ring is invisible in light mode]** [`site/style.css:108`](https://github.com/gnoverse/gno-mcp/blob/main/site/style.css#L108)
  — `outline: 2px solid var(--mint)` measures 1.95:1 on `--bg` and 1.79:1 on
  `--surface`. WCAG 2.2 SC 1.4.11 requires 3:1.
  <details><summary>details</summary>

  This affects every focusable element on the page: the section nav, both hero
  buttons, the copy button, every documentation link, the footer. The same
  ring measures 9.92:1 against the dark background, so the failure is confined
  to the light theme.

  Fix: `--accent-ink` is already defined as `#3b7a64` in the light block and
  reaches 5.06:1 on white.
  </details>

- **[four light-mode text contrast failures]** [`site/style.css:18-19`](https://github.com/gnoverse/gno-mcp/blob/main/site/style.css#L18-L19)
  — `--label` and `--fg-faint` fall under 4.5:1 on white at sizes with no
  large-text exemption.
  <details><summary>details</summary>

  | Element | Token | Size | Ratio |
  |:--------|:------|-----:|------:|
  | `.tool-strip-note`, the link to `docs/tools.md` | `--fg-faint` | 13px | 3.74:1 |
  | `.eyebrow`, once per section | `--label` | 11px | 4.38:1 |
  | `.tool-row .needs`, five instances | `--label` | 11px | 4.38:1 |
  | `.step-n` | `--label` | 12px bold | 4.38:1 |

  The other 16 pairs measured pass in light mode, several above 8:1. Dark mode
  passes all 20.

  Fix: raise the alpha on both tokens in the light block. `#0a0a0a99` clears
  4.5:1 for both.
  </details>

- **[the primary call to action lands on a command that cannot be copied]** [`site/index.html:235`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L235)
  — "Get started" scrolls to `#install`, where step 01 repeats the curl
  command as `.mini-cmd` with no copy button. Only the hero copy at
  [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L38)
  has one.

- **[no Open Graph or Twitter Card tags]** [`site/index.html:3-12`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L3-L12)
  — the head carries `description`, `viewport`, `theme-color`, and a favicon.
  There is no `og:*`, no `twitter:*`, and no canonical.
  <details><summary>details</summary>

  The footer links X, Discord, and Telegram. The page is built to be shared
  into the three surfaces that unfurl link previews, and it will unfurl as a
  bare URL in all three. `robots.txt` and `sitemap.xml` both return 404.
  </details>

- **[mobile loses all section navigation]** [`site/style.css:158`](https://github.com/gnoverse/gno-mcp/blob/main/site/style.css#L158)
  — `@media (max-width: 860px) { .site-nav { display: none } }` with no
  replacement. On a five-section page the mobile header keeps the wordmark and
  the GitHub link, and navigation becomes scrolling. Desktop has no scroll-spy
  or `aria-current` either.

## Nits

- **[the session example demos the opposite of the guidance beside it]** [`site/index.html:140`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L140)
  — the showcased prompt sets a two-week expiry. Ten lines down,
  [`site/index.html:150`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L150)
  says keep scopes tight and spend limits low, dropping the third item from
  [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/security.md)
  §2, which is to keep `expires_in` short. The feature is labeled Pre-release
  in the same panel.
- **[checksum verification is stated unconditionally]** [`site/index.html:229`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L229)
  — "It verifies the checksum".
  [`scripts/install.sh:97-99`](https://github.com/gnoverse/gno-mcp/blob/main/scripts/install.sh#L97-L99)
  warns and continues when neither `sha256sum` nor `shasum` is present.
- **[two "needs" chips describe one tool out of the row]** [`site/index.html:187`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L187)
  — "Connect and configure" is chipped "A gnoweb URL", but the row opens on
  `gno_profile_list`, which
  [`docs/tools.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/tools.md)
  describes as plain config, never dialed. Same shape at
  [`site/index.html:192`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L192):
  "Your signature for sessions" chips a row where six of nine tools are key
  management.
- **[the chain count is 2 on the page and 3 in the docs]** [`site/index.html:84`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L84)
  — [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/gnomcp.md)
  ships `testnet`, its sunset predecessor, and `local`.
- **[dead theme system]** [`site/style.css:61-87`](https://github.com/gnoverse/gno-mcp/blob/main/site/style.css#L61-L87)
  — complete `:root[data-theme="light"]` and `:root[data-theme="dark"]`
  palettes. Nothing in the HTML or the JS sets `data-theme`, and there is no
  toggle, so the OS preference cannot be overridden.
- **[`app.js` has no enclosing function]** [`site/app.js:1`](https://github.com/gnoverse/gno-mcp/blob/main/site/app.js#L1)
  — the leading two-space indent is left over from an inline `<script>`.
  `saEvent` and `io` land on `window`.
- **[copy confirmation is silent to screen readers]** [`site/app.js:15`](https://github.com/gnoverse/gno-mcp/blob/main/site/app.js#L15)
  — `flash` swaps `btn.textContent` on an element with no live region.
- **[the clipboard fallback shows a Mac-only shortcut]** [`site/app.js:27`](https://github.com/gnoverse/gno-mcp/blob/main/site/app.js#L27)
  — `flash("Press ⌘C")`. That branch runs on non-secure contexts and older
  browsers, which skew away from macOS.
- **[fonts are not preloaded]** [`site/index.html:11`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L11)
  — both self-hosted woff2 files are discovered only after `style.css` parses,
  so the hero takes a round trip of swapped text on first visit.
- **[dead id]** [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/main/site/index.html#L38)
  — `id="install-cmd"` is referenced nowhere.
  [`site/app.js:21`](https://github.com/gnoverse/gno-mcp/blob/main/site/app.js#L21)
  resolves the target through `btn.parentElement.querySelector("code")`.
- **[the repo does not link back to the site]** — `homepage` on
  [gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp) is null, and so is
  `description`.
- **[no 404 page]** — an unknown path returns Netlify's default.

## Open questions

- Is the `data-theme` block the start of a toggle that was cut, or dead weight
  to delete?
- Claude Desktop appears in the "Works with" row and in
  [`README.md`](https://github.com/gnoverse/gno-mcp/blob/main/README.md), but
  [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/main/docs/gnomcp.md)
  covers it only under "Other MCP clients". Is a named section wanted, or does
  the generic one carry it?

## Disclosure note

Nothing here is exploitable against deployed code. The Critical finding is a
gap between the page copy and the project's own published `docs/security.md`,
which already documents the residual risk in full. No new vector is described.
