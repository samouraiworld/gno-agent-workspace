# PR #5563: feat(gnodev): add gnodev version command

**URL:** https://github.com/gnolang/gno/pull/5563
**Author:** AmozPay | **Base:** master | **Files:** 4 | **+8 -4**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR adds a `gnodev version` subcommand for consistency with `gno version` and `gnokey version`. It modifies four files:

1. **`tm2/pkg/crypto/keys/client/version.go`** — Removes the `*BaseCfg` parameter from `NewVersionCmd`, changing its signature from `NewVersionCmd(rootCfg *BaseCfg, io commands.IO)` to `NewVersionCmd(io commands.IO)`. The `rootCfg` was never used inside the function, so this is a clean simplification.

2. **`gno.land/pkg/keyscli/root.go`** — Updates the `gnokey` call site to match the new signature: `client.NewVersionCmd(io)` instead of `client.NewVersionCmd(cfg, io)`.

3. **`contribs/gnodev/main.go`** — Imports `tm2/pkg/crypto/keys/client` and adds `client.NewVersionCmd(stdio)` as a subcommand of `gnodev`.

4. **`contribs/gnodev/Makefile`** — Adds a `VERSION` variable (same formula as `gno.land/Makefile` and `gnovm/Makefile`) and injects it via `-ldflags` into `tm2/pkg/version.Version` at build time.

The PR addresses issue #5550, which requests `gnodev version` for parity with the other CLI tools.

## Test Results

- **Existing tests:** FAIL — `tm2/pkg/crypto/keys/client/version_test.go:23` calls `NewVersionCmd(nil, io)` with the old two-argument signature and does not compile. The test file was not updated to match the signature change.
- **Edge-case tests:** skipped

## Critical (must fix)

- [ ] `tm2/pkg/crypto/keys/client/version_test.go:23` — `NewVersionCmd(nil, io)` uses the old two-argument signature. The test does not compile. Must be updated to `NewVersionCmd(io)`.

## Warnings (should fix)

- [ ] `tm2/pkg/crypto/keys/client/version.go:20` — Hardcodes `"gnokey version:"` in the output. When invoked from `gnodev version`, it prints `gnokey version: <hash>` instead of `gnodev version: <hash>`, which is confusing and defeats the purpose of the feature. The `gno` binary has its own `newGnoVersionCmd` in `gnovm/cmd/gno/version.go` that prints `"gno version:"` — gnodev should similarly have a self-contained command printing `"gnodev version:"`, or the shared command should accept a binary name parameter.
- [ ] `contribs/gnodev/main.go:12` — Imports `tm2/pkg/crypto/keys/client` solely for the version command. This couples gnodev to the keys client package unnecessarily. A lightweight local version command (like `gnovm/cmd/gno/version.go`) would avoid the dependency and naturally produce the correct output string.

## Nits

- [ ] `contribs/gnodev/Makefile:2-3` — Two blank lines between `VERSION` and `GNOROOT_DIR`; the rest of the Makefile uses single blank lines.
- [ ] `contribs/gnodev/main.go:66-68` — The blank line between `cmd.AddSubCommands(localcmd)` and `cmd.AddSubCommands(NewStagingCmd(stdio))` was removed, but the original had one. Minor style inconsistency.

## Missing Tests

- [ ] No test verifying that `gnodev version` actually works (e.g. a build + run smoke test in CI or a unit test in the gnodev package).

## Suggestions

- Replace the `client.NewVersionCmd` reuse with a simple local function in `contribs/gnodev/main.go` that prints `"gnodev version:" + version.Version`. This is what `gnovm/cmd/gno/version.go` does — 15 lines, zero external dependencies beyond `tm2/pkg/version` and `tm2/pkg/commands`. It solves both the wrong output string and the unnecessary import at once.
- If the intent is to share the version command across binaries, refactor it to accept a binary name: `NewVersionCmd(name string, io commands.IO)`.

## Questions for Author

- Was the "gnokey version:" output string when running `gnodev version` intentional, or was this an oversight?
- Why import the `keys/client` package rather than writing a self-contained version command like `gno` does?

## Verdict

REQUEST CHANGES — The test file doesn't compile (critical), and the version output says "gnokey" when run from gnodev, which is the opposite of what issue #5550 asks for.
