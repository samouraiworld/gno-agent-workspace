# PR [#5888](https://github.com/gnolang/gno/pull/5888): feat(gnovm): phase 2 — inert package storage and oracle activation

URL: https://github.com/gnolang/gno/pull/5888
Author: moul | Base: master | Files: 12 | +940 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: b8818eb8e (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5888 b8818eb8e`

**TL;DR:** Adds a new chain policy where anyone can upload a package but it is parked "inert" (stored, not typechecked, not run) until a trusted approver address sends a message that typechecks it, runs it, and makes it live. The goal is to keep the expensive Go typechecker off the block-execution critical path.

**Verdict: REQUEST CHANGES** — activation persists a package with zero storage-deposit accounting (free permanent storage + a corrupted deposit baseline), and the inert+enable route skips the package-policy checks the normal path enforces (dev/draft/private). Both are correctness gaps, not just staging TODOs.

## Summary
Phase 2 of the code-submission permissioning work. A new `code_submission_policy = "inert"` param makes `MsgAddPackage` store the package at `inert_pkg:<path>` in the iavlStore without typecheck or execution; a new `pkg_approvers` allowlist gates two new messages, `MsgEnablePackage` (typecheck + run + delete the inert copy) and `MsgDisablePackage` (approver-gated stub, returns an error). The store gains `AddInertPackage` / `GetInertPackage` / `DelInertPackage`. Inert keys are disjoint from `pkg:` and the package index counter is untouched, so inert packages stay invisible to `GetPackage`, `IterMemPackage`, and `FindPathsByPrefix`. The normal `AddPackage` path charges a refundable storage deposit proportional to the persisted byte delta; the inert branch and `EnablePackage` charge none.

```
MsgAddPackage (policy=inert)          MsgEnablePackage (approver only)
  namespace + CLA + SendCoins           typecheck (TCLatestStrict)
  AddInertPackage  ── inert_pkg:path      RunMemPackage(save)  ── pkg:path (+ realm objects)
  (no typecheck, no run,                  DelInertPackage
   no storage deposit)                    (no processStorageDeposit)
```

## Glossary
- storage deposit: per-realm refundable charge for on-chain storage, locked on positive byte delta via `processStorageDeposit`, tracked as `rlm.Deposit`/`rlm.Storage`.
- addpkg: the `MsgAddPackage` transaction that uploads a package or realm.
- Store: backing store for packages and objects (`defaultStore`), layered over tm2 IAVL.
- realm: stateful on-chain package under `r/` whose objects persist across transactions.

## Critical (must fix)
None.

## Warnings (should fix)
- **[enabling a package hands out free permanent storage and poisons its deposit math]** [`gno.land/pkg/sdk/vm/keeper.go:854`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L854) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L854) — `EnablePackage` runs `RunMemPackage(save)` but never calls `processStorageDeposit`, so activation locks no deposit and the realm's accounting starts from a false zero.
  <details><summary>details</summary>

  Normal `AddPackage` calls `vm.processStorageDeposit(...)` right after `RunMemPackage` ([keeper.go:783](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L783) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L783)), which locks a refundable deposit proportional to `RealmStorageDiffs()` and sets `rlm.Storage`/`rlm.Deposit`. `EnablePackage` persists the same realm objects but omits that call, and the inert `AddPackage` branch also returns before it ([keeper.go:658](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L658) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L658)). Two consequences. First, the activated package gets free permanent on-chain storage that every normal deployment pays for. Second, and worse, `rlm.Storage` and `rlm.Deposit` both stay 0 while the realm holds real persisted bytes, so every later storage-deposit computation for that realm runs off a wrong baseline: a future write locks against a phantom-zero prior size, and a future release refund of `rlm.Deposit*released/rlm.Storage` divides by a zero `Storage`. Behavioral proof [repro](comment_claude-opus-4-8.md): normal addpkg of a stateful package records `Storage=3040 Deposit=304000`; the same source via inert+enable records `Storage=0 Deposit=0`. Fix: charge the storage deposit on activation the way `AddPackage` does, so `rlm.Storage`/`rlm.Deposit` reflect the persisted bytes.
  </details>

- **[inert route skips the dev / draft / private package checks the normal path enforces]** [`gno.land/pkg/sdk/vm/keeper.go:803-858`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L803-L858) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L803-L858) — neither the inert `AddPackage` branch nor `EnablePackage` re-checks the gnomod policy gates, so a package the normal path rejects becomes live through inert+enable.
  <details><summary>details</summary>

  The normal path enforces, after typecheck, a block of keeper-only policy checks: no development packages (`gm.HasReplaces()`), no public override of a private package, private-only-for-realms, no post-genesis draft (`gm.Draft && ctx.BlockHeight() > 0`), and no deprecated `gno.mod` file ([keeper.go:683-698](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L683-L698) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L683-L698)). The inert branch ([keeper.go:640-660](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L640-L660) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L640-L660)) re-parses gnomod only to set Module/Creator/Height, and `EnablePackage` only typechecks and runs. None of those five gates run on the inert+enable route, and `gnomod.WriteString` preserves `replace`/`draft` from the submitted manifest into the inert copy, so a development or draft package activates on-chain despite being rejected through `MsgAddPackage`. Fix: run the same gnomod policy checks at enable time (or store them at submission and re-assert on enable).
  </details>

- **[activation runs the package's init as the oracle, not the submitter]** [`gno.land/pkg/sdk/vm/keeper.go:831`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L831) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L831) — `EnablePackage` sets `OriginCaller` to the approver, so any `init` that records the deploying caller attributes the realm to the oracle.
  <details><summary>details</summary>

  Normal `AddPackage` runs init with `OriginCaller: creator.Bech32()` ([keeper.go:736](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L736) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L736)); `EnablePackage` runs it with `OriginCaller: msg.Approver.Bech32()` and `OriginSend: std.Coins{}`. A realm whose `init` sets an owner from `std.PreviousRealm().Address()` / `std.OriginCaller()` records the approver, while `gnomod.toml` records the submitter as `Creator`, so on-chain identity and metadata disagree. Any deploy-time payment logic reading `OriginSend` also sees empty, even though the submitter's `msg.Send` was already transferred to the package address at submission. Fix: decide whose identity activation should carry and make the runtime caller and the recorded creator agree.
  </details>

## Nits
- [`gno.land/pkg/sdk/vm/keeper.go:640-660`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L640-L660) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L640-L660) — the inert branch silently ignores `msg.MaxDeposit`; once activation charges a deposit, the submitter's declared cap has nowhere to apply.
- [`gno.land/pkg/sdk/vm/keeper.go:658`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L658) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L658) — `AddInertPackage` is an unconditional `iavlStore.Set`, so a second inert submission at the same path overwrites the first with no "already inert" guard; the first submitter's `msg.Send` is already spent to the package address.

## Missing Tests
- **[deposit invariant is untested on the new path]** [`gno.land/pkg/sdk/vm/keeper_inert_test.go:1`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper_inert_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper_inert_test.go#L1) — no test asserts that enabling a package accounts storage and locks a deposit.
  <details><summary>details</summary>

  The lifecycle test checks resolvability and callability but never inspects `rlm.Storage`/`rlm.Deposit`, so the accounting gap ships green. The ready-to-add test in [`tests/enable_package_storage_deposit_test.go`](tests/enable_package_storage_deposit_test.go) asserts the post-fix invariant (deposit charged on enable) against the normal-addpkg baseline; it fails at b8818eb8e. See [repro](comment_claude-opus-4-8.md).
  </details>
- **[no integration coverage]** [`gno.land/pkg/sdk/vm/keeper_inert_test.go:1`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper_inert_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper_inert_test.go#L1) — the author's own test plan lists the end-to-end txtar (set policy, submit, verify not callable, enable, verify callable) as unchecked. Not posted; author already tracks it.
- **[policy-bypass path untested]** [`gno.land/pkg/sdk/vm/keeper_inert_test.go:1`](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper_inert_test.go#L1) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper_inert_test.go#L1) — no test that a draft or `replace`-carrying package is rejected on the inert+enable route the way it is on the normal path.

## Suggestions
- [`gnovm/pkg/gnolang/store.go:1070-1076`](https://github.com/gnolang/gno/blob/b8818eb8e/gnovm/pkg/gnolang/store.go#L1070-L1076) · [↗](../../../../../.worktrees/gno-review-5888/gnovm/pkg/gnolang/store.go#L1070-L1076) — inert bytes are a raw `iavlStore.Set`, not part of the realm object graph, so `RealmStorageDiffs` can never see them and the storage-deposit machinery cannot bill them even in principle. Under `inert` (permissionless submission) this is a state-growth surface: any address can park large packages in consensus state for one-time gas, with no refundable lock and no eviction if never enabled. Consider a size cap or an inert-submission deposit.
  <details><summary>rationale</summary>

  The typechecker DoS moves off the critical path, but a raw-storage-bloat surface opens in its place. Gas is a one-time charge; the storage deposit is the designed anti-bloat lock, and it does not reach inert entries.
  </details>

## Verified
- Storage-deposit gap, behavioral: normal permissionless addpkg of a stateful package records `Storage=3040 Deposit=304000`; the identical source via inert submit + oracle enable records `Storage=0 Deposit=0`. Ready-to-add test fails at b8818eb8e. [repro](comment_claude-opus-4-8.md).
- Inert keys do not leak into package enumeration: `AddInertPackage` writes only `inert_pkg:<path>` and never bumps the package index counter, while `IterMemPackage` walks the counter and `FindPathsByPrefix` ranges the `pkg:` keyspace only ([store.go:1100-1150](https://github.com/gnolang/gno/blob/b8818eb8e/gnovm/pkg/gnolang/store.go#L1100-L1150) · [↗](../../../../../.worktrees/gno-review-5888/gnovm/pkg/gnolang/store.go#L1100-L1150)). Confirmed by the lifecycle test: `GetPackage` returns nil while inert.
- App-hash shift is expected and correctly pinned: `DefaultParams` sets `CodeSubmissionPolicy` to the non-empty default `"permissionless"` ([params.go:98](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/params.go#L98) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/params.go#L98)), so serialized Params gain amino field 14 and the app hash moves; `apphash_crossrealm38_test.go` is the only apphash pin in the package and its constant is updated.
- CI `build` red is infrastructure, not this PR: `misc/gendocs` installs `golang.org/x/pkgsite@latest` which resolved to `v0.3.0` requiring `go >= 1.26.0` against the runner's `go 1.25.9`. Unrelated to the diff.
- Tests green at b8818eb8e: `TestVMKeeperInertPackageLifecycle`, `TestVMKeeperEnablePackageRejectsInvalidCode`, `TestVMKeeperDisablePackageNotImplemented`, `TestParamsString`, `TestAppHashCrossrealm38`.

## Open questions
- `msg.Send` in inert mode transfers coins to the derived package address at submission ([keeper.go:655](https://github.com/gnolang/gno/blob/b8818eb8e/gno.land/pkg/sdk/vm/keeper.go#L655) · [↗](../../../../../.worktrees/gno-review-5888/gno.land/pkg/sdk/vm/keeper.go#L655)); if the package is never enabled the coins sit at an address with no code to move them. Deferred: no path to reclaim, but low-frequency and policy-gated.
- The ADR file is named `pr5885_...` though this is PR 5888; cosmetic, not a code concern, not posted.
