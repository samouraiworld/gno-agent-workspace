# PR [#6027](https://github.com/gnolang/gno/pull/6027): fix(grc20reg): key token registrations by slug aliases

URL: https://github.com/gnolang/gno/pull/6027
Author: notJoon | Base: master | Files: 12 | +221 -104
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 854b03529 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6027 854b03529`

**TL;DR:** `grc20reg` is the on-chain directory where a realm publishes a GRC20 token under a string key so other realms can find it. Today that key is built from the token's symbol, so one realm can never publish two tokens sharing a symbol. This PR makes the key a caller-chosen name instead, so a realm can publish several same-symbol tokens and can publish one token under several names.

**Verdict: NEEDS DISCUSSION** — the change is correct and green, but it removes the last guard that kept every registered token's `Token.ID()` distinct, and PR [#6028](https://github.com/gnolang/gno/pull/6028) rewrites the same function with a different event schema and a different fix for that exact hole; the merge order is a maintainer call (1 Warning, 1 Missing test, 3 Nits, 2 Suggestions).

## Verify first
- [`grc20reg.gno:47-50`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L47-L50) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L47) — the origin check must keep building its prefix from `token.GetSymbol()`, never from `slug`; if it took the slug the caller would pick both sides of the comparison. Read the two lines, then run `go run ../gnovm/cmd/gno test ./gno.land/r/demo/defi/grc20reg/` from `examples/` and confirm `TestRegisterRejectsTokenFromDifferentRealm` still passes.
- [`grc20reg.gno:51`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51) — `registry.Has(key)` is now the whole duplicate guard, and it only sees slugs. Drop [`tests/duplicate_token_id_filetest.gno`](tests/duplicate_token_id_filetest.gno) into `examples/gno.land/r/demo/defi/grc20reg/filetests/` and run the package: two tokens carrying one `Token.ID()` both land in the registry.
- [`grc20reg.gno:61`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61) — this PR keeps the register event's key named `token_path` and adds `token_id`; PR [#6028](https://github.com/gnolang/gno/pull/6028) renames the same field to `token_key`. Diff the two `chain.Emit` blocks and settle the schema once, so indexers see one break instead of two.

## Summary
`grc20reg.Register` built its key from `fqname.Construct(rlmPath, token.GetSymbol())`, so a realm got one registry slot per symbol and the caller-supplied `slug` was validated and emitted but discarded. That is issue [#5988](https://github.com/gnolang/gno/issues/5988): a bridge realm minting two distinct tokens that both call themselves `USDC` can register only one. The PR makes `slug` mandatory, keys on `fqname.Construct(rlmPath, slug)`, and keeps the symbol only inside the token-origin prefix check, so the realm still has to own the token it registers. It also adds the `cur.IsCurrent()` guard the interrealm docs prescribe, adds `token_id` to the register event, and swaps the deprecated `md.EscapeText` for `sanitize.InlineText` in `Render`. No in-tree registry key actually moves: every updated call site passes its own symbol as the slug, so `foo20.FOO`, `wugnot.wugnot`, `test20.TST` and `bar20.BAR` all keep their old keys.

Reading order: [`grc20reg.gno`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno), then [`grc20reg_test.gno`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno) and [`filetests/issue_5988_filetest.gno`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/filetests/issue_5988_filetest.gno) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/filetests/issue_5988_filetest.gno), then the four one-line call-site updates in `foo20`, `wugnot`, `test20`, `bar20`, then the two doc-comment files in `p/demo/tokens/grc20`, then the quarantined `tokenhub` pair.

## Diagram

```
                        before (master)                     after (854b03529)

Register(cur, token, slug)
  slug ................. validated, then discarded          required, becomes the key
  key .................. rlmPath "." token.GetSymbol()      rlmPath "." slug
  origin check ......... HasPrefix(token.ID(), key+".")     HasPrefix(token.ID(),
                                                              rlmPath "." symbol ".")
  duplicate guard ...... registry.Has(key)                  registry.Has(key)
                           key carries the symbol,            key carries only the slug,
                           so it also rejects a second        so a second token with the
                           token with the same symbol   -->   same Token.ID() gets in
                           and the same Token.ID()

  event ................ token_path = rlmPath.symbol        token_path = rlmPath.slug
                                                            token_id   = token.ID()
