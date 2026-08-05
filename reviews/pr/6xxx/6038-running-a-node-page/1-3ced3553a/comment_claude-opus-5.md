# Review: PR [#6038](https://github.com/gnolang/gno/pull/6038)
Event: REQUEST_CHANGES

## Body
No row in the table satisfies both conventions the new [Deployment files](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L19) section states. Repros run at 3ced3553a.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6038-running-a-node-page/1-3ced3553a/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## docs/builders/running-a-node.md:81-82 [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L81-L82)
The one-line installer cannot resolve a version today, so the only executable instruction on the page ends in an error. The `--version <tag>` escape [`misc/install.sh`](https://github.com/gnolang/gno/blob/3ced3553a/misc/install.sh#L294-L298) suggests does not work either, since every release on this repository is `chain/*`. The same binary is reachable through `make install.gnoland`, at [`gno.land/cmd/gnoland/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/gno.land/cmd/gnoland/README.md?plain=1#L18-L22).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6038 -R gnolang/gno
sh misc/install.sh --full --dir /tmp/gno-install-check; echo "exit=$?"
rm -rf /tmp/gno-install-check
```

```
[gno-install] error: no v* release found; pass --version <tag> explicitly (see https://github.com/gnolang/gno/releases)
exit=1
```
</details>

## docs/builders/running-a-node.md:50 [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L50)
Nit: [`misc/deployments/gnoland1`](https://github.com/gnolang/gno/tree/chain/gnoland1/misc/deployments/gnoland1) ships `gen-genesis.sh` and a `make generate` step, not a genesis file. The page this line points at hedges the same claim, at [`gnoland-networks.md:26`](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L26).

## docs/builders/running-a-node.md:72-73 [↗](../../../../../.worktrees/gno-review-6038/docs/builders/running-a-node.md#L72-L73)
Nit: [`misc/deployments/gnoland1/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/misc/deployments/gnoland1/README.md?plain=1#L89) step 5 sends a validator candidate to a Signal group, not Discord. A candidate reading both cannot tell which one is live.

## docs/resources/gnoland-networks.md:8 [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L8)
Nit: [`misc/loop`](https://github.com/gnolang/gno/blob/3ced3553a/misc/loop/README.md?plain=1#L20-L30) holds none of the three things the [Deployment files](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L25-L27) section defines, and Staging is not a network anyone joins. The same link sits in the infrastructure table at [line 44](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L44) with an accurate description.

## docs/resources/gnoland-networks.md:24 [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L24)
This link, and the one in the infrastructure table at [line 43](https://github.com/gnolang/gno/blob/3ced3553a/docs/resources/gnoland-networks.md?plain=1#L43), both resolve to `master`, which is the copy the two bullets between them tell the reader not to use. `master`'s [`misc/deployments/gnoland1/config.toml`](https://github.com/gnolang/gno/blob/master/misc/deployments/gnoland1/config.toml#L150) names one persistent peer against the [`chain/gnoland1`](https://github.com/gnolang/gno/blob/chain/gnoland1/misc/deployments/gnoland1/config.toml#L150) copy's three, and has no `govdao-scripts/`. An operator who follows this link and runs the `cp config.toml gnoland-data/config/config.toml` step at [`misc/deployments/gnoland1/README.md`](https://github.com/gnolang/gno/blob/3ced3553a/misc/deployments/gnoland1/README.md?plain=1#L50) starts against the superseded peer set.

## docs/resources/gnoland-networks.md:25 [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L25)
Nit: two of the directories this section indexes have no `config.toml`: [`topaz.gno.land`](https://github.com/gnolang/gno/tree/chain/topaz/misc/deployments/topaz.gno.land) and [`test12.gno.land`](https://github.com/gnolang/gno/tree/master/misc/deployments/test12.gno.land).

## docs/resources/gnoland-networks.md:31 [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L31)
Nit: `<name>` matches nothing on the Topaz row, whose branch is `chain/topaz`, directory is `topaz.gno.land` and chain ID is `topaz-1`. Betanet lines up, so a reader who checks the first row reads the rule as literal.

## docs/resources/gnoland-networks.md:35-37 [↗](../../../../../.worktrees/gno-review-6038/docs/resources/gnoland-networks.md#L35-L37)
Betanet's release tags carry no binary and no checksum file, so an operator on the row marked current finds none of what this promises. Container images hang off no release tag either; they are published to [`ghcr.io/gnolang/gno`](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/install.md?plain=1#L80).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6038 -R gnolang/gno
for t in chain/gnoland1.0 chain/gnoland1.1 chain/topaz chain/test13; do
  printf '%s: ' "$t"
  gh api "repos/gnolang/gno/releases/tags/$t" --jq '[.assets[].name]|join(" ")'
done
```

```
chain/gnoland1.0: genesis.json gno.wasm root.zip
chain/gnoland1.1: 
chain/topaz: CHECKSUMS.txt genesis.json gnokey_darwin_amd64 gnokey_darwin_arm64 gnokey_linux_amd64 …
chain/test13: CHECKSUMS.txt genesis.json gnokey_darwin_amd64 gnokey_darwin_arm64 gnokey_linux_amd64 …
```
</details>
