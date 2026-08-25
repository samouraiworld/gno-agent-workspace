# PR [#5871](https://github.com/gnolang/gno/pull/5871): feat(examples/r/docs): revive and correct r/docs

URL: https://github.com/gnolang/gno/pull/5871
Author: gfanton | Base: master | Files: 131 | +1543 -1200
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: d8aa8541a (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5871 d8aa8541a`
Overview: [visual overview](../overview.html)

Round 2. The head advanced from 7c41a297d to d8aa8541a over nine commits, all of them answers to round 1. The `current_guard` rule was retargeted at secondary realm parameters, five fixtures teaching the redundant first-`cur` check were rewritten, `r/docs/routing` registers the bare wildcard segment, the `complexargs` code span moved to `sanitize.InlineCode`, MiniSocial v2 and the `banker` guard gained tests, the reentrancy and Render-optional claims were reconciled, and `charts` went back to quarantine. Every round 1 finding is resolved and re-verified here. The two Warnings below are new, both in the rewritten detector.

## Overview

Thirty documentation realms sat in `examples/quarantined/`, where they compile and run their tests but are not part of the package set a chain deploys. This change moves them back under `examples/gno.land/r/docs/` with `git mv` and corrects them on the way. The largest correction is one claim stated in five places: the runtime mints the first `cur realm` of a crossing function per frame, so an `IsCurrent()` check on it can never fail, and only a realm value the caller chose is worth testing. Four of the five copies are prose. The fifth runs, and this round rewrites it: `misc/audit-pattern-harness`'s `current_guard` rule now flags a secondary `rlm realm` parameter read before its own `IsCurrent()`, where before it flagged any `.Previous()` at all.

**Verdict: REQUEST CHANGES** — the retargeted rule aims at the parameter a caller can actually forge, and every round 1 finding is fixed, but the rewrite reads a function's parameters only off a line that starts with `func `, so a realm parameter on a func literal is invisible to it and the caller resolver in [`p/demo/tokens/grc20`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L17) went from reported at 7c41a297d to silent at this head (2 Warnings, 1 Missing test, 1 Nit).

## Verify first

- [`misc/audit-pattern-harness/internal/auditpattern/run.go:300`](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/internal/auditpattern/run.go#L300) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/internal/auditpattern/run.go#L300) — confirm the scanner reaches a realm parameter that sits on a closure. Add [`tests/current_guard_gaps_test.go`](tests/current_guard_gaps_test.go) to `internal/auditpattern/` and run `go test ./internal/auditpattern -run TestCurrentGuard` from `misc/audit-pattern-harness`: the same body scores 1 hit as a top-level func and 0 as a func literal.
- [`misc/audit-pattern-harness/internal/auditpattern/run.go:274`](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/internal/auditpattern/run.go#L274) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/internal/auditpattern/run.go#L274) — confirm this list is the set of reads the rule means to catch. The realm type answers eleven methods besides `IsCurrent`, [defined from `uverse.go:1569`](https://github.com/gnolang/gno/blob/d8aa8541a/gnovm/pkg/gnolang/uverse.go#L1569-L1802) · [↗](../../../../../.worktrees/gno-review-5871/gnovm/pkg/gnolang/uverse.go#L1569-L1802) down, and three of them are here.
- [`examples/gno.land/p/nt/mux/v0/router.gno:44`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/p/nt/mux/v0/router.gno#L44) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/nt/mux/v0/router.gno#L44) — confirm the realm-side registration is the whole defence. `reqParts[i]` is still read before the loop breaks on `*`, so any two-segment wildcard pattern panics on a one-segment request, and what keeps `r/docs/routing` off that path is [the bare segment registered beside it](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/routing/routing.gno#L31) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/routing/routing.gno#L31). `grep -rn 'HandleFunc("[^"]*\*' examples/gno.land` returns that one live registration and two in the mux tests.

## Summary

The rewrite of `current_guard` answers round 1 in the right direction: the flagged construct is now one a caller can control. A secondary realm parameter takes whatever the caller passes, and `helper(0, cur.Previous())` reaches the body with `IsCurrent()` false, where a first-position `cur` cannot be reached with anything but the live token. The rule's own [fixture pair](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno#L9-L15) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno#L9-L15) now carries that shape on both sides, and the five other fixtures that opened with the redundant check are clean: `grep -rn "if !cur.IsCurrent()" misc/` returns nothing.

What the rewrite lost is coverage. The old rule scanned every line of a file for `.Previous()`; the new one scans a line only while a parameter list it parsed is in scope, and that list comes from `func ` at the start of a trimmed line. A closure never provides one, so its realm parameter is unguarded for free. The read set narrowed at the same time, from any `.Previous()` to three named accessors, which leaves `String()`, `Sub()`, `Subpath()` and the four `Is*` predicates outside the rule while [the guide](https://github.com/gnolang/gno/blob/d8aa8541a/docs/resources/gno-ai-contract-review.md?plain=1#L29) · [↗](../../../../../.worktrees/gno-review-5871/docs/resources/gno-ai-contract-review.md#L29) tells a reviewer to check every other realm value a function receives.

Reading order: `misc/audit-pattern-harness/internal/auditpattern/run.go`, then its README and the six fixture families it rewrites, then `docs/resources/gno-ai-contract-review.md`, then `routing`, `complexargs`, `security_patterns`, then the three new test files.

## Examples

| Realm value handed to a function | What the caller can pass | `IsCurrent()` | Flagged at d8aa8541a |
|---|---|---|---|
| first parameter `cur realm` | only `cur` or `cross(name)`, the preprocessor refuses the rest | always true | no, correctly |
| secondary `rlm realm`, read with `Previous()` | any realm value, `cur.Previous()` included | caller's choice | yes |
| secondary `rlm realm`, read with `String()`, `Sub()`, `IsUserCall()` | the same | caller's choice | no |
| secondary `rlm realm` on a func literal, any read | the same | caller's choice | no |

## Warnings (should fix)

- **[a realm parameter on a closure is never scanned]** `misc/audit-pattern-harness/internal/auditpattern/run.go:300` — [`guardedRealmParams`](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/internal/auditpattern/run.go#L303) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/internal/auditpattern/run.go#L303) runs only where a trimmed line starts with `func `, so a func literal's realm parameter never enters `guarded` and every read of it passes.
  <details><summary>details</summary>

  A closure is where a realm value most often arrives as a plain parameter, because a non-crossing callback cannot take one in first position. [`CallerTeller`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L16-L21) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L16-L21) returns one that resolves caller identity from `rlm.Previous().Address()`, and the guard it relies on lives in the Teller methods, which is exactly the helper-chain case [the README asks an auditor to check by hand](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/README.md?plain=1#L114-L116) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/README.md#L114-L116). It was in the rule's output at 7c41a297d and is not at this head. The same shape carries the DAO executors in [`z_commondao_execute_0_filetest.gno:33-37`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/p/nt/commondao/v0/filetests/z_commondao_execute_0_filetest.gno#L33-L37) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/nt/commondao/v0/filetests/z_commondao_execute_0_filetest.gno#L33-L37), which read `sub.Address()` with no `IsCurrent()` anywhere in the file.

  Measured with [`tests/current_guard_gaps_test.go`](tests/current_guard_gaps_test.go), which writes one body twice and scans each:

  | Shape | Hits |
  |---|---|
  | `func accountFn(_ int, rlm realm) address` reading `rlm.Previous()` | 1 |
  | the same body as `accountFn: func(_ int, rlm realm) address` | 0 |

  ```
  --- FAIL: TestCurrentGuardScansFuncLiterals (0.01s)
      current_guard_gaps_test.go:55: func literal: expected 1 hit, got 0
  ```

  The README documents two limits on this rule, the same-function scope and the one-line signature, and neither covers this. Fix: match a `func(` literal as well as a `func ` declaration when filling `guarded`, or read the parameter lists with `go/parser`, which also retires the one-line-signature limit.
  </details>

- **[three of eleven realm reads count as a read]** `misc/audit-pattern-harness/internal/auditpattern/run.go:274` — [`realmValueRead`](https://github.com/gnolang/gno/blob/d8aa8541a/misc/audit-pattern-harness/internal/auditpattern/run.go#L273-L280) · [↗](../../../../../.worktrees/gno-review-5871/misc/audit-pattern-harness/internal/auditpattern/run.go#L273-L280) matches `.Previous()`, `.Address()` and `.PkgPath()`, so an unverified realm value read any other way is clean.
  <details><summary>details</summary>

  The realm type answers `String()`, `Sub()`, `Subpath()`, `IsUser()`, `IsUserCall()`, `IsUserRun()`, `IsCode()` and `IsEphemeral()` on top of the three. `Sub()` is the one with teeth: it [mints a sub-realm identity from the value](https://github.com/gnolang/gno/blob/d8aa8541a/gnovm/pkg/gnolang/uverse.go#L1715-L1730) · [↗](../../../../../.worktrees/gno-review-5871/gnovm/pkg/gnolang/uverse.go#L1715-L1730) and the harness's own quarantined hits show a banker built from one. The `Is*` predicates are authorization answers, and [the guide's second quick check](https://github.com/gnolang/gno/blob/d8aa8541a/docs/resources/gno-ai-contract-review.md?plain=1#L52-L60) · [↗](../../../../../.worktrees/gno-review-5871/docs/resources/gno-ai-contract-review.md#L52-L60) is about reading `IsUserCall()` correctly. In the live tree [`r/sys/users/errors.gno:48`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/sys/users/errors.gno#L48) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/sys/users/errors.gno#L48) stores `caller.String()` off a secondary parameter and the rule says nothing.

  ```
  --- FAIL: TestCurrentGuardCoversIdentityAccessors (0.01s)
      current_guard_gaps_test.go:69: rlm.String(): expected 1 hit, got 0
      current_guard_gaps_test.go:69: rlm.Sub("treasury"): expected 1 hit, got 0
      current_guard_gaps_test.go:69: rlm.Subpath(): expected 1 hit, got 0
      current_guard_gaps_test.go:69: rlm.IsUser(): expected 1 hit, got 0
      current_guard_gaps_test.go:69: rlm.IsUserCall(): expected 1 hit, got 0
      current_guard_gaps_test.go:69: rlm.IsUserRun(): expected 1 hit, got 0
      current_guard_gaps_test.go:69: rlm.IsCode(): expected 1 hit, got 0
  ```

  Fix: invert the test and treat any `rlm.` selector other than `IsCurrent()` as a read, which needs no list and does not go stale when the realm type gains a method.
  </details>

## Nits

- **[the file opens on the claim it spent a commit removing]** `examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy.gno:1` — the package doc still reads "explains why Gno is not exposed to Solidity-style reentrancy", while [line 43](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy.gno#L43) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy.gno#L43) says it does not make reentrancy impossible. 6501efbab removed the same claim from three passages further down and from `banker`, and [the new test](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy_test.gno#L14) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/soliditypatterns/reentrancy/reentrancy_test.gno#L14) reads `Render("")`, which is the one string the line is not in. Line 1 is outside the diff's hunks, since the file arrives as an 85% rename, so this goes in the Body rather than an anchor.

## Missing Tests

- **[the backtick that motivated the line is not asserted]** `examples/gno.land/r/docs/complexargs/complexargs.gno:74` — [`z_filetest.gno`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/docs/complexargs/z_filetest.gno#L11) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/complexargs/z_filetest.gno#L11) sets the name to `Bob`, so the golden output is identical under the helper this line replaced.
  <details><summary>details</summary>

  `SetMyObject` takes any string from any account, and the reason `InlineCode` is right here is that its fence outscans backticks in the content. A second filetest beside the existing one covers it, [`tests/z_sanitize_filetest.gno`](tests/z_sanitize_filetest.gno), asserting the two-backtick fence:

  ```
  Value of myObject: ``CustomType{Name: x`y, Numbers: 1}``
  ```

  On the `ufmt.Sprintf` plus `InlineText` line it replaced, the same filetest reports:

  ```
  -Value of myObject: ``CustomType{Name: x`y, Numbers: 1}``
  +Value of myObject: `CustomType{Name: x\`y, Numbers: 1}`
  ```

  Fix: add the filetest. `registry` and `userprofile` took the same swap and have no test file at all, so the one filetest is what pins the helper choice for the branch.
  </details>

## Verified

- The retargeted rule reports 3 hits over `examples/gno.land` and 6 over `examples/quarantined`, matching the author's counts. All three live hits are secondary parameters read before their own `IsCurrent()`: [`r/gov/dao/v3/impl/govdao.gno:43`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L43) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L43) and [`r/sys/names/verifier.gno:196-198`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/r/sys/names/verifier.gno#L196-L198) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/sys/names/verifier.gno#L196-L198).
- The exemption for a first-position `cur` is enforced by the VM, not by convention. A realm-typed first parameter under any other name is refused at preprocess with ``a crossing function's first realm argument must have name `cur` ``, and so is any first argument that is not `cur` or `cross(name)`, including an identifier holding `cur.Previous()`. Deployed as a realm, `helper(0, cur.Previous())` returns `IsCurrent()` false while the same helper called with the frame's own token returns true.
- `r/docs/routing:wildcard` answers. Added as a `TestRoutes` case in `gno.land/pkg/gnoweb/app_test.go` beside the existing `wildcard/foo` case, both pass at this head, where the bare segment failed at 7c41a297d with `slice index out of bounds: 1 (len=1)`.
- MiniSocial v2's new window test fails on the comparison it replaces. Restoring `post.updatedAt.After(post.createdAt.Add(time.Minute * 10))` gives `error mismatch, expected update window expired, got %!s(<nil>) - edit past the window must be refused`.
- Green at d8aa8541a: `gno test` over `r/docs/soliditypatterns/...`, `minisocial/...`, `security_patterns`, `home`, `complexargs`, `routing`; `go test ./...` in `misc/audit-pattern-harness`; `go test ./gno.land/pkg/gnoweb/ -run TestRoutes`; and the `current-guard` slice through `auditpattern -gno-bin`, where both fixtures compile and score their expected hit counts.

## Open questions

- No CI job runs the harness. [`ci-dir-misc.yml`](https://github.com/gnolang/gno/blob/d8aa8541a/.github/workflows/ci-dir-misc.yml#L26-L32) · [↗](../../../../../.worktrees/gno-review-5871/.github/workflows/ci-dir-misc.yml#L26-L32) drives a fixed list of five programs and `audit-pattern-harness` is not one, so neither Warning above reddens anything. Not posted, since nothing about CI belongs in a review comment.
- [`p/samcrew/piechart/README.md:40`](https://github.com/gnolang/gno/blob/d8aa8541a/examples/gno.land/p/samcrew/piechart/README.md?plain=1#L40) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/samcrew/piechart/README.md#L40) links `/r/docs/charts:piechart`, which quarantine does not deploy. Raised in round 1 while the branch deleted `charts` outright; bdda91e22 put it back in quarantine, so the link is now exactly as dead as on master and the branch no longer causes it. It is the only link to a quarantined package left in the live tree, measured over every `.gno` and `.md` under `examples/gno.land`.
