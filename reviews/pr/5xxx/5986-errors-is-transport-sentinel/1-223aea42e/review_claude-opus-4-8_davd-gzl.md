# PR [#5986](https://github.com/gnolang/gno/pull/5986): fix(p2p): use errors.Is instead of errors.As for errTransportClosed sentinel

URL: https://github.com/gnolang/gno/pull/5986
Author: ygd58 | Base: master | Files: 2 | +9 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 223aea42e (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5986 223aea42e`

**TL;DR:** The peer-to-peer switch shuts down its accept loop when the network transport reports that it has closed. It recognised that report with the wrong comparison, which also overwrote the shared marker value it was comparing against; this PR swaps in the comparison that only reads.

**Verdict: APPROVE** — the one-line change is the correct comparison and removes a real data race; nothing pins it in the test suite, and one added test comment mis-describes the failure that remains (1 Missing test, 1 Nit).

## Summary

`errTransportClosed` is built by [`tm2/pkg/errors.New`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/errors/errors.go#L147) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/errors/errors.go#L147), whose declared return type is the [`errors.Error`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/errors/errors.go#L138-L143) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/errors/errors.go#L138-L143) interface, so `&errTransportClosed` handed `errors.As` an interface target. `errors.As` then matched the first error in the chain whose dynamic type implements that interface and assigned it back into the package-level variable, turning a comparison into a write. Every `MultiplexSwitch` in a process shares that variable, so two accept loops closing at once write it concurrently while other goroutines read it at [`transport.go:103`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L103) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L103). `errors.Is` compares by identity and writes nothing.

The interface target also widened the match. Every sentinel in that [var block](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L24-L30) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L24-L30) implements `errors.Error`, so `errors.As` returned true for `errDuplicateConnection` and for a `fmt.Errorf`-wrapped `errIncompatibleNodeInfo`, left the sentinel holding that other error, and took the closed-transport branch. That never fired in production: [`MultiplexTransport.Accept`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L97-L108) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L97-L108) can only return `ctx.Err()`, the sentinel itself, or the two `fmt.Errorf` wraps of net errors at [`transport.go:341`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L341) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L341) and [`transport.go:347`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L347) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L347), and neither net error implements `errors.Error`. So the real transport selects the same case before and after, and the fix is a correctness and race fix, not a behavior change on the production path.

## Examples

`errors.As(err, &errTransportClosed)` evaluated in `tm2/pkg/p2p`, target static type `errors.Error`:

| `err` | old `errors.As` | sentinel after | `errors.Is` |
| --- | --- | --- | --- |
| `errTransportClosed` | true | unchanged | true |
| `errDuplicateConnection` | true | `duplicate peer connection` | false |
| `fmt.Errorf("…, %w", errIncompatibleNodeInfo)` | true | `incompatible node info` | false |
| std `errors.New("boom")` | false | unchanged | false |
| `fmt.Errorf("unable to lookup peer IPs, %w", <net error>)` | false | unchanged | false |

## Glossary

- sentinel error: a package-level error value matched by identity with `errors.Is`; a tm2 sentinel's address is an interface target, so `errors.As` matches siblings and overwrites the shared variable.
- flappy test: a tm2 test named `TestFlappy*` gated on `STABILITY_FILTER=flappy`, run only by `tm2/Makefile`'s `_test.flappy` target and never by CI's plain `go test ./...`.

## Fix

Before, [`switch.go:637`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch.go#L637) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch.go#L637) asked whether the accept error could be extracted into the sentinel variable, which is a write. After, it asks whether the error is the sentinel, which is a read. The case sits third in the [switch](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch.go#L631-L645) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch.go#L631-L645), after the nil and context arms, so context errors never reached the old write path either way.

## Benchmarks / Numbers

| Run | `-race` | With `errors.As` | At 223aea42e |
| --- | --- | --- | --- |
| `-run TestFlappyBroadcastTxForPeerStopsWhenPeerStops -count=100` | no | 100/100 pass | — |
| same test, `-count=20` / `-count=50` | yes | data race on run 1 | 50/50 pass |
| full `tm2/pkg/bft/mempool` package, 1 run | yes | — | both Flappy tests fail |
| full `tm2/pkg/bft/mempool` package, 3 runs | no | 2/2 fail (1 run) | 5/6 test instances fail |

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[misdirected follow-up]** [`tm2/pkg/bft/mempool/reactor_test.go:201-207`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/bft/mempool/reactor_test.go#L201-L207) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/bft/mempool/reactor_test.go#L201-L207) — the remaining failure is neither `-race`-only nor intermittent.
  <details><summary>details</summary>

  Measured on the reviewed commit: `STABILITY_FILTER=flappy go test ./tm2/pkg/bft/mempool/` with no `-race` failed 5 of 6 test instances over 3 full-package runs, and one full-package run with `errors.As` restored failed both. The leaked goroutines are the same `MConnection` `sendRoutine`/`recvRoutine` pair the note names, so the diagnosis is right and only the two qualifiers are wrong. The note also points at "the accompanying fix", which resolves to nothing once the branch is squashed into master. Fix: drop the `-race` and "intermittently" qualifiers and name the file and line the fix landed in.
  </details>

## Missing Tests

- **[fix can be reverted invisibly]** [`tm2/pkg/p2p/switch.go:637`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch.go#L637) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch.go#L637) — the accept loop receiving another sentinel from that var block, the one case `errors.Is` and `errors.As` disagree on.
  <details><summary>details</summary>

  [`TestSwitchAcceptLoopTransportClosed`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch_test.go#L896-L923) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch_test.go#L896-L923) feeds the accept loop the sentinel itself, which both comparisons match, and it passes with the line reverted, so the fix can be undone without turning anything red. On a sibling such as `errDuplicateConnection`, `errors.As` matched it, exited the loop, and left the shared variable holding it, while `errors.Is` keeps the loop running. Fix: add [`switch_accept_sentinel_test.go`](tests/switch_accept_sentinel_test.go), which drives the loop with `errDuplicateConnection` and asserts both that the loop survives and that the sentinel still reads `transport is closed`; it fails on the first assertion with the line reverted.
  </details>

## Suggestions

None.

## Verified

- Restoring `errors.As(err, &errTransportClosed)` on [`switch.go:637`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch.go#L637) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch.go#L637) reproduces the data race on the first `-race` run of `TestFlappyBroadcastTxForPeerStopsWhenPeerStops`; both sides of the report are `errors.As` called from two accept-loop goroutines. At 223aea42e the same command is clean over 50 runs.
- The old expression matched by interface, not by identity. With the target's static type being [`errors.Error`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/errors/errors.go#L138-L143) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/errors/errors.go#L138-L143), `errors.As` returned true for `errDuplicateConnection` and for a wrapped `errIncompatibleNodeInfo` and left the sentinel holding each in turn, and false for a std-library error. See the Examples table.
- [`MultiplexTransport.Accept`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L97-L108) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L97-L108) returns only `ctx.Err()`, the sentinel, and the two net-error wraps from [`newMultiplexPeer`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/transport.go#L333-L369) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/transport.go#L333-L369); handshake failures are logged inside the transport's own accept loop and never surface. The switch therefore takes the same case before and after for the real transport.
- No other `errors.As` call in `tm2`, `gno.land`, `gnovm`, or `contribs` targets a shared variable: all 14 other non-test call sites point at a local declared immediately above the call or at a fresh `new(...)`.
- The two `-race` failures left in `tm2/pkg/p2p` are in test code, at [`switch_test.go:452`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch_test.go#L452) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch_test.go#L452) and [`switch_test.go:113`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch_test.go#L113) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch_test.go#L113), and reproduce identically with the fix reverted.
- The `tm2/pkg/bft/mempool` full-package leak reproduces with the fix reverted and without `-race`, so it predates this diff.
- Green at 223aea42e: `go test ./tm2/pkg/p2p/` and `go test ./tm2/pkg/bft/mempool/` in CI shape, plus `-race` `-count=50` on the isolated Flappy test.

## Open questions

- The two `-race` races left in `tm2/pkg/p2p` are both test-code bugs, not switch bugs: [`TestMultiplexSwitch_AcceptLoop`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch_test.go#L428-L482) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch_test.go#L428-L482) writes a plain `bool` from the `Remove` callback and reads it from the test goroutine, and [`TestMultiplexSwitch_Broadcast`](https://github.com/gnolang/gno/blob/223aea42e/tm2/pkg/p2p/switch_test.go#L109-L113) · [↗](../../../../../.worktrees/gno-review-5986/tm2/pkg/p2p/switch_test.go#L109-L113) reassigns `sw.peers` after `OnStart`. Both are cheap fixes and both are in the way of anyone running `tm2/pkg/p2p` under `-race`; out of scope here, so not posted.
- Nothing in the repo runs these mempool tests with `-race`: [`tm2/Makefile`'s `_test.flappy`](https://github.com/gnolang/gno/blob/223aea42e/tm2/Makefile#L49-L53) · [↗](../../../../../.worktrees/gno-review-5986/tm2/Makefile#L49-L53) omits it, and [CI's test step](https://github.com/gnolang/gno/blob/223aea42e/.github/workflows/_ci-go.yml#L124) · [↗](../../../../../.worktrees/gno-review-5986/.github/workflows/_ci-go.yml#L124) runs a plain `go test ./...` with no `STABILITY_FILTER`, so Flappy tests are skipped there entirely. The race was therefore invisible to every run the project performs; worth raising on gnolang/gno#202 as an argument for a `-race` lane, but not this author's job.
- CI has not run the test suite on this PR. The `ci / tm2` workflow is absent from `gh pr checks`, which is the usual first-time-contributor approval gate rather than a code problem, so it is not posted.
