# PR #5123: fix: gnokey add -derivation-path flag

URL: https://github.com/gnolang/gno/pull/5123
Author: D4ryl00 | Base: master | Files: 2 | +412 -55
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `365eb907c` (stale — +8 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5123 365eb907c`

**Verdict: APPROVE** — fixes a real bug (`--derivation-path` printed addresses but never persisted derived keys); collision handling, passphrase preflight, and tests look correct. One minor warning on silent duplicate-path collapse, otherwise nits and questions.

## Summary

`gnokey add --derivation-path <p>` previously printed derived addresses without saving them, then persisted only the non-derived key under the base name (issue #5122). This PR makes the flag actually create and store one keybase entry per derivation path. Single-path invocations keep the base name; multi-path invocations name entries `<base>-a<account>i<index>`. Passphrases for all derived keys are collected up front (preflight) so a late mismatch aborts before any key is written. Collision handling reuses the existing `handleCollision` per derived entry.

## Glossary

- `BIP44Params` — parsed `(purpose, coinType, account, change, addressIndex)` from a derivation path string ([`hd/hdpath.go`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/hd/hdpath.go#L34) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/hd/hdpath.go#L34)).
- `deriveKeyName` — builds the per-path stored name: base for single path, `base-a<A>i<I>` for multi ([`add.go:433`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L433-L439) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L433-L439)).
- `handleCollision` — pre-existing helper that prompts the user when a name or address already exists in the keybase ([`add.go:551`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L551) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L551)).
- preflight passphrase loop — collect every passphrase before persisting any key, so a mismatch (`errPassphraseMismatch`) leaves the keybase untouched ([`add.go:303-313`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L303-L313) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L303-L313)).

## Fix

Before: a single path through `execAdd` parsed `-derivation-path` only to feed `printDerive`, then called `CreateAccount(name, ..., cfg.Account, cfg.Index)`, so the persisted key ignored the supplied path. After: the function splits into two branches — empty `cfg.DerivationPath` keeps the legacy behavior; non-empty walks every path through `NewParamsFromPath`, computes a deterministic stored name via `deriveKeyName`, runs `handleCollision` per entry, collects all passphrases in a preflight loop, then calls `CreateAccountBip44` per surviving entry. Mnemonic output via `printCreate` is gated on `i == 0` to avoid repeating it for each derived key.

## Critical (must fix)

None.

## Warnings (should fix)

- **[silent overwrite on duplicate `--derivation-path`]** [`tm2/pkg/crypto/keys/client/add.go:261-275`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L261-L275) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L261-L275) — passing the same path twice produces two entries with identical `deriveKeyName`; the first write is silently overwritten by the second.
  <details><summary>details</summary>

  The pre-write collision check at [`add.go:282-301`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L282-L301) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L282-L301) only consults the existing keybase, not the rest of `entries`. With two passes through the create loop using the same `(name, address)`, `dbKeybase.writeInfo` ([`tm2/pkg/crypto/keys/keybase.go:455-486`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/keybase.go#L455-L486) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/keybase.go#L455-L486)) just overwrites the prior record. The user is still prompted twice for a passphrase; the first passphrase is then discarded with no warning. I confirmed this by running a one-off test (mnemonic + two repeats of `44'/118'/0'/0/0`): the command returns `nil`, `kb.List()` shows a single key, and the second passphrase silently wins.

  Fix: dedupe `cfg.DerivationPath` (or `entries`) up front and either error or print a warning when duplicates are dropped — same shape as the `--account/--index` warning at [`add.go:252-254`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L252-L254) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L252-L254).

  Repro:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5123 -R gnolang/gno
  cat > tm2/pkg/crypto/keys/client/dup_path_test.go <<'EOF'
  package client

  import (
  	"context"
  	"strings"
  	"testing"
  	"time"

  	"github.com/gnolang/gno/tm2/pkg/commands"
  	"github.com/gnolang/gno/tm2/pkg/crypto/keys"
  )

  func TestAdd_DerivePath_Duplicates(t *testing.T) {
  	kbHome := t.TempDir()
  	mnemonic := generateTestMnemonic(t)
  	baseOptions := BaseOptions{InsecurePasswordStdin: true, Home: kbHome}
  	path := "44'/118'/0'/0/0"

  	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
  	defer cancelFn()

  	io := commands.NewTestIO()
  	io.SetIn(strings.NewReader(mnemonic + "\npass1\npass1\npass2\npass2\n"))

  	cmd := NewRootCmdWithBaseConfig(io, baseOptions)
  	args := []string{
  		"add", "--insecure-password-stdin", "--home", kbHome, "--recover", "key",
  		"--derivation-path", path, "--derivation-path", path,
  	}
  	if err := cmd.ParseAndRun(ctx, args); err != nil {
  		t.Fatalf("unexpected error: %v", err)
  	}

  	kb, _ := keys.NewKeyBaseFromDir(kbHome)
  	all, _ := kb.List()
  	t.Logf("Total keys persisted for duplicate paths: %d", len(all))
  	if len(all) != 1 {
  		t.Fatalf("expected silent collapse to 1 key, got %d", len(all))
  	}
  }
  EOF
  go test -v -run TestAdd_DerivePath_Duplicates ./tm2/pkg/crypto/keys/client/
  rm tm2/pkg/crypto/keys/client/dup_path_test.go
  ```
  </details>

## Nits

- [`tm2/pkg/crypto/keys/client/add.go:136-150`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L136-L150) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L136-L150) — `NewParamsFromPath` is called twice per path (validation here, then again at [`add.go:264`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L264) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L264) when building `entries`). Either parse once and stash the `*BIP44Params`, or drop the validation loop and let the entries loop surface the error.
- [`tm2/pkg/crypto/keys/client/add.go:332`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L332) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L332) — `printDerive` iterates the original `cfg.DerivationPath`; when an entry was collision-renamed (no new key created for it), the "[Derived Accounts]" list still includes the path/address. Not wrong, but mildly inconsistent with the `printCreate` loop below which only walks `infos`.
- [`tm2/pkg/crypto/keys/client/add.go:438`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L438) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L438) — naming format `%s-a%di%d` is fine for the path subset the regex allows (`44'/118'/<acc>'/0/<idx>`); document the format above the function so consumers (scripts grepping `gnokey list`) know what to expect.
- [`tm2/pkg/crypto/keys/client/add.go:252-254`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L252-L254) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L252-L254) — warning is printed only when `Account != 0 || Index != 0`. Consider including it whenever `--account`/`--index` were explicitly set, even when set to `0` — but Go flag parsing makes that hard without a sentinel, so leave as-is unless you already have a pattern in this codebase.

