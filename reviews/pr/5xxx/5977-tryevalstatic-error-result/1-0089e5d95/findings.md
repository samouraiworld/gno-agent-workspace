# Findings in posting order, from round assemble: 3 to post, 0 SKIP, 0 refuted kept out

## gnovm/pkg/gnolang/preprocess.go:2064 [gh](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess.go#L2064) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: restoring the base's skip of the same-realm check for a *ConstExpr callee leaves every related test green
Check: bash tests/judge-1001-constexpr-callee.sh <head clone> <base clone>, part 2: restore the base's skip for a *ConstExpr callee at head and rerun TestTryEvalStatic and TestFiles/zrealm_crossrealm
Details: tryEvalStatic returns err nil on its *ConstExpr early return; at base the `if err == nil` branch skipped the check for that callee, at head the else-if chain runs. With `err` forced non-nil for a *ConstExpr callee at head, TestTryEvalStatic passes and all 57 zrealm_crossrealm filetests pass. The only Gno source shape reaching the check with a *ConstExpr callee is a typed-nil conversion (candidate 1001), whose verdict is the same either way, so pinning the rejection of a *ConstExpr callee needs a Go test that builds the CallExpr.
Evidence: mutation at head: 'ok github.com/gnolang/gno/gnovm/pkg/gnolang 0.011s' for TestTryEvalStatic; 57 '--- PASS' and 'ok ... 3.791s' for TestFiles/zrealm_crossrealm
Artifact: tests/judge-1001-constexpr-callee.sh
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: `(func(cur realm))(nil)(cur)` reaches the cur-call check with a *ConstExpr callee, the path the change reroutes, and no fixture builds it
Check: bash tests/judge-1001-constexpr-callee.sh <head clone> <base clone>, part 1: filetest `(func(cur realm))(nil)(cur)` inside a crossing func with a print on a *ConstExpr callee, at head and base
Details: At head the *ConstExpr callee now runs the else-if chain and reaches GetUnboundFunc, which returns nil for the nil value, so preprocessing falls through and the call fails at runtime with 'call of nil function'; at base the same input took the `err == nil` skip and ends with the same runtime error. The ADR's alternative 4 rejects such a filetest as not constructible; this input constructs it. A grep of gnovm/tests, gno.land/pkg/integration/testdata and examples for `)(nil)(cur` returns nothing.
Evidence: head: 'JUDGE_CONSTFUNC err= false undef= false fvnil= true', '+runtime error: call of nil function' at zrealm_curcall_constnil.gno:5; base: the same two lines
Artifact: tests/judge-1001-constexpr-callee.sh

## gnovm/pkg/gnolang/preprocess_tryevalstatic_test.go:68 [gh](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess_tryevalstatic_test.go#L68) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: TestTryEvalStatic_ErrSetOnFailure's err comes from EvalStatic's not-preprocessed precondition, so it stays green when tryEvalStatic loses every evaluation failure
Check: bash tests/judge-1-errset-shapes.sh <head clone>: log the err of Nx("Undeclared") beside preprocessed callees, then swallow every panic but EvalStatic's not-preprocessed guard and rerun TestTryEvalStatic
Details: Nx("Undeclared") is never preprocessed, so m.EvalStatic panics at machine.go:1060-1064 before m.Run. The failures the sole caller meets come from inside evaluation of a preprocessed callee: a preprocessed interface-method selector `fooer.Foo` panics 'cannot resolve an interface path at static time'. With tryEvalStatic changed to swallow every panic except the precondition one, both PR tests still pass.
Evidence: JUDGE unpreprocessed Nx(Undeclared): err=recovered panic with: Machine.EvalStatic(x) expression not yet preprocessed: Undeclared<VPInvalid(0)>; JUDGE preprocessed fooer.Foo: err=recovered panic with: cannot resolve an interface path at static time undefined=true; mutation: --- PASS: TestTryEvalStatic_ErrNilOnSuccess, --- PASS: TestTryEvalStatic_ErrSetOnFailure
Artifact: tests/judge-1-errset-shapes.sh
State: CONFIRMED, band: Missing test, angle: reach
TL;DR: the negative control never reaches m.Run, so a regression dropping err for a panic raised during evaluation keeps it green
Check: bash tests/judge-1-errset-shapes.sh <head clone>: print err for the test's input and for a preprocessed callee; mutate tryEvalStatic to drop evaluation panics and rerun TestTryEvalStatic_ErrSetOnFailure
Details: Same mechanism read independently: machine.go:1060 `if x.GetAttribute(ATTR_PREPROCESSED) == nil { panic(...) }` fires on the bare Nx("Undeclared"). A preprocessed undeclared name cannot be built at all, since Preprocess rejects it, so the test's 'undeclared name' only ever reaches the precondition. A preprocessed `fooer.Foo` fails inside evaluation and is the shape the caller's `if err != nil` branch exists for.
Evidence: mutation keeping only the not-preprocessed panic: '--- PASS: TestTryEvalStatic_ErrSetOnFailure (0.00s)', 'ok github.com/gnolang/gno/gnovm/pkg/gnolang'
Artifact: tests/judge-1-errset-shapes.sh

## gnovm/pkg/gnolang/preprocess.go:4344 [gh](https://github.com/gnolang/gno/blob/0089e5d957dd05eb3892122d050c04a0fda6a1e9/gnovm/pkg/gnolang/preprocess.go#L4344) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the comment `// FIX: no panic -> eval succeeded -> leave err nil` names the patch, not what the return does
Check: sed -n 4344p gnovm/pkg/gnolang/preprocess.go at head
Details: Read at head: line 4344 is `return // FIX: no panic -> eval succeeded -> leave err nil`. No enabled linter checks comment wording.
Evidence: preprocess.go:4344 `return // FIX: no panic -> eval succeeded -> leave err nil`
