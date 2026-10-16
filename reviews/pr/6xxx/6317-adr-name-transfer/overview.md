# `TransferTo`, `OfferTransfer` and `effectiveOwner`: moving a registered name to a new address

PR: [gnolang/gno#6317](https://github.com/gnolang/gno/pull/6317)
Review files: [reviews/pr/6xxx/6317-adr-name-transfer](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6317-adr-name-transfer)

## TLDR

Today a record's
[`UserData.addr`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L38-L42) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L38)
never changes, and a live private package is replaced only by its stamped
creator. The ADR, a design with no code, moves a whole record to a fresh key
through `OfferTransfer` and `AcceptTransfer` or a T1 vote, and has the keeper
ask `NamespaceOwner` who owns existing packages instead of reading the stamp,
all dormant until an activation transaction at a genesis-replay upgrade. It
exists so a rotated, moved or lost key does not cost the name and its private
realms. The author now proposes narrowing it to package ownership and
governance recovery, beside the teams design in
[#6341](https://github.com/gnolang/gno/pull/6341).

## Why one fixed address is not enough

Three questions resolve to the same address today, and that address never
changes. The author's own split in the
[discussion](https://github.com/gnolang/gno/pull/6317#issuecomment-6184871940)
names them:

| Question | Answered today by |
|---|---|
| Who is `@alice`? | the record's `addr`, set once by [`registerUser`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L81) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L81) |
| Who may deploy new paths under `r/alice/...`? | the holder of the **current** name, through [`resolveCurrentName`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/names/verifier.gno#L102-L108) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/names/verifier.gno#L102) |
| Who may replace a live private package? | the stamped creator **and** the namespace check, in [`checkRedeployPermission`](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L723-L728) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L723) |

Each existing escape route closes. A rename
[adds the new name as one more key](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L213-L214) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L213)
pointing at the same record, so the address stays. Deleting
[only sets a flag](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L273) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L273)
and leaves every name key in place, so registering the name again from a new
key
[fails with `ErrNameTaken`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L93-L95) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L93),
and the deleted address itself
[gets `ErrDeletedUser`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L106-L109) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L106).

A rename also costs the user their own private packages. The verifier
[refuses an alias](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/names/verifier.gno#L104) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/names/verifier.gno#L104),
and the keeper
[runs that namespace check](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L1137) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L1137)
on a private redeploy as well as the creator check, so after renaming
`alice` to `alice2` nobody can redeploy `r/alice/home`, its creator included.
The ADR records this in its
[Context](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L39-L41) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L39).
The creator binding came in with
[#6171](https://github.com/gnolang/gno/pull/6171), which is why the
discussion calls the keeper half of this ADR a reversal of that PR.

## How a transfer works, in 4 steps

The example is a user whose record has `id` 42, `rev` 3, current name
`alice2` and alias `alice`, held by `g1old`, moving to a fresh key `g1new`.

1. **`g1old` offers.**
   [`OfferTransfer(cur, g1new)`](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L215-L221) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L215)
   in `r/sys/namereg/v0` checks the registry is not paused and `g1new` has no
   entry in `addressStore`, live or deleted. It stores
   `pendingTransfers[42] = {to: g1new, rev: 3, expiresAt: now + 604800}`.
   The call must come straight from an account and must not be
   [signed with a session key](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L206-L212) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L206),
   since a session whose `AllowPaths` covers namereg could otherwise give the
   name away.
2. **`g1new` accepts.**
   [`AcceptTransfer(cur, 42)`](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L224-L231) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L224)
   runs before `expiresAt` and checks that `ResolveID(42).Rev()` is still 3.
   Any rename, delete or transfer since the offer has raised `rev`, so it
   kills the offer. Accepting is a transaction signed by `g1new`, which proves
   the recipient holds the key: a mistyped address cannot swallow a name.
3. **`r/sys/users` moves the record.** namereg calls
   [`TransferTo`](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L110-L123) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L110),
   which removes the `g1old` key, sets `u.addr = g1new`, writes the `g1new`
   key, raises `rev` to 4 and emits `Transferred(42, "alice2", g1old, g1new)`.
   The keys `alice` and `alice2` already point at the record, so both now
   resolve to `g1new` with no alias walk. `g1old` is a fresh address again and
   may register a new name under a new `id`.
4. **The keeper follows the record.** `g1new` may deploy new paths under
   `alice2`, today's rule. Redeploying the private `r/alice/home`, an alias
   path still stamped to `g1old`, now asks `NamespaceOwner("alice")`, gets
   `(g1new, "owned", false)`, and accepts `g1new` while refusing `g1old`.

Where the old key is lost or stolen, steps 1 and 2 are replaced by
[`ProposeTransferUser`](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L126-L141) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L126),
a GovDAO proposal that only T1 members vote on. It binds to the `id` and not
to `rev`, because a thief holding the key could bounce the record between
their own addresses and raise `rev` on every hop to void the vote.

