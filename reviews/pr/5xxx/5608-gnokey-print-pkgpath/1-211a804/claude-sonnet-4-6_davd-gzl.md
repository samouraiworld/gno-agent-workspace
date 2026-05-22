# PR #5608: feat(gnokey): print pkgpath after maketx addpkg

**URL:** https://github.com/gnolang/gno/pull/5608
**Author:** davd-gzl | **Base:** master | **Files:** 5 | **+37 -0**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

After a successful `gnokey maketx addpkg --broadcast` (or `gnokey broadcast` of a pre-signed addpkg tx), this PR prints the deployed package path on its own line:

```
PKGPATH:    gno.land/r/<addr>/mypkg
```

The implementation adds 6 lines to `PrintTxInfo` in `gno.land/pkg/keyscli/root.go`: it iterates `tx.Msgs`, type-asserts each to `vm.MsgAddPackage`, and if matched prints `PKGPATH:    <path>`. This correctly handles multi-message transactions (emits one line per `MsgAddPackage` in tx order), works via any entry point that calls `PrintTxInfo` (`maketx addpkg --broadcast`, `gnokey broadcast`), and is silent for `call`/`run`/`send` transactions.

The PR went through a refactoring journey (5 commits): it started by hooking into `addpkg.go`'s `OnTxSuccess` callback, then moved the logic into `PrintTxInfo` for consistency. The final design is clean and follows the existing callback pattern.

**Files changed:**
- `gno.land/pkg/keyscli/root.go` — core logic in `PrintTxInfo`
- `gno.land/pkg/integration/testdata/addpkg.txtar` — adds PKGPATH assertion to existing single-msg test
- `gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar` — new txtar test for multi-msg broadcast
- `docs/builders/deploy-packages.md` — adds PKGPATH to example output
- `docs/users/interact-with-gnokey.md` — adds PKGPATH to example output and explanation

## Test Results

- **Existing tests:** PASS — `go test ./gno.land/pkg/keyscli/...` passes (4 tests, 0.124s)
- **Integration txtar tests:** Not run locally (require gnoland binary); covered by CI
- **Edge-case tests:** skipped (change is small and well-constrained)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gno.land/pkg/keyscli/root.go:88` — `addPkg.Package` is a pointer (`*std.MemPackage`) and is accessed without a nil check. In practice this is safe: `PrintTxInfo` is only invoked on a committed transaction, and `ValidateBasic` (called in the ABCI ante handler at `tm2/pkg/sdk/baseapp.go:597`) would have already panicked on a nil `Package`. However, when `gnokey broadcast` reads a tx from a user-supplied file (`broadcast.go:66-71`), there is no call to `ValidateBasic` before `PrintTxInfo` — a hand-crafted JSON file with `"package": null` would cause a nil dereference panic in `PrintTxInfo` after a successful (but unusual) broadcast response. Consider adding `if addPkg.Package != nil` as a guard.

- [ ] `docs/users/interact-with-gnokey.md:217-225` — The example output block omits the `INFO:` line that `PrintTxInfo` actually emits between `TX HASH` and `PKGPATH`. The actual output is:
  ```
  TX HASH:    ...
  INFO:       ...
  PKGPATH:    ...
  ```
  The rendered docs will mislead users about the exact field order. Either add `INFO:` to the example or note that it may be empty.

## Nits

- [ ] `gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar:1-3` — The file comment says the test verifies that `PrintTxInfo` derives PKGPATH from `tx.Msgs`, but the test also implicitly verifies that `gnokey broadcast` (not just `maketx addpkg`) invokes `PrintTxInfo` via `OnTxSuccess`. The comment is accurate but a minor extra note about this would improve readability.

- [ ] `gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar:12` — `-quiet=false` is passed to `gnokey broadcast`. This is a root-level `BaseCfg` flag (defined in `tm2/pkg/crypto/keys/client/root.go:74-79`), not a `BroadcastCfg` flag, so it is valid. However it is the default value (`false`), so the flag is redundant and can be dropped.

- [ ] `gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar:23-24` — The hardcoded creator address `g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5` matches `DefaultAccount_Address` (the `test1` key), but a reader must cross-reference `node_testing.go` to verify this. A brief inline comment would aid future maintainers.

## Missing Tests

- [ ] No unit test for `PrintTxInfo` itself. The PKGPATH output path is covered only by txtar integration tests which require a running node. A table-driven unit test in `root_test.go` (or a new `root_test.go`) exercising: (1) tx with one `MsgAddPackage`, (2) tx with two `MsgAddPackage` msgs, (3) tx with only `MsgCall` (no PKGPATH output expected) would provide fast, reliable regression coverage independent of integration infrastructure.

- [ ] The `addpkg_multi_msg.txtar` test does not assert the absence of PKGPATH in the `gnokey sign` step (sign-only, no broadcast), which would confirm that PKGPATH is not printed in the non-broadcast path. Low priority since the logic is guarded by `if cfg.RootCfg.Broadcast`, but worth noting.

## Suggestions

- The `addpkg.go` `OnTxSuccess` callback is now identical to the one set in `root.go:38-39` (both call `PrintTxInfo(tx, res, io)`). The per-command callback override was needed when PKGPATH was printed only in `addpkg`, but now that it lives in `PrintTxInfo` the override is redundant — any caller that goes through the root command's `NewRootCmd` already has the right callback set. Consider removing the per-command `OnTxSuccess` override in `addpkg.go:143-145` (and similarly in `call.go:150-152` and `run.go:160-162`) and relying solely on the root-level callback. This would reduce duplication and ensure consistent behavior if `PrintTxInfo` is updated again in the future.

## Questions for Author

- Is there a scenario where `Package.Path` could legitimately be empty for a committed `MsgAddPackage`? `ValidateBasic` rejects empty paths, but could a tx pass consensus with an empty path (e.g., through a genesis state or migration)?
- Should PKGPATH also be printed when `gnokey maketx addpkg` is used without `--broadcast` (i.e., printed alongside the JSON output)? This would allow users to see the path even before broadcasting.

## Verdict

APPROVE — The implementation is correct, well-scoped, and the refactor into `PrintTxInfo` is the right design. The nil-pointer warning for `Package` and the missing `INFO:` line in docs are worth addressing but are not blockers.
