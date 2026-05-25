# PR #5049: fix(gnokey): inject block height when not provided in ABCI requests

URL: https://github.com/gnolang/gno/pull/5049
Author: davd-gzl | Base: master | Files: 11 | +69 -76
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — fix is correct in shape but ships three real problems: error responses still leak `height: 0`, the `gnokey_qpaths.txtar` golden→regex conversion silently loses ordering/completeness checks, and the branch is now `mergeable: CONFLICTING` against master (Jan 2026 base vs current master with store-API + `haltTime` churn).

## Summary

ABCI queries returned `height: 0` because `handleQueryApp` and several handlers (`auth`, `bank`, `params`, `vm`) never propagated the request height into the response. Fix injects `app.LastBlockHeight()` into `req.Height` in `handleQueryApp` when zero (mirroring existing logic in `handleQueryStore` and `handleQueryCustom`), and adds `res.Height = req.Height` in every successful handler return path. Integration test goldens were swapped for per-line regex stdout assertions to side-step a startup race: the genesis txs trigger an immediate "proof block" (height 2), so queries may observe height 1 or 2 depending on timing.

```
Before:                          After:
client -> RequestQuery{H:0}      client -> RequestQuery{H:0}
  baseapp.Query                    baseapp.Query
    handleQueryCustom                handleQueryCustom
      req.H = LastBlockHeight()       req.H = LastBlockHeight()
      handler.Query(req)              handler.Query(req)
        res.Data = ...                  res.Height = req.Height  <-- new
        return res {H:0}                res.Data = ...
                                        return res {H:N}
```

## Glossary

- `RequestQuery` — ABCI query envelope; carries `Height int64` (0 = "latest" by convention).
- `ResponseQuery` — ABCI query reply; `Height` is what the client prints/uses for proof verification.
- `handleQueryApp` / `handleQueryStore` / `handleQueryCustom` — three branches in `BaseApp.Query` for `.app/*`, `.store/*`, and module-routed (`auth/`, `bank/`, `vm/`, `params/`) queries respectively.
- "proof block" — the immediate post-genesis block (height 2) produced because genesis txs change the app hash, so the very first user query lands on either height 1 or 2.

## Fix