## The parts, at a glance

All rows are proposed. The second column links the ADR section;
the third names the current code each part changes, on master.

| Part | Proposed in | Changes | Job |
|---|---|---|---|
| `id`, `rev` and two side B+ trees | [§1](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L75-L91) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L75) | [`users/store.gno`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L13-L14) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L13) | Gives each record a permanent number and a counter every mutation raises, kept out of `UserData` so replayed registrations store nothing new. |
| Authoritative-record guard | [§1](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L92-L109) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L92) | [`makeUserDataSafe`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/users.gno#L56-L66) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/users.gno#L56) callers | Refuses a rename, delete or transfer on a copy handed out by the read-only trees, by requiring `addressStore.Get(u.addr) == u`. |
| `TransferTo`, `IsAddressEligible` | [§1](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L110-L125) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L110) | `r/sys/users` | Moves the record for a whitelisted controller, as [`UpdateName` already authorizes `rlm.Address()`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/store.gno#L241) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/store.gno#L241). |
| `ProposeTransferUser`, id-bound proposals | [§1](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L126-L159) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L126) | [`ProposeNewName`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/namereg/v0/admin.gno#L62) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/namereg/v0/admin.gno#L62) and the users proposals | Recovery by T1 vote; every rename and delete proposal resolves its target to an `id` when created, so an address reused after a transfer is never hit. |
| T1 filter on controller proposals | [§2](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L170-L189) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L170) | [`users/admin.gno`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/users/admin.gno#L40) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/users/admin.gno#L40) | Narrows who votes on adding a controller, since controllers now decide who holds private packages. |
| Offer, accept, cancel, prune | [§3](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L191-L233) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L191) | `r/sys/namereg/v0` | The user-led path: one offer per record in a B+ tree keyed by `id`, 7-day expiry, no fee. |
| `NamespaceOwner` | [§4](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L245-L264) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L245) | [`verifier.gno`](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/names/verifier.gno#L75-L77) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/names/verifier.gno#L75) | A strict query returning owner, status and the pause flag in one call. |
| `effectiveOwner` | [§5](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L266-L386) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L266) | `keeper.go`, `keeper_inert.go` | One resolver behind every ownership-gated keeper decision. |
| Activation transaction | [§6](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L411-L430) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L411) | the upgrade's genesis file | Assigns ids, writes reservations, then sets the flag every new behaviour waits on. |
| `PkgCreator` on `MsgEnablePackage` | [Consequences](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L610-L645) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L610) | `vm.proto`, `pb3_gen.go`, `contribs/gpao` | Lets an approval pin who the parked bytes run `init()` as, since content hashes exclude `[addpkg]`. |

## How effective ownership works in the keeper

**One resolver.** Today each operation compares addresses its own way: the
stamp compare in
[`checkRedeployPermission`](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L724) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L724),
the parked-submitter compare in the
[park guard](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L956-L966) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L956),
and
[`isLivePackageOwner`](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper_inert.go#L665-L675) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper_inert.go#L665)
for rejection. The ADR routes all of them through
[`effectiveOwner`](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L272-L278) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L272),
which calls `NamespaceOwner` and maps its status to an owner. The mapping,
from the ADR:

| `NamespaceOwner` status | Effective owner |
|---|---|
| `owned` | the record's address, for the current name and every alias |
| `deleted` | none, so owner-gated writes are frozen |
| `unregistered` | none, as today |
| `address` | the address, when nothing is live or it is also the stamped creator |
| `disabled`, or names not deployed | the stamped creator, as today |
| the call errors | none: an error never falls back to the creator |

**Replacement drops the stamp.** On a live private package the submitter must
be the effective owner, and that test
[replaces both](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L291-L305) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L291)
the creator check and the namespace check, on the
[normal path](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L822-L830) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L822),
the inert park path and
[`EnablePackage`](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper_inert.go#L257-L261) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper_inert.go#L257).
Keeping the creator check as an extra condition would stop every new owner,
and keeping it as an alternative would let a stolen old key still replace
code, which the ADR rejects in its
[alternatives](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L495-L498) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L495).
The stamp itself is never rewritten: it is also the identity `init()` runs
as, and a redeploy by the new owner carries the new owner's stamp.

**Pause moves into the keeper.** Pause lives
[inside the verifier today](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/sys/names/verifier.gno#L231-L233) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/sys/names/verifier.gno#L231),
so dropping the namespace call on a replacement would drop pause with it.
`NamespaceOwner` returns the flag, and the keeper
[refuses every submission, park and activation](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L306-L311) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L306)
while names is enabled and paused.

