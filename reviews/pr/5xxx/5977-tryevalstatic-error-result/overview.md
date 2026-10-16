# `tryEvalStatic`'s error result
Written by claude-opus-5-5.
PR: [gnolang/gno#5977](https://github.com/gnolang/gno/pull/5977)

## TLDR

`tryEvalStatic` evaluates an expression on a throwaway machine. Before, its deferred `recover()` set `err` even when nothing panicked, so `err` was non-nil on every evaluated input, and its one caller, the same-realm check for a `f(cur, ...)` call, skipped the check on `err == nil`. The two inversions cancelled. After, `err` is nil on success, and the caller skips on `err != nil`. Every evaluated callee reaches the same verdict; a `*ConstExpr` callee, which `tryEvalStatic` returns without evaluating, goes through the check instead of skipping it. An ADR records why the two lines move together, and a Go test pins the error contract.

## Before and after

Both columns describe how the check at `preprocess.go:2064` routes each callee shape, through `tryEvalStatic`.

| Callee at a `f(cur, ...)` call | Before | After |
| --- | --- | --- |
| a package-level func, `Foo(cur)` | `err` non-nil, falls to the `else if` chain, which compares `fv.PkgPath` | `err` nil, same chain, same verdict |
| one evaluation cannot resolve, `fooer.Foo(cur)` | `err` non-nil, `tv` undefined, skipped by `IsUndefined()` | skipped by `err != nil` |
| a `*ConstExpr`, `(func(cur realm))(nil)(cur)` | `err` nil, skipped | runs the chain; `GetUnboundFunc()` is nil, so the runtime decides |

## Read the code in this order

### 1. `hot` · The recover guard, [`preprocess.go`](https://github.com/gnolang/gno/pull/5977/files#diff-ebdb0494324ba82339ca88219105656407e3f3a49240cf32c24c9a266d31e0f4R4342-R4345)
Returns from the deferred function when `recover()` gives nil, so a successful evaluation leaves `err` nil.
**If wrong:** `err` reports failure on success, and the caller skips the same-realm check for every resolved callee.
```go
r := recover()
if r == nil {
	return // FIX: no panic -> eval succeeded -> leave err nil
}
```

### 2. `hot` · The same-realm check, [`preprocess.go`](https://github.com/gnolang/gno/pull/5977/files#diff-ebdb0494324ba82339ca88219105656407e3f3a49240cf32c24c9a266d31e0f4R2064-R2066)
Defers to the runtime when the callee cannot be evaluated, and otherwise rejects a callee whose `FuncValue` lives in another realm.
**If wrong:** a package carrying a cur-call into another realm deploys, and fails only when the call runs.
```go
ftv, err := tryEvalStatic(store, ctxpn, last, n.Func)
if err != nil {
	// Couldn't resolve statically; e.g. somefunc()(cur,...) or a
	// local variable. Defer to the runtime check.
} else if ftv.IsUndefined() {
```

### 3. `cold` · The contract test, [`preprocess_tryevalstatic_test.go`](https://github.com/gnolang/gno/pull/5977/files#diff-6890a8f52fee3ffc9be46e97d3b88717d9662bcb856ade7ca6e12a00e6665ad6R32-R71)
`TestTryEvalStatic_ErrNilOnSuccess` asserts a nil `err` for a resolved package-level func; `TestTryEvalStatic_ErrSetOnFailure` asserts a non-nil one for an unpreprocessed name.
**If wrong:** the recover guard can be dropped with no Go test failing.

### 4. `cold` · The decision record, [`pr5977_tryevalstatic_err_contract.md`](https://github.com/gnolang/gno/pull/5977/files#diff-5093904e1bea3b22699eae1eb4cfd128129451993ea7b549b8a38afaa8d8e354R1-R117)
Explains why fixing either line alone disables the compile-time check, and which existing tests catch it.
**If wrong:** the next reader fixes the `recover()` alone.

## Words used here

| Name | What it is |
| --- | --- |
| `tryEvalStatic` | Evaluates an expression at preprocess time on a throwaway machine, returning `(tv TypedValue, err error)`; a `*ConstExpr` returns its value with no evaluation. |
| cur-call | A non-crossing call of a crossing function, `f(cur, ...)`, allowed only into the caller's own realm. |
| `*ConstExpr` | An expression the preprocessor already folded to a value, such as a typed-nil conversion. |
| `GetUnboundFunc` | Returns the `*FuncValue` a `TypedValue` holds, or nil for a nil value. |
| `EvalStatic` | The machine method `tryEvalStatic` calls; it panics on an expression without `ATTR_PREPROCESSED`. |
