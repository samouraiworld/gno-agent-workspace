# PR #5655: fix(gnoweb): change gnoweb to find usernames with hyphens

**URL:** https://github.com/gnolang/gno/pull/5655
**Author:** jeronimoalbi | **Base:** master | **Files:** 4 | **+63 -26**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7

## Summary

Two-layer fix so gnoweb and the VM keeper accept namespaces/usernames containing hyphens (a shape already permitted by `examples/gno.land/r/sys/users/store.gno` `reName = ^[a-z][a-z0-9]*([_-][a-z0-9]+)*$`):

1. `gno.land/pkg/gnoweb/weburl/url.go` — collapses the previous `rePkgPath` + `reUserPath` + `reNamespace` triple into a single `reGnolandPath = ^/[rpu]/([a-z][a-z0-9_/-]*)*$` (used by `IsValidPath`, `Namespace`) plus a more permissive `reURLPath = ^/[a-z0-9_/-]*$` (used by `ParseFromURL` so any in-shape URL parses; semantic validation moves to `IsRealm/IsPure/IsUser`). `IsPure/IsRealm/IsUser` now combine the `/r/|/p/|/u/` prefix check with `IsValidPath()`. `IsDir` is refactored to `strings.HasSuffix`. The user-only regex `[a-zA-Z0-9_]` is dropped, so user paths now share the realm/pure shape (lowercase, digits, `_`, `-`).
2. `gno.land/pkg/sdk/vm/keeper.go` — `reUserNamespace` (used by `QueryPaths` for `@name[/sub]` lookups) gains `-` in its char class: `^[~_a-zA-Z0-9/-]+$`.

Tests added: a parse case for `/r/hyphen-simple/test`, updated `TestIsValidPath` cases (including the new `{Path: "/", Valid: false}` change and `/r/hyphen-valid`, `/p/hyphen-valid/path`), and `TestNamespace` cases flipped (`/r/a-b/c → a-b`, `/r/valid-ns → valid-ns`, `/x/ns → ""`). The integration txtar `gnokey_qpaths.txtar` now loads `gno.land/{r,p}/foo-bar/...` and queries `@foo-bar` / `@foo-bar/baz` with golden output.

Codeowner review still required (alexiscolin OR gfanton APPROVED, per bot check).

## Test Results
- **Existing tests:** PASS for the changed surface: `go test ./gno.land/pkg/gnoweb/weburl/...` PASS; `go test ./gno.land/pkg/gnoweb/markdown/...` PASS; `go test ./gno.land/pkg/gnoweb/components/...` PASS; `go test -run TestTestdata/gnokey_qpaths ./gno.land/pkg/integration` PASS.
  - `go test ./gno.land/pkg/gnoweb` FAIL on `TestRoutes`, but this is **unrelated to the PR** — it reproduces identically with the four PR files reverted to `origin/master` (genesis tx fails because `examples/gno.land/r/tests/vm/subtests/subtests.gno:29:13` references `cur.SentCoins`, which is undefined in the current `realm` type on master). CI is green for the PR.
- **Edge-case tests:** skipped (focused regex exploration done inline; findings below).

## Critical (must fix)
- None.

## Warnings (should fix)
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:155` — `IsUser()` is now satisfied by any `/u/...` path that matches `reGnolandPath`, including multi-segment ones like `/u/alice/sub`. The previous `reUserPath = ^/u/[a-zA-Z0-9_]+$` explicitly forbade `/` after `/u/`. As a consequence `Username()` (`return gnoURL.Path[3:]`) will now return `"alice/sub"` for such a URL, and the handler at `gno.land/pkg/gnoweb/handler_http.go:416` (`strings.TrimPrefix(gnourl.Path, "/u/")`) will issue `Realm(ctx, "/r/alice/sub/home", "")`. It is a behavior change with no test coverage. Recommend either constraining `IsUser` to single-segment (e.g. add `&& !strings.Contains(gnoURL.Path[3:], "/")`) or constraining `Username()` to return `""` when the residue contains `/`.
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:173` — The repeated group `([a-z][a-z0-9_/-]*)*` only requires the leading rune of the *first* sub-segment to be `[a-z]`; once a `/` is consumed inside the inner char class, the next character can be `-` or `_`. As a result `/r/foo/-bar`, `/r/foo/_bar`, `/r/foo//bar`, `/r/foo-/bar` and `/r/foo--bar` all match `IsValidPath`. Some of these (double-slash, leading `-`/`_` on a sub-segment) are inconsistent with the spec embodied by `reName` in `r/sys/users`. The chain layer ultimately rejects bad pkg paths, but the weburl validator could be tightened (e.g. `^/[rpu]/([a-z][a-z0-9_-]*)(/[a-z][a-z0-9_-]*)*/?$`) to catch them at parse time. Not a security issue, just permissiveness drift.
- [ ] `gno.land/pkg/sdk/vm/keeper.go:1133` — `reUserNamespace = ^[~_a-zA-Z0-9/-]+$` similarly accepts `-foo`, `foo-`, `--bar`, `foo--bar`. Lookups against non-existent paths return empty, so this is informational, not exploitable. The `invalid username format` branch (returned on `!reUserNamespace.MatchString`) still has no Go-level unit test; only the txtar covers the happy path now. Adding a `TestQueryPaths_HyphenatedAndInvalid` in `gno.land/pkg/sdk/vm/keeper_test.go` would close the gap that gfanton flagged in PR comments.
- [ ] `gno.land/pkg/gnoweb/weburl/url_test.go:293` — Changing `{Path: "/", Valid: false}` is a behavior change for `IsValidPath`. It is safe today because `handler_http.go:169` matches `r.RequestURI == "/"` before calling `IsPure/IsUser`, but the change is undocumented and would silently break any future caller that relied on the prior contract. Worth a `// NOTE: "/" is no longer considered valid; root is handled separately by the HTTP layer.` comment near `reGnolandPath`.

