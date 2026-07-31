# PR [#6025](https://github.com/gnolang/gno/pull/6025): fix(gno.land): type-check production files only at AddPackage

URL: https://github.com/gnolang/gno/pull/6025
Author: omarsy | Base: master | Files: 6 | +484 -92
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 0b83500c6 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-6025 0b83500c6`

**TL;DR:** When you deploy a package to gno.land, the chain was also checking the package's test files for errors. Doing that made the chain read files off the individual node operator's hard drive, so two nodes running the same transaction could charge different fees or disagree on whether it succeeded. This PR stops the chain from checking test files. It still stores them and still rejects a package if any file is unreadable garbage.

**Verdict: APPROVE** — the fix is correct and the determinism property is verified end to end; the consensus break is real and disclosed, so it needs an upgrade boundary before it ships (2 Missing Tests, 2 Nits, 1 Suggestion).

## Verify first

- [`gnovm/pkg/gnolang/gotypecheck.go:521`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L521) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L521) — everything rests on the production pass covering exactly the file set the VM runs. Confirm by reading [`filterTests`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L725-L736) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L725-L736) against [`MPFProd`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/mempackage.go#L354-L355) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/mempackage.go#L354-L355): both reduce to the same two filename suffixes, so no file can reach the VM un-type-checked.
- Deploy gas for a package carrying a `_test.gno` drops 86% and a historical out-of-gas `AddPackage` would now succeed. Confirm an upgrade boundary or relaunch is scheduled before merge; run `go test ./gno.land/pkg/integration/ -run 'TestTestdata/addpkg_testfile_restart_gas$'` on master to see the old price.

## Summary

`AddPackage` type-checked the `_test.gno` and `_filetest.gno` files it stores. Resolving a test file's stdlib imports went through [`TypeCheckOptions.TestGetter`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L147-L149) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L147-L149), which merged the chain-stored stdlib with an overlay read at runtime from `$GNOROOT/gnovm/tests/stdlibs/` — node-local filesystem state inside a gas-metered consensus computation, memoised in a process-lifetime map. On the merge base the same deploy is priced 15,671,980 or 20,595,400 gas depending only on whether the node restarted between transactions, a 4,921,074 swing against a 20,000,000 block limit. That forked topaz-1 at block 301381 ([#6024](https://github.com/gnolang/gno/issues/6024)).

The fix adds [`ProdOnly`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L163-L181) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L163-L181), which stops `typeCheckMemPackage` after the production pass, and sets it at [`AddPackage`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/sdk/vm/keeper.go#L643) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/sdk/vm/keeper.go#L643), [`Run`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/sdk/vm/keeper.go#L1041) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/sdk/vm/keeper.go#L1041), and the three stdlib load sites. With no remaining path to the overlay, `testStdlibCache` and its mutex are deleted.

## Diagram

The four type-check passes, and which of them can reach the operator's filesystem:

```
typeCheckMemPackage(mpkg, wtests)
  |
  +-- GoParseMemPackage        parses EVERY .gno file        <- unchanged, load-bearing
  |     gofs  = prod + same-package _test.gno
  |     _gofs = xxx_test package files
  |     tgofs = _filetest.gno
  |
  +-- PASS 1  cfg.Check(pgofs = filterTests(gofs))   gimp.testing = false  -> getter
  |                                                                          (chain store)
  |   ====== ProdOnly returns here (gotypecheck.go:521) ======
  |
  +-- PASS 2  cfg.Check(gofs)          w/ tests      gimp.testing = TRUE  -> tgetter ---+
  +-- PASS 3  cfg.Check(_gofs2)        xxx_test      gimp.testing = TRUE  -> tgetter ---+--> $GNOROOT
  +-- PASS 4  cfg.Check(tgofs2)        _filetest     gimp.testing = TRUE  -> tgetter ---+    overlay
