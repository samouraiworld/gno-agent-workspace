# PR #5575: WIP Feat/maxwell/genproto2 testability

**URL:** https://github.com/gnolang/gno/pull/5575
**Author:** ltzmaxwell | **Base:** master | **Files:** 11 | **+975 -304**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR improves the genproto2 amino codec in three areas:

1. **Centralizes the `IsByteLengthWrapped` predicate** — Previously, the "struct || time.Duration || (non-byte list)" check was duplicated verbatim across `gen_marshal.go`, `gen_unmarshal.go`, and `gen_size.go`. Any drift between copies caused wire/decode mismatches (PR #5569 was a nil_elements variant). The predicate is now a single method on `*TypeInfo` in `codec.go`, and all three generators call it.

2. **Adds `DumpWire` debug aid** — A new schema-less wire-format tracer (`dumpwire.go`) that parses raw proto3 TLV structure and produces annotated, indented output with byte offsets. Integrated into the `compareEncoding` test helper so byte-mismatch failures now show which field diverges instead of just hex dumps.

3. **Fixes nil-pointer unmarshal for `amino:"nil_elements"` + ByteLength-wrapped elements** — The old `writeUnpackedListUnmarshal` for pointer elements with `NilElements` only handled the `ertIsPointer` case by unconditionally decoding as a non-nil value, so a `0x00` nil sentinel was misinterpreted. The new code introduces a proper `fbz`-based nil check: `DecodeByteSlice` → if `len(fbz) == 0` store nil, else decode from `fbz`. This is applied for both `writeImplicit` and `isStructLike` (ByteLength-wrapped) element types. For non-ByteLength-wrapped pointer elements (e.g. `*int32`), the existing `bz[0] == 0x00` check remains.

Additionally, two new helper methods are extracted: `writeByteSliceElementPayloadDecode` and `writeImplicitStructPayloadDecode`, which take a `srcVar` parameter instead of hardcoding `"bz"` or `"fbz"`. This eliminates the variable-name collision that the old `writeImplicitStructDecode` had when called from contexts where `fbz` was already in scope. The generated `pb3_gen.go` files reflect the `fbz` → `_inner` rename in implicit-struct decode paths.

The PR depends on #5569 (fix: unmarshal amino nil pointers).

## Test Results

- **Existing tests:** PASS — all `tm2/pkg/amino/...` and `tm2/pkg/bft/types/...` tests pass locally.
- **Edge-case tests:** skipped (CI coverage is 82.4% patch, with gaps in `dumpwire.go` and `gen_unmarshal.go` noted by Codecov).
- **CI:** `check` and `main / lint` FAILED (likely unrelated format/lint issues in the large generated `pb3_gen.go` file).

## Critical (must fix)

- [ ] `tm2/pkg/amino/genproto2/gen_unmarshal.go:606` — The `bz[0] == 0x00` nil-sentinel check for non-ByteLength-wrapped pointer elements with `NilElements` is fragile. It reads the raw byte from `bz` (the outer stream) *before* consuming the field-key, so `bz[0]` is the first byte *after* the field key has already been consumed by the caller. This means it's checking the first byte of the length-prefixed payload, not a standalone `0x00` sentinel. For a `Typ3ByteLength` field, the byte after the field key is the length uvarint — and a `0x00` there means "length = 0" which is the nil sentinel. However, this only works because `DecodeByteSlice` hasn't been called yet. If the field key decoding consumes a different number of bytes than expected (e.g. multi-byte field number), `bz[0]` may not point to the length prefix. **Consider always using `DecodeByteSlice` → `len(fbz) == 0` for nil detection, even for primitive pointer types, to avoid this fragility.** This matches the approach used for struct-like elements and eliminates the `bz[0] == 0x00` heuristic entirely.

## Warnings (should fix)

- [ ] `tm2/pkg/amino/dumpwire.go:63` — The varint output prints `int64(v)` as the signed interpretation and `v` as unsigned, but `int64(v)` is *not* a zigzag decode — it's just a raw reinterpret cast. For sint32/sint64 fields (which use zigzag encoding), this will show misleading values. Add a note in the output or doc that the signed value is NOT zigzag-decoded, or add `(no zigzag)` annotation.

- [ ] `tm2/pkg/amino/dumpwire.go:105` — `couldBeMessage` is a heuristic that can misclassify short byte sequences (2+ bytes) that happen to parse as valid TLV. For example, the string `"P"` (`0x08 0x01`) decodes as "field 1, Typ3Varint = 1" and would be rendered as a nested message instead of raw bytes. The doc already notes this limitation, but it could cause confusion when diffing traces for string fields. Consider adding a `/* maybe-string */` annotation when a ByteLength payload could be either a message or a string.

- [ ] `tm2/pkg/amino/is_byte_length_wrapped_test.go:34` — `"slice of string"` is tested as `IsByteLengthWrapped=true`, which is correct per the current predicate (strings are `Typ3ByteLength`, so `[]string` is a non-byte slice). But the test name is slightly misleading — it's not the string itself that's wrapped, it's the slice-of-string that needs ByteLength wrapping per element. Consider renaming to `"slice of ByteLength-elem (string)"` for clarity.

- [ ] `tm2/pkg/amino/genproto2/gen_unmarshal.go:588-626` — The nil_elements handling for ertIsPointer now has three branches (writeImplicit+isStructLike, isStructLike only, neither). The `writeImplicit || isStructLike` branch correctly uses `DecodeByteSlice` + `len(fbz) == 0` for nil detection. The `!writeImplicit && !isStructLike` branch uses `bz[0] == 0x00`. But there's a subtle asymmetry: for `writeImplicit && !isStructLike` elements (nested packed lists of primitives behind a pointer with nil_elements), the code falls into the `isStructLike` branch because `writeImplicit` is checked first. This works but the control flow could be clearer — consider restructuring as two clean cases: (A) ByteLength-wrapped or implicit → use DecodeByteSlice, (B) primitive → use bz[0] == 0x00.

## Nits

- [ ] `tm2/pkg/amino/dumpwire.go:175` — The constant `cap = 32` shadows the built-in `cap()` function. Use a different name like `maxHexBytes`.

- [ ] `tm2/pkg/amino/dumpwire_test.go:41` — The `putUvarint` function duplicates the standard library's `binary.PutUvarint`. A comment explains it's "inline so this file has no dependency on internal helpers," but `encoding/binary` is a stdlib package with no internal dependency issues. Consider using the standard library instead.

- [ ] `tm2/pkg/amino/dumpwire_test.go:14-17` — The `wireBuilder.fieldKey` method computes `uint64(fnum)<<3 | uint64(typ)` which is the protobuf field key encoding. This duplicates logic from the amino package's own encoding. Not a bug, but worth noting for maintenance.

## Missing Tests

- [ ] `tm2/pkg/amino/dumpwire.go` — Codecov reports 62.5% coverage (40 lines uncovered). Missing coverage for: `Typ38Byte` truncated path (line 68-71), `Typ34Byte` truncated path (line 78-81), `couldBeMessage` returning false on `fnum == 0` (line 137), and the `default` branch for unknown Typ3 (line 115-118). Add tests for truncated 8-byte and 4-byte fields, zero field number, and unknown Typ3 values.

- [ ] `tm2/pkg/amino/genproto2/gen_unmarshal.go` — Codecov reports 69.2% coverage (20 lines missing). The new `writeByteSliceElementPayloadDecode` `case einfo.IsAminoMarshaler` branch (line 687-692) and the `writeImplicitStructPayloadDecode` array branch (line 717-724) are untested. Add test types that exercise AminoMarshaler elements inside ByteLength-wrapped lists and implicit-struct arrays.

- [ ] No test for `[]*time.Duration` with `nil_elements` — The `isStructLike` path (using `IsByteLengthWrapped` for time.Duration) combined with nil_elements pointer decode is a new combination. Add a `FuzzNilElements`-style test with a `[]*time.Duration` field tagged `amino:"nil_elements"` containing nil entries.

- [ ] No test for `[]*string` with `nil_elements` — This exercises the `bz[0] == 0x00` path in the new code. The existing `FuzzNilElements` only tests struct-like pointer types.

## Suggestions

- Consider regenerating *all* `pb3_gen.go` files (not just `tests/` and `bft/types/`) to ensure the `_inner` rename is applied consistently across the codebase. If only two files were regenerated, the others still use the old `fbz` variable name in implicit-struct decode, which works but creates an inconsistency that could confuse future reviewers.

- The `DumpWire` function is a debug aid that parses wire data without a schema. For production use (e.g., in error messages returned to callers), consider whether the `couldBeMessage` heuristic could leak information about wire structure. Currently it's only used in test failure messages, which is fine.

## Questions for Author

- The PR description says "depend on: #5569." Is #5569 already merged into this branch, or does this PR need to be rebased after #5569 lands? The first commit (`3d3524b`) appears to be from D4ryl00 with message "fix: unmarshal amino nil pointers" — is this cherry-picked from #5569?

- The PR title includes "WIP" — what remains to be done before it's ready for review? Are the `pb3_gen.go` files auto-generated and expected to change further?

- The `writeByteSliceElementPayloadDecode` method adds `case einfo.IsAminoMarshaler` and `default` branches that didn't exist in the old `writeByteSliceElementDecode`. Was the old code missing these cases (potential bug), or are they new code paths introduced by the refactor that can't actually be reached from the current callers?

## Verdict
REQUEST CHANGES — The `bz[0] == 0x00` nil-sentinel heuristic for non-ByteLength-wrapped pointer elements in nil_elements lists is fragile and should be replaced with the safer `DecodeByteSlice`-based approach used for struct-like elements. Test coverage gaps for the new unmarshal paths and DumpWire edge cases should also be addressed before merging.