## Nits
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:175-181` — Docstring: "It just validates ... but it doesn't validates semantics" — typo "validates → validate" (twice). Also "Use `IsPure()`, `IsRealm()` or similar" — comma before `or`, or use "Use `IsPure()` / `IsRealm()` / `IsUser()` for specific cases." for clarity.
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:189` — Comment `// skip /x/` is misleading now that `x` is no longer accepted (only `/r/`, `/p/`, `/u/`). Replace with `// skip "/r/", "/p/" or "/u/"`.
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:144,149,154` — "URL path prefix represents..." reads oddly. Suggest "checks if the URL path starts with `/p/` (and is a valid Gno.land path)" or similar.
- [ ] `gno.land/pkg/integration/testdata/gnokey_qpaths.txtar:60-68` — Consider adding a negative case: `gnokey query vm/qpaths --data "@-foo"` returning `invalid username format` would lock the rejection contract for purely hyphen-prefixed names if you want to tighten `reUserNamespace` later.

## Missing Tests
- [ ] `Namespace()` does not exercise the latent pre-existing `idx > 1` quirk: for `/r/a/b/c` the function returns `"a/b/c"` (whole residue) rather than `"a"`, because `strings.Index("a/b/c", "/")` is `1`, not `> 1`. The PR adds `{Path: "/r/a-b/c", Expected: "a-b"}` (idx=3, passes) but not `{Path: "/r/a/b", Expected: "a"}`. This is pre-existing, not caused by the PR, but easy to surface while the file is open. (`gno.land/pkg/gnoweb/weburl/url.go:190`)
- [ ] No Go-level test in `gno.land/pkg/sdk/vm/keeper_test.go` for the new `reUserNamespace` hyphen support or for the "invalid username format" rejection — only the txtar covers it.
- [ ] No test that `/u/alice/sub` either gets handled gracefully or rejected by `IsUser()` / `Username()`.

## Suggestions
- Anchor the per-segment rule explicitly to forbid sub-segments starting with `-` or `_`: `^/[rpu]/([a-z][a-z0-9_-]*(/[a-z][a-z0-9_-]*)*)?/?$`. This also rejects `//` consecutive slashes and trailing/leading garbage, and matches `reName` more closely.
- The PR-body justification "Also `IsValidPath()` is not being used anywhere" (from a thread reply) is no longer true after this PR — `IsPure/IsRealm/IsUser/Namespace` all call it. Keep that in mind for future renames; consider unexporting `IsValidPath` if it's purely an implementation detail. (`gno.land/pkg/gnoweb/weburl/url.go:179`)
- The package-level regex compilation is fine, but consider pre-computing the prefix check + path validation in a single regex per kind (`^/r/[a-z]...$`) so `IsRealm`/`IsPure`/`IsUser` don't both `HasPrefix` and `IsValidPath`. Minor.

## Questions for Author
- Was the broadening of `IsUser()` to accept `/u/alice/sub` intentional, or an inadvertent side-effect of unifying the three regexes? If unintentional, can `IsUser()` (or `Username()`) be tightened in this PR?
- Per moul's comment ("worried about allowing hyphens in usernames"): the canonical `reName` in `r/sys/users` and `r/sys/namereg/v1` already mandates hyphens be sandwiched between alphanumerics (`[_-][a-z0-9]+`). Should gnoweb mirror that stricter shape rather than the looser one used here, so an invalid URL fails fast at the edge instead of after the on-chain lookup?
- The `IsValidPath("/") = false` flip: any downstream consumer (Argonaut bookmarks, gnobro, third-party tooling) that depended on the previous "yes" answer for `/`? A `CHANGELOG` note would help.

## Verdict
APPROVE (with comments) — the core fix is correct, minimal, and well-covered by the new txtar; CI is green; the only failing test on my machine (`TestRoutes`) is a pre-existing master regression unrelated to this PR (reproduces with the four PR files reverted). The warnings above are quality-of-life improvements; the `IsUser`/`Username` widening for multi-segment `/u/x/y` paths is the one worth a follow-up before merge.
