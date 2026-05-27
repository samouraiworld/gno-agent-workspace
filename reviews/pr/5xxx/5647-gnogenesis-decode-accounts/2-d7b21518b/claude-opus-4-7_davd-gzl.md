# PR #5647: fix(gnogenesis): decode auth/accounts as gnoland.GnoAccount

URL: https://github.com/gnolang/gno/pull/5647
Author: @gfanton | Base: master | Files: 1 | +7 -11
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5647 d7b21518b` (then `gh -R gnolang/gno pr checkout 5647` inside it)

Verdict: APPROVE — fix is correct, mirrors the established `gnoclient.QueryAccount` pattern, and CI is green; only remaining gaps are a silent fallthrough on the `IsZero` guard and a missing unit test for the decode path.

## Summary

`queryAccountAtHeight` in [`source_rpc.go`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L252-L281) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L252-L281) queries `auth/accounts/<addr>` on a live node to resolve per-signer `accNum` and `finalSeq` for hardfork genesis export. The old decoder tried `struct{BaseAccount std.BaseAccount}` then bare `std.BaseAccount`; both failed on gno.land because the auth handler marshals `*gnoland.GnoAccount` (BaseAccount embedded + `Attributes BitSet`), and amino's strict-field policy rejects every unknown key. Every signer fell back to `accNum=0`, producing genesis files that `validateSignerInfo` rejects loudly when `balances[0].Address` is not a tx signer, or silently corrupting target state when it is. The fix decodes directly into `gnoland.GnoAccount` — same pattern as [`gnoclient.QueryAccount`](https://github.com/gnolang/gno/blob/d7b21518b/gno.land/pkg/gnoclient/client_queries.go#L59-L65) · [↗](../../../../../.worktrees/gno-review-5647/gno.land/pkg/gnoclient/client_queries.go#L59-L65) — and returns `&acc.BaseAccount`.

## Glossary

- `GnoAccount` — gno.land's concrete account type: `std.BaseAccount` + `Attributes BitSet` ([`types.go:48`](https://github.com/gnolang/gno/blob/d7b21518b/gno.land/pkg/gnoland/types.go#L48-L51) · [↗](../../../../../.worktrees/gno-review-5647/gno.land/pkg/gnoland/types.go#L48-L51)).
- `validateSignerInfo` — preflight added in #5511 that rejects genesis with two addresses claiming the same accNum ([`app.go:880`](https://github.com/gnolang/gno/blob/d7b21518b/gno.land/pkg/gnoland/app.go#L880) · [↗](../../../../../.worktrees/gno-review-5647/gno.land/pkg/gnoland/app.go#L880)).
- `signerState` — per-signer tracker holding `accNum` and `finalSeq` from the RPC query ([`source_rpc.go:50-56`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L50-L56) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L50-L56)).

## Fix

Before: two-pass decode (wrapper then bare BaseAccount), both rejected by amino because neither shape matches the on-wire `{"BaseAccount":{...},"attributes":...}`. After: single decode into `gnoland.GnoAccount`, returning the embedded `BaseAccount` pointer. The load-bearing fact is that the auth handler at [`handler.go:194-203`](https://github.com/gnolang/gno/blob/d7b21518b/tm2/pkg/sdk/auth/handler.go#L194-L203) · [↗](../../../../../.worktrees/gno-review-5647/tm2/pkg/sdk/auth/handler.go#L194-L203) marshals whatever `acck.GetAccount` returns, which on gno.land is always `*GnoAccount` because that's what `ProtoGnoAccount` registers at [`app.go:109`](https://github.com/gnolang/gno/blob/d7b21518b/gno.land/pkg/gnoland/app.go#L109) · [↗](../../../../../.worktrees/gno-review-5647/gno.land/pkg/gnoland/app.go#L109). A new `acc.Address.IsZero()` guard at [`source_rpc.go:277`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L277) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L277) returns nil for empty/null responses.

## Critical (must fix)

None.

## Warnings (should fix)

- [silent fallthrough re-creates the same accNum=0 bug] [`source_rpc.go:277`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L277) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L277) — `acc.Address.IsZero()` returns `nil` with no warning to the IO writer.
  <details><summary>details</summary>

  When the server returns `"null"` (unknown address) or a malformed-but-decodable payload, amino unmarshals into a zero-value `GnoAccount` and this branch fires. The caller at [`source_rpc.go:85-91`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L85-L91) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L85-L91) treats `nil` as "leave accNum=0", which is the exact bug this PR fixes — just gated on a different failure mode. The decode-error branch two lines above logs a warning; this one should too. Fix: emit `io.Printf("\n  WARNING: empty account %s at height %d\n", addr, height)` before the `return nil`.
  </details>

## Nits

- [`source_rpc.go:269-270`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L269-L270) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L269-L270) — comment doesn't mention that `gnoclient.QueryAccount` already uses this exact pattern; a one-line "matches gnoclient.QueryAccount" would help future readers see this isn't a novel choice.

## Missing Tests

- [no coverage of decode path] [`source_rpc.go:252-281`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L252-L281) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L252-L281) — `queryAccountAtHeight` has no unit test.
  <details><summary>details</summary>

  The whole point of this PR is that the previous decoder was silently broken for years and no test caught it. A table-driven test in `package fork` using `httptest.Server` returning `amino.MarshalJSONIndent(&gnoland.GnoAccount{...})` would lock in the fix and catch the next protocol drift (e.g. if `GnoAccount` grows another field, or if a different concrete type ever ships through this handler on gno.land). Cases worth covering: well-formed account, account with non-zero `Attributes`, `"null"` response, malformed JSON, RPC-level error.
  </details>

## Suggestions

- [`source_rpc.go:265-267`](https://github.com/gnolang/gno/blob/d7b21518b/contribs/gnogenesis/internal/fork/source_rpc.go#L265-L267) · [↗](../../../../../.worktrees/gno-review-5647/contribs/gnogenesis/internal/fork/source_rpc.go#L265-L267) — `gnoclient.QueryAccount` adds `string(qres.Response.Data) == "null"` alongside the length check at [`client_queries.go:55`](https://github.com/gnolang/gno/blob/d7b21518b/gno.land/pkg/gnoclient/client_queries.go#L55) · [↗](../../../../../.worktrees/gno-review-5647/gno.land/pkg/gnoclient/client_queries.go#L55); aligning the two callsites avoids relying on `IsZero` to catch the null case after a successful decode. Either both call sites pre-filter `"null"`, or both rely on the post-decode guard — pick one, document why.

## Questions for Author

- Any historical hardfork genesis files generated through this path that need regeneration, or is this strictly a pre-merge fix? The PR description suggests every prior `gnogenesis fork generate` against gno.land was producing wrong output; if any of those files were used downstream, they need to be flagged.
- `grep -rn "BaseAccount.*json" contribs/gnogenesis` to verify no other decode site in this contrib has the same shape mismatch — none surfaced in this PR's diff but worth one sweep.
