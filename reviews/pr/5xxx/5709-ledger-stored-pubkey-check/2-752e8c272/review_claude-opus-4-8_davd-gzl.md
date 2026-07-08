# PR [#5709](https://github.com/gnolang/gno/pull/5709): fix(gnokey): bind Ledger signing to the stored pubkey identity

URL: https://github.com/gnolang/gno/pull/5709
Author: tbruyelle | Base: master | Files: 2 | +57 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `752e8c272` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5709 752e8c272`

Round 2. Head advanced `37db202e` → `752e8c272` (round 1 sha now GC'd). Diff against `merge-base(master, HEAD)` is byte-for-byte the round-1 fix: 9 lines in `keybase.go`, 48-line regression test, `+57 -0`. Findings carried forward, re-verified, re-anchored; verdict unchanged. CI red on this PR is codecov-upload only: every failing `test` job (`main`, `gnodev`, `gnobr`, ...) has its `Go test` step green and fails at `Upload coverage to Codecov`.

**TL;DR:** gnokey never checked that the Ledger you plug in to sign is the same Ledger that created the key. This PR adds that check, so signing with the wrong device at the same BIP44 path is rejected instead of silently producing a signature under the wrong identity.

**Verdict: APPROVE** — small, targeted security fix mirroring the cosmos-sdk `SignWithLedger` guard; regression test exercises the device-swap scenario; second-line signature self-check is defensive but harmless. Only nits and optional coverage.

## Summary

`dbKeybase.Sign` for a Ledger key reconstructs `PrivKeyLedgerSecp256k1` via [`NewPrivKeyLedgerSecp256k1Unsafe(info.Path)`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L37-L50) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L37-L50), which reads the pubkey from whichever device is plugged in *right now* at the BIP44 path. The stored `info.PubKey` from the original `CreateLedger` was never compared against the live device's pubkey, so a key reference created on device A could silently sign with device B sitting at the same `44'/118'/0'/0/0` path: operator picks the wrong USB device, signature goes to the wrong identity, no warning. PR adds two guards in [`tm2/pkg/crypto/keys/keybase.go`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L250-L276) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L250-L276): (1) reject when the live device's pubkey does not match `info.PubKey`; (2) verify the produced signature against the live pubkey to catch malformed device output. Reported in NEWTENDG-269.

## Glossary

- `ledgerInfo` — keybase entry storing only `{Name, PubKey, Path}`; no privkey, signature is offloaded to the device.
- `info.PubKey` — pubkey captured at `CreateLedger` time and persisted; the trusted identity.
- `NewPrivKeyLedgerSecp256k1Unsafe` — rebuilds a `PrivKeyLedgerSecp256k1` from path alone, caching the pubkey read from the currently-connected device, no user confirmation.
- `validateKey` (inside `ledger_secp256k1.sign`) — compares a pubkey freshly read from the device against the `PrivKeyLedgerSecp256k1.CachedPubKey` field; catches a device swap *between* reconstruction and signing within one Sign call, but not the create-vs-sign swap fixed here.

## Fix

Before: the `ledgerInfo` branch of [`dbKeybase.Sign`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L250-L254) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L250-L254) called `NewPrivKeyLedgerSecp256k1Unsafe(info.Path)` and proceeded to sign with whatever the live device returned at that BIP44 path; the stored `info.PubKey` was unused. After: immediately after reconstruction, [`priv.PubKey().Equals(info.PubKey)`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L256-L260) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L256-L260) is required; post-sign, [`pub.VerifyBytes(msg, sig)`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L273-L274) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L273-L274) re-verifies the device's signature against its own pubkey before returning. Both guards target the `ledgerInfo` case only; local-key signing is untouched.

Verified on `752e8c272`: removing the first guard makes the regression test fail (control reaches `priv.Sign`, then dies in `convertDERtoBER` with `malformed signature: too short` on the mock's nil signature), confirming the guard is what produces the mismatch rejection rather than an incidental later failure.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`tm2/pkg/crypto/keys/keybase.go:273`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L273) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L273) — the second guard could live inside the `case ledgerInfo:` block instead of re-asserting `isLedger` after the type switch.
  <details><summary>details</summary>

  Restructuring it as part of the case removes the post-switch `info.(ledgerInfo)` re-check and keeps all ledger-specific behavior together. Current shape works; purely a structural preference, both read cleanly. No change needed.
  </details>

- [`tm2/pkg/crypto/keys/keybase.go:257-258`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L257-L258) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L257-L258) — error message is long and uses `%q` for the name but bare `%s` for both pubkeys.
  <details><summary>details</summary>

  Bech32-formatted pubkeys are self-delimited, so `%q` on the name is appropriate but the line runs ~150 chars and reads heavy. Cosmetic. Fix: trim, e.g. `fmt.Errorf("ledger pubkey mismatch for key %q: device=%s stored=%s", nameOrBech32, priv.PubKey(), info.PubKey)`.
  </details>

