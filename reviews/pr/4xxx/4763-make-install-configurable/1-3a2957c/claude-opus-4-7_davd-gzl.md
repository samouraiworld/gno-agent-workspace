# PR #4763: feat: make misc/install.sh configurable

URL: https://github.com/gnolang/gno/pull/4763
Author: aeddi | Base: master | Files: 2 | +348 -54
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `3a2957c` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4763 3a2957c`

**Verdict: NEEDS DISCUSSION** — superseded by merged PR #5492, which replaced `misc/install.sh` wholesale with a binary-downloading installer; before reviewing the diff on its own merits, decide whether to close, rebase against the new script (mostly a rewrite), or salvage just the per-component selection idea as a small follow-up to #5492.

## Summary

Adds `--gno / --gnokey / --gnodev / --gnobro / --uninstall` flags to the source-build `misc/install.sh`, splits the monolithic `make install` call into per-tool `make install.<tool>` calls, renames `GNO_DIR` env var to `GNOROOT` (aligning with the gno binary's env), switches uninstall to read `go env GOBIN` (falling back to `GOPATH/bin`), and adds a 213-line GitHub Actions workflow that exercises every flag combination plus reinstall and custom-GNOROOT scenarios. Default behavior (no flags = all four tools) is preserved.

The blocker is not the code: master commit [`68cb7b898` (#5492, May 13 2026)](https://github.com/gnolang/gno/pull/5492) rewrote `misc/install.sh` end-to-end (139 → 440 lines, `#!/bin/bash` → `#!/bin/sh`, source-build → binary download with checksum verification, `--from-source` fallback, `--full` to add `gnoland`, `--help`, `--version`, `--dir`). The PR's last commit `3a2957c33` (Oct 1 2025) predates that rewrite by seven months, and the PR has been stale-bot flagged since Feb 2026.

## Glossary

- `GNOROOT` — env var the gno binary uses to locate its source tree (`gnovm/pkg/gnoenv/gnoroot.go:42`). This PR aligns the install script with that name (was `GNO_DIR`).
- `install.<tool>` — per-tool Makefile targets (`install.gno`, `install.gnokey`, `install.gnodev`, `install.gnobro`) defined in [`Makefile:31-47`](https://github.com/gnolang/gno/blob/3a2957c/Makefile#L31-L47) · [↗](../../../../../.worktrees/gno-review-4763/Makefile#L31-L47). The aggregate `install` target only covers the first three.

## Fix

Before this PR, [`misc/install.sh`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh) called `make install` (gno+gnokey+gnodev) then `make install.gnobro` unconditionally, and uninstalled by hardcoding `$(go env GOPATH)/bin/<tool>`. After, `parse_args` flips per-tool booleans from CLI flags (defaulting all-true when no args), each `install.<tool>` target is gated on its boolean, and uninstall consults `go env GOBIN` first via a new `get_go_bin` helper that handles the empty-default case Go itself doesn't fix ([issue #34522](https://github.com/golang/go/issues/34522)). The load-bearing trick is [`parse_args:97-98`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L97-L98) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L97-L98): if any arg is supplied, all four install flags reset to `false` first, then are re-enabled per matched flag — so `--gno` means "only gno", not "gno plus the defaults".

## Critical (must fix)

- **[obsoleted by master]** `misc/install.sh:1-228` — entire script was replaced by merged PR #5492 seven months after this PR's last commit; the diff applies cleanly only because the worktree's `origin/master` ref is stale, not because the change is still mergeable.
  <details><summary>details</summary>

  The PR's HEAD is [`3a2957c33`](https://github.com/gnolang/gno/pull/4763/commits/3a2957c336b078f0c58699efc733cd1c80655696) (2025-10-01, last master merge). On 2026-05-13, [`68cb7b898` (#5492)](https://github.com/gnolang/gno/commit/68cb7b898fda10bf95d558086e90b0317dbf5bf5) landed a complete rewrite of `misc/install.sh`: `#!/bin/sh` (POSIX), 440 lines, binary download from GitHub Releases as the default mode, checksum verification, `--from-source` fallback that does `make install install.gnobro` + `install.gnoweb` in one go, optional `--full` adding `gnoland`, `--help`, `--version <tag>`, `--dir <path>`, plus stack-based xtrace suppression so a `GITHUB_TOKEN` doesn't leak. Stale-bot has already flagged it (2026-02-16). A rebase will hit a near-total conflict — there is essentially no overlap between this PR's `parse_args` and the new installer's flag set. Fix: pick one of (a) close this PR; (b) reopen as a small follow-up to #5492 that adds per-component selection to either `--from-source` (which currently always builds all four/five tools) or the binary-download mode (which currently always installs the full `COMPONENTS` list); (c) confirm with the author that the source-build script is still wanted alongside the new binary installer, and decide where it should live.
  </details>

## Warnings (should fix)

- **[flag combo silently ignored]** [`misc/install.sh:200-215`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L200-L215) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L200-L215) — `--uninstall --gno` removes all four binaries, not just `gno`; per-tool flags are silently ignored in the uninstall path.
  <details><summary>details</summary>

  `parse_args` accepts `--gno`/`--gnokey`/etc. and `--uninstall` from the same flag namespace, but `uninstall_gno()` does four unconditional `rm -f` calls and an `rm -rf "$gnoroot"` regardless of which install booleans were set. A user who types `bash install.sh --uninstall --gnobro` reasonably expects "uninstall gnobro only" — they get a full wipe including the source tree. The CI workflow doesn't exercise this combo so it would never surface there. Fix: either reject mixing `--uninstall` with per-tool flags (`error "--uninstall cannot be combined with per-tool flags"; exit 1`), or honor them and only `rm` the binaries whose booleans are true. The first option is simpler and keeps the contract narrow.
  </details>

