# PR #5317: fix(example): add reverse index for MembersByTier

URL: https://github.com/gnolang/gno/pull/5317
Author: ltzmaxwell | Base: master | Files: 6 | +118 -74
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `44b92cb` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5317 44b92cb`

**Verdict: NEEDS DISCUSSION** — refactor is internally correct and tests pass, but the load-bearing question raised by [@ajnavarro](https://github.com/gnolang/gno/pull/5317#issuecomment-4192423955) ("is this even necessary for 3 tiers?") is unanswered; concurrently the value→pointer receiver change is a backwards-incompatible storage-shape break for the already-deployed `gnoland1` betanet.

## Summary

`MembersByTier` previously held a single `*avl.Tree` (tier → addr → member). `GetMember` and `RemoveMember` iterated every tier on every call. PR adds a reverse `addrToTier` index, switches all method receivers and the package-level `members` var (plus the three `proposalStatus` vote sets) from value-type `MembersByTier` to `*MembersByTier`, and rewrites `SetMember`/`GetMember`/`RemoveMember`/`DeleteAll` to use the new index. Tier count is fixed at 3 (T1/T2/T3), so the iteration cost the PR removes is O(3) — the trade is one extra `avl.Tree` of `address → tier name` strings duplicating roughly the member-count footprint.

## Glossary

- `MembersByTier` — wrapper around an AVL tree of `tier name → (address → *Member)`.
- `addrToTier` — new reverse-index AVL tree of `address → tier name`.
- `proposalStatus` — per-proposal vote-tally struct in `impl`, holds three `*MembersByTier` (Yes/No/All).
- `loader.gno` — package-level `init` that runs on every chain replay/genesis and sets up T1/T2/T3.

## Fix

`GetMember`/`RemoveMember` now do one lookup in `addrToTier` then one lookup in the matching tier subtree, instead of iterating all tiers. `SetMember` checks the reverse index for duplication before writing both trees. `DeleteAll` replaces both inner trees with fresh `avl.NewTree()` instances rather than iterating and removing entries from the live outer tree. Receiver type changed from value to pointer ([`types.gno:30-42`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L30-L42) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L30-L42)) so the new `addrToTier` field stays addressable across calls; this propagates to `Tier.{MaxSize,MinSize,PowerHandler}` callbacks, `Get()`, `GetTierPower`, and `proposalStatus.{Yes,No,All}Votes`.

## Critical (must fix)

None.

## Warnings (should fix)

- **[necessity not justified vs the @ajnavarro comment]** [@ajnavarro](https://github.com/gnolang/gno/pull/5317#issuecomment-4192423955) [`memberstore.gno:21-25`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L21-L25) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L21-L25) — author has not replied to the "is this even necessary" question; the PR doubles storage for a sub-millisecond optimization on a fixed N=3.
  <details><summary>details</summary>

  Tier set is hard-coded to three values (`T1`/`T2`/`T3`) in [`memberstore.gno:21-25`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L21-L25) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L21-L25) and seeded once by `loader.gno` and `init.gno`. Adding/removing tiers requires a governance proposal (`setTiers` is package-private) — there is no foreseeable path to N>>3. The pre-PR `GetMember` cost was at most 3 tier lookups + 3 subtree `Get`s; the post-PR cost is 1 lookup in `addrToTier` + 1 in the matching tier. In exchange, every `SetMember`/`RemoveMember` now writes/deletes in two trees, and storage for the reverse index roughly equals one address-string per member. For a realm whose total membership is governance-bounded (T1 minimum 70, soft cap on T2/T3 derived from T1), the storage cost dominates the savings. Fix: either justify the change with a concrete profile/benchmark showing a real hotspot, or close the PR. If kept, add a one-line ADR explaining why the reverse index pays for itself.
  </details>

- **[storage-shape break for already-deployed betanet]** [`types.gno:30-33`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L30-L33) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L30-L33) — `MembersByTier` gains a field and changes from value to pointer; the v3 govDAO package is listed in [`misc/deployments/gnoland1/packages.gen.txt:56`](https://github.com/gnolang/gno/blob/44b92cb/misc/deployments/gnoland1/packages.gen.txt#L56) · [↗](../../../../../.worktrees/gno-review-5317/misc/deployments/gnoland1/packages.gen.txt#L56), so a chain that already persisted state under the old shape cannot replay.
  <details><summary>details</summary>

  The package `gno.land/r/gov/dao/v3/memberstore` is deployed to `gnoland1` (betanet) via genesis (see [`misc/deployments/gnoland1/govdao_prop1.gno:25`](https://github.com/gnolang/gno/blob/44b92cb/misc/deployments/gnoland1/govdao_prop1.gno#L25) · [↗](../../../../../.worktrees/gno-review-5317/misc/deployments/gnoland1/govdao_prop1.gno#L25) which calls `memberstore.Get().SetMember(...)`). Gno realm storage is type-shape sensitive: any chain that booted on the pre-PR struct and persisted a `MembersByTier{Tree: ...}` value cannot deserialize the new `*MembersByTier{Tree: ..., addrToTier: ...}`. The PR also changes `proposalStatus.{Yes,No,All}Votes` from value to pointer, which would break any persisted proposal status the same way. This is fine if `gnoland1` is still treated as a redeployable testnet (genesis regenerated on every chain restart), but the PR doesn't say so and there's no migration shim. Fix: confirm with maintainers that no live state needs to be migrated, and add a one-line note to the PR body. If state must survive, the PR cannot land as-is.
  </details>

- **[error-precedence flipped]** [`types.gno:71-77`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L71-L77) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L71-L77) — `SetMember` now reports "tier does not exist" before "member already exists"; pre-PR order was reversed.
  <details><summary>details</summary>

  Pre-PR ([git history](https://github.com/gnolang/gno/pull/5317/files)): `SetMember` called `GetMember(addr)` first, returning `&ErrMemberAlreadyExists{Tier: t}` if the address was anywhere; only if no duplicate did it check `mbt.Has(tier)`. Post-PR: tier-existence check comes first. Net effect: a caller passing an invalid tier on a duplicate address now sees the tier error instead of the typed `ErrMemberAlreadyExists`. No caller in `impl/` currently inspects this distinction (`NewAddMemberRequest` validates the tier with `GetTier` before calling `SetMember` at [`impl/prop_requests.gno:57-64`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L57-L64) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L57-L64)), so no consumer breaks today, but the typed error contract changed silently. Fix: either restore the old order (check duplicate first) for behavioural parity, or note the change in the PR body so downstream realms importing `memberstore.ErrMemberAlreadyExists` don't get caught.
  </details>

## Nits

- [`types.gno:31-32`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L31-L32) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L31-L32) — embedded `*avl.Tree` is still unnamed; the field comment on line 32 (`addrToTier`) reads cleanly because it has a name, but the embedded tree on line 31 has only the comment `tier name -> address -> member`. Giving it a name (`byTier *avl.Tree`) and dropping the embedding would make every site that calls `mbt.Has`/`mbt.Get`/`mbt.Set`/`mbt.Iterate` explicit about which tree it touches — currently the reader has to remember "embedded one is the tier tree, named one is the reverse index."
- [`types.gno:109`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L109) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L109) — comment "removes a member from any tier" is now stale; with the reverse index there's only ever one tier to remove from. Update to "removes a member by address."
- [`types.gno:101`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L101) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L101), [`types.gno:119`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L119) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L119) — the two "inconsistent state" panics are good defensive checks, but the message could include the address to ease debugging if it ever fires.

## Missing Tests

- **[no test for nil-receiver guard]** [`types.gno:35-37`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L35-L37) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L35-L37) — `addrToTier` is only initialised via `NewMembersByTier`; if a future caller forgets and zero-values a `MembersByTier`, every method nil-panics.
  <details><summary>details</summary>

  Pre-PR the zero value `MembersByTier{}` was usable (embedded pointer was nil but methods worked through `Has`/`Get` which themselves nil-guard). Post-PR, `SetMember` calls `mbt.addrToTier.Get(...)` which dereferences `nil`. Only one constructor exists today ([`types.gno:35`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L35) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L35)) and every caller in `examples/gno.land/r/gov/dao/v3/` uses it, but the type is exported and a downstream realm composing `proposalStatus`-like structs by hand would hit this. Fix: either add a defensive `if mbt.addrToTier == nil { mbt.addrToTier = avl.NewTree() }` lazy init in `SetMember`/`GetMember`/`RemoveMember`, or document loud-and-clear that `NewMembersByTier` is required.
  </details>

- **[no test for `DeleteAll` invariant]** [`types.gno:39-42`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L39-L42) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L39-L42) — `DeleteAll` replaces both trees but there's no test that exercises insert → DeleteAll → insert-same-addr to confirm the reverse index is genuinely cleared.

## Suggestions

- [`types.gno:69-86`](https://github.com/gnolang/gno/blob/44b92cb/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L69-L86) · [↗](../../../../../.worktrees/gno-review-5317/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L69-L86) — if the reverse index stays, consider folding the second [@davd-gzl](https://github.com/gnolang/gno/pull/5317#discussion_r2960777297) suggestion: store only `addrToTier` and derive tier sizes by counting, instead of keeping the tier→addr→member tree at all. The current design stores every address twice. Out of scope for this PR but worth a follow-up.

## Questions for Author

- Reply to [@ajnavarro](https://github.com/gnolang/gno/pull/5317#issuecomment-4192423955): what hot path motivated this? With N=3 tiers, the iteration was already O(1)-ish — is there profiling data?
- Is `gnoland1` betanet expected to genesis-replay (so the storage-shape change is harmless), or is there persisted v3 govDAO state that needs a migration plan?
- Was the swap of error precedence in `SetMember` (tier-not-exist vs. duplicate) intentional?
