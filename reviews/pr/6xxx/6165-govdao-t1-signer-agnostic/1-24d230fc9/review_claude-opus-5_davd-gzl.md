# PR [#6165](https://github.com/gnolang/gno/pull/6165): chore: make the govDAO T1 extend script signer-agnostic

URL: https://github.com/gnolang/gno/pull/6165
Author: moul | Base: master | Files: 1 | +36 -13
Reviewed by: davd-gzl | Model: claude-opus-5, effort xhigh | Commit: 24d230fc9 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6165 24d230fc9`
Open the code: [github.dev](https://github.dev/moul/gno/tree/24d230fc9) or [vscode.dev](https://vscode.dev/github/moul/gno/tree/24d230fc9), locally `./scripts/review-worktrees.sh gno 6165`
Overview: [overview](../overview.md)

## Overview

`misc/govdao-scripts/extend-govdao-t1.sh` writes a small gno program to a temp
file and broadcasts it with `gnokey maketx run`, which seats a fixed list of
addresses as govDAO T1 members. The list was written as "the six members who are
not moul", so only moul could sign it. This change turns the six calls into a
seven-entry table plus a loop, reads each address's current tier with
[`GetMember`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L94)
and skips the ones already seated, which drops the signer's own entry whoever
signs. The loop also makes a rerun idempotent, where the old `must()` turned the
first already-seated address into a panic that aborted the whole transaction.

**Verdict: NEEDS DISCUSSION** — the loop and the skip do what the description
says, proven by a run, but the transaction never reaches them: the composite
literal on the seat line aborts it, and on every network the script names
`memberstore.Get` refuses a `maketx run` outright, so signer-agnostic is not the
reason it fails for aeddi (2 warnings, 1 nit).

## Verify first

- [`misc/govdao-scripts/extend-govdao-t1.sh:50`](https://github.com/gnolang/gno/blob/24d230fc9/misc/govdao-scripts/extend-govdao-t1.sh#L50) · [↗](../../../../../.worktrees/gno-review-6165/misc/govdao-scripts/extend-govdao-t1.sh#L50). Decide whether a `maketx run` is still the delivery mechanism at all. Read the last line of each network's bootstrap, `grep -n UpdateImpl misc/deployments/*/govdao_prop1*.gno misc/deployments/*/transactions/base/bootstrap/govdao_prop1_*.gno`, and confirm every one of them passes a non-empty `AllowedDAOs`.
- [`misc/govdao-scripts/extend-govdao-t1.sh:59`](https://github.com/gnolang/gno/blob/24d230fc9/misc/govdao-scripts/extend-govdao-t1.sh#L59) · [↗](../../../../../.worktrees/gno-review-6165/misc/govdao-scripts/extend-govdao-t1.sh#L59). `&memberstore.Member{...}` is an allocation of another realm's type. Run [`tests/govdao_t1_roster_open.txtar`](tests/govdao_t1_roster_open.txtar) and read the abort message.

## Summary

Two runtime rules decide whether this script can write the member tree, and
neither is visible to a static check.
[`memberstore.Get`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L182-L189)
gates on the calling realm's pkgpath, which for a `maketx run` is
`gno.land/e/<signer>/run` per
[`keeper.go:1398`](https://github.com/gnolang/gno/blob/24d230fc9/gno.land/pkg/sdk/vm/keeper.go#L1398),
and the VM refuses a realm allocating a type another realm owns. Both fire
before a single member is seated, and both predate this branch: the same two
aborts reproduce on the merge base bc35e978a. `gno lint` is clean on the
extracted program and its type check is live, catching a deliberate
`ms.GetMemberXXX` typo, so the verification in the description could not have
seen either.

The roster loop itself is correct. With
[`NewMember(3)`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L21)
in place of the literal, a run inside the bootstrap window seats six entries and
prints `skip Manfred -- already T1` for the seventh.

## Fix

Six `must(ms.SetMember(...))` statements become `t1Roster`, a slice of name and
address pairs, and a loop over it. Each iteration reads the address's tier and
either prints a skip line or seats it with three invitation points, and a
`SetMember` error now panics inline instead of through the deleted `must`
helper. The load-bearing constraint is that
[`SetMember`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L75-L78)
refuses an address that sits in any tier rather than moving it, so a seven-entry
target list needs the read-then-skip that the loop adds.

## Warnings (should fix)

- **[access control]** `misc/govdao-scripts/extend-govdao-t1.sh:50` — `memberstore.Get` refuses a `maketx run` on all five networks the script's own header names, so the script cannot seat anyone on any of them.
  <details><summary>details</summary>

  `Get` reads the calling realm's pkgpath and panics unless
  [`InAllowedDAOs`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/proxy.gno#L231-L240)
  holds it. The empty list that lets any caller through exists only for the
  genesis window, and every network closes that window on its last bootstrap
  line: [gnoland1](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/gnoland1/govdao_prop1.gno#L118-L120),
  [test13](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/test13.gno.land/transactions/base/bootstrap/govdao_prop1_test13.gno#L109),
  [pearl](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/pearl.gno.land/transactions/base/bootstrap/govdao_prop1_pearl.gno#L49),
  [sapphire](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/sapphire.gno.land/transactions/base/bootstrap/govdao_prop1_sapphire.gno#L49)
  and [topaz](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/topaz.gno.land/transactions/base/bootstrap/govdao_prop1_topaz.gno#L49).
  Signer identity plays no part: the check is on the ephemeral realm's path, and
  the signer's own T1 membership is never read. So the description's account of
  the failure, `ErrMemberAlreadyExists` on the `// Aeddi` line, names a line the
  transaction never reaches, and making the roster signer-agnostic changes
  nothing about who can run it. Reproduced in
  [`tests/govdao_t1_roster_locked.txtar`](tests/govdao_t1_roster_locked.txtar),
  which seats the signer as sole T1, locks `AllowedDAOs` to the production value
  and gets
  `this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run`.
  Fix: seat a T1 member through a govDAO proposal built by
  [`NewAddMemberRequest`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L80),
  the one path `AllowedDAOs` still admits, and retire the direct-write script.
  </details>

