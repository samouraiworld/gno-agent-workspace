# PR #5655: fix(gnoweb): change gnoweb to find usernames with hyphens

URL: https://github.com/gnolang/gno/pull/5655
Author: jeronimoalbi | Base: master | Files: 4 | +63 -26
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `418812d` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5655 418812d`

## Summary

Three places in the read path of gnoweb and the VM keeper were silently filtering out registered namespaces that contain hyphens, even though `r/sys/users.reName` (`^[a-z][a-z0-9]*([_-][a-z0-9]+)*$`) has been accepting `[_-]` as an internal separator for a while. The PR plugs the three gaps:

1. `gno.land/pkg/gnoweb/weburl/url.go` — the URL grammar is overhauled. The old `rePkgPath` (`^/[a-z0-9_/]*$`) plus `reUserPath` (`^/u/[a-zA-Z0-9_]+$`) plus `reNamespace` (`^/[a-z]/[a-z][a-z0-9_/]*$`) are collapsed into a single `reGnolandPath = ^/[rpu]/([a-z][a-z0-9_/-]*)*$` that anchors on the `[rpu]` prefix and admits `-`. A separate broad `reURLPath = ^/[a-z0-9_/-]*$` is used inside `ParseFromURL` to gate generic URL paths before semantic dispatch. `IsPure/IsRealm/IsUser` now also call `IsValidPath()` (previously they only string-prefix-matched).

2. `gno.land/pkg/sdk/vm/keeper.go` — the qpaths name validator `reUserNamespace` gains `-`: `^[~_a-zA-Z0-9/]+$` → `^[~_a-zA-Z0-9/-]+$`. Without this, `gnokey query vm/qpaths --data "@foo-bar"` would 400 with "invalid username format" even though the namespace is registrable.

3. `gno.land/pkg/integration/testdata/gnokey_qpaths.txtar` — adds two hyphenated packages (`r/foo-bar/baz`, `p/foo-bar/pkg`) plus two `@foo-bar` qpath queries with golden output to lock in the new behavior.

Design notes worth flagging:
- The PR tightens semantics by anchoring on `/[rpu]/` — paths like `/x/ns` and bare `/` are no longer `IsValidPath`. The author confirmed this is intentional (see alexiscolin's review thread).
- The URL regex is more permissive than `r/sys/users.reName`: it accepts trailing hyphens (`/r/foo-`), consecutive hyphens (`/r/foo--bar`), `[_-]` mixed (`/r/foo-_bar`), double slashes (`/r/foo//bar`), and segments starting with `-` after the first segment (`/r/foo/-bar`). Treated as a syntactic gate for routing, not as a registration validator — that's `r/sys/users` job. In practice the VM's own `Re_name` at `gnovm/pkg/gnolang/mempackage.go:69` rejects those same shapes at deploy time, so they 404 one layer later rather than at URL parse. The only edge case where the loose regex matters is `/r/foo//bar`, where `//` can normalize inconsistently across caches/proxies.
- Related: moul raised concerns about hyphens in usernames generally. That concern is upstream (the `reName` decision in `r/sys/users.store`), not introduced by this PR.

## Test Results
- Existing tests: PASS (`go test ./gno.land/pkg/gnoweb/weburl/...`)
- CI: all checks green (CodeQL, build, codecov, etc.)
- Edge-case tests: skipped (the existing table-driven tests cover the headline cases; the regex behavior on `-/--//-leading` was probed via a one-off Go script and the findings are captured below)

## Critical (must fix)
None.

## Warnings (should fix)
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:173` — `/r/foo//bar` passes `IsValidPath`; double-slash paths normalize inconsistently across caches/proxies and should be rejected or normalized.
  <details><summary>details</summary>

  The regex is `^/[rpu]/([a-z][a-z0-9_/-]*)*$`. The character class `[a-z0-9_/-]*` after the first segment-leading `[a-z]` admits `//`, so `/r/foo//bar` returns `true` from `IsValidPath` and `"foo"` from `Namespace()`.

  Unlike the other looseness in this regex (trailing hyphen, consecutive hyphens, segment-leading hyphen — all rejected by the VM's own `Re_name` at `gnovm/pkg/gnolang/mempackage.go:69`, so they 404 one layer later and there's no practical harm), `//` is special: HTTP intermediaries, browsers, and the gateway sometimes collapse it to `/` and sometimes don't. That can lead to cache keys that disagree with VM lookup keys, or to a path that round-trips through `Encode` differently than it came in. Worth pinning a test case and either rejecting `//` in the regex or normalizing it during `ParseFromURL`.

  Suggested anchor for the segment grammar (rejects empty segments while still allowing hyphens): `^/[rpu]/([a-z][a-z0-9_-]*)(/[a-z][a-z0-9_-]*)*/?$` or similar.
  </details>