**Rejection checks the submitter and approver first.** The parked submitter
and an approver
[reject without calling the resolver](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L326-L337) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L326),
as they do in
[`RejectPackage` today](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper_inert.go#L478-L485) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper_inert.go#L478).
Only a sender relying on owner standing reaches the resolver. Under `owned`
the effective owner replaces the live stamped creator, so the old key loses
rejection too; under every other status the stamped creator keeps it, since
rejection is cleanup.

**Replay skips it.** During `IsGenesisReplay`
[the resolver and every new gate are skipped](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L359-L370) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L359),
the same way
[the creator check is waived today](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L822) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L822),
because a replay reproduces transactions the source chain already accepted.

## How activation works

**Why an upgrade.** The sys realms are public, so they cannot be replaced in
place, and `r/sys/users` holds the registry under an unversioned path. The
change therefore
[ships as a chain upgrade by genesis replay](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L390-L394) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L390),
in which the fork tool swaps the realms' source inside their original genesis
transactions. All of history then runs against the new realm code.

**Dormant until a flag.** Ids, the guard, id-bound proposals, the T1 filters
and transfers all stay
[switched off](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L396-L410) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L396)
until one flag is set. Without that, strict replay fails: a historical
controller proposal passed by T2 and T3 votes would meet a T1 filter that did
not exist when it ran.

**One transaction sets it.** The
[activation transaction](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L411-L430) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L411)
is appended after all replayed history, at height 0, and refuses a second
run. In order it repairs records split by earlier mutations of copies,
assigns ids in raw `addressStore` key order, writes the agreed name
reservations, checks both indexes, and sets the flag last. Before it runs,
every open address-bound rename or delete proposal must be closed, or one
executed after the address is reused would hit the wrong record.

**The keeper goes live at boot.** The keeper rule applies as soon as the
upgraded chain starts, while transfers wait for the flag, so a namespace is
never moved before someone can redeploy its packages.

