# Review: [#5080](https://github.com/gnolang/gno/pull/5080)
Event: COMMENT

## Body
[AI review, claude-opus-5 xhigh] (not manually verified)
Status: NEEDS DISCUSSION

The VM-param mechanism this branch proposes is on master already, reached from the other side, so what the branch still carries that master does not is two lines and one deletion:

- [`checkNamespacePermission`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L481-L484) returns nil on an empty `sysnames_pkgpath` and [`genesis_params.toml`](https://github.com/gnolang/gno/blob/master/gno.land/genesis/genesis_params.toml#L9) pins the path for gno.land, which is what @thehowl and @mvallenet asked for on this thread.
- The default is still [`gno.land/r/sys/names`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L37), and [`applyLegacyDefaults`](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/params.go#L538-L540) rewrites an explicitly empty param back to it, so a chain cannot turn enforcement off through genesis params at all today. Flipping that constant to `""` is the change here nothing on master supersedes, and it is what gets @moul the gnodev and local environments where anyone can publish.
- The genesis skip at `ctx.BlockHeight() == 0` also has no counterpart: nothing on master waives the namespace check for a genesis transaction.
- `r/sys/names` meanwhile grew a `paused` flag, [`ProposeSetPaused`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/sys/names/verifier.gno#L168-L208) and a GovDAO T1 admin around [`Enable()`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/sys/names/verifier.gno#L113-L123), so deleting `Enable`/`IsEnabled` now takes out one arm of a governance surface rather than a one-shot toggle.
