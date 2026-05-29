# PR #5712: feat(tm2/std,gnovm): drop _filetest.gno suffix requirement

URL: https://github.com/gnolang/gno/pull/5712
Author: davd-gzl | Base: master | Files: 457 | +887 -184
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `76cc43e33` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5712 76cc43e33`

**Verdict: REQUEST CHANGES** — the suffix-drop renames create flat-basename collisions (root `X.gno` + `filetests/X.gno` both become `MemFile.Name="X.gno"`) in 5 example packages, and `ValidateMemPackageAny` rejects the resulting package. This breaks `LoadPackagesFromDir`, the exact path `gnoland start` uses to load the examples genesis — the node fails to boot the default examples set. The amino *binary* / on-chain hash claim is solid, but the on-disk rename half of the PR is not safe to merge as-is.

## Summary

The PR replaces filename-substring classification of `MemFile`s with an explicit `MemFile.Kind` enum, and drops the `_filetest.gno` suffix for any `.gno` under a `filetests/` subdir. The Go-side design is clean: `Kind` is amino-skipped, so on-chain `MemPackage` hashes are bit-identical (verified). The problem is the 435-file rename: once the suffix is gone, a filetest whose basename matches a sibling production file collapses to the **same** `MemFile.Name`, because `MemFile.Name` is a flat basename (the `filetests/` dir is a disk-routing convention only). `MemPackage.Uniq()` then rejects the package as having a duplicate filename, and genesis loading aborts. 7 such collisions exist across `mdform`, `message`, `treasury/v0`, `r/docs/moul_md`, and `r/gnoland/home`.

```
on disk (PR):                       in-memory MemPackage (MPUserAll):
  mdform/                             Files:
    mdform.gno          (prod)  ───────► {Name:"mdform.gno", Kind:PackageSource}
    filetests/                          {Name:"mdform.gno", Kind:Filetest}     ← same Name
      mdform.gno        (filetest)─┘     → Uniq() / ValidateMemPackageAny() = "duplicate file name"
