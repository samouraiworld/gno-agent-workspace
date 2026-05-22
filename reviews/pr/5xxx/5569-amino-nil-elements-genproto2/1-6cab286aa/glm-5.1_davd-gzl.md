# PR #5569: fix(tm2/amino): preserve nil_elements in genproto2

**URL:** https://github.com/gnolang/gno/pull/5569
**Author:** D4ryl00 | **Base:** master | **Files:** 5 | **+432 -291**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

Fixes a genproto2 unmarshal regression from #5282. When `amino:"nil_elements"` is set on a `[]*Struct` field, the generated unmarshal decoder did not preserve positional `nil` entries. An empty `ByteLength` entry (`0x00`) on the wire was decoded as a non-nil zero-value struct instead of `nil`. This caused a consensus failure: `types.Commit.Precommits` uses `nil_elements`, and losing nil positions in that list produces a `wrong LastCommit` error on the next block, wedging the chain.

The fix adds `nil_elements`-aware unmarshal logic in `gen_unmarshal.go` that matches the reflect codec's behavior: empty entries decode to `nil`, non-empty entries stay on the generated fast path. It also refactors `writeByteSliceElementDecode` and `writeImplicitStructDecode` into payload-level helpers (`writeByteSliceElementPayloadDecode`, `writeImplicitStructPayloadDecode`) so the nil-detection check can reuse them with a pre-read byte slice.

The generated `pb3_gen.go` files are regenerated accordingly, and a new test type `FuzzNilElements` plus a consensus-level `TestBlockBinary2RoundTripPreservesNilLastCommitEntries` test validate the fix end-to-end.

## Test Results

- **CI:** All checks pass (build, lint, test, e2e, genproto, analyze, CodeQL).
- **Codecov:** Patch coverage 88.4% — 42 lines missing coverage, mostly in the regenerated `pb3_gen.go` dead-code paths and the `gen_unmarshal.go` code-generator branches. The critical runtime paths (nil_elements struct-like branch, block round-trip) are covered.
- **Existing review:** ajnavarro flagged a misleading comment in the first commit; the author addressed it in the second commit by rewriting the comment to be more precise (line 619). jefft0 approved.

## Critical (must fix)

None.

## Warnings (should fix)

1. **Variable shadowing risk in nil_elements struct-like path.** `gen_unmarshal.go:594-607` — The nil_elements `isStructLike` branch reads `fbz` from `DecodeByteSlice(bz)`, then passes `"fbz"` as `srcVar` to `writeImplicitStructPayloadDecode` / `writeByteSliceElementPayloadDecode`. Inside `writeImplicitStructPayloadDecode` (line 716), a new `_inner` variable is introduced specifically to avoid colliding with a caller's `fbz`. This works, but it's fragile: if any future caller passes a `srcVar` that equals `_inner`, the same collision reappears. Consider documenting the reserved variable names or using a more systematically unique naming scheme (e.g. `_payload0`, `_payload1`).

2. **Non-struct-like nil_elements path reads raw `bz[0]` without consuming the ByteLength wrapper first.** `gen_unmarshal.go:609-616` — For `NilElements && !writeImplicit && !isStructLike` (e.g. `[]*string` with nil_elements), the code checks `bz[0] == 0x00` and skips one byte. This matches the reflect codec (`binary_decode.go:643-647`) which also does `slide(&bz, &n, 1)` on `0x00`. However, the `else` branch calls `writeByteSliceElementDecode` which, for non-struct-like elements, calls `writePrimitiveDecodeFrom(sb, accessor, einfo, fopts, indent, "bz")` — this decodes from the remaining `bz` starting at the field's ByteLength length-prefix, which is correct. But the two branches read the wire format at different abstraction levels (one peeks at the raw byte, the other uses DecodeByteSlice), making this asymmetric and harder to reason about. This matches the reflect codec's approach, so it's not wrong, but a comment explaining the asymmetry would help future readers.