```

`tgetter` is read only when `gimp.testing` is true, and `gimp.testing` is set only inside passes 2-4. Sub-imports inherit `wtests = gimp.testing && ...`, so no nested call can flip it back on. Cutting at pass 1 removes the filesystem edge structurally rather than guarding it.

## Fix

Before, `AddPackage` passed `wtests = nil` to [`typeCheckMemPackage`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L441) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L441), meaning run every pass. After, `ProdOnly` maps to a pointer-to-false, which the pre-existing early return at [`gotypecheck.go:521`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L521-L527) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L521-L527) already understood as stop after production. The load-bearing constraint is that parsing stays complete: [`pkgData.parseFile`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/doc/pkg.go#L124-L133) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/doc/pkg.go#L124-L133) returns an error for an unparseable `_test.gno`, so an implementation that filtered the mempackage instead would let one deploy and then break `vm/qdoc` for that package permanently, since the bytes are immutable in state.

## Benchmarks / Numbers

Two identically-shaped packages, each carrying a `_test.gno` that imports `testing`. Merge base d1a33f574.

| Arrangement | `bar` | `baz` | Spread |
|---|---|---|---|
| master, node restart between | 20,593,054 | 20,595,400 | 2,346 |
| master, no restart | 20,593,054 | 15,671,980 | 4,921,074 |
| PR, node restart between | 2,862,220 | 2,862,220 | 0 |
| PR, no restart | 2,862,220 | 2,862,220 | 0 |

Same transaction shape, 4,921,074 gas of spread on master decided by node process history alone, against a 20,000,000 topaz block limit. Deploy cost falls 86.1% (20,593,054 → 2,862,220).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[assertion cannot fail for the wrong reason]** [`gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar:31`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L31) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L31) — the `baz` case asserts `type check failed`, which any type error in the package satisfies, while its siblings at [lines 39](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L39) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L39) and [42](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L42) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/integration/testdata/addpkg_testfile_typecheck.txtar#L42) name the exact symbol.
  <details><summary>details</summary>

  The case exists to prove a broken production file is still rejected. A generic message match also passes if the rejection came from somewhere else in the package, so the case would stay green if the production pass stopped running. Fix: assert `undefinedInProd`, matching the precision of the `aaa` and `bbb` cases.
  </details>

- **[comment wrapping]** [`gno.land/pkg/sdk/vm/keeper.go:529`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/sdk/vm/keeper.go#L529) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/sdk/vm/keeper.go#L529) — the line runs to 108 columns where the rest of the block wraps near 75. Not posted, no change needed: [`.github/golangci.yml`](https://github.com/gnolang/gno/blob/0b83500c6/.github/golangci.yml?plain=1#L12-L33) · [↗](../../../../../.worktrees/gno-review-6025/.github/golangci.yml#L12-L33) runs `default: none` with no `lll` in the enable list, so no linter enforces this and it changes no meaning.

## Missing Tests

- **[a gnovm option with no gnovm test]** [`gnovm/pkg/gnolang/gotypecheck.go:181`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L181) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L181) — `ProdOnly` is defined in `gnovm/pkg/gnolang` but every test that exercises it lives in `gno.land/pkg/integration`.
  <details><summary>details</summary>

  [`gotypecheck_test.go:400-403`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck_test.go#L400-L403) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck_test.go#L400-L403) builds one shared `TypeCheckOptions` for the whole table and never sets the new field, so `go test ./gnovm/...` covers none of it. A contributor changing the pass ordering in `typeCheckMemPackage` gets no signal from the package's own suite; the failure surfaces two modules away in a txtar. Fix: add the four-case test shipped at [`tests/prodonly_x_test.go`](tests/prodonly_x_test.go), which pins that a broken test file is rejected without `ProdOnly` and accepted with it, that production still cannot reach a test-only symbol, and that an unparseable test file is still rejected. Passes at 0b83500c6.
  </details>

- **[tripwire nobody trips]** [`gnovm/pkg/gnolang/gotypecheck.go:118`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L118-L120) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L118-L120) — the nil-getter panic has no test, so nothing detects if a later refactor makes it unreachable or turns it back into a nil dereference.
  <details><summary>details</summary>

  The branch exists so that dropping `ProdOnly` from a keeper call site fails with a named cause instead of an opaque internal error. It is currently dead by construction: both `AddPackage` and `Run` pass a non-nil `Getter`, and the nil-getter `tgetter` is gated behind `gimp.testing`, which `ProdOnly` prevents. Fix: a test calling `TypeCheckMemPackage` with a package that imports a stdlib, `TestGetter` nil and `ProdOnly` false, asserting the panic message.
  </details>

## Suggestions

- **[the warning is not where the next contributor looks]** [`gnovm/pkg/gnolang/gotypecheck.go:147-149`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L147-L149) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L147-L149) — the `TestGetter` doc says what the field does but not that setting it on a consensus path reintroduces the filesystem dependency this PR removes.
  <details><summary>details</summary>

  Someone wiring a new keeper call site reads the field doc, not the panic string at [line 118](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L118) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L118), which they only see once they have already made the mistake and run the code. Fix: one clause on the field saying it must stay nil on consensus paths.
  </details>

## Verified

- On the merge base d1a33f574 the same `AddPackage` transaction shape costs 15,671,980 or 20,595,400 gas depending only on whether `gnoland restart` ran earlier in the process; on 0b83500c6 it costs 2,862,220 in both arrangements. Copying the PR's new [`addpkg_testfile_restart_gas.txtar`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/integration/testdata/addpkg_testfile_restart_gas.txtar) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/integration/testdata/addpkg_testfile_restart_gas.txtar) onto the merge base makes it fail; the 4,921,074 spread is the bug it pins. [repro](comment_claude-opus-5.md)
- The production pass type-checks exactly the set the VM runs: [`filterTests`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/gotypecheck.go#L725-L736) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/gotypecheck.go#L725-L736) drops `_test.gno` from `gofs`, `GoParseMemPackage` already routed `_filetest.gno` to `tgofs`, and [`MPFProd`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/mempackage.go#L354-L355) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/mempackage.go#L354-L355) filters the same two suffixes.
- A `_test.gno` declaring a package name that disagrees with the production files is still rejected, by mempackage validation rather than the type-check, so the loosened type-check opens no gap there. Deploying one fails with `expected package name "mm" but got "totallydifferent"` at 0b83500c6.
- The three stdlib call sites are unaffected in practice: [`defaultStore.GetMemPackage`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/gnolang/store.go#L1107-L1109) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/gnolang/store.go#L1107-L1109) returns the production blob, so `ProdOnly` changes no verdict there. It does remove a latent hazard: with `wtests == nil` the with-tests pass reassigns the returned `*types.Package`, which [`Initialize`](https://github.com/gnolang/gno/blob/0b83500c6/gno.land/pkg/sdk/vm/keeper.go#L177-L181) · [↗](../../../../../.worktrees/gno-review-6025/gno.land/pkg/sdk/vm/keeper.go#L177-L181) writes into the shared type-check cache.
- `go test -race ./gno.land/pkg/sdk/vm/` is clean at 0b83500c6, after the deletion of `testStdlibCache` and its `sync.RWMutex`. CI runs no race build, so nothing else covers the concurrency side of removing that cache.
- Both tests shipped under [`tests/`](tests/prodonly_x_test.go) run green at 0b83500c6: the four `ProdOnly` cases, and the nil-getter panic case embedded in [`comment_claude-opus-5.md`](comment_claude-opus-5.md), which confirms the new branch panics with the exact message it claims rather than being unreachable from every caller.
- Green at 0b83500c6: `TestTestdata/{addpkg_testfile_typecheck,addpkg_testfile_restart_gas,issue_2763,addpkg_import_testdep_gas,restart_gas}`, `go test ./gno.land/pkg/sdk/vm/`, `go test ./gnovm/pkg/gnolang/ -run 'TypeCheck|Import|GoParse'`, `go vet`, `gofmt`. The pre-existing exact-gas pins in `addpkg_import_testdep_gas.txtar` and `restart_gas.txtar` still hold, because the packages they measure carry no test files.

## Open questions

- `vm/qdoc` parses stored `_test.gno` files into [`pkgData.testFiles`](https://github.com/gnolang/gno/blob/0b83500c6/gnovm/pkg/doc/pkg.go#L131) · [↗](../../../../../.worktrees/gno-review-6025/gnovm/pkg/doc/pkg.go#L131) and never reads the slice. It can only fail on them, never surface them. Not posted: dead-store cleanup unrelated to this diff.
- A `_test.gno` importing a package that does not exist now deploys successfully; before, the xxx_test pass rejected it. This is the intended loosening rather than a defect, since the chain cannot run the file, but it means a deployed package can carry a test file nobody can ever type-check. Not posted: the PR already names client-side `gno lint` at deploy as the follow-up.
