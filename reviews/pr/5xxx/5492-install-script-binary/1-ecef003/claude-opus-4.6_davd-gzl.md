# PR #5492: feat(misc): rewrite install.sh as precompiled binary downloader

**URL:** https://github.com/gnolang/gno/pull/5492
**Author:** notJoon | **Base:** master | **Files:** 1 | **+170 -123**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR replaces the existing `misc/install.sh` — a bash script that cloned the repo and ran `make install` (requiring Go and git) — with a POSIX-sh installer that downloads precompiled binaries from GitHub Releases.

The new script detects the host platform (linux/darwin × amd64/arm64), downloads the matching `gno_<ver>_<os>_<arch>.tar.gz` archive from the GitHub Releases API, verifies the SHA256 checksum, and installs binaries to `$HOME/.gno/bin`. It supports `--version`, `--dir`, `--uninstall`, `--help` flags and corresponding environment variables. The uninstall path also cleans up legacy source-build artifacts (`$GOPATH/bin`, `~/.gno/src`).

Part of the broader onboarding improvement effort (#5459) to bring gno's install experience on par with Solana/Foundry-style one-liners.

**Key design decisions:**
- Uses GitHub REST API (`api.github.com`) instead of public download URLs (which currently 404 for this repo)
- Falls back to `awk`/`sed` when `jq` is unavailable for JSON parsing
- Hardened transport with `--proto =https --tlsv1.2`
- Default components: `gno`, `gnokey`, `gnodev`, `gnobro`

**Discussion context:** Reviewers raised two important concerns: (1) `releases/latest` resolves to `chain/gnoland1.1` which has no binary assets, and (2) potential GitHub API rate limiting for unauthenticated requests. The author acknowledged both.

## Test Results
- **Existing tests:** N/A — this is a standalone shell script with no automated tests
- **CI:** All checks pass
- **Edge-case tests:** skipped — shell script, no test framework in place

## Critical (must fix)
- [ ] `misc/install.sh:9` — **COMPONENTS list does not match actual release assets.** The script defines `COMPONENTS="gno gnokey gnodev gnobro"`, but the only existing release (`v0.1.0`) contains `gno`, `gnokey`, `gnoland`, `gnoweb` — **not** `gnodev` or `gnobro`. Meanwhile `gnoland` and `gnoweb` are present in the archive but not in the COMPONENTS list and will be silently skipped. The `[ -f "$TMP/ext/$c" ] || continue` on line 143 means missing components are silently ignored, so the install won't fail, but users will get an incomplete install with no indication that `gnodev` and `gnobro` were unavailable. Either update COMPONENTS to match the actual release contents, or add a warning when expected components are missing.

- [ ] `misc/install.sh:110` — **`--version latest` is broken today.** `releases/latest` resolves to tag `chain/gnoland1.1` which has zero binary assets. The script will die at line 127 (`"$ARCHIVE is not an asset of $VERSION"`). Since `latest` is the default, the primary install path (`curl ... | sh`) fails out of the box. The author acknowledged this in comments but the script needs to either: (a) filter for `v*` tags when resolving "latest", (b) document the `--version v0.1.0` requirement prominently, or (c) hardcode a minimum working version until the release strategy is settled.

## Warnings (should fix)
- [ ] `misc/install.sh:96-101` — **The awk JSON fallback for `asset_url` is fragile.** It relies on the GitHub API's current pretty-printed JSON field order (`"url"` appearing before `"name"` within each asset object). The API does not guarantee field ordering. If GitHub changes the serialization order, the awk pattern will silently match the wrong URL or return empty. The `sed` fallback for `release_tag` (line 86) has a similar fragility but is less risky since `tag_name` is a top-level field. Consider documenting the assumption explicitly, or making the awk parser more robust (e.g., track asset boundaries with `{` / `}` braces).

- [ ] `misc/install.sh:106` — **`--proto =https` requires curl 7.64+.** Older systems (e.g., CentOS 7, older Ubuntu LTS) may ship with curl < 7.64 where this flag is unsupported and will cause a hard failure. Since the script targets POSIX-sh for broad compatibility, consider either checking the curl version first, or making the proto flag conditional.

- [ ] `misc/install.sh:131-132` — **No GitHub API rate limit handling.** Unauthenticated GitHub API requests are limited to 60/hour per IP. In CI/CD, corporate NAT, or hackathon environments, this limit can be hit easily. The script makes 3 API calls (metadata + archive + checksums) per run. When rate-limited, curl will get a 403 with a JSON error body, but the `die` messages won't indicate rate limiting as the cause. Consider: (a) checking for a `GITHUB_TOKEN` env var and passing `-H "Authorization: token $GITHUB_TOKEN"` if set, (b) detecting 403 responses and printing a helpful message.

- [ ] `misc/install.sh:146` — **`xattr` fallback logic is incorrect.** The line `[ "$OS" = "darwin" ] && xattr -d ... 2>/dev/null || true` has a subtle shell precedence issue. If `OS` is "darwin" and `xattr` fails (e.g., attribute not set), the `&&` chain short-circuits to `|| true`, which is fine. But if `OS` is NOT "darwin", `[ "$OS" = "darwin" ]` returns false, triggering `|| true`, which is also fine. However, if `xattr` succeeds but returns a non-zero exit for some other reason, the `|| true` masks it. This works in practice but is confusing. Clearer: `if [ "$OS" = "darwin" ]; then xattr ... 2>/dev/null || true; fi`.

- [ ] `misc/install.sh:164` — **Uninstall removes `gnoland` and `gnoweb` but doesn't install them.** The uninstall function removes `gno gnokey gnodev gnobro gnoland gnoweb` but COMPONENTS only installs `gno gnokey gnodev gnobro`. This asymmetry is confusing. Either both lists should match, or the uninstall should dynamically discover what's in INSTALL_DIR.

## Nits
- [ ] `misc/install.sh:72-74` — Stray blank line between `else die ...` and `fi`. The closing `fi` should immediately follow the `else` clause without the blank line for consistent style.
- [ ] `misc/install.sh:86` — The `sed` fallback for `release_tag` uses `sed -n 's/.../.../p'` while the comment says "fall back to awk". The function actually uses `sed`, not `awk`. The comment at line 75 should say "awk/sed" for accuracy.

## Missing Tests
- [ ] No automated tests exist for the installer script. Shell scripts of this complexity benefit from at least a smoke test (e.g., `shellcheck` linting in CI, a mock-server integration test, or at minimum a `--help` + `--version unknown-tag` error-path test). — `misc/install.sh`
- [ ] `shellcheck` analysis would catch several potential issues (e.g., unquoted `$CURL` expansion on lines 116/131/132 undergoes word splitting by design, but shellcheck would flag it — worth a `# shellcheck disable=SC2086` annotation to show it's intentional).

## Suggestions
- Add `GITHUB_TOKEN` support for authenticated API requests. This is standard practice for installer scripts (rustup, foundryup, etc.) and solves the rate-limit concern raised by @jeronimoalbi. Implementation: `[ -n "${GITHUB_TOKEN:-}" ] && AUTH_HEADER="-H \"Authorization: token $GITHUB_TOKEN\""`. — `misc/install.sh:106`
- Add a `--verify-only` or `--dry-run` flag that shows what would be installed without downloading. Useful for CI integration and debugging. — `misc/install.sh:38`
- The PR description mentions `curl | sh` as the primary install method, but the `--help` output shows `curl --proto '=https' --tlsv1.2 -sSf`. The README/docs install command should match. The `-sSf` flag differs from the PR body's `-fsSL`. Standardize on one. — `misc/install.sh:23`
- Consider pinning a known-good fallback version when `latest` has no assets, rather than failing. E.g., `VERSION_FALLBACK="v0.1.0"` used when the resolved latest has no matching archive. — `misc/install.sh:110`

## Questions for Author
- The PR body says "goreleaser tarballs (`gno_<ver>_<os>_<arch>.tar.gz`) from `v*` tags" — but there is no `.goreleaser.yml` in the repo. What creates the `v0.1.0` release? Is this format guaranteed for future releases?
- The `v0.1.0` archive contains `gnoland` and `gnoweb` but not `gnodev` or `gnobro`. Are future releases expected to include `gnodev`/`gnobro`, or should the COMPONENTS list be updated to match what's actually available?
- Is there a plan to fix the `releases/latest` resolution before merging this PR? Without it, the default `curl | sh` one-liner — the primary use case — doesn't work.
- Should the script support `GITHUB_TOKEN` for authenticated requests, given the API rate-limit concern raised by @jeronimoalbi?

## Verdict
REQUEST CHANGES — The default install path (`curl | sh` without `--version`) is broken because `releases/latest` resolves to a tag with no binary assets, and the COMPONENTS list doesn't match what's actually in the existing release. These are blocking issues that make the script non-functional for its primary use case. The awk JSON fallback fragility and missing rate-limit handling are secondary but should be addressed before merge.
