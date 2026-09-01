# Review: [#5080](https://github.com/gnolang/gno/pull/5080)
Event: COMMENT

## Body
Where this branch stands against master today.

Master already carries the mechanism this branch proposed: [`checkNamespacePermission`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L481-L484) reads the `sysnames_pkgpath` VM param and returns nil when it is empty, and [`genesis_params.toml`](https://github.com/gnolang/gno/blob/master/gno.land/genesis/genesis_params.toml#L9) sets it to `gno.land/r/sys/names`. That is the ask this thread ended on, so the two now differ in three places:

- The default. [`sysNamesPkgDefault`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L37) is `gno.land/r/sys/names` on master and `""` here, and master's [`applyLegacyDefaults`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L538-L540) rewrites an explicitly empty param back to that default, so no chain can currently turn enforcement off through genesis params at all.
- Genesis. This branch returns early at `ctx.BlockHeight() == 0`; master runs the check on genesis transactions like any other.
- The realm. This branch deletes `Enable` and `IsEnabled` from `r/sys/names`, where master [kept them and grew a `paused` flag, `ProposeSetPaused` and a GovDAO T1 admin](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/sys/names/verifier.gno#L113-L208) around them.

The conflict is those files plus signature drift: master's [`checkNamespacePermission`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L480) takes `params Params` instead of reading the params itself, and [`callRealmBool`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L421) gained a `chainDomain` argument, so the call site here no longer compiles against it. 8 of the 12 files conflict: `verifier.gno`, `verifier_test.gno`, `keeper.go`, `keeper_test.go`, `params.go`, `testscript_gnoland.go`, `addpkg_namespace.txtar` and `user_journey.txtar`, over the 371 commits master has taken since the merge base in March.