## Missing Tests

- **[duplicate paths]** [`tm2/pkg/crypto/keys/client/add_test.go:701`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add_test.go#L701) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add_test.go#L701) — no test covers `--derivation-path X --derivation-path X` (same path twice). Tied to the Warning above; once you decide on behavior (error, warn, or silently collapse), pin it with a test.
- **[force + collision]** no test verifies that `--force --derivation-path` skips `handleCollision` and overwrites. The single-key path has [`force flag overrides without prompt`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add_test.go#L558) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add_test.go#L558); the derivation branch deserves the equivalent.
- **[partial collision then success]** with three paths where path #2 prompts an overwrite (`y`) and path #3 succeeds normally — verifies the `pendingEntries` rewrite at [`add.go:300`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L300) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L300) and the per-entry passphrase ordering.

## Suggestions

- [`tm2/pkg/crypto/keys/client/add.go:160-198`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L160-L198) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L160-L198) — `getMnemonic` is defined inline but used in both branches. Extracting it to a package-level helper (named `readOrGenerateMnemonic` or similar) would make the two branches in `execAdd` symmetric and easier to follow.
- [`tm2/pkg/crypto/keys/client/add.go:343-346`](https://github.com/gnolang/gno/blob/365eb907c/tm2/pkg/crypto/keys/client/add.go#L343-L346) · [↗](../../../../../.worktrees/gno-review-5123/tm2/pkg/crypto/keys/client/add.go#L343-L346) — printing the mnemonic only on `i == 0` is correct, but worth a one-line comment ("mnemonic is identical for every derived key; print once") so the next reader doesn't reflexively "fix" it to print per key.

## Questions for Author

- Was the duplicate-path silent-collapse a deliberate choice (treat repeats as idempotent) or an oversight? If deliberate, please document it; if not, the Warning above proposes a fix shape.
- Is there a downstream tool that parses key names to recover the `(account, index)` pair, or is `printDerive` the only contract? The `<base>-a<A>i<I>` format is fine either way, but if a downstream depends on it, lock it in with a doc comment.
- Should `--derivation-path` imply `--recover` (or at least warn) when used without it? A fresh mnemonic spread across N derived keys is a valid use case, but it's unusual enough that an explicit warning would help.
