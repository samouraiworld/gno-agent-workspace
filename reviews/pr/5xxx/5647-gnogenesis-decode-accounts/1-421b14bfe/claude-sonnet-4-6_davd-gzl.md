# PR #5647: fix(gnogenesis): decode auth/accounts as gnoland.GnoAccount

**URL:** https://github.com/gnolang/gno/pull/5647
**Author:** @gfanton | **Base:** master | **Files:** 1 | **+7 -11**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

The `queryAccountAtHeight` function in `contribs/gnogenesis/internal/fork/source_rpc.go` fetches account state from a live node RPC endpoint (`auth/accounts/<addr>`) to resolve signer `accNum` and `finalSeq` when generating hardfork genesis files. Before this PR, the function attempted to decode the response in two passes: first as `struct { BaseAccount std.BaseAccount }`, then as `std.BaseAccount` directly. Both paths were silently broken.

**Root cause:** The auth handler (`tm2/pkg/sdk/auth/handler.go:184`) calls `amino.MarshalJSONIndent(ah.acck.GetAccount(ctx, addr), ...)`. On gno.land the keeper returns `*gnoland.GnoAccount` (which embeds `std.BaseAccount` and adds an `Attributes BitSet` field). The amino JSON codec serializes this concrete struct as:

```json
{
  "BaseAccount": {
    "address": "...", "coins": "...", "public_key": null,
    "account_number": "0", "sequence": "0"
  },
  "attributes": "0"
}
```

Amino's struct decoder (`decodeReflectJSONStruct`) enforces a strict no-unknown-fields policy (returns error on any unknown key). This means:

- The wrapper `struct { BaseAccount std.BaseAccount }` failed because it didn't know the `"attributes"` key.
- The `std.BaseAccount` fallback also failed because it didn't know the `"BaseAccount"` key.

Both errors were silently swallowed and `queryAccountAtHeight` returned `nil` for every account, so every signer was assigned `accNum=0` and `finalSeq=0` — producing invalid hardfork genesis files.

**The fix:** Decode directly into `gnoland.GnoAccount`, which matches the server's concrete type exactly. This is identical to the pattern already used in `gno.land/pkg/gnoclient/client_queries.go:59-65` (`gnoclient.QueryAccount`). A defensive `acc.Address.IsZero()` guard is added to reject a decode that produced no meaningful address.

The PR eliminates dead code (the try-wrapper-first dance), is correct for all existing `GnoAccount` data (base fields + attributes), and matches the established pattern in gnoclient.

**Verified by experiment:** Running `amino.MarshalJSONIndent` on a `std.Account` interface holding `*GnoAccount` produces exactly the nested-struct format shown above. Decoding into `gnoland.GnoAccount` succeeds and round-trips `attributes` correctly. Both the old wrapper and the old BaseAccount fallback fail with "unknown JSON field" errors.

## Test Results

- **Existing tests:** PASS — `go test ./internal/fork/...` passes in 10s. No regressions.
- **Edge-case tests:** Manual experiment confirmed: `amino.UnmarshalJSON(serverJSON, &gnolandGnoAccount)` correctly decodes address, coins, and attributes. Both old decode paths fail. Not written as persistent files per parent agent instructions, but results are embedded above.

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `contribs/gnogenesis/internal/fork/source_rpc.go:277` — The `acc.Address.IsZero()` guard after successful decode silently returns `nil` with no warning to the caller or to the IO writer. If this edge case ever fires (e.g. server returned an account with no address due to a node bug), the signer falls back to `accNum=0/finalSeq=0`, which is the same wrong state this PR fixes. At minimum, log a warning like the decode-error path does above it. Low urgency since the `IsZero` case is extremely unlikely for a well-formed node response, but it would help debuggability.

## Nits

- [ ] `contribs/gnogenesis/internal/fork/source_rpc.go:269-280` — The code comment says "the auth handler returns the concrete account type the gno.land app installs" — accurate, but adding a note pointing to `gnoclient.QueryAccount` (same pattern) would help future readers understand this isn't a novel choice.

## Missing Tests

- [ ] `contribs/gnogenesis/internal/fork/source_rpc.go` — No unit test for `queryAccountAtHeight` decode path. A table-driven test that calls `queryAccountAtHeight` against an `httptest.Server` returning `amino.MarshalJSONIndent` of a `GnoAccount` (with and without non-zero `Attributes`) would lock in this fix and catch regressions. The function is on an unexported method on an unexported type, so a test in the same package (`package fork`) would work directly.

## Suggestions

- The existing `gnoclient.QueryAccount` returns `&qret.BaseAccount` without the `IsZero` guard (it only checks for empty/null response data). The PR adds the `IsZero` guard as extra defence. That's fine, but consider whether the two callsites should align — either both check `IsZero`, or neither does. The discrepancy is minor but could confuse future readers.

## Questions for Author

- Was there a hardfork genesis file generated with the broken `accNum=0` values that needs to be regenerated? The commit message mentions this produces files that `validateSignerInfo` rejects, so presumably any generated file was already invalid and this is a pre-merge fix only.
- Were there any other callers of the two-pass decode pattern elsewhere in gnogenesis that might have the same issue? A `grep` for `BaseAccount.*json` in `contribs/gnogenesis` would be worth running.

## Verdict

APPROVE — The fix is correct, well-motivated, and exactly mirrors the established `gnoclient.QueryAccount` pattern; the only gap is a missing unit test for the decode path.