- **[correctness]** `misc/govdao-scripts/extend-govdao-t1.sh:59` — `&memberstore.Member{InvitationPoints: 3}` allocates another realm's type, so the transaction aborts before the first member is seated.
  <details><summary>details</summary>

  A gno value belongs to the realm that allocated it, and the run realm may not
  allocate a `memberstore.Member`. The VM raises
  `cannot allocate gno.land/r/gov/dao/v3/memberstore.Member in realm gno.land/e/<signer>/run`
  and the whole `MsgRun` reverts, so no entry survives even in the genesis
  window where `AllowedDAOs` is empty.
  [`NewMember`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L21)
  is the constructor that allocates inside `memberstore`, and both
  [`govdao_prop1_pearl.gno:44`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/pearl.gno.land/transactions/base/bootstrap/govdao_prop1_pearl.gno#L44)
  and
  [`govdao_prop1_test13.gno:97`](https://github.com/gnolang/gno/blob/24d230fc9/misc/deployments/test13.gno.land/transactions/base/bootstrap/govdao_prop1_test13.gno#L97)
  already call it where this script writes the literal. Both forms type-check,
  so nothing static separates them. Reproduced in
  [`tests/govdao_t1_roster_open.txtar`](tests/govdao_t1_roster_open.txtar), whose
  commented assertions are the measured post-fix output: with `NewMember(3)` the
  same run prints six `seat` lines, `skip Manfred -- already T1`, and `OK!`.
  Fix: call `memberstore.NewMember(3)`.
  </details>

## Nits

- **[docs]** `misc/govdao-scripts/README.md:17` — the command list still reads `add 6 T1 members to govDAO (one-time bootstrap)`, where the roster is now seven and rerunning is the point of the change.

## Verified

- The allowlist refusal is a live run, not a source read:
  [`tests/govdao_t1_roster_locked.txtar`](tests/govdao_t1_roster_locked.txtar)
  boots a node, locks `AllowedDAOs` to `gno.land/r/gov/dao/v3/impl` exactly as
  the pearl bootstrap does, and the signed `maketx run` comes back with the
  panic from
  [`memberstore.gno:188`](https://github.com/gnolang/gno/blob/24d230fc9/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L188).
- Both aborts reproduce at the merge base bc35e978a with that revision's own
  script body, so neither is caused by this branch. The base run stops at
  `cannot allocate` on its first `SetMember` line.
- The loop's own logic was run to completion with the allocation fixed: six
  seats, one skip, `STORAGE DELTA: 7085 bytes` on the receipt.
- `gno lint` is clean on the extracted program and non-vacuous: renaming
  `ms.GetMember` to `ms.GetMemberXXX` gives
  `ms.GetMemberXXX undefined (type memberstore.MembersByTier has no field or method GetMemberXXX) (code=gnoTypeCheckError)`.
  The untyped string constants in the roster literal assign to the `address`
  field with no conversion, which is why dropping `address(...)` costs nothing.
- No CI job reaches either finding.
  [`ci-dir-misc.yml`](https://github.com/gnolang/gno/blob/24d230fc9/.github/workflows/ci-dir-misc.yml?plain=1#L26-L40)
  runs `go test` over a fixed list of Go programs under `misc/`, and nothing in
  it compiles or executes a shell script or the gno source a script embeds. The
  head is green on every check.
- Invariant catalog walked. Caller and access control is the class this diff
  lands in, and it is the first Warning. Realm state safety holds: a mid-loop
  panic reverts the whole transaction, so no partial seating survives. `t1Roster`
  is a package-level var written once and read-only afterwards, the shape the
  catalog names as safe. Determinism, gas, coin and banker, storage deposit,
  panic handling and VM semantics are untouched.

## Open questions

- The script validates no address, so a typo in the roster seats an unusable
  entry that only `RemoveMember` clears. Not posted: it predates the branch, it
  is one keystroke from the same risk in every sibling script under
  `misc/govdao-scripts/`, and the fix belongs in `SetMember` rather than in
  seven callers.
