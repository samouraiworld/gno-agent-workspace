# PR #4831: fix(gnovm): allow []byte -> string cast on realm owned fields

URL: https://github.com/gnolang/gno/pull/4831
Author: Villaquiranm | Base: master | Files: 3 | +119 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE with caveats** — fix is correct and the security-critical copy semantics hold; remaining issues are test-coverage gaps (txtar asserts only `OK!`, never the converted values), an out-of-date interrealm doc that still prints the old guard, and one style nit. Round 2 since [round 1 (`5c30fba`)](../1-5c30fba/claude-sonnet-4-6_davd-gzl.md); the `baseOf(...)` wrap from notJoon's comment landed in `c45a2ed`, the new commit `e10f5c9` added the declared-type txtar.

## Summary

The PR relaxes the cross-realm conversion guard in [doOpConvert](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/op_expressions.go#L770-L779) so a realm can stringify a `[]byte`, `[]uint8`, `[]int32`, or any declared type whose base is one of those slice kinds, when the source value is owned by an external realm. Before this change, every cross-realm `string(externalSlice)` panicked with `illegal conversion of readonly or externally stored value` — the issue #4825 repro. The fix is sound because `ConvertTo` ([values_conversions.go:1039-1084](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/values_conversions.go#L1039-L1084)) always allocates a fresh string via `alloc.NewString(string(...))` — every branch copies the bytes, no alias survives to the source slice — and `PrimitiveType.IsImmutable() == true` means the result has no write path back. Two new txtar integration tests cover the common kinds and the declared-type case.

## Glossary

- **doOpConvert** — VM opcode handler for type conversions; enforces cross-realm safety in two "cases".
- **IsReadonly(xv)** — true when `xv` is reached through a foreign realm's storage; trips the Case 1 guard.
- **baseOf(t)** — unwraps a `*DeclaredType` to its underlying type; necessary so `type Bytes []byte` is recognized as a slice.
- **N_Readonly** — TypedValue flag set on values traversed from another realm's persistent state.

## Fix

Before: any conversion of a non-immutable, externally-readonly value panicked unconditionally — the only exception was when the source type was the converting realm's own declared type. After: an additional carve-out lets the source be a slice whose element kind is `Uint8Kind` or `Int32Kind` and the target be `StringKind`. The load-bearing constraint is that the resulting string is always a fresh, immutable allocation — no pointer or back-reference into the source slice is retained — and the target type (`string`) is primitive-immutable, so no later write path exists. The `baseOf(xv.T).(*SliceType)` (introduced in `c45a2ed` after notJoon's review) unwraps named slice types so `type Bytes []byte` is recognized; without it the original commit would have left declared slice types broken (notJoon's repro on `interrealm_read_only_declared_type.txtar`).

## Critical (must fix)

None — the relaxation is narrowly scoped and the copy invariant holds across every branch of `ConvertTo`.

## Warnings (should fix)

- **[test only asserts transaction success, not values]** [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:6`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L6) — the only assertion is `stdout 'OK!'`. The `Exec` body has three `println("Equals")` calls and one `println(string(msg32))`, none of which are checked. A regression where the conversion silently produced an empty or wrong string would still pass.
  <details><summary>details</summary>

  The `OK!` line confirms the tx didn't panic — that's already a strong signal because the bug shape is a panic. But it leaves a wide hole: if the conversion path were changed in the future to skip the copy (or to produce a typed-zero string for foreign sources), the test would still pass. Fix: add explicit `stdout 'Equals'` assertions (the txtar runner supports repeated `stdout` matches) plus `stdout 'hello bug'` for the rune-slice case. Cost is two extra lines, payoff is the test actually proves the conversion produced the expected bytes.
  </details>

- **[interrealm doc no longer matches the code]** [`gno/docs/resources/gno-interrealm-v2.md:505-513`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/docs/resources/gno-interrealm-v2.md#L505-L513) — the "§9.1 Case 1" code block reproduces the pre-PR `doOpConvert` guard verbatim, including the `panic("illegal conversion of readonly or externally stored value")` as the unconditional `else`. After this PR there is a second carve-out for `[]byte`/`[]int32 → string` that the doc doesn't mention.
  <details><summary>details</summary>

  This doc is the canonical reference for cross-realm safety reasoning (§9 is titled "Conversion Guards"). Leaving it stale means a future reviewer reading it as ground truth will believe `string(externalSlice)` panics, then be surprised by the PR's behaviour. Fix: update the §9.1 snippet to include the new `baseOf(xv.T).(*SliceType)` branch with a one-paragraph justification: copy semantics in `ConvertTo` + `PrimitiveType.IsImmutable() == true` on the target. The skill guidance flags docs touched by the change — `doOpConvert` is directly documented here.
  </details>

- **[no VM filetest in `gnovm/tests/files/`]** `gnovm/tests/files/zrealm_p_convert_readonly_ok_filetest.gno` (sibling) — the existing security invariant has a fast VM filetest; the new relaxation does not. Integration txtar tests run a full chain and are slow; the VM filetest suite is the natural home for "this conversion is/isn't allowed across realms".
  <details><summary>details</summary>

  [`zrealm_p_convert_readonly_ok_filetest.gno`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/tests/files/zrealm_p_convert_readonly_ok_filetest.gno) demonstrates the existing struct-conversion-still-panics invariant in 25 lines. A parallel `zrealm_p_convert_bytes_to_string_filetest.gno` showing `string(externalBytes)` now succeeds (and printing the result) plus a `..._slice_to_struct_filetest.gno` showing that non-string conversions on the same foreign slice still panic, would document the boundary precisely. Fix: add two filetests in `gnovm/tests/files/` covering the allowed and still-disallowed cases. These run under `go test ./gnovm/pkg/gnolang/ -run TestFiles/` in seconds.
  </details>

## Nits

- [`gnovm/pkg/gnolang/op_expressions.go:773-778`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/op_expressions.go#L773-L778) — empty `if` body with an explanatory comment, followed by `else { panic }`, is unusual Go. Inverting reads better:
  ```go
  if !isBytesArray || t.Kind() != StringKind {
      panic("illegal conversion of readonly or externally stored value")
  }
  // []byte/[]int32 → string: copy semantics, immutable result, safe.
  ```

- [`gnovm/pkg/gnolang/op_expressions.go:774-775`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/op_expressions.go#L774-L775) — comment says `Allow conversion from []byte to string` but the branch also accepts `[]int32`. Either widen to `Allow conversion of []byte or []rune (i.e. []int32) to string` or rename `isBytesArray` to `isStringConvertibleSlice`.

- [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:27-35`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L27-L35) — mixed indentation: `Message()` uses a tab, `Message8`/`Message32`/`MessageS` use four spaces. Gno convention is tabs.

- [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:17-20`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L17-L20) — double spaces in `var msg  =`, `var msg8  =`, `var msgS  =`, `var msg32  =`. Single space.

- [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:73`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L73) and [`gno.land/pkg/integration/testdata/interrealm_read_only_declared_type.txtar:38`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only_declared_type.txtar#L38) — no trailing newline.

- [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:58-59`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L58-L59) — trailing-whitespace-only lines (just a tab).

- [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:68-70`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L68-L70) — the `bytes.Equal([]byte(msgS), msgLocal)` block: `msgS` is `type S string`, so `S.IsImmutable() == true` via `DeclaredType.IsImmutable() -> PrimitiveType.IsImmutable()`. The Case 1 guard at [`op_expressions.go:764`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/op_expressions.go#L764) short-circuits on `!xv.T.IsImmutable()` — this conversion was never blocked by the bug, with or without the PR. Including it in the test is noise. If the intent is "string-like declared types also work", a comment would be clearer; otherwise drop it.

## Missing Tests

- **[Uint8Kind path]** [`gno.land/pkg/integration/testdata/interrealm_read_only.txtar:60,64`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only.txtar#L60-L64) — these read `[]byte(msg)` and `[]byte(msg8)` where source and target both have `[]byte` kind. The preprocessor likely elides these as no-op conversions ([preprocess.go:1513](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/preprocess.go#L1513) skips when source and target TypeIDs match), so they never reach `doOpConvert`. The only line that actually exercises the new `Uint8Kind+StringKind` branch is... none in this test. `string(msg32)` exercises `Int32Kind`. Add an explicit `string(msg)` to cover the `[]byte → string` path that is the headline of the PR.
  <details><summary>details</summary>

  This is the bug from the linked issue (#4825), and yet the integration test never converts `[]byte → string` — only `[]byte → []byte` (no-op) and `[]int32 → string`. The `interrealm_read_only_declared_type.txtar` does exercise `string(Bytes)` so the case is covered overall, but it's worth fixing the obvious-coverage gap in the primary txtar too. Fix: add `if string(msg) == "hello bug" { println("Equals") }` and a matching `stdout 'Equals'`.
  </details>

- **[no negative test for still-blocked conversions]** [`gno.land/pkg/integration/testdata/`](https://github.com/gnolang/gno/tree/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata) — neither new txtar verifies that a non-string target (`[]int32(externalBytes)`, `[]float64(...)`, or a struct conversion) still panics. The existing `zrealm_launder_*` and `zrealm_p_convert_readonly_ok_filetest.gno` cover the unrelaxed paths, but having a sibling case in the same test file makes the boundary explicit alongside the allowed case.

## Suggestions

- [`gnovm/pkg/gnolang/op_expressions.go:771`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gnovm/pkg/gnolang/op_expressions.go#L771) — consider also checking `baseOf(t).Kind() == StringKind` rather than `t.Kind() == StringKind`. The current check is fine because `Kind()` on a `*DeclaredType` whose base is `string` already returns `StringKind`, but using `baseOf` consistently on both sides (the loop and the gas calc on lines 789/794 already use `baseOf(t)`) is more uniform. Optional.

- [`gno.land/pkg/integration/testdata/interrealm_read_only_declared_type.txtar`](https://github.com/gnolang/gno/blob/310bc011759c4532bdc7ce5ff7426e08ba78e69f/gno.land/pkg/integration/testdata/interrealm_read_only_declared_type.txtar) — consider merging the two txtars. The declared-type case is two extra lines on top of the first test; one file with the full matrix (named `interrealm_string_conversions.txtar` or similar) reads better than two files split by feature increment.

- Consider an ADR in `gnovm/adr/`. The change weakens a security-sensitive guard, and the reasoning (copy in `ConvertTo`, immutable target) is non-trivial. An ADR captures it for future auditors who shouldn't have to reverse-engineer it from the git log.

## Questions for Author

- The PR description says `[]byte -> string` but the implementation also allows `[]int32 → string`. Was the `Int32Kind` addition motivated by the [@omarsy](https://github.com/gnolang/gno/pull/4831#discussion_r2411562767) suggestion or by an independent use case? A one-line "also covers `string([]rune)` because runes are `[]int32` under the hood" in the PR body would help future readers.
- Why not also handle the symmetric case `string(externalRealm) → []byte`? Today `[]byte(externalRealmString)` works because strings are primitive-immutable (Case 1 short-circuits on `xv.T.IsImmutable()`), but is there a reason this PR is one-way? Worth a sentence in the commit message.
- CI shows three failing checks (`docs`, `gno-checks / lint`, `main / test`). Inspection of the logs shows: `main/test` failure is in `params_valset_rotation_throttle.txtar` (unrelated, "rotation throttled"), `docs` is a remote-link check on `https://docs.gno.land/` (unrelated), `gno-checks/lint` is a pre-existing lint failure on `gno.land/r/ursulovic/registry` (unrelated). All three look like flakes/master-state issues, not PR regressions. Worth confirming with a re-run before merge.