3. **`FuzzNilElements` test struct only covers struct-pointer slices.** `fuzz_types.go:283-288` — `FuzzNilElements` has `[]*FuzzFieldInfo` and `[]*GnoVMPos`, both struct-pointer slices with nil_elements. It does not exercise the non-struct-like nil_elements path (e.g. `[]*string`, `[]*int32` with nil_elements). While the reflect codec's fuzz tests cover these indirectly, the generated-code-specific test `TestRoundtripBinary2_FuzzNilElements` only validates the struct-pointer path, leaving the non-struct-like nil_elements decode untested at the genproto2 level.

## Nits

1. **`fbz` variable name reuse across generated code.** The rename from `fbz` to `_inner` in the implicit-struct payload decode is good, but the outer nil_elements struct-like branch (line 594) still uses `fbz` for the pre-read byte slice, and then passes it to `writeByteSliceElementPayloadDecode` / `writeImplicitStructPayloadDecode` which may internally also use `fbz` for a different purpose. This currently works due to the `_inner` rename, but the naming inconsistency makes the generated code harder to audit.

2. **Second commit message "chore: improve comment" is vague.** The second commit only changes the comment at `gen_unmarshal.go:619` (per ajnavarro's review). The message could note it addresses a review comment about comment accuracy.

## Missing Tests

1. **Non-struct-like nil_elements roundtrip test.** No genproto2-specific test exercises `[]*string` or `[]*int32` with `amino:"nil_elements"` — only the struct-pointer path is covered by `FuzzNilElements`. A dedicated test type with `Entries []*string \`amino:"nil_elements"\`` would close this gap.

2. **Array-with-nil-elements test.** The fix handles the `isArray` code path in `storeElem` but no test exercises a fixed-size array of pointer elements with nil_elements.

3. **AminoMarshaler pointer element with nil_elements.** The `writeByteSliceElementPayloadDecode` has an AminoMarshaler case (line 694-699) that would be triggered for pointer elements whose value type implements `AminoMarshaler` with nil_elements set. No test exercises this combination.

## Suggestions

1. **Add a comment above the nil_elements struct-like branch** (line 593-607) explaining *why* we read the full ByteSlice first and check `len(fbz) == 0` for nil, vs. the non-struct-like branch that peeks at `bz[0]`. Something like:
   ```
   // Struct-like elements are always wrapped in a ByteLength field,
   // so we must consume the length prefix first. An empty payload
   // (zero-length ByteSlice) signals nil. This differs from the
   // non-struct-like path which can peek at the raw byte.
   ```

2. **Consider renaming `srcVar` parameter** in `writeByteSliceElementPayloadDecode` / `writeImplicitStructPayloadDecode` to something that signals it's an arbitrary caller-provided variable name, to make the shadowing concern more visible.

3. **The `reflect_test.go:105` `lossyDecode` flag for `FuzzNilElements`** can now be removed or flipped to `false` since genproto2 correctly preserves nil elements. This would tighten the test to demand strict equality.

## Questions for Author

1. Does the `lossyDecode` workaround in `reflect_test.go:105` still need to exist for `FuzzNilElements`, or should it be removed now that the genproto2 codec preserves nil elements correctly?

2. Was the non-struct-like nil_elements path (line 608-616) tested manually or with any fuzz runs? The reflect codec has a slightly different structure for this path and I want to confirm the generated code produces byte-identical output.

3. The `anyDepth` parameter is correctly passed through in `writeByteSliceElementPayloadDecode` (line 692: `UnmarshalBinary2(cdc, fbz, anyDepth)`), but in the nil_elements non-struct-like path (line 614), `writeByteSliceElementDecode` is used which also passes `anyDepth` indirectly. Can you confirm that `anyDepth` is preserved correctly through both nil_elements code paths?

## Verdict

**Approve with minor suggestions.** The fix correctly addresses the consensus-critical regression: nil entries in `Commit.Precommits` are now preserved through genproto2 round-trip, matching the reflect codec's behavior. The refactoring into payload-level helpers is clean and the generated code is correct. The main gap is test coverage for the non-struct-like nil_elements path and the AminoMarshaler+nil_elements combination. The second commit adequately addresses ajnavarro's review feedback. No critical issues found.
