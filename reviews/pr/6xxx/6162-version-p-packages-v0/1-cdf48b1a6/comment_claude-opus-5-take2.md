# Review: [#6162](https://github.com/gnolang/gno/pull/6162)
Event: COMMENT
Model: claude-opus-5, effort high
Commit: cdf48b1a6 (latest)
Status: second pass over the same head. [Review 5177930827](https://github.com/gnolang/gno/pull/6162#pullrequestreview-5177930827) already went out from this account and carries three inline comments; nothing here repeats it, and GitHub will not accept new comments on a submitted review, so this posts as its own.
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/)

## Body

- [`"/r/demo/grc20factory:"`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L168) renders an info link to a path no package holds, the realm sitting at `r/demo/defi/grc20factory`, and that realm reaches the gnoland1 genesis set so the string freezes with it.

<details><summary>every hardcoded self-link in a deployed realm</summary>

Three `"/r/…"` or `"/p/…"` literals under `examples/gno.land/**` name no package at this head, and the diff edits all three files:

| literal | in | what it should name |
|---|---|---|
| `/r/demo/grc20reg:` | [`grc20reg.gno:158`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/nt/grc20reg/v0/grc20reg.gno#L158) | `r/nt/grc20reg/v0` |
| `/r/demo/grc20factory:` | [`grc20factory.gno:168`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L168) | `r/demo/defi/grc20factory` |
| `/p/gnops/valopers` | [`proposal.gno:76`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L76) | `r/gnops/valopers`, and this one is proposal title text rather than a link |

All three read the same at the merge base. The sweep resolves every such literal against the 325 `gnomod.toml` module paths under `examples/`, counting a prefix match as resolving. The first is [already raised](https://github.com/gnolang/gno/pull/6162#discussion_r3988191161).
</details>

## misc/deployments/gnoland1/packages.gen.txt:29 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/misc/deployments/gnoland1/packages.gen.txt#L29) · [↗](../../../../../.worktrees/gno-review-6162/misc/deployments/gnoland1/packages.gen.txt#L29)

Suggestion: `boards2/v1` keeps a number whose `v0` never existed here, the absence this PR used to renumber `r/gov/dao/v3`.

<details><summary>what the history and the counts show</summary>

`git log --diff-filter=A` over each `gnomod.toml` path: `r/gnoland/boards2/v0` never existed, [`r/sys/namereg/v1`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/gno.land/r/sys/namereg/v1/gnomod.toml#L1) never had a `v0` either, and `r/gov/dao/v1` never existed while `r/gov/dao/v2` was added by [#2581](https://github.com/gnolang/gno/pull/2581) and later deleted. Both surviving `v1` packages reach the gnoland1 genesis set.

The rule this extends is namespace-scoped rather than `/p/`-scoped: [#5220](https://github.com/gnolang/gno/pull/5220) moved the `nt` namespace, `r/nt/*` included, and [`docs/resources/gno-packages.md:98-107`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/docs/resources/gno-packages.md?plain=1#L98-L107) states a version suffix as optional using `r/` paths for its examples. Of the 28 realms the resolved gnoland1 set pulls in, 17 carry no version; `r/gov/dao` is a stable path over `v0/impl` and the other 16 carry none anywhere in their subtree, among them `r/sys/params`, `r/sys/names` and `r/gnoland/wugnot`, each in a namespace the blacklist reserves.
</details>

## SKIP examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml:1 [gh](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1) · [↗](../../../../../.worktrees/gno-review-6162/examples/quarantined/gno.land/p/nt/grc1155/gnomod.toml#L1)

Nit: `grc1155` and its [`grc777`](https://github.com/gnolang/gno/blob/cdf48b1a6a48aa02113e36a1415aca1a88d15e2a/examples/quarantined/gno.land/p/nt/grc777/gnomod.toml#L1) sibling become the only two of the 42 `nt` packages carrying no version.

Skipped: [posted already](https://github.com/gnolang/gno/pull/6162#discussion_r3988511699) in a weaker form naming two neighbours rather than the whole namespace; editing that comment is its own decision.