- [`tm2/pkg/crypto/keys/keybase_ledger_test.go:51-52`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase_ledger_test.go#L51-L52) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go#L51-L52) — the `Discover` closure captures the outer `device` *variable*, so reassigning `device = ...` swaps what `Discover()` returns; that mechanism is the whole point of the test and goes unstated.
  <details><summary>details</summary>

  A reader who skims this and doesn't notice that `device` is read on every `Discover()` call won't understand why the second `device = newSwappableLedgerMock(t)` flips the test from passing to failing. Fix: add `// Discover() closes over device, so swapping the local rewires what subsequent calls see.` above the closure.
  </details>

## Missing Tests

- [`tm2/pkg/crypto/keys/keybase_ledger_test.go`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase_ledger_test.go) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase_ledger_test.go) — no positive test asserting a same-device sign still succeeds, and the swap test can't reach the real risk.
  <details><summary>details</summary>

  `TestSignLedgerMismatchedDevice` and `TestCreateLedger` never drive a successful `Sign`. The `swappableLedgerMock` inherits `MockLedger.SignSECP256K1`, which returns `nil`, so no test produces a verifiable signature. Two consequences: the happy path (`kb.Sign` succeeds when the same device is plugged in) is unasserted, and removing guard #1 makes the swap test fail in `convertDERtoBER` (`malformed signature: too short`) rather than at a silently-wrong signature, so the test proves the mismatch *message* but not the *silent-wrong-identity* scenario the fix defends against. A sign-capable mock returning a real DER signature under a controllable key would lock the happy path and let the swap test demonstrate the actual bug. Optional; the sign path is well-trodden.
  </details>

- [`tm2/pkg/crypto/keys/keybase.go:273`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L273) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L273) — no coverage for the second guard (malformed-device-output path).
  <details><summary>details</summary>

  The `pub.VerifyBytes(msg, sig)` check meant to catch a device that returns a syntactically valid but cryptographically invalid signature has zero coverage. A test stubbing `SignSECP256K1` to return a well-formed-but-wrong 64-byte signature would assert the `ledger produced an invalid signature` path. Optional, defense-in-depth; the guard is small enough to read by inspection.
  </details>

## Suggestions

- [`tm2/pkg/crypto/keys/keybase.go:255`](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L255) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L255) — name the window each of the three pubkey checks on the Ledger sign path protects, so a later refactor doesn't delete one as redundant.
  <details><summary>details</summary>

  Three pubkey checks now exist on the Ledger sign path: this new one (stored vs live, in keybase), `validateKey` inside `sign()` ([ledger_secp256k1.go:208](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L208) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/ledger/ledger_secp256k1.go#L208), reconstructed-cached vs re-read live), and the post-sign `VerifyBytes`. The first two look like duplicates on a quick read but cover different windows. Fix: a one-line comment naming the window each one protects helps future maintainers not delete one thinking it's redundant.
  </details>

## Open questions

- Mirror the guard at `CreateLedger` time too. The comment on [keybase.go:121](https://github.com/gnolang/gno/blob/752e8c272/tm2/pkg/crypto/keys/keybase.go#L121) · [↗](../../../../../.worktrees/gno-review-5709/tm2/pkg/crypto/keys/keybase.go#L121) already hints at it ("Once Cosmos App v1.3.1 is compulsory, it could be possible to check that pubkey and addr match"). `CreateLedger` reads the pubkey once via the safe path (`getPubKeyAddrSafe`, user-confirmed) and doesn't cross-check `getPubKeyUnsafe` on the same device, so a device returning different keys on confirmed vs unconfirmed paths still slips through at create time. Out of scope, tracked by the existing TODO; not posted.
- Was the second guard (`pub.VerifyBytes`) added because of an observed malformed-signature failure in the field, or purely belt-and-braces against a hostile device? `validateKey` already runs before `SignSECP256K1`, so the post-sign verify only adds coverage for "right key, but the device returned a sig that doesn't verify under its own pubkey." Low-value for the author; not posted.
