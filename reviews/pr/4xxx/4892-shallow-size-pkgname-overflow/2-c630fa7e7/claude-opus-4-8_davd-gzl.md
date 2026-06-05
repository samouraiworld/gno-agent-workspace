# PR #4892: fix(gnovm): include missing field in shallow size calculation + add overflow protection

URL: https://github.com/gnolang/gno/pull/4892
Author: davd-gzl | Base: master | Files: 41 | +382 -66
Reviewed by: davd-gzl | Model: claude-opus-4-8 (self-review — flag for second human reviewer) | Commit: `c630fa7e7` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4892 c630fa7e7`

Round 2 of [round 1](../1-5a0ec86/claude-opus-4-7_davd-gzl.md) (verdict REQUEST CHANGES). The PR was rebased onto current master since 5a0ec86; this round evaluates whether the round-1 blockers were resolved.

**Verdict: APPROVE** — every round-1 blocker is resolved: the PR was rebased so the constant-drift conflict is gone (it no longer touches the `_alloc*Value` block, inheriting master's `unsafe.Sizeof`-anchored values), `AllocatePackageValue` is deleted, the `alloc_11a.gno` "should not happen" panic assertion and the `slice_alloc.gno` boundary flip are both gone (only golden-value bumps remain), and the ADR was rewritten to state overflow protection is retained and to document the `PreprocessFiles` lint-path asymmetry. The net code change is small, symmetric, and CI-green on Go 1.25.9. One author-thread question (necessity of `overflow.*` in `GetShallowSize`, cc'd to @thehowl) is unanswered but non-blocking. A second human reviewer should confirm before merge given the self-review.

## Summary

Fixes [issue #4791](https://github.com/gnolang/gno/issues/4791): `PackageValue.GetShallowSize()` returned bare `allocPackage`, ignoring `PkgName`, `PkgPath`, and `FNames`/file-block metadata, so GC-recount and store-load over-reported relative to what creation charged. The fix introduces `packageValueSize(pkgName, pkgPath, fnames)` as the single source of truth, called from both the allocation paths (`NewPackageValue`, the non-main branch of `NewPackage`, and `AddFileBlock` for incremental file blocks) and from `GetShallowSize`, guaranteeing alloc == recount. Gas goldens move up slightly (string-content + per-file-block charges). Round 2 changes vs round 1: rebase onto master, plus two cleanup commits that de-dup the string-size helper (`allocStringSize`) and stop double-counting the 16-byte string header for `PkgName`/`PkgPath` (those headers already live inside `sizeof(PackageValue)`).

```
creation path                                  read-back path
─────────────                                  ──────────────
NewPackageValue:  Allocate(pkgValueSize(n,p,nil))   GetShallowSize() = pkgValueSize(n,p,FNames)
                + AllocateBlock(numNames)            store load: charges GetShallowSize() once
AddFileBlock x k: Allocate(fileBlockEntrySize)       GC recount: re-sums GetShallowSize()
─────────────                                  ──────────────
Σ = pkgValueSize(n,p,FNames) + block      ==   pkgValueSize(n,p,FNames) + block   ✓ symmetric
```

## Glossary

- `packageValueSize(name, path, fnames)` — single source of truth for a `PackageValue`'s shallow cost; `allocPackage` + backing bytes of the two strings + one `fileBlockEntrySize` per fname.
- `fileBlockEntrySize(fname)` — incremental cost of one file block: `allocStringSize(len)` (FNames slot header + bytes) + 16 (FBlocks interface slot) + 16 (map key header) + 8 (map value `*Block`).
- `allocStringSize(n)` — full standalone string cost: `allocString` (16-byte header + heap overhead) + `n` content bytes.
- `fallbackAllocator` — master's shared non-charging allocator (`NewAllocator(math.MaxInt64)`, no gas meter); used by lint/import-only paths where master removed the old nil-`*Allocator` no-op.

## Round-1 findings: disposition

| Round-1 finding | Severity | Status in c630fa7e7 |
|---|---|---|
| Premise superseded by master / `_allocPackageValue` constant drift | Critical | Resolved — rebased; PR no longer edits the constant block, inherits master's `_allocPackageValue = 296` |
| `alloc_11a.gno` asserts "should not happen" GC panic | Critical | Resolved — file no longer modified by the PR |
| `slice_alloc.gno` flipped success→failure at threshold | Warning | Resolved — item count unchanged; only the gas golden bumps `70970781 → 70970844` |
| `AllocatePackageValue` dead code | Warning | Resolved — function deleted; only the ADR references it historically |
| `PreprocessFiles` lint-path asymmetry undocumented | Warning | Resolved — now passes `fallbackAllocator` with inline comment + dedicated ADR section |
| ADR claims overflow "removed" but code retains it | Warning | Resolved — ADR section rewritten to "Overflow protection retained in GetShallowSize" |
| `getFBlocksMap()` `// XXX, pass in allocator` | Warning | Resolved — comment replaced with rationale (entries counted via `packageValueSize(FNames)`) |
| No alloc-vs-recount round-trip test | Missing test | Partial — `TestPackageValueGetShallowSize` covers the arithmetic incl. FNames/file blocks; no end-to-end store round-trip test |
| `nodes.go` non-main branch duplicates `NewPackageValue` body | Suggestion | Open (carried forward, low priority) |

## Fix

