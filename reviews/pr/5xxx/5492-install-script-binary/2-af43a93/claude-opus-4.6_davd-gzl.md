# PR #5492: feat(misc): rewrite install.sh as precompiled binary downloader

**URL:** https://github.com/gnolang/gno/pull/5492
**Author:** notJoon | **Base:** master | **Files:** 1 | **+211 -118**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

Second review round. Since the first review (commit `ecef003`), the following changes were made:

1. **`latest` resolution fixed** — The script now fetches `/releases?per_page=30` and picks the first non-prerelease `v*` tag instead of relying on `/releases/latest` (which resolves to `chain/gnoland1.1` with no assets). This addresses the previous Critical #2.
2. **`gnoweb` added to default COMPONENTS** — Default set is now `gno gnokey gnodev gnobro gnoweb`. A `--full` flag adds `gnoland`. This partially addresses the previous Critical #1 (component mismatch) and the uninstall asymmetry warning.
3. **"Getting started" banner added** — Per reviewer suggestion from @alexiscolin.

## Test Results
- **Existing tests:** N/A — standalone shell script, no automated tests
- **CI:** All checks pass
- **Edge-case tests:** skipped

## Critical (must fix)

- [ ] `misc/install.sh:10` — **COMPONENTS list still includes `gnodev` and `gnobro` which don't exist in any release.** The `v0.1.0` release (the only one with binaries) contains `gno`, `gnokey`, `gnoland`, `gnoweb` — not `gnodev` or `gnobro`. The `[ -f ] || continue` silently skips missing binaries so the install won't error, but users get 3 of 5 expected components with no warning. This was flagged in review round 1 and remains unfixed. At minimum, add a warning when expected components are not found in the archive.

## Warnings (should fix)

- [ ] `misc/install.sh:101-107` — **The `latest_v_tag` awk/sed fallback doesn't filter out prereleases.** The jq path correctly filters `prerelease == false`, but the sed fallback (`sed -n 's/.*"tag_name": *"\(v[^"]*\)".*/\1/p' | head -1`) just grabs the first `v*` tag, which could be a prerelease. Minor risk today but will break when prerelease tags are published.

- [ ] `misc/install.sh:117-121` — **awk JSON fallback for `asset_url` fragility** remains from round 1. Still relies on GitHub's JSON field ordering. Not blocking but worth a comment acknowledging the assumption.

- [ ] `misc/install.sh:126` — **`--proto =https` curl version requirement** remains from round 1. `curl < 7.64` will fail with an unhelpful error.

- [ ] `misc/install.sh:135` — **`per_page=30` may miss the latest `v*` tag** if more than 30 non-v releases are published between v releases. Currently not a risk, but consider bumping or paginating.

- [ ] `misc/install.sh:159-160` — **No `GITHUB_TOKEN` support** for authenticated API requests. Still relevant per @jeronimoalbi's comment. Could be a follow-up PR.

## Nits
- [ ] `misc/install.sh:80-82` — Stray blank line between `else die ...` and `fi` (unchanged from round 1).
- [ ] `misc/install.sh:10-11` — `FULL_COMPONENTS` duplicates `COMPONENTS` plus `gnoland`. Consider `FULL_COMPONENTS="$COMPONENTS gnoland"` to avoid divergence.

## Missing Tests
- [ ] No `shellcheck` lint or smoke tests. Same as round 1.

## Suggestions
- Warn when a component from the list is not found in the archive rather than silently skipping. A simple `log "warning: $c not found in archive, skipping"` before `continue` on line 176 would make incomplete installs discoverable. — `misc/install.sh:176`
- The uninstall list (line 210) now includes `gnoweb` which matches the new default, but the legacy `$GOPATH/bin` cleanup (line 217) still uses the old 4-component list without `gnoweb`. Consider adding `gnoweb` there too. — `misc/install.sh:217`

## Questions for Author
- `gnodev` and `gnobro` are still in the default COMPONENTS but absent from all existing releases. Are these expected to appear in the next release? If not, the default list should match reality.
- Is `GITHUB_TOKEN` support planned as a follow-up, or should it be included here?

## Verdict
REQUEST CHANGES — The `latest` resolution is fixed (good), but the silent skipping of missing components (`gnodev`, `gnobro`) means the default install delivers an incomplete set without any feedback to the user. This was flagged in round 1. Adding a warning log is a one-line fix. The rest are non-blocking improvements that can land in follow-ups.
