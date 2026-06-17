# PR #5743: fix(p/nt/markdown/sanitize): case-insensitive mailto ?body= reject

URL: https://github.com/gnolang/gno/pull/5743
Author: davd-gzl | Base: master | Files: 3 | +22 -6
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `cb3be8c26` (stale — +26 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5743 cb3be8c26`

**Verdict: APPROVE** — correct, minimal, tested fix for the case-insensitivity hole; the percent-encoding bypass [@thehowl](https://github.com/gnolang/gno/pull/5743#discussion_r) flagged is real and still open, but it predates this PR and is a fair follow-up, not a regression. Three maintainers already approved.

## Summary
`linkSchemeAllowed` rejected `mailto:` prefill-phishing links by substring-matching `?body=` / `&body=`. RFC 6068 §2 makes the header field name case-insensitive, so `?BODY=` / `?Body=` slipped through and rendered a clickable `mailto:` href that pre-fills the message body. The fix lowercases the URL before the contains check. Covered by five new `TestURL` rows and one gnoweb golden fixture.

## Glossary
- `linkSchemeAllowed` — internal allowlist gate in `sanitize/v0`; returns false to make `URL()` emit `""`.
- `PercentEncodeURL` — `chain/markdown` helper that percent-encodes unsafe bytes; preserves already-valid `%XX` escapes.
- hfname — RFC 6068 mailto header field name (`body`, `subject`, `cc`, `bcc`).

## Fix
Before: the guard compared the raw string, so any non-lowercase spelling of `body` bypassed it. After: it lowercases a copy first, so `?BODY=`, `?Body=`, and the `&BODY=` second-param form all reject. See [`sanitize.gno:1624-1632`](https://github.com/gnolang/gno/blob/cb3be8c26/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1624-L1632) · [↗](../../../../../.worktrees/gno-review-5743/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1624-L1632). The package doc at [`sanitize.gno:1058`](https://github.com/gnolang/gno/blob/cb3be8c26/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1058) · [↗](../../../../../.worktrees/gno-review-5743/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1058) still says "?body= or &body=" without the case note; minor, see Nits.

## Critical (must fix)
None.

## Warnings (should fix)
None. The percent-encoding gap below is real but out of scope for this PR (pre-existing, author and reviewers agree on deferring) — recording it as a Suggestion so it is not lost.

## Nits
- [`sanitize.gno:1058`](https://github.com/gnolang/gno/blob/cb3be8c26/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1058) · [↗](../../../../../.worktrees/gno-review-5743/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1058) — `URL` doc comment still describes the rule as "?body= or &body=" with no mention of case-insensitivity; the inline comment at L1625 now does. One-line tweak keeps the exported doc honest.

## Missing Tests
None blocking. The new rows cover `?BODY=`, `?Body=`, `&body=` second-param, `&BODY=` second-param, and a `?subject=` allow-through. The one scenario not asserted is the percent-encoded bypass below, which would fail today — appropriate to add alongside the fix, not here.

## Suggestions
- [`sanitize.gno:1624-1632`](https://github.com/gnolang/gno/blob/cb3be8c26/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1624-L1632) · [↗](../../../../../.worktrees/gno-review-5743/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1624-L1632) — substring match still misses percent-encoded `body`, e.g. `?%62ody=`. Decode the query before matching, or reject any `mailto:` with a non-empty query.
  <details><summary>details</summary>

  The guard compares a lowercased copy of the raw string, but never percent-decodes. `PercentEncodeURL` ([`markdown.go:137-152`](https://github.com/gnolang/gno/blob/cb3be8c26/gnovm/stdlibs/chain/markdown/markdown.go#L137-L152) · [↗](../../../../../.worktrees/gno-review-5743/gnovm/stdlibs/chain/markdown/markdown.go#L137-L152)) only encodes a bare `%` not followed by two hex digits, so a valid `%62` passes through unchanged. The rendered href then reads `mailto:a@b.com?%62ody=phish`, which the browser decodes to `body=phish` at click time — the exact prefill-phishing payload the filter exists to block. This is what [@thehowl](https://github.com/gnolang/gno/pull/5743#discussion_r) meant by "decode using net/url so we cannot be tricked by percent-encoding." His second comment goes further: a `mailto:` can also carry `subject`, `cc`, `bcc`, which the current filter allows (confirmed below), so the robust fix is to reject any `mailto:` whose RawQuery is non-empty rather than blocklisting `body` field by field.

  Confirmed empirically against this commit:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5743 -R gnolang/gno
  cat > examples/gno.land/p/nt/markdown/sanitize/v0/zz_bypass_test.gno <<'EOF'
  package sanitize

  import "testing"

  func TestMailtoPercentEncodedBodyBypass(t *testing.T) {
  	got := URL("mailto:a@b.com?%62ody=phish")
  	if got != "" {
  		t.Errorf("BYPASS: percent-encoded body= survived, got %q", got)
  	}
  }
  EOF
  gno test -v -run TestMailtoPercentEncodedBodyBypass ./examples/gno.land/p/nt/markdown/sanitize/v0/
  rm examples/gno.land/p/nt/markdown/sanitize/v0/zz_bypass_test.gno
  ```

  ```
  === RUN   TestMailtoPercentEncodedBodyBypass
      BYPASS: percent-encoded body= survived, got "mailto:a@b.com?%62ody=phish"
  --- FAIL: TestMailtoPercentEncodedBodyBypass (0.00s)
  FAIL
  ```

  Fix: percent-decode the query segment before the contains check, or (simpler and stricter, matching the cc/bcc concern) reject any `mailto:` containing `?`. Note the existing golden fixture `url-mailto-cc-bcc-reject.txtar` is misnamed — its expected output allows the cc/bcc link, documenting current behavior rather than rejecting it.
  </details>

## Questions for Author
- Is the percent-encoding / full-RawQuery rejection planned as a follow-up PR, or intentionally left as accepted residual risk? The current fix is strictly an improvement either way.
