# PR #5775: feat(val-scenarios): run scenarios without docker via a local runtime backend

URL: https://github.com/gnolang/gno/pull/5775
Author: D4ryl00 | Base: master | Files: 17 | +750 -203
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `996467249` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5775 996467249`

> Deep multi-lens re-review of the same commit (red-team, blue-team, correctness, shell-safety, plus a critic and a claim-verification pass). No new commits since round 1; verdict unchanged. The lenses added two non-blocking Warnings and two Nits below; the critic confirmed no false-green path and the claim-verifier confirmed every cited fact.

**TL;DR:** The `misc/val-scenarios` harness runs gno validator consensus tests, today only by spinning up every node and CLI call in Docker. This PR adds a second backend, `RUNTIME=local`, that runs the same scenarios against native `gnoland`/`gnokey`/`gnogenesis`/`valsignerd` binaries on `127.0.0.1`, behind the same `make` commands, plus speed and reliability fixes (skip example genesis for consensus-only scenarios, free-port picking, a port-bind-race retry). It touches only the test harness, no production code.

**Verdict: APPROVE** — test-harness-only change, all 17 CI scenario jobs green, jefft0 approved, and I reproduced the docker and local backends locally (build + scenario-01 consensus-only + scenario-04 realm/tx + scenario-05 skip). No blocking findings. Two non-blocking Warnings worth a glance: the local backend supervises nothing, so a node that crashes at startup is caught only by a 120s RPC timeout, and `RUNTIME` is unvalidated so any typo silently runs the docker path.

## Summary
Adds a runtime seam to `lib/scenario.sh`: every primitive that differs between Docker and native execution (one-shot CLI runs, node/signer lifecycle, peer addressing, port resolution, log capture, reset, IP rotation) branches on `RUNTIME`, while genesis generation and all scenario logic stay shared. The local backend assigns deterministic per-node host ports (base+index, shifting up via `_pick_free_port` when taken) and supervises nodes/signers as background processes. Three orthogonal fixes ride along: consensus-only scenarios set `SCENARIO_GENESIS_EXAMPLES=false` to skip replaying ~347 example packages at every `InitChain` (scenario-08 ~180s→~85s), a `compose_up_one` retry absorbs Docker's ephemeral host-port race on single-service starts, and `gnogenesis` gets `-i` so its piped empty password is not read as EOF. Sentry scenarios (05) become docker-only via `skip_unless_docker`, since nodes sharing `127.0.0.1` cannot be network-isolated.

## Glossary
- **consensus-only scenario** — a scenario whose assertions depend only on the genesis validator set (halt/resume, reset, voting power, signer rules), not on any on-chain realm state.
- **example genesis** — the ~347 packages under `examples/` plus the on-chain PoA valset realm, loaded into genesis and replayed by every node at `InitChain`.
- **controllable signer** — a `valsignerd` sidecar fronting a validator's key, with an HTTP control API to drop/delay signatures.

## Fix
No bug fix in the headline sense: the diff is an additive backend plus three small reliability/speed fixes. The runtime seam lives in [`lib/scenario.sh`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh); each scenario opts into the speed path with one `SCENARIO_GENESIS_EXAMPLES=false` line near its top.

## Verification
Built and ran from the PR worktree at `996467249`:

| Check | Result |
|-------|--------|
| `make build-binaries` (new target) | builds gnoland, gnokey, gnogenesis, valsignerd into `bin/` |
| `make scenario-01.local` (consensus-only, EXAMPLES=false) | pass — 4 validators on 26700-26703, chain advanced, halted below quorum, resumed after reset |
| `make scenario-04.local` (realm/tx, full genesis) | pass — counter realm deployed, txs broadcast, val3 reset/restarted, all 4 synced to h=63 |
| `make scenario-05.local` (sentry) | skipped cleanly, exit 0 |
| `bash -n` on `lib/scenario.sh` + all 18 scenarios | no syntax errors |
| `_pick_free_port` unit test | skips a busy 127.0.0.1 port, dedups within a scenario via `_CLAIMED_PORTS` |
| `gh pr checks` | all 17 scenario jobs + build/lint/e2e green; only red is the merge-requirements bot awaiting a tech-staff review |

`SCENARIO_GENESIS_EXAMPLES=false` is set on exactly the scenarios that touch no realm/tx/governance op (01, 02, 03, 05, 07, 08, 09, 11, 14, 15, 16; grep over each confirms the only other matches are prose comments), and the realm/governance scenarios (04, 06, 12, 13, 17, 18) plus the scenario-10 canary keep the full genesis. Fidelity claim holds.

Deep-pass false-green check (CI-invisible): traced the halt/reset assertions end to end to confirm a local node *process* death cannot masquerade as a legitimate consensus state. The halt assertions are bracketed by mandatory advance re-probes, and the height read at the start of `assert_chain_advances` ([`lib/scenario.sh:1312`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1312) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1312)) runs `curl -fsS` with no `|| 0` fallback under `set -e`, so a dead probe node aborts the scenario rather than passing it. `assert_chain_advances` also requires the node to actively produce new blocks, which a crashed or stuck node cannot fake. No backend-specific false-green path exists.

## Critical (must fix)
None.

## Warnings (should fix)
- **[a crashed local node is caught only by a 120s RPC timeout]** `misc/val-scenarios/lib/scenario.sh:961-975` — the local backend supervises nothing, so a node that dies at startup goes undetected until an unrelated timeout fires.
  <details><summary>details</summary>

  `_local_start_node` launches `gnoland start` in the background, stores `$!` in `NODE_PID[$node]`, and immediately `disown`s it ([`lib/scenario.sh:961-975`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L961-L975) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L961)), so nothing watches the process after that. The only thing standing between a dead node and a generic error is `wait_for_rpc`, which is doing its job correctly: it polls `/status` for up to 120s and times out ([`lib/scenario.sh:882-893`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L882-L893) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L882)). So `wait_for_rpc` is not the defect; the gap is the absence of any liveness check on the disowned process. If gnoland exits immediately (a port `_pick_free_port` thought free but a non-listening process now holds, genesis replay failing at InitChain, a bad gnoroot, a corrupt data-dir after a reset), the loop runs the entire window, multiplied per node, before failing with a timeout that hides the real cause. The docker path needs no equivalent because `compose up` surfaces a non-zero exit and the crash-loop is visible in container state; only the unsupervised local process can die silently. Not a false-green: it does eventually fail. Confirmed behaviorally on `996467249`: the PR's own `wait_for_rpc`, driven against a crashed node (dead PID plus a closed RPC port), consumed its entire timeout and failed without ever consulting `NODE_PID` ([repro](comment_claude-opus-4-8.md)); separately, a freshly built `gnoland start` was observed exiting non-zero in under a second at startup on a genesis/gnoroot error, so the dead-process state is readily reachable. Fix: have the local startup watch `NODE_PID[$node]` (cheapest place is a `kill -0` inside the RPC wait loop) and fail immediately with the node's log path the moment the process is gone.
  </details>

- **[a RUNTIME typo silently runs docker]** `misc/val-scenarios/lib/scenario.sh:20` — `RUNTIME` is never validated, so any value other than exactly `local` falls through to the docker execution path.
  <details><summary>details</summary>

  `RUNTIME="${RUNTIME:-docker}"` and every dispatch is `[ "$RUNTIME" = "local" ]` (take the local path) else docker; there is no guard rejecting unknown values ([`lib/scenario.sh:20`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L20) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L20)). A developer who runs `RUNTIME=lcoal make scenario-01` gets the docker path, not the local run they asked for, and since `require_tools` only adds `docker` to the required set when `RUNTIME=docker` exactly ([`lib/scenario.sh:152-164`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L152-L164) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L152)), the typo path does not even pre-check that docker is present, so the failure is confusing rather than a clear "unknown RUNTIME". Fix: add one guard after line 20, e.g. `case "$RUNTIME" in docker|local) ;; *) die "unknown RUNTIME=$RUNTIME (want docker|local)";; esac`.
  </details>

## Nits
- `misc/val-scenarios/lib/scenario.sh:145` — the free-port probe is best-effort: the port is claimed at `register_node` time but only bound much later in `start_all_nodes`, so an unrelated process can grab it in the window and the only symptom is a 120s `wait_for_rpc` timeout. This is the exact failure mode the helper exists to reduce, narrowed but not closed. Acknowledged in the helper's own comment and acceptable for a local dev harness; noting it so a future reader knows the residual race is known, not missed. [`lib/scenario.sh:128-150`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L128-L150) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L128)

- `misc/val-scenarios/lib/scenario.sh:1599` — local sentry-IP-rotation stands in by bumping the P2P port by a fixed `+1000`. With the default bases (P2P 26800, control 28080) a 13-node scenario could in principle land a bumped P2P port on top of a control port, but no in-tree scenario rotates a sentry under the local runtime (scenario-05 is the only rotation and it `skip_unless_docker`s), so this is dead in practice. Flagging only so a future local sentry scenario picks a collision-checked port instead of a blind `+1000`. [`lib/scenario.sh:1599`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1599) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1599)

- `misc/val-scenarios/lib/scenario.sh:1698-1709` — cleanup is registered as `trap scenario_finish EXIT` only, never `INT TERM`. Bash runs the EXIT trap on Ctrl-C, so the common interactive case is covered, but the gnoland/valsignerd children are `disown`ed, so a SIGTERM (CI step timeout, a parent `make` killed) bypasses the trap and orphans the node processes holding ports 26700+. Fix: `trap scenario_finish INT TERM EXIT` with a double-run guard, or drop the `disown` so the shell reaps the children. [`lib/scenario.sh:1698-1709`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1698-L1709) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1698)

- `misc/val-scenarios/lib/scenario.sh:1155-1167` — `reset_node` writes the reset state literal `{"height":"0","round":"0","step":0}` twice: once natively (local branch) and once byte-for-byte inside the docker `sh -c` string. If the reset contract changes (state format, an added dir to clear) an editor must touch both branches and could update one, drifting the backends silently. Same shape in `safe_reset_node`. Fix: hoist the reset sequence into one `_reset_data_dir <dir>` helper both branches call. [`lib/scenario.sh:1155-1167`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1155-L1167) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1155)

## Missing Tests
None. The harness is itself the test surface; the 17 docker CI scenarios cover the docker path unchanged, and the local path is exercised by the same scenario scripts (verified locally above). CI does not run the local backend, but adding a `make test.local` CI job would duplicate consensus coverage already provided by the docker matrix for no new signal, and is reasonably out of scope.

## Suggestions
- `misc/val-scenarios/lib/scenario.sh:1078-1083` — under the local runtime `start_all_nodes` starts sentries then validators sequentially, each waiting for its own RPC before the next. Docker's path does the same per group, so behavior matches; a future optimization could start the local node processes in parallel and wait afterward, since per-node RPC comes up independently of quorum (already noted in the inline comment). Not worth doing now. [`lib/scenario.sh:1074-1087`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1074-L1087) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1074)

- `.github/workflows/ci-val-scenarios.yml:82` — three `cache-to` lines gain `,ignore-error=true`, making GHA cache-export failures non-fatal on the docker CI build. Benign and arguably an improvement, but it is a docker-CI-path change not called out in the PR's "3 fixes" framing; one line in the description would let reviewers know the build semantics shifted. [`ci-val-scenarios.yml:82`](https://github.com/gnolang/gno/blob/996467249/.github/workflows/ci-val-scenarios.yml#L82) · [↗](../../../../../.worktrees/gno-review-5775/.github/workflows/ci-val-scenarios.yml#L82)

## Open questions
- CI exercises only the docker backend; the local backend (and the Makefile entry point, which CI bypasses by invoking scenario scripts directly) is verified by contributors and by jefft0's macOS run, not by CI. If the local path or the Makefile dispatch regresses it surfaces only in local use, not in PR checks. Not posted: a `make scenario-01.local` smoke job or a `bash -n` + `make -n` lint step would close the gap cheaply, but adding it is a deliberate scope/runtime-cost tradeoff for the maintainers, not a defect in this PR.
- The `-i` in `run_gnogenesis` ([`lib/scenario.sh:433`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L433) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L433)) is the `docker run -i` flag, not a gnogenesis CLI flag; the password-reading `txs add packages --insecure-password-stdin` needs stdin kept open. The code is correct. Not posted: only the PR description's "was broken on master" framing is loose (master's package-add call already carried `-i`); not a code issue.
