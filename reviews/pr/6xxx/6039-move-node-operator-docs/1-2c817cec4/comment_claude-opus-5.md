# Review: PR [#6039](https://github.com/gnolang/gno/pull/6039)
Event: REQUEST_CHANGES

## Body
[`tm2/adr/adr-003-tmkms-compat.md:262`](https://github.com/gnolang/gno/blob/2c817cec4/tm2/adr/adr-003-tmkms-compat.md?plain=1#L262) still links the guide by its old path, and names it again at [line 117](https://github.com/gnolang/gno/blob/2c817cec4/tm2/adr/adr-003-tmkms-compat.md?plain=1#L117) and [line 237](https://github.com/gnolang/gno/blob/2c817cec4/tm2/adr/adr-003-tmkms-compat.md?plain=1#L237). Nothing catches it: [`docs/Makefile:2`](https://github.com/gnolang/gno/blob/2c817cec4/docs/Makefile#L2) runs the link checker with `-path` set to `docs/`, so `tm2/adr/` is never scanned.

Verified on 2c817cec4: a `gnoland` built from this branch emits the listen addresses, flags, defaults and secrets keys the README quotes.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6039-move-node-operator-docs/1-2c817cec4/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## docs/builders/running-a-node.md:65 [↗](../../../../../.worktrees/gno-review-6039/docs/builders/running-a-node.md#L65)
https://docs.gno.land/validators/tmkms becomes a 404 with this move, and this line names tmkms in prose where [the link that used to reach it](https://github.com/gnolang/gno/blob/3ced3553a/docs/builders/running-a-node.md?plain=1#L68) stood, so nothing on the site points there. The docs site's redirect block [says the old URLs are indexed and linked externally, and to extend it when docs paths move again](https://github.com/gnolang/docs.gno.land/blob/b11d650be/netlify.toml#L22-L24).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6039 -R gnolang/gno
curl -s -o /dev/null -w 'validators/tmkms      -> %{http_code}\n' -L https://docs.gno.land/validators/tmkms
curl -s -o /dev/null -w 'validators/nosuchpage -> %{http_code}\n' -L https://docs.gno.land/validators/nosuchpage
printf 'files left under docs/validators: '; git ls-tree -r --name-only HEAD -- docs/validators | wc -l
```

```
validators/tmkms      -> 200
validators/nosuchpage -> 404
files left under docs/validators: 0
```
</details>

## docs/resources/gnoland-networks.md:35-37 [↗](../../../../../.worktrees/gno-review-6039/docs/resources/gnoland-networks.md#L35-L37)
The bullet promises binaries, container images, `genesis.json` and its checksum on the matching release tag, but `chain/gnoland1.1` carries no assets at all and `chain/gnoland1.0` carries three, none of them a binary or a checksum. Container images hang off no release tag; they are published to `ghcr.io/gnolang/gno`, as [`install.md:80`](https://github.com/gnolang/gno/blob/2c817cec4/docs/builders/install.md?plain=1#L80) already says. The claim holds for `chain/topaz` and `chain/test13`, which carry eighteen assets each.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6039 -R gnolang/gno
gh api repos/gnolang/gno/releases \
  --jq '.[] | "\(.tag_name)\tassets=\(.assets|length)\t\(.assets|map(.name)|join(","))"'
```

```
chain/topaz	assets=18	CHECKSUMS.txt,genesis.json,gnokey_darwin_amd64,…
chain/test13	assets=18	CHECKSUMS.txt,genesis.json,gnokey_darwin_amd64,…
chain/gnoland1.1	assets=0	
chain/test12	assets=1	genesis.json
chain/gnoland1.0	assets=3	genesis.json,gno.wasm,root.zip
```
</details>

## docs/resources/gnoland-networks.md:43-45 [↗](../../../../../.worktrees/gno-review-6039/docs/resources/gnoland-networks.md#L43-L45)
`../../misc/deployments` ships as `/misc/deployments` on docs.gno.land and returns 404, and the two rows beside it do the same. All three were absolute GitHub URLs before this diff, and [`running-a-node.md:84`](https://github.com/gnolang/gno/blob/2c817cec4/docs/builders/running-a-node.md?plain=1#L84) takes the same shape. [`make -C docs lint`](https://github.com/gnolang/gno/blob/2c817cec4/docs/Makefile#L2) passes because the checker only [stats the path on disk](https://github.com/gnolang/gno/blob/2c817cec4/misc/docs/tools/linter/links.go#L80-L81).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6039 -R gnolang/gno
curl -s -L https://docs.gno.land/resources/gno-packages | grep -o 'href="/examples[^"]*"' | head -2
for p in /examples/gno.land/p/nt/avl/v0 /misc/deployments /misc/loop /contribs/tx-archive; do
  printf 'docs.gno.land%s -> ' "$p"
  curl -s -o /dev/null -w '%{http_code}\n' -L "https://docs.gno.land$p"
done
```

```
href="/examples/gno.land/p/nt/avl/v0/README.md"
href="/examples/gno.land/p/nt/avl/v0"
docs.gno.land/examples/gno.land/p/nt/avl/v0 -> 404
docs.gno.land/misc/deployments -> 404
docs.gno.land/misc/loop -> 404
docs.gno.land/contribs/tx-archive -> 404
```
</details>

## gno.land/cmd/gnoland/README.md:26 [↗](../../../../../.worktrees/gno-review-6039/gno.land/cmd/gnoland/README.md#L26)
This command exits 1 with `no v* release found` and installs nothing: [`misc/install.sh`](https://github.com/gnolang/gno/blob/2c817cec4/misc/install.sh#L218-L220) resolves `latest` to the newest non-prerelease tag starting with `v`, and all eight releases the repository publishes are `chain/*`. The `--version <tag>` escape it suggests does not work either, since the `v1.1.0` and `v1.0.0` git tags carry no GitHub release behind them. The defect is in the script and predates this diff; what this adds is a second place sending a reader there, beside [`running-a-node.md:86-87`](https://github.com/gnolang/gno/blob/2c817cec4/docs/builders/running-a-node.md?plain=1#L86-L87).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6039 -R gnolang/gno
curl -fsSL https://raw.githubusercontent.com/gnolang/gno/master/misc/install.sh | sh -s -- --full
echo "exit=$?"
```

```
[gno-install] error: no v* release found; pass --version <tag> explicitly (see https://github.com/gnolang/gno/releases)
exit=1
```
</details>

## gno.land/cmd/gnoland/README.md:45-51 [↗](../../../../../.worktrees/gno-review-6039/gno.land/cmd/gnoland/README.md#L45-L51)
The last line stops at `missing genesis.json`, because nothing in this block writes one. [`gnogenesis`](https://github.com/gnolang/gno/blob/2c817cec4/contribs/gnogenesis/README.md?plain=1#L3), the tool that makes the file, is never named in this README.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6039 -R gnolang/gno
go build -o /tmp/gnoland ./gno.land/cmd/gnoland
mkdir -p /tmp/step && cd /tmp/step
/tmp/gnoland config init
/tmp/gnoland secrets init > /dev/null
/tmp/gnoland start -chainid mychain -genesis genesis.json
ls
cd - && rm -rf /tmp/step /tmp/gnoland
```

```
Default configuration initialized at gnoland-data/config/config.toml
missing genesis.json
gnoland-data
```
</details>

## examples/gno.land/r/gnops/valopers/init.gno:91 [↗](../../../../../.worktrees/gno-review-6039/examples/gno.land/r/gnops/valopers/init.gno#L91)
Suggestion: the realm is in betanet's genesis package list at [`packages.gen.txt:43`](https://github.com/gnolang/gno/blob/2c817cec4/misc/deployments/gnoland1/packages.gen.txt#L43), so https://gno.land/r/gnops/valopers keeps serving `gnops.io/articles/guides/become-testnet-validator/` on the live chain until an on-chain upgrade lands.

## gno.land/cmd/gnoland/README.md:87-89 [↗](../../../../../.worktrees/gno-review-6039/gno.land/cmd/gnoland/README.md#L87-L89)
Suggestion: [PR #6023](https://github.com/gnolang/gno/pull/6023) wires [`config.P2P.Seeds`](https://github.com/gnolang/gno/blob/2c817cec4/tm2/pkg/p2p/config/config.go#L30) into the switch, so this instruction inverts the moment it merges.
