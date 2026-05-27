# PR #5714: feat(markdown): chain/markdown stdlib + p/nt/markdown/sanitize/v0 + safe p/moul/md helpers

URL: https://github.com/gnolang/gno/pull/5714
Author: jaekwon | Base: master | Files: 230 | +5572 -1684
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `84b818eb` (stale — +13 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5714 84b818eb`

**Verdict: REQUEST CHANGES** — split the PR (8 unrelated commits, including a consensus-breaking apphash change), fix CI (gofmt + broken doc link), close the multi-line LRD strip bypass.

## Summary

The stated PR ships a three-layer markdown sanitization stack: 8 byte-level Go natives under `chain/markdown` (StripBidi, NormalizeBreaks, EscapeInline/Title, PercentEncodeURL, MatchCharsetN, CodeFence, EscapeBlockHazards), 21 slot-targeted gno helpers under `p/nt/markdown/sanitize/v0` (Block, InlineText, URL, ImageURL, UserName, BechString, CodeBlock, etc.), and a `p/moul/md` integration that internally sanitizes `Link`/`UserLink`/`Image`/`InlineCode`/`CodeBlock`/`Blockquote`/`FootnoteDefinition`/`LinkReferenceDefinition`. Plus 103 golden fixtures wired through goldmark, native_gas calibration for the 8 natives, and a Python fitter.

Two structural problems sit on top of solid sanitization work. First, the PR bundles 8 unrelated commits (commits `47e355f`…`b54b178`) — type-driven PkgID stamping for `*StructValue`, a `.seal` marker on `realm`, a lint preprocess-panic fix, plus ADR doc renames. One of those (`47e355f`) changes [`apphash_crossrealm38_test.go`'s expected hash](https://github.com/gnolang/gno/blob/84b818eb/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52) · [↗](../../../../../.worktrees/gno-review-5714/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52) — a consensus break shipped under a "markdown safety" title. Second, the multi-line LRD case bypasses `sanitize.Block`'s strip: `[foo]:\nhttp://attacker\n` survives intact and lets user content define link targets that realm-side `[foo]` shortcut references resolve against.

CI is red: gofmt/goimports on 4 files, plus a broken doc link (`interrealm-v2.md` vs the actual `interrealm_v2.md`).

```
sanitize.Block input:                EscapeBlockHazards output:
┌─────────────────────────┐          ┌─────────────────────────┐
│ [foo]:                  │          │ [foo]:                  │  ← line 1: not LRD
│ http://attacker         │  ─────►  │ http://attacker         │     (no nonblank after `]:`)
│                         │          │                         │
│ realm-side [foo] resolves to attacker URL after concat       │
└─────────────────────────┘          └─────────────────────────┘
```

## Glossary

- `LRD` — CommonMark Link Reference Definition (`[label]: url "title"`), CM §4.7. Other inline content references it via `[text][label]` or shortcut `[label]`.
- `EscapeBlockHazards` — single-pass line scanner in `chain/markdown` that strips LRD definitions, escapes line-leading block markers, escapes `][`/`[^`, folds Unicode separators, auto-closes open fences.
- `isLRDDefinition` — internal predicate inside EscapeBlockHazards that decides whether a line is an LRD opener (and therefore gets stripped).
- `apphash` — deterministic state-hash that all nodes must agree on; a change is a consensus break.

## Fix

`sanitize.Block` chains `NormalizeBreaks` → `StripBidiAndZeroWidth` → `EscapeBlockHazards`. The block-hazards pass walks lines and strips ones detected as LRDs; it escapes line-leading block markers (`#`, `>`, `-`, `*`, `+`, ordered-list digits, thematic breaks) and setext underlines, escapes `][` (ref link use) and `[^` (footnote ref use), folds U+2028/U+2029/U+0085 to `\n`, treats gno-* extension delimiters specially, and auto-closes any open code fence at EOF. The realm-facing layer wraps each helper in a slot-targeted contract (`InlineText`, `Block`, `LinkTitle`, `TableCell`, `HTMLEscape`, `URL`, `ImageURL`, …) — see [`sanitize.gno`](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno).

## Critical (must fix)