- [ ] `gno.land/pkg/gnoweb/weburl/url.go:179` — `IsValidPath("/")` changed from `true` to `false`; public API change, callers outside this package may break.
  <details><summary>details</summary>

  `IsValidPath` is exported on `GnoURL`. The `weburl` package is consumed by gnoweb internals, but the type is also a public surface (see `gno.land/pkg/gnoweb/handler_http.go` and various tests). The test case was flipped from `{Path: "/", Valid: true}` to `{Path: "/", Valid: false}` — a deliberate semantic tightening.

  Internal `grep -rn IsValidPath gno.land/` only shows callers inside `weburl/`, but third-party tools (gnodev plugins, gnobro, downstream forks) that build on `weburl.GnoURL` may rely on bare-root validity. Worth mentioning explicitly in the PR description as a behavior change so release notes pick it up. Same goes for the removal of `reNamespace` — paths like `/x/ns` now return `""` from `Namespace()` instead of `"ns"` (also flipped in tests at `url_test.go:326`).
  </details>

- [ ] `gno.land/pkg/sdk/vm/keeper.go:1133` — `reUserNamespace` now allows leading/trailing hyphens and mixed case; intentional looseness should be documented.
  <details><summary>details</summary>

  `reUserNamespace = ^[~_a-zA-Z0-9/-]+$` is the qpaths `@<name>` validator. With `-` added it now accepts:
  - `@-foo` (leading hyphen — can never resolve to a `reName`-valid namespace)
  - `@foo-` (trailing hyphen — same)
  - `@FOO` (uppercase — `reName` is lowercase-only)
  - `@foo--bar` (consecutive hyphens — same)

  These will never match a registered package because `r/sys/users.reName` rejects them, so the queries return empty results rather than the "invalid username format" error. The looseness is consistent with the regex's role as a syntactic gate, but a one-line comment explaining "intentionally loose; semantic validity is enforced downstream by `r/sys/users.reName`" would prevent the next reader from filing this as a bug.
  </details>

## Nits
- [ ] `gno.land/pkg/gnoweb/weburl/url.go:178` — typo: "doesn't validates" → "doesn't validate".

- [ ] `gno.land/pkg/integration/testdata/gnokey_qpaths.txtar:48,52` — two blank lines were removed unrelated to the feature. Harmless, but it inflates the diff and risks merge-conflict noise on a high-churn integration file.

- [ ] `gno.land/pkg/gnoweb/weburl/url.go:184` — `Namespace()` doc example uses `/r/test/foo` → `"test"`; the function returns `"test"` only because `idx=4`. For a single-char namespace (`/r/a/b`), `idx=1` and the function returns the full `"a/b"` (pre-existing bug, not introduced here — see Missing Tests below).

## Missing Tests
- [ ] `gno.land/pkg/gnoweb/weburl/url_test.go:288` — `IsValidPath` table omits the permissive edge cases the new regex accepts.
  <details><summary>details</summary>

  Currently tested: `/r/hyphen-valid`, `/p/hyphen-valid/path`. Missing:
  - `/r/-foo` (expected `false` — locks in the leading-hyphen rejection alexiscolin asked about)
  - `/r/foo-` (currently `true` — surprising; worth pinning whichever way it lands)
  - `/r/foo--bar` (currently `true`)
  - `/r/foo//bar` (currently `true` — double-slash path; should probably be `false`)
  - `/r/foo/-bar` (currently `true` — segment-leading hyphen)

  Without these in the table, future tightening of `reName` semantics can silently regress gnoweb's behavior.
  </details>

