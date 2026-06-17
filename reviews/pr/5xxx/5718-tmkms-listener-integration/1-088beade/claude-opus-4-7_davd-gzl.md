# PR #5718: feat(tm2/privval): tmkms-compat (3/3) — listener integration, config, docs, integration tests, CI

URL: https://github.com/gnolang/gno/pull/5718
Author: clockworkgr | Base: feat/tmkms-compat/02-secret-conn-signer-client | Files: 15 | +1477 -73
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `088beade` (stale — +4 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5718 088beade`

Verdict: REQUEST CHANGES — lint is red on the new file (unused var + non-wrapping `%v`-on-error + gofmt drift in `upstream_config.go`), the dedicated `ci-tmkms-integration.yml` job dies on `tmkms --version` (tmkms 0.15.0's clap CLI rejects the flag) so the integration test never actually runs in CI for this PR, and `docs/validators/tmkms.md` cites a `tmkms init --pubkey-only` invocation that isn't real. Fix those three and the stack lands.

## Summary
PR3 of the tmkms-compat stack. Adds the operator-facing surface: a new `[priv_validator.tmkms_listener]` config block (mutually exclusive with `[remote_signer]`), the listener-stack factory in `NewPrivValidatorFromConfig` (TCPListener/UnixListener → SignerListenerEndpoint → SignerClient → RetrySignerClient), a build-tagged integration test that spawns a real `tmkms 0.15.0` softsign binary and verifies the full PubKey + SignVote × 2 + SignProposal flow, a dedicated CI workflow that builds tmkms from cargo, ADR-003 with the design rationale, and `docs/validators/tmkms.md`. Also reverts two wire-format mistakes from PR2's "phase 6": Timestamp and PartSetHeader are now ALWAYS emitted (year-0001 and `Some(empty)` respectively) because tendermint-rs rejects nil for either — the integration test surfaced both. The only API change widens `NewPrivValidatorFromConfig`'s return type from `*PrivValidator` to `types.PrivValidator`; tm2's only in-tree caller (`node.go:144`) already consumes the interface.

## Glossary
- `SignerListenerEndpoint` — port of cometbft v0.39.1's signer-listener; validator listens, signer dials in.
- `SignerClient` — port of cometbft's signer client; implements `types.PrivValidator` over the endpoint.
- `RetrySignerClient` — wrapper that retries transient errors; passes signer-side refusals (`WrappedRemoteSignerError`) through immediately.
- `connGen` — atomic counter incremented on every conn install; SignerClient re-verifies the signer's pubkey when generation advances.
- `IsEnabled()` — `c != nil && c.ListenAddr != ""`; gates whether tmkms mode is on.

## Fix
Before: tm2's only privval modes were the local file-backed signer and the `[remote_signer]` (gnokms) client. After: a new `[tmkms_listener]` config block enables the upstream-protocol listener path that PR2 built. `NewPrivValidatorFromConfig` ([`tm2/pkg/bft/privval/config.go:145-171`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/config.go#L145-L171) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/config.go#L145-L171)) dispatches on `cfg.TmkmsListener.IsEnabled()`; when enabled, `newTmkmsListenerPrivValidator` ([`tm2/pkg/bft/privval/config.go:178-234`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/config.go#L178-L234) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/config.go#L178-L234)) opens the bound listener (and chmods UDS sockets to `0600`), wraps it with the upstream-compat `TCPListener`/`UnixListener`, runs `endpoint.Start()` via `NewSignerClient`, then blocks in `sc.Init(WaitForConnectionTimeout)` until the signer dials in. The factory does not wrap with `FileState` — tmkms owns HRS authority via its own `consensus.json`. Validation in `TmkmsListenerConfig.ValidateBasic` ([`tm2/pkg/bft/privval/upstream_config.go:116-134`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream_config.go#L116-L134) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream_config.go#L116-L134)) refuses an empty allowlist (fail-open footgun), an unsupported `protocol_version`, an empty `chain_id`, and mutual exclusion with `[remote_signer]`.

## Critical (must fix)

- **[lint red on the new file blocks merge]** [`tm2/pkg/bft/privval/upstream_config.go:26`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream_config.go#L26) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream_config.go#L26) — `errInvalidTmkmsListenAddr` is unused, the `var` block has gofmt-disallowed double-space alignment, and the `ParseAllowlist` error wraps `err` with `%v` instead of `%w`.
  <details><summary>details</summary>

  `main / lint` CI run fails with four findings on this file. (1) `errInvalidTmkmsListenAddr` ([line 26](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream_config.go#L26) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream_config.go#L26)) is declared but never referenced anywhere in the tree — `grep -rn errInvalidTmkmsListenAddr tm2/` returns only the declaration. (2) The `var (` block uses extra spaces for column alignment that gofmt removes; the block currently fails `gofmt -d`. (3) [Line 127](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream_config.go#L127) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream_config.go#L127): `fmt.Errorf("%w: %v", errInvalidTmkmsAllowedPubkeys, err)` should be `"%w: %w"` so `errors.Is(returned, hexDecodeErr)` works (errorlint). Reproduce locally with `golangci-lint run ./tm2/pkg/bft/privval/...`. Fix: delete the unused err sentinel (or use it in `ValidateBasic` to wrap a malformed `listen_addr` parse, which currently can't fail given `osm.ProtocolAndAddress` accepts anything), run `gofmt -w`, and switch the verb.
  </details>

- **[CI workflow dies before the integration test runs]** [`.github/workflows/ci-tmkms-integration.yml:65`](https://github.com/gnolang/gno/blob/088beade/.github/workflows/ci-tmkms-integration.yml#L65) · [↗](../../../../../.worktrees/gno-review-5718/.github/workflows/ci-tmkms-integration.yml#L65) — `tmkms --version` is not a valid invocation in tmkms 0.15.0; clap rejects with `unexpected argument '--version' found`.
  <details><summary>details</summary>

  From the failing run log of `tmkms / softsign / v0.34 handshake`: cargo install completes, then the very next line `tmkms --version` exits with code 2 — `error: unexpected argument '--version' found / Usage: tmkms <COMMAND>`. tmkms 0.15.0's CLI is structured as `tmkms <subcommand>`; the version subcommand is `tmkms version` (no leading dashes). Because this is the smoke test in the install step, the workflow fails before the Go integration test ever runs — so the dedicated job that this PR is built around never actually exercises the wire-compat path in CI for this PR. The integration test itself is well-written and looks correct on inspection; the gate that surfaces a regression to reviewers is what's broken. Fix: replace `tmkms --version` with `tmkms version` (or just `command -v tmkms` if version logging isn't load-bearing).
  </details>

## Warnings (should fix)

- **[docs reference a non-existent tmkms command]** [`docs/validators/tmkms.md:132`](https://github.com/gnolang/gno/blob/088beade/docs/validators/tmkms.md#L132) · [↗](../../../../../.worktrees/gno-review-5718/docs/validators/tmkms.md#L132) — `tmkms init --pubkey-only <path>` is presented as the way to print an existing key's pubkey, but tmkms's `init` subcommand creates a new config tree and doesn't accept `--pubkey-only` against an existing softsign key.
  <details><summary>details</summary>

  An operator following this guide to populate `allowed_kms_pubkeys` will hit `error: unexpected argument '--pubkey-only' found` from clap, then either start digging in tmkms source or paste a wrong placeholder into the allowlist. The actual flow for getting the hex pubkey of a softsign identity is to run tmkms once with the validator config and read the value tmkms logs when establishing the SecretConnection (or compute it from the seed file via `openssl` / a small Go helper). The doc page is otherwise excellent; just this one command is wrong. Fix: either link directly to tmkms's softsign README (which has the working derivation incantation) or ship a short Go helper alongside the operator guide.
  </details>

- **[unused err sentinel suggests a missing listen_addr validation]** [`tm2/pkg/bft/privval/upstream_config.go:26`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream_config.go#L26) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream_config.go#L26) — `errInvalidTmkmsListenAddr` is declared but `ValidateBasic` does no parse on `ListenAddr` before handing it to `osm.ProtocolAndAddress` + `net.Listen`.
  <details><summary>details</summary>

  Today a typo like `listen_addr = "tpc://0.0.0.0:26659"` falls through validation (`osm.ProtocolAndAddress` accepts anything before the `://`) and only surfaces in `net.Listen("tpc", ...)` returning `unknown network tpc` from inside `newTmkmsListenerPrivValidator`. Catching this in `ValidateBasic` lets the operator see "invalid tmkms_listener.listen_addr" instead of buried in node startup logs. Fix: in `ValidateBasic`, check `protocol, _ := osm.ProtocolAndAddress(c.ListenAddr); protocol != "tcp" && protocol != "unix"` and return `errInvalidTmkmsListenAddr`. This also turns the unused-var lint finding above into a used sentinel.
  </details>

- **[mutual-exclusion check could be tightened to catch a misordered config]** [`tm2/pkg/bft/privval/config.go:84-103`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/config.go#L84-L103) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/config.go#L84-L103) — the order is: validate RemoteSigner unconditionally, then validate TmkmsListener if non-nil, then check mutual exclusion. A user who configures both with an invalid `RemoteSigner.AuthorizedKeys` and an enabled tmkms listener sees the RemoteSigner key parse error instead of the mutual-exclusion error.
  <details><summary>details</summary>

  Not a correctness issue — both configs are rejected — but the diagnostic is misleading. An operator who intended to migrate from gnokms to tmkms and forgot to comment out the `[remote_signer]` block won't recognize from "invalid authorized key" that the actual instruction is "drop one of the two blocks". Fix: hoist the mutual-exclusion check ABOVE the per-mode validation so the clearer error wins.
  </details>

## Nits

- [`tm2/pkg/bft/privval/upstream/tmkms_integration_test.go:172-181`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream/tmkms_integration_test.go#L172-L181) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream/tmkms_integration_test.go#L172-L181) — the comment about `StringTracer` panicking on nil Timestamp is now misleading: the wire path was fixed (see translator_pb.go's "Timestamp ALWAYS emitted") so the test isn't dodging that any more, it just uses a populated time for realism. Reword or drop.
- [`tm2/adr/adr-003-tmkms-compat.md:184`](https://github.com/gnolang/gno/blob/088beade/tm2/adr/adr-003-tmkms-compat.md#L184) · [↗](../../../../../.worktrees/gno-review-5718/tm2/adr/adr-003-tmkms-compat.md#L184) — "Wire-format requirements (learned from Phase 7)" still names a "Phase" that the cleanup commit (7e3a23ff) intentionally scrubbed everywhere else. Either keep the phase numbering everywhere or strip it from the ADR too for consistency.
- [`.github/workflows/ci-tmkms-integration.yml:70`](https://github.com/gnolang/gno/blob/088beade/.github/workflows/ci-tmkms-integration.yml#L70) · [↗](../../../../../.worktrees/gno-review-5718/.github/workflows/ci-tmkms-integration.yml#L70) — `-timeout 60s` is tight for cold-CI runs of a tmkms binary that hasn't seen the test fixtures before. The test's own `testWaitForConnection = 20s` plus three roundtrips fits, but a flaky scheduler or slow disk could push past. Bump to 120s — the rest of the workflow's `timeout-minutes: 30` already absorbs it.
- [`docs/validators/tmkms.md:78`](https://github.com/gnolang/gno/blob/088beade/docs/validators/tmkms.md#L78) · [↗](../../../../../.worktrees/gno-review-5718/docs/validators/tmkms.md#L78) — example uses `chain_id = "test4"`; pick a name that doesn't collide with a real testnet handle to avoid an operator copy-pasting it into mainnet config.
- [`tm2/pkg/bft/privval/upstream/translator_pb.go:67-75`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/upstream/translator_pb.go#L67-L75) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/upstream/translator_pb.go#L67-L75) — comment says "tendermint-rs / tmkms" rejects missing Timestamp; tighten to name the actual decoder error (`MissingTimestamp`) for searchability.

## Missing Tests

- **[no test that ValidateBasic catches a malformed `listen_addr`]** [`tm2/pkg/bft/privval/config_test.go:73-105`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/config_test.go#L73-L105) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/config_test.go#L73-L105) — the existing tests cover empty allowlist and bad protocol_version but not a typo in `listen_addr` itself.
  <details><summary>details</summary>

  Pairs with the warning on `errInvalidTmkmsListenAddr` being unused. Once the protocol-prefix check is added to `ValidateBasic`, add a subtest setting `ListenAddr = "tpc://127.0.0.1:0"` and assert `errors.Is(err, errInvalidTmkmsListenAddr)`.
  </details>

- **[no Close() cleanup test on the listener path]** [`tm2/pkg/bft/privval/config_test.go:208-291`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/config_test.go#L208-L291) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/config_test.go#L208-L291) — the UDS-perms and listener-release tests both end with the factory returning an error (Init timed out). There's no test for the happy path where a signer dials in, Init succeeds, then the returned `types.PrivValidator` is `Close()`d and the port comes free.
  <details><summary>details</summary>

  The happy-path teardown is the operator-visible failure mode that matters most — a stuck port across `systemctl restart gnoland` is the kind of bug that surfaces only in production. Could be a focused test in `signer_listener_endpoint_test.go` instead (`bug_fixes_test.go` already covers the Stop-vs-Close distinction at a lower level); a small end-to-end one in `config_test.go` using the in-tree dialer/listener pair would belt-and-suspenders it.
  </details>

## Suggestions

- [`.github/workflows/ci-tmkms-integration.yml:46-58`](https://github.com/gnolang/gno/blob/088beade/.github/workflows/ci-tmkms-integration.yml#L46-L58) · [↗](../../../../../.worktrees/gno-review-5718/.github/workflows/ci-tmkms-integration.yml#L46-L58) — the cache key `tmkms-${{ runner.os }}-v0.15.0` doesn't include the rust-toolchain pin or the cargo lockfile, so a toolchain bump won't invalidate the cache and could ship a binary built against a different `Cargo.lock`. Add `hashFiles('rust-toolchain.toml')` (or pin the toolchain in the workflow and include it in the key) so version bumps automatically force a rebuild.
- [`tm2/pkg/bft/privval/config.go:160-162`](https://github.com/gnolang/gno/blob/088beade/tm2/pkg/bft/privval/config.go#L160-L162) · [↗](../../../../../.worktrees/gno-review-5718/tm2/pkg/bft/privval/config.go#L160-L162) — when `Init` times out, the operator sees `wait for signer: connection timed out` without any hint about how long they actually waited. Threading `cfg.WaitForConnectionTimeout` into the error message (`wait for signer (waited %s): %w`) makes the production-failure mode self-diagnosing.

## Questions for Author

- The integration-test job uses path-filter triggering, but its `secret_connection.go` path watches `tm2/pkg/p2p/conn/secret_connection.go` — the upstream-compat copy lives at `tm2/pkg/bft/privval/upstream/secret_connection.go` (already covered by the `upstream/**` glob), so the chain-p2p path entry is just there as a canary in case someone "fixes" the divergence by editing both. Intentional? Worth a one-line comment if so.
- The ADR says "future tmkms releases may drop v0.34 and force a v0.38 migration; this is a tracked follow-up, not a blocker." Is there a tracking issue you'd like cross-linked from the ADR so it doesn't get lost?
