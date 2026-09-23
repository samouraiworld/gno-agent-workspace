# PR [#6230](https://github.com/gnolang/gno/pull/6230): docs: fix realm, caller, and payment imports after the `std` split

Verdict: REQUEST CHANGES, because three example blocks the branch rewrites still fail to type-check, the failure the pull request exists to remove.
Event: REQUEST_CHANGES
Model: deepseek-v4.1-flash, standard review
Commit: 7f309e625
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6230 7f309e625`
Round: 1 finder (claims), no reflector (one bundle, under its floor of two), 4 candidates, the three Warnings run by their finder and rerun by a judge that was not its finder, the Suggestion judged by read; none refuted

## Body

- Suggestion: the branch fixes five pages and stops there; [`docs/resources/gno-security-guide.md:338`](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/docs/resources/gno-security-guide.md?plain=1#L338) still accepts a payment through `banker.OriginSend()`, which [moved out of `chain/banker`](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L64), so code copied from it does not compile.

## docs/resources/effective-gno.md:748 [gh](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/docs/resources/effective-gno.md?plain=1#L748) · [↗](../../../../../.worktrees/gno-review-6230/docs/resources/effective-gno.md#L748) · Warning

Related: `otherrealm.Register(mySafeObject)` passes `mySafeObject` while the same block declares `mySafeObj`, so the copied block does not compile. Rename the argument to `mySafeObj`.

## docs/resources/gno-interrealm-v2.md:737 [gh](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/docs/resources/gno-interrealm-v2.md?plain=1#L737) · [↗](../../../../../.worktrees/gno-review-6230/docs/resources/gno-interrealm-v2.md#L737) · Warning

Related: `realmA.PublicCrossing(cross)` passes the `cross` builtin, whose type is `func(rlm realm) realm`, but the call requires a realm value, so the block does not compile. Pass the current realm value instead of the builtin.

## docs/resources/gno-interrealm.md:932 [gh](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/docs/resources/gno-interrealm.md?plain=1#L932) · [↗](../../../../../.worktrees/gno-review-6230/docs/resources/gno-interrealm.md#L932) · Warning

Related: `AnotherPublic(cross)` passes the `cross` builtin but the call requires a realm value, and `AnotherPublic(cur)` names `cur` although the enclosing `func Public(_ realm)` discards its parameter. Neither call compiles; name the parameter `func Public(cur realm)` and pass the realm value to both calls.

## docs/resources/gno-stdlibs.md:753 [gh](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/docs/resources/gno-stdlibs.md?plain=1#L753) · [↗](../../../../../.worktrees/gno-review-6230/docs/resources/gno-stdlibs.md#L753) · Suggestion

Suggestion: the parameters list offers `BankerTypeReadonly` as a value of `bt`, but `NewBanker` [panics on it and points callers to `NewReadonlyBanker`](https://github.com/gnolang/gno/blob/7f309e62530ebe047c942e27a4ef99df2737f813/gnovm/stdlibs/chain/banker/banker.gno#L118-L120), so the documented call panics. Drop the `BankerTypeReadonly` bullet and name `banker.NewReadonlyBanker()` for read-only access.
