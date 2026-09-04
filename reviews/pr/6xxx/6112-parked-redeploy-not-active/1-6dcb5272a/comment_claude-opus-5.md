# Review: [#6112](https://github.com/gnolang/gno/pull/6112)
Event: APPROVE

## Body
The status board records `simulate says the enable would fail: invalid package` for the case this opens up, so a submitter whose live private realm has a redeploy parked reads `pending` and a string carrying no path and no hash. [`enable`](https://github.com/gnolang/gno/blob/6dcb5272a/contribs/gpao/oracle.go#L810) returns `sim.Error` alone, and the ["it changed after review" sentence](https://github.com/gnolang/gno/blob/6dcb5272a/gno.land/pkg/sdk/vm/keeper_inert.go#L120-L123) the description promises stays in `sim.Log`.

<details><summary>the two strings side by side</summary>

Built by putting v1 live and v2 parked on a private realm, then handing the daemon the older v1, which is what a `-start-height` catch-up replays.

```
sim.Error = invalid package
sim.Log   = msg:0,success:false,log:--= Error =--
Data: vm.InvalidPackageError{abciError:vm.abciError{}}
Msg Traces:
    0  gno/gno.land/pkg/sdk/vm/errors.go:104 - the parked source at
       gno.land/r/test/stale is not what was approved (approved cee6a23c...,
       parked daf4644f...); it changed after review
```

`sim.Error` is what the board's reason carries.
</details>
