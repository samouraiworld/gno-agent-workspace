# PR [#5756](https://github.com/gnolang/gno/pull/5756): feat(examples): add `memberstorage` boards sub-package

URL: https://github.com/gnolang/gno/pull/5756
Author: jeronimoalbi | Base: master | Files: 12 | +1365 -35
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `f9121247a` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5756 f9121247a`

Round 2. Head advanced `d940e681b` → `f9121247a` by three master merges (`#5747` N_Readonly-taint removal, `#5759` boards2 paginated comments, `#5872` filetest→Example conversion) plus a golden refresh. No `memberstorage`/`permissions` logic changed (byte-identical since `d940e681b`); the only PR-file deltas are the `readme_filetest.gno` → `example_test.gno` conversion pulled in from master `#5872` and moul's `z_4_a` golden update. The Critical was re-run on `f9121247a` and still reproduces; anchors re-cut against the merged tree; verdict unchanged.

**TL;DR:** The PR gives Boards2 its own member-storage package so it no longer depends on the CommonDAO library. The replacement drops a de-duplication guarantee the old storage had, so a Boards2 member who was first invited as a guest and then given a role is counted and listed twice and can't be fully removed.

**Verdict: REQUEST CHANGES** — the Boards2 migration silently drops the de-duplication guarantee the old `commondao` storage provided: a user added as a guest and later given a role is counted twice by `UsersCount()`, emitted twice by `IterateUsers()`, and `RemoveUser()` can't clean up the leftover. This path is reachable from the live invite→promote flow.

## Summary

The PR extracts a standalone `gno.land/p/gnoland/boards/exts/memberstorage` package and rewires Boards2's `permissions` package onto it, removing the `commondao` dependency. The new package is simpler by design: it keeps an ungrouped member set and per-group sets as separate, non-deduplicated stores. The old `commondao/v0/exts/storage` ext did the opposite: it mirrored every grouped member into one root set via a message broker, so member counts and iteration were always unique.

That mirror is what made `Permissions.UsersCount()` and `IterateUsers()` correct. Without it, the permissions layer now leaks a stale entry whenever a user transitions from guest (ungrouped) to a role (grouped): `SetUserRoles` clears role groups the user is in, but never removes the user from the ungrouped set, so the user ends up in both. `UsersCount()` (backed by `GetTotalSize`, sum of all set sizes) then over-counts, and `IterateUsers()` (backed by `Iterate`, walk-each-set-in-turn) emits the user once per set.

```
guest accepted:           ungrouped = {u}                 UsersCount=1   Iterate -> [u]
promoted to "admin":      ungrouped = {u}, admin = {u}     UsersCount=2   Iterate -> [u, u]
                          ^ stale entry never cleared      ^ want 1       ^ want [u]
```

## Glossary

- `memberStorage` — new storage: `members` (ungrouped `addrset.Set`) + optional `grouping`. `Size`/`Has`/`IterateByOffset` see only the ungrouped set.
- `GetTotalSize` / `HasMember` / `Iterate` — free funcs that span ungrouped + all groups; `GetTotalSize` and `Iterate` are explicitly non-deduplicated (one count/visit per occurrence).
- `commondao/v0/exts/storage` — the old member storage; mirrored grouped members into one root set, so counts/iteration were unique.
- `SetUserRoles(user)` with no roles — Boards2's "guest" state: user goes into the ungrouped set.

## Fix

In [`SetUserRoles`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L146) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L146) the user is added to the ungrouped set when made a guest ([L167](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L167) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L167)), but the "clear current roles" loop ([L154-163](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L154-L163) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L154-L163)) only walks `GetMemberGroups`, which never reports the ungrouped set. Add an unconditional `ps.members.Remove(user)` at the top of `SetUserRoles` (the guest branch re-adds it), and have `RemoveUser` also `ps.members.Remove(user)` in its grouped branch ([L184-200](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L184-L200) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L184-L200)). Alternatively, restore the dedup invariant in the storage layer (the [`member_storage.gno:121`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L121) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L121) TODO already flags this).

## Critical (must fix)

