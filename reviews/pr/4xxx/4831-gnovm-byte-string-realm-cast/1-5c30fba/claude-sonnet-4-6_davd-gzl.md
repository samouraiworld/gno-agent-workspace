# PR #4831: fix(gnovm): allow []byte -> string cast on realm owned fields

**URL:** https://github.com/gnolang/gno/pull/4831
**Author:** Villaquiranm | **Base:** master | **Files:** 2 | **+81 -1**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR relaxes the readonly/external-realm conversion guard in `doOpConvert`
(`gnovm/pkg/gnolang/op_expressions.go`) to allow two previously-blocked conversions
when the source value is owned by an external realm:

- `[]byte` (or `[]uint8`) → `string`
- `[]int32` (rune slice) → `string`

Before this fix, any conversion of an externally-realm-owned, non-immutable value
panicked with `"illegal conversion of readonly or externally stored value"`. The guard
exists to prevent an attacker from obtaining a mutable alias into another realm's state.

The fix is correct for the two permitted conversions because Go's `string()` conversion
always copies the underlying byte/rune data — it never borrows a reference into the
original slice. This is confirmed in `ConvertTo` (`values_conversions.go` lines
1051–1081): a new `StringValue` is allocated via `alloc.NewString(string(data))` in every
code path, including the data-backed array path. The resulting string is immutable
(`PrimitiveType.IsImmutable() == true`), so no mutation path exists back to the original
realm state.

The security exploit test (`gnovm/tests/files/exploit/realm_exploiter.gno`) still
panics correctly — the fix is narrowly scoped to slice-to-string conversions.

One commit (`add several string types`) extends the original `Uint8Kind`-only check to
also cover `Int32Kind`, enabling `string([]int32)` from cross-realm values. The txtar
integration test exercises:
- `[]byte(msg)` where `msg` is already `[]byte` — **optimized away by the preprocessor**
  (`preprocess.go:1513` skips conversions when source and target TypeIDs are equal)
- `[]byte(msg8)` same as above — optimized away
- `[]byte(msgS)` where `msgS` is `type S string` — handled by
  `PrimitiveType.IsImmutable() == true`, never reaches the guard
- `string(msg32)` where `msg32` is `[]int32` — **the only path that exercises the new code**

The PR does not include an ADR, and the test does not verify computed output values.

## Test Results

