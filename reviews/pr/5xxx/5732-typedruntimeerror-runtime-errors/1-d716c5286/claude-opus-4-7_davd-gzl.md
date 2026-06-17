# PR #5732: fix(gnovm): typedRuntimeError for runtime errors

URL: https://github.com/gnolang/gno/pull/5732
Author: Villaquiranm | Base: master | Files: 6 | +81 -4
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `d716c5286` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5732 d716c5286`

**Verdict: REQUEST CHANGES** — author explicitly flags the PR as incomplete; ~20 other `typedString("runtime error: ...")` panic sites still emit string-typed panics, so `recover().(error)` works on exactly one runtime-panic shape and silently fails on every other (nil-deref via field path, division by zero, slice OOB, negative shift, makeslice, call-of-nil-func, map-key, etc.). Either migrate all sites in this PR or split into a tracked epic and gate merge on completion.

## Summary

Introduces a new uverse declared type `.runtimeError` (struct with one `msg` string field, native `Error()` method) and routes one panic site — `VPDerefValMethod` with a nil receiver in `(*TypedValue).GetPointerToFromTV` — through a new `typedRuntimeError(msg)` helper instead of `typedString("runtime error: nil pointer dereference")`. The new message also enriches the text from generic "nil pointer dereference" to Go's `value method <pkg>.<Type>.<Method> called using nil *<Type> pointer` format, matching `test/fixedbugs/issue19040.go` and resolving the symptom in #5667.

Why it matters: Go runtime panics implement `error`; Gno's emitted `string`. Realm code that does `recover().(error)` on any nil-pointer panic currently silently mis-asserts (`ok=false`) and the panic propagates as `string doesn't implement interface {Error() string}`. This is a real footgun — and now it's partially fixed in one path while ~20 sister paths still fail identically.

## Glossary

- `.runtimeError`: new uverse-internal `DeclaredType` (struct `{msg string}` + native `Error() string` method) that satisfies the Gno `error` interface. Name starts with `.` (uverse convention, like `.uverse`, `.grealm`).
- `typedRuntimeError(msg)`: helper returning a `TypedValue{T: gRuntimeErrorType, V: &StructValue{...}}` — replacement for `typedString` at panic sites that want recover-as-error semantics.
- `VPDerefValMethod`: ValuePath type used when calling a value-receiver method through a pointer (e.g. `(*S)(nil).String()` or `i.Hello()` where `i` is an interface holding `(*S)(nil)`). The only path migrated by this PR.
- `typedString`: existing helper (`values.go:2770`) used by the other ~20 panic sites; produces a string-typed `TypedValue` that does **not** implement `error`.

## Fix

Before: panic at [`values.go:1828`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L1828) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L1828) used `typedString("runtime error: nil pointer dereference")` — `recover()` returned a `string`-typed value; `recover().(error)` failed.

After: same site now builds a context-aware message (`value method <PkgPath>.<Type>.<Method> called using nil *<Type> pointer` when `tv.T` is `*PointerType` over `*DeclaredType`, falling back to the old text otherwise) and wraps it in `typedRuntimeError`, returning a `.runtimeError`-typed value that satisfies the `error` interface. Three filetests (`ptr11.gno`, `ptr11a.gno`, `ptr11b.gno`) and one new txtar (`nil_ptr_method_recover.txtar`) cover the recovered and unrecovered messages. See [`uverse.go:26-40`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L26-L40) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L26-L40) for the type, [`uverse.go:567-578`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L567-L578) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L567-L578) for registration, and [`values.go:2779-2784`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L2779-L2784) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L2779-L2784) for the helper.

## Critical (must fix)

