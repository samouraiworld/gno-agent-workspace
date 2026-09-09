# Giving every stored Gno object its own account address

An explainer for [gnolang/gno#6139](https://github.com/gnolang/gno/pull/6139),
written by claude-opus-5.

## TLDR

Every GRC20 token carries a text identifier that rides in the `token` attribute
of each `Transfer`, `Approval`, `Mint` and `Burn` event. Until this branch its
tail was a number the creating contract chose, so one contract could give two
independent ledgers the same identifier and nobody reading the chain could tell
their events apart. That is
[issue 6026](https://github.com/gnolang/gno/issues/6026).

The branch stops inventing identifiers. The virtual machine already gives every
stored object a serial number of its own, and this branch hashes that number
into a `g1…` account address:
[`DeriveObjectIDCryptoAddr`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/misc.go#L206)
in the machine, the new built-in
[`chain/runtime.ObjectID(v)`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.gno#L21)
for contract code, and
[`Token.ID()`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/token.gno#L145-L147)
returning it. The `id seqid.ID` parameter every caller used to pass is gone from
`grc20.NewToken`.

## Concepts

**A realm is a contract with permanent state.** Its package path starts with
`/r/`, and everything it keeps between transactions lives under that path. A
`/p/` package is a shared library with no state of its own.

**Every stored object already has a two-part serial number.** The machine calls
it an `ObjectID`, and it is `PkgID` plus `NewTime`. `PkgID` names the contract
that owns the object and is set the moment the object is allocated;
[`PkgIDFromPkgPath`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L97)
computes it by hashing the path and then overwriting the top four bits with
policy flags, one of which is
[reserved and always zero](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L94).
`NewTime` is a per-contract counter, not a clock, and it is stamped only when
the object is written out.

**Nothing is written out until a contract call returns.** The stamp happens in
[`assignNewObjectID`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L1986),
which refuses to re-stamp an object that already carries a number, so a serial
that exists never moves. An object the running call just created has `PkgID` and
no `NewTime` yet, and the branch calls that the unstamped window.

**An address is a truncated hash of a text preimage, and the prefix keeps
spaces apart.** A contract's own address is
[`AddressFromPreimage("pkgPath:" + path)`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/misc.go#L199).
The new derivation uses `objectid:` in front of the object's serial for the same
reason: two families that share a 20-byte address space must not be able to
reach each other's values.

## The preimage, and what it hashes to

The whole derivation is one line,
[`misc.go:219`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/misc.go#L219):

```
"objectid:" + PkgID.String() + ":" + strconv.FormatUint(NewTime, 10)
```

`PkgID.String()` is
[`fmt.Sprintf("RID%X", …)`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L74-L76)
over 20 bytes, so it is the fixed string `RID` plus exactly 40 hex characters
and the second colon always falls in the same place. The values below are the
after state, computed by running the branch's own `gnovm/pkg/gnolang` package
against the vectors in
[`1-011afff91/tests/objectid_golden_test.go`](1-011afff91/tests/objectid_golden_test.go):

| contract path | `NewTime` | preimage | address |
| --- | --- | --- | --- |
| `gno.land/r/demo/foo20` | 1 | `objectid:RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80:1` | `g1un39xxhnkdlj46586xvclnsk5xaw04jlrnyape` |
| `gno.land/r/demo/foo20` | 2 | `objectid:RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80:2` | `g1wr733ezlqykpj87fvl3p5n63u9nnq6vyuhju2a` |
| `gno.land/p/demo/tokens/grc20` | 7 | `objectid:RID4FBD19DF645A50B847D72ED838E39F596646CAE1:7` | `g1dmfquaplgaf0wakjhx5ftlnkdntua67j6dtyyj` |
| `chain/runtime` | 4294967296 | `objectid:RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50:4294967296` | `g1x53ple3fg0nghfnx0vu2xddnmd99h5mn5xv3nk` |
| `chain/runtime` | 18446744073709551615 | `objectid:RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50:18446744073709551615` | `g18de5twlh8fpwgll5925uyf3faul4xwmn7uluhp` |

The contract path itself never enters the preimage. Only its 20-byte `PkgID`
does, so a path of any length or byte content produces a preimage of the same
shape.

## Before and after

| | before | after |
| --- | --- | --- |
| `NewToken` signature | `NewToken(name, symbol string, decimals int, id seqid.ID, rlm realm)` | [`NewToken(name, symbol string, decimals int, rlm realm)`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/token.gno#L48) |
| `Token.ID()` | `gno.land/r/demo/defi/foo20.FOO.0000000` | a `g1…` address, [derived from the object](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/token.gno#L145-L147) |
| readable name | inside `ID()` | its own accessor, [`TokenPath()`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/token.gno#L153) |
| the `token` event attribute | the identifier | the readable path |
| the `id` event attribute | absent | the address |
| two ledgers, one contract, one symbol | can share one identifier | [distinct addresses](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/filetests/token_identity_filetest.gno#L3-L8) |
| value before the object is stored | available at construction | `""` |
| registry key | `<contract>.<SYMBOL>` | [`<contract>.<slug>`, the contract's own choice](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L46) |

## Which object the built-in answers for

`chain/runtime.ObjectID(v)` does not read `v`. It asks the machine which stored
object stands behind `v`, through
[`TypedValue.GetFirstObject`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L413),
and derives the address from that object's serial. For a pointer that resolver
returns the
[container the pointer points into](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L415-L416),
and for a slice it returns the backing array.

```
   contract code                     the machine                      the address
  ┌────────────────┐         ┌──────────────────────────┐         ┌──────────────┐
  │ runtime.       │         │ GetFirstObject(v)        │         │ objectid:    │
  │   ObjectID(&x) │── v ───▶│   pointer → its base     │── the ──│   <PkgID>:   │
  │                │         │   slice   → its array    │  stored │   <NewTime>  │
  │                │         │   struct  → itself       │  object │   hashed to  │
  │                │◀── g1…  │   int,nil → panic        │         │   20 bytes   │
  └────────────────┘         └──────────────────────────┘         └──────────────┘
```

The after values below come from running
[`1-011afff91/tests/zz_objectid_alias.gno`](1-011afff91/tests/zz_objectid_alias.gno)
as a filetest under the branch's own `gnovm/pkg/gnolang`:

| the value asked about | address |
| --- | --- |
| `&h`, a package-level struct | `g18808jrvpvzp89efeegj033ajaf7wxh5c8e9ypc` |
| `&h.A`, its first field | `g1xg33mkdhght44wzdccuns7470ude5cvlns8mxz` |
| `&h.B`, its second field | `g1xg33mkdhght44wzdccuns7470ude5cvlns8mxz` |
| `&arr`, a package-level array | `g1pjvjcvsxwxlen779g693gk9t6j9s6lmfpm9akv` |
| `&arr[0]` | `g1g96sx2rwe2jzwzq46hl3u0xr2ed89gkl82dz00` |
| `&arr[1]` | `g1g96sx2rwe2jzwzq46hl3u0xr2ed89gkl82dz00` |

> [!NOTE]
> A value the running call created has no `NewTime` yet, so the built-in
> answers `""` for it. Events are written at
> [`chain.Emit`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L57-L64)
> time and never rewritten, so a `NewToken` event fired before the token is
> stored carries an empty `id` permanently. The branch works around that in
> `grc20reg.Register` by writing the entry through
> [`setEntry(cross(cur), …)`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L55),
> because returning from an explicit cross runs the contract's write-out pass
> and stamps the token before the event goes out.

## Also in the branch

- The registry keys entries by a slug the registering contract picks rather
  than by the token's symbol, and checks provenance against
  [`Token.OrigRealm()`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L42).
- The `register` event gains `token_id` and `token_key`,
  [`grc20reg.gno:57-64`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L57-L64).
- The code generator that links a built-in's Gno declaration to its Go
  implementation now accepts an empty-interface Go parameter,
  [`misc/genstd/mapping.go:139-150`](https://github.com/gnolang/gno/blob/011afff91/misc/genstd/mapping.go#L139-L150),
  which is what lets the new built-in take `v interface{}` on both sides.
- A gas price is registered for the built-in,
  [`native_gas.go:143`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/native_gas.go#L143).
  Without a row there the first call under a real gas meter panics.
- An ADR records the decision and the four rejected alternatives,
  [`gnovm/adr/prxxxx_objectid_derived_ids.md`](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/prxxxx_objectid_derived_ids.md?plain=1#L1).

## Review files

[Review files for this PR](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/pr/6xxx/6139-objectid-derived-ids)
