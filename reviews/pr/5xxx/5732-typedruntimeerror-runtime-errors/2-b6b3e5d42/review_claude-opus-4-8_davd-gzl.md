# PR [#5732](https://github.com/gnolang/gno/pull/5732): fix(gnovm): typedRuntimeError for runtime errors

URL: https://github.com/gnolang/gno/pull/5732
Author: Villaquiranm | Base: master | Files: 13 | +181 -65
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `b6b3e5d42` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5732 b6b3e5d42`

Round 2 (head advanced `d716c5286` → `b6b3e5d42`, patch-ids differ: real PR content). Round 1's blocking finding (incomplete migration) is resolved: the migration is now complete. Verdict moves REQUEST CHANGES → APPROVE.

**TL;DR:** In Go, a runtime panic (nil dereference, divide by zero, index out of range, ...) is a value that implements `error`, so `recover().(error)` works. Gno emitted plain strings, so the same assertion failed. This PR routes every VM runtime panic through a new built-in `.runtimeError` type that implements `error`, giving Gno the same `recover().(error)` behavior as Go.

**Verdict: APPROVE** — migration complete (60 sites converted, zero string-typed `runtime error:` panics remain), Go message parity verified, and recovered `.runtimeError` values persist correctly across transactions. Non-blocking: a stale PR-body line and two small hardening suggestions below. Both maintainers (jefft0, thehowl) have already approved.

## Summary

Round 1 reviewed one migrated panic site and found the other ~20 still emitted string-typed panics, so `recover().(error)` succeeded on one shape and silently failed on the rest. Since then thehowl completed the migration: every VM panic that Go represents as a `runtime.Error` now carries a `typedRuntimeError` value. A grep of `gnovm/pkg/gnolang/*.go` finds 60 `typedRuntimeError` call sites and zero remaining `typedString("runtime error: ...")` sites. Gno-specific panics (readonly tainting, `revive()`), the generic `PanicString` helper, and internal "should not happen" asserts correctly stay plain strings, matching Go, which marks only genuine runtime faults as `runtime.Error`.

The value is a uverse `DeclaredType` named `.runtimeError` (struct `{msg string}` + a native `Error()` method); the leading `.` keeps it unspellable from user code. `recover().(error)` now succeeds uniformly; `println(recover())` still prints the message via the stringer; `recover().(string)` now returns `ok=false` (the Go-parity direction).

## Examples

| Gno panic | `recover().(error)` before | after |
|-----------|---------------------------|-------|
| `var p *int; _ = *p` | ok=false (string) | ok=true, `runtime error: nil pointer dereference` |
| `a/0` | ok=false (string) | ok=true, `runtime error: division by zero` |
| `s[5]` (len 3) | ok=false (string) | ok=true, `runtime error: slice index out of bounds: 5 (len=3)` |
| `(*T)(nil)` value-method via iface | ok=false (string) | ok=true, `value method main.T.F called using nil *T pointer` |

## Glossary

- uverse: the GnoVM universe block, outermost scope holding built-in types/values/functions (`makeUverseNode`); built-in-only names carry the `uversePkgPath` and a leading `.`.
- `.runtimeError`: the new uverse `DeclaredType` (struct `{msg string}` + native `Error()`) that satisfies the Gno `error` interface.
- `typedRuntimeError(msg)`: helper returning a `TypedValue{T: gRuntimeErrorType, V: &StructValue{...}}`; the replacement for `typedString` at runtime-panic sites.
- `typedString`: existing helper producing a `string`-typed `TypedValue` that does not implement `error`; retained for Gno-specific and internal panics.
- Exception: the VM's Go-level panic wrapper; gno `recover()` returns `Exception.Value`.

## Fix

Runtime-panic sites that used `typedString("runtime error: …")` now use `typedRuntimeError(…)`, which builds a `.runtimeError`-typed value. The type is declared at [`uverse.go:34-45`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/uverse.go#L34-L45) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L34-L45), registered with its native `Error()` at [`uverse.go:572-583`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/uverse.go#L572-L583) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L572-L583), and the helper is [`values.go:2853-2858`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/values.go#L2853-L2858) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L2853-L2858). The nil-pointer value-method path keeps its richer Go-matching message at [`values.go:1826-1836`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/values.go#L1826-L1836) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L1826-L1836). Coverage: [`recover26.gno`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/tests/files/recover26.gno) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/tests/files/recover26.gno) asserts `recover().(error)` across ten panic shapes; [`ptr11c.gno`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/tests/files/ptr11c.gno) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/tests/files/ptr11c.gno) mirrors Go's `test/fixedbugs/issue19040.go`.

## Verification

Verified on `b6b3e5d42` (checks CI does not run):

- The nil-pointer value-method message matches the real Go compiler byte-for-byte: `go run` of the equivalent `main.T.F` program prints `value method main.T.F called using nil *T pointer`, identical to `ptr11c.gno`'s golden.
- A recovered `.runtimeError` survives a store round-trip: a realm that recovers a divide-by-zero and assigns it to a package-level `error` persists it (storage delta 426 bytes, storage-fee charged), and a later transaction reads `stored.Error()` back as `runtime error: division by zero`. Deserialization of the new uverse type across tx boundaries works. See [`tests/recover_runtimeerror_persist.txtar`](tests/recover_runtimeerror_persist.txtar).
- `recover().(string)` now returns `ok=false` for a VM runtime panic (the Go-parity direction); no in-tree realm relies on that assertion for VM panics (`grep -rn 'recover().(string)' examples/` is empty; the one `r.(string)` use in `examples/quarantined/.../tokenhub_test.gno` recovers a user `panic("...")`, which is unchanged).

## Critical (must fix)

None.

## Warnings (should fix)

None. Round 1's Warnings are resolved or reclassified:
- The `recover().(string)` type change is the PR's intended Go-parity behavior; it is documented in PR-body point 2 and has zero in-tree blast radius (see Verification).
- The nil-pointer-method-via-realm concern is covered: the persistence round-trip is verified above.

## Nits

None.

## Missing Tests

- **[recovered runtime error persisting across transactions is untested]** [`recover26.gno`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/tests/files/recover26.gno) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/tests/files/recover26.gno) — `recover26.gno` proves `recover().(error)` within one VM run; nothing exercises a recovered `.runtimeError` stored in realm state and read back in a later block.
  <details><summary>details</summary>

  The `.runtimeError` value is now storable in on-chain state (a realm can `stored = recover().(error)`), so its amino serialization and reload matter. A committed txtar locks that down. Verified passing on `b6b3e5d42`: [`tests/recover_runtimeerror_persist.txtar`](tests/recover_runtimeerror_persist.txtar) stores a recovered divide-by-zero in tx1 and reads `runtime error: division by zero` back in tx2. Fix: add it (or an equivalent) under `gno.land/pkg/integration/testdata/`.
  </details>

## Suggestions

- [`values.go:2853-2858`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/values.go#L2853-L2858) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L2853-L2858) — `typedRuntimeError` allocates a `*StructValue` and a `[]TypedValue` on the Go heap, outside `m.Alloc`, whereas the sibling `typedString` is documented "does not allocate" at [`values.go:2843`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/values.go#L2843) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/values.go#L2843). Not exploitable: the panic path is bounded by opcode gas and persistence is storage-fee-metered (verified: 426-byte storage delta). Several call sites are pure functions with no `Machine` in scope, so routing through `m.Alloc` is not uniformly possible; a one-line comment noting the intentional skip would prevent a future reader assuming the "does not allocate" contract carries over. Review-only.
- [`uverse.go:578-582`](https://github.com/gnolang/gno/blob/b6b3e5d42/gnovm/pkg/gnolang/uverse.go#L578-L582) · [↗](../../../../../.worktrees/gno-review-5732/gnovm/pkg/gnolang/uverse.go#L578-L582) — the native `Error()` does an unguarded `arg0.TV.V.(*StructValue)`. Unreachable in practice: `Error()` is only dispatchable on a `.runtimeError`, which `typedRuntimeError` always constructs with a `*StructValue`, so a malformed receiver cannot reach it. Defensive only; review-only.

## Open questions

- PR-body point 3 still reads "Only migrates one of ~20 panic sites … This creates an inconsistency," describing the pre-completion state; the code now migrates all sites. The author said in-thread the description was updated, but point 3 is still stale. Worth a one-line fix so anyone landing on the PR is not told a merge-ready change is incomplete. Not a code finding; noted for the author, low priority.
- The struct shape `{msg string}` is now persistable in realm state (verified). If a future PR adds fields (stack, code), `.runtimeError` values persisted by old blocks carry the old shape. General schema-evolution concern for any persisted type; not actionable here.
- Gno's type-assertion panic text (`int is not of type string`) differs from Go's (`interface conversion: int is not string`). Pre-existing, unchanged by this PR (only the value type changed), so out of scope; flagging in case message parity is later wanted.