```

## Examples

| `Register(cross(cur), Token, slug)` | before | after |
|---|---|---|
| `slug = "FOO"`, symbol `FOO` | `…/foo20.FOO` | `…/foo20.FOO` |
| `slug = ""` | `…/foo20.FOO` | panics `empty slug` |
| `slug = "mySlug"`, symbol `TST` | `…/foo.TST` | `…/foo.mySlug` |
| same token, second call, new slug | panics `token already registered` | second key created |
| two same-symbol tokens, distinct ids, two slugs | panics `token already registered` | both registered |
| two tokens sharing one `Token.ID()`, two slugs | panics `token already registered` | both registered |

## Fix
The key derivation moves from the token to the caller: [`grc20reg.gno:46`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L46) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L46) builds `rlmPath.slug`, and the symbol survives only in the separately constructed origin prefix at [`grc20reg.gno:47-50`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L47-L50) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L47). Splitting the two lets a realm publish several same-symbol tokens while still being unable to publish another realm's token. The load-bearing constraint is that both the symbol charset in [`token.gno:79-91`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L79-L91) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/p/demo/tokens/grc20/token.gno#L79) and the slug charset in [`grc20reg.gno:126-136`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L126-L136) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L126) exclude `.` and `/`, so `fqname.Parse` splits the key back into exactly the realm path and the slug that built it.

## Critical (must fix)
None.

## Warnings (should fix)
- **[a guard that used to hold now only holds by convention]** [`grc20reg.gno:51`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51) — two tokens carrying a byte-identical `Token.ID()` can now both be registered, under two slugs, so the id in a `Transfer` or `Mint` event no longer picks out one registry entry.
  <details><summary>details</summary>

  `Token.ID()` is `rlmPath.symbol.<seqid>` and the seqid comes from the caller ([`token.gno:53`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/demo/tokens/grc20/token.gno#L53) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/p/demo/tokens/grc20/token.gno#L53)), so a realm that passes `0` twice mints two objects sharing one id. On the merge base that was unreachable in the registry: the key carried the symbol, so the second registration hit `token already registered`. Now the key carries only the slug, and `registry.Has` never looks at the token. The register event does emit `token_id` ([`grc20reg.gno:61`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61)), but `Transfer`, `Approval`, `Mint` and `Burn` carry the id alone, so a consumer that maps an id back to a registry entry now finds two. That is issue [#6026](https://github.com/gnolang/gno/issues/6026), and PR [#6028](https://github.com/gnolang/gno/pull/6028) closes it by making the id generator-issued and checking the issuer in `Register`. Confirmed with [`tests/duplicate_token_id_filetest.gno`](tests/duplicate_token_id_filetest.gno): it passes at merge-base d1a33f574 and fails at 854b03529, where both tokens register and the run prints `both registered under one token id: true First Second`. Fix: reject a token whose id already sits in the registry, or land [#6028](https://github.com/gnolang/gno/pull/6028)'s issuer check first so the collision cannot be minted.
  </details>

## Nits
- **[an event field that no longer means what it is named]** [`grc20reg.gno:57`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L57) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L57) — `token_path` now holds the alias, and an indexer that assumed `HasPrefix(token.ID(), token_path+".")` loses that invariant silently; PR [#6028](https://github.com/gnolang/gno/pull/6028) renames the same field `token_key`.
  <details><summary>details</summary>

  Before the change, `token_path` was `rlmPath.symbol`, a genuine prefix of `Token.ID()`. It is now `rlmPath.slug`, which shares only the realm path with the id. The new `token_id` attribute at [`grc20reg.gno:61`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L61) gives a correct consumer everything it needs, so nothing breaks that reads both fields; the cost is a stale name plus a second rename when [#6028](https://github.com/gnolang/gno/pull/6028) lands. Fix: pick one name across both PRs.
  </details>
- **[a rendered link that goes nowhere]** [`grc20reg.gno:93`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L93) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L93) — the home listing builds every "info" link as `/r/demo/grc20reg:<key>`, but the realm's own module path is `gno.land/r/demo/defi/grc20reg`, so each link 404s.
  <details><summary>details</summary>

  The realm declares `module = "gno.land/r/demo/defi/grc20reg"` ([`gnomod.toml:1`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/gnomod.toml#L1) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/gnomod.toml#L1)) and no `examples/gno.land/r/demo/grc20reg/` directory exists, so the `defi` segment is missing from the link. The defect predates the branch, but the diff rewrites this exact line and updates the golden that pins the wrong path at [`grc20reg_test.gno:22`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L22) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L22), so it costs one segment to fix here. Fix: build the prefix from the realm's own path.
  </details>
- **[a test whose name promises more than it checks]** [`grc20reg_test.gno:96-103`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L96-L103) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L96) — `TestRegisterPanicsOnInvalidSlug` only exercises the length branch, never the character branch. `TestValidateSlug` covers the characters against the helper directly, so the gap is cosmetic; not posted, no change needed.

## Missing Tests
- **[the property aliasing rests on is never asserted]** [`grc20reg_test.gno:73-74`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73-L74) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73) — `TestRegisterAllowsMultipleAliases` compares `Token.ID()` strings, which two separate objects would also satisfy, so nothing pins that both aliases resolve to one object.
  <details><summary>details</summary>

  Aliasing stores one `*grc20.Token` pointer under several keys, and both entries share the single `*PrivateLedger` behind it. If persistence ever split that object, each alias would carry its own balances while the id comparison in the current test still passed. A pointer comparison plus one balance read closes it in two lines. Verified both directions at 854b03529: [`tests/alias_identity_test.gno`](tests/alias_identity_test.gno) passes in-process, and [`tests/grc20reg_alias_identity.txtar`](tests/grc20reg_alias_identity.txtar) mints through one alias in one transaction and reads `same=true a=100 b=100` back through both in the next.
  </details>

## Suggestions
- **[a guard that cannot fire, kept for the shape]** [`grc20reg.gno:36`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36) — the author's own thread marks the new `cur.IsCurrent()` check "unnecessary" and the VM agrees: `cross` rejects a stale realm value first, so nothing reaches this line with one.
  <details><summary>details</summary>

  Every cross-realm caller must write `Register(cross(cur), …)`, and the `cross` builtin runs the same predicate `IsCurrent` runs, panicking `cross: rlm is not the current cur` before the callee's body starts ([`uverse.go:1836-1838`](https://github.com/gnolang/gno/blob/854b03529/gnovm/pkg/gnolang/uverse.go#L1836-L1838) · [↗](../../../../../.worktrees/gno-review-6027/gnovm/pkg/gnolang/uverse.go#L1836)); the VM's own comment states no second check is needed downstream ([`uverse.go:1816-1817`](https://github.com/gnolang/gno/blob/854b03529/gnovm/pkg/gnolang/uverse.go#L1816-L1817) · [↗](../../../../../.worktrees/gno-review-6027/gnovm/pkg/gnolang/uverse.go#L1816)). Demonstrated with [`tests/stale_cur_rejected_filetest.gno`](tests/stale_cur_rejected_filetest.gno), which captures a cur from a returned sibling frame: the abort comes from `cross`, never from `grc20reg`. The only path that skips `cross` is a same-realm call from inside `grc20reg` itself, and none exists. Removing the three lines leaves the suite green, so nothing in CI defends them either. Against that, [`gno-ai-contract-review.md` §1](https://github.com/gnolang/gno/blob/854b03529/docs/resources/gno-ai-contract-review.md?plain=1#L12) prescribes the guard before every `cur.Previous()` read and it costs one comparison. Fix: keep it as documented defense in depth, or drop it as dead code, but decide rather than leave the thread open.
  </details>
- **[one token can now occupy the directory many times]** [`grc20reg.gno:85-96`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L85-L96) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L85) — a realm may register the same token under unbounded aliases, and the home `Render` still iterates the whole tree with pagination left as a TODO, so the listing can show one token repeatedly. Each alias costs the registrant a storage deposit, so it is self-limiting; not posted.

## Colliding work: PR [#6028](https://github.com/gnolang/gno/pull/6028)
[#6028](https://github.com/gnolang/gno/pull/6028) rewrites the same `Register` body and nine of this PR's twelve files, so a textual conflict is certain. The semantic differences a merge has to settle:

| | this PR | [#6028](https://github.com/gnolang/gno/pull/6028) |
|---|---|---|
| key | `rlmPath.slug`, slug required | `rlmPath.slug`, slug optional (empty key falls back to `rlmPath`) |
| register event | `token_path` + new `token_id` | `token_key` + `token_id` |
| origin check | prefix `rlmPath.symbol.` | prefix `rlmPath.` |
| id uniqueness | left to the registering realm | generator-issued id, issuer verified in `Register` |
| `cur.IsCurrent()` | added | absent |
| `grc20.NewToken` | signature unchanged | takes `*identifier.Generator` instead of `seqid.ID` |

[#6028](https://github.com/gnolang/gno/pull/6028) already contains this PR's headline change, slug-keyed aliasing, and additionally closes the duplicate-id hole this PR opens. Landing [#6028](https://github.com/gnolang/gno/pull/6028) first leaves this PR with a much smaller remainder: required slug, `cur.IsCurrent()`, the `sanitize.InlineText` swap, the ADR, and the `filetests/issue_5988_filetest.gno` golden, which needs regenerating because its hardcoded ids (`…DUP.0000001`) become generator codes. Landing this PR first means [#6028](https://github.com/gnolang/gno/pull/6028) reworks its own `Register` again and ships a second event-schema rename in the same release, and leaves a window in which the registry accepts colliding ids.

Neither order is free, and the two costs are not the same kind. The remainder cost is rebase work a maintainer can see and price. The window cost is issue [#6026](https://github.com/gnolang/gno/issues/6026) reachable in the registry for however long the two PRs are apart, and [#6028](https://github.com/gnolang/gno/pull/6028) carries six Warnings of its own, so that gap is not obviously short. Readiness points the other way: this PR is one Warning from mergeable, and its Warning is exactly what [#6028](https://github.com/gnolang/gno/pull/6028) fixes.

A third order costs neither. This PR keeps its slug key and adds a duplicate-id rejection of its own, an id-to-key index checked in `Register`, which closes issue [#5988](https://github.com/gnolang/gno/issues/5988) without widening [#6026](https://github.com/gnolang/gno/issues/6026). [#6028](https://github.com/gnolang/gno/pull/6028) then replaces that check with the generator, code it is rewriting anyway. The constraint, whichever order wins, is that the symbol leaves the key only once something else keeps two byte-identical `Token.ID()` values out of the registry.

## Verified
- Aliases survive persistence as one object: [`tests/grc20reg_alias_identity.txtar`](tests/grc20reg_alias_identity.txtar) registers one token under `a` and `b`, mints 100 through the realm's ledger in one transaction, and reads `same=true a=100 b=100` back through both aliases in the next. No committed test crosses a transaction boundary on the new alias path.
- The duplicate-id delta belongs to this diff, not to master: [`tests/duplicate_token_id_filetest.gno`](tests/duplicate_token_id_filetest.gno) passes at merge-base d1a33f574 (`token already registered`) and fails at 854b03529, where both registrations succeed.
- No in-tree registry key moves. Every updated call site passes its own symbol as the slug, so the integration test that hardcodes `gno.land/r/demo/defi/foo20.FOO` ([`grc20_registry_emit.txtar:14`](https://github.com/gnolang/gno/blob/854b03529/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14) · [↗](../../../../../.worktrees/gno-review-6027/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L14)) still passes untouched.
- The `sanitize.InlineText` swap changes no rendered byte: `md.EscapeText` on master already delegates to `sanitize.InlineText` ([`md.gno:414-416`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/p/moul/md/md.gno#L414-L416) · [↗](../../../../../.worktrees/gno-review-6027/examples/gno.land/p/moul/md/md.gno#L414)), and the unchanged `Render` goldens confirm it. It is a deprecation cleanup, not a hardening.
- The new `cur.IsCurrent()` block is unreachable: [`tests/stale_cur_rejected_filetest.gno`](tests/stale_cur_rejected_filetest.gno) hands `Register` a cur captured from a returned sibling frame and the abort comes from `cross`, before `Register`'s body runs. Deleting the three lines also leaves the whole `grc20reg` suite green, so no test defends them.
- Suites green at 854b03529 from the PR worktree: `grc20reg` (unit plus the new filetest), `p/demo/tokens/grc20`, `foo20`, `wugnot`, `atomicswap`, `grc20factory`, `gov/dao/v3/treasury/test`, and `TestTestdata/grc20_registry_emit`. `gno lint` is clean over `grc20reg`, `grc20`, `foo20` and `wugnot`; sanity-checked by renaming `token.GetSymbol` to a nonexistent method and seeing three type-check errors.

## Existing threads
- notJoon, on his own PR, [`grc20reg.gno:38`](https://github.com/gnolang/gno/pull/6027#discussion_r3689218181): "unnecessary", pointing at the new `cur.IsCurrent()` guard. Unresolved. The guard Suggestion above confirms it: the check cannot fire, and the open decision is whether to keep it for the documented shape.

## Open questions
- `validateSlug` now returns an error but accepts `""`, and `Register` catches the empty case separately at [`grc20reg.gno:39-41`](https://github.com/gnolang/gno/blob/854b03529/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39-L41). A second caller of the helper would not inherit that check. Only one caller exists today; not posted.
- `Get` and `MustGet` return the stored `*grc20.Token` pointer, which the contract-review checklist treats as a mutator handle when the type has exported write methods. `Token`'s writes all sit behind `PrivateLedger` and the teller types, so no handle leaks, and the shape predates this branch. Not posted.
- CI is fully green; the red `Merge Requirements` status is the bot waiting for a review-team approval, not a code problem.