- **[user-controlled LRDs survive multi-line shape]** [`gnovm/stdlibs/chain/markdown/markdown.go:348-380`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.go#L348-L380) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.go#L348-L380) — `isLRDDefinition` only matches single-line LRDs; CM §4.7 allows the URL on the next line, and goldmark resolves them.
  <details><summary>details</summary>

  `isLRDDefinition` returns false unless the URL begins on the same line as `[label]:` ([line 374-378](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.go#L374-L378) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.go#L374-L378): requires whitespace then a non-whitespace byte on the same line). CM §4.7 ("optional whitespace including up to one line ending") explicitly permits the URL on the next line, and the gnoweb goldmark chain parses that as a valid LRD. A user-supplied `[foo]:\nhttp://attacker\n` therefore passes through `sanitize.Block` unchanged and provides the `[foo]` link target for any realm-side `[foo]` shortcut reference on the same page.

  Verified end-to-end against gnoweb's goldmark chain — user-content LRD resolves a realm-side `[foo]` shortcut to `http://attacker.example`. Repro in `tests/multiline_lrd_bypass_test.go`. The inline reproduction is also runnable below.

  The package doc at [`sanitize.gno:141-153`](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L141-L153) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L141-L153) acknowledges shortcut-reference collision but frames it as "user content uses `[help]` to invoke realm-defined LRDs" — the inverse direction (user content *defines* an LRD that realm references resolve against) is not covered and is the actual bypass. The same file's threat-model section ([line 141-146](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L141-L146) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L141-L146)) claims "user content cannot define a reference that other parts of the page might resolve against" — that claim is currently false for multi-line LRDs.

  Goldmark mitigates the worst case (`javascript:` href gets rewritten to empty by the gnoweb extension chain), so this is not an XSS today; the attacker still gets to choose any `http(s)://` target the realm's `[foo]` references would otherwise resolve to.

  Fix: extend `isLRDDefinition` to also strip lines shaped `^ {0,3}\[[^\]]+\]:[ \t]*$` (label-only opener), or — more robustly — escape the opening `[` whenever a line matches that shape, so the next line's URL becomes a normal paragraph rather than an LRD continuation. Add a golden fixture pair (`block-lrd-multiline-strip.txtar` + the trailing-space variant) so the regression sticks.

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5714 -R gnolang/gno
  cat > /tmp/lrd_bypass.go <<'EOF'
  package main

  import (
      "bytes"
      "fmt"

      "github.com/gnolang/gno/gno.land/pkg/gnoweb/markdown"
      "github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
      cm "github.com/gnolang/gno/gnovm/stdlibs/chain/markdown"
      "github.com/yuin/goldmark"
      "github.com/yuin/goldmark/parser"
      "github.com/yuin/goldmark/text"
  )

  func main() {
      userIn := "[foo]:\nhttp://attacker.example\n"
      sanitized := cm.EscapeBlockHazards(userIn)
      fmt.Printf("sanitize.Block output unchanged: %v\n  %q\n", sanitized == userIn, sanitized)

      page := sanitized + "\nThe realm refers to [foo].\n"
      gnourl, _ := weburl.Parse("https://gno.land/r/test")
      ctxOpts := parser.WithContext(markdown.NewGnoParserContext(markdown.GnoContext{GnoURL: gnourl}))
      ext := markdown.NewGnoExtension()
      m := goldmark.New()
      ext.Extend(m)
      node := m.Parser().Parse(text.NewReader([]byte(page)), ctxOpts)
      var buf bytes.Buffer
      m.Renderer().Render(&buf, []byte(page), node)
      fmt.Println("---")
      fmt.Println(buf.String())
  }
  EOF
  go run /tmp/lrd_bypass.go
  # Expect (current bug): href="http://attacker.example" on the [foo] anchor.
  # Post-fix expectation: href="" (no LRD survives the strip).
  rm /tmp/lrd_bypass.go
  ```
  </details>

- **[scope creep — 8 unrelated commits, one consensus-breaking]** [`gnovm/pkg/gnolang/alloc.go`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/pkg/gnolang/alloc.go) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/pkg/gnolang/alloc.go), [`gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:52`](https://github.com/gnolang/gno/blob/84b818eb/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52) · [↗](../../../../../.worktrees/gno-review-5714/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L52) — markdown PR ships interrealm v2 PkgID stamping, `.seal` marker, lint fix, ADR docs.
  <details><summary>details</summary>

  Commits `47e355f`…`b54b178` (the first 8 of 15 in the branch) are interrealm v2 work bundled into a markdown-titled PR:

  - `47e355f` (`feat(gnovm): type-driven PkgID stamping for *StructValue at allocation`) — changes [`alloc.go`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/pkg/gnolang/alloc.go) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/pkg/gnolang/alloc.go), `op_exec.go`, `op_expressions.go`, `values.go`, rewrites 4 zrealm filetests, and updates the consensus apphash constant in `apphash_crossrealm38_test.go:52` (`77eeee…` → `26d3fb…`). Consensus break, no link to the markdown topic.
  - `69b1955` (`feat(interrealm): seal realm interface with dot-named marker method`) — adds a `.seal` native method to `uverse.go`.
  - `841374a` (`fix(gno/lint): swallow expected preprocess panic on filetests`) — gno/lint behavior change.
  - `34579ac`, `1e44d3f`, `0e2dda6`, `df0862f`, `b54b178` — interrealm ADR doc edits + the "Layer N" → "borrow rule #N" rename across 55 files.

  None of those belong in a PR titled and described as "leaf sanitization for markdown safety." A reviewer pulling this in for the markdown work has to either approve the apphash change blind or hold up sanitization while VM reviewers look at the realm/interrealm changes. Also: the rename commit changes Go test files referenced by docs links — the broken `interrealm-v2.md` link below is a direct consequence (see CI Critical below).

  Fix: rebase the branch onto master, drop the first 8 commits into separate PRs scoped to their actual topics (one for the VM `*StructValue` stamping + apphash, one for `realm` sealing, one for the lint fix, one for the ADR renames), and keep this PR's diff to the markdown layer + its native-gas / calibration / golden fixtures / downstream filetest updates. The markdown work is reviewable in isolation; the VM changes need their own dedicated review (incl. the apphash diff justification).
  </details>

- **[CI red: gofmt/goimports]** [`gnovm/stdlibs/chain/markdown/markdown.go:246`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.go#L246) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.go#L246), [`gnovm/stdlibs/chain/markdown/markdown_test.go:8`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown_test.go#L8) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown_test.go#L8), [`gnovm/stdlibs/native_gas.go:130`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/native_gas.go#L130) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/native_gas.go#L130), `gnovm/cmd/calibrate/markdown_bench_test.go:70` — 4 files fail `gofmt` + `goimports` in CI.
  <details><summary>details</summary>

  `main / lint` CI job [run 26386541013](https://github.com/gnolang/gno/actions/runs/26386541013/job/77666187446) reports 8 issues (4 gofmt, 4 goimports) on the same 4 files. The `style(p/moul/md,p/nt/markdown/sanitize/v0): apply gno fmt` commit at the tip applied gno-side formatting but missed the Go side. Visible in [markdown.go:246-250](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.go#L246-L250) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.go#L246-L250): `prevNonBlank  bool` (double-space alignment of struct-ish var block). Fix: `gofmt -w` over the four files.
  </details>

- **[CI red: broken docs link]** [`docs/resources/gno-interrealm-v2.md:8`](https://github.com/gnolang/gno/blob/84b818eb/docs/resources/gno-interrealm-v2.md#L8) · [↗](../../../../../.worktrees/gno-review-5714/docs/resources/gno-interrealm-v2.md#L8), [`docs/resources/gno-interrealm.md:5`](https://github.com/gnolang/gno/blob/84b818eb/docs/resources/gno-interrealm.md#L5) · [↗](../../../../../.worktrees/gno-review-5714/docs/resources/gno-interrealm.md#L5) — link points at `../../gnovm/adr/interrealm-v2.md` (hyphen), but the file is at `gnovm/adr/interrealm_v2.md` (underscore).
  <details><summary>details</summary>

  `docs` CI job [run 26386540919](https://github.com/gnolang/gno/actions/runs/26386540919/job/77666187118) fails the doc linker:

  ```
  Could not find files with the following paths:
  >>> ../../gnovm/adr/interrealm-v2.md (found in file: file:///home/runner/work/gno/gno/docs/resources/gno-interrealm.md)
  >>> ../../gnovm/adr/interrealm-v2.md (found in file: file:///home/runner/work/gno/gno/docs/resources/gno-interrealm-v2.md)
  ```

  Same file even references `gnovm/adr/interrealm_v2.md` (underscore — correct) two lines later, so it's a typo in the link target. Fix: change both occurrences to `interrealm_v2.md`. This is part of the scope-creep ADR-docs commits and should be fixed before they get rebased into their own PR.
  </details>

## Warnings (should fix)

- **[mailto body= check is case-sensitive]** [`examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:1076`](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1076) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1076) — `?Body=phish` and `?BODY=phish` bypass the `?body=` / `&body=` reject.
  <details><summary>details</summary>

  `linkSchemeAllowed` rejects mailto: prefill phishing only for literal lowercase `body=`. RFC 6068 §5 says mailto: header field names are case-insensitive in URI processing — mail clients honour `?Body=`, `?BODY=`, mixed case the same as `?body=`. Easy bypass:

  ```
  mailto:victim@x.com?Body=please+send+seed
  ```

  passes `URL()`, comes out percent-encoded but otherwise intact.

  Fix: lowercase the query slice before substring-matching, or use a small parser. Same case-folding would extend trivially to `subject=` / `cc=` / `bcc=` if the intent is to harden mailto: prefill more broadly (see related test-name mismatch nit below).
  </details>

- **[uppercase HTTP/HTTPS/MAILTO are rejected outright]** [`examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:1071-1079`](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1071-L1079) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1071-L1079) — schemes are case-insensitive per RFC 3986; `HTTPS://example.com` is rejected and the link href becomes empty.
  <details><summary>details</summary>

  `linkSchemeAllowed` only special-cases lowercase `http://`, `https://`, `mailto:`. An uppercase scheme falls through to `hasURLScheme(s)`, which accepts any letter case for the scheme detection — so `HTTPS://X` is detected as having a scheme that is *not* in the allowlist and gets rejected. RFC 3986 §3.1 explicitly mandates "scheme is case-insensitive". This is a false-positive rejection that will silently neutralize legitimate user-typed links. Confirmed via standalone repro.

  Fix: case-fold the scheme prefix check (`strings.EqualFold(s[:7], "http://")` etc.), or extract the scheme via `hasURLScheme`'s walk and compare it case-insensitively against the allowlist.
  </details>

- **[Claude Co-Authored-By trailers in commit messages]** 11 of 15 commits in the branch carry `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>`.
  <details><summary>details</summary>

  Per the global guidance the workspace runs on (and a common ask across gnolang reviewers): AI co-authorship trailers do not carry useful signal and are arguably misleading on legal grounds. Disclose AI usage in the PR description if needed; do not embed it in the git trailers — they propagate everywhere git uses commit metadata.

  Fix: rebase + strip the trailer (`git rebase -i --autosquash master`, or `git filter-branch` / `git filter-repo` if doing it in bulk).
  </details>

## Nits

- [`gnovm/stdlibs/chain/markdown/markdown.gno:7`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.gno#L7) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.gno#L7) — doc comment refers to `p/nt/md/sanitize`, but the actual path landed as `p/nt/markdown/sanitize/v0`. Same outdated reference in [line 38, 44, 80, 99](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.gno#L38) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.gno#L38).
- [`gno.land/pkg/gnoweb/markdown/golden/sanitize/url-mailto-cc-bcc-reject.txtar`](https://github.com/gnolang/gno/blob/84b818eb/gno.land/pkg/gnoweb/markdown/golden/sanitize/url-mailto-cc-bcc-reject.txtar) · [↗](../../../../../.worktrees/gno-review-5714/gno.land/pkg/gnoweb/markdown/golden/sanitize/url-mailto-cc-bcc-reject.txtar) — filename ends in `-reject` but the case actually *accepts* the URL (`output.md` echoes the input). Either rename to drop `-reject` or extend `linkSchemeAllowed` to actually reject `cc=`/`bcc=`/`subject=` if that was the original intent.
- [`gnovm/stdlibs/chain/markdown/markdown.go:436-494`](https://github.com/gnolang/gno/blob/84b818eb/gnovm/stdlibs/chain/markdown/markdown.go#L436-L494) · [↗](../../../../../.worktrees/gno-review-5714/gnovm/stdlibs/chain/markdown/markdown.go#L436-L494) — `escapeLineLeader` does not handle a bare `-` / `*` / `+` line (no trailing space) as a list-marker opener. CM treats these as empty list items; gnoweb's goldmark chain may or may not (depends on enabled extensions). Worth a golden fixture either way.

## Missing Tests

- **[multi-line LRD strip]** [`gno.land/pkg/gnoweb/markdown/golden/sanitize/`](https://github.com/gnolang/gno/blob/84b818eb/gno.land/pkg/gnoweb/markdown/golden/sanitize/) · [↗](../../../../../.worktrees/gno-review-5714/gno.land/pkg/gnoweb/markdown/golden/sanitize/) — no fixture exercises the `[label]:\nURL` shape.
  <details><summary>details</summary>

  The Critical above is silent for the reader who skims `block-lrd-strip.txtar` only (which exercises the single-line shape) — please add a paired `block-lrd-multiline-strip.txtar` (URL on next line) and `block-lrd-multiline-trailing-space.txtar` (trailing whitespace on the `]:` line) plus a CONTEXT that lets the reader confirm `[foo]` no longer resolves to user-supplied URL through the gnoweb chain. Reproducer: see Critical above; standalone Go test at `tests/multiline_lrd_bypass_test.go` in this review folder.
  </details>

- **[scheme case-insensitivity]** No fixture covers `HTTPS://` / `Https://` / mixed-case `MAILTO:`. Add to `golden/sanitize/url-*.txtar`.

- **[mailto Body= case-insensitive reject]** No fixture covers `?Body=` / `&BODY=`. Add a counterpart to `url-mailto-body-reject.txtar`.

## Suggestions

- [`examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno:1043-1064`](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1043-L1064) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1043-L1064) — `LinkReferenceDefinition` validates label via `FootnoteLabel` (`^[A-Za-z0-9_-]{1,64}$`). The doc encourages "namespaced labels using dashes (`r-myrealm-help`)". A realm that follows the namespacing convention is safe against shortcut-reference collision. Consider strengthening the helper to *require* a namespace separator (e.g. an internal "must contain `-`" check that mirrors `r/sys/users` rules), so the safer pattern is the default rather than convention. Minor — doc-vs-enforcement choice.
- [`gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go:163-189`](https://github.com/gnolang/gno/blob/84b818eb/gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go#L163-L189) · [↗](../../../../../.worktrees/gno-review-5714/gno.land/pkg/gnoweb/markdown/sanitize_integration_test.go#L163-L189) — per-case `NewMachineWithOptions(...)` + `RunMemPackage` adds noticeable latency (~3 min total CI per the PR's `main / test` run at 22m47s — most of which is `chain/markdown` related goldens). Cache the loaded driver mempackage once, then `m.SetActivePackage(pv)` against the same pre-loaded base store. Optional polish.

## Questions for Author

- The PR description says "first PR in a series of 2 or 3" with a `<gno-foreign>` sandbox coming next. Is the multi-line LRD strip considered in-scope for this PR or deferred? It looks like leaf sanitization (this PR's stated scope), not the foreign-markdown sandbox.
- The interrealm/PkgID/`.seal` commits — were they intentionally bundled, or just left from a working branch? They have separate owners on the repo and warrant their own PR/review path.
- For `BechString` HRP-empty mode ([sanitize.gno:728-741](https://github.com/gnolang/gno/blob/84b818eb/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L728-L741) · [↗](../../../../../.worktrees/gno-review-5714/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L728-L741)), the helper accepts any 1-16-char lowercase prefix as HRP. Is the intent to let realms display arbitrary bech32 strings without committing to a family, or should the family list be allowlisted (`g`, `gpub`, `cosmos`, …) to prevent display of attacker-chosen prefixes that mimic UI-trusted families?
