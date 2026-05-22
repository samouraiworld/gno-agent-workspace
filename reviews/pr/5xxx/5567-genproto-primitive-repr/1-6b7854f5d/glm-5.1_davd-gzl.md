# PR #5567: WIP Fix/maxwell/genproto primitive repr list

**URL:** https://github.com/gnolang/gno/pull/5567
**Author:** ltzmaxwell | **Base:** master | **Files:** 3 | **+53 -35**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR fixes a mismatch between the protobuf schema generator (`typeToP3Type` in `genproto.go`) and the Go↔pb bindings generator (`p3goTypeExprString` in `bindings.go`) when a slice/array element type implements `AminoMarshaler` with a **primitive** repr (string, int64, uint8, []byte, etc. — anything that isn't struct or interface).

**Root cause:** `typeToP3Type` (schema side) dereferences through `info.ReprType.Type` and emits the bare primitive wire type (e.g. `string`, `sint64`). But `p3goTypeExprString` (bindings side), when called on the registered element type, hits the `pkg != nil && GoPkgPath != ""` branch and returns `*pkgPb.TypeName` — a wrapper message. The generated `make([]pboete_, N)` then uses the wrapper type while the .proto schema declares a `repeated string` (or similar). This mismatch causes incorrect serialization of types like `[]crypto.Address` (where `crypto.Address = [20]byte` has `MarshalAmino() (string, error)`).

**Three changes:**

1. **`tm2/pkg/amino/genproto/bindings.go:528-531`** — New guard: when the slice element is an `AminoMarshaler` with a non-struct, non-interface repr, recompute `pboete_` by calling `p3goTypeExprString` on `gooreType.ReprType` instead of `gooreType`, forcing it to emit the primitive type that matches the schema.

2. **`misc/genproto/genproto.go`** — Cleans up the old genproto entry point: removes 16 production package imports (bitarray, merkle, abci, btypes, etc.) that are now handled by `misc/genproto2`. Only `tests.Package` remains, needed for the fuzz oracle's `pbbindings.go`. Updated `LongHelp` to document this.

3. **`.github/workflows/ci-codegen-verify.yml`** — Adds a new `genproto2` CI job (runs `make -C misc/genproto2` + diff check + `go build ./...`), adds `misc/genproto2/**` to the trigger paths, and adds `go build ./...` to the existing `genproto` job.

**Design note:** The fix only patches `go2pbStmts` (Go→pb direction). The reverse `pb2goStmts` path doesn't need it because the `IsAminoMarshaler` branch at line 694 already normalizes `goorType = gooType.ReprType` before entering the slice/array switch, so the recursive `subStmts` calls already operate on the correct repr type.

## Test Results

- **Existing tests:** PASS — `TestAminoMarshaler*`, `TestCrossPkg*`, `TestRoundtripBinary2_*` all pass.
- **Edge-case tests:** skipped
- **CI:** `genproto` job cancelled (6h timeout, likely infra issue, not a code failure). `genproto2` passed. One `check` failure reported but unrelated to these files.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `misc/genproto/genproto.go:28-31` — The comment mentions "_testCodec in reflect_test.go type- asserts amino.PBMessager on fuzzed values" but the correct interface name is `PBMessager` (single 'e') throughout the codebase. Verify this isn't a typo for `PBMessager` vs `PBMessager` — actually on inspection, the codebase uses `PBMessager` consistently so this is fine, but the comment is a bit misleading since it references internal test details that may change.

- [ ] `tm2/pkg/amino/genproto/bindings.go:528-530` — The condition checks `gooreType.ReprType.Type.Kind()` but does not guard against `gooreType.ReprType` being nil. If `IsAminoMarshaler` is true, `ReprType` should always be populated by the codec, but a nil dereference here would panic without a clear message. Consider adding a defensive check or at minimum a comment noting the invariant.

## Nits

- [ ] `.github/workflows/ci-codegen-verify.yml:66-69` — The new `genproto2` job doesn't install `libprotobuf-dev` or `protobuf-compiler` or run `make -C misc/devdeps install`, unlike the `genproto` job. This is correct since `genproto2` doesn't need protoc (it generates Go directly), but a comment explaining why would help future readers.

- [ ] `misc/genproto/genproto.go:16-18` — The multi-line `LongHelp` string concatenation could use a raw string for readability.

- [ ] Commit history is messy (multiple "fixup" and "cleanup" commits). Should be squashed before merge.

## Missing Tests

- [ ] No test exercises the `bindings.go` fix specifically through the old genproto path (`misc/genproto`). The existing tests (`aminomarshaler_list_test.go`) exercise the `genproto2` path. Since `misc/genproto` now only generates for `tests.Package`, the `pbbindings.go` output (which uses the old `bindings.go` code) should be regenerated and verified to compile and pass the 3-way fuzz oracle. The CI `genproto` job covers this, but a local regeneration + test run would be more convincing.
- [ ] No test for `[]HostRepr` (AminoMarshaler with `[]byte` repr inside a slice) through the old pbbindings path. This is a case where the repr is `[]byte` (Kind=Slice, not Struct or Interface), so it should be caught by the new condition.
- [ ] No test for `[]AminoMarshalerInt5` (int→string repr inside a slice) through the old pbbindings path.

## Suggestions

- The comment block at `bindings.go:504-527` is excellent and thorough. Consider extracting the rationale into an ADR in `tm2/adr/` since this is a non-trivial fix touching code-generation correctness. The PR description is empty, and an ADR would help future contributors understand the schema/bindings alignment invariant.

## Questions for Author

- Why is the PR still marked WIP? The fix looks complete and tests pass. Is there remaining work?
- The `genproto` CI job timed out (6h). Was this observed on prior PRs too, or is it specific to this branch? The `genproto` job in master CI typically completes in ~1 minute.
- Does the `pbbindings.go` file in `tm2/pkg/amino/tests/` need to be regenerated as part of this PR (since the code-gen logic changed)? The CI `genproto` job's diff-check would catch this, but it timed out.

## Verdict
REQUEST CHANGES — The core fix is correct and well-documented, but the PR needs: (1) squashing of fixup commits, (2) confirmation that the `genproto` CI job passes (it timed out), (3) verification that the regenerated `pbbindings.go` in `tests/` matches the new logic, and (4) ideally an ADR documenting the schema/bindings alignment invariant for the genproto codebase. Remove the WIP label once ready.
