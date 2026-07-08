# PR [#5907](https://github.com/gnolang/gno/pull/5907): fix(grc20reg): prevent token path overwrite and aliases

URL: https://github.com/gnolang/gno/pull/5907
Author: notJoon | Base: master | Files: 8 | +96 -55
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: fba248c95 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5907 fba248c95`

**TL;DR:** `grc20reg` is the on-chain directory where a realm publishes its GRC20 token so other realms can look it up by a string key. This PR makes that key always the token's own identity (`realmPath.SYMBOL`), rejects a token whose identity does not match the registering realm, and blocks both overwriting an existing entry and registering one token under a second alias key.

**Verdict: REQUEST CHANGES** — the key format change breaks an existing integration test the PR did not update, so the `main / test` job is red; the vestigial `slug` still shows up in the register event where it now maps to no key.

## Summary
Before this PR, `Register` built the key from `cur.Previous().PkgPath()` plus a caller-supplied `slug`, so any realm could overwrite an existing entry or register the same token under many alias keys, and a realm could register under a path it does not own. The PR pins the key to `fqname.Construct(rlmPath, token.GetSymbol())`, asserts that equals `token.ID()` so only the token's declaring realm can register it, and rejects duplicate keys. The rewrite is sound in isolation and well covered by new unit tests, but it is a breaking change to the key format, and one downstream consumer of the old bare-path key was missed: the `grc20_registry_emit` integration test still looks up `foo20` by its bare path and now panics, turning CI red.

## Examples
| Register call | Key before | Key after |
|---|---|---|
| `foo20.init`: `Register(cross, Token, "")` | `gno.land/r/demo/defi/foo20` | `gno.land/r/demo/defi/foo20.FOO` |
| `Register(cross, Token, "mySlug")` (same token, second call) | `...foo20.mySlug` (new alias entry) | panics `token already registered` |
| `Register` of a token declared in another realm | accepted under caller path | panics `token ID mismatch` |

## Glossary
- crossing / `cross`: a call into a crossing function `func F(cur realm, ...)`; the callee reads its caller through `cur.Previous()`.
- realm: stateful on-chain package under `r/`; `cur.Previous().PkgPath()` is the unforgeable caller path.
- txtar: testscript-based integration tests under `gno.land/pkg/integration/testdata/`.

## Fix
The registry key moves from caller-chosen (`rlmPath[.slug]`) to token-derived (`rlmPath.symbol`, asserted equal to `token.ID()`) in [`grc20reg.gno:30-37`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30-L37) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30). Nil tokens, cross-realm tokens, and duplicate keys now panic before any `registry.Set`, so no partial write occurs. The load-bearing constraint: `grc20.NewToken` already forbids `.` `/` and whitespace in a symbol ([`token.gno:38-39`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/p/demo/tokens/grc20/token.gno#L38-L39) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/p/demo/tokens/grc20/token.gno#L38)), so the `rlmPath.symbol` key round-trips unambiguously through `fqname.Parse`.

## Critical (must fix)
- **[a token lookup that used to work now panics, CI is red]** [`gno.land/pkg/integration/testdata/grc20_registry_emit.txtar:14`](https://github.com/gnolang/gno/blob/fba248c95/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14) · [↗](../../../../../.worktrees/gno-review-5907/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14) — the registry key changed from the bare realm path to `realmPath.SYMBOL`, but this test still looks up `foo20` by its bare path, so `MustGet` panics `unknown token` and the `main / test` job fails.
  <details><summary>details</summary>

  `foo20.init` registers with an empty slug ([`foo20.gno:25`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/r/demo/defi/foo20/foo20.gno#L25) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/foo20/foo20.gno#L25)), so before this PR its key was `gno.land/r/demo/defi/foo20` and after it is `gno.land/r/demo/defi/foo20.FOO`. The test's `TransferBy` call passes the bare path `gno.land/r/demo/defi/foo20`, which now resolves to nothing and panics in `MustGet` at `grc20reg.gno:59`. Reproduced on the reviewed head; the one-line fix (append `.FOO` to the lookup arg on line 14) turns the test green, confirmed by re-running it. This is the only registry-lookup consumer keyed on the old bare path across the repo; `wugnot`/`bar20`/`test20`/`grc20factory` register and read their own tokens and are unaffected. The second red subtest in the same job, `govdao_proposal_change_law` (out of gas by 34 units), is an unrelated borderline flake, not caused by this PR. Fix: update the `TransferBy` argument on line 14 to `gno.land/r/demo/defi/foo20.FOO`. Repro in [comment.md](comment_claude-opus-4-8.md).
  </details>

## Warnings (should fix)
- **[register event advertises a slug that maps to no registry key]** [`grc20reg.gno:43`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) — `slug` no longer affects the registry key but is still validated and emitted in the register event, so a caller that passes one has it silently dropped and an indexer reading `slug` gets a value that resolves to no entry.
  <details><summary>details</summary>

  The key is now built solely from `token.GetSymbol()` ([`grc20reg.gno:31`](https://github.com/gnolang/gno/blob/fba248c95/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L31) · [↗](../../../../../.worktrees/gno-review-5907/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L31)); `slug` is only run through `validateSlug` and emitted. Two in-tree callers still pass a meaningful value: `grc20factory` passes the symbol and `tokenhub.RegisterToken` passes a user slug, both now ignored for the key. Confirmed behaviorally: the added `TestRegistry` registers with slug `"mySlug"` and then retrieves the token by `realmAddr + ".TST"`, not by the slug. The register event still carrying `"slug", slug` alongside the real `token_path` misleads any consumer indexing on it. Fix: drop `slug` from the event, and either remove the now-legacy parameter or document that it is ignored.
  </details>

## Nits
None.

## Missing Tests
None. The new unit tests cover the overwrite, alias, cross-realm-mismatch, and nil cases directly; the gap is the un-updated integration test above, not missing coverage.

## Suggestions
- `examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno:34` — the ID-match check narrows `tokenhub.RegisterToken` to tokens declared in the tokenhub realm itself.
  <details><summary>details</summary>

  `RegisterToken` calls `grc20reg.Register(cross(cur), token, slug)` ([`tokenhub.gno:34`](https://github.com/gnolang/gno/blob/fba248c95/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L34) · [↗](../../../../../.worktrees/gno-review-5907/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/tokenhub.gno#L34)), so `rlmPath` inside `Register` is tokenhub. Because the key must equal `token.ID()`, only a token whose declaring realm is tokenhub can be registered through it; an external token now panics `token ID mismatch`. The PR's test edits route around this by dropping the `testing.SetRealm(NewCodeRealm(testRealmPkgPath))` lines so the tokens are created under tokenhub itself, which makes the suite pass but hides that the hub's cross-realm registration path is now dead. tokenhub is quarantined so deployment impact is nil; worth confirming the intent and noting it, since the intended "hub registers anyone's token" pattern is no longer possible under the new self-registration model. Not posted: quarantined, and the narrowing is the deliberate security property of this PR.
  </details>

## Open questions
- The `slug` parameter is kept for signature compatibility but is now legacy. Whether to drop it entirely (breaking the 3-arg call sites) or keep it as a documented no-op is a maintainer call; only the misleading event emission is posted.
</content>
</invoke>