- **[incomplete migration — same bug survives in ~20 sister sites]** [`values.go:1813,2052,2059,2269,2348`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L1813) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L1813), [`op_binary.go:936,1035,1297,1381`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/op_binary.go#L936) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/op_binary.go#L936), [`op_call.go:13,50`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/op_call.go#L13) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/op_call.go#L13), [`op_exec.go:216`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/op_exec.go#L216) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/op_exec.go#L216), [`op_expressions.go:120,163`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/op_expressions.go#L120) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/op_expressions.go#L120), [`machine.go:2738`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/machine.go#L2738) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/machine.go#L2738), [`uverse.go:1081,1116,1119,1122`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L1081) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L1081) — `recover().(error)` returns `ok=false` for every runtime panic except the one migrated path.
  <details><summary>details</summary>

  Author already names this in the PR body ("Only migrates one of ~20 panic sites… This PR is incomplete as before merging we should migrate all occurrences of `typedString(\"runtime error:\")...`"). Surfacing it as Critical so the merge gate is explicit.

  **Shape:** every `typedString("runtime error: …")` and `typedString(fmt.Sprintf("runtime error: …"))` in `gnovm/pkg/gnolang/` is a panic value that fails the `error` assertion. `grep -n "runtime error" gnovm/pkg/gnolang/*.go` returns ~25 sites; the PR migrates one.

  **What you see:** `recover().(error)` works on nil-method-call-via-interface (this PR), but a user who writes the same idiom for `*x` deref, `a/0`, slice OOB, `var f func(); f()`, etc. gets `ok=false` and their `defer` propagates `string doesn't implement interface {Error() string}` — i.e. a fix that silently doesn't apply.

  **Why it matters:** a partial migration is worse than none — it sets the expectation that `recover().(error)` works (per `ptr11a.gno`), then breaks that expectation on neighbouring panics. Realm code that tries `if e, ok := recover().(error); ok { handle(e) }` will silently swallow some panics and crash on others depending on which VM site emitted them. Determinism note: the panic-recovery codepath is on the deterministic side of consensus; changing a panic value's type is a state-machine change for any contract that recovers and inspects the recovered value.

  **Fix:** before merge, route every remaining `typedString("runtime error: …")` and `typedString(fmt.Sprintf("runtime error: …"))` in `gnovm/pkg/gnolang/` through `typedRuntimeError`. The mechanical version is a one-line swap per site (no message change required); the richer version (per-site context like the diff does for `dt`) can land separately. If the scope is too large for one PR, gate merge of #5732 on an issue tracking the rest, and call out the partial coverage in `docs/` as a known footgun until then.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5732 -R gnolang/gno
  cat > /tmp/main.gno <<'EOF'
  package main

  func main() {
  	defer func() {
  		r := recover()
  		if e, ok := r.(error); ok {
  			println("error path:", e.Error())
  			return
  		}
  		println("recover().(error) FAILED — r is type:", r)
  	}()
  	a, b := 1, 0
  	println(a / b)
  }
  EOF
  go run ./gnovm/cmd/gno run /tmp/main.gno
  rm /tmp/main.gno
  ```
  ```
  panic running expression main(): runtime error: division by zero
  	string doesn't implement interface {Error func() string} (missing method Error)
  Stacktrace:
  panic: runtime error: division by zero
  main<VPBlock(1,0)>()
      main//tmp/main.gno:9
  panic: string doesn't implement interface {Error func() string} (missing method Error)
  defer func(){ ... }()
      main//tmp/main.gno:5
  ```

  The exact same `recover().(error)` idiom that `ptr11a.gno` validates as the desired post-PR behaviour fails one panic site over.
  </details>

## Warnings (should fix)

- **[silent behavior change for `recover().(string)`]** [`values.go:1827-1838`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L1827-L1838) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L1827-L1838) — pre-PR, `recover().(string)` on a nil-method-call returned the message; post-PR it returns `ok=false`.
  <details><summary>details</summary>

  This is the Go-parity behaviour, so it's defensible, but the PR description doesn't mark it as a breaking change. Any existing Gno realm doing `if s, ok := recover().(string); ok { log(s) }` to surface VM panics will silently stop logging the migrated path. Combined with the partial migration above, the same realm will keep logging non-migrated panic shapes — inconsistent behaviour across panic types.

  **Fix:** add a `BREAKING CHANGE:` line to the PR body (and the eventual commit) calling out that runtime-error panic values change type from `string` to `.runtimeError`, and update any docs/examples that show `recover().(string)` for VM panics. Cross-check `examples/` for the idiom: `grep -rn "recover().(string)" examples/`.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5732 -R gnolang/gno
  cat > /tmp/main.gno <<'EOF'
  package main

  type S struct{}
  func (s S) Hello() string { return "hi" }

  func main() {
  	defer func() {
  		s, ok := recover().(string)
  		println("string assertion ok=", ok, "msg=", s)
  	}()
  	v := (*S)(nil)
  	v.Hello()
  }
  EOF
  go run ./gnovm/cmd/gno run /tmp/main.gno
  rm /tmp/main.gno
  ```
  ```
  string assertion ok= false msg=
  ```
  </details>

- **[unmetered allocation on panic path]** [`values.go:2779-2784`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L2779-L2784) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L2779-L2784) — `typedRuntimeError` does `&StructValue{Fields: []TypedValue{…}}` directly; bypasses `m.Alloc`.
  <details><summary>details</summary>

  `typedString` is documented as "does not allocate" ([`values.go:2769`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L2769) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L2769)) because `StringValue` is the type and the string itself is the value — no Go heap object beyond the immutable string header. `typedRuntimeError` allocates a `*StructValue` and a `[]TypedValue{...}` slice — both Go-heap objects — without going through the VM allocator. For a panic path this is fine in practice (panics aren't a hot loop) but it does mean the new path is invisible to `m.Alloc`'s accounting if a realm catches the panic and stashes the value. Either route through `m.Alloc.NewStruct` for symmetry with normal struct construction, or document why the panic path intentionally skips the allocator.
  </details>

- **[missing test: nil-pointer-method via interface returns the right error in a realm context]** [`gnovm/cmd/gno/testdata/test/nil_ptr_method_recover.txtar`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/cmd/gno/testdata/test/nil_ptr_method_recover.txtar) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/cmd/gno/testdata/test/nil_ptr_method_recover.txtar) — the new txtar runs `gno run`, not `gno test`, and doesn't exercise a realm `.gno` package.
  <details><summary>details</summary>

  The recover-as-error contract is most load-bearing inside realms (where consensus-relevant code may catch a runtime panic and inspect it). The existing coverage exercises only the `gno run .` path and a few uverse filetests. Add a txtar under `gno.land/pkg/integration/testdata/` that imports the new type via interface from inside a deployed realm and asserts `recover().(error).Error()` returns the expected message — matches the shape of existing tests like `examples/gno.land/...` that round-trip realm state.
  </details>

