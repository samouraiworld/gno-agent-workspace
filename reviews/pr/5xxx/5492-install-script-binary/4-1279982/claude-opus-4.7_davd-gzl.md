# PR #5492: feat(misc): rewrite install.sh as precompiled binary downloader

**URL:** https://github.com/gnolang/gno/pull/5492
**Author:** notJoon | **Base:** master | **Files:** 2 | **+346 -113**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.7

## Summary

Fourth review round (commit `1279982`). Since round 3 (`8ba469d`) the author pushed several cleanups and a structural split:

1. **`uninstall.sh` split out** (`bf2936c`) — A dedicated `misc/uninstall.sh` (70 lines) now owns the uninstall flow. `install.sh` no longer accepts `--uninstall`; the help text points users at `misc/uninstall.sh`. The new file uses `FULL_COMPONENTS` for both the install-dir cleanup and the legacy `$GOPATH/bin` cleanup — this closes the round-3 Warning about `gnoweb` being omitted from the legacy cleanup.
2. **Zero-components failure path** (`0fae7a5`) — `install_gno` now tracks `installed_count` and calls `die "no expected binaries found..."` when nothing was installed, instead of silently printing the success banner. Closes round-3 Warning #2.
3. **`resolve_asset` reuses `$CURL`** (`3ea2675`) — The helper no longer hardcodes its own `curl --proto =https ...` flags; both branches (with/without token) start from `$CURL` and add `-H Accept:`, `-o /dev/null`, `-w '%{redirect_url}'`. The intentional `-L` omission is now documented in a block comment. Closes round-3 Warning #1.
4. **DRY xtrace suspension** (`0fae7a5`) — New `suspend_xtrace`/`restore_xtrace` helpers use a stack string (`_xt_stack`) with leftmost push/pop so nested suspensions compose correctly. Replaces the inline `case "$-" in *x*) ...` stanzas.
5. **Rate-limit hint** (`0f026a2`) — `api_get` now captures response headers to `$TMP/api_headers` via `-D` and, on curl failure with no token, greps for `x-ratelimit-remaining: 0` and logs a hint to set `GITHUB_TOKEN`. Closes the round-3 Suggestion.
6. **Comment pruning** (`84c1f9f`) — WHAT-only comments dropped; the remaining comments explain non-obvious decisions (why `-L` is omitted from `resolve_asset`, why `/releases/latest` is avoided, etc.).

`install.sh` is a 310-line POSIX-sh script; `uninstall.sh` is a 70-line companion. Both pass `sh -n` syntax check. No behavioural change to the happy path vs round 3; the feature surface is the same (`--version`, `--dir`, `--full`, `--help`, `GNO_VERSION`, `GNO_INSTALL_DIR`, optional `GITHUB_TOKEN`).

