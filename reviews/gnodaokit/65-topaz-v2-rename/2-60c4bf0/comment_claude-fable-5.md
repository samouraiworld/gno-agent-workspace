# Review: PR [#65](https://github.com/samouraiworld/gnodaokit/pull/65)
Event: REQUEST_CHANGES

## Body
The avl work is confirmed. Verified on 60c4bf0 against the launched chain rather than the genesis script: `vm/qpaths gno.land/p/samcrew` on `rpc.topaz.testnets.gno.land` returns exactly piechart, tablesort and urlfilter, and `vm/qfile` shows live `p/demo/svg` declaring `Style *avl.Tree` over `p/nt/avl/v0` whose `Get` is single-value. Both surviving changes are required by that genesis, and no `/v2` occurrence remains.

The blocker is the commit underneath. 15dbc83 rewrote caller identity and realm threading across the framework and has no review on this PR or on [#64](https://github.com/samouraiworld/gnodaokit/pull/64). It is byte-identical here, so two defects ship with it. Running two adversarial realms against the branch at fc4052651: a DAO queried from a second realm reports its caller identity as the empty string, and an EditProfile action executed by that realm records the profile write under `gno.land/r/test/caller` while the DAO's own init records `gno.land/r/test/daorlm`. Fixture and output: [tests/](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/gnodaokit/65-topaz-v2-rename/tests).

The suite passes with both live, because every test drives the DAO from inside its own realm and neither defect appears without a second one.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnodaokit/65-topaz-v2-rename/2-60c4bf0/review_claude-fable-5.md [↗](review_claude-fable-5.md)

## gno/p/daokit/daokit.gno:17
`Execute` takes the realm from its caller, so an action crosses under whichever realm invoked the DAO rather than the DAO's own. `NewEditProfileHandler` writes with `setter(cross(rlm), k, v)`, so the profile write lands on the calling realm. The same realm flows through `InstantExecute` into a caller-supplied target DAO, so a parent executing on a sub-DAO overwrites the parent's profile with the sub-DAO's payload.

<details><summary>repro</summary>

Two realms, one owning the DAO and one calling in, plus a recording profile setter. Full fixture: https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/gnodaokit/65-topaz-v2-rename/tests

```
setter observed: gno.land/r/test/daorlm|DisplayName=victim-dao
setter observed: gno.land/r/test/daorlm|Bio=d
setter observed: gno.land/r/test/daorlm|Avatar=
setter observed: gno.land/r/test/caller|Bio=written-by-caller
```

The first three are the DAO's own `basedao.New` init. The fourth is an EditProfile action executed by `r/test/caller`.
</details>

## gno/p/basedao/basedao.gno:42
`Execute` gained a realm here but `Propose` and `Vote` below it did not, so `CallerIDFn` stays a bare `func() string` reading `unsafe.PreviousRealm()`, which is documented to name the outermost-crossing realm rather than the immediate caller in a non-crossing helper. A caller from another realm therefore resolves to the empty string, and `MembersStore.AddMember` stores an empty address as a member without validation. One empty-address member opens propose, vote and execute to every realm.

<details><summary>repro</summary>

```
CallerID seen by the DAO when r/test/probe calls in: []
```

With an empty-address member registered, `r/test/caller` then proposed, voted and executed on the DAO. Without it the same run panics `caller is not a member`. Fixture: https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/gnodaokit/65-topaz-v2-rename/tests
</details>

## gno/p/basedao/README.md:410
The member-only example gates on `unsafe.PreviousRealm()` inside a non-crossing `Post`, so any realm a member touches can post as that member. The next line calls `dao.DAO`, and this branch removed the exported `DAO` variable from every demo realm, so the snippet no longer compiles either.

## Makefile:1
Merging closes [#64](https://github.com/samouraiworld/gnodaokit/pull/64) and reverts its Makefile delta: this branch pins gno 2c7f1abe, the [#64 head](https://github.com/samouraiworld/gnodaokit/commit/523bf58) moved the pin to ba9da8eb, and the deployer's CI is now on fc4052651. Decide which pin the port ships with.

## gno/p/daokit/actions.gno:94
Nit: `NewExecuteLambdaHandler` discards its realm into `_` and invokes a `func()` payload, so a lambda action can never perform the cross-realm call the threading was added for. `NewInstantExecuteHandler` two functions below forwards it, and nothing in the type or doc comment marks lambdas as the non-crossing kind.

## gno/p/basedao/basedao.gno:128
Nit: `New` stores `Realm: unsafe.CurrentRealm()` despite holding an authoritative `rlm realm` that cannot lie. The two agree in the demos because `New` is only called from `init(cur realm)`, so this is latent, but `DAOPrivate.Realm` feeds the private-extension gate.
