# Review: [#5080](https://github.com/gnolang/gno/pull/5080)
Event: COMMENT

## Body
Where this branch stands against master.

Master already has the mechanism: [`checkNamespacePermission`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L481-L484) reads the `sysnames_pkgpath` param and returns nil when it is empty, and [`genesis_params.toml`](https://github.com/gnolang/gno/blob/master/gno.land/genesis/genesis_params.toml#L9) sets it. Three differences are left:

- Default. [`sysNamesPkgDefault`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L37) is `gno.land/r/sys/names` on master, `""` here. Master's [`applyLegacyDefaults`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L538-L540) also rewrites an empty param back to that default, so no chain can turn enforcement off through genesis params today.
- Genesis. This branch skips the check at `ctx.BlockHeight() == 0`. Master does not.
- Realm. This branch deletes `Enable` and `IsEnabled` from `r/sys/names`. Master [kept them and added `paused`, `ProposeSetPaused` and a GovDAO T1 admin](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/sys/names/verifier.gno#L113-L208).

Master has taken 371 commits since the merge base in March, and 8 of the 12 files conflict. [`checkNamespacePermission`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L480) now takes `params Params` and [`callRealmBool`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L421) takes a `chainDomain`, so the call site here needs rewriting rather than merging.
