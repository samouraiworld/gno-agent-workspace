# PR #5301: feat(amino): support reserved field numbers via blank identifier fields

URL: https://github.com/gnolang/gno/pull/5301
Author: thehowl | Base: master | Files: 15 | +818 -63
Reviewed by: davd-gzl | Model: claude-opus-4-7 (1M context)

Verdict: REQUEST CHANGES — sound design and good test coverage for the middle-reserved case, but the reflection decode path ([`tm2/pkg/amino/binary_decode.go:1058-1068`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/binary_decode.go#L1058-L1068)) still rejects old wire bytes when the reserved slot is the LAST field of the struct, silently defeating the backward-compat guarantee for any future user who reserves a trailing field. Genproto2-registered types (incl. `StaticBlock`) dispatch to generated code and are unaffected; unregistered types use reflection and break.

## Summary

Adds Protobuf-3-style reserved field numbers to amino via `_ struct{} \`amino:"reserved"\``. The codec records reserved slots in `StructInfo.Reserved`, the reflection decoder skips wire bytes carrying lower-than-expected fnums (a fix to my earlier review comment that the previous draft only handled the codec layer), the genproto2 generator emits a per-typ3 skip stub for each reserved fnum, and `genproto` propagates the reservation into `.proto` output. The headline application removes `StaticBlock.Externs` (fnum 10) while preserving `Parent` at fnum 11 — wire-compatible thanks to the new skip stub generated into `gnovm/pkg/gnolang/pb3_gen.go`. Two misuse cases (`_` without tag; tag on named field) now panic at codec init.

## Glossary

- `StructInfo.Reserved` — `[]uint32` of fnums consumed by `_ \`amino:"reserved"\`` placeholder fields.
- `decodeReflectBinaryStruct` — the reflection-based binary decoder in `tm2/pkg/amino/binary_decode.go`; the slow-path used when a type is not registered with genproto2.
- `UnmarshalBinary2` — generated decoder method per type in `*/pb3_gen.go`; the fast-path used for registered types.
- `genproto2` — second-generation generator that emits typed `*Binary2` methods alongside `.proto` text.
- typ3 — wire-format type tag (Varint, 8Byte, ByteLength, 4Byte) carried with each field number in the encoded stream.

## Fix

Pre-PR: amino auto-assigned `BinFieldNum = len(infos) + 1` per exported field with no way to leave a hole; removing a field shifted every subsequent fnum, breaking the wire format. After: `_` fields tagged `amino:"reserved"` consume a fnum in `nextFieldNum` ([`codec.go:711-724`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L711-L724)) and are recorded in `sinfo.Reserved`. The reflection decoder, when wire `fnum < field.BinFieldNum`, now consumes the payload via `consumeAny(typ, bz)` and resumes the field loop ([`binary_decode.go:1004-1032`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/binary_decode.go#L1004-L1032)); the old strict-equality sanity check is removed. The genproto2 generator emits a `case <rnum>:` per-typ3 skip stub after the per-field cases ([`gen_unmarshal.go:316-345`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/genproto2/gen_unmarshal.go#L316-L345)). The load-bearing constraint: `_` fields must be processed BEFORE `isExported()` (which would skip them as unexported) — handled at [`codec.go:716`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L716).

## Critical (must fix)

- [trailing reserved fnum errors on decode] [`tm2/pkg/amino/binary_decode.go:1058-1068`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/binary_decode.go#L1058-L1068) — Reflection decoder rejects wire bytes left over after the last declared field, so reserving the LAST field of a struct still breaks backward-compat.
  <details><summary>details</summary>

  The reflection decoder's outer loop is field-by-field over `info.Fields`. `info.Fields` excludes reserved slots (they live in `info.Reserved` instead). When wire bytes carry a fnum that's higher than every declared field, the field loop exits, and the post-loop guard at line 1058 (`Reject unknown trailing fields`) returns `"unknown field number N for <Type>"`. That guard, added by #5590 along with the cross-codec parity harness, predates this PR and is correct in isolation — but for a struct whose terminal field is reserved, old encoded data carries exactly that fnum and trips it.

  Run the trailing-reserved repro alongside the middle-reserved control (`reviews/pr/5xxx/5301-amino-reserved-fields/1-8ef338b94/tests/trailing_reserved_test.go`): middle passes, trailing fails with `unknown field number 2 for amino_test.NewStruct`. The generated path (`UnmarshalBinary2`) is unaffected because it dispatches via a single switch keyed on fnum, where trailing reserved fnums hit the emitted `case rnum:` stub. So `StaticBlock` itself (where the reserved is between `Consts`/10 and `Parent`/11, and registered with genproto2) is safe — but the feature is advertised as a general migration tool, and any future user who reserves a trailing field on a type that uses the reflection codec will see silent decode failures on old data.

  **Repro:**

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5301 -R gnolang/gno
  cat > tm2/pkg/amino/trailing_reserved_repro_test.go <<'EOF'
  package amino_test

  import (
      "testing"
      "github.com/gnolang/gno/tm2/pkg/amino"
      "github.com/stretchr/testify/assert"
  )

  func TestReservedFieldTrailing_Repro(t *testing.T) {
      type Old struct{ A, B string }
      type New struct {
          A string
          _ [0]struct{} `amino:"reserved"`
      }
      cdc := amino.NewCodec()
      bz, _ := cdc.Marshal(Old{A: "hello", B: "removed"})
      var got New
      err := cdc.Unmarshal(bz, &got)
      assert.NoError(t, err)
      assert.Equal(t, "hello", got.A)
  }
  EOF
  go test -v -run TestReservedFieldTrailing_Repro ./tm2/pkg/amino/
  rm tm2/pkg/amino/trailing_reserved_repro_test.go
  ```

  Observed: `unknown field number 2 for amino_test.New`.

  Fix: in `decodeReflectBinaryStruct`'s post-loop guard ([`binary_decode.go:1058-1068`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/binary_decode.go#L1058-L1068)), instead of erroring on leftover bytes, drain them via `consumeAny(typ, bz)` while validating `fnum > lastFieldNum` and `fnum` is registered in `info.Reserved` (or, less strict but consistent with proto3, simply skip unknown trailing fnums). Add the trailing-reserved case to `TestReservedFieldDecodeBackwardCompat` so the regression is pinned.
  </details>

## Warnings (should fix)

- [misuse panics swallowed by outer recover] [`tm2/pkg/amino/codec.go:700-705`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L700-L705) — Both new init-time misuse panics are masked by the pre-existing `recover()` in `parseStructInfoWLocked`, replaced with a generic `"panic parsing struct <T>"`.
  <details><summary>details</summary>

  The PR's new panics — `blank identifier field at index N must have amino:"reserved" tag` and `amino:"reserved" tag is only valid on blank identifier (_) fields, not "X"` — are useful, specific messages. But the existing defer/recover at [`codec.go:700-705`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L700-L705) catches them and re-panics with `fmt.Sprintf("panic parsing struct %v", rt)`, dropping the inner message entirely. The user sees `panic parsing struct amino_test.S` with no hint about which field is wrong or what tag is required. The PR's own `TestReservedFieldMisuse` doesn't catch this because `assert.Panics` only checks that *some* panic happens, not the message.

  Verified via `go test -v -run TestReservedFieldExtraTag ./tm2/pkg/amino/` with a struct using `\`amino:"reserved,extra"\`` on a `_` field — output is `panic parsing struct amino_test.S` with no detail.

  Fix: change the recover handler to wrap with both the type and original panic, e.g. `panic(fmt.Sprintf("panic parsing struct %v: %v", rt, ex))`. Pre-existing behavior is unhelpful for all panic causes; the PR makes the cost concrete because it relies on the messages being readable.
  </details>

- [strict equality on amino tag breaks tag composition] [`tm2/pkg/amino/codec.go:718`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L718) — `field.Tag.Get("amino") != "reserved"` rejects any combined tag like `amino:"reserved,extra"`, panicking with the misleading "must have amino:reserved tag" message even though the tag IS present.
  <details><summary>details</summary>

  Pre-PR amino conventions allow multi-value amino tags split on `,` (see [`parseFieldOptions` at codec.go:823](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L823-L833) where `aminoTag` is split on `,` and each part matched against `"reserved"`, `"unsafe"`, etc.). The reserved-detection path takes the unparsed tag verbatim. There's no real use case today for combining `reserved` with another flag, but the asymmetry will surprise readers, and the panic message ("must have amino:\"reserved\" tag") is wrong — it does have the tag.

  Fix: use the same split-and-contains pattern as `parseFieldOptions` to detect `"reserved"` anywhere in the tag list, OR explicitly forbid composition with a clearer error.
  </details>

## Nits

- [`tm2/pkg/amino/codec.go:826`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L826) — Panic message says `not %q` referring to the field name; reads slightly awkwardly. Suggest: `... only valid on blank identifier (_) fields; got named field %q`.
- [`gnovm/pkg/gnolang/nodes.go:1612`](../../../../../.worktrees/gno-review-5301/gnovm/pkg/gnolang/nodes.go#L1612) — Production use writes `struct{}`; ADR + tests + fixtures write `[0]struct{}`. Both are zero-size and functionally identical. Pick one for consistency (ADR uses `[0]struct{}`; aligning the production site avoids the cross-file drift).
- [`tm2/adr/pr5301_amino_reserved_field_numbers.md:48-51`](../../../../../.worktrees/gno-review-5301/tm2/adr/pr5301_amino_reserved_field_numbers.md#L48-L51) — The ADR says misuse "fails loudly at codec initialisation rather than silently producing wrong field numbers." With the outer recover swallowing messages (see Warning above), the loud-failure claim is half-true in practice.
- [`tm2/pkg/amino/genproto2/gen_unmarshal.go:316-345`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/genproto2/gen_unmarshal.go#L316-L345) — The reserved-emission block is ~30 lines of `sb.WriteString` calls. A single `text/template` block keyed on `rnum` would be far more readable; current shape is hard to diff if the typ3 set changes.

## Missing Tests

- [trailing-reserved-field decode through reflection] [`tm2/pkg/amino/codec_test.go:347`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec_test.go#L347) — `TestReservedFieldDecodeBackwardCompat` only exercises middle-position reservation. Add the trailing case to pin the Critical bug.
  <details><summary>details</summary>

  See the repro under Critical. Once the decoder is fixed, this test belongs alongside the existing one in `codec_test.go`. The genproto2 path already has comparable coverage via `TestPos_ReservedDeclared_DecodesOldBytes`, but reflection-path coverage is the gap.
  </details>

- [leading-reserved-field decode] [`tm2/pkg/amino/codec_test.go:347`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec_test.go#L347) — No test for `_ struct{} \`amino:"reserved"\`` as the FIRST field of a struct.
  <details><summary>details</summary>

  Skim of `decodeReflectBinaryStruct` suggests it works (the first field's BinFieldNum starts at 2, the inner skip loop handles `fnum=1 < 2`), but no test asserts it. Add a `TestReservedFieldLeading` to round out the (leading, middle, trailing) matrix. Trivial to add; same shape as the existing middle test.
  </details>

- [reserved field with multi-valued amino tag] [`tm2/pkg/amino/codec.go:718`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L718) — No test pins the strict-equality behavior either way. Add one for whichever shape (accept or reject) the Warning resolves to.

## Suggestions

- [`tm2/pkg/amino/codec.go:782`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/codec.go#L782) — `sinfo = StructInfo{Fields: infos, Reserved: sinfo.Reserved}` rebuilds `sinfo` to swap the named-field bag. Cleaner: initialize `sinfo.Fields = infos` directly at the end and skip the literal. Minor.
- [`tm2/pkg/amino/genproto2/gen_unmarshal_reserved_test.go:1-321`](../../../../../.worktrees/gno-review-5301/tm2/pkg/amino/genproto2/gen_unmarshal_reserved_test.go#L1-L321) — The buggy-mimic test infrastructure is impressively thorough but the case-2 snapshot will need a maintenance touch if the generator's typ3 list expands. Consider an `// UPDATE: re-extract via TestExtractCase2 helper.` pointer next to the snapshot so the next person to add a typ3 finds the snapshot location.

## Questions for Author

- Did you intentionally restrict reserved fields to be "between" declared fields, or is the trailing-reserved case considered a supported migration? The ADR doesn't call out the limitation.
- Does the codebase have an existing pattern for "reserved range" (e.g. `amino:"reserved 10-15"` for bulk holes)? Not asking you to add it here — just curious if the design intentionally avoided it.
