# Findings in posting order, from round assemble: 5 to post, 0 SKIP, 0 refuted kept out

## Body

- Warning: gnovm/pkg/gnolang/nodes.go:2389 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/nodes.go#L2389), outside the diff, so no anchor
  State: CONFIRMED, band: Warning, angle: removed
  TL;DR: y, x := 1, struct{A int "b"}{2} redeclares x of type struct{A int "a"} and silently changes its static type
  Check: tests/solo-finder-2-redeclare-tag-variant.gno as a filetest, head and base; sed -n 2389p nodes.go
  Details: StaticBlock.Define2 guards a redeclaration with `tv.T.TypeID() != old.T.TypeID()`, which ignores tags, embeddedness and variadicity; the diff moved the function-redefinition sibling in preprocess.go to identicalTypes and left this one. Predates the diff.
  Evidence: Head and base Output `2 1`; TypeCheckError 'cannot use struct{A int}{…} (value of type struct{A int "b"}) as struct{A int "a"} value in multiple assignment'. nodes.go:2389 `} else if tv.T.TypeID() != old.T.TypeID() {`.
  Artifact: tests/solo-finder-2-redeclare-tag-variant.gno
- Warning: gnovm/pkg/gnolang/type_check.go:585 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/type_check.go#L585), outside the diff, so no anchor
  State: CONFIRMED, band: Warning, angle: catalog
  TL;DR: f(s...) with s []int and f(xs ...any) preprocesses and runs, and the callee's xs[0] = "hello" lands in the caller's []int
  Check: tests/solo-finder-1-vararg-spread-iface.gno as a filetest, Output filled by -update-golden-tests, in scratch worktrees at head b42a521b1 and merge base 4c9de5225
  Details: type_check.go:584-585 `if cdt.Vrd { return checkAssignableTo(n, st.Elt, cdt.Elt)`: a spread slice needs only element assignability, the recursion the diff removed from the *PointerType case, where Go needs []int assignable to []any. Predates the diff (same line at the merge base), in scope as the missed member of the class the diff sweeps.
  Evidence: Head Output `hello` / `2`, base Output `hello` / `2`; TypeCheckError only: 'cannot use s (variable of type []int) as []any value in argument to f; ... cannot use []E{…} (value of type []E) as []error va…'.
  Artifact: tests/solo-finder-1-vararg-spread-iface.gno
  State: CONFIRMED, band: Warning, angle: removed
  TL;DR: ts []T passed as ...I lets the callee store a U in the caller's []T: ts[0].x then prints "hello" and ts[0].M() runs U.M
  Check: tests/solo-finder-2-vararg-slice-assignable.gno as a filetest, Output filled by -update-golden-tests, head and base
  Details: Same line and same edit as candidate 1; the draft carries both cases in one Body bullet.
  Evidence: Head Output `hello` / `5`, base the same; TypeCheckError 'cannot use ts (variable of type []T) as []I value in argument to f'.
  Artifact: tests/solo-finder-2-vararg-slice-assignable.gno
- Warning: gnovm/pkg/gnolang/uverse.go:858 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/uverse.go#L858), outside the diff, so no anchor
  State: CONFIRMED, band: Warning, angle: claims
  TL;DR: copy(dst []int, src []string) preprocesses and runs, leaving the string "x" in an int slot, because copy declares dst and src as the independent generics X and Y
  Check: grep -c '^// Error:' copy_struct_tags.gno; tests/solo-finder-1-copy-elem-mismatch.gno as a filetest, head and base
  Details: uverse.go:857-859 declares copy with GenT("X", nil) and GenT("Y", nil), so specifyType's identity check never compares the two element types. copy_struct_tags.gno carries a TypeCheckError and no Error line, so it pins the VM accepting the copy. Re-anchored from the test to the builtin, where the edit goes, outside the diff so a Body bullet. Predates the diff; in scope because the diff adds a test asserting the behaviour.
  Evidence: grep -c '^// Error:' copy_struct_tags.gno: 0. Head and base Output `1 x`, TypeCheckError 'invalid copy: arguments dst (variable of type []int) and src (variable of type []string) have different element types int and string'.
  Artifact: tests/solo-finder-1-copy-elem-mismatch.gno

## gnovm/pkg/gnolang/types.go:1253 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/types.go#L1253) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: x.(I) walks the whole method signature on every assertion for the same flat gas, so the CPU a realm buys per unit of gas grows with signature size
Check: go test -run TestSF2IdentityCost -v in the head scratch tree; the draft's repro block run verbatim with the gno on PATH in place of its go build line
Details: The diff replaces `dmtid := mt.TypeID() ... if dmtid != imtid` with `if !identicalTypes(mt, im.Type)` in VerifyImplementedBy; the gas charged (OpCPUSlopeTypeAssertIface per method) is not touched by the diff. The base compare also scaled with signature size through the string compare, 70 to 115 times less.
Evidence: Microbench at head: fields=100 1159 vs 14 ns/op (82.8x), fields=1000 10890 vs 97 (112.3x), fields=10000 120463 vs 1681 (71.7x). Draft block, 20000 asserts: 263 B GAS 42886054 TestLoop 0.01s; 36363 B GAS 42886532 0.56s; 90603 B GAS 42900410 1.39s.
Artifact: tests/solo-finder-2-identity-cost_test.go
State: CONFIRMED, band: Suggestion, angle: claims
TL;DR: deployed code asserting x.(interface{ M(...int) }) on a T with M([]int) answers true at the base and false at head, a replay-visible change
Check: tests/judge-3-runtime-iface-assert.gno at head and at the merge base
Details: The ADR's Decision and Consequences record the runtime interface-assertion change as intended, while its line 132 lists type switches among the unchanged runtime semantics; the contradiction is ADR text, so the draft skips it.
Evidence: Head Output false / default / false; base Output true / case I / true.
Artifact: tests/judge-3-runtime-iface-assert.gno

## gnovm/tests/files/interface51.gno:12 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/tests/files/interface51.gno#L12) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: no filetest runs x.(I) or a type-switch case I where the method differs from I's only in variadicity or a tag
Check: grep -rlE 'M\(\.\.\.|\(\.\.\.int\)' gnovm/tests/files; read typeassert_struct_tags.gno and typeswitch_struct_tags.gno; tests/judge-3-runtime-iface-assert.gno head and base
Details: The three interface-variadic filetests fail at preprocess; typeassert_struct_tags and typeswitch_struct_tags assert concrete struct types only. The judge's filetest answers false/default/false at head and true/case I/true at the base, so it catches a revert of the runtime half and nothing in the diff does.
Evidence: Head Output false / default / false; base Output true / case I / true.
Artifact: tests/judge-3-runtime-iface-assert.gno