## Test Results
- **Existing tests:** N/A — standalone shell scripts, no automated test suite
- **CI:** All checks pass
- **Syntax:** `sh -n` clean on both files
- **Edge-case tests:** skipped (no release with matching binaries available to exercise end-to-end on this branch; `v0.1.0` can still be used for a smoke test per @jeronimoalbi's round-3 note)

## Critical (must fix)

None.

## Warnings (should fix)

- [ ] `misc/install.sh:216` — **`per_page=30` still not bumped.** Same issue flagged in rounds 2 and 3. With the repo churning out `chain/*` tags, a future burst of 30+ non-`v*` releases between `v*` tags will make `latest` resolution fail. Bumping to `per_page=100` (GitHub's max) is a one-char change and eliminates the risk without adding pagination complexity.

- [ ] `misc/uninstall.sh:50-56` — **Legacy `$GOPATH/bin` cleanup is skipped when `go` is not on PATH.** The block is guarded by `command -v go`, so users who installed gno via the old source-build flow and have since removed Go will leave stale binaries in `$GOPATH/bin`. Consider falling back to `${GOPATH:-$HOME/go}/bin` when `go` is unavailable (still guarded by `[ -d ]` so it's a no-op when the directory doesn't exist).

- [ ] `misc/install.sh:114-118` — **Rate-limit hint only fires on curl non-zero exit.** `-f` makes curl fail on HTTP ≥ 400, so 403 rate-limit responses do trigger the hint. However, if GitHub ever returns 200 with an empty body on exhaustion (they don't today, but the behaviour is undocumented) the hint path would be skipped. Minor, but the check could additionally run when the response is empty/unparseable. Not blocking.

## Nits

- [ ] `misc/install.sh:78-80` — **Stray blank line inside the `sha256sum`/`shasum` check** (between `else die ...` and `fi`). Flagged in rounds 1, 2, and 3; still present.
- [ ] `misc/install.sh:10-11` — `FULL_COMPONENTS="gno gnokey gnodev gnobro gnoweb gnoland"` still duplicates the `COMPONENTS` list. `FULL_COMPONENTS="$COMPONENTS gnoland"` avoids divergence. Flagged in rounds 2 and 3.
- [ ] `misc/install.sh:287-299` — The `gnodev`/gno.land references in the getting-started banner assume `gnodev` was installed. If the archive didn't contain `gnodev` (v0.1.0 case), the banner tells the user to run a binary that isn't on disk. A conditional banner is overkill; a one-line caveat ("if gnodev was installed in this run") could help, or simply keep docs-link entries and drop the `gnodev` bullet.
- [ ] `misc/uninstall.sh:50` — `gobin="$(go env GOPATH 2>/dev/null)/bin"` will yield `"/bin"` if GOPATH env lookup fails or returns empty; the `!= "/bin"` guard catches that but the logic would read more clearly as `gopath=$(go env GOPATH); [ -n "$gopath" ] && gobin="$gopath/bin"`.

## Missing Tests

- [ ] Still no `shellcheck` CI step. Flagged in all three prior rounds. A minimal `shellcheck -s sh misc/install.sh misc/uninstall.sh` job on changes to `misc/*.sh` is low-cost and would have caught the round-1/2 regressions automatically. Acceptable as a follow-up per the author's response.

## Suggestions

- Consider a `--components` or `-c "gno,gnokey"` flag for users who want a subset (e.g. CI only needs `gnokey`). The current `--full` toggle is binary; a user-specified list scales better. Follow-up OK. — `misc/install.sh:46-56`
- For macOS, the per-binary `xattr -d com.apple.quarantine` runs only on installed binaries. A single best-effort `xattr -cr "$INSTALL_DIR" 2>/dev/null || true` after the install loop is cheaper and also clears the directory-level quarantine that `curl | sh` may leave. Carried from round 3. — `misc/install.sh:274`
- `uninstall.sh` could print the files it is about to remove (as @jeronimoalbi suggested). A loop emitting `log "removing $INSTALL_DIR/$c"` before `rm -f` adds four lines and makes destructive actions auditable, without needing a `--dry-run` flag. — `misc/uninstall.sh:46-55`

## Questions for Author

- Was there a reason to drop `--uninstall` from `install.sh` entirely instead of keeping a thin shim that calls out to `uninstall.sh`? Users who have `install.sh` bookmarked/curled will now need to learn the new URL. A one-line hint in the `die` path for `--uninstall` (e.g. "use misc/uninstall.sh") would smooth the transition.
- `gnodev` and `gnobro` remain in the default `COMPONENTS`. With the round-3 warning path now firing on missing binaries, the first-time user still sees "warning: expected binaries missing" on any `--version v0.1.0` install. Is the plan to cut a new tagged release soon that includes all five, so the warning becomes a transient state, or should the default list be pruned to match what actually ships?

## Verdict

APPROVE — All round-3 Critical/Warning items are addressed (installed_count guard, FULL_COMPONENTS in legacy cleanup, `$CURL` reuse in `resolve_asset`, `GITHUB_TOKEN` rate-limit hint). The remaining findings are low-risk maintenance items (`per_page=30`, banner assumes gnodev, blank-line nit, FULL_COMPONENTS duplication) that can land in follow-ups. The uninstall split is a clean separation. Good iteration.
