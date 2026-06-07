# PR #5049: fix(gnokey): inject block height when not provided in ABCI requests

URL: https://github.com/gnolang/gno/pull/5049
Author: davd-gzl | Base: master | Files: 16 | +175 -128
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `141424b0` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5049 141424b0`

**Verdict: APPROVE** — round-2 re-review. All three round-1 blockers are now closed: branch merged master (`mergeable: MERGEABLE`), the unit-test gap is filled (`TestBaseAppQueryInjectsHeight`), and the error-path `height: 0` is resolved-by-design per mvallenet (handlers own their response; errors go through `ABCIResponseQueryFromError` which intentionally omits height). The three new master-side vm handlers (`qeval_json`/`qobject_json`/`qobject_binary`) now echo height like their siblings. Remaining items are Nits: the golden→regex conversion drops byte-exact/ordering checks (accepted by author for the proof-block race), and the height==0 sentinel is still uncommented. CI red is infra-only (Codecov GPG outage + docs link-checker), not this PR.

## What changed since round 1 (`2f68c13` → `141424b0`)

- **Merged `origin/master`** (`e001de2`): resolves the round-1 CONFLICTING warning. `gh pr view 5049 --json mergeable` now reports `MERGEABLE`.
- **Unit test simplified and de-paralleled** (`88d238a`, `73c8a0d`): `TestBaseAppQueryInjectsHeight` covers all three Query branches (`.app`/`.store`/custom) for both latest-injection and explicit-height-preservation. Closes the round-1 "no unit test" gap.
- **Three new vm handlers echo height** (`fd0ef18`): `queryEvalJSON`, `queryObjectJSON`, `queryObjectBinary` landed on master after round 1 without `res.Height`; now patched for consistency with the other seven vm queries.
- **Test assertions hardened/relaxed** (`11a9408`, `141424b`): `params_valset` asserts the single-validator-power-10 invariant instead of a byte-exact baseline (height advances, genesis key is per-run-random); vm-json queries pin `@type`/`ObjectID` substrings plus a height regex.
- **auth handler threads `req`** into the master-added `querySessions`/`querySession` so they echo height too ([`auth/handler.go:225`](https://github.com/gnolang/gno/blob/141424b0/tm2/pkg/sdk/auth/handler.go#L225) · [↗](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/auth/handler.go#L225), [`:243`](https://github.com/gnolang/gno/blob/141424b0/tm2/pkg/sdk/auth/handler.go#L243) · [↗](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/auth/handler.go#L243)).

## Summary

ABCI queries returned `height: 0` because `handleQueryApp` and the module handlers never propagated the request height into the response. The fix injects `app.LastBlockHeight()` into `req.Height` when zero in `handleQueryApp` (mirroring `handleQueryStore`/`handleQueryCustom`), and sets `res.Height = req.Height` on every successful handler return across `auth`/`bank`/`params`/`vm`. Integration goldens were swapped for per-line regex stdout assertions to side-step the post-genesis "proof block" race (queries see height 1 or 2 depending on timing).

## Verification

Ran on `141424b0` in a clean worktree:

```
go test ./tm2/pkg/sdk/ -run TestBaseAppQueryInjectsHeight -v
  PASS (custom/store/app-version/app-explicit-height all green)

go test ./gno.land/pkg/integration/ -run 'TestTestdata/(gnokey_qpaths|qeval_json|qobject_json|qobject_binary|params_valset_multi_entry_same_op|adduserfrom|gnokey|gnoweb_airgapped|event_multi_msg)$'
  ok  github.com/gnolang/gno/gno.land/pkg/integration  8.982s
