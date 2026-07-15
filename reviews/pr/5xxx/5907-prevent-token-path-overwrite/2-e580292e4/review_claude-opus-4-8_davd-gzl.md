# PR [#5907](https://github.com/gnolang/gno/pull/5907): fix(grc20reg): prevent token path overwrite and aliases

URL: https://github.com/gnolang/gno/pull/5907
Author: notJoon | Base: master | Files: 9 | +96 -55
Reviewed by: davd-gzl | Model: claude-opus-4-8 (high, deep) | Commit: e580292e4 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5907 e580292e4`

Round 2 (deep, multi-lens). Head advanced fba248c95 → e580292e4 (+2 commits: master merge, then a one-line fix). The round-1 Critical is resolved: the integration test now looks up `foo20` by its `.FOO` symbol key and passes. The round-1 slug-in-event finding is carried but re-graded to a Nit here: the event also emits the authoritative `token_path` and `symbol`, so a correct indexer is unaffected. Three lenses (red / blue / correctness) plus two critics and a claim-verification pass ran on this head; no Critical found.

**TL;DR:** `grc20reg` is the on-chain directory where a realm publishes its GRC20 token so other realms can look it up by a string key. This PR makes that key always the token's own identity (`realmPath.SYMBOL`), rejects a token whose identity does not match the registering realm, and blocks both overwriting an existing entry and registering one token under a second alias key.

**Verdict: APPROVE** — round-1 CI blocker fixed, hardening is sound and unbypassable; remaining items (vestigial `slug` in the register event, a nil-token test gap, a panic-wording nit) are all non-blocking.

## Summary
Before this PR, `Register` built the key from `cur.Previous().PkgPath()` plus a caller-supplied `slug` and wrote with `registry.Set`, so any realm could overwrite an existing entry, register the same token under many alias keys, or register under a path it does not own. The PR pins the key to `fqname.Construct(rlmPath, token.GetSymbol())`, asserts that equals `token.ID()` so only the token's declaring realm can register it, and rejects nil tokens and duplicate keys. Round 1 was `REQUEST CHANGES` because one downstream consumer of the old bare-path key (`grc20_registry_emit`) was missed and turned CI red; the new head fixes exactly that line. All in-tree consumers now pass.

## Examples
| Register call | Key before | Key after |
|---|---|---|
| `foo20.init`: `Register(cross, Token, "")` | `gno.land/r/demo/defi/foo20` | `gno.land/r/demo/defi/foo20.FOO` |
| `Register(cross, Token, "mySlug")` (same token, second call) | `...foo20.mySlug` (new alias entry) | panics `token already registered` |
| `Register` of a token declared in another realm | accepted under caller path | panics `token ID mismatch` |
| `Register(cross, nil, "")` | stored typed-nil, later `Get` faults | panics `nil token` |

## Glossary
- crossing / `cross`: a call into a crossing function `func F(cur realm, ...)`; the callee reads its caller through `cur.Previous()`.
- realm: stateful on-chain package under `r/`; `cur.Previous().PkgPath()` is the unforgeable caller path.
- txtar: testscript-based integration tests under `gno.land/pkg/integration/testdata/`.

## Fix
The registry key moves from caller-chosen (`rlmPath[.slug]`) to token-derived (`rlmPath.symbol`, asserted equal to `token.ID()`) in [`grc20reg.gno:30-37`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30-L37) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30). Nil tokens, cross-realm tokens, and duplicate keys all panic before any `registry.Set`, so no partial write occurs. The load-bearing constraint: `grc20.NewToken` forbids `.` `/` and whitespace in a symbol ([`token.gno:76-86`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/p/demo/tokens/grc20/token.gno#L76-L86) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/p/demo/tokens/grc20/token.gno#L76)), so the `rlmPath.symbol` key round-trips unambiguously through `fqname.Parse`.

## Critical (must fix)
None. The round-1 Critical (`grc20_registry_emit.txtar` looking up `foo20` by its bare path) is fixed on this head: the `TransferBy` argument is now `gno.land/r/demo/defi/foo20.FOO` ([`grc20_registry_emit.txtar:14`](https://github.com/gnolang/gno/blob/e580292e4/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14) · [↗](../../../../../.worktrees/gno-review-5907/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14)) and the test passes.

## Warnings (should fix)
None.

## Nits
- **[event carries a value that is no longer part of the key]** [`grc20reg.gno:43`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) — `slug` no longer affects the registry key but is still validated and emitted in the register event, so a caller that passes one has it silently dropped and an indexer reading `slug` gets a value that resolves to no entry.
  <details><summary>details</summary>

  The key is now built solely from `token.GetSymbol()` ([`grc20reg.gno:31`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L31) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L31)); `slug` only runs through `validateSlug` and is emitted alongside the new authoritative `token_path` and `symbol`. Behaviorally confirmed: a `Register(cross, token, "ignoredSlug")` emits `token_path=<realm>.EVT` while still carrying `slug=ignoredSlug`, so an indexer reconstructing `pkgpath + slug` computes a key that does not exist. Downgraded from a round-1 Warning: for this demo realm the harm needs an indexer that reads the stale `slug` and ignores the authoritative `token_path`/`symbol` in the same event, and no in-tree caller emits a misleading slug (foo20/test20/wugnot/bar20 pass `""`; grc20factory passes symbol-as-slug so its key still matches). The field is dead weight. Fix: drop `slug` from the event, or document at [`grc20reg.gno:23`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L23) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L23) that a non-empty slug is validated and emitted but never part of the key.
  </details>
- **[panic reads as an internal invariant, not a usage error]** [`grc20reg.gno:33`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L33) — registering from the wrong realm, or a direct EOA call (empty `PkgPath`), panics `token ID mismatch`, which reads like a broken internal assertion rather than "register from the token's own realm." A message naming the constraint would orient a realm author faster.

## Missing Tests
- **[nil-guard hardening has zero coverage]** [`grc20reg.gno:24`](https://github.com/gnolang/gno/blob/e580292e4/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L24) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L24) — the new `if token == nil` guard is real hardening but no test exercises it.
  <details><summary>details</summary>

  Removing the guard makes a nil token abort deep in `GetSymbol` as `value method gno.land/p/demo/tokens/grc20.Token.GetSymbol called using nil *Token pointer` instead of the clean `grc20reg: nil token`. Verified by reverting the guard in the worktree: the reject message degrades to the opaque nil-pointer form. The ready-to-add test asserts the clean message and locks the early reject; it passes on the head and fails with the guard removed. Test: [`tests/register_nil_token_test.gno`](tests/register_nil_token_test.gno).
  </details>

## Suggestions
- **[registration-proxy pattern is now silently dead — quarantined]** [`tokenhub.gno:33`](https://github.com/gnolang/gno/blob/e580292e4/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L33) · [↗](../../../../../.worktrees/gno-review-5907/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L33) — the ID-match check narrows `tokenhub.RegisterToken` to tokens declared in the tokenhub realm itself.
  <details><summary>details</summary>

  `RegisterToken` calls `grc20reg.Register(cross(cur), token, slug)`, so inside `Register` the caller path is always tokenhub. Because the key must equal `token.ID()`, only a token whose declaring realm is tokenhub can be registered through it; an external token now panics `token ID mismatch`. The PR's test edits route around this by dropping the `testing.SetRealm(NewCodeRealm(testRealmPkgPath))` lines so the tokens are created under tokenhub itself, which makes the suite pass but also removes coverage of the exact cross-realm path the PR hardens. tokenhub is quarantined (not loaded on-chain), so deployment impact is nil. This narrowing is the deliberate security property of the PR: registration is now self-sovereign and no intermediary can register another realm's token. Worth a one-line doc note on `RegisterToken` that it only accepts tokenhub's own tokens. Not posted: quarantined and by-design.
  </details>

## Verified
- Round-1 blocker fixed: `go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20_registry_emit'` passes; the `.FOO` key resolves where the bare path panicked in round 1.
- ID-match assertion is unbypassable: `fqname.Construct(rlmPath, symbol)` always takes the dotted branch (symbol is non-empty by `validSymbol`), and `token.ID() = origRealm + "." + symbol` shares the byte-identical suffix, so `key != token.ID()` reduces to `rlmPath != origRealm`. Direct EOA calls (`PkgPath == ""`) and cross-realm tokens both fail it.
- No partial write on a panic path: all three guards (nil / ID-mismatch / duplicate) run before the single `registry.Set`; reverting the nil guard degrades the reject to an opaque `GetSymbol` nil-pointer abort.
- No remaining old-bare-path consumer: swept every `grc20reg.Get/MustGet/GetRegistry/Register` caller in `examples/` and `gno.land/`; the txtar was the only one, `treasury.gno` uses an empty runtime-set key list, and the untouched `treasury_test.gno` already builds keys in `Construct(path, symbol)` form.
- Suites green at e580292e4: grc20reg, atomicswap, foo20, wugnot, bar20, grc20factory, treasury/test, quarantined tokenhub, and the `grc20_registry_emit` txtar.

## Open questions
- The registry key format changes for every already-registered token (`wugnot` → `gno.land/r/gnoland/wugnot.wugnot`, etc.). No in-tree consumer hardcodes the old format, but any already-deployed off-tree indexer keyed on a bare path silently gets `nil` or panics. grc20reg is a demo realm so impact is bounded; not posted, but a one-line note in the PR description would help downstream. No ADR accompanies the format change.
- `slug` is validated even though it is ignored for the key, so a caller can be rejected in `validateSlug` for an invalid value that has no effect. Folded into the slug Warning rather than posted separately.
- No test asserts that two distinct symbols from the same realm produce distinct keys (the grc20factory multi-token pattern). `TestRegisterRejectsOverwrite` covers same-symbol collision and the happy path incidentally; the multi-symbol case is only covered indirectly. Low value, not posted.
