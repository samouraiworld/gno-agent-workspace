# Claims: #6327, round 2 at da4a1dfa0

A single-finding pass, run by hand from a security read of `latest_blogposts`; no finders, judges or writer stage ran, so this file holds the one row and its runs.

| Line | # | Band | State | Base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`gnoalias.sh:37`](https://github.com/gnolang/gno/blob/da4a1dfa0805bf95acb56049579cd4e030dd126c/misc/deployments/home-alias/gnoalias.sh#L37) | 1 | Nit | CONFIRMED | new, the base script has no blog query | not posted |

## 1. A multi-line blog title reaches `home.md` outside the bold title

- Claim: `{ print }` at line 37 copies every line between the first `<gno-columns>` and `</gno-columns>` that is not a `### ` line, so a title holding newlines puts its later lines into `home.md` unchanged.
- Check, the realm: `r/gnoland/blog` `ModAddPost` as admin with the title `"Upgrade notice\n\n## Action required: move your GNOT\n[Migrate now](/r/attacker/migrate)"`, `gno test -v -run TestNewlineTitle` with `gno` built from da4a1dfa0 under go1.25.9, in a copy of `examples/`. It passed and rendered the title's lines verbatim: [`tests/render_newline_title.md`](tests/render_newline_title.md). `prepareAndSetPost` only runs `strings.TrimSpace` on the title, `blog.gno:176`.
- Check, the script: `latest_blogposts` extracted from the script at da4a1dfa0 with `qrender` stubbed to that render, under bash with gawk 5. The `## ` line and the link line came out on lines of their own. The whole block is [`tests/newline_title_repro.sh`](tests/newline_title_repro.sh), run verbatim in a fresh copy.
- Fix, run both ways under gawk: [`tests/fix.diff`](tests/fix.diff). On the injected render it drops the three title lines and keeps the tags, the first title line and the three dates. On the live mainnet index, `vm/qrender gno.land/r/gnoland/blog:` read 2026-11-19, [`tests/render_mainnet.md`](tests/render_mainnet.md), its output is identical to the unfixed script's. Not run under busybox awk, which the alpine image ships; the regex uses only POSIX ERE alternation and bracket expressions.
- Band: Nit. Only the blog admin, its moderators or a passed govDAO proposal write titles, and they can already show the same lines on the blog page; what the line changes is that they reach the homepage, against the script's own comment at lines 29-30.
- Not raised before: round 1's two script threads, `discussion_r4297641063` and `discussion_r4297641085`, cover the failed-query path and the interval.