**Ids stay fixed afterwards.** Once ids appear in transactions, a later
replay upgrade must not renumber them, so new reservations go after history.
The fork tool also has to
[keep inherited `PastChainIDs`](https://github.com/clockworkgr/gno/blob/cffd989cd/gno.land/adr/pr6317_name_transfer.md?plain=1#L447-L454) · [↗](../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L447),
which `buildHardforkGenesis`
[replaces with the immediate predecessor](https://github.com/gnolang/gno/blob/24279be63/contribs/gnogenesis/internal/fork/generate.go#L456) · [↗](../../../../.worktrees/gno-review-6317-base/contribs/gnogenesis/internal/fork/generate.go#L456)
today.

## Before and after

**Before:** a record's address is fixed, new paths go to the current name's
holder through `IsAuthorizedAddressForNamespace`, and a private package is
replaced only by its stamped creator holding the current name, through
`checkRedeployPermission` and `checkNamespacePermission`.

**After, as proposed:** a record moves to a fresh address through
`OfferTransfer` and `AcceptTransfer` or a T1 vote, new paths still go to the
current name's holder, and a private package under the current name or any
alias is replaced by whoever `effectiveOwner` names, whatever its stamp says.

## Where the discussion is heading

moul opened [#6341](https://github.com/gnolang/gno/pull/6341) as a companion
ADR and set out the split in a
[comment](https://github.com/gnolang/gno/pull/6317#issuecomment-6176965792).
gnolang/gno#6341 keeps the name where it is and makes the set of addresses allowed to
act for it editable: `r/<name>/home` declares a member list in a new
`r/sys/teams`. Rotating a key becomes adding a member and removing one. The
comment names two cases teams cannot reach. A lost key or a dead holder cannot
sign the team changes, and no realm can call `checkRedeployPermission`, so a
private realm stays bound to its creator. It calls this ADR's resolver the
only proposal that changes the second.

clockworkgr
[answered](https://github.com/gnolang/gno/pull/6317#issuecomment-6184871940)
with a joint design that splits identity, namespace authority and package
ownership:

1. **Teams, realm-only.** gnolang/gno#6341's member list grants new deploys, and the
   name stays put.
2. **One fixed entry point.** `sysnames_pkgpath` is pointed once at a small
   `r/sys/authority` realm, and governance plugs later policy into it, so no
   further parameter switch is needed and no `home` imports a names version.
3. **Package ownership follows namespace authority.** The keeper asks the
   entry point who may replace, park, enable or reject, which is this ADR's
   section 5 narrowed. Admins may maintain any package and members only
   their own, a scoped reversal of gnolang/gno#6171. Today's rules hold until the
   policy is switched on.
4. **Recovery by governance.** A T1 proposal sets a new admin set for a
   namespace and its aliases, and can freeze it first.
5. **Moving the identity** goes to a later, separate ADR. It is the only part
   that touches `r/sys/users`, so the record ids, the guard and the namereg
   offers leave this one.

The reply also lists where the joint design departs from gnolang/gno#6341: new paths
only under an address or a live record's current name, a team honoured only
once the holder or governance adopts it, admin and publisher roles, and a
crossing `SetMember(cur realm, …)` entry point. It walks two mainnet names,
`aib` and `samcrew`, through the plan, and ends by offering to rework gnolang/gno#6317
along these lines.

## Words used here

| Word | Meaning |
|---|---|
| record | One `*UserData` in `r/sys/users`: an address, its current name and a deleted flag, reachable from every name key that points at it. |
| alias | A name the record renamed away from; its key in `nameStore` still points at the record and `ResolveName` returns `isCurrent == false`. |
| stamped creator | The `[addpkg] creator` the keeper writes into a deployed package's `gnomod.toml`; `init()` runs as this address. |
| effective owner | Proposed: the address the keeper treats as owner of a namespace's existing packages, from `NamespaceOwner`. |
| `id`, `rev` | Proposed: a record's permanent number, never reused, and a counter raised by every rename, delete and transfer after activation. |
| controller | A realm on the `r/sys/users` whitelist, such as `r/sys/namereg/v0`, allowed to register, rename and delete any record. |
| T1 | The GovDAO membership tier with the [highest base voting power, 3](https://github.com/gnolang/gno/blob/24279be63/examples/gno.land/r/gov/dao/memberstore/v0/memberstore.gno#L30-L37) · [↗](../../../../.worktrees/gno-review-6317-base/examples/gno.land/r/gov/dao/memberstore/v0/memberstore.gno#L30); `NewFilterByTier(memberstore.T1)` limits a proposal's voters to it, not who may create it. |
| park | Under the `inert` submission policy, a deploy is stored without running until an approver sends `MsgEnablePackage`. |
| session key | A delegated key with an `AllowPaths` list; `runtime.GetSessionInfo()` reports whether the transaction used one. |
| genesis replay | An upgrade that boots a new chain by re-delivering the old chain's accepted transactions; `IsGenesisReplay` is true throughout. |
| activation flag | Proposed: the `r/sys/users` switch the activation transaction sets last; every new realm behaviour checks it. |
| `sysnames_pkgpath` | The VM parameter naming the realm the keeper asks about namespaces; [an empty value means `gno.land/r/sys/names`](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/params.go#L539-L541) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/params.go#L539), and a path with nothing deployed [turns the check off](https://github.com/gnolang/gno/blob/24279be63/gno.land/pkg/sdk/vm/keeper.go#L503-L507) · [↗](../../../../.worktrees/gno-review-6317-base/gno.land/pkg/sdk/vm/keeper.go#L503). |

<details>
<summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or
a generated file. The pull request edits one file, so every
stop is a section of it.

### 1. `hot` · Who controls packages today, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R25-R52)
The five keeper checks the rest of the ADR rewrites, as they stand today.
**If wrong:** the ownership rules below replace a check that is not there, or
miss one that is.

### 2. `hot` · The resolver and its status table, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R266-R287)
Decides who owns existing packages under each namespace status, and that an
error grants nobody.
**If wrong:** a stolen old key or a failing realm call keeps the right to
replace private code.

Proposed, from the ADR:

```go
func (vm *VMKeeper) effectiveOwner(ctx sdk.Context, st gno.TransactionStore,
    params Params, pkgPath string, stamped crypto.Address) (owner crypto.Address, paused bool, err error)
```

### 3. `hot` · Where the resolver is applied, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R289-R349)
Replaces the creator and namespace checks on the normal, park and enable
paths, and moves pause into the keeper.
**If wrong:** an alias package is updatable on one path and blocked on
another, or pause stops covering redeploys.

### 4. `hot` · The record move and its guard, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R73-R161)
Adds `id` and `rev`, refuses mutations on read-only copies, and binds
proposals to an `id`.
**If wrong:** a copy is moved while the name keys stay on the real record,
or a proposal lands on a stranger who reused the address.

### 5. `hot` · Offer and accept, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R191-R233)
Lets the holder hand the record to a key that proves itself by accepting.
**If wrong:** a session key or a stale offer gives a name away.

### 6. `hot` · Activation, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R388-R467)
Keeps every new realm rule off during replay and switches it on once, after
history.
**If wrong:** the upgraded chain fails to replay its own history, or ids
shift between upgrades.

### 7. `hot` · The authorization matrix, [`gno.land/adr/pr6317_name_transfer.md`](https://github.com/gnolang/gno/pull/6317/files#diff-a4a979c0eb8dd087b6d5bc018b502e18f140b2ab3e8f54110efda42a5e7aff51R469-R491)
Every operation against every namespace status, the table the keeper tests
are written from.
**If wrong:** a cell the prose sections disagree with ships untested.

</details>
