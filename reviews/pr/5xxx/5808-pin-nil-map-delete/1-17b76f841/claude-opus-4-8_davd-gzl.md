# PR #5808: test(gnovm): pin nil-map delete semantics (follow-up to #5196)

URL: https://github.com/gnolang/gno/pull/5808
Author: omarsy | Base: master | Files: 3 | +107 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 17b76f841 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5808 17b76f841`
Overview: [visual overview](https://samouraiworld.github.io/gno-agent-workspace/reviews/pr/5xxx/5808-pin-nil-map-delete/overview.html) · [↗](../overview.html)

**TL;DR:** Go's spec says `delete` on a nil map is a no-op. The actual VM fix already merged in #5196; this PR only adds tests and an ADR to pin the guard's full semantics, including one deliberate place where gno no-ops but the Go compiler panics (deleting with an unhashable key on a nil map).

**Verdict: APPROVE** — pure test + ADR follow-up, no runtime change; all three filetests pass and the documented gc divergence reproduces exactly as described. No findings.

## Summary
#5196 landed the one-line guard in the `delete` builtin (early return when the map value is nil, before the `*MapValue` type assertion that previously crashed) with a single basic filetest. This PR adds nothing to the runtime: it pins the guard's behavior across the forms that reach a nil map (package var, struct field, function return, conversion literal, cross-realm value), plus the one deliberate divergence from the gc compiler, and records two design decisions in an ADR so they aren't re-litigated. The divergence: `delete(nilMap, unhashableKey)` no-ops in gno but panics `hash of unhashable type` under gc; gno follows the spec text and stays consistent with its own pre-existing nil-map *read* behavior.

## Glossary
- **gc** — the standard Go compiler/runtime, used as the spec oracle.
- **filetest** — `gnovm/tests/files/*.gno` with an `// Output:` block the VM must match.
- **readonly taint** — cross-realm write protection: a value owned by another realm is read-only to the current one.

## Fix
No code change here. The behavior under test lives at [`uverse.go:978-979`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L978-L979) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L978-L979): `if arg0.TV.V == nil { return }`, sitting above both the `*MapValue` assertion at [`uverse.go:981`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L981) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L981) and the readonly check at [`uverse.go:983-984`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/uverse.go#L983-L984) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/uverse.go#L983-L984). The PR adds [`delete1.gno`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/delete1.gno) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno), [`zrealm_mapnil.gno`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/zrealm_mapnil.gno) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/zrealm_mapnil.gno), and the ADR.

## What I verified

The guard's two key properties both hold against the source:

- **Readonly ordering is unobservable.** A nil value cannot carry readonly taint: [`IsReadonlyBy`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/ownership.go#L461) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/ownership.go#L461) switches on `tv.V`'s concrete type and a nil `V` falls to the `default` case, [`return false`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/pkg/gnolang/ownership.go#L526-L527) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/pkg/gnolang/ownership.go#L526-L527). So moving the nil guard above or below the readonly check is behaviorally identical.
- **The gc divergence is real.** Verified against go1.26.4: all hashable-key nil-map deletes no-op, and `delete(nilMap, []int{1})` panics `hash of unhashable type: []int` under gc while gno no-ops it (pinned at [`delete1.gno:24`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/delete1.gno#L24) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/delete1.gno#L24)). gno's choice is internally consistent: a nil-map *read* with an unhashable key also no-ops in gno but panics under gc, so this PR brings `delete` in line with gno's existing read behavior, not into a new inconsistency.

Both new filetests pass, and #5196's [`map48.gno`](https://github.com/gnolang/gno/blob/17b76f841/gnovm/tests/files/map48.gno) · [↗](../../../../../.worktrees/gno-review-5808/gnovm/tests/files/map48.gno) still passes.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5808 -R gnolang/gno
go test -run 'TestFiles/^(delete1|zrealm_mapnil|map48)\.gno$' ./gnovm/pkg/gnolang/
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.8s
```

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None blocking. Coverage is already broad (package var, struct field, function return, conversion, cross-realm, unhashable key). A nil map passed as a function *parameter* is the only common form not exercised, and it reduces to the same nil-`V` path, so not worth adding.

## Suggestions
None.

## Open questions
- The unhashable-key no-op is a permanent silent divergence from gc until gno's map-key hashing gains a recoverable "hash of unhashable type" panic. Not posted: it's a deliberate decision with a concrete revisit trigger, nothing for the author to act on in this PR.
