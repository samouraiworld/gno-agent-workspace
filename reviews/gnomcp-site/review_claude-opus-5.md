# Site review: [mcp.gno.dev](https://mcp.gno.dev/)

URL: https://mcp.gno.dev/
Source: [`site/`](https://github.com/gnoverse/gno-mcp/tree/ef19be4/site) in
[gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp) | Host: Netlify, no
build step
Reviewed by: davd-gzl | Model: claude-opus-5 | Fetched: 2026-07-30
Rechecked: 2026-08-01 against `ef19be4`, still the tip of `main` and unchanged
under `site/`, `docs/`, and `scripts/`. Every measurement below was re-run.

Line numbers cite
[`site/index.html`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html)
and
[`site/style.css`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css).
Both were byte-compared against the deployed files: identical, so every line
number below resolves in the repo.

**TL;DR:** The one-page promo site for gnomcp, the MCP server that lets an AI
agent read, audit, and write on gno.land. It pitches the server plus the gno
skill, lists the 26 tools, states six security invariants, and ends on the
one-line installer.

**Verdict: REQUEST CHANGES** — the security section states an absolute the
project's own [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md)
does not support, and two of the six advertised clients have no install path
anywhere on the page (1 Critical, 7 Warnings, 14 Nits).

## Verify first

- [`site/index.html:212`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L212)
  against [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md)
  §4: read the four bullets under the untrusted-content envelope and confirm
  whether "Everything read from the chain is wrapped" holds for `gno_read`
  with `full=true`, for error text, and for `structuredContent`.
- [`site/index.html:46`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L46)
  against [`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L52-L57):
  the harness allowlist is `claude|gemini|codex|opencode|none`, where `none`
  is the opt-out sentinel, not a client. Confirm what a Cursor user is meant
  to run after the curl command.
- [`site/style.css:108-109`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L108-L109)
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
| Every external link | 9 anchor targets and 3 third-party asset URLs, 12 of 12 resolve 200 |
| Deployed HTML and CSS vs `site/` | byte-identical |
| WCAG 2.1 contrast, 20 foreground/background pairs, both themes | dark 20/20, light 16/20 |
| Focus indicator contrast, WCAG 2.2 SC 1.4.11 | dark passes, light fails |
| Heading outline | one `h1`, no skipped levels |
| Cold-load transfer weight | 71.1 KB: 11.3 KB brotli over HTML, CSS, JS plus 59.8 KB of woff2 |
| Response headers vs `netlify.toml` | CSP, HSTS, nosniff, frame-deny all served |

## What holds

The Content-Security-Policy is `default-src 'none'` with no `unsafe-inline` on
`style-src`, plus `frame-ancestors`, `base-uri`, and `form-action` all set to
`'none'` and a per-host allowlist for Simple Analytics. Motion handling is
correct in both directions: [`site/app.js`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js)
adds the `.js` class that arms the reveal-hiding CSS only after confirming both
`IntersectionObserver` support and no reduced-motion preference, so content is
never stranded invisible, and `scroll-behavior` resets to `auto` under reduced
motion. Fonts carry `font-display: swap` and a one-year immutable cache.

## Critical (must fix)

- **[unbackable absolute in the security section]** [`site/index.html:212`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L212)
  — "Everything read from the chain is wrapped as untrusted content, so a
  malicious realm can't steer the agent" states a guarantee that
  [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md)
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

- **[two advertised clients have no install path]** [`site/index.html:46`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L46)
  — the "Works with" row names Claude Desktop and Cursor. Neither appears
  again anywhere on the page.
  <details><summary>details</summary>

  [`scripts/install.sh:52-57`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L52-L57)
  accepts `claude|gemini|codex|opencode`, plus `none` to wire nothing, and
  [`site/index.html:236`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L236)
  says so accurately: Claude Code, Gemini CLI, and Codex are wired
  automatically, OpenCode gets printed steps. A Cursor user is recruited by
  the hero, runs the one command, and receives a binary and silence.

  [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md)
  already carries a Cursor section and an "Other MCP clients" section. The
  page never links to either. The only route to those docs is the "Read the
  docs" button in the last block of the page.

  Fix: link the client row to `docs/gnomcp.md#install`.
  </details>

- **[focus ring is invisible in light mode]** [`site/style.css:108-109`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L108-L109)
  — `outline: 2px solid var(--mint)` measures 1.95:1 on `--bg` and 1.79:1 on
  `--surface`. WCAG 2.2 SC 1.4.11 requires 3:1.
  <details><summary>details</summary>

  This affects every focusable element on the page: the section nav, both hero
  buttons, the copy button, every documentation link, the footer. The same
  ring measures 9.94:1 against the dark background, so the failure is confined
  to the light theme.

  Fix: `--accent-ink` is already defined as `#3b7a64` in the live
  [`:root`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L32)
  and reaches 5.06:1 on white.
  </details>

- **[four light-mode text contrast failures]** [`site/style.css:18-19`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L18-L19)
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

- **[the primary call to action lands on a command that cannot be copied]** [`site/index.html:235`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L235)
  — "Get started" scrolls to `#install`, where step 01 repeats the curl
  command as `.mini-cmd` with no copy button. Only the hero copy at
  [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38)
  has one.

- **[no Open Graph or Twitter Card tags]** [`site/index.html:3-12`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L3-L12)
  — the head carries `description`, `viewport`, `theme-color`, and a favicon.
  There is no `og:*`, no `twitter:*`, and no canonical.
  <details><summary>details</summary>

  The footer links X, Discord, and Telegram. The page is built to be shared
  into the three surfaces that unfurl link previews, and it will unfurl as a
  bare URL in all three. `robots.txt` and `sitemap.xml` both return 404.
  </details>

- **[no way to bypass the header, WCAG 2.4.1]** [`site/index.html:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L15)
  — the page has no skip link. `grep -ci skip site/index.html` returns 0.
  <details><summary>details</summary>

  The header carries the wordmark, five section links, and the GitHub link, so
  a keyboard or screen-reader user crosses seven controls before the `h1`.
  `main` already has `id="top"` at
  [`site/index.html:29`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L29),
  so the fix is one anchor at the top of `body` pointing at it. SC 2.4.1 is
  Level A, the same tier as the focus-ring failure above.
  </details>

- **[mobile loses all section navigation]** [`site/style.css:158`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L158)
  — `@media (max-width: 860px) { .site-nav { display: none } }` with no
  replacement. On a five-section page the mobile header keeps the wordmark and
  the GitHub link, and navigation becomes scrolling. Desktop has no scroll-spy
  or `aria-current` either.

## Nits

- **[the session example demos the opposite of the guidance beside it]** [`site/index.html:140`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L140)
  — the showcased prompt sets a two-week expiry. Ten lines down,
  [`site/index.html:150`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L150)
  says keep scopes tight and spend limits low, dropping the third item from
  [`docs/security.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/security.md)
  §2, which is to keep `expires_in` short. The feature is labeled Pre-release
  in the same panel.
- **[checksum verification is stated unconditionally]** [`site/index.html:229`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L229)
  — "It verifies the checksum".
  [`scripts/install.sh:97-99`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L97-L99)
  warns and continues when neither `sha256sum` nor `shasum` is present.
- **[two "needs" chips describe one tool out of the row]** [`site/index.html:187`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L187)
  — "Connect and configure" is chipped "A gnoweb URL", but the row opens on
  `gno_profile_list`, which
  [`docs/tools.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/tools.md)
  describes as plain config, never dialed. Same shape at
  [`site/index.html:192`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L192):
  "Your signature for sessions" chips a row where six of nine tools are key
  management.
- **[the chain count is 2 on the page and 3 in the docs]** [`site/index.html:85`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L85)
  — [`docs/gnomcp.md:128`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md#L128)
  ships `testnet`, its sunset predecessor, and `local`. The weakest finding in
  this review: the third profile is labeled sunset, so reading "chains to start
  on" as excluding it is defensible, and the stat may be deliberate. Raised
  only because the two numbers are one click apart.
- **[dead theme system]** [`site/style.css:61-88`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L61-L88)
  — complete `:root[data-theme="light"]` and `:root[data-theme="dark"]`
  palettes. Nothing in the HTML or the JS sets `data-theme`, and there is no
  toggle, so the OS preference cannot be overridden. What ships instead is the
  live [`:root`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L14)
  plus the [`prefers-color-scheme: dark`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L44)
  override, which duplicates these two blocks.
- **[`app.js` has no enclosing function]** [`site/app.js:1`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L1)
  — the leading two-space indent is left over from an inline `<script>`.
  `saEvent` and `io` land on `window`.
- **[copy confirmation is silent to screen readers]** [`site/app.js:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L15)
  — `flash` swaps `btn.textContent` on an element with no live region.
- **[the clipboard fallback shows a Mac-only shortcut]** [`site/app.js:27`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L27)
  — `flash("Press ⌘C")`. That branch runs on non-secure contexts and older
  browsers, which skew away from macOS.
- **[fonts are not preloaded]** [`site/index.html:11`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L11)
  — both self-hosted woff2 files are discovered only after `style.css` parses,
  so the hero takes a round trip of swapped text on first visit.
- **[dead id]** [`site/index.html:38`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L38)
  — `id="install-cmd"` is referenced nowhere.
  [`site/app.js:21`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/app.js#L21)
  resolves the target through `btn.parentElement.querySelector("code")`.
- **[the analytics pixel loosens the site's own referrer policy]** [`site/index.html:280`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L280)
  — the `noscript` pixel sets `referrerpolicy="no-referrer-when-downgrade"`,
  which sends the full URL to `queue.simpleanalyticscdn.com`. The
  `Referrer-Policy` header the site serves for every other request is
  `strict-origin-when-cross-origin`, set at
  [`netlify.toml:15`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/netlify.toml#L15).
  The attribute is the one place the page opts out of its own header.
- **[the third-party script carries no integrity]** [`site/index.html:279`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/index.html#L279)
  — `scripts.simpleanalyticscdn.com/latest.js` runs with no `integrity` and no
  `crossorigin`, and the CSP admits the whole host. A rolling `latest.js`
  cannot be pinned by hash, so the fix is a versioned URL or nothing; worth
  stating as an accepted risk rather than leaving unremarked, since the CSP is
  otherwise `default-src 'none'`.
- **[the repo does not link back to the site]** — `homepage` on
  [gnoverse/gno-mcp](https://github.com/gnoverse/gno-mcp) is null, and so is
  `description`.
- **[no 404 page]** — an unknown path returns Netlify's default.

## Open questions

- Is the `data-theme` block the start of a toggle that was cut, or dead weight
  to delete?
- Claude Desktop appears in the "Works with" row and in
  [`README.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/README.md), but
  [`docs/gnomcp.md`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/docs/gnomcp.md)
  covers it only under "Other MCP clients". Is a named section wanted, or does
  the generic one carry it?

## Recheck, 2026-08-01

Every finding was re-derived from a fresh clone at `ef19be4` and from the live
deploy. All 19 original findings still stand, and the three deployed files are
still byte-identical to `site/`. Four numbers moved and three findings were
added:

| Was | Now | Why |
|:----|:----|:----|
| `style.css:108` | `108-109` | 108 is the `a:focus-visible, button:focus-visible` selector; the `outline` declaration is 109 |
| allowlist `claude\|gemini\|codex\|opencode` | `…\|none` | [`scripts/install.sh:54`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/scripts/install.sh#L54) also takes `none`, the opt-out sentinel. The finding is unchanged: no client value covers Cursor or Claude Desktop |
| 8 external links | 9 anchors, 12 URLs | the count missed one anchor and the three third-party assets. All resolve 200 |
| `style.css:61-87` | `61-88` | the `[data-theme="dark"]` block closes on 88 |

Added: the missing skip link (Warning), the analytics pixel's `referrerpolicy`
override, and the un-pinned third-party script.

Contrast arithmetic re-run from the hex values, all four confirmed to two
decimals: mint on white 1.95:1, on `--surface` 1.79:1, on the dark background
9.94:1; `--fg-faint` 3.74:1; `--label` 4.39:1; the proposed `#0a0a0a99` 5.25:1
and `--accent-ink` 5.06:1. Weight re-measured at 71.1 KB, matching the ~71 KB
originally reported.

## Recheck, 2026-08-03

Run before filing the issue. `ef19be4` is still the tip of `main`, and the three
deployed files are still byte-identical to `site/`. All 22 findings stand. Every
check below was re-run from a fresh clone and against the live deploy:

| Re-run | Result |
|:-------|:-------|
| 26 tool names in the strip vs `docs/tools.md` | `diff` of both sorted lists is empty |
| `r/gnoland/wugnot`, `r/gov/dao`, `r/demo/profile` in gnolang/gno `examples/` | all three resolve |
| 9 anchors, 2 third-party assets, and the `raw.githubusercontent.com` install URL the terminal block prints | 12 of 12 return 200 |
| `robots.txt`, `sitemap.xml`, unknown path | 404, 404, 404 |
| CSP, HSTS, nosniff, frame-deny, `Referrer-Policy` | all five served |
| `homepage` and `description` on the repo | both empty |
| `grep -ci skip`, `og:`/`twitter:`/canonical, `rel="preload"` | 0, 0, 0 |
| `.copy-btn` and `.mini-cmd` occurrences | 1 each, so the install step has no copy button |
| `id="install-cmd"` references | one, its own definition |
| Four contrast ratios recomputed from the hex values | 3.74, 4.38, 1.95, 1.79, 9.94, 5.06, 5.25 all confirmed |
| Font sizes behind the contrast table | `.8125rem`, `.6875rem` bold, `.6875rem` bold, `.75rem` bold — none large-text exempt |
| `docs/security.md` §4 | three uncovered channels still named; the Critical stands verbatim |

Three anchors moved:

| Was | Now | Why |
|:----|:----|:----|
| `index.html:84` | `85` | 84 is the `hero-stats` container; the chain stat is 85 |
| `index.html:46` | `46-48` | 46 opens `.client-row`; the client names are on 48 |
| `style.css:61-87` | `61-88` | the `[data-theme="dark"]` block closes on 88 |

One claim was corrected rather than moved: `--accent-ink` was described as
living "in the light block", meaning the dead `:root[data-theme="light"]`. It is
in the live [`:root`](https://github.com/gnoverse/gno-mcp/blob/ef19be4/site/style.css#L32)
as well, which is what makes the fix a one-token swap. The dark-ring ratio was
9.92 in the body and 9.94 in the first recheck; 9.94 is correct.

## Disclosure note

Nothing here is exploitable against deployed code. The Critical finding is a
gap between the page copy and the project's own published `docs/security.md`,
which already documents the residual risk in full. No new vector is described.
