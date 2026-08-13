# PR [#6062](https://github.com/gnolang/gno/pull/6062): fix(gnovm,gno.land): three ways coins were lost or over-authorized around the send-envelope

URL: https://github.com/gnolang/gno/pull/6062
Author: jaekwon | Base: master | Files: 34 | +1318 -66
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: f6dd8ad37 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6062 f6dd8ad37`

## Overview

Attaching coins to a transaction moves them into the called realm's address before its code
runs. Three things went wrong around that envelope. A `BankerTypeOriginSend` banker kept its
authority after the message ended, because the spend limit was looked up from whatever message
was running rather than stored in the banker, so a banker handed to another realm and stashed
there re-armed against a later, unrelated envelope and spent the granter's own balance. A realm
that never looked at the envelope kept the coins with no error and usually no way to return
them. And coins attached to the deployment of a pure `p/` package landed in an address nothing
can ever spend from.

The branch fixes each in a different place. The banker now carries a live realm handle, which
the persistence walk refuses to store, so the value cannot cross a message boundary; a second
gate checks at spend time that the spending address is the address this message's envelope was
credited to. `MsgCall` now tracks whether the paid realm made the envelope observable and fails
the message when a non-empty envelope went unread. `MsgAddPackage` rejects a non-empty payment
to a non-realm path.

**Verdict: NEEDS DISCUSSION** — all three fixes hold under attack, and the open item is the one
the author already raised: the payable check credits the realm the VM is *borrowed* into, not
the realm whose code is running, which refuses a payment master accepts (1 Warning, 1 Missing test, 3 Nits).

## Verify first

- [`gnovm/stdlibs/chain/banker/banker.gno:192`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.gno#L192) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.gno#L192) — the new `rlm realm` field changes the stored shape of every persisted `banker`. [`banker_persistence.txtar:98`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/banker_persistence.txtar#L98) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/banker_persistence.txtar#L98) writes and reads such values inside one chain, so it cannot see a `BankerTypeRealmSend` banker written by an older node; decide whether any live chain holds one before merging.
- [`gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:129`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L129) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L129) — the only pinned app hash in the tree, and the branch moves it. `go test ./gno.land/pkg/sdk/vm/ -run TestAppHash` passes at f6dd8ad37, so the pin matches; what needs a decision is the coordinated upgrade it implies.
- [`gno.land/pkg/sdk/vm/keeper.go:954`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L954) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L954) — one line decides which transactions stop being valid. Run [`tests/zz_payable_borrow.txtar`](tests/zz_payable_borrow.txtar) and read which of its two `shop` calls is refused.

## Summary

The three defects are each named in the code with a test that fails before the fix. Defect one is
[`banker.gno:152-168`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.gno#L152-L168) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.gno#L152-L168) pinning the banker to a realm value plus the recipient gate at
[`banker.go:83`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.go#L83) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.go#L83); defect two is the `MsgCall` check at
[`keeper.go:954`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L954) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L954); defect three is the deploy check at
[`keeper.go:649`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L649) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L649). I ran four escape routes against the lifetime pin and all four abort before any
coin moves. The one shipped hazard is the realm attribution behind the payable check: it reads
`Machine.Realm`, which the borrow rules point at the owner of the receiver object, so a realm
reading its own envelope through an object another realm owns is refused while the identical
direct read is accepted.

Reading order: [`execctx/context.go`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/internal/execctx/context.go#L49-L148) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/internal/execctx/context.go#L49-L148) for the three new context fields, then
[`banker.gno`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.gno#L117-L192) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.gno#L117-L192) and [`banker.go`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.go#L55-L112) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.go#L55-L112), then the two keeper checks, then the fixtures.

## The three defects

| Defect | Fix | Case that failed before | Result at 754780601 | Result at f6dd8ad37 |
|---|---|---|---|---|
| An OriginSend banker outlives its message | [`banker.gno:152-168`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.gno#L152-L168) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.gno#L152-L168) | `esc.SaveClosure` stashes one in a closure | `OK!`, banker persists | `cannot persist realm value` |
| A realm is paid and never looks | [`keeper.go:954`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L954) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L954) | `payable.Ignores` with `-send 400ugnot` | `OK!`, 400 stranded | `never read the send-envelope` |
| Coins deployed to a pure package | [`keeper.go:649`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L649) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L649) | `addpkg -pkgpath gno.land/p/test/lib -send 700000ugnot` | `OK!`, 700000 unrecoverable | `can never spend it` |

The banker row is measured with [`tests/zz_originsend_escape.txtar`](tests/zz_originsend_escape.txtar), which drives four routes past the
persistence walk: a closure capture, a pointer inside a struct inside a slice, a map value, and a
bound method value. Every route aborts at f6dd8ad37. On the merge base the closure route
succeeds, so the pin is what closes it and not a pre-existing refusal.

## Warnings (should fix)

- **[correctness: a valid payment is refused]** [`gnovm/stdlibs/chain/runtime/unsafe/unsafe.go:39`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/runtime/unsafe/unsafe.go#L39) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/runtime/unsafe/unsafe.go#L39) — the read is credited to `Machine.Realm`, which the borrow rules point at the owner of the receiver object rather than at the realm whose code is running, so a paid realm reading its own envelope through an object another realm owns has its payment refused.
  <details><summary>details</summary>

  [`MarkOriginSendObservedBy`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/internal/execctx/context.go#L122-L129) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/internal/execctx/context.go#L122-L129) compares the path it is handed against `OriginSendRecipientPath`, and the
  caller hands it `m.Realm.Path`. `Machine.Realm` is the borrowed realm: [`machine.go:2607-2618`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/gnolang/machine.go#L2607-L2618) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/pkg/gnolang/machine.go#L2607-L2618)
  switches it to the constructing realm of a real, foreign-stamped receiver whenever a
  `/p/`-declared method runs, and [`machine.go:2641-2650`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/gnolang/machine.go#L2641-L2650) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/pkg/gnolang/machine.go#L2641-L2650) does the same for a closure's capture
  realm. Both directions of the mismatch are reachable, and the refusal one is a behaviour the
  merge base does not have.

  [`tests/zz_payable_borrow.txtar`](tests/zz_payable_borrow.txtar) builds them from outside the boundary, with no change to any stdlib.
  `shop.Buy` reads its own envelope through a `*tok.T` that `r/test/host` owns and is refused;
  `shop.BuyDirect` reads the same envelope directly and is accepted. `payee.Ignores` never reads
  its envelope and is accepted, because `helper` read it through an object `payee` owns, which
  is the shape [`payable_callee_read.txtar:12`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/payable_callee_read.txtar#L12) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/payable_callee_read.txtar#L12) pins shut for a plain callee.

  ```
  > gnokey maketx call -pkgpath gno.land/r/test/payee -func Ignores -send 400ugnot ...
  ("helper-saw=400" string)
  > gnokey maketx call -pkgpath gno.land/r/test/payee -func Balance ...
  ("balance=400" string)
  > ! gnokey maketx call -pkgpath gno.land/r/test/shop -func Buy -send 400ugnot ...
  Data: coins were sent but the called function never read them
      0  gno/gno.land/pkg/sdk/vm/errors.go:99 - 400ugnot sent to gno.land/r/test/shop.Buy, which never read the send-envelope
  > gnokey maketx call -pkgpath gno.land/r/test/shop -func BuyDirect -send 400ugnot ...
  ("bought=400" string)
  ```

  The same file at the merge base 754780601 reports `bought=400` for `shop.Buy`, so the refusal
  arrives with this branch. Fix: decide between the exact answer, which needs the paid realm's
  presence anywhere on the frame stack, and keeping the single comparison; either way pin the
  shape with a fixture, since [`payable_callee_read.txtar:12`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/payable_callee_read.txtar#L12) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/payable_callee_read.txtar#L12) covers only the plain-callee shape.
  </details>

## Missing Tests

- **[the rule cannot be exercised locally]** [`gnovm/pkg/test/test.go:66`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/test/test.go#L66) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/pkg/test/test.go#L66) — `OriginSendObserved` is seeded for the gno test harness and nothing ever reads it, so a `SEND:` filetest over a function that never touches the envelope passes green while the chain refuses the same call.
  <details><summary>details</summary>

  [`test.Context`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/test/test.go#L66) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/pkg/test/test.go#L66) allocates the flag and [`X_setContext`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/tests/stdlibs/testing/context_testing.go#L129-L155) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/tests/stdlibs/testing/context_testing.go#L129-L155) keeps the recipient
  in step with it, but no harness path consults it the way
  [`keeper.go:954`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L954) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L954) does. A realm author therefore gets no local signal that a payment path is
  invalid on chain, and no filetest can guard the new rule against regression.

  [`tests/zz_unobserved_send_filetest.gno`](tests/zz_unobserved_send_filetest.gno) is the case: `SEND: 400ugnot`, a `main` that calls a
  function with no notion of payment, and an `// Output:` line asserting the 400 sitting in the
  realm's address. It passes at f6dd8ad37, while
  [`payable_observed_send.txtar:30-32`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/payable_observed_send.txtar#L30-L32) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/payable_observed_send.txtar#L30-L32) shows the chain refusing the same shape.

  ```
  $ go test -run 'TestFiles/zz_unobserved_send_filetest.gno$' ./gnovm/pkg/gnolang/ -v
  --- PASS: TestFiles/zz_unobserved_send_filetest.gno (2.89s)
  ```

  Fix: make the harness fail a test the chain would fail, so the filetest above turns red until
  the fixture reads its envelope.
  </details>

## Nits

- **[stale arithmetic]** [`gno.land/pkg/gnoclient/integration_test.go:684`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/gnoclient/integration_test.go#L684) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/gnoclient/integration_test.go#L684) — the breakdown comment still charges `Send 1000000` and its total matches neither the old assertion nor the new one.
  <details><summary>details</summary>

  The line reads `999999654370 = 10000000000000 - (GasFee 2100000 + Storage Deposit 176800 +
  Storage Deposit 177900 + Send 1000000)` while
  [`integration_test.go:687`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/gnoclient/integration_test.go#L687) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/gnoclient/integration_test.go#L687) now asserts `9999997545300`. The send it subtracts is the one this
  commit removed. Fix: drop the `Send 1000000` term and recompute, or delete the comment.
  </details>

- **[half-set pair]** [`gno.land/pkg/sdk/vm/keeper.go:745`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L745) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L745) — `AddPackage` sets `OriginSendRecipient` and leaves `OriginSendRecipientPath` empty, so the two disagree, which [`context_testing.go:150-152`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/tests/stdlibs/testing/context_testing.go#L150-L152) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/tests/stdlibs/testing/context_testing.go#L150-L152) says must never happen.
  <details><summary>details</summary>

  Nothing reads either field on this path today: `init()` has no `cur realm`, so
  [`NewBanker`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.gno#L117) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.gno#L117) cannot be reached during a deploy, and `OriginSendObserved` is nil there so
  the marks are no-ops. The cost is the next reader who extends the payable check to deploys
  and finds the address gate passing while the path gate never matches. Fix: set both or
  neither.
  </details>

- **[unrelated to the send-envelope]** [`docs/MANIFESTO.md:188-189`](https://github.com/gnolang/gno/blob/f6dd8ad37/docs/MANIFESTO.md?plain=1#L188-L189) · [↗](../../../../../.worktrees/gno-review-6062/docs/MANIFESTO.md#L188-L189) — a manifesto paragraph loses an external link and gets rewrapped inside a funds-safety change, which is what `git blame` will show for that paragraph.
  <details><summary>details</summary>

  The edit is unrelated to all three defects and to every test the branch touches. It also leaves
  a double space after `weapons.` on the next line. Fix: split it out, or say in the body why it rides
  along.
  </details>

## Verified

- The lifetime pin holds against four persistence routes. `esc.SaveClosure`, `esc.SaveDeep`,
  `esc.SaveMap` and `esc.SaveMethod` in [`tests/zz_originsend_escape.txtar`](tests/zz_originsend_escape.txtar) each abort with
  `cannot persist realm value`, `esc.Revive` finds nothing in any of the four stashes, and the
  victim treasury reads `balance=1000000` unchanged. The same file at the merge base 754780601
  gets `OK!` on the closure route.
- Same-message delegation still works: `esc.Delegate` hands the banker to another realm that
  spends it from the recipient's address inside the message and returns `OK!`.
- The pinned realm handle is not a capability leak to the receiving realm. `rlm` is an
  unexported field of the unexported `banker` type, gno has no reflection, and the only read is
  the nil check at [`banker.gno:241`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.gno#L241) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.gno#L241).
- Adding two error types to [`package.go`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/package.go#L32-L33) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/package.go#L32-L33) does not move any existing encoding: amino
  derives a type URL from the registered full name at [`pkg.go:337-340`](https://github.com/gnolang/gno/blob/f6dd8ad37/tm2/pkg/amino/pkg/pkg.go#L337-L340) · [↗](../../../../../.worktrees/gno-review-6062/tm2/pkg/amino/pkg/pkg.go#L337-L340), never from the
  registration index. Both new types are empty structs, so the message text stays out of the
  hashed result and rides `Result.Log`, the same split
  [`errors.go:27-33`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/errors.go#L27-L33) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/errors.go#L27-L33) describes for `TypeCheckError`.
- `MsgRun` cannot spend against its envelope: [`keeper.go:1113`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L1113) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L1113) leaves the recipient empty and
  `pkgAddr` is the caller, so the transfer is a self-transfer and the gate at
  [`banker.go:83`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/banker/banker.go#L83) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/banker/banker.go#L83) refuses any address.
- Green at f6dd8ad37: `go test ./gno.land/pkg/sdk/vm/...`, `./gnovm/pkg/test/...`,
  `./gnovm/stdlibs/...`, `contribs/gnodev ./pkg/proxy/...`, and each txtar the diff adds or edits
  run one at a time. Two txtars the branch does not touch,
  `update_storage_params` and `storage_deposit_price_change`, died on `connection refused` when
  the whole suite ran in parallel on this box and pass alone; reported as environment, not as a
  result.

## Invariant catalog

| Class | Touched | Outcome |
|---|---|---|
| Determinism | yes | One string compare and one address compare on the marking path; `send.String()` reaches only the unhashed log. No map order, clock, randomness or pointer identity. |
| Gas | yes | `NewBanker` gains a branch and a field write, so every construction costs a few more ops. Nothing is charged after the work it bills and nothing can overflow. |
| Realm state safety | yes | The rejection returns before `processStorageDeposit` and the message is cache-wrapped, so the credit is discarded; `payable_observed_send.txtar` reads `balance=500` after the refused call. The observed flag is set-only, so re-entry cannot clear it. |
| Caller & access control | yes | `NewBanker` still gates `BankerTypeOriginSend` on `rlm.Previous().IsUserCall()`, the strict predicate. The new gate adds an address comparison and no stack walk. The attribution mismatch is the Warning above. |
| Coin & banker | yes | The limit check and `OriginSendSpent` accumulation are unchanged and still run after the mark. `!send.IsZero()` treats an all-zero `Coins` as absent. |
| Storage deposit | yes | Ordering only: a refused message never reaches `processStorageDeposit`. Ratio and overflow math untouched. |
| Global mutable state | yes | `OriginSendObserved` is a per-message allocation, never a package-level var. |
| Error & panic handling | yes | Go paths return `ErrUnobservedSend` and `ErrUnspendableSend`; the gno path panics. No swallowed error. |
| VM-fault recoverability | yes | The new banker gate uses `m.PanicString`, so gno `recover()` catches it, matching the limit check beside it. |
| VM semantics vs Go | no | No interpreter change; `banker.gno` gains a field. |
| Type-check & preprocess | yes | The quarantined `banktest` filetests compile and pass under `gno test`. |
| Realm audit patterns | no | The branch adds no realm outside test fixtures. |

## Scope

Fifteen files carry test expectations the branch edits rather than adds.

| File | Edit | Follows from |
|---|---|---|
| `contribs/gnodev/pkg/proxy/path_interceptor_test.go` | three `NewMsgCall` sends dropped | defect two |
| `gno.land/pkg/gnoclient/integration_test.go` | deploy send dropped, package-address assertion removed, deployer balance +1000000 | defect three |
| `gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go` | pinned hash bumped | defect one, stdlib source is genesis state |
| `gno.land/pkg/sdk/vm/keeper_test.go` | `Echo` and `Absorb` read the envelope, `Do` called with no send | defect two |
| `gno.land/pkg/integration/testdata/user_journey.txtar` | `-send` dropped from `TransferOwnership` | defect two |
| `gno.land/pkg/integration/testdata/valopers.txtar` | `-send` dropped from the passing `Register` | defect two |
| `banker_persistence.txtar`, five `allowancesender_*.txtar` | `_ = unsafe.OriginSend()` added | defect two |
| three quarantined `banktest` filetests | `testing.SetRealm(testing.NewCodeRealm(...))` added | defect one, harness change |

One of these loses coverage. [`valopers.txtar:40-45`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/valopers.txtar#L40-L45) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/valopers.txtar#L40-L45) no longer attaches a payment to `Register`,
and no accepted registration carrying one is covered anywhere: the three subtests that raise the
fee all assert a rejection, and `successful registration` at
[`valopers_test.gno:154-160`](https://github.com/gnolang/gno/blob/f6dd8ad37/examples/gno.land/r/gnops/valopers/valopers_test.gno#L154-L160) · [↗](../../../../../.worktrees/gno-review-6062/examples/gno.land/r/gnops/valopers/valopers_test.gno#L154-L160) runs with the fee at zero, so `Register` never reaches its
envelope read. That gap predates the branch. `integration_test.go` drops the
only assertion that a deploy-time send lands in the package address; [`addpkg_unspendable_send.txtar:30-31`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/addpkg_unspendable_send.txtar#L30-L31) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/addpkg_unspendable_send.txtar#L30-L31)
replaces it for realms with `balance=700000`.

Everything else follows from one of the three defects except two files.
[`AGENTS.md:54-59`](https://github.com/gnolang/gno/blob/f6dd8ad37/AGENTS.md?plain=1#L54-L59) · [↗](../../../../../.worktrees/gno-review-6062/AGENTS.md#L54-L59) adds a verification rule, which the PR body ties to the branch's own
history. [`docs/MANIFESTO.md:188`](https://github.com/gnolang/gno/blob/f6dd8ad37/docs/MANIFESTO.md?plain=1#L188) · [↗](../../../../../.worktrees/gno-review-6062/docs/MANIFESTO.md#L188) is the Nit above.

## Open questions

- `-send` on a deploy is documented as a transfer to the realm being deployed, in
  [`interact-with-gnokey.md:166`](https://github.com/gnolang/gno/blob/f6dd8ad37/docs/users/interact-with-gnokey.md?plain=1#L166) · [↗](../../../../../.worktrees/gno-review-6062/docs/users/interact-with-gnokey.md#L166) and
  [`getting-started.md:356`](https://github.com/gnolang/gno/blob/f6dd8ad37/docs/builders/getting-started.md?plain=1#L356) · [↗](../../../../../.worktrees/gno-review-6062/docs/builders/getting-started.md#L356). Neither promises a `p/` deploy works, so both stay true; a
  follow-up doc naming the new refusal on a call would still save a support round. Not posted, no
  line in this diff carries it.
- [`valopers.txtar:19`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/valopers.txtar#L19) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/valopers.txtar#L19), [`:25`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/valopers.txtar#L25) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/valopers.txtar#L25) and [`:31`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/valopers.txtar#L31) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/integration/testdata/valopers.txtar#L31) still attach `20000000ugnot` to calls that fail inside the
  VM, so the keeper check is never reached and nothing strands. Consistent with the passing case
  would mean dropping those too. Not posted, no behaviour turns on it.