## Nits

- [`uverse.go:38-39`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L38-L39) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L38-L39) — comment says "Error() method defined in makeUverseNode()" but the function name is `makeUverseNode`; harmless typo of intent (parens shouldn't be there). Either remove the parens or link to `uverse.go:568`.
- [`values.go:1832-1834`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L1832-L1834) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L1832-L1834) — `fmt.Sprintf("value method %s.%s.%s called using nil *%s pointer", dt.PkgPath, dt.Name, path.Name, dt.Name)`. If `dt.PkgPath == ""` (unlikely but not impossible — anonymous packages, edge constructions), the message becomes `.S.String called using nil *S pointer` with a leading dot. Worth a guard or a one-line test that constructs the edge.
- [`uverse.go:29`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L29) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L29) — `gRuntimeErrorType` follows the `g*Type` naming for `gErrorType` / `gStringerType`, good consistency.

## Missing Tests

- **[no test for fallback message when `pt.Elt` is not a `*DeclaredType`]** [`values.go:1830-1836`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/values.go#L1830-L1836) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L1830-L1836) — only the "rich message" path is covered; the fallback (`msg = "runtime error: nil pointer dereference"` retained) has no direct test.
  <details><summary>details</summary>

  Add a filetest covering a nil pointer to a non-`DeclaredType` element type so the fallback branch is exercised. Without it, a future refactor could break the fallback without any test catching it.
  </details>

- **[no test for `recover().(error)` from inside a realm crossing call]** — same scope as the Warning above. Worth a `.txtar` if not added before merge.

## Suggestions

- [`uverse.go:573-577`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L573-L577) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L573-L577) — the native `Error()` body asserts `arg0.TV.V.(*StructValue)` unguarded. If the receiver is somehow nil-valued (shouldn't happen for this type, but defensive), this is a host-level panic that escapes the VM. A guarded assertion (`sv, ok := arg0.TV.V.(*StructValue); if !ok { m.PushValue(typedString("")); return }`) would be safer.
- Consider an `IsRuntimeError(t Type) bool` helper symmetric with `IsErrorType` at [`uverse.go:65-72`](https://github.com/gnolang/gno/blob/d716c5286/gnovm/pkg/gnolang/uverse.go#L65-L72) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L65-L72) — useful once more panic sites are migrated and host-side code needs to distinguish runtime panics from user panics.
- The migration of remaining sites could share a tiny helper like `typedRuntimeError(fmt.Sprintf("nil slice index out of range"))` rather than re-string-formatting each site, to keep the prefix convention consistent.

## Questions for Author

- Plan for migrating the remaining ~20 sites — same PR (extended), follow-up PR(s), or a tracked epic? The PR body says "before merging we should migrate all occurrences"; the merge gate should be explicit.
- Should the partial migration land as `experimental` / behind a flag, or is the breaking change to `recover().(string)` acceptable on master without a deprecation cycle?
- Why a struct over `&PrimitiveValue{}`-style or a typed string `type runtimeError string` with a method — was the struct shape chosen for future fields (stack, error code), or is `{msg string}` permanent? Affects whether realms persisting `.runtimeError` values across blocks would survive a future schema bump.
- The new message format matches Go for `value method` calls. Will the future migration of other sites also adopt Go's exact wording (e.g. Go's nil-deref message is `"runtime error: invalid memory address or nil pointer dereference"`, not Gno's current `"runtime error: nil pointer dereference"`)?
