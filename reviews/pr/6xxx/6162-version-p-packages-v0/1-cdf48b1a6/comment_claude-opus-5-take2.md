# Review: [#6162](https://github.com/gnolang/gno/pull/6162)
Event: COMMENT
Model: claude-opus-5, effort high
Commit: cdf48b1a6 (latest)
Status: second pass, same head. [Review 5177930827](https://github.com/gnolang/gno/pull/6162#pullrequestreview-5177930827) is submitted and takes no new comments, so this posts on its own and repeats none of it.
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/)

## Body

- The info link on each token row, [`grc20factory.gno:168`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L168), points at `/r/demo/grc20factory`, and the realm lives at `r/demo/defi/grc20factory`.

<details><summary>why genesis is the last chance, and the two siblings</summary>

The realm reaches the gnoland1 genesis set, and [`keeper.go:743-744`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/gno.land/pkg/sdk/vm/keeper.go#L743-L744) rejects a re-add of any package that is not `Private`, so the string cannot be corrected at that path afterwards.

Two more `"/r/…"` or `"/p/…"` literals under `examples/gno.land/**` point at paths no package occupies, all three in files the diff edits:

| literal | in | what it should name |
|---|---|---|
| `/r/demo/grc20reg:` | [`grc20reg.gno:158`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L158) | `r/nt/grc20reg/v0`, [already raised](https://github.com/gnolang/gno/pull/6162#discussion_r3988191161) |
| `/r/demo/grc20factory:` | [`grc20factory.gno:168`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L168) | `r/demo/defi/grc20factory` |
| `/p/gnops/valopers` | [`proposal.gno:76`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L76) | `r/gnops/valopers`, and this one is proposal title text rather than a link |

Resolved against the 325 `gnomod.toml` module paths under `examples/`; none changed at the merge base.
</details>

## misc/deployments/gnoland1/packages.gen.txt:29 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/gnoland1/packages.gen.txt#L29) · [↗](../../../../../.worktrees/gno-review-6162/misc/deployments/gnoland1/packages.gen.txt#L29)

Suggestion: `boards2/v1` and `r/sys/namereg/v1` never had a `v0`, the absence that renumbered `r/gov/dao/v3`.

| package | `v0` | in gnoland1 genesis | this PR |
|---|---|---|---|
| [`r/gnoland/boards2/v1`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/gnoland/boards2/v1/gnomod.toml#L1) | never existed | yes | untouched |
| [`r/sys/namereg/v1`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/sys/namereg/v1/gnomod.toml#L1) | never existed | yes | untouched |
| `r/gov/dao/v3` | never existed, `v2` deleted | yes | renumbered to `v0` |

## SKIP examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml:1 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1) · [↗](../../../../../.worktrees/gno-review-6162/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1)

Nit: `grc1155` and its [`grc777`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/grc777/gnomod.toml#L1) sibling become the only two of the 42 `nt` packages carrying no version.

Skipped: [posted already](https://github.com/gnolang/gno/pull/6162#discussion_r3988511699) naming two neighbours, not the namespace; editing it is a separate call.