```

Coverage of every vm query handler's success path confirmed — all ten set `res.Height` before `res.Data` ([`vm/handler.go:142,154,192,205,218,231,244,282,296,309`](https://github.com/gnolang/gno/blob/141424b0/gno.land/pkg/sdk/vm/handler.go#L142) · [↗](../../../../../.worktrees/gno-review-5049/gno.land/pkg/sdk/vm/handler.go#L142)). The two `res.Data` sites without a height set are tx message results (`handleMsgCall`/`handleMsgRun`, `sdk.Result` not `ResponseQuery`), correctly excluded.

## CI status

`mergeStateStatus: BLOCKED` is the approval gate (review/triage-pending), not a test failure. The 15 red "test"/"build docker images" jobs all fail at the **Codecov upload step** (`Could not verify signature ... fail_ci_if_error: true`) — an infra-wide GPG outage hitting every PR, after the Go tests passed. The red `docs` job is the remote-link checker (`-treat-urls-as-err=true`) flagging pre-existing `rpc.gno.land:443` and `gno.land/r/...` URLs across `gas-fees.md`, `example-boards.md`, `gno-packages.md`, etc. — files this PR never touches. None of the failures are attributable to this change; matches the PR description's note that TM2/master CI is independently red.

## Critical (must fix)

None.

## Warnings (should fix)

None. (Round-1's three Warnings — CONFLICTING merge, golden→regex strength, no unit test — are resolved, accepted-as-Nit, and done respectively. See "What changed".)

## Nits

- [`gno.land/pkg/integration/testdata/qobject_json.txtar:28-32`](https://github.com/gnolang/gno/blob/141424b0/gno.land/pkg/integration/testdata/qobject_json.txtar#L28-L32) · [↗](../../../../../.worktrees/gno-review-5049/gno.land/pkg/integration/testdata/qobject_json.txtar#L28-L32) — the golden→regex conversion drops the deterministic `Hash` fields from the assertion, not just the racy `height` line. Only `height` actually races; `ObjectID`/`Hash`/`ModTime` are stable, so the body could have stayed `cmp` against a golden with just the height line regex-normalized. Net effect: a future amino-encoding regression in the inlined `HeapItemValue` wrapper (Hash drift, RefCount change) now slips past these tests. Same for `qobject_binary.txtar`. Accepted-by-author tradeoff for readability; flagging the coverage cost.
- [`gno.land/pkg/integration/testdata/gnokey_qpaths.txtar:54-56`](https://github.com/gnolang/gno/blob/141424b0/gno.land/pkg/integration/testdata/gnokey_qpaths.txtar#L54-L56) · [↗](../../../../../.worktrees/gno-review-5049/gno.land/pkg/integration/testdata/gnokey_qpaths.txtar#L54-L56) — `stdout 'data: '` for the `gno.land/r/ccc` (no-match) case asserts presence of the `data: ` line but no longer asserts emptiness: an accidental path leak after `data: ` would pass. The old `ccc-qpaths.stdout.golden` pinned `data: ` followed by nothing. Carried over from round 1; still a real (small) strength loss. The `! stderr '.+'` additions are a genuine improvement over the old `cmp stderr empty_file`, so stderr coverage is unchanged.
- [`tm2/pkg/sdk/baseapp.go:429`](https://github.com/gnolang/gno/blob/141424b0/tm2/pkg/sdk/baseapp.go#L429) · [↗](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/baseapp.go#L429) — the `if req.Height == 0` sentinel is now in all three branches (`handleQueryApp`/`handleQueryStore`/`handleQueryCustom`) with the same `// when a client did not provide a query height, manually inject the latest` comment, but none document that `0` is the ABCI "latest" convention rather than a literal genesis-height request. One line — `// Height == 0 is the documented "latest block" sentinel.` — would save the next reader the ambiguity. "manually" is still filler.

## Missing Tests

None blocking. `TestBaseAppQueryInjectsHeight` covers the three branches at the BaseApp level; the per-handler `res.Height = req.Height` lines are exercised through the integration suite. A direct handler-level Go test asserting each module handler echoes `req.Height` would be cheap insurance against a future "remove redundant assignment" refactor silently regressing the bug, but that's a hardening nice-to-have, not a gap that blocks merge.

## Suggestions

- [`gno.land/pkg/sdk/vm/handler.go:142-309`](https://github.com/gnolang/gno/blob/141424b0/gno.land/pkg/sdk/vm/handler.go#L142-L309) · [↗](../../../../../.worktrees/gno-review-5049/gno.land/pkg/sdk/vm/handler.go#L142-L309) — ten near-identical `res.Height = req.Height; res.Data = ...` pairs. A tiny `okResponse(req, data)` helper would centralize the echo and make "every success path carries height" structurally true rather than convention-by-repetition. Optional; current form is greppable and the redundancy is the point of being explicit.

## Questions for Author

None outstanding. The two round-1 open questions (error-path height, proof-block-race regex strategy) were settled in-thread: errors intentionally omit height (mvallenet), and the regex workaround is the accepted approach over a harness-level wait-for-height-2.
