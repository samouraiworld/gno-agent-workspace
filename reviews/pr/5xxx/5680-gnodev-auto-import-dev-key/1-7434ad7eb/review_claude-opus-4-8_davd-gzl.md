# PR #5680: feat(gnodev): auto-import the dev key into the local keybase

URL: https://github.com/gnolang/gno/pull/5680
Author: davd-gzl | Base: master | Files: 7 | +477 -22
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: 7434ad7eb (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5680 7434ad7eb`

Mode: review-then-improve. A 4-lens workflow (red / blue / correctness / docs) surfaced 14 raw findings; adversarial verification confirmed 11. All 11 are resolved in the worktree (uncommitted on branch `feat/gnodev-auto-import-devtest`). See **Resolutions applied**.

**TL;DR:** On startup gnodev now writes the public deployer seed into `~/.config/gno` under the name `dev`, so `gnokey ... dev` signs against the genesis-funded address with no setup. Opt out with `-no-dev-key`. The review found the import could abort gnodev on a degraded keybase and that the ADR described the `test1` collision backwards; both are fixed.

**Verdict: NEEDS DISCUSSION** — the original PR is sound in intent, but two design calls needed deciding: whether the import may ever block startup (it shouldn't), and what happens when the deployer address already exists under another name (the ADR claimed "two names", the keybase actually deletes the old one). Both now resolved in the worktree; one tradeoff (below) is yours to confirm.

## Summary
The feature is small and well-tested (8 tests at review time). Two real issues sat under it. First, `ensureDevKey` returned errors fatally from three keybase operations while every sibling branch warned and continued, so a locked, corrupt, file-backed, or unwritable keybase aborted gnodev boot with an opaque leveldb message. Second, the keybase enforces one name per address: importing `dev` for an address already stored as `test1` silently deletes `test1`, the exact opposite of the ADR's "get two names ... this is benign" and of the PR's "existing entries are never overwritten". The fix makes the import fully best-effort and skips entirely when the address is already present under any name.

## Design decision to confirm
The address-already-present guard preserves a user's existing `test1` entry by **skipping** the `dev` import for them. Tradeoff: those users keep signing under `test1` and do **not** get a `dev` name, so doc snippets that say "use `dev`" won't apply to them. The alternative (import `dev` anyway) would delete their `test1`. Preserving the existing entry matches the PR's stated "never overwritten" promise and is the non-destructive choice, so that is what's implemented. Flip it if you'd rather canonicalize everyone to `dev`.

## Resolutions applied
All edits are in `.worktrees/gno-review-5680`, uncommitted, on the PR branch.

- `contribs/gnodev/setup_address_book.go` — `ensureDevKey` rewritten: returns nothing, every keybase failure warns and continues; new `openKeybase` recovers the panic `keys.NewKeyBaseFromDir` raises on an unwritable home; address-present-under-any-name short-circuit added; default-home guard now uses `filepath.Clean` on both sides; opt-out log string `-no-dev-key`.
- `contribs/gnodev/setup_address_book_test.go` — 4 tests added (address-under-other-name preserved, broken keybase degrades, cannot-create-default-home, unwritable home recovers); opt-out assertion updated. 12 tests, all pass.
- `gno.land/adr/pr5680_gnodev_auto_import_dev_key.md` — Decision steps, Alternative 3, Consequences rewritten to the real behavior; `gnokey export dev` corrected (armored privkey, not seed); flag spelled `-no-dev-key`.
- `docs/builders/getting-started.md`, `docs/builders/local-dev-with-gnodev.md` — flag `--no-dev-key` → `-no-dev-key`, one em-dash removed, overwrite wording matched to new behavior.
- `docs/builders/example-minisocial-dapp.md` — stale `default address imported name=test1` banner updated to the two-line `dev key imported` / `default address resolved from keybase` output (this file was missed by the original PR).

## Verified live (on 7434ad7eb, pre-fix behavior, and on the fixed worktree)
- Fresh `GNOHOME`: gnodev creates the home, logs `dev key imported name=dev` then `default address resolved from keybase name=dev`; `gnokey list` shows `dev` at `g1jg8m…` and it is signable. Matches the doc banner.
- Second boot on the same home: `dev key already present in keybase, skipping`.
- Pre-fix, a regular-file `GNOHOME` and a locked keybase each aborted gnodev with `unable to load keybase: ... <leveldb error>` (reproduced in scratch tests); post-fix both warn and boot.

## Warnings (should fix) — resolved
- **[import can block gnodev boot]** [`setup_address_book.go:105`](https://github.com/gnolang/gno/blob/7434ad7eb/contribs/gnodev/setup_address_book.go#L105) · [↗](../../../../../.worktrees/gno-review-5680/contribs/gnodev/setup_address_book.go#L105) — A locked, corrupt, file-backed, or unwritable keybase made `ensureDevKey` return an error that propagated through `setupAddressBook` and `App.Setup` (`unable to load keybase`), aborting startup. Inconsistent with every other branch, which warns and continues. On a fresh install the import now creates the home and opens the keybase, so contention that previously couldn't happen can now block boot. Fix: best-effort throughout (warn + return), plus `openKeybase` recover for the unwritable-home panic.
- **[ADR backwards; `test1` silently deleted]** [`pr5680_..._dev_key.md`](https://github.com/gnolang/gno/blob/7434ad7eb/gno.land/adr/pr5680_gnodev_auto_import_dev_key.md?plain=1#L152) · [↗](../../../../../.worktrees/gno-review-5680/gno.land/adr/pr5680_gnodev_auto_import_dev_key.md) — `dbKeybase.writeInfo` ([keybase.go:471-479](https://github.com/gnolang/gno/blob/7434ad7eb/tm2/pkg/crypto/keys/keybase.go#L471-L479)) deletes any existing name pointing at the same address before writing a new one. So `CreateAccount("dev", DefaultDeployerSeed, ...)` deletes a pre-existing `test1`. The ADR said users "get *two* names ... benign" and the PR said "existing entries are never overwritten"; both were wrong for this case. Reproduced: pre-seed `test1` at the deployer address, run `ensureDevKey`, `HasByName("test1")` flips true→false. Fix: skip the import when the address is already present under any name; ADR/docs corrected.

## Nits — resolved
- [`setup_address_book.go:90`](https://github.com/gnolang/gno/blob/7434ad7eb/contribs/gnodev/setup_address_book.go#L90) · [↗](../../../../../.worktrees/gno-review-5680/contribs/gnodev/setup_address_book.go#L90) — A default home that is a regular file aborted startup with an opaque leveldb error. Subsumed by the best-effort fix.
- [`setup_address_book.go:95`](https://github.com/gnolang/gno/blob/7434ad7eb/contribs/gnodev/setup_address_book.go#L95) · [↗](../../../../../.worktrees/gno-review-5680/contribs/gnodev/setup_address_book.go#L95) — Default-home guard was a raw string compare; a path-equivalent `-home` (trailing slash) silently disabled the import. Fix: `filepath.Clean` both sides.
- [`pr5680_..._dev_key.md`](https://github.com/gnolang/gno/blob/7434ad7eb/gno.land/adr/pr5680_gnodev_auto_import_dev_key.md?plain=1#L58) · [↗](../../../../../.worktrees/gno-review-5680/gno.land/adr/pr5680_gnodev_auto_import_dev_key.md) — ADR said `gnokey export dev` recovers "the seed"; `execExport` produces an armored, password-encrypted private key, not the BIP-39 mnemonic. Reworded.
- [`example-minisocial-dapp.md:151`](https://github.com/gnolang/gno/blob/7434ad7eb/docs/builders/example-minisocial-dapp.md?plain=1#L151) · [↗](../../../../../.worktrees/gno-review-5680/docs/builders/example-minisocial-dapp.md#L151) — Stale `default address imported name=test1` banner the binary can no longer produce. Updated.
- [`getting-started.md:145`](https://github.com/gnolang/gno/blob/7434ad7eb/docs/builders/getting-started.md?plain=1#L145) · [↗](../../../../../.worktrees/gno-review-5680/docs/builders/getting-started.md#L145) — Flag spelled `--no-dev-key` in prose but `-no-dev-key` in the README, `-h` output, and every sibling gnodev flag. Normalized to single dash here and in `local-dev-with-gnodev.md` and the log string.

## Suggestions — resolved
- [`setup_address_book.go:105`](https://github.com/gnolang/gno/blob/7434ad7eb/contribs/gnodev/setup_address_book.go#L105) · [↗](../../../../../.worktrees/gno-review-5680/contribs/gnodev/setup_address_book.go#L105) — Dead error branch: `keys.NewKeyBaseFromDir` always returns nil error ([utils.go:14](https://github.com/gnolang/gno/blob/7434ad7eb/tm2/pkg/crypto/keys/utils.go#L14)); the unwritable-home case it looked like it handled actually panics inside `NewLazyDBKeybase`. Fix: `openKeybase` recovers that panic and surfaces it as a normal error.

## Missing Tests — resolved
- [`setup_address_book_test.go`](https://github.com/gnolang/gno/blob/7434ad7eb/contribs/gnodev/setup_address_book_test.go) · [↗](../../../../../.worktrees/gno-review-5680/contribs/gnodev/setup_address_book_test.go) — No test for: the non-NotFound `GetByName` error branch, the `EnsureDir`-fails branch, the deployer-address-under-another-name case. Added `TestEnsureDevKey_DeployerAddressUnderOtherNameIsPreserved`, `TestEnsureDevKey_BrokenKeybaseDegradesGracefully`, `TestEnsureDevKey_CannotCreateDefaultHome`, `TestEnsureDevKey_UnwritableHomeDegradesGracefully`.

## Open questions
- `ImportKeybase` (the read path, pre-existing) still fails loud on a genuinely broken existing keybase; left as-is since failing loud on a real user keybase is arguably correct and it predates this PR. Worth a separate look if the best-effort philosophy should extend there.

## Dropped by verification (3)
- Nil-deref panic in `osm.DirExists` for a file-with-subpath home: not reachable from this code path.
- "Existing conflicting entry must be asserted byte-for-byte intact": the conflict branch does zero writes; the address assertion already covers it.
- "Doc tells users to reuse `dev` on staging/testnet": the sentence is scoped to the local-chain section, no cross-network claim.