- **[`local var=$(cmd)` masks failures]** [`misc/install.sh:62-63`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L62-L63) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L62-L63), [`misc/install.sh:84`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L84) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L84), [`misc/install.sh:128`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L128) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L128), [`misc/install.sh:201-202`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L201-L202) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L201-L202) — `local foo=$(cmd)` always returns 0 (the `local` builtin's exit status), defeating `set -e` for the inner command.
  <details><summary>details</summary>

  Standard bash gotcha: `set -e` aborts on a failing simple command, but `local x=$(failing_cmd)` is one command whose status comes from `local`, not from `$()`. In practice the affected calls (`go env GOBIN`, `go env GOPATH`, `go version`, the helper functions) are unlikely to fail in interesting ways, but `get_gno_root` and `get_go_bin` themselves call `exit 1` internally so they do the right thing on the failure paths that matter. The risk is mostly future maintenance — someone adds a failing branch to one of those helpers and the caller silently continues with an empty string. Fix: split declaration from assignment — `local gobin; gobin=$(go env GOBIN 2>/dev/null || echo '')` — so the assignment's status is what `set -e` sees.
  </details>

- **[reset-all-then-set idiom is non-obvious]** [`misc/install.sh:97-98`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L97-L98) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L97-L98) — `read -r INSTALL_GNO INSTALL_GNOKEY INSTALL_GNODEV INSTALL_GNOBRO <<< 'false false false false'` is a clever one-liner that obscures intent and breaks if anyone adds a fifth tool without updating both the variable list and the literal.
  <details><summary>details</summary>

  The contract is "if any flag is supplied, all install defaults reset to false, then each matched flag re-enables one tool". The `read <<<` form does that in one line but reads like a serialization trick. Four explicit `INSTALL_GNO=false; INSTALL_GNOKEY=false; ...` lines say the same thing and are grep-friendly. More importantly, the literal `'false false false false'` is positionally coupled to the variable list — drop or reorder a name and the wrong defaults land silently. Fix: replace with explicit assignments.
  </details>

- **[uninstall fallback doesn't match Go's actual default]** [`misc/install.sh:65-72`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L65-L72) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L65-L72) — when both `GOBIN` and `GOPATH` env are unset, `go env GOPATH` still returns the *default* (`$HOME/go`), so the fallback is mostly fine, but the `error "Could not determine Go binary installation path"` branch only fires if `go env GOPATH` itself errors — which is essentially unreachable.
  <details><summary>details</summary>

  Not a correctness bug, but the error branch is dead code: `go env GOPATH` returns a non-empty default even with no env vars set. The branch can only be reached if `go` is missing, but `check_go` already exited before `uninstall_gno` runs. Fix: either drop the dead branch with a comment, or document that the helper assumes a working `go` binary (the call site does).
  </details>

- **[uninstall removes binaries it never installed]** [`misc/install.sh:205-208`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L205-L208) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L205-L208) — same issue as today: `uninstall_gno` `rm -f`s all four binaries unconditionally, which can clobber binaries the user installed by hand (`go install` from a clone, distro package, brew). Not introduced by this PR but worth flagging since the PR otherwise advertises per-tool granularity.
  <details><summary>details</summary>

  `rm -f` is silent on missing files so there's no user-visible failure, but a user who installed `gnokey` via `go install` and `gnodev` via this script then runs `--uninstall` loses both. The wider point: the source-build installer doesn't track what it installed, so it can only do a "remove everything that matches these names" sweep. If per-tool uninstall lands (see the flag-combo warning above), this gets worse — `--uninstall --gnokey` would silently delete a hand-installed gnokey. Out of scope to fix here, but PR #5492's binary-download path has the same property and is worth thinking about together.
  </details>

## Nits

