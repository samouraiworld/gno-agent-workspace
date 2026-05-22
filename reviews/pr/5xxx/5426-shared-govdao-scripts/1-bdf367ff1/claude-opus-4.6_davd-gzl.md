# PR #5426: refactor(govdao): shared scripts with deployment wrapper

**URL:** https://github.com/gnolang/gno/pull/5426
**Author:** moul | **Base:** master | **Files:** 12 | **+165 -66**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR refactors the govdao operational shell scripts from a deployment-specific location (`misc/deployments/gnoland1/govdao-scripts/`) into a shared location (`misc/govdao-scripts/`) that can be reused across multiple deployments (gnoland1 and test12). The key changes are:

1. **Moved 8 shell scripts** (`add-validator.sh`, `add-validator-from-valopers.sh`, `rm-validator.sh`, `extend-govdao-t1.sh`, `restrict-account.sh`, `unrestrict-account.sh`, `set-cla.sh`, `set-valoper-minfee.sh`) from `misc/deployments/gnoland1/govdao-scripts/` to `misc/govdao-scripts/`.

2. **Modified scripts to use environment variables** instead of hardcoded values. Each script now uses `${VAR:?error}` patterns for required variables (e.g., `GNOKEY_NAME`, `REMOTE`, `CHAIN_ID`), making them deployment-agnostic.

3. **Added two thin wrapper scripts** (`misc/deployments/gnoland1/govdao` and `misc/deployments/test12.gno.land/govdao`) that set deployment-specific environment variables and delegate to the shared scripts. Wrappers provide usage help, colorized output, and a `list` command.

4. **Added a shared README** at `misc/govdao-scripts/README.md` documenting the available scripts.

The design separates deployment configuration (wrapper) from operational logic (shared scripts), following DRY principles. Scripts generate Gno source code via heredocs and pipe them to `gnokey maketx run` for on-chain governance proposals.

## Test Results

- **Existing tests:** N/A — this PR is purely shell script refactoring with no Go or Gno test files. All CI checks pass.
- **Edge-case tests:** skipped — shell scripts for operational tooling; no test harness available.

## Critical (must fix)

- [ ] `misc/deployments/gnoland1/govdao_prop1.gno:45` — Stale reference to `govdao-scripts/add-validator.sh`. This file uses `// govdao-scripts/add-validator.sh $ADDR $PUB_KEY $POWER` as a usage comment. After this PR, the script has moved to `../../govdao-scripts/add-validator.sh` (or is invoked via the `./govdao` wrapper). The relative path no longer resolves from the deployment directory. While this is a comment, it serves as documentation for operators — incorrect paths will cause confusion.

- [ ] `misc/deployments/gnoland1/gen-genesis.sh:28` — Stale reference to `govdao-scripts/extend-govdao-t1.sh`. Line 28 sources `govdao-scripts/extend-govdao-t1.sh` with a relative path: `source govdao-scripts/extend-govdao-t1.sh`. After this PR, the directory `govdao-scripts/` no longer exists under `misc/deployments/gnoland1/`. This will break the genesis generation script at runtime.

## Warnings (should fix)

- [ ] `misc/govdao-scripts/add-validator-from-valopers.sh:15` — Comment says `# GAS_WANTED: Gas wanted for the transaction (default: 10000000)` but the actual default on line 22 is `GAS_WANTED="${GAS_WANTED:-50000000}"` (50M, not 10M). This 5x discrepancy will mislead operators relying on the comment.

- [ ] `misc/deployments/gnoland1/govdao:21-48` vs `misc/deployments/test12.gno.land/govdao:21-47` — The two wrapper scripts are ~95% identical. Lines 21-48 (the `usage()` function, argument parsing, `list` command, script dispatch, and execution logic) are duplicated verbatim. Only 3 environment variable values and the display name differ (lines 5-13). If a new script is added or the wrapper logic changes, both files must be updated in sync. Consider extracting the common wrapper logic into a shared file sourced by both, with only the env var block deployment-specific.

## Nits

- [ ] `misc/govdao-scripts/README.md:1-14` — The README lists scripts and their arguments but doesn't mention the wrapper scripts or how to invoke them. Since the shared scripts now require environment variables that are set by the wrappers, the README should reference the wrapper entry point (e.g., `misc/deployments/gnoland1/govdao <script> [args]`) as the primary invocation method.

- [ ] `misc/deployments/test12.gno.land/govdao:1` — The shebang is `#!/bin/bash` which is correct, but the file is named just `govdao` without a `.sh` extension. This is intentional (it acts as a command), but it's inconsistent with the shared scripts which all use `.sh` extensions. Minor style point — no action required if this is deliberate.

## Missing Tests

- [ ] No automated tests exist for any of the govdao scripts (before or after this PR). Given that these scripts generate Gno source code dynamically and execute governance proposals, a dry-run or syntax-check test (e.g., `bash -n` on each script, or a mock `gnokey` that captures generated code) would catch heredoc syntax errors and missing variable issues. This is a pre-existing gap, not introduced by this PR.

## Suggestions

- **Extract shared wrapper logic:** The duplicated wrapper boilerplate could be refactored into a `misc/govdao-scripts/wrapper-common.sh` that both deployment wrappers source. Each deployment wrapper would then only need to set its 3-4 env vars and call the common logic. This would prevent drift and make adding new deployments trivial. Reference: `misc/deployments/gnoland1/govdao:21-48` and `misc/deployments/test12.gno.land/govdao:21-47`.

- **Update stale callers as part of this PR:** Since the move is the primary change, updating the 2 stale references (`govdao_prop1.gno:45` and `gen-genesis.sh:28`) in the same PR would prevent a broken state on master. The `gen-genesis.sh` reference is particularly important as it's executable code, not just a comment.

## Questions for Author

- Is `misc/deployments/gnoland1/gen-genesis.sh:28` still actively used? If so, the `source govdao-scripts/extend-govdao-t1.sh` path needs updating since the directory was removed. Was this an intentional omission or an oversight?

- For the wrapper scripts: is there a plan to add more deployments beyond gnoland1 and test12? If so, extracting the shared wrapper logic now would save effort later.

## Verdict

REQUEST CHANGES — Two stale file references will break `gen-genesis.sh` at runtime and mislead operators via `govdao_prop1.gno`. These should be fixed in this PR to avoid shipping a broken state on master.