Before: `PackageValue.GetShallowSize()` returned `allocPackage` alone; `AddFileBlock` charged nothing; non-main `NewPackage` charged nothing for the package value. After: `packageValueSize` is computed identically on both the charge and the read-back sides. `AddFileBlock(alloc, fname, fb)` charges `fileBlockEntrySize` per append; the lint path passes `fallbackAllocator` so nothing is charged for the throwaway `pv`. See [`alloc.go:671-695`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/alloc.go#L671-L695) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L671-L695).

The load-bearing constraint: at creation FNames is empty and each file block is charged separately, so `Σ charged = packageValueSize(name, path, FNames)`, which is exactly what `GetShallowSize` returns on store-load ([`store.go:546`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/store.go#L546) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/store.go#L546)) and GC recount. The header-vs-backing-bytes split matters: `PkgName`/`PkgPath` add only `_allocHeap + bytes` (their 16-byte headers are already inside `sizeof(PackageValue)`), while each fname in `fileBlockEntrySize` adds a full `allocStringSize` because those headers live in the slice/map backing arrays, not the struct ([`alloc.go:660-686`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/alloc.go#L660-L686) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L660-L686)).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/alloc.go:101`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/alloc.go#L101) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L101) — the net diff deletes a blank line in the estimated-constants block. Cosmetic; harmless.
- [`gnovm/pkg/gnolang/alloc_test.go:39`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/alloc_test.go#L39) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc_test.go#L39) — `TestAllocConstantsMatchActualSizes` overlaps with master's `TestAllocConstSizes` ([`alloc.go:160`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/alloc.go#L160) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc.go#L160)). Two checks now assert the same `unsafe.Sizeof` parity; consolidate to one. Also drop the `✓`/emoji in the `t.Logf` per repo style.

## Missing Tests

- **[no end-to-end store round-trip test for the symmetry invariant]** [`gnovm/pkg/gnolang/alloc_test.go`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/alloc_test.go) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/alloc_test.go) — `TestPackageValueGetShallowSize` validates the `packageValueSize` arithmetic but not that `Σ(charged at creation) == GetShallowSize() post-load/recount`.
  <details><summary>details</summary>

  The new unit test confirms `GetShallowSize` returns the formula, and the txtar gas goldens implicitly cover the creation charge. What is still unguarded is a direct assertion that constructing a package + adding file blocks charges exactly what a later GC recount re-sums. A regression test that snapshots `alloc.bytes` after `NewPackageValue` + N `AddFileBlock`, forces `GarbageCollect`, and asserts the recount equals the snapshot would catch a future field added to `PackageValue` and forgotten in `packageValueSize`. Non-blocking: the invariant is currently enforced structurally (one shared function on both sides), so the gap is decay-prevention, not a live bug.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/nodes.go:1353-1376`](https://github.com/gnolang/gno/blob/c630fa7e7/gnovm/pkg/gnolang/nodes.go#L1353-L1376) · [↗](../../../../../.worktrees/gno-review-4892/gnovm/pkg/gnolang/nodes.go#L1353-L1376) — the non-main branch repeats `NewPackageValue`'s `Allocate(packageValueSize(...)) + AllocateBlock(...)` shape inline. The branches diverge on PkgID stamping and Realm setup, so full consolidation is non-trivial and matches master's existing structure; carried forward as low priority from round 1.

## Questions for Author

- The `overflow.Addp/Mulp` wrapping in `GetShallowSize`/`packageValueSize` answers [@ltzmaxwell's thread](https://github.com/gnolang/gno/pull/4892#discussion_r) (cc'd to @thehowl) with the rationale "size is computed before `Allocate`'s `maxBytes` check, and `loadObjectSafe` adds the size before allocating, so a silent wrap could mis-report." That rationale is now in the ADR. @thehowl never replied on the thread — worth a ping to close it before merge, since it is the one open design question.

## Verification

CI: all 84 checks pass on Go 1.25.9 (2 skips, 0 failures). Locally I ran `go test ./gnovm/pkg/gnolang/ -run TestAlloc` (ok), `go test ./gno.land/pkg/sdk/vm/ -run Gas` (ok), and `go test ./gno.land/pkg/integration/ -run 'TestTestdata/(gc|gnokey_gasfee|restart_gas)'` (ok).

One local-only failure: `TestFiles/alloc_0.gno` expects `bytes:7266`, got `7874`. This is a Go-version artifact, not a PR bug — the `bytes:` golden derives from `unsafe.Sizeof`, and on Go 1.26.3 clean master also fails `alloc_0.gno` (expected 6918, got 7526, the same +608 delta). CI runs Go 1.25.9 where the PR's goldens are correct. The whole alloc filetest suite is brittle across Go versions for this reason — the same class of cross-version golden fragility round 1 flagged on `slice_alloc.gno`.

```bash
# from a local clone of gnolang/gno (Go 1.25.x):
gh pr checkout 4892 -R gnolang/gno
go test ./gnovm/pkg/gnolang/ -run 'TestAlloc'
go test ./gno.land/pkg/sdk/vm/ -run Gas
go test ./gno.land/pkg/integration/ -run 'TestTestdata/(gc|gnokey_gasfee|restart_gas)'
```

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang
ok  	github.com/gnolang/gno/gno.land/pkg/sdk/vm
ok  	github.com/gnolang/gno/gno.land/pkg/integration
```

---

Self-review caveat: produced under the GitHub account of the PR author. Treat as a sanity check, not approval. A second human reviewer (@ltzmaxwell, who left the original threads, or @thehowl, cc'd on the overflow question) should confirm before merge.