- [`.github/workflows/install.yml:1-9`](https://github.com/gnolang/gno/blob/3a2957c/.github/workflows/install.yml#L1-L9) · [↗](../../../../../.worktrees/gno-review-4763/.github/workflows/install.yml#L1-L9) — no path filter (`paths: [misc/install.sh, .github/workflows/install.yml]`), so every PR triggers ~8 minutes of CI runtime on jobs that can only fail when this file changes.
- [`.github/workflows/install.yml:181-213`](https://github.com/gnolang/gno/blob/3a2957c/.github/workflows/install.yml#L181-L213) · [↗](../../../../../.worktrees/gno-review-4763/.github/workflows/install.yml#L181-L213) — `test-mixed-flags` installs but never uninstalls, so it doesn't catch uninstall regressions for mixed flag installs. Cheap to add.
- [`misc/install.sh:69-72`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L69-L72) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L69-L72) — indentation of the `error`/`exit` lines uses 2 spaces inside a 4-space-indented block; pre-existing in [`install.sh:131-133`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L131-L133) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L131-L133), so consistent with the file but not consistent within itself.
- [`misc/install.sh:191`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L191) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L191) — `"${failed_tools[*]}"` joins with the first char of IFS (a space by default). Works, but `(IFS=,; echo "${failed_tools[*]}")` or a printf loop reads more obviously.

## Missing Tests

- **[uninstall+per-tool combo]** [`.github/workflows/install.yml:42-86`](https://github.com/gnolang/gno/blob/3a2957c/.github/workflows/install.yml#L42-L86) · [↗](../../../../../.worktrees/gno-review-4763/.github/workflows/install.yml#L42-L86) — no test for `--uninstall --gno` (or any per-tool + uninstall combo). The Warning above predicts a user-surprising behavior that this would catch.
  <details><summary>details</summary>

  Add a job that does: install all → `--uninstall --gno` → assert only `gno` is missing and `gnokey/gnodev/gnobro` remain. Once that test exists, the maintainer has to choose: make it pass (honor per-tool flags in uninstall) or make it fail loudly (reject the combo in `parse_args`). Either way the silent-wipe footgun goes away.
  </details>

- **[reinstall over existing install]** [`.github/workflows/install.yml:87-146`](https://github.com/gnolang/gno/blob/3a2957c/.github/workflows/install.yml#L87-L146) · [↗](../../../../../.worktrees/gno-review-4763/.github/workflows/install.yml#L87-L146) — `test-all-tools` does install → uninstall → install, but doesn't test install → install (no uninstall between). @moul's review comment asked for "install, uninstall, install" specifically; the second install needs to validate that `git fetch --depth 1 && git reset --hard origin/master` on an existing checkout still works ([`install.sh:139-148`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L139-L148) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L139-L148)).
  <details><summary>details</summary>

  The intent of @moul's [comment](https://github.com/gnolang/gno/pull/4763#issuecomment-3302444994) was to catch Go-dep/cache breakage on a second build, which `install → uninstall (removes source) → install` doesn't really exercise — the second install starts from a clean clone. Add `install → install` and `install → install --gnodev` paths.
  </details>

- **[GOBIN explicitly set]** [`misc/install.sh:62-72`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L62-L72) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L62-L72) — no CI job sets `GOBIN=/some/path` and verifies uninstall reads from there. `test-custom-gno-dir` tests GNOROOT but not GOBIN, even though both env vars were added/touched by this PR.
  <details><summary>details</summary>

  Trivial to add: set `GOBIN=/tmp/custom-bin` in the job env, install, verify binaries land in `/tmp/custom-bin`, uninstall, verify they're gone. Covers the path through `get_go_bin` that the codecov badge claims is covered but actually isn't reachable from any matrix dimension.
  </details>

## Suggestions

- [`misc/install.sh:91-124`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L91-L124) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L91-L124) — consider rejecting `--uninstall` mixed with per-tool flags up-front; cleaner than the current "per-tool flags silently overridden" behavior.
  <details><summary>details</summary>

  Couple of lines in `parse_args`: after the loop, if `UNINSTALL=true` and any per-tool flag was passed, `error "--uninstall takes no other flags"; exit 1`. Aligns with the script's existing "unknown flag = exit 1" strictness.
  </details>

- [`misc/install.sh:4`](https://github.com/gnolang/gno/blob/3a2957c/misc/install.sh#L4) · [↗](../../../../../.worktrees/gno-review-4763/misc/install.sh#L4) — usage line shows tool flags but not `--uninstall` in the main usage; line 6 has a separate uninstall line. Either follow @moul's [suggestion](https://github.com/gnolang/gno/pull/4763#discussion_r2355105393) to fold uninstall into the main usage, or leave the two-line split — but pick one consistently. Today line 4 advertises four flags and line 6 advertises a fifth, which is the worst of both worlds.

## Questions for Author

- Given #5492 already landed, is the intent here to (a) close, (b) port the per-component selection idea on top of the new installer, or (c) keep the source-build script as a separate `misc/install-from-source.sh` and turn this PR into that file?
- `parse_args` resets all install flags to false on any arg — is `--uninstall` alone meant to be a "reset and uninstall everything" idiom, or were the per-tool flags expected to be honored by uninstall? The current behavior is the first; the docs don't say.
- The Makefile aggregate `install` target builds gnokey+gno+gnodev (no gnobro) — should the script's "default install" really build all four, or should it match `make install`? Default behavior changes either way; today it's "all four", which is a small behavior change vs. the previous script that called `make install` (3 tools) then `make install.gnobro` (4 tools as a best-effort that only printed a warn on failure). Worth being explicit.
