# PR #5715: fix(gnovm): recoverable panic on nil interface method call

URL: https://github.com/gnolang/gno/pull/5715
Author: omarsy | Base: master | Files: 3 | +122 -1
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `6d5b684c4` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5715 6d5b684c4`

**Verdict: APPROVE** — one-line fix converts a raw Go panic into a `*Exception` so user `recover()` catches it; matches the documented in-file convention and the precedent set by PRs #5711, #5195, #4452, #3856. CI green, filetest added, no regressions in the `interface*` filetests.

## Summary

Calling a method on a nil interface (`var i I; i.M()`) raised a raw string panic in [`values.go:1977-1979`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1977-L1979) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1977-L1979). The VM's recover loop in [`machine.go:1641-1650`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/machine.go#L1641-L1650) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/machine.go#L1641-L1650) only catches Go-level panics whose value is `*Exception` — anything else is re-raised — so user-level `defer/recover()` never observed the panic and the host program died with a Go stack trace. The fix replaces the raw panic with `panic(&Exception{Value: typedString("runtime error: method call on nil interface")})`, which routes through the existing pushPanic path and becomes catchable in Gno.

## Glossary

- `GetPointerToFromTV` — selector dispatch for typed values; called from `doOpSelector`, the assignment LHS handler, and the debugger.
- `VPInterface` — value-path kind for selecting a method via an interface receiver.
- `IsUndefined()` — `tv.T == nil`, i.e. nil interface; distinct from a non-nil interface holding a typed-nil receiver.
- `*Exception` — the only Go-panic shape that `runOnce` re-throws into the Gno recover machinery.

## Fix

[`values.go:1977-1979`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1977-L1979) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1977-L1979) swaps the raw `panic("interface method call on undefined value")` for a `*Exception` panic with the `runtime error:` prefix established by [#5501](https://github.com/gnolang/gno/pull/5501) and the descriptive style of [#5711](https://github.com/gnolang/gno/pull/5711)'s `runtime error: call of nil function`. `GetPointerToFromTV` has no `*Machine`, so `m.Panic()` is unavailable — the file already uses raw `panic(&Exception{...})` for the same reason in five other places ([1659](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1659) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1659), [1686](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1686) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1686), [1813](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1813) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1813), [1829](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1829) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1829), [2021](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L2021) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L2021)), and the convention is documented at [`machine.go:2804`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/machine.go#L2804) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/machine.go#L2804). The branch already lands on master @ `a7e4c34b0` (PR #5711); this PR closes the symmetric gap for interfaces.

The fix also benefits the `VPDerefInterface` path (`(*pi).M()`) without extra code: [`values.go:1844-1847`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/pkg/gnolang/values.go#L1844-L1847) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/pkg/gnolang/values.go#L1844-L1847) rewrites `path.Type = VPInterface` before falling into the same case, so any nil-interface reached through a deref also produces the recoverable panic.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/adr/pr5715_recoverable_nil_interface_method_call.md`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/adr/pr5715_recoverable_nil_interface_method_call.md) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/adr/pr5715_recoverable_nil_interface_method_call.md) — 96-line ADR for a 1-line diff. Matches recent precedent (#5711 ships without one but #5501 / similar do); no objection if the project wants the audit trail.
- The Gno message `runtime error: method call on nil interface` is more descriptive than Go's `runtime error: invalid memory address or nil pointer dereference` for the same program. ADR calls this out explicitly and prefers the Gno-specific wording, matching #5711's `call of nil function`. Worth a maintainer note if exact Go parity is a goal anywhere downstream.

## Missing Tests

- [`gnovm/tests/files/interface49.gno`](https://github.com/gnolang/gno/blob/6d5b684c4/gnovm/tests/files/interface49.gno) · [↗](../../../../../.worktrees/gno-review-5715/gnovm/tests/files/interface49.gno) covers the bare nil-interface case. A typed-nil receiver test (`var p *S; var i I = p; i.M()` — should call the method, not panic) would lock in that `IsUndefined()` distinguishes `tv.T == nil` from a non-nil interface holding a typed-nil concrete value. The fix is correct here by construction (the check is `dtv.T == nil`, which is false for typed nil), but the regression risk is non-zero if someone later "tightens" the nil check. Not a blocker.

## Suggestions

None.

## Questions for Author

None.

---

### Verification

- `go test -v -run 'TestFiles/interface49.gno$' ./gnovm/pkg/gnolang/` → PASS.
- `go test -v -run 'TestFiles/interface' ./gnovm/pkg/gnolang/` → all 30+ `interface*.gno` filetests pass.
- `go vet ./gnovm/pkg/gnolang/` → clean.
- Pre-existing master failures in `TestFiles/types/{and,eql,or}_*.gno` (typecheck error-message wording) reproduce on `origin/master` and are unrelated.
- `grep -rn "interface method call on undefined"` outside the ADR → zero hits; no downstream test or stdlib code depends on the old string.

### Repro

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5715 -R gnolang/gno
cat > gnovm/tests/files/repro_nil_iface.gno <<'EOF'
// run
package main

type I interface{ M() }

func main() {
	defer func() {
		err := recover()
		if err == nil {
			panic("panic expected")
		}
		println(err)
	}()
	var i I
	i.M()
}

// Output:
// runtime error: method call on nil interface
EOF
go test -v -run 'TestFiles/repro_nil_iface.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/repro_nil_iface.gno
```
