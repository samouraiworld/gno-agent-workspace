# Review: PR [#65](https://github.com/samouraiworld/gnodaokit/pull/65)
Event: REQUEST_CHANGES

## Body
Confirms the genesis analysis in the [update comment](https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5004757733), checked at the announced launch tip fc4052651 rather than the manifest. Verified on 0612859:
- the curated genesis set from [gen-genesis.sh](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L66)'s own package list, over the [examples tree it builds from](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L421), is exactly 90 packages. The only samcrew entries are piechart, tablesort and urlfilter; `p/samcrew/{daokit,basedao,daocond}` sit under [examples/quarantined](https://github.com/gnolang/gno/tree/fc40526511474e40b8a66419f5ba28255085bc08/examples/quarantined/gno.land/p/samcrew)
- the 6-package suite runs green at fc4052651 both as published and with the `/v2` renames stripped
- every non-samcrew import of this branch is in that genesis set, `r/demo/profile` and `p/demo/svg` included

Recommended shape: drop the 3 module renames and 44 import repoints, keep the avl repoint, the `ntavl` svg-boundary alias and `realmid`. The avl repoint stands on its own: genesis avl `Get` is [single-value](https://github.com/gnolang/gno/blob/fc4052651/examples/gno.land/p/nt/avl/v0/tree.gno#L58) and [svg types `Style` with it](https://github.com/gnolang/gno/blob/fc4052651/examples/gno.land/p/demo/svg/svg.gno#L15). The deployer needs the matching flip: [imports.sh](https://github.com/samouraiworld/samcrew-deployer/blob/9b2b22e/lib/imports.sh#L92) and the `PKG_SUFFIX` plumbing in [deploy.sh](https://github.com/samouraiworld/samcrew-deployer/blob/9b2b22e/projects/gnodaokit/deploy.sh#L63) rewrite to `/v2` today.

topaz.gno.land and rpc.topaz.gno.land still serve 404, so the strip inherits the deployer's existing gate: confirm the launched chain's `build_version` is fc4052651 before the ceremony.

<details><summary>repro: genesis package set</summary>

```bash
# from a local clone of gnolang/gno:
git fetch origin fc40526511474e40b8a66419f5ba28255085bc08
git checkout fc4052651
cd examples
go run ../gnovm/cmd/gno tool deplist -test-dep \
  ./gno.land/r/sys/... ./gno.land/r/gov/... ./gno.land/r/gnoland/blog/... \
  ./gno.land/r/gnoland/wugnot/... ./gno.land/r/gnoland/coins/... \
  ./gno.land/r/gnoland/boards2/... ./gno.land/r/gnops/valopers/... \
  ./gno.land/p/onbloc/uint256 ./gno.land/p/onbloc/int256 ./gno.land/p/onbloc/json \
  ./gno.land/r/sys/validators/v3 ./gno.land/r/demo/defi/grc20reg > /tmp/topaz-pkgs.txt
wc -l < /tmp/topaz-pkgs.txt
grep -cE "samcrew/(daokit|basedao|daocond|realmid)|quarantined" /tmp/topaz-pkgs.txt
grep -oE "p/samcrew/[a-z]+" /tmp/topaz-pkgs.txt | sort -u
rm /tmp/topaz-pkgs.txt; git checkout -
```

```
90
0
p/samcrew/piechart
p/samcrew/tablesort
p/samcrew/urlfilter
```
</details>

<details><summary>repro: suite at the launch tip, both variants</summary>

```bash
# self-contained; clones gno at the launch tip, gnodaokit PR 65 and samcrew-deployer:
git clone --filter=blob:none https://github.com/gnolang/gno /tmp/gno-topaz
git -C /tmp/gno-topaz fetch origin fc40526511474e40b8a66419f5ba28255085bc08
git -C /tmp/gno-topaz checkout fc4052651
git clone https://github.com/samouraiworld/gnodaokit /tmp/gdk
git -C /tmp/gdk fetch origin pull/65/head && git -C /tmp/gdk checkout FETCH_HEAD
git clone --depth 1 https://github.com/samouraiworld/samcrew-deployer /tmp/scd
EX=/tmp/gno-topaz/examples/gno.land
mkdir -p $EX/p/samcrew/daocond $EX/p/samcrew/daokit $EX/p/samcrew/basedao $EX/r/samcrew/daodemo
cp -r /tmp/scd/deps/avl        $EX/p/samcrew/avl
cp -r /tmp/gdk/gno/p/realmid   $EX/p/samcrew/realmid
cp -r /tmp/gdk/gno/p/daocond   $EX/p/samcrew/daocond/v2
cp -r /tmp/gdk/gno/p/daokit    $EX/p/samcrew/daokit/v2
cp -r /tmp/gdk/gno/p/basedao   $EX/p/samcrew/basedao/v2
for d in simple_dao custom_condition custom_resource; do cp -r /tmp/gdk/gno/r/daodemo/$d $EX/r/samcrew/daodemo/$d; done
cd /tmp/gno-topaz/examples
export GNOROOT=/tmp/gno-topaz
for p in p/samcrew/daocond/v2 p/samcrew/daokit/v2 p/samcrew/basedao/v2 \
         r/samcrew/daodemo/simple_dao r/samcrew/daodemo/custom_condition r/samcrew/daodemo/custom_resource; do
  go run ../gnovm/cmd/gno test ./gno.land/$p
done
# variant with the /v2 renames stripped:
rm -rf $EX/p/samcrew/daocond $EX/p/samcrew/daokit $EX/p/samcrew/basedao $EX/r/samcrew/daodemo
cp -r /tmp/gdk/gno/p/daocond $EX/p/samcrew/daocond
cp -r /tmp/gdk/gno/p/daokit  $EX/p/samcrew/daokit
cp -r /tmp/gdk/gno/p/basedao $EX/p/samcrew/basedao
mkdir -p $EX/r/samcrew/daodemo
for d in simple_dao custom_condition custom_resource; do cp -r /tmp/gdk/gno/r/daodemo/$d $EX/r/samcrew/daodemo/$d; done
find $EX/p/samcrew/daocond $EX/p/samcrew/daokit $EX/p/samcrew/basedao $EX/r/samcrew/daodemo \
  -type f \( -name '*.gno' -o -name '*.toml' \) \
  -exec sed -i 's#/samcrew/daocond/v2#/samcrew/daocond#g; s#/samcrew/daokit/v2#/samcrew/daokit#g; s#/samcrew/basedao/v2#/samcrew/basedao#g' {} +
for p in p/samcrew/daocond p/samcrew/daokit p/samcrew/basedao \
         r/samcrew/daodemo/simple_dao r/samcrew/daodemo/custom_condition r/samcrew/daodemo/custom_resource; do
  go run ../gnovm/cmd/gno test ./gno.land/$p
done
rm -rf /tmp/gno-topaz /tmp/gdk /tmp/scd
```

```
ok      ./gno.land/p/samcrew/daocond/v2 	0.94s
ok      ./gno.land/p/samcrew/daokit/v2 	0.91s
ok      ./gno.land/p/samcrew/basedao/v2 	1.08s
ok      ./gno.land/r/samcrew/daodemo/simple_dao 	1.12s
ok      ./gno.land/r/samcrew/daodemo/custom_condition 	1.08s
ok      ./gno.land/r/samcrew/daodemo/custom_resource 	1.07s
# stripped variant:
ok      ./gno.land/p/samcrew/daocond 	0.96s
ok      ./gno.land/p/samcrew/daokit 	0.90s
ok      ./gno.land/p/samcrew/basedao 	1.12s
ok      ./gno.land/r/samcrew/daodemo/simple_dao 	1.07s
ok      ./gno.land/r/samcrew/daodemo/custom_condition 	1.10s
ok      ./gno.land/r/samcrew/daodemo/custom_resource 	1.10s
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnodaokit/65-topaz-v2-rename/review_claude-fable-5.md [↗](review_claude-fable-5.md)

## Makefile:1
Merging closes [#64](https://github.com/samouraiworld/gnodaokit/pull/64) and reverts its Makefile delta: this branch pins gno 2c7f1abe, the [#64 head](https://github.com/samouraiworld/gnodaokit/commit/523bf58) moved the pin to ba9da8eb, and the deployer's CI is now on fc4052651. Decide which pin the port ships with.

## SKIP gno/p/daokit/gnomod.toml:1
Already raised: https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5004757733
The `/v2` renames target packages absent from the topaz-1 genesis. Confirmation at the launch tip and repro in the Body.