```

## Glossary

- **filetest** — standalone `.gno` test run as its own package. Pre-PR: `*_filetest.gno`. Post-PR: any `.gno` under `filetests/`, classified by `MemFile.Kind = KindFiletest`.
- **`MemFile.Name`** — a flat basename (no slashes); enforced by `reFileName` and `ValidateMemPackageAny`. The `filetests/` subdir is **not** part of the name.
- **`MemFileKind`** — new `uint8` enum (`KindUnknown`/`PackageSource`/`Test`/`XTest`/`Filetest`/`Other`). Amino-skipped; JSON/YAML-serialized.
- **`MPUserAll`** — read mode that includes prod, test, and filetest files. Used by `gno test`, lint, and `LoadPackagesFromDir` (genesis).

## Fix

`MemFile` gains a `Kind` field; `(*MemFile).IsFiletest()` prefers `Kind`, falling back to the `_filetest.gno` suffix when `Kind == KindUnknown` ([`memfile.go:155-166`](https://github.com/gnolang/gno/blob/76cc43e33/tm2/pkg/std/memfile.go#L155-L166) · [↗](../../../../../.worktrees/gno-review-5712/tm2/pkg/std/memfile.go#L155-L166)). `ReadMemPackage` stamps `Kind=KindFiletest` on any `.gno` in `filetests/` and drops the old same-name collision guard ([`mempackage.go:788-808`](https://github.com/gnolang/gno/blob/76cc43e33/gnovm/pkg/gnolang/mempackage.go#L788-L808) · [↗](../../../../../.worktrees/gno-review-5712/gnovm/pkg/gnolang/mempackage.go#L788-L808)). `WriteTo`/`Download` route filetests under `filetests/` via `DiskSubdir()`. `FileKind` becomes a type alias of `MemFileKind`, preserving the `gno list -json` map-key shape through `MarshalText`. ~25 classifier call sites switch from suffix-sniffing to `IsFiletest()`/`GetMemFileKind()`.

## Critical (must fix)

- **[node won't boot — examples genesis load fails on duplicate filename]** [`mempackage.go:267-280`](https://github.com/gnolang/gno/blob/76cc43e33/tm2/pkg/std/memfile.go#L267-L280) · [↗](../../../../../.worktrees/gno-review-5712/tm2/pkg/std/memfile.go#L267-L280) — dropping the suffix on a filetest whose basename matches a sibling prod file produces two `MemFile`s with identical `Name`; `Uniq()` → `ValidateMemPackageAny()` rejects the package, and `gnoland start`'s `LoadPackagesFromDir(examples)` aborts.
  <details><summary>details</summary>

  `MemFile.Name` is a flat basename — the `filetests/` subdir is a disk-routing convention, not part of the name ([`memfile.go:115-122`](https://github.com/gnolang/gno/blob/76cc43e33/tm2/pkg/std/memfile.go#L115-L122) · [↗](../../../../../.worktrees/gno-review-5712/tm2/pkg/std/memfile.go#L115-L122)). So `mdform/mdform.gno` (prod) and `mdform/filetests/mdform.gno` (filetest, renamed from `mdform_filetest.gno` in commit `5038f9d2f`) both read into `MemFile{Name:"mdform.gno"}`. `MemPackage.Uniq()` compares lowercased names and returns `duplicate file name "mdform.gno"` ([`memfile.go:266-277`](https://github.com/gnolang/gno/blob/76cc43e33/tm2/pkg/std/memfile.go#L266-L277) · [↗](../../../../../.worktrees/gno-review-5712/tm2/pkg/std/memfile.go#L266-L277)); `ValidateMemPackageAny` calls it indirectly via the read path.

  This is the genesis path: [`genesis.go:193`](https://github.com/gnolang/gno/blob/76cc43e33/gno.land/pkg/gnoland/genesis.go#L193) · [↗](../../../../../.worktrees/gno-review-5712/gno.land/pkg/gnoland/genesis.go#L193) reads each package with `MPUserAll` (filetests included), then [`genesis.go:227`](https://github.com/gnolang/gno/blob/76cc43e33/gno.land/pkg/gnoland/genesis.go#L227) · [↗](../../../../../.worktrees/gno-review-5712/gno.land/pkg/gnoland/genesis.go#L227) calls `ValidateMemPackageAny`. Both `gnoland start` ([`start.go:439`](https://github.com/gnolang/gno/blob/76cc43e33/gno.land/cmd/gnoland/start.go#L439) · [↗](../../../../../.worktrees/gno-review-5712/gno.land/cmd/gnoland/start.go#L439)) and the integration test node ([`node_testing.go:148`](https://github.com/gnolang/gno/blob/76cc43e33/gno.land/pkg/integration/node_testing.go#L148) · [↗](../../../../../.worktrees/gno-review-5712/gno.land/pkg/integration/node_testing.go#L148)) load the whole `examples/` tree this way, so the first collision aborts the load. The same guard fires in the on-chain `MsgAddPackage` handler ([`keeper.go:614`](https://github.com/gnolang/gno/blob/76cc43e33/gno.land/pkg/sdk/vm/keeper.go#L614) · [↗](../../../../../.worktrees/gno-review-5712/gno.land/pkg/sdk/vm/keeper.go#L614)).

  All 7 collisions (none `draft`/ignored), each introduced by this PR's rename — on master the filetests keep the `_filetest.gno` suffix, so the names are distinct:
  - `p/jeronimoalbi/mdform/mdform.gno`
  - `p/jeronimoalbi/message/broker.gno`
  - `p/nt/treasury/v0/{treasury,banker_coins,banker_grc20}.gno`
  - `r/docs/moul_md/moul_md.gno`
  - `r/gnoland/home/home.gno` (the gno.land homepage realm)

  CI did not catch this: only the lightweight bot checks ran on this fork PR (`gh pr checks` shows `process-pr`/`define-prs-matrix` pass; the integration matrix that uses `node_testing.go` did not run).

  Fix: a filetest and a prod file with the same basename are fundamentally indistinguishable in a flat-`Name` MemPackage — the `filetests/` dir separates them on disk but not in `MemFile.Name`. Pick one: (a) don't drop the suffix for files whose stripped basename would collide with a root file (keep those as `_filetest.gno`); (b) exclude filetests from the on-chain/genesis MemPackage entirely (the ADR says filetests "don't ship on-chain anyway" — but `MPUserAll` currently includes them, and that's a behavior change to weigh); or (c) make `Uniq`/`GetFile`/validation key on `(Name, Kind)` so a filetest and a prod file can coexist — but then every `GetFile(name)` caller becomes ambiguous (see warning below). Whichever path, add the regression test in Missing Tests.

  **Repro:**

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5712 -R gnolang/gno
  export GNOROOT=$PWD
  cat > gno.land/pkg/gnoland/zz_repro_test.go <<'EOF'
  package gnoland

  import (
  	"path/filepath"
  	"testing"

  	"github.com/gnolang/gno/tm2/pkg/crypto"
  	"github.com/gnolang/gno/tm2/pkg/std"
  )

  func TestRepro5712(t *testing.T) {
  	creator := crypto.MustAddressFromString("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
  	dir := filepath.Join("..", "..", "..", "examples")
  	txs, err := LoadPackagesFromDir(dir, creator, std.Fee{})
  	t.Logf("loaded %d txs, err = %v", len(txs), err)
  	if err != nil {
  		t.Fatalf("examples genesis load failed: %v", err)
  	}
  }
  EOF
  go test ./gno.land/pkg/gnoland/ -run TestRepro5712 -v 2>&1 | grep -E "loaded|FAIL|duplicate|ok "
  rm gno.land/pkg/gnoland/zz_repro_test.go
  ```

  ```
  zz_repro_test.go:18: loaded 0 txs, err = unable to load package ".../examples/gno.land/p/jeronimoalbi/message": invalid package: duplicate file name "broker.gno"
  --- FAIL: TestRepro5712 (0.31s)
  FAIL
  ```

  On master the same test passes (filetests retain `_filetest.gno`, so basenames are unique).
  </details>