- **[promoted guests counted & listed twice; can't be removed]** [`exts/permissions/permissions.gno:154-163`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L154-L163) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L154-L163) — `SetUserRoles` clears role groups but never the ungrouped set, so a guest promoted to a role is over-counted by `UsersCount()`/`IterateUsers()` and `RemoveUser` leaves a ghost.
  <details><summary>details</summary>

  **Mechanism:** `SetUserRoles(user)` with no roles adds `user` to the ungrouped set ([L166-168](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L166-L168) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L166-L168)). On a later `SetUserRoles(user, "admin")`, the clear loop iterates `GetMemberGroups(user)` — which is empty, because the user is in the ungrouped set, not a group — so nothing is cleared, and the user is then added to the `admin` group ([L172-180](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L172-L180) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L172-L180)). The user now occupies both sets. `GetTotalSize` ([member_storage.gno:116-133](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L116-L133) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L116-L133)) sums both → 2. `Iterate` ([member_storage.gno:153-211](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L153-L211) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L153-L211)) visits both → the user appears twice (both rows carry the role, since `IterateUsers` re-resolves roles per occurrence via `GetMemberGroups`). `RemoveUser` ([L184-200](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L184-L200) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L184-L200)) takes the grouped branch (groups non-nil), removes from the `admin` group, returns `true`, and never touches the ungrouped set — so the ghost survives removal and `HasUser` still reports `true`.

  **Reachable from Boards2, not just unit-level:** [`AcceptInvite`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L105) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L105) calls `SetUserRoles(user)` (guest, ungrouped), then [`ChangeMemberRole`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/public.gno#L649) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/public.gno#L649) calls `SetUserRoles(member, role)`. The member-list render at [render.gno:249-255](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/render.gno#L249-L255) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/render.gno#L249-L255) sizes its pager with `UsersCount()` and walks `IterateUsers` — so a promoted-from-guest user shows as a duplicate row, and the inflated total can push a real member off the last page. `UseSingleUserRole` does not protect this path (the second occurrence is ungrouped vs. grouped, not two roles).

  **Repro** (run from a local clone of gnolang/gno):

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5756 -R gnolang/gno
  cat > examples/gno.land/p/gnoland/boards/exts/permissions/zz_repro_test.gno <<'EOF'
  package permissions

  import (
  	"testing"

  	"gno.land/p/gnoland/boards"
  )

  func TestZZGuestPromoteDoubleCount(t *testing.T) {
  	user := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
  	perms := New(WithSuperRole("owner"))
  	perms.AddRole("admin", testPermA)

  	perms.SetUserRoles(user)          // guest -> ungrouped set
  	perms.SetUserRoles(user, "admin") // promote -> ungrouped entry never cleared

  	println("UsersCount =", perms.UsersCount(), "(want 1)")
  	visits := 0
  	perms.IterateUsers(0, 100, func(u boards.User) bool { visits++; return false })
  	println("IterateUsers visits =", visits, "(want 1)")
  	println("ghost ungrouped Has =", perms.members.Has(user), "(want false)")
  }
  EOF
  cd examples && gno test -v -run TestZZGuestPromoteDoubleCount \
    ./gno.land/p/gnoland/boards/exts/permissions/
  rm gno.land/p/gnoland/boards/exts/permissions/zz_repro_test.gno
  ```

  ```
  UsersCount = 2 (want 1)
  IterateUsers visits = 2 (want 1)
  ghost ungrouped Has = true (want false)
  ```

  Fix: see the Fix section — unconditional `ps.members.Remove(user)` in `SetUserRoles`, plus the same in `RemoveUser`'s grouped branch.
  </details>

## Warnings (should fix)

None.

## Nits

- [`exts/memberstorage/member_group.gno:29`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_group.gno#L29) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_group.gno#L29) — the canonical-type checker here is named `IsCanonical`, while its siblings are `IsCanonicalMemberStorage` ([member_storage.gno:45](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L45) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L45)) and `IsCanonicalMemberGrouping` ([member_grouping.gno:43](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L43) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L43)). Rename to `IsCanonicalMemberGroup` for consistency.
- [`exts/memberstorage/member_grouping.gno:88-91`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L88-L91) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L88-L91) — `Delete` discards `bptree.Remove`'s result and always returns `nil`, so deleting a non-existent group is a silent success. Asymmetric with `Add`, which errors on a duplicate. Either error on missing or drop the vestigial `error` return.
- [`exts/memberstorage/member_storage.gno:39-45`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L39-L45) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L39-L45) — the three `IsCanonical*` guards carry a doc directive ("Use this function at any public entry point that accepts X from an external caller before invoking its methods"), but the package's own consumer (`permissions`) never calls any of them (`New()` builds its own canonical storage). The guards are for downstream consumers that accept externally-supplied storages — worth saying so in the doc, since no shipped caller demonstrates the pattern.
- [`exts/permissions/permissions.gno:213`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L213) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L213) — `IterateUsers` now returns the real `stopped` value from `Iterate`. The old version discarded `IterateByOffset`'s result and bare-`return`ed the zero `stopped=false`, so early-stop never propagated. This is a latent-bug fix; flagging so it's intentional, not accidental.

## Missing Tests

- **[the Critical path is entirely uncovered]** [`exts/permissions/permissions_test.gno`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno?plain=1) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno) — no test exercises guest→promotion. The ready-to-add regression is [`tests/guest_promote_test.gno`](tests/guest_promote_test.gno): `SetUserRoles(u)` then `SetUserRoles(u, role)`, asserting `UsersCount()==1`, a single `IterateUsers` emission, and `HasUser(u)` false after `RemoveUser(u)`. It fails red on `f9121247a` (`UsersCount = 2, want 1`; `IterateUsers visits = 2, want 1`; ghost survives `RemoveUser`) and passes once the Critical is fixed.
  <details><summary>details</summary>

  The existing `TestBasicPermissionsIterateUsers` ([permissions_test.gno:510](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno?plain=1#L510) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno#L510)) masks the related multi-role variant: it calls `IterateUsers(0, len(users), …)` with `count == distinct-user-count`, so the count budget is exhausted before the duplicate occurrence is reached and `users[i]` indexing happens to line up. With `count >= UsersCount()` (exactly what `render.gno` and `z_iterate_realm_members_00_filetest.gno:28` do) the duplicate surfaces. This is why CI is green despite the Critical.
  </details>
- [`exts/memberstorage/member_grouping.gno:102`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L102) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L102) — `GetMemberGroups` is public but has no direct unit test (only exercised transitively). Add table cases: member in 0 / 1 / 2+ groups, and a `New()` storage (nil grouping → nil). Non-blocking coverage gap.
- [`exts/memberstorage/member_storage.gno:159`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L159) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L159) — `Iterate` is well-tested, but a few boundary cases are unisolated: an empty group earlier in the walk order, an offset landing exactly on a group boundary, and a duplicate present in both the ungrouped set and a group (current tests only duplicate across two groups). Non-blocking coverage gap.

## Suggestions

- [`exts/permissions/permissions.gno:213-226`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L213-L226) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L213-L226) — iteration order moved from address order (old unified root) to ungrouped-then-group-name order. Both changed filetest goldens now bake this in: [`z_iterate_realm_members_00_filetest.gno`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/filetests/z_iterate_realm_members_00_filetest.gno) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/filetests/z_iterate_realm_members_00_filetest.gno) and [`z_4_a_filetest.gno`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/hub/filetests/z_4_a_filetest.gno) · [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/hub/filetests/z_4_a_filetest.gno), whose members now list `admin` before `owner` (group-name order) rather than address order. No Boards2 consumer depends on member ordering, but any external `IterateUsers` consumer that assumed address-sorted output is silently affected. Worth a line in the PR description or the `IterateUsers` doc.

## Open questions

- Is the loss of de-duplication intentional? The old `commondao/v0/exts/storage` was purpose-built to keep one unique root set ("quick and inexpensive checks for the number of total unique storage users"); `memberstorage` drops that and `GetTotalSize`/`Iterate` document the non-dedup contract. If Boards2 only ever needs single-role-per-user and never mixes guest+role for the same address, the dedup may be unnecessary — but the guest→promote flow above shows it currently can mix. Framed in comment.md as the decision behind the Critical, not as a separate posted question.
- `#5872` (master) converted the package's `readme_filetest.gno` into `example_test.gno` (`ExamplePermission`); it runs green on `f9121247a`. Base change, no PR finding.
