# Review: PR [#65](https://github.com/samouraiworld/gnodaokit/pull/65)
Event: REQUEST_CHANGES

## Body
The avl work checks out. Verified on 60c4bf0 against the launched chain: `vm/qpaths gno.land/p/samcrew` returns exactly piechart, tablesort and urlfilter, and `vm/qfile` shows live `p/demo/svg` declaring `Style *avl.Tree` over `p/nt/avl/v0`, whose `Get` is single-value.

The blocker is 15dbc83 underneath, unchanged here and unreviewed on this PR and on [#64](https://github.com/samouraiworld/gnodaokit/pull/64). Two defects ship with it, both inline: a DAO cannot identify a caller from another realm, and an action executed by that realm records its profile write under the caller instead of the DAO. Fixture and output: [tests/](https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/gnodaokit/65-topaz-v2-rename/tests).

The suite passes with both live, because every test drives the DAO from inside its own realm.

[README.md:245](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/README.md?plain=1#L245) still tells integrators to export `DAO`, while this commit unexported it to `localDAO` in all three demos. Exporting it is what makes the `Execute` defect below reachable, and #64's dropped commit is what fixes the line.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnodaokit/65-topaz-v2-rename/2-60c4bf0/review_claude-fable-5.md [↗](review_claude-fable-5.md)

## gno/p/daokit/daokit.gno:17
`Execute` takes its realm from the caller, so a DAO action crosses under whichever realm invoked it and the profile write lands on that caller. The same realm flows through `InstantExecute` into a caller-supplied target DAO, so a parent executing on a sub-DAO overwrites its own profile. It travels to implementers too: `Execute` sits on the `daokit.DAO` and `ActionHandler` interfaces, so anything governance swaps in through `ChangeDAOImplementation` receives the caller's realm rather than inert identity.

<details><summary>repro</summary>

Two realms, one owning the DAO and one calling in, with a recording profile setter. `NewEditProfileHandler` writes with `setter(cross(rlm), k, v)`, so the recorded realm is whichever one was passed to `Execute`. Fixture: https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/gnodaokit/65-topaz-v2-rename/tests

```
setter observed: gno.land/r/test/daorlm|DisplayName=victim-dao
setter observed: gno.land/r/test/daorlm|Bio=d
setter observed: gno.land/r/test/daorlm|Avatar=
setter observed: gno.land/r/test/caller|Bio=written-by-caller
```

The first three are the DAO's own `basedao.New` init. The fourth is an EditProfile action executed by `r/test/caller`.
</details>

## gno/p/basedao/basedao.gno:42
`CallerIDFn` is a bare `func() string` reading `unsafe.PreviousRealm()`, which does not name the immediate caller across these non-crossing methods: a cross-realm caller resolves to the empty string. `AddMember` accepts that empty address as a member without validating it, so one such member authenticates every cross-realm caller. `Execute` here took a realm; `Propose` and `Vote` did not.

<details><summary>repro</summary>

```
CallerID for cross-realm caller = []
```

With one empty-address member registered, `r/test/caller` proposed, voted and executed on a DAO owned by `r/test/daorlm`. Without it the same run panics `caller is not a member`. Fixture: https://github.com/samouraiworld/gno-agent-workspace/tree/main/reviews/gnodaokit/65-topaz-v2-rename/tests
</details>

## gno/p/basedao/README.md:410
The member-only example authenticates with `unsafe.PreviousRealm()` inside a non-crossing `Post`, which names the outermost crossing realm rather than the immediate caller. It is the framework's only worked answer to gating on membership, so the pattern propagates.

## gno/p/daokit/actions.gno:94
Nit: `NewExecuteLambdaHandler` drops its realm into `_` and invokes a `func()` payload, so a lambda action cannot make the cross-realm call the threading was added for. `NewInstantExecuteHandler` below it forwards the realm, and nothing marks lambdas as the non-crossing kind.

## gno/p/basedao/basedao.gno:128
Nit: `New` stores `Realm: unsafe.CurrentRealm()` although it already holds `rlm`. Both agree in the demos, where `New` only runs from `init(cur realm)`, but `DAOPrivate.Realm` feeds the private-extension gate.
