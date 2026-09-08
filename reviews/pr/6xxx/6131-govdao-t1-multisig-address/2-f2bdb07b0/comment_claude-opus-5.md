# Review: [#6131](https://github.com/gnolang/gno/pull/6131)
Event: COMMENT

## Body
The substitution is complete and provably pure: every added line equals its removed line once both addresses map to one token, apart from three comment lines in the `pearl`, `sapphire` and `topaz` builders that drop the word `gnoland1`. The new address decodes to twenty bytes and round-trips, a one-character corruption is rejected, and no hex or base64 form of either address survives anywhere in the tree. Two constants the change touches are pinned by nothing, both of which predate this branch.

## examples/gno.land/r/gnoland/blog/admin_test.gno:27 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin_test.gno#L27) · [↗](../../../../../.worktrees/gno-review-6131/examples/gno.land/r/gnoland/blog/admin_test.gno#L27)
`clearState` runs first in every test in the package and assigns a second hardcoded copy of the address over [`adminAddr`](https://github.com/gnolang/gno/blob/f2bdb07b0/examples/gno.land/r/gnoland/blog/admin.gno#L20), so this realm's suite stays green whatever the source constant says.

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

## misc/deployments/topaz.gno.land/gen-genesis.sh:137 [gh](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L137) · [↗](../../../../../.worktrees/gno-review-6131/misc/deployments/topaz.gno.land/gen-genesis.sh#L137)
Suggestion: `topaz` reads this variable only into the substep label at [`:722`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/topaz.gno.land/gen-genesis.sh#L722) while the caller that ships comes from `meta.json`, so a half-applied swap would log one address and cut genesis with another. [`pearl/gen-genesis.sh:798-800`](https://github.com/gnolang/gno/blob/f2bdb07b0/misc/deployments/pearl.gno.land/gen-genesis.sh#L798-L800) is the comparison that catches it.
