# PR #5775: feat(val-scenarios): run scenarios without docker via a local runtime backend

URL: https://github.com/gnolang/gno/pull/5775
Author: D4ryl00 | Base: master | Files: 17 | +750 -203
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 996467249 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5775 996467249`

**TL;DR:** The `misc/val-scenarios` harness runs gno validator consensus tests, today only by spinning up every node and CLI call in Docker. This PR adds a second backend, `RUNTIME=local`, that runs the same scenarios against native `gnoland`/`gnokey`/`gnogenesis`/`valsignerd` binaries on `127.0.0.1`, behind the same `make` commands, plus speed and reliability fixes (skip example genesis for consensus-only scenarios, free-port picking, a port-bind-race retry). It touches only the test harness, no production code.

**Verdict: APPROVE** — test-harness-only change, all 17 CI scenario jobs green, jefft0 approved, and I reproduced the docker and local backends locally (build + scenario-01 consensus-only + scenario-04 realm/tx + scenario-05 skip). No blocking findings; two minor nits below.

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

`SCENARIO_GENESIS_EXAMPLES=false` is set on exactly the 12 scenarios that touch no realm/tx/governance op (grep over each confirms the only matches are prose comments), and the realm/governance scenarios (04, 06, 12, 13, 17, 18) plus the scenario-10 canary keep the full genesis. Fidelity claim holds.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- `misc/val-scenarios/lib/scenario.sh:145` — the free-port probe is best-effort: the port is claimed at `register_node` time but only bound much later in `start_all_nodes`, so an unrelated process can grab it in the window and the only symptom is a 120s `wait_for_rpc` timeout. This is the exact failure mode the helper exists to reduce, narrowed but not closed. Acknowledged in the helper's own comment and acceptable for a local dev harness; noting it so a future reader knows the residual race is known, not missed. [`lib/scenario.sh:128-150`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L128-L150) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L128)

- `misc/val-scenarios/lib/scenario.sh:1599` — local sentry-IP-rotation stands in by bumping the P2P port by a fixed `+1000`. With the default bases (P2P 26800, control 28080) a 13-node scenario could in principle land a bumped P2P port on top of a control port, but no in-tree scenario rotates a sentry under the local runtime (scenario-05 is the only rotation and it `skip_unless_docker`s), so this is dead in practice. Flagging only so a future local sentry scenario picks a collision-checked port instead of a blind `+1000`. [`lib/scenario.sh:1599`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1599) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1599)

## Missing Tests
None. The harness is itself the test surface; the 17 docker CI scenarios cover the docker path unchanged, and the local path is exercised by the same scenario scripts (verified locally above). CI does not run the local backend, but adding a `make test.local` CI job would duplicate consensus coverage already provided by the docker matrix for no new signal, and is reasonably out of scope.

## Suggestions
- `misc/val-scenarios/lib/scenario.sh:1078-1083` — under the local runtime `start_all_nodes` starts sentries then validators sequentially, each waiting for its own RPC before the next. Docker's path does the same per group, so behavior matches; a future optimization could start the local node processes in parallel and wait afterward, since per-node RPC comes up independently of quorum (already noted in the inline comment). Not worth doing now. [`lib/scenario.sh:1074-1087`](https://github.com/gnolang/gno/blob/996467249/misc/val-scenarios/lib/scenario.sh#L1074-L1087) · [↗](../../../../../.worktrees/gno-review-5775/misc/val-scenarios/lib/scenario.sh#L1074)

## Open questions
- CI exercises only the docker backend; the local backend is verified by contributors and by jefft0's macOS run, not by CI. If the local path regresses it will surface only in local use, not in PR checks. Not posted: adding a local CI job is a deliberate scope/runtime-cost tradeoff for the maintainers, not a defect in this PR.
