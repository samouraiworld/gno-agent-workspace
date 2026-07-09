# PR [#5890](https://github.com/gnolang/gno/pull/5890): feat(gnovm): realm.Sub(subpath) sub-realm identities + NewBanker IsCurrent fix

URL: https://github.com/gnolang/gno/pull/5890
Author: jaekwon | Base: master | Files: 50 | +2632 -232
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 8a115c8ca (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5890 8a115c8ca`

**TL;DR:** Lets a realm mint an identity token for one of its internal actors (say a DAO in a registry) so downstream callees can tell that actor apart from the host realm and give it its own chain address, while a caller minting the token keeps its normal position in the call chain. Separately fixes a hole where a called realm could build a coin-spending banker over whoever called it and drain them.

**Verdict: APPROVE** — sub-realm minting is unforgeable (strict live-cur pointer identity plus a caller-namespace check plus a frozen subpath grammar), the banker caller-drain fix holds (reverting it reproduces the drain), and `unsafe.*` stays in agreement with the `cur` API across the split; no blocking concerns.

## Summary
Adds `cur.Sub(subpath)` to the `realm` interface: it returns a first-class realm value whose pkgpath is the synthesized `host:subpath` and whose address is `chain.PackageAddress(host:subpath)`. `:` is reserved (rejected at package validation), so a synthesized path can never collide with a deployed package and exact-match pkgpath auth is never silently broadened. The sub-token's `Previous()` equals the host's `Previous()` (the host is not inserted as a chain step); an internal parent reference anchors `IsCurrent`/`cross` staleness. Independently, `NewBanker` now requires `rlm.IsCurrent()`: without it a callee could pass `cur.Previous()` (address = caller) into a `RealmSend` banker and drain its caller through the `pkgAddr==from` gate. Also removes the legacy `cross1` migration sentinel across uverse/preprocess/transpiler/typechecker. Consensus-affecting: the `gno0p9` realm interface grows two methods (additive on a sealed interface) and `cross1` stops type-checking, so it rides a coordinated node upgrade.

## Examples
| Call | Result |
|---|---|
| `cur.PkgPath()` (host) | `gno.land/r/nt/commondao/v0` |
| `cur.Sub("dao/42").PkgPath()` | `gno.land/r/nt/commondao/v0:dao/42` |
| `cur.Sub("dao/42").Address()` | `chain.PackageAddress("gno.land/r/nt/commondao/v0:dao/42")` |
| `cur.Sub("dao/42").Previous()` | `cur.Previous()` (host not a chain step) |
| callee after `target.Foo(cross(sub), ...)`: `cur.Previous().Subpath()` | `dao/42` |
| `cur.Sub("Dao")` / `"a b"` / `"a:b"` / `".."` / `"dao/"` | panic (frozen grammar) |
| `cur.Sub("dao/42")` on an `/e/…/run` cur | panic (ephemeral cannot mint) |
| foreign realm calling `rlm.Sub("x")` on a passed-in cur | panic (caller not in receiver's namespace) |
| `sub.Sub("b")` (sub-of-sub) | panic (receiver pkgpath already synthesized) |

## Glossary
- **cur realm** — the interface handle a crossing function receives, carrying the live realm-context identity.
- **PreviousRealm / cur.Previous()** — the realm-context that cross-called into the current one.
- **IsCurrent** — true iff the realm value is the topmost live crossing frame's cur; the anti-spoof gate `cross(rlm)` also uses.
- **borrow rule #1 (declaring-realm borrow)** — invoking `/r/X`-declared code sets `m.Realm` to `/r/X` for the call, without shifting realm-context.
- **HIV** — the heap-item value backing a realm handle; identity is compared by pointer.

## Fix
Two independent changes. (1) `NewBanker` gains an `rlm.IsCurrent()` gate at [`banker.gno:101`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/stdlibs/chain/banker/banker.gno#L101) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L101) plus a sub-token banker-type restriction (RealmSend only) at [`banker.gno:110`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/stdlibs/chain/banker/banker.gno#L110) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L110). (2) `Sub` is a new `.grealm` native at [`uverse.go:1704`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/pkg/gnolang/uverse.go#L1704) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L1704) with a strict entry guard, backed by the reserved-`:` check in [`mempackage.go:1140`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/pkg/gnolang/mempackage.go#L1140) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/mempackage.go#L1140) and `unsafe.*` parity via [`PresentedRealmAt`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/pkg/gnolang/uverse.go#L628) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L628) read from [`execctx.GetRealm`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/stdlibs/internal/execctx/realm.go#L19) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/internal/execctx/realm.go#L19).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. Coverage is unusually complete: the accepting and rejecting sides of the subpath grammar are both pinned ([`zrealm_sub_validation.gno`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/tests/files/zrealm_sub_validation.gno) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/tests/files/zrealm_sub_validation.gno), [`zrealm_sub_derive_reject.gno`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/tests/files/zrealm_sub_derive_reject.gno) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/tests/files/zrealm_sub_derive_reject.gno)); foreign-mint rejection, stale-token recoverable vs `cross` fatal, persistence refusal, sub-of-sub, and the banker caller-drain are each a filetest or txtar; MsgCall and MsgRun entry both drive the `unsafe.*` parity assertion.

## Suggestions
None.

## Verified
- Reverting the `NewBanker` `IsCurrent` gate (rewriting the `if !rlm.IsCurrent()` panic at [`banker.gno:101`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/stdlibs/chain/banker/banker.gno#L101) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/banker/banker.gno#L101) to `if false && …`) flips `banker_security.txtar` TEST 6: the malicious realm's `Attack` tx succeeds (GAS USED shown, no panic) and moves the victim's coins, where the shipped test expects `banker can only be instantiated for the current realm`. Confirms the gate closes a real caller-drain, not a theoretical one. [repro](comment_claude-opus-4-8.md)
- Sub-token minting cannot be forged: to pass, the receiver HIV must be pointer-identical to the topmost crossing frame's Cur ([`uverse.go:1704`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/pkg/gnolang/uverse.go#L1704) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L1704) strict guard, deliberately not the relaxed `IsCurrent`) AND `m.Realm.Path` must equal the host pkgpath. A foreign realm holding a passed-around cur satisfies the first but fails the second under borrow rule #1 (`zrealm_sub_foreign.gno` green); a sub-token's own HIV is never a frame's Cur, so sub-of-sub fails the first (`zrealm_sub_nested.gno` green). Walked the guard combination against both.
- `unsafe.{Current,Previous}Realm()` agrees with `cur`/`cur.Previous()` for a sub identity presented across a real package boundary, and the `execctx.GetRealm` rewrite leaves ordinary (non-sub) cross results unchanged: the broad `zrealm_crossrealm*` filetest sweep is green with only golden hash/size shifts from the interface growing, no `unsafe.*` output divergence.
- Tests run green at 8a115c8ca: `zrealm_sub*`, `zrealm_banker_previous`, `zrealm_cross1_removed`, `zrealm_seal_realm` filetests; `TestIsValidSubpath`, `TestRealmLegacyThreeFieldShape`, `TestSubRealmPathErrorTotalCap`, `TestValidateMemPackageAny_ColonReserved`, `TestSubRealmGasMirrorsPackageAddress`, `TestVMKeeperRunSubEphemeral`; `banker_security` and `subrealm_run_parity` txtars.

## Open questions
- The subpath grammar (`isValidSubpath`/`isValidSubpathSegment`) is duplicated byte-for-byte between the Go native ([`uverse.go`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/pkg/gnolang/uverse.go#L491) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/pkg/gnolang/uverse.go#L491)) and the `.gno` mirror ([`address.gno`](https://github.com/gnolang/gno/blob/8a115c8ca/gnovm/stdlibs/chain/address.gno#L40) · [↗](../../../../../.worktrees/gno-review-5890/gnovm/stdlibs/chain/address.gno#L40)); the two are cross-checked only through filetests (`zrealm_sub_validation.gno` vs `zrealm_sub_derive_reject.gno` share a reject set), not a single Go-level equivalence test. Drift risk is low and mitigated; not worth a blocking or posted comment, noted for whoever touches the grammar next.
