# PR #5719: fix(gnokms): signer-side HRS double-sign gate + bft/blockchain test un-flap

URL: https://github.com/gnolang/gno/pull/5719
Author: clockworkgr | Base: master | Files: 8 | +668 -43
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5719 5f24e28` (then `gh -R gnolang/gno pr checkout 5719` inside it)

**Verdict: REQUEST CHANGES** — HRS gate is correct and well-tested; the "un-flap `TestBadBlockStopsPeer`" claim is empirically false (test still fails ~10% locally and once in CI on the same `len(peers)==3` assertion at [`reactor_test.go:249`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/bft/blockchain/reactor_test.go#L249) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/bft/blockchain/reactor_test.go#L249)).

## Summary

Two orthogonal changes. The gnokms half adds a signer-side `(height, round, step)` gate ([`state_signer.go`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go)) that refuses any Sign request that does not strictly advance HRS or that returns same-HRS with non-byte-identical bytes — closing the slashing failure mode where a validator-host snapshot restore (or dual-replica misconfig) causes the soft signer to re-sign at a stale height. State is persisted on the signer host before the signature is returned. The bft/blockchain half adds `MakeConnectedPeer` (singular, in `tm2/pkg/internal/p2p/p2p.go`) to attach a 5th peer to an already-running mesh and drops the `Flappy` filter from `TestBadBlockStopsPeer` — but the underlying race at line 249 (`reactorPairs[1]` not yet at 3 peers when `reactorPairs[3]` caught up) is unchanged, and the test still flakes.

## Glossary

- **HRS** — `(Height, Round, Step)`, the consensus message coordinates used to detect double-signs.
- **`CanonicalVote` / `CanonicalProposal`** — amino-encoded tm2 message shapes; the bytes a validator hands to the signer.
- **`FileState`** — persistent HRS+SignBytes+Signature record on disk ([`tm2/pkg/bft/privval/state/state.go`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/bft/privval/state/state.go) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/bft/privval/state/state.go)). Already used validator-side; reused here on the signer.
- **tmkms `consensus.json`** — the equivalent gate in upstream tmkms.

## Fix

Before: `gnokms` delegated all slashing prevention to the validator's `priv_validator_state.json`; a validator that restored a snapshot or ran two replicas would re-issue Sign requests at stale HRS and the signer would happily sign them. After: every Sign call passes through `HRSGuardedSigner.Sign` ([`state_signer.go:117-176`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L117-L176) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L117-L176)) which (1) decodes SignBytes via `classifySignBytes` to extract HRS, (2) `state.CheckHRS` enforces strict monotonicity against the persisted file, (3) same-HRS replays are allowed only when `bytes.Equal(signBytes, state.SignBytes)` (idempotent retransmit), (4) on a strictly-newer HRS the inner signer is called and `state.Update` persists *before* the signature returns — if persistence fails the signature is dropped. The load-bearing constraint is that persistence happens before the signature leaves the process, so a crash between sign and persist cannot create an unrecorded signature. The blockchain half adds `MakeConnectedPeer` ([`tm2/pkg/internal/p2p/p2p.go:176-214`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/internal/p2p/p2p.go#L176-L214) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/internal/p2p/p2p.go#L176-L214)) for attaching a single new peer to an existing mesh, and rewrites `TestBadBlockStopsPeer` to use it instead of the broken `options`-mutation loop.

## Critical (must fix)

- **[un-flap claim doesn't hold — test still flakes ~10%]** [`reactor_test.go:249`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/bft/blockchain/reactor_test.go#L249) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/bft/blockchain/reactor_test.go#L249) — `assert.Equal(t, 3, len(reactorPairs[1].reactor.Switch.Peers().List()))` fails intermittently; the Flappy filter was removed without fixing the underlying race.
  <details><summary>details</summary>

  The PR's last commit "test(bft/blockchain): un-flap TestBadBlockStopsPeer" (5f24e282c) drops `testutils.FilterStability(t, testutils.Flappy)` and renames `TestFlappyBadBlockStopsPeer` → `TestBadBlockStopsPeer`. The setup-fix commit (a90e872ec) only addresses the 5th-peer-attach side of the test (lines 254-272), not the line-249 assertion. CI failure on this PR is exactly this line (run [26403750519](https://github.com/gnolang/gno/actions/runs/26403750519/job/77722056624): `Error: Not equal: expected: 3 actual: 2`). I reproduced locally: in `go test -count=20 -parallel 4 -run TestBadBlockStopsPeer`, 2/20 runs failed with the same `actual: 2`. Same setup, no race fixed.

  The race: after `MakeConnectedPeers` returns, the loop at line 244 waits only on `reactorPairs[3].reactor.pool.IsCaughtUp()` — it does not wait for `reactorPairs[1]`'s peer set to settle. The 4-peer mesh is built async via separate goroutines (`connectPeers` in `tm2/pkg/internal/p2p/p2p.go:158-162`), and only `reactorPairs[3]` is gated. `reactorPairs[1]` can still be at 2 peers when the assertion fires.

  **Repro:** from a local clone of gnolang/gno:
  ```bash
  gh pr checkout 5719 -R gnolang/gno
  cd tm2
  go test -count=20 -parallel 4 -run 'TestBadBlockStopsPeer' ./pkg/bft/blockchain/ 2>&1 | grep -E '(PASS|FAIL)'
  ```

  Fix: either (a) re-add a wait-for-all-peers loop before line 249 (poll all reactorPairs for `len(peers)==3` with a timeout), or (b) restore `FilterStability(Flappy)` until the race is actually fixed. Shipping with the filter removed but the race intact regresses CI signal — every future PR that lands here will see a 5-10% spurious failure rate on this assertion.
  </details>

## Warnings (should fix)

- **[fail-closed on persist error is correct but observability is thin]** [`state_signer.go:166-173`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L166-L173) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L166-L173) — if `state.Update` fails after inner signer succeeded, the signature is dropped and an error is logged, but there's no metric/alert hook.
  <details><summary>details</summary>

  The behavior is correct: inner key has signed bytes for HRS=X, but state file write failed, so the gate cannot prove on the next request whether X was already issued. Dropping the signature is the right call. However, in production this is an "alarming silent partial sign" event — the inner key signed once, the validator got an error, and the operator has no signal beyond a log line. A counter (or at minimum a deliberately-LOUD log level + suggested operator action in the message) would help.

  Fix: bump log level and include a concrete operator-action line, e.g.: `g.logger.Error("hrs-guard: STATE FILE WRITE FAILED — inner key emitted a signature that was dropped; investigate disk before next block. State path: <path>", ...)`. Optionally expose a `state_persist_failures_total` counter via the existing telemetry hooks if the surrounding service has them.
  </details>

- **[no integration test against `RemoteSignerServer`]** [`state_signer_test.go`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer_test.go) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer_test.go) — coverage is all in-process with a `fakeSigner`; the end-to-end "gate fires when called via the privval wire" path is untested.
  <details><summary>details</summary>

  The unit tests are good — monotonic, regressions, idempotent replay, same-HRS-conflict, restart, garbage. But `HRSGuardedSigner` is only useful when wrapped in `NewSignerServer` ([`server.go:124`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/server.go#L124) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/server.go#L124)), and that wiring is silent in tests — `TestRunSignerServer` exercises lifecycle but not actual sign requests. A test that (a) brings up a `RemoteSignerServer` with the guard, (b) connects a client, (c) sends a Sign request, (d) sends a same-HRS-different-bytes request, (e) verifies the second is rejected over the wire — would prove the guard is actually in the request path. Right now the test that the guard is even installed is "we read server.go and it looks right".

  Fix: add a wire-level test in `server_test.go` that drives `RunSignerServer` with the guard and asserts the same-HRS-conflict path returns an error over the privval protocol.
  </details>

- **[`state.String()` access through unexported field bypasses encapsulation]** [`server.go:128`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/server.go#L128) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/server.go#L128) — `guardedSigner.state.String()` reaches into the `state` field directly.
  <details><summary>details</summary>

  Works because both files are in package `common`, but it leaks the internal field name into the call site and means any future refactor (e.g., moving the guard to a sub-package) needs to also add an exported accessor. A `LastHRS() string` (or similar) method on `*HRSGuardedSigner` would be cheaper to maintain and reads better at the log call site.
  </details>

## Nits

- [`state_signer.go:34-35`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L34-L35) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L34-L35) — the `hrs-guard:` prefix appears both as part of `ErrUnparseableSignBytes` / `ErrSameHRSConflict` message strings and as the wrapping prefix at [line 137](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L137) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L137), [line 172](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L172) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L172). Wrapping a `CheckHRS` error produces `hrs-guard: height regression: expected >= 10, got 9` — clean. But `ErrSameHRSConflict` is returned bare, so the message reads `hrs-guard: same HRS with non-identical SignBytes` (good). Consistent, just worth noting the convention.
- [`state_signer.go:93-109`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L93-L109) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L93-L109) — `classifySignBytes` tries `CanonicalVote` first, then `CanonicalProposal`. The comment correctly notes that wire-type at field 4 discriminates, but a malformed vote could still unmarshal "successfully" with garbage if the bytes happen to be amino-compatible — the `vote.Type == PrevoteType || PrecommitType` check is the only real guard. Worth a one-line comment that this is the load-bearing check, not the unmarshal result.
- [`reactor_test.go:274`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/bft/blockchain/reactor_test.go#L274) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/bft/blockchain/reactor_test.go#L274) — `for !X && Y != 0` reads as "while not caught up and have peers"; equivalent to "exit when caught up OR no peers". Pre-existing; not a bug, but a comment would help future readers.
- [`README.md:43`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/README.md#L43) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/README.md#L43) — claim "tmkms-compat path… **shipped**" links to `docs/validators/tmkms.md`; verify that doc actually exists on master at merge time (the source PR description says it lands in #5718, which this PR is independent of). If the doc isn't on master yet, the cross-link will dangle until #5718 merges.

## Missing Tests

- **[wire-level guard activation]** [`server.go:124`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/server.go#L124) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/server.go#L124) — see warning above.
- **[concurrent Sign requests]** [`state_signer.go:117-119`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L117-L119) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L117-L119) — `mu.Lock()` is taken on every call, but no test asserts serialization under concurrent callers. A simple `t.Parallel`-style test launching N goroutines all calling `Sign(sameHRS, diffBytes)` and asserting exactly one succeeds would prove the lock is doing its job. Single-conn serial accept in the surrounding server makes this less critical, but the guard itself documents thread-safety implicitly via `mu` — worth a test.
- **[step-equal-but-SignBytes-nil edge]** [`state.go:97-100`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/bft/privval/state/state.go#L97-L100) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/bft/privval/state/state.go#L97-L100) — `CheckHRS` returns `errNoSignBytes` when HRS matches but `fs.SignBytes == nil`. Hard to reach in practice (only if state file was tampered with), but the guard's behavior in that case (returns the wrapped error, refuses to sign) is correct and untested.

## Suggestions

- [`server.go:128`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/server.go#L128) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/server.go#L128) — log on startup whether the state file was loaded vs freshly created. The README warns that a missing state file on a recovering validator is dangerous ([`README.md:176`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/README.md#L176) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/README.md#L176)), so surfacing the "freshly created — make sure this is a new key" signal at INFO level would help operators catch the recovery footgun.
  <details><summary>details</summary>

  Right now `LoadOrMakeFileState` silently chooses path A vs B. The log line currently says `hrs-guard active state_file=… last={H: 0, R: 0, S: 0}`, which is the same string in both cases. Distinguishing "loaded existing state, last sign at H=…" vs "no state file found; created empty (this is wrong if you're recovering an existing validator)" would close the most likely operator-error gap the README itself flags.
  </details>

- [`state_signer.go:54-72`](https://github.com/gnolang/gno/blob/5f24e28/contribs/gnokms/internal/common/state_signer.go#L54-L72) · [↗](../../../../../.worktrees/gno-review-5719/contribs/gnokms/internal/common/state_signer.go#L54-L72) — consider validating that the state file's recorded pubkey (if any) matches `inner.PubKey()`. The validator-side `NewPrivValidator` does this at [`privval.go:147-150`](https://github.com/gnolang/gno/blob/5f24e28/tm2/pkg/bft/privval/privval.go#L147-L150) · [↗](../../../../../.worktrees/gno-review-5719/tm2/pkg/bft/privval/privval.go#L147-L150) via signature verification. The signer-side could mirror the check to catch key/state mix-ups (different key paired with old state).
  <details><summary>details</summary>

  Scenario: operator rotates the gnokms key but accidentally points at the old `signer_state.json`. The HRS gate would then "protect" using a state file from a different key. Worst case: the new key signs at H=last-of-old-key+1, which is fine for the new key but the state record (SignBytes/Signature) is from the old key. Idempotent-replay path (`bytes.Equal`) would return an old-key signature — guarded against by inner.Sign being called only for new HRS, so old signatures don't escape on this path. Realistically the worst-case is "no extra protection, no extra footgun" — but a startup check that `signer.PubKey().VerifyBytes(state.SignBytes, state.Signature)` (when both are set) would surface the misconfiguration loudly.
  </details>

## Questions for Author

- The PR description Test plan calls out `TestRunSignerServer/listener_not_free` as a pre-existing macOS flake — has the `TestBadBlockStopsPeer` flake recurred on your local runs since the rename, or did the change happen to mask it on whichever machine you verified `STABILITY_FILTER=flappy go test -count=5` on? My count=20 -parallel=4 reproduces ~10%.
- Was the integration-test failure (`params_valset_rotation_throttle.txtar`) on this PR's CI determined to be unrelated? It does not touch the file and the failure scenario is valoper rotation throttle, not signer-related — but worth confirming before relying on the green half of CI.
