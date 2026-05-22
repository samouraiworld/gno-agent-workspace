# PR #5492: feat(misc): rewrite install.sh as precompiled binary downloader

**URL:** https://github.com/gnolang/gno/pull/5492
**Author:** notJoon | **Base:** master | **Files:** 1 | **+288 -115**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

Third review round (commit `8ba469d`). Since the second review (commit `af43a93`), the author addressed all previously open issues:

1. **Missing binary warning added** — The install loop now tracks missing components in a `$missing` variable and emits `log "warning: expected binaries missing from $ARCHIVE:${missing}"` at the end. This directly addresses the round-2 Critical.
2. **`GITHUB_TOKEN` support added** — New functions `api_get()`, `resolve_asset()`, and `capture_github_token()` handle authenticated API requests. `GITHUB_TOKEN` is moved to a non-exported `GH_API_TOKEN` and unset from the environment to prevent leaking to child processes. Xtrace (`set -x`) is temporarily disabled around sensitive operations. Addresses the round-2 Warning and @jeronimoalbi's comment.
3. **`latest_v_tag` awk fallback now filters prereleases** — The awk parser now pairs each `tag_name` with the subsequent `prerelease` flag within the same object and skips prerelease entries. Addresses the round-2 Warning.
4. **`resolve_asset()` for CDN URL resolution** — Asset API URLs are first resolved to their signed CDN redirect URLs (without forwarding auth headers), then downloaded from the CDN. This is a security best practice that avoids forwarding `Authorization` headers across host boundaries.
5. **Error messages improved** — The "no v* release found" error now includes a link to the releases page.
6. **`uninstall_gno` uses `$FULL_COMPONENTS`** — Instead of a hardcoded list, the uninstall loop iterates `$FULL_COMPONENTS`.

The PR replaces `misc/install.sh` with a POSIX-sh (`#!/bin/sh`) script that downloads precompiled binaries from GitHub Releases, removing the Go and git dependencies. It detects host OS/arch, verifies SHA256 checksums, supports `--version`/`--dir`/`--full`/`--uninstall` flags, and handles the repo's unusual release structure (`chain/*` tags without binaries vs `v*` tags with goreleaser artifacts).

Part of issue #5459 (UX-1: Setup is too slow and fragile).

## Test Results
- **Existing tests:** N/A — standalone shell script, no automated test suite
- **CI:** All checks pass (build, lint, e2e-test, etc.)
- **Edge-case tests:** skipped

## Critical (must fix)

None. The round-2 Critical (silent skipping of missing components) is now addressed by the warning on line 259.

## Warnings (should fix)

- [ ] `misc/install.sh:106-124` — **`resolve_asset()` duplicates curl security flags instead of reusing `$CURL`.** The function has its own `curl --proto =https --tlsv1.2 -fsS --retry 3 --retry-delay 2` invocation instead of deriving from `$CURL` (which adds `-L`). If someone updates `$CURL` flags, `resolve_asset` won't pick up the change. Consider factoring out the common flags or adding a comment explaining the intentional `-L` omission.

- [ ] `misc/install.sh:249-259` — **Script succeeds even if ALL components are missing.** If the archive contains no matching components, the user gets a warning but the script exits 0 and prints "installed into $INSTALL_DIR" and the getting-started banner. This is misleading. Consider adding a check: if `$missing` equals `$components`, `die` instead of `log`.

- [ ] `misc/install.sh:296` — **Legacy `$GOPATH/bin` cleanup omits `gnoweb`.** The install-dir cleanup (line 289) now uses `$FULL_COMPONENTS` (includes `gnoweb`), but the legacy cleanup hardcodes `gno gnokey gnodev gnobro`. Users who manually installed `gnoweb` from source won't have it cleaned up by `--uninstall`.

- [ ] `misc/install.sh:201` — **`per_page=30` may miss the latest `v*` tag.** This repo has many `chain/*` releases; with 30+ non-v releases between v releases, the v* tag could fall off the first page. Consider bumping to 100 (GitHub's max) or paginating.

## Nits

- [ ] `misc/install.sh:82-84` — Stray blank line between `else die ...` and `fi` (noted in rounds 1 and 2, still present).
- [ ] `misc/install.sh:10-11` — `FULL_COMPONENTS` duplicates `COMPONENTS`. Consider `FULL_COMPONENTS="$COMPONENTS gnoland"` to avoid divergence (noted in round 2, still present).

## Missing Tests

- [ ] No `shellcheck` lint or automated smoke tests (noted in rounds 1 and 2). A minimal CI step running `shellcheck -s sh misc/install.sh` would catch common POSIX violations and undefined variable references.

## Suggestions

- When `curl` fails against the unauthenticated API with a rate-limit error (HTTP 403/429), the error message could suggest setting `GITHUB_TOKEN`. This was suggested by @jeronimoalbi in review comments. The `die` on line 202 could check the HTTP status or at least mention the token in the error text. — `misc/install.sh:202`
- The `xattr -d com.apple.quarantine` on line 257 only removes the quarantine attribute from binaries that exist. On first run, macOS will still quarantine the entire `$INSTALL_DIR` if it was created by a `curl|sh` process. Consider also running `xattr -d com.apple.quarantine "$INSTALL_DIR"` (best-effort) or `xattr -cr "$INSTALL_DIR"` to recursively clear it. — `misc/install.sh:257`

## Questions for Author

- `gnodev` and `gnobro` are still in the default COMPONENTS but absent from the only binary release (`v0.1.0`). The warning now makes this discoverable, but the first-time user experience is still 3-of-5 components. Should these be removed from the default list until releases include them, or is the next release expected to bundle all five?
- The `resolve_asset()` function uses a separate `curl` invocation instead of `$CURL`. Was this intentional only to omit `-L`, or is there another reason? A comment explaining the choice would help future maintainers.

## Verdict

APPROVE — All previously open Critical and Warning items have been addressed. The remaining findings are maintenance/UX improvements (flag duplication in `resolve_asset`, zero-components-should-fail, legacy cleanup gap, per_page limit) that can land in follow-ups. The script is well-structured, POSIX-compliant, and handles the repo's unusual release layout correctly. Good work on the `GITHUB_TOKEN` security handling (unsetting the exported var, xtrace suppression, CDN auth isolation).
