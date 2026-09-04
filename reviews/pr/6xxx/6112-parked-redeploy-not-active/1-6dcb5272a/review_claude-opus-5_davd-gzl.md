# PR [#6112](https://github.com/gnolang/gno/pull/6112): fix(gpao): stop reporting a parked redeploy as already active

URL: https://github.com/gnolang/gno/pull/6112
Author: gfanton | Base: master | Files: 3 | +216 -17
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 6dcb5272a (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6112 6dcb5272a`
Overview: [overview](../overview.md)

## Overview

gpao approves packages parked by a chain running the `inert` code submission
policy. Before sending an
[`MsgEnablePackage`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/msgs.go#L284)
it asks whether the chain already holds the package live, and it asked
[`vm/qfile`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/handler.go#L113),
which reads the live key space alone. A private realm can be live and carry a
parked redeploy at the same time, so that probe answered "live" for a path with
a redeploy still waiting: gpao recorded `approved` with the reason "already
active on-chain" and never sent the message. This change swaps the probe for
[`vm/qpkgmeta_json`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/handler.go#L122),
which reads both key spaces, and treats only "live with nothing pending" as
settled.

**Verdict: APPROVE** — the redeploy is enabled in the cell that used to drop it,
and the one behaviour this costs is overwritten by the next candidate at the
same path (1 nit).

## Verify first

- [`contribs/gpao/oracle.go:662`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L662) · [↗](../../../../../.worktrees/gno-review-6112/contribs/gpao/oracle.go#L662) — the settled test is `Status == live && !Pending`. Confirm no fifth `PackageMeta` shape exists by reading the status assignments in [`keeper_inert.go:524-534`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L524-L534); a shape off that list falls through to "not settled", which sends an enable.
- [`contribs/gpao/oracle.go:506`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L506) · [↗](../../../../../.worktrees/gno-review-6112/contribs/gpao/oracle.go#L506) — the skip that keeps a catch-up run inside `-max-spend`. Run [`casespace_test.go`](tests/casespace_test.go) and read its `spend charged` column: [the debit lands before the broadcast](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L534), so every cell answering "not settled" costs allowance whether or not a transaction goes out.

## Summary

The bug is one query reading one key space.
[`QueryFile`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper.go#L1798)
resolves against the live store, and
[`AddPackage`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper.go#L730)
refuses to park over a live package only when the live one is public, so a
private realm holds a live blob and a parked blob at once and `vm/qfile` reports
the first. The replacement probe,
[`QueryPackageMeta`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L519),
reads both spaces and sets
[`Pending`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L531)
from the parked one even when the live one answers. The fix is
[`isSettled`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L642)
plus the node-free classifier
[`pkgMetaSettled`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L657),
and the call site keeps the old fail-open posture: a query error, a response
error and unparsable bytes all send the enable.

## Benchmarks / Numbers

Every row is a real chain state built on an in-memory gno.land node under the
`inert` policy and driven through
[`handleCandidate`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L443).
`old isActive` is the deleted probe, copied verbatim into the harness, so both
answers come from one run against one chain. Harness:
[`casespace_test.go`](tests/casespace_test.go).

| chain at the path | candidate | old `isActive` | new `isSettled` | board status | board reason | spend debited |
| --- | --- | --- | --- | --- | --- | --- |
| absent | any | false | false | `pending` | `simulate says the enable would fail: invalid package path` | 1000000 |
| inert, nothing live | the parked blob | false | false | `approved` | | 1000000 |
| live public, nothing parked | the live blob | true | true | `approved` | `already active on-chain` | 0 |
| live private, nothing parked | the live blob | true | true | `approved` | `already active on-chain` | 0 |
| live private, redeploy parked | the parked v2 | true | **false** | `approved` | | 1000000 |
| live private, redeploy parked | the live v1 | true | **false** | `pending` | `simulate says the enable would fail: invalid package` | 1000000 |
| the same path, next candidate | the parked v2 | true | false | `approved` | | 1000000 |
| live private, nothing parked | a superseded v1 | true | true | `approved` | `already active on-chain` | 0 |

Row five is the fix: the redeploy is enabled instead of dropped. Row six is what
that costs, and row seven measures how long the cost lasts. The board is
[keyed by path](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/status.go#L59),
so the next candidate at the same path replaces the misleading entry, and the
parked blob's `MsgAddPackage` is always the later block, so a catch-up run
reaching row six always reaches row seven after it.

The live-public-and-parked cell is unreachable through `MsgAddPackage`. The
harness asserts that refusal directly. The code comment at
[`oracle.go:632-637`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L632-L637)
names a governance route that reaches it, and that route is read from
[`EnablePackage`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L170),
not built.

## Nits

- **[submitter-facing reporting]** [`contribs/gpao/oracle.go:810`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L810) · [↗](../../../../../.worktrees/gno-review-6112/contribs/gpao/oracle.go#L810) — the status board's reason for the newly reachable cell is `invalid package`, and the chain's own sentence naming the two hashes is dropped.
  <details><summary>details</summary>

  The pull request body has the replayed candidate recording `pending` with the
  chain's "it changed after review" reason. What the board carries is
  `simulate says the enable would fail: invalid package`. A submitter reading
  `/status/<pkgpath>` for a realm that is live, with only its redeploy waiting,
  gets `pending` and a string with no path, no hash and no cause in it.

  `enable` wraps `sim.Error`, the typed error, and the sentence
  [`EnablePackage`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L120-L123)
  wrote sits in `sim.Log`, which nothing reads. Both strings from one run of
  [`casespace_test.go`](tests/casespace_test.go):

  ```
  sim.Error = invalid package
  sim.Log   = msg:0,success:false,log:--= Error =--
  Data: vm.InvalidPackageError{abciError:vm.abciError{}}
  Msg Traces:
      0  gno/gno.land/pkg/sdk/vm/errors.go:104 - the parked source at
         gno.land/r/test/stale is not what was approved (approved cee6a23c...,
         parked daf4644f...); it changed after review
  ```

  This predates the branch and the absent cell reaches it too. It is in scope
  because the branch routes a class of live private realms onto that reporting
  path for the first time. Fix: carry `sim.Log` into the returned error so the
  board reason names what the chain refused.
  </details>

## Verified

- The catch-up skip survives the change for every settled state.
  [`EnablePackage`](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L367)
  deletes the parked blob on success, so `Pending` falls back to false and the
  three settled cells above still skip at zero spend. A fix that left the parked
  blob in place would have turned every already-approved package into a repeat
  enable attempt, and no test in the branch would have said so.
- `vm/qpkgmeta_json` and the `inert` policy shipped in one commit, 132e9a0ec, so
  no chain gpao can usefully watch answers `unknown vm query endpoint` for the
  new probe and falls open on every candidate.
  `git log -S QueryPackageMetaJSON -- gno.land/pkg/sdk/vm/handler.go`.
- Tests green at 6dcb5272a in `contribs/gpao`:
  [`TestRedeployParkedOverLivePrivateRealmIsEnabled`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/issettled_test.go#L41),
  [`TestPkgMetaSettled`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/issettled_test.go#L160)
  and `TestPreflightCaseSpace`.

## Open questions

- The last row of the table is the failure the branch fixes, in the one case the
  branch leaves standing: the board records `approved` and "already active
  on-chain" for a candidate whose bytes were replaced before any enable landed,
  so a submission that never went live is reported as live. Only the creator can
  replace their own parked bytes, and the board documents itself as
  [the latest word on a path](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/status.go#L59)
  rather than on a submission. Not posted: the body already declines to make the
  pre-flight content-aware, and this is that decision rather than an oversight.
- Nothing pins the stale-replay row. A subtest asserting the board entry for a
  replayed candidate at a live-and-parked path would freeze the cost the body
  accepts, and [`casespace_test.go`](tests/casespace_test.go) is the shape. Not
  posted: the behaviour is deliberate and the description states it.
- The doc comment on
  [`TestPkgMetaSettled`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/issettled_test.go#L158-L159)
  says it covers "the three answers vm/qpkgmeta_json can give and the two
  failure shapes". The table under it holds four `PackageMeta` shapes and one
  failure shape, and the two failure shapes it means, a transport error and a
  response error, live in `isSettled` where no test reaches them. Not posted: a
  finding about a code comment's wording changes no behaviour.
- `isSettled` takes a `ctx` it never uses, inherited from `isActive`, because
  `gnoclient.Query` has no context parameter. Not posted: no linter in
  [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/6dcb5272a/.github/golangci.yml#L13-L14)
  reports an unused parameter, and no change is needed.