`handleQueryApp` now mirrors the height-injection pattern that already existed in `handleQueryStore` and `handleQueryCustom`: when `req.Height == 0`, replace with `app.LastBlockHeight()` ([`baseapp.go:414-417`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/baseapp.go#L414-L417)). The four custom-route handlers (`auth/handler.go`, `bank/handler.go`, `params/handler.go`, `vm/handler.go`) each gain one line per success branch: `res.Height = req.Height` before the existing `res.Data = ...` (e.g. [`vm/handler.go:133`](../../../../../.worktrees/gno-review-5049/gno.land/pkg/sdk/vm/handler.go#L133), [`auth/handler.go:78`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/auth/handler.go#L78)). The load-bearing observation is that `handleQueryCustom` injects into the *request*, not the response — so each handler must echo the height back on its own. Five integration testdata files relax `height: 0` assertions to `height: [1-9][0-9]*`; `gnokey_qpaths.txtar` further drops `cmp` against eight golden files in favor of inline per-line `stdout` regexes.

## Critical (must fix)

- **[error path still returns height 0]** [`tm2/pkg/sdk/auth/handler.go:62`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/auth/handler.go#L62) — every handler error branch goes through `sdk.ABCIResponseQueryFromError`, which does not set `Height`, so failures still print `height: 0` and the fix is half-applied.
  <details><summary>details</summary>

  Look at the helper at [`tm2/pkg/sdk/helpers.go:65-69`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/helpers.go#L65-L69): `ABCIResponseQueryFromError` sets `res.Error` and `res.Log` and returns — `res.Height` stays zero-valued. Every `res = sdk.ABCIResponseQueryFromError(err)` in `auth`, `bank`, `vm`, `params` (and the `return sdk.ABCIResponseQueryFromError(err)` shortcuts) therefore reverts to the bug the PR is fixing the moment a query errors. From the user's perspective: `gnokey query auth/accounts/g1invalid...` still reports `height: 0` while `gnokey query auth/accounts/g1valid...` reports the real height. That's a worse UX than uniform `height: 0` — debuggers can't tell from the header alone whether they're seeing a stale-state response or an error.

  Fix: either teach `ABCIResponseQueryFromError` to take a `height int64` parameter (the cleanest answer — one site to change, all callers benefit), or set `res.Height = req.Height` on every error branch after the call. The handler-by-handler patch is bigger but doesn't touch a shared helper. Prefer the helper change.
  </details>

## Warnings (should fix)

- **[branch divergence — mergeable: CONFLICTING]** PR-wide — base is Jan 2026; current master has unrelated `haltTime`, store-API (`Get(nil, key)` -> `Get(key)`), and `trace` import churn in the same `baseapp.go` block.
  <details><summary>details</summary>

  `gh pr view 5049 --json mergeable` reports `CONFLICTING`. `git diff origin/master -- tm2/pkg/sdk/baseapp.go` shows the merge surface: import set (`os`, `syscall`, removed `store/trace`), `haltTime` field on `BaseApp`, and `mainStore.Get(nil, key)` -> `mainStore.Get(key)` in `initFromMainStore`. None of these collide semantically with the height fix, but they will need a manual rebase before merge. The PR description acknowledges TM2 CI was red on master at the time — that's been fixed since.

  Fix: rebase onto current `origin/master`, re-run the integration suite, force-push.
  </details>

- **[golden->regex weakens qpaths assertions]** [`gno.land/pkg/integration/testdata/gnokey_qpaths.txtar:15-85`](../../../../../.worktrees/gno-review-5049/gno.land/pkg/integration/testdata/gnokey_qpaths.txtar#L15-L85) — converting 8 `cmp stdout *.golden` to per-line `stdout 'regex'` drops ordering and completeness checks.
  <details><summary>details</summary>

  `cmp stdout aaa-qpaths.stdout.golden` asserted byte-for-byte equality: exactly these three paths, in this order, with no extras. The replacement is three independent `stdout 'path'` regexes, each of which matches if the substring appears anywhere — extra paths silently pass, missing paths between asserted ones silently pass, and reordering passes. For a `qpaths` endpoint whose contract is "return paths matching prefix X, sorted, deduplicated" this is meaningful regression in test strength.

  Concretely, [`testdata/gnokey_qpaths.txtar:54-57`](../../../../../.worktrees/gno-review-5049/gno.land/pkg/integration/testdata/gnokey_qpaths.txtar#L54-L57) is `stdout 'data: '` — the regex matches `data: ` followed by anything, so the "no matches" case (the old `ccc-qpaths.stdout.golden` was literally `height: 0\ndata: \n`) no longer asserts emptiness. Any future regression where `qpaths "gno.land/r/ccc"` accidentally returns a path with a leading space after `data: ` would slip through.

  Fix: keep `cmp stdout` against goldens for the body. The height race only affects the `height:` line — either (a) regex-replace just the height line via a post-processing step (`sed -i 's/^height: .*/height: <H>/' stdout` before `cmp`), or (b) split assertions: regex on the first line, `cmp` on the rest. Or (c) eliminate the race upstream by waiting for height >= 2 in the test harness before issuing queries.
  </details>

- **[height==0 conflated with "not provided"]** [`tm2/pkg/sdk/baseapp.go:415`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/baseapp.go#L415) — a client legitimately requesting `Height: 0` (e.g. pre-genesis snapshot, in a future feature) cannot distinguish from default.
  <details><summary>details</summary>

  The convention `Height == 0 means latest` is widely-used in Cosmos-derived chains but it's a sentinel, not a type-level invariant. This PR doesn't introduce the conflation — `handleQueryStore` and `handleQueryCustom` already do it — but it propagates it to the third branch (`handleQueryApp`) without comment. A `*int64` field, or a `HeightHint` enum, would be cleaner, but that's an ABCI types change with ecosystem-wide blast radius. Worth a code comment at minimum so the next reader knows the contract.

  Fix: add a one-line `// Height == 0 is the documented "latest block" sentinel; see ABCI Query convention.` above each of the three `if req.Height == 0` sites. Or open a follow-up to formalize the sentinel.
  </details>

- **[no unit test for height injection]** [`tm2/pkg/sdk/baseapp_test.go`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/baseapp_test.go) — `grep -n Height tm2/pkg/sdk/baseapp_test.go` returns zero hits for the new injection logic.
  <details><summary>details</summary>

  The fix is verified only through the txtar integration suite, which is slow and tests user-visible CLI output rather than the SDK contract. A direct Go test on `BaseApp.Query(abci.RequestQuery{Path: "auth/accounts/...", Height: 0})` asserting `res.Height > 0` would (a) run in milliseconds, (b) protect the contract independent of CLI formatting drift, and (c) cover all three branches (`handleQueryApp` / `handleQueryStore` / `handleQueryCustom`) and the four module handlers. There's an existing test scaffold at [`baseapp_test.go:464`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/baseapp_test.go#L464) (`testHandler`) you can reuse.

  Fix: add `TestBaseAppQueryInjectsHeight` exercising one call per branch, plus one assertion per handler that `res.Height` is non-zero on success. Combined with the Critical above, also assert that error responses propagate height once that's fixed.
  </details>

## Nits

- [`gno.land/pkg/sdk/vm/handler.go:133-261`](../../../../../.worktrees/gno-review-5049/gno.land/pkg/sdk/vm/handler.go#L133-L261) — seven copies of `res.Height = req.Height; res.Data = []byte(...)` could collapse into a small helper, e.g. `okResponse(req, []byte(result))`. Minor — current shape is greppable.
- `tm2/pkg/sdk/baseapp.go:414` — comment says "manually inject the latest" — "manually" is filler. "inject latest block height" reads cleaner.
- `gno.land/pkg/integration/testdata/gnokey_qpaths.txtar:97-99` — the `invalid-name-qpaths.*.golden` blocks are still defined but no longer matched by `cmp` (the diff shows them retained but other goldens removed) — verify the surviving ones are actually used.

## Missing Tests

- **[handler-level unit test]** [`tm2/pkg/sdk/auth/handler_test.go`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/auth/handler_test.go) — `queryAccount`/`queryGasPrice` should be tested with both `Height: 0` and `Height: N` inputs, asserting `res.Height` echoes the request.
  <details><summary>details</summary>

  Same gap for `bank.queryBalance`, `params.Query`, and the seven `vm/handler.go` queries. The fix is one line per handler — the test that this line stays in place is also one line per handler. Cheap insurance against a future "clean up redundant assignments" refactor silently regressing the bug.
  </details>

- **[error-path height assertion]** [`tm2/pkg/sdk/helpers.go:65`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/helpers.go#L65) — once Critical is addressed, add an assertion that error responses also carry the request height.

## Suggestions

- [`tm2/pkg/sdk/baseapp.go:414`](../../../../../.worktrees/gno-review-5049/tm2/pkg/sdk/baseapp.go#L414) — extract the `if req.Height == 0 { req.Height = app.LastBlockHeight() }` block into a small `app.resolveQueryHeight(req)` method. Three call sites with identical logic invite drift; one method documents the sentinel once and centralizes the contract.

## Questions for Author

- Why not also propagate height in `ABCIResponseQueryFromError`? The reviewer thread on `baseapp.go:519` settled on "let handlers build their response" — but failing branches use the same helper, so the height is silently dropped only on errors. Was that intentional or an oversight?
- The proof-block race is documented in commit `8847cc6c` but only papered over with regex. Is there a follow-up to wait for height >= 2 in the txtar harness, or to delay the genesis-induced app-hash change? The regex workaround propagates into every future query test that cares about height.
- For `vm/qpaths`, was the goldens removal driven only by the height race, or also by ordering instability in the path list?
