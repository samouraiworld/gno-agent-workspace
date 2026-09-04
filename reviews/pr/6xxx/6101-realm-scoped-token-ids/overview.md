# Letting the VM name a GRC20 token instead of its author

An explainer for [gnolang/gno#6101](https://github.com/gnolang/gno/pull/6101),
written by claude-opus-5.

## TLDR

Every GRC20 token carries a text identifier that rides in the `token` attribute
of each `Transfer`, `Approval`, `Mint` and `Burn` event it emits. Until this
branch the tail of that identifier was a number the creating contract chose, so
one contract could hand two independent ledgers the same identifier and nobody
reading the chain could tell their events apart. That is
[issue 6026](https://github.com/gnolang/gno/issues/6026).

The branch takes the number away from the contract and gives it to the virtual
machine. A new built-in,
[`runtime.NewRealmID()`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/chain/runtime/native.go#L45-L60),
returns `<contract path>:<counter>` for the contract the machine is currently
running, and
[`grc20.NewToken`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L46)
stores that string as the token's identifier. The `id` parameter every caller
used to pass is gone from the signature.

## Concepts

**A realm is a contract with permanent state.** Its package path starts with
`/r/`, and everything it stores between transactions lives under that path. A
`/p/` package is a shared library with no state of its own: when a contract
calls into one, the library's code runs against the calling contract's state.

**`Realm.Time` is a per-contract counter, not a clock.** The virtual machine
gives every stored object an identity of the form `<contract>:<n>`, and `n`
comes from this counter, bumped once per object as the transaction is written
out. The counter is saved with the contract and survives restarts. The branch
draws token identifiers from that same counter, so a token identifier and an
object identity can never carry the same number for the same contract.

**Which contract the machine is "currently running" is not always the caller.**
A call into a `/r/` package that is not a crossing call moves the machine onto
that package's state for the duration of the call, described as borrow rule #1
in [`machine.go`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/pkg/gnolang/machine.go#L2545-L2548).
`runtime.NewRealmID()` reads
[`m.Realm`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/chain/runtime/native.go#L49),
so it answers with whichever contract that rule last selected.

**Issuance is off unless the transaction keeps its writes.** The identifier is
only unique because the counter is saved, so a context that throws its writes
away must not hand one out. A new flag on the execution context,
[`RealmIDEnabled`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/internal/execctx/context.go#L46),
carries that decision.

| Execution path | Issuance | Why |
| --- | --- | --- |
| [`AddPackage`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/sdk/vm/keeper.go#L1072-L1074) | on | deployment runs the contract's `init` and keeps the result |
| [`EnablePackage`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/sdk/vm/keeper_inert.go#L287-L289) | on | same, for a contract deployed inert and switched on later |
| [`Call`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/sdk/vm/keeper.go#L1184-L1186) | on | an ordinary transaction |
| [`Run`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/sdk/vm/keeper.go#L1457-L1459) | on | the script is thrown away, the contracts it calls are not |
| [`gno test`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/pkg/test/test.go#L76) | on | so tests build tokens the way a chain does |
| `vm/qeval`, `vm/qrender` | off | a query discards everything it writes |
| the namespace check the deployer runs | off | a read-only callout into `sys/names` |

## Before and after

| | before | after |
| --- | --- | --- |
| signature | `NewToken(name, symbol string, decimals int, id seqid.ID, rlm realm)` | `NewToken(name, symbol string, decimals int, rlm realm)` |
| `foo20`'s identifier | `gno.land/r/demo/defi/foo20.FOO.0000000` | `gno.land/r/demo/defi/foo20:22` |
| two tokens, same symbol, same contract | can share one identifier | never share one |
| symbol inside the identifier | yes | no, it stays metadata |
| the contract that issued it | first component, up to the first `.` | first component, up to the `:` |
| registry key | `<contract>.<symbol>` | unchanged |
| new accessor | none | [`Token.GetOriginRealm()`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L121) |

## Where the number comes from

The counter is already carrying the contract's stored objects when the first
token is minted, so the first identifier is never `:1`, and its value counts
everything the contract persisted before that point. Two deployments of one
contract path, differing only by three declarations no identifier code reads,
were run through the deployment keeper and asked for their first identifier:

| deployed source | first identifier |
| --- | --- |
| the contract alone | `gno.land/r/test/idsource:7` |
| the same contract plus three unused declarations | `gno.land/r/test/idsource:17` |

The same effect shows in the branch's own scripted chains: `foo20` is
`gno.land/r/demo/defi/foo20:22` when the chain loads it at
[genesis](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L26)
and `gno.land/r/demo/defi/foo20:24` when a transaction
[deploys it with one extra file](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/pkg/integration/testdata/grc20_id_persists_cross_realm.txtar#L16-L17).

## What an event reader gets

```
   contract A                       /p/demo/tokens/grc20                the chain
  ┌──────────────┐                 ┌──────────────────────┐            ┌──────────────┐
  │ NewToken(    │──── call ──────▶│ rlm.IsCurrent()      │            │              │
  │   …, cur)    │                 │   ├─ origRealm ◀─────┼── rlm ─────┤ the value    │
  │              │                 │   │   stored on the  │            │ A threaded   │
  │              │                 │   │   token          │            │ in           │
  │              │                 │   └─ id ◀────────────┼── m.Realm ─┤ the contract │
  │              │                 │       emitted in     │            │ the machine  │
  │              │                 │       every event    │            │ is running   │
  └──────────────┘                 └──────────────────────┘            └──────────────┘
```

The two values reaching `NewToken` come from different places. `origRealm` is
the contract whose realm value passed
[`IsCurrent`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/p/demo/tokens/grc20/token.gno#L23-L25);
the identifier is the contract
[`m.Realm`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/stdlibs/chain/runtime/native.go#L59)
names. The registry authorises on the first
([`grc20reg.Register`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40)),
and every event carries the second.

## Also in the branch

- Object finalisation gains a guard against the counter wrapping past its
  maximum, [`realm.go`](https://github.com/gnolang/gno/blob/911e1a57a/gnovm/pkg/gnolang/realm.go#L2030-L2032).
- Thirty-five call sites drop their `seqid` argument, including
  [`foo20`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/demo/defi/foo20/foo20.gno#L22),
  [`wugnot`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L26)
  and [`grc20factory`](https://github.com/gnolang/gno/blob/911e1a57a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L38).
- The `NewToken` event drops its `realm` attribute.
- An ADR records the decision,
  [`gno.land/adr/prxxxx_grc20_realm_ids.md`](https://github.com/gnolang/gno/blob/911e1a57a/gno.land/adr/prxxxx_grc20_realm_ids.md?plain=1#L1).

## Review files

[Review files for this PR](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6101-realm-scoped-token-ids)
