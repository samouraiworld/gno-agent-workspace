# PR #5405: fix(tm2): print hashes and byte values as base64 instead of hex

**URL:** https://github.com/gnolang/gno/pull/5405
**Author:** thehowl | **Base:** master | **Files:** 8 | **+27 -17**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary
This PR converts hash and raw byte value formatting from uppercase hex (`%X`) to base64 (`base64.StdEncoding`) across several display/logging paths in `tm2/pkg/bft/`. The motivation is consistency with the RPC layer, which already encodes all `[]byte` values as base64 via Amino's JSON marshaller.

The changes span 8 files across 4 packages:
- **consensus/replay.go** — `appHash` in two ABCI handshake log lines
- **consensus/state_test.go** — block hash in two `t.Logf` diagnostic lines
- **mempool/clist_mempool.go** — `txID()` function used in mempool log labels
- **state/errors.go** — `BlockHashMismatchError.Error()` message
- **state/execution.go** — `appHash` in "Committed state" log line
- **types/block.go** — `Data.StringIndented()` tx hashes and `BlockID.String()`
- **types/vote.go** — `Vote.String()` validator address, block hash, and signature fingerprints
- **types/proposal.go** — `Proposal.String()` signature fingerprint

The PR description correctly states no consensus consequences — all changes are on display/logging paths. However, two existing snapshot tests (`TestVoteString`, `TestProposalString`) that compare exact `String()` output against hardcoded hex values were not updated, causing test failures. Additionally, the conversion is incomplete: `PartSetHeader.String()`, `LastStateMismatchError.Error()`, and `NoTxResultForHashError.Error()` still use `%X`, creating mixed-format output in composite strings.

## Test Results
- **Existing tests:** FAIL
  - `TestVoteString` — FAIL (`tm2/pkg/bft/types/vote_test.go:282`): expected hex `6AF1F4111082`, got base64 `avH0ERCC`
  - `TestProposalString` — FAIL (`tm2/pkg/bft/types/proposal_test.go:48`): expected hex `010203`/`000000000000`, got base64 `AQID`/`AAAAAAAA`
- **Edge-case tests:** skipped

## Critical (must fix)
- [ ] `tm2/pkg/bft/types/vote_test.go:282` — `TestVoteString` fails because expected strings still contain hex values. The test must be updated to expect base64 output.
- [ ] `tm2/pkg/bft/types/proposal_test.go:48` — `TestProposalString` fails for the same reason. The expected string contains hex like `010203`, `626C6F636B70`, `000000000000` which are now base64-encoded.

## Warnings (should fix)
- [ ] `tm2/pkg/bft/types/part_set.go:63` — `PartSetHeader.String()` still uses `%X` for `fingerprint(psh.Hash)`. Since `BlockID.String()` now uses base64 but embeds `PartsHeader` via `%v` (which calls `PartSetHeader.String()`), the output is now a mix of base64 (for `blockID.Hash`) and hex (for `PartsHeader.Hash`). This inconsistency is visible in `Proposal.String()` output: `Proposal{12345/23456 (AQID:111:626C6F636B70, -1) AAAAAAAA @ ...}` — `AQID` is base64 but `626C6F636B70` is still hex from `PartSetHeader`.
- [ ] `tm2/pkg/bft/state/errors.go:64-65` — `LastStateMismatchError.Error()` still uses `%X` for `Core` and `App` hashes. This is inconsistent with `BlockHashMismatchError` (line 57) which was converted.
- [ ] `tm2/pkg/bft/state/errors.go:85` — `NoTxResultForHashError.Error()` still uses `%X`. Same inconsistency.
- [ ] `tm2/pkg/bft/state/validation_test.go:114,119` — These tests construct expected error strings using `%X` format for hash validation errors. If `BlockHashMismatchError.Error()` changes to base64, the test expectations may also need updating.

## Nits
- [ ] `tm2/pkg/bft/types/block.go:726` — `Data.StringIndented()` prints `data.hash` with `%v` (not `%X` or base64): `fmt.Sprintf("Data{\n%s  %v\n%s}", ...)`. For a `[]byte`, `%v` prints as `[10 20 ...]` — a third format entirely. This pre-exists the PR but is worth noting as another inconsistency.

## Missing Tests
- [ ] The two failing snapshot tests (`TestVoteString`, `TestProposalString`) need updated expected strings matching the new base64 format.

## Suggestions
- Convert `PartSetHeader.String()` at `part_set.go:63` as well, to avoid mixed hex/base64 in composite `String()` outputs.
- Convert `LastStateMismatchError` and `NoTxResultForHashError` for full consistency.
- Consider whether this is a breaking change for any tooling that parses log output or error messages (e.g., monitoring dashboards, log parsers). The PR description correctly notes no consensus impact, but operational tooling may need updating.
- The remaining ~15 `%X` sites in `tm2/pkg/bft/` (in test files, panic messages, etc.) could be addressed in a follow-up for completeness.

## Questions for Author
- Was the decision to keep `PartSetHeader.String()` on hex intentional or an oversight?
- Any reason `LastStateMismatchError` and `NoTxResultForHashError` were left on hex?

## Verdict
REQUEST CHANGES — Two snapshot tests fail; the hex-to-base64 conversion is incomplete, leaving mixed-format output in composite strings.