- **Existing tests:** PASS — `TestFiles/zrealm_crossrealm*`, `TestFiles/exploit/realm_exploiter.gno`, `TestFiles/convert*` all pass.
- **Integration test:** PASS — `TestTestdata/interrealm_read_only` passes.
- **Edge-case tests:** skipped

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/op_expressions.go:709` — The type assertion `xv.T.(*SliceType)` does not call `baseOf()`, so named slice types (e.g. `type MyBytes []byte`, `type Runes []int32`) from external realms will still panic on `string(myBytesValue)`. This is inconsistent with Go semantics and leaves a class of valid conversions blocked. The rest of the file uses `baseOf()` for type assertions (e.g. lines 15, 43, 146, 358). The fix should be `baseOf(xv.T).(*SliceType)`.

- [ ] `gno.land/pkg/integration/testdata/interrealm_read_only.txtar:6` — The test only asserts `stdout 'OK!'` (transaction broadcast success). It does not verify that the conversions produce correct output. The `Exec` function calls `println("Equals")` twice and `println(string(msg32))`, but none of these are checked. A test that passes for the wrong reason (e.g. the relevant path was never exercised) provides false confidence.

- [ ] `gno.land/pkg/integration/testdata/interrealm_read_only.txtar` — No VM-level filetest (`gnovm/tests/files/`) is added. The existing pattern for security-adjacent behavior changes is to add a filetest showing the allowed case and, where appropriate, a sibling file showing the still-disallowed cases. This makes the intent explicit and is caught by the fast `go test ./gnovm/pkg/gnolang/ -run TestFiles/` suite without requiring a full blockchain node.

## Nits

- [ ] `gnovm/pkg/gnolang/op_expressions.go:712` — Comment says `"Allow conversion from []byte to string"` but the code also allows `[]int32 → string`. Update to `"Allow read-only conversion of []byte or []int32 to string"`.

- [ ] `gnovm/pkg/gnolang/op_expressions.go:709-716` — The empty `if` body with a comment, followed by an `else { panic }`, is an unusual Go pattern. Inverting the condition is cleaner:
  ```go
  if !isBytesArray || t.Kind() != StringKind {
      panic("illegal conversion of readonly or externally stored value")
  }
  ```

- [ ] `gno.land/pkg/integration/testdata/interrealm_read_only.txtar:28,31,35` — Mixed indentation: `Message()` uses a tab, `Message8()`, `Message32()`, `MessageS()` use four spaces. Gno convention is tabs.

- [ ] `gno.land/pkg/integration/testdata/interrealm_read_only.txtar:73` — File does not end with a newline (confirmed: last byte is `}`). POSIX and most Go tooling expect a trailing newline.

- [ ] `gno.land/pkg/integration/testdata/interrealm_read_only.txtar:17-18` — Double spaces in `var msg  =` and `var msg8  =` declarations. Single space is conventional.

- [ ] `gno.land/pkg/integration/testdata/interrealm_read_only.txtar:58-59` — Two lines that contain only a tab character (trailing whitespace on otherwise empty lines in `Exec`).

## Missing Tests

- [ ] No VM filetest covering `string(external_realm_[]byte)` — the stated purpose of the PR. `string(msg32)` ([]int32) is the only path exercised; the Uint8Kind branch is never triggered in the test suite.
- [ ] No negative test asserting that other conversions on readonly slices (e.g. `append(externalSlice, x)` via type conversion, or `[]SomeType(externalSlice)`) still panic. The exploit test covers struct→struct but not slice→other-slice.
- [ ] No test for `type MyBytes []byte` in an external realm followed by `string(myBytesValue)` — this case is broken after the fix (would still panic) but is not documented or tested.

## Suggestions

- Use `baseOf(xv.T).(*SliceType)` instead of `xv.T.(*SliceType)` to correctly handle named slice types. This aligns with Go's conversion rules (`string(myBytesValue)` is legal in Go if the underlying type is `[]byte`) and with how the rest of `op_expressions.go` handles type assertions.
- Add a `gnovm/adr/pr4831_allow_byte_string_cast.md` ADR. The change modifies a security-sensitive runtime guard; an ADR documents the reasoning (copy semantics, immutability of string result) so future contributors can verify and build on it. The AGENTS.md guidance lists this as expected for non-trivial VM changes.
- Consider adding a `gnovm/tests/files/zrealm_crossrealm35.gno` filetest (allowed case) and `zrealm_crossrealm35b.gno` or similar (still-disallowed case) to make the behavior unambiguous in the fast test suite.
- The txtar test's `stdout 'OK!'` assertion should be supplemented with output checks, e.g.:
  ```
  stdout 'hello bug'
  ```
  for the `println(string(msg32))` line, so the test fails if the conversion silently produces wrong output.

## Questions for Author

- Was the original bug report specifically about `[]byte → string`, `[]int32 → string`, or both? The commit history shows `Int32Kind` was added in a separate commit (`add several string types`), but there's no test that directly triggers the `Uint8Kind` path. Is there a repro case for `string(external_[]byte)` that was previously failing?
- Have you considered using `baseOf(xv.T)` to handle named slice types (e.g. `type IP []byte` in `net.IP`)? The current fix doesn't help users who have defined a named byte-slice type in their realm.
- Why is `[]byte(msgS)` included in the txtar test? `msgS` is of type `S` (derived from `string`), which is immutable — it bypasses the readonly check entirely and was never affected by the bug. Including it creates noise without adding coverage.

## Verdict

REQUEST CHANGES — The core logic is correct and the security rationale is sound, but the incomplete handling of named slice types (`baseOf` omission) leaves the fix narrower than stated, the integration test doesn't verify computed output, and no VM-level filetest covers the primary use case.
