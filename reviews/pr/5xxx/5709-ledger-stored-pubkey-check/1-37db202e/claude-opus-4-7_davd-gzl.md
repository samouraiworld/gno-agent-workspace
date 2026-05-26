# PR #5709: fix(gnokey): bind Ledger signing to the stored pubkey identity

URL: https://github.com/gnolang/gno/pull/5709
Author: tbruyelle | Base: master | Files: 2 | +57 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — small, targeted security fix mirroring the cosmos-sdk `SignWithLedger` guard; test exercises the swap scenario; second-line signature self-check is defensive but harmless. Only nits.

## Summary

`dbKeybase.Sign` for a Ledger key reconstructs `PrivKeyLedgerSecp256k1` via [`NewPrivKeyLedgerSecp256k1Unsafe(info.Path)`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L37-L50), which reads the pubkey from whichever device is plugged in *right now* at the BIP44 path. The stored `info.PubKey` from the original `CreateLedger` was never compared against the live device's pubkey, so a key reference created on device A could silently sign with device B sitting at the same `44'/118'/0'/0/0` path — operator picks the wrong USB device, signature goes to the wrong identity, no warning. PR adds two guards in [`tm2/pkg/crypto/keys/keybase.go`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L250-L276): (1) reject when the live device's pubkey does not match `info.PubKey`; (2) verify the produced signature against the live pubkey to catch malformed device output.

## Glossary

- `ledgerInfo` — keybase entry storing only `{Name, PubKey, Path}`; no privkey, signature is offloaded to the device.
- `info.PubKey` — pubkey captured at `CreateLedger` time and persisted; the trusted identity.
- `NewPrivKeyLedgerSecp256k1Unsafe` — rebuilds a `PrivKeyLedgerSecp256k1` from path alone, with the cached pubkey read from the currently-connected device, no user confirmation.
- `validateKey` (inside `ledger_secp256k1.sign`) — compares pubkey freshly read from the device against the `PrivKeyLedgerSecp256k1.CachedPubKey` field; this catches a device swap *between* reconstruction and signing within one Sign call, but not the create-vs-sign swap fixed here.

## Fix

Before: the `ledgerInfo` branch of [`dbKeybase.Sign`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L250-L254) called `NewPrivKeyLedgerSecp256k1Unsafe(info.Path)` and proceeded to sign with whatever the live device returned at that BIP44 path; the stored `info.PubKey` was unused. After: immediately after reconstruction, [`priv.PubKey().Equals(info.PubKey)`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L256-L260) is required; post-sign, [`pub.VerifyBytes(msg, sig)`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L273-L275) re-verifies the device's signature against its own pubkey before returning. Both guards target the `ledgerInfo` case only; local-key signing is untouched.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`tm2/pkg/crypto/keys/keybase.go:273`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L273) — the second guard could live inside the `case ledgerInfo:` block instead of re-asserting `isLedger` after the type switch.
  <details><summary>details</summary>

  Restructuring it as part of the case removes the post-switch `info.(ledgerInfo)` re-check and keeps all ledger-specific behavior together. Either move `priv.Sign(msg)` plus the verify inside the case, or short-circuit on local keys by computing `sig, err := priv.Sign(msg)` first and re-checking the type. Current shape works — purely a structural preference; both reads cleanly.
  </details>

- [`tm2/pkg/crypto/keys/keybase.go:257-259`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L257-L259) — error message is long and uses `%q` for the name (Go-escaped) but bare `%s` for both pubkeys.
  <details><summary>details</summary>

  Bech32-formatted pubkeys are already self-delimited; `%q` on the name is appropriate but the line is ~150 chars and reads heavy. Could trim to `fmt.Errorf("ledger pubkey mismatch for key %q: device=%s stored=%s", nameOrBech32, priv.PubKey(), info.PubKey)`. Cosmetic.
  </details>

- [`tm2/pkg/crypto/keys/keybase_ledger_test.go:53`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go#L53) — test cleanup is correct, but the closure `func() (ledger.SECP256K1, error) { return device, nil }` captures the outer `device` *variable* (so reassigning `device = ...` swaps what `Discover()` returns). Worth a one-line comment, the mechanism is the whole point of the test.
  <details><summary>details</summary>

  A reader who skims this and doesn't notice that `device` is read on every `Discover()` call won't understand why the second `device = newSwappableLedgerMock(t)` flips the test from passing to failing. A `// Discover() closes over device, so swapping the local rewires what subsequent calls see.` line on top of the closure would make the swap mechanism explicit.
  </details>

## Missing Tests

- [`tm2/pkg/crypto/keys/keybase_ledger_test.go`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go) — no coverage for the second guard (malformed-device-output path).
  <details><summary>details</summary>

  The added test exercises only the first guard (live pubkey ≠ stored pubkey). The `pub.VerifyBytes(msg, sig)` check on [keybase.go:273](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L273) — meant to catch a device that returns a syntactically valid but cryptographically invalid signature — has zero coverage. A second test stubbing `SignSECP256K1` to return a well-formed-but-wrong 64-byte signature would assert the "ledger produced an invalid signature" error path. Optional, defense-in-depth coverage; the guard is small enough to read by inspection.
  </details>

- [`tm2/pkg/crypto/keys/keybase_ledger_test.go`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go) — no positive test asserting a same-device sign still works.
  <details><summary>details</summary>

  The existing `TestCreateLedger` doesn't call `Sign`. With this PR, there is no test demonstrating `kb.Sign(...)` succeeds when the same device is plugged in — only the failure path. The mock returns nil for `SignSECP256K1`, so a real positive test needs a sign-capable mock that produces a verifiable DER sig. Not strictly required (the path is well-trodden), but a passing case would lock the happy path against future regressions.
  </details>

## Suggestions

- [`tm2/pkg/crypto/keys/keybase.go:255`](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L255) — note in the comment that this guard is the create-vs-sign safety net, distinct from `validateKey`'s reconstruct-vs-sign safety net inside `priv.Sign()`.
  <details><summary>details</summary>

  Three pubkey checks now exist on the Ledger sign path: this new one (stored vs live, in keybase), `validateKey` inside `sign()` ([ledger_secp256k1.go:208](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L208), reconstructed-cached vs re-read live), and the post-sign `VerifyBytes`. The first two look like duplicates on a quick read but cover different windows. A 1-line comment naming the window each one protects helps future maintainers not delete one thinking it's redundant.
  </details>

- Consider mirroring this guard at `CreateLedger` time as well.
  <details><summary>details</summary>

  The comment on [keybase.go:122](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L122) already hints at this: "Once Cosmos App v1.3.1 is compulsory, it could be possible to check that pubkey and addr match". `CreateLedger` reads the pubkey once via the "safe" path (`getPubKeyAddrSafe`, user-confirmed) — but doesn't sanity-check against `getPubKeyUnsafe` on the same device, so a device that returns *different* keys on confirmed vs unconfirmed paths would still slip through at create time. Out of scope for this PR; tracked by the existing TODO.
  </details>

## Questions for Author

- Was the second guard (`pub.VerifyBytes(msg, sig)`) chosen because of an observed malformed-signature failure in the field, or purely as belt-and-braces against a hostile device? Asking because the inner `validateKey` already runs before the actual `SignSECP256K1` call and would catch most "wrong device returned wrong key" cases — the post-sign verify only catches "right key, but the device returned a sig that doesn't actually verify under its own pubkey".

- The cosmos-sdk reference cited in the PR body (`crypto/keyring/keyring.go` `SignWithLedger`) — was the comparison done against the current `cosmos-sdk@main` or an older version? Worth pinning the commit/tag in the commit message so future readers can diff drift.
