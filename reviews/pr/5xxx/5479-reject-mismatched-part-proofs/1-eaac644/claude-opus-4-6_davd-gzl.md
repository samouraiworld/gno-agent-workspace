# PR #5479: fix(tm2): reject block parts with mismatched proofs in AddPart

**URL:** https://github.com/gnolang/gno/pull/5479
**Author:** thehowl | **Base:** master | **Files:** 2 | **+58 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a security vulnerability in `PartSet.AddPart()` where a Byzantine peer could send a `Part` with `part.Index != part.Proof.Index` and pass the merkle verification, but have its bytes stored at the wrong slot in the part set.

The root cause: `SimpleProof.Verify()` (at `tm2/pkg/crypto/merkle/simple_proof.go:73`) uses `sp.Index` and `sp.Total` from the `SimpleProof` struct internally (via `computeHashFromAunts` at line 143), NOT the `part.Index` from the `Part` struct. The method's own doc comment at line 72 even says: "Check sp.Index/sp.Total manually if needed." This means a Byzantine peer could craft a Part with `part.Index=0` but `part.Proof.Index=1` (carrying the legitimate proof for part 1). The merkle check would pass using the proof's internal index, but the bytes would be stored at slot 0 (the wrong position). Once all parts are assembled, the block would be undecodable because bytes are in wrong slots.

The fix adds two guard checks at `part_set.go:216-221` before the merkle verification:
1. `part.Proof.Index != part.Index` — rejects index mismatch
2. `part.Proof.Total != ps.total` — rejects total mismatch

Both return `ErrPartSetInvalidProof`. This is defense-in-depth: even if `Verify()` were later changed to validate these fields, the explicit check ensures the storage index matches the proof index.

Two new tests are added: `TestAddPartSwappedIndex` (swapped index attack) and `TestAddPartWrongTotal` (tampered total).

## Test Results
- **Existing tests:** PASS — `go test -v -run 'TestPartSet|TestAddPart' ./tm2/pkg/bft/types/...` all pass in the worktree.
- **CI status:** All checks pass (green).
- **Edge-case tests:** Skipped — the two new tests cover the exact attack vector; the fix is a simple guard clause.

## Critical (must fix)

None

## Warnings (should fix)

None

## Nits

- [ ] `tm2/pkg/bft/types/part_set.go:216-221` — The two checks share the same error (`ErrPartSetInvalidProof`). For debugging, it might be useful to differentiate between an index mismatch and a total mismatch (e.g., separate error variables or wrapping with context). Minor — the current approach is consistent with how the existing merkle check at line 224 also returns `ErrPartSetInvalidProof`.

## Missing Tests

- [ ] No test for the scenario where `part.Proof.Total` is *less* than `ps.total` (only greater-than is tested at `part_set_test.go:133`). The current check `part.Proof.Total != ps.total` covers both directions, so the code is correct, but an explicit test for `Total - 1` would be a small addition for completeness.

## Suggestions

- The `SimpleProof.Verify()` method at `tm2/pkg/crypto/merkle/simple_proof.go:73` could be hardened to optionally accept expected index/total parameters, making it impossible for callers to forget this validation. However, that's a larger change outside this PR's scope, and the current fix is sufficient.

## Questions for Author

None — the fix is clean, minimal, and directly addresses the vulnerability.

## Verdict

**APPROVE** — Targeted security fix for a real Byzantine attack vector. The guard checks are correct, well-commented, and well-tested. No regressions.