## Warnings (should fix)

- **[`GetFile(name)` silently shadows the filetest with the prod file]** [`memfile.go:286-294`](https://github.com/gnolang/gno/blob/76cc43e33/tm2/pkg/std/memfile.go#L286-L294) · [↗](../../../../../.worktrees/gno-review-5712/tm2/pkg/std/memfile.go#L286-L294) — `GetFile` returns the first `Name` match. For a colliding package it always returns the prod file, never the filetest. Lint reads the per-filetest body via `mfile := mpkg.GetFile(fname)` ([`lint.go:325`](https://github.com/gnolang/gno/blob/76cc43e33/gnovm/cmd/gno/lint.go#L325) · [↗](../../../../../.worktrees/gno-review-5712/gnovm/cmd/gno/lint.go#L325)), so directive parsing (`parsePkgPathDirective`, `filetestExpectsFailure`) would run against the wrong (prod) body. `gno lint ./examples/.../mdform` exits 0 today because lint reads with `MPAnyAll` and never calls `Uniq`, but the body it lints for that filetest is the prod file's, not the filetest's. Any fix that lets the two coexist (critical option (c)) makes this latent shadowing a live bug; a fix that keeps names unique (option (a)) avoids it.

- **[removed collision guard was the early-warning that #5704 review asked to keep]** [`mempackage.go:799-807`](https://github.com/gnolang/gno/blob/76cc43e33/gnovm/pkg/gnolang/mempackage.go#L799-L807) · [↗](../../../../../.worktrees/gno-review-5712/gnovm/pkg/gnolang/mempackage.go#L799-L807) — `ReadMemPackage` previously errored `cannot add %q in filetests: same filename in package dir` at read time; the same guard was removed from `examplespkgfetcher` ([`examplespkgfetcher.go:70-95`](https://github.com/gnolang/gno/blob/76cc43e33/gnovm/pkg/packages/pkgdownload/examplespkgfetcher/examplespkgfetcher.go#L70-L95) · [↗](../../../../../.worktrees/gno-review-5712/gnovm/pkg/packages/pkgdownload/examplespkgfetcher/examplespkgfetcher.go#L70-L95)). The prior review on #5704 ([`5704-lint-filetest-isolation/2-050597de7`](../../5704-lint-filetest-isolation/2-050597de7/claude-opus-4-7_davd-gzl.md), "stale filetest at package root collides on next read") explicitly flagged this collision risk and asked to keep or relocate a guard. Removing it entirely (per commit `021e02dd5`, to unblock `mdform`/`treasury`) is what lets the duplicate reach `ValidateMemPackageAny` instead of failing with a clear, located message. Whatever the critical-fix direction, restore a guard that fails fast at read time with the file path.

- **[`Kind` leaks into amino-JSON for every prod file, not just filetests]** [`memfile.go:62-66`](https://github.com/gnolang/gno/blob/76cc43e33/tm2/pkg/std/memfile.go#L62-L66) · [↗](../../../../../.worktrees/gno-review-5712/tm2/pkg/std/memfile.go#L62-L66) — the ADR frames `Kind` as a filetest concern and notes it's "serialized to JSON," but `omitempty` only drops `KindUnknown` (0). A freshly-read prod file is `KindPackageSource` (1), so amino-JSON now emits `"kind":"PackageSource"` on it.
  <details><summary>details</summary>

  Verified: `amino.MarshalJSON(&MemFile{Name:"foo.gno", Body:"...", Kind:KindPackageSource})` → `{"name":"foo.gno","body":"...","kind":"PackageSource"}`. amino *binary* is byte-identical with and without `Kind` (on-chain hashes stable — the load-bearing claim holds), but genesis docs are amino-JSON. So `gnoland genesis ... add packages` output now carries a `kind` field on every prod file. Existing committed genesis fixtures decode fine (absent `kind` → `KindUnknown`, re-omitted), so no golden break — but freshly generated genesis.json changes shape. Deterministic across nodes, so not consensus-splitting; still, the ADR should state that `Kind` appears in JSON transit for all classified files, not only filetests. Low impact, but undocumented.
  </details>

## Nits

- [`mempackage.go:1214`](https://github.com/gnolang/gno/blob/76cc43e33/gnovm/pkg/gnolang/mempackage.go#L1214) · [↗](../../../../../.worktrees/gno-review-5712/gnovm/pkg/gnolang/mempackage.go#L1214) — `WriteToMemPackage` now recovers `MemFile.Name` via `strings.TrimPrefix(fpath, mpkg.Path+"/")` instead of `filepath.Base`. Under the flat-`Name` invariant the two are equivalent (parser filenames are `path.Join(mpkg.Path, file.Name)`), but `TrimPrefix` silently returns the full path if `mpkg.Path` isn't an exact prefix (e.g. empty path), where `filepath.Base` cannot. Leftover from the abandoned prefix-in-`Name` approach; `filepath.Base` is the more robust choice now that `Name` is always flat.

## Missing Tests

- **[no test reads a package with root `X.gno` + `filetests/X.gno`]** [`mempackage_test.go:44-53`](https://github.com/gnolang/gno/blob/76cc43e33/gnovm/pkg/gnolang/mempackage_test.go#L44-L53) · [↗](../../../../../.worktrees/gno-review-5712/gnovm/pkg/gnolang/mempackage_test.go#L44-L53) — the existing `"duplicate"` case constructs two literal same-name `MemFile`s by hand; nothing exercises the realistic `ReadMemPackage(dir)` → `filetests/` path producing a colliding bare `Name`. A test that lays down `pkg/x.gno` + `pkg/filetests/x.gno` on disk, reads with `MPUserAll`, and asserts the intended behavior (clean error, or both coexisting) would have caught the critical and locks in whichever fix is chosen.
- **[no `LoadPackagesFromDir` / genesis round-trip over a filetest-bearing package]** — `gno.land/pkg/gnoland/genesis_test.go` covers synthetic temp dirs only. A test loading a real examples package that has both a root file and a same-named filetest (e.g. `mdform`) would assert the genesis path stays green.

## Suggestions

- Consider whether filetests belong in the genesis/on-chain MemPackage at all. The ADR's own rationale ("filetests are a development/test artifact and don't ship on-chain") suggests `LoadPackagesFromDir` could read with a filetest-excluding mode rather than `MPUserAll` — which would both dodge this collision class and shrink genesis. Larger change; worth a separate discussion, not a blocker on its own.

## Questions for Author

- Was the integration matrix (the jobs that spin up a node via `node_testing.go`) actually run on this PR? If not, the examples-load break would be invisible until merge — worth forcing a full CI run before re-review.
- For the 5 colliding packages: is the intent that a filetest may share a basename with a prod file (requiring `Kind`-aware `Uniq`/`GetFile`), or should the suffix-drop simply skip files that would collide? The two fixes have very different blast radii.
