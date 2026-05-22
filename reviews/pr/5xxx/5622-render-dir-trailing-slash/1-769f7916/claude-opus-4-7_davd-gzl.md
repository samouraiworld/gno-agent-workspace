# PR #5622: feat(gnoweb): differenciate render and dir view with $dir

**URL:** https://github.com/gnolang/gno/pull/5622
**Author:** AmozPay | **Base:** master | **Files:** 8 | **+232 -23**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7 (1M context)

## Summary

Closes #5581. Replaces the implicit "trailing slash means directory view" convention with an explicit `$dir` web-query selector, mirroring the existing `$source` / `$help` scheme. Trailing-slash URLs still work but now 302-redirect to a canonical (slash-stripped) form.

Implementation:

- [weburl/url.go:163-168](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/weburl/url.go#L163-L168) — `IsDir()` now requires `WebQuery.Has("dir")` instead of trailing-slash detection. The doc comment names `$dir` as canonical.
- [handler_http.go:161-164](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L161-L164) — early-return redirect: any non-file URL whose path ends in `/` and is longer than 3 chars (skipping bare `/r/`, `/p/`, `/u/`) gets 302'd to its canonical form.
- [handler_http.go:194-217](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L194-L217) — new `canonicalPathURL(r)` builds the redirect target by splitting the raw escaped path on `:` first, then `$`, trimming the trailing slash from the prefix, and reattaching the suffix and `RawQuery`.
- Defensive `strings.TrimSuffix(gnourl.Path, "/")` added to four call sites (`GetRealmView`, `GetHelpView`, `GetSourceView`, `ServeSourceDownload`) so backend lookups never see the trailing slash even if the redirect path is bypassed.
- [components/views/directory.html:15](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/components/views/directory.html#L15) — explorer-mode child links now emit `$dir` instead of trailing `/`.
- Tests: `TestIsDir` (5 cases, weburl), `TestCanonicalPathURL` (4 cases, internal), `TestHTTPHandler_Get_CanonicalizesTrailingSlashPaths` (3 cases), `TestDirectoryView_ExplorerLinksUseDirSelector`, plus updates to existing tests to use the new URL forms.

## Test Results

- **Existing tests:** PASS — `go test ./gno.land/pkg/gnoweb/...` clean (~67s for the gnoweb package, all golden + table tests pass).
- **CI:** all jobs green; `Merge Requirements` red is the codeowner gate (gnoweb codeowner approval pending). Not a code issue.
- **Edge-case tests:** 4 written, 4 PASS (saved under `tests/edge_test.go`):
  - `TestCanonicalPathURL_ArgsAndWebQuery` — `/r/demo/foo/:bob$source` → `/r/demo/foo:bob$source`. ✓
  - `TestCanonicalPathURL_DoubleTrailingSlash` — `/r/demo/foo//` only trims one slash → `/r/demo/foo/`; second slash triggers another redirect. Documents behavior, not a bug.
  - `TestCanonicalPathURL_DollarBeforeColon` — `/r/foo$bar:baz` not corrupted by Cut order.
  - `TestCanonicalPathURL_EmptyAfterDelimiter` — `/r/foo/:` collapses cleanly to `/r/foo` (empty suffix branch).

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] [handler_http.go:194-217](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L194-L217) — `canonicalPathURL` re-parses what `weburl.ParseFromURL` already split. The same canonical URL can be obtained by `gnourl.Path = strings.TrimSuffix(gnourl.Path, "/"); gnourl.EncodeWebURL()`. The duplication is justified only because `EncodeWebURL` reorders WebQuery keys alphabetically (`$source&file=…` would become `$file=…&source`), and you want the redirect Location to mirror what the user typed. **Add a one-line comment** to that effect — otherwise it reads as parser duplication and a future cleanup will tempt someone to "simplify" it back into `EncodeWebURL`. Same applies to the magic `> 3` length check ("skip bare `/r/`, `/p/`, `/u/`").
- [ ] [handler_http.go:161](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L161) — uses `http.StatusFound` (302). For URL canonicalization, `http.StatusMovedPermanently` (301) is the conventional choice — it tells crawlers and intermediaries the new URL is the permanent location, avoiding wasted round trips on every request. 302 is fine as a transition default but worth flipping to 301 once the new scheme is settled. Worth a brief mention in the PR description either way.
- [ ] Four sites (`GetRealmView` [:358](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L358), `GetHelpView` [:499](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L499), `GetSourceView` [:601](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L601), `ServeSourceDownload` [:751](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L751)) now do `pkgPath := strings.TrimSuffix(gnourl.Path, "/")` independently. The early redirect (line 161) means production callers should never reach these with a trailing slash. Either:
  - drop the duplication and trust the redirect, or
  - centralize the trim in `weburl.ParseFromURL` so `gnourl.Path` is always slash-free.
  The current state is "belt + suspenders + extra suspenders" and the inconsistency (some places use `gnourl.Path` raw, e.g. `gnourl.Path` in log lines at [:605](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L605), [:610](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L610), vs `pkgPath` in client calls) is a footgun for the next change.

## Nits

- [ ] [handler_http.go:160](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L160) — the redirect block sits between the parse error and the download-flow branch with no separating comment. A one-liner `// Canonicalize trailing-slash paths via 302 redirect.` would help future readers.
- [ ] [weburl/url.go:163-168](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/weburl/url.go#L163-L168) — `IsDir()` semantics changed silently (used to be "path ends with `/`"). Worth a `// BREAKING:` note in the godoc or PR description for downstream consumers of `weburl` (if any exist outside this package — `grep` shows only internal use, so risk is low).
- [ ] [handler_http_test.go:175](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http_test.go#L175) — new `{Path: "/r/mock/path$bogus", Status: http.StatusOK, Contain: "[example.com]/r/mock/path"}` row asserts unknown web-query keys fall through to the realm view. Good defensive test, but unrelated to the `$dir` change — call it out in the commit message or split it.
- [ ] [components/views/directory.html:15](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/components/views/directory.html#L15) — `{{ if $.Mode.IsExplorer }}$dir{{ end }}` works but `$dir` is now a magic string repeated across handler + template. A template global / shared constant would prevent future drift. Low priority.
- [ ] `// First fecth the realm` typo at [handler_http.go:360](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L360) — pre-existing, not introduced by this PR. Drive-by fix opportunity.

## Missing Tests

- [ ] Trailing slash on a `/u/` path (user view) — the redirect-skip threshold (`> 3`) covers `/u/` but not `/u/alice/`. Should redirect to `/u/alice`. Not in `TestHTTPHandler_Get_CanonicalizesTrailingSlashPaths`. (`gno.land/pkg/gnoweb/handler_http_test.go`)
- [ ] Trailing slash on a `/p/` path (pure package) — same as above. (`gno.land/pkg/gnoweb/handler_http_test.go`)
- [ ] Bare `/r/`, `/p/`, `/u/` — verify they DON'T redirect (the `> 3` skip). One row each in the canonicalization table would pin the threshold semantics. (`gno.land/pkg/gnoweb/handler_http_test.go`)
- [ ] Combined `:args` + `$webquery` + trailing slash — e.g. `/r/foo/:bob$source` → `/r/foo:bob$source`. Covered in my edge tests but not in the PR's. (`gno.land/pkg/gnoweb/handler_http_internal_test.go`)
- [ ] HTTP-level test that `$dir` actually serves the directory view (not just that explorer-mode links emit it) — the existing `TestHTTPHandler_DirectoryViewExplorerMode` uses `$dir`, which covers it indirectly, but a focused test would be clearer.

## Suggestions

- The behavior pair (#5618 exposes "Render" link, #5622 makes `$dir` explicit) is meant to land together. Consider stacking them in a single PR or noting the dependency in the description so a reviewer landing #5622 alone doesn't accidentally produce a UX where neither URL form is discoverable from the dir view.
- The PR description is one line. For a behavior change that flips the meaning of trailing-slash URLs across the gnoweb surface (and changes a few status codes from 404 to 302), a short before/after table — "what URL serves what now" — would speed reviewer comprehension.
- Consider documenting the URL scheme in `docs/` (or whichever gnoweb doc page exists). Today the `$source`/`$help`/`$dir` family is only discoverable by reading handler code.

## Questions for Author

- Why 302 over 301? Both work; 301 is the conventional canonicalization status and avoids the round trip on subsequent visits.
- Is there an intent to also redirect `/r/foo` (no slash, current realm-render URL) to `/r/foo$dir` when the realm has no Render? Current behavior falls through to dir view in `GetRealmView` ([:366](.worktrees/gno-review-5622/gno.land/pkg/gnoweb/handler_http.go#L366)) without changing the URL — fine, just confirming intent.
- The four `strings.TrimSuffix(gnourl.Path, "/")` additions look defensive given the early redirect. Is there a code path where the redirect can be bypassed and these still need to fire? If not, prune them; if yes, a comment naming that path would help.

## Verdict

**APPROVE** (with warnings worth addressing). Solid, well-tested change that cleans up an ambiguous URL convention. CI green, codecov 100% on patched lines, no correctness issues found across edge tests. The duplication-and-redundancy concerns under Warnings are quality-of-life, not blockers — would be nice to address before merge but the change is safe to land as-is. Pair with #5618 for the full UX improvement.