- [ ] `gno.land/pkg/gnoweb/weburl/url_test.go:316` — `Namespace()` table omits the single-char + sub-path case `/r/a/b`.
  <details><summary>details</summary>

  With current code, `Namespace("/r/a/b")` returns `"a/b"` (not `"a"`) because of the `if idx > 1` guard at `url.go:190`. The guard treats single-char namespaces as "no slash worth splitting on" and returns the entire remainder.

  Pre-existing — not introduced by this PR — but the test table already covers `/r/a` and `/r/a_b/c`, so adding `/r/a/b` is one line and would surface the bug. If the intended semantics are "split on the first slash regardless of namespace length", the guard should be `idx >= 1` (or just `idx != -1`) and a separate commit could fix it.
  </details>

- [ ] `gno.land/pkg/integration/testdata/gnokey_qpaths.txtar` — no negative test for hyphen-validator edge cases.
  <details><summary>details</summary>

  The new golden tests confirm `@foo-bar` works. There's no test for `@-foo`, `@foo-`, or `@FOO` to confirm whether qpaths returns "invalid username format" or empty results. Given moul's worry about hyphen-confusable namespaces, locking down the leading/trailing-hyphen behavior at the integration layer would close the loop.
  </details>

## Suggestions
- `gno.land/pkg/gnoweb/weburl/url.go:173` — consider mirroring `r/sys/users.reName` more tightly.
  <details><summary>details</summary>

  If the regex matched registered-namespace shape exactly (`[a-z][a-z0-9]*([_-][a-z0-9]+)*`), gnoweb would reject unresolvable paths at parse time instead of at lookup time. Cleaner error path and removes the "permissive gate vs strict registration" gap that this review's first warning describes. Trade-off: any future loosening of `reName` requires a coordinated change here. If the team prefers the loose-gate posture (so gnoweb survives small registration-policy tweaks without redeploy), keep as-is and add the documenting comment.
  </details>

- `gno.land/pkg/sdk/vm/keeper.go:1133` — `reUserNamespace` and `reGnolandPath` are now two separate regexes that need to stay roughly in sync.
  <details><summary>details</summary>

  Different surfaces, different anchors (one expects `@<name>`, the other expects `/<r|p|u>/<name>/...`), but both gate "is this a syntactically plausible namespace?". A small shared helper (`isValidNameRune` or a single `reName`-derived regex) would prevent these from drifting. Not blocking — just flagging the duplication.
  </details>

## Questions for Author
- The regex `^/[rpu]/([a-z][a-z0-9_/-]*)*$` is looser than `reName` — is that intentional, or should it tighten to the registration shape?
  <details><summary>details</summary>

  In particular: trailing hyphens, consecutive hyphens, mixed `_-`, and segment-leading hyphens (`/r/foo/-bar`) all pass. The author's reply on alexiscolin's earlier comment indicates intent ("the regex was anchored on `[a-z]`"), but only the first character is anchored — subsequent chars are unconstrained beyond the character class. A one-line answer in the PR thread (or a code comment) settles it.
  </details>

- Re: moul's hyphen-vs-underscore concern — is the policy decision (`reName` already allows both) being revisited as part of this PR, or is this strictly the downstream-plumbing fix?
  <details><summary>details</summary>

  My read is the latter: `reName` already accepted hyphens (intentionally, to mirror gno's package-name `Re_name` shape), and this PR just makes gnoweb/keeper honor what registration has been minting. If the team wants to revisit the policy, that's a separate change to `examples/gno.land/r/sys/users/store.gno:27` plus migration story. Confirming this framing in a reply to moul should unblock his approval.
  </details>

- Should `/e/<addr>/run` paths be a future case, or are ephemeral packages permanently out of gnoweb's surface?
  <details><summary>details</summary>

  `reGnolandPath` is `[rpu]` only — no `e`. That's correct today: ephemeral `MsgRun` packages (`gno.land/e/<g1-addr>/run`, see [keeper.go:1009](gno.land/pkg/sdk/vm/keeper.go#L1009)) are per-transaction and gnoweb has no handler for them (`handler_http.go` only dispatches on `IsRealm/IsPure/IsUser`). No reason to expose them in the URL grammar. Worth a brief comment in the regex docstring noting why `e` is excluded, so future contributors don't add it by reflex.
  </details>

## Verdict
APPROVE — the fix is correct, scoped, and covered by tests; the warnings are about regex looseness and a public-API semantic change that should be acknowledged in the description but don't block merge.
