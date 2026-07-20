# Review: PR [#5945](https://github.com/gnolang/gno/pull/5945)
Event: REQUEST_CHANGES

## Body
Rebuilt `genesis.json` from fc4052651 and got sha256 2dd049f973b82858727440df9aff5722cb0b322fd00890f40f2b0688276898ff, the same digest GitHub reports for the `genesis.json` asset on the [chain/topaz release](https://github.com/gnolang/gno/releases/tag/chain%2Ftopaz). The published artifact and a local rebuild are byte-identical. Booting a node on it, the strict replay report is `total: 94, ok: 94, failed: 0`, and the bootstrap transactions all took effect: namespace enforcement is on, `AllowedDAOs` is locked to `gno.land/r/gov/dao/v3/impl`, and both founding valoper profiles resolve.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5945-topaz-testnet-release-candidate/1-fc4052651/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## misc/deployments/topaz.gno.land/gen-genesis.sh:861 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L861)
Critical: the node is declared ready as soon as `auth/accounts` answers, about four seconds before `bank/balances` returns anything, so the measure pass reads every fee payer as zero and credits each on/e the full `INITIAL_BALANCE`. In the produced genesis the nine fee payers hold 8,999,999,775,461,906 ugnot against an actual burn of 224,538,094; seven of those addresses come from third-party [`[addpkg] creator` fields](https://github.com/gnolang/gno/blob/fc4052651/gno.land/pkg/gnoland/genesis.go#L198-L207) in `gnomod.toml` and an eighth is the gnoland1 GovDAO T1 multisig. Fixing this changes the genesis digest, so `CHECKSUMS_DATA` and the sha in [`VALIDATOR.md`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/VALIDATOR.md?plain=1#L45) have to move with it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5945 -R gnolang/gno
cd misc/deployments/topaz.gno.land
./gen-genesis.sh >/tmp/topaz-build.log 2>&1

# 1000000000000000 is INITIAL_BALANCE, the internal over-provisioning float.
# No genesis balance should equal it: fee payers are meant to land at zero.
echo "fee payers credited the float: $(jq -r '.app_state.balances[]' genesis.json | grep -c '=1000000000000000ugnot$')"
grep -c 'All balances zero' /tmp/topaz-build.log
rm -f genesis.json && rm -rf work
```

```
fee payers credited the float: 9
1
```
</details>

## misc/deployments/topaz.gno.land/gen-genesis.sh:895 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L895)
`${r:-0}` returns 0 both when the account is empty and when the node returned no balance at all. Zero is what [the verify pass asserts](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L921), so rerunning the measurement cannot catch a read that never happened, and the build prints `All balances zero — fee-payer costs verified` over untouched floats.

## misc/deployments/topaz.gno.land/gen-genesis.sh:947-949 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L947)
The fee payer set is wider than the deployer and the names admin: the build reports `Found 9 unique creator/caller addresses`, seven of them from `[addpkg] creator` fields in `gnomod.toml` that appear automatically whenever such a package enters `FILTERED_PACKAGES`. Nothing enforces the stated no-overlap property, and [`LoadFromEntries`](https://github.com/gnolang/gno/blob/fc4052651/gno.land/pkg/gnoland/balance.go#L203) assigns into a map without checking for an existing key, so a faucet address that was also a creator would silently overwrite its exact-burn entry. Same claim in [`README.md:21`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/README.md?plain=1#L21) and [`gen-genesis.sh:786`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/gen-genesis.sh#L786).

## misc/deployments/topaz.gno.land/gen-genesis.sh:53 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L53)
Nit: `1783868400` is 2026-07-12 15:00 UTC, already past, so block 1 would carry a backdated timestamp. Moving it changes the genesis digest, so `CHECKSUMS_DATA` and [`VALIDATOR.md:45`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/VALIDATOR.md?plain=1#L45) need the same update.

## misc/deployments/topaz.gno.land/.gitignore:1-5 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/.gitignore#L1)
Nit: both comments describe the abandoned two-phase pipeline. The script is single-phase and moves `genesis.json` straight out of `work/`.

## misc/deployments/topaz.gno.land/govdao-exec.sh:3 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/govdao-exec.sh#L3)
Nit: the usage line says `./govdao`, which is [gnoland1's filename](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/gnoland1/govdao#L3); this file is `govdao-exec.sh`.

## misc/deployments/topaz.gno.land/gen-genesis.sh:105 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L105)
Suggestion: the "~9.2x headroom under int64 max" note holds per faucet, but ten faucets at 1e18 sum to 1e19, above the int64 ceiling `Coin.Amount` uses, and [`Coin.Add`](https://github.com/gnolang/gno/blob/fc4052651/tm2/pkg/std/coin.go#L128-L136) panics on overflow. Nothing on chain sums the supply today, since [`SDKBanker.TotalCoin`](https://github.com/gnolang/gno/blob/fc4052651/gno.land/pkg/sdk/vm/builtins.go#L42-L44) is unimplemented.

## misc/deployments/topaz.gno.land/gen-genesis.sh:82 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L82)
Suggestion: the comment covers the power value, not the set size. With two validators at power 60, either one going down halts the chain, and recovery needs a `r/sys/validators/v3` proposal that itself needs blocks. gnoland1 launched with [seven](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/gnoland1/gen-genesis.sh#L29), so is two deliberate for topaz?

## misc/deployments/topaz.gno.land/gen-genesis.sh:643 [↗](../../../../../.worktrees/gno-review-5945/misc/deployments/topaz.gno.land/gen-genesis.sh#L643)
Suggestion: `-test-dep` takes the set from 79 to 90 packages, pulling in test-only ones like [`r/tests/vm`](https://github.com/gnolang/gno/blob/fc4052651/examples/gno.land/r/tests/vm/tests.gno#L1-L9). gnoland1 uses the same flag, so this is inherited, but neither [`README.md`](https://github.com/gnolang/gno/blob/fc4052651/misc/deployments/topaz.gno.land/README.md?plain=1#L17) nor the PR description says the 90 include test dependencies.

<details><summary>the 11 added packages</summary>

`p/demo/nestedpkg`, `p/jeronimoalbi/expect`, `p/moul/typeutil`, `p/nt/testutils/v0`, `p/nt/uassert/v0`, `p/nt/urequire/v0`, `p/onbloc/diff`, `p/sunspirit/table`, `r/demo/defi/grc20factory`, `r/tests/vm`, `r/tests/vm/subtests`
</details>
