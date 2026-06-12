# PR #5657: feat(gnoclient): Add account session support

URL: https://github.com/gnolang/gno/pull/5657
Author: jefft0 | Base: master | Files: 5 | +501 -9
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 7262dc3c5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5657 7262dc3c5`

**TL;DR:** gnoclient (the Go SDK for talking to a gno.land node) gains account-session support, mirroring the session feature gnokey already has. A "session" is a delegated signing key that a master account authorizes to sign transactions on its behalf within spend limits. The PR adds a session-account query, three session-management transaction helpers (create / revoke / revoke-all), and teaches the signer to sign as a session by tagging its signature with the session address.

**Verdict: REQUEST CHANGES** — two real defects surfaced by adversarial tests: `SignTx` panics on a session-signed multi-signer tx ([`client_txs.go:435`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/client_txs.go#L435) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/client_txs.go#L435)), and `SignerFromKeybase.Validate` always errors for a session signer ([`signer.go:48-49`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/signer.go#L48-L49) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/signer.go#L48-L49)). The single-signer happy path is correct and well tested.

## Summary
The PR adds `QuerySessionAccount` (hits the new `auth/accounts/<master>/session/<session>` endpoint), `CreateSession` / `RevokeSession` / `RevokeAllSessions` (plus their `New…Tx` builders) wrapping the `auth` package's three session messages, and a `GetMaster()` method on the `Signer` interface. When a signer has a non-zero `Master`, `SignTx` queries the session account (not the master account) for the account/sequence numbers, `Sign` places the signature in the slot keyed by the master address, and `SignTx` stamps that signature's `SessionAddr` with the session address — exactly the wire shape the ante handler expects ([`auth/ante.go:126-138`](https://github.com/gnolang/gno/blob/7262dc3c5/tm2/pkg/sdk/auth/ante.go#L126-L138) · [↗](../../../../../.worktrees/gno-review-5657/tm2/pkg/sdk/auth/ante.go#L126-L138)). The single-signer flow matches gnokey's `maketx` ([`maketx.go:260-262`](https://github.com/gnolang/gno/blob/7262dc3c5/tm2/pkg/crypto/keys/client/maketx.go#L260-L262) · [↗](../../../../../.worktrees/gno-review-5657/tm2/pkg/crypto/keys/client/maketx.go#L260-L262)) and is verified by three passing integration tests. The two defects below live in code paths the PR's own tests do not exercise: a tx with two distinct signer addresses, and the standalone `Validate` pre-flight check.

## Glossary
- **master account** — the account that owns a session and pays its gas/spend.
- **session account** — a delegated key authorized to sign for the master within limits.
- **SessionAddr** — the per-signature field that tells the ante handler "this slot is signed by session X of the master in this slot".
- **signer set** — `tx.GetSigners()`, the deduplicated list of msg-signer addresses; one signature slot per entry.

## Critical (must fix)
None.

## Warnings (should fix)
- **[panics instead of erroring on a multi-signer session tx]** `client_txs.go:435` — `SignTx` calls `.PubKey.Address()` on every signature slot, but unsigned slots have a nil PubKey, so a session-signed tx with a second signer panics with a nil-pointer dereference.
  <details><summary>details</summary>

  When `Master` is set, `SignTx` loops over `signedTx.Signatures` to find the slot to stamp with `SessionAddr`, matching on `signedTx.Signatures[i].PubKey.Address() == signerInfo.GetAddress()` ([`client_txs.go:432-439`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/client_txs.go#L432-L439) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/client_txs.go#L432-L439)). `Sign` only fills the one slot it produced and leaves the others initialized with `PubKey: nil` ([`signer.go:96-103`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/signer.go#L96-L103) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/signer.go#L96-L103)). A single-signer session tx has exactly one slot, so the common path is fine; a tx whose messages carry two distinct signer addresses has a second, nil-PubKey slot, and `nil.Address()` panics. Confirmed behaviorally: [repro](comment_claude-opus-4-8.md) (`TestSessionSignTxMultiSigner` panics at `client_txs.go:435`). Fix: skip slots with a nil PubKey, or match the slot by the master signer address the way `Sign` does, before reading `.Address()`.
  </details>

- **[Validate always fails for a session signer]** `signer.go:48-49` — `SignerFromKeybase.Validate` builds its probe `MsgCall` with the session address as `Caller`, but `Sign` searches the signer set for the master address, so a correctly configured session signer's `Validate` always returns "not in signer set".
  <details><summary>details</summary>

  `Validate` signs a blank `MsgCall{Caller: caller.GetAddress()}` where `caller` is the session account ([`signer.go:42-57`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/signer.go#L42-L57) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/signer.go#L42-L57)). When `Master` is non-zero, `Sign` sets `addr = s.GetMaster()` and looks for `addr` in the signer set ([`signer.go:121-140`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/signer.go#L121-L140) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/signer.go#L121-L140)). The set here is `[session_address]`, master is absent, so `Sign` returns "not in signer set" and `Validate` errors for every session signer. `Validate` is a user-facing pre-flight check (not called on the send/call hot path, which is why the integration tests pass), so this breaks the API for session users rather than the transaction flow. Confirmed behaviorally: [repro](comment_claude-opus-4-8.md) (`TestSessionSignerValidate` errors). Fix: set the probe `MsgCall.Caller` to the master address when `Master` is non-zero, so the address `Sign` searches for is present in the signer set.
  </details>

## Nits
- `client_txs.go:253` — the `NewCreateSessionTx` doc says "The Creator and SessionKey fields must be set", which is right; the analogous `NewRevokeSessionTx` ([`client_txs.go:303`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/client_txs.go#L303) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/client_txs.go#L303)) repeats the same line correctly, but the three `New…Tx` builders are byte-for-byte identical apart from the message type. A single generic helper parameterized on the message slice would remove ~90 duplicated lines. Optional.

## Missing Tests
- **[multi-signer and Validate paths are untested]** `session_test.go:1` — the three integration tests only cover a single session signer on the happy path; neither the multi-signer `SignTx` slot-matching nor `SignerFromKeybase.Validate` for a session signer is exercised, which is why both Warnings above shipped green.
  <details><summary>details</summary>

  The added [`session_test.go`](https://github.com/gnolang/gno/blob/7262dc3c5/gno.land/pkg/gnoclient/session_test.go) · [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/session_test.go) drives Call/Send/Run with one session signer and asserts the node accepts them — good coverage of the wire shape. It never builds a tx with two signers (the panic path) and never calls `signer.Validate()` (the always-error path). Both gaps are covered by the adversarial artifact in [`tests/session_adversarial_test.go`](tests/session_adversarial_test.go), which fails at this head and should pass once the Warnings are fixed.
  </details>

## Suggestions
None.

## Open questions
- The session feature intersects the known single-master-signature limitation (issue #5731, demonstrated by PR #5793). A session tx that also needs a second, independent signer is exactly the multi-signer case that panics today; whether gnoclient should support that combination at all is a design question, but it should error cleanly rather than panic regardless. Not posted as a separate finding — folded into the Warning.
</content>
</invoke>
