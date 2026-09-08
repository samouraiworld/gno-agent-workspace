# Review: [#6131](https://github.com/gnolang/gno/pull/6131)
Event: APPROVE

## Body
[`pages/admin.gno:18`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/gnoland/pages/admin.gno#L18) and [`releases_example/example.gno:12`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/quarantined/gno.land/r/demo/releases_example/example.gno#L12) are asserted by no test, each holding the only copy of its address, so a partial sweep past either is silent. Both predate this change and sit outside its scope.

The change itself is right: all 333 changed line pairs are identical once each address maps to one token, apart from three comment lines dropping `gnoland1`, and the new value matches the source it is derived from.

## examples/gno.land/r/gnoland/blog/admin_test.gno:27 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin_test.gno#L27)
Nit: `clearState` runs first in every test in the package and assigns a second hardcoded copy of the address over [`adminAddr`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20), so this realm's suite stays green whatever the source constant says.

```suggestion
	adminAddr = initialAdminAddr
```

<details><summary>repro</summary>

Add `var initialAdminAddr = adminAddr` beside `var cur realm` for the suggestion above to compile.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6131 -R gnolang/gno
export GNOROOT=$PWD
go build -o /tmp/gno ./gnovm/cmd/gno
sed -i '20s/g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf/g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh/' \
  examples/gno.land/r/gnoland/blog/admin.gno
/tmp/gno test -C examples ./gno.land/r/gnoland/blog
git checkout -- examples/gno.land/r/gnoland/blog/admin.gno && rm -f /tmp/gno
```

The realm's admin is wrong and its own suite does not notice:

```
ok      ./gno.land/r/gnoland/blog 	3.16s
```

With the two-line change applied, the same revert fails and names both addresses:

```
invalid render output.
expected "…Published by g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf to Gno.land's blog…"
got      "…Published by g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh to Gno.land's blog…"
failed: "TestPackage"
FAIL    ./gno.land/r/gnoland/blog 	3.17s
```

</details>
