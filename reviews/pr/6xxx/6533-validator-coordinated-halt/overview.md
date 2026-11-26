# Section 6, "Coordinated upgrades", in the mainnet and onyx `VALIDATOR.md`
Written by claude-opus-5-5.
PR: [gnolang/gno#6533](https://github.com/gnolang/gno/pull/6533)

## TLDR

The validator guides explained how to sync a new node across past upgrades, but not what to do when an upgrade is voted while the node is running. The change adds a short section telling operators what to prepare before the agreed stop, how to recognise the stop, and how to switch to the new version safely. Since a stopped node keeps its process running, operators learn that they must stop and restart it themselves.

## The governance halt height

A coordinated upgrade starts with a GovDAO proposal that writes two chain params, `node:p:halt_height` and `node:p:halt_min_version`. On every block the gno.land EndBlocker reads `halt_height`, and when the current block matches it logs `GovDAO halt height reached, will halt after this block` and arms the halt. Every node runs the same check on the same block, which is why every node stops at the same height. The new section tells operators to follow the proposals on `r/gov/dao`, because the network's `UPGRADES.md` only gains the release's row after the halt.

## The config `halt_height`

`config.toml` has its own `halt_height`, meant for halts that cannot go through a vote, such as a devnet or one node. `gnoland start` reads it once at startup and writes it into the same in-memory field the governance param uses, so whichever writes last wins. A future value left in the file therefore makes a node stop at its own height, out of step with the network. The section asks operators to run `gnoland config get halt_height` before the halt and expect `0`. That command reads the file, not the running node: a value changed while the node runs only takes effect at the next start.

## What the halt does to the process

The block at the halt height is committed; the stop fires when the next block begins, as a panic in `BaseApp.BeginBlock`. On a node at the tip the consensus routine catches that panic, so the node stops producing blocks while its process and RPC stay up. A supervisor that restarts on exit, such as systemd `Restart=on-failure`, therefore does not fire, and the section tells the operator to stop the process and wait for it to exit before starting the new one, so two instances never sign at once. A node still catching up through block sync at that moment has no such catch and exits with code 2.

## `halt_min_version` and the switch

After the halt, a binary below `halt_min_version` refuses to start, so only the new release can carry the chain on. The section tells operators to install that release beforehand, verify it against the release's `CHECKSUMS.txt`, keep a `GNOROOT` checkout of the same tag, and start it on the same data directory, where it replays the next block from the block store. The chain resumes once validators holding more than two thirds of the voting power run it.

## Words used here

| Name | What it is |
| --- | --- |
| `node:p:halt_height` | Chain param a GovDAO proposal sets; `0` means no scheduled halt. |
| `node:p:halt_min_version` | Chain param naming the lowest binary version allowed to start after the halt; empty means no gate. |
| `halt_height` in `config.toml` | Per-node halt height, read once at `gnoland start`; default `0`. |
| `gnoland config get` | Prints a key from `./gnoland-data/config/config.toml`, or the file `--config-path` names. |
| `CHECKSUMS.txt` | The release's file of binary checksums, written by the release workflow. |
| `GNOROOT` | Path to a gno source checkout the node loads the standard libraries from. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file.

### 1. `warm` · The new section, [`misc/deployments/mainnet.gno.land/VALIDATOR.md`](https://github.com/gnolang/gno/pull/6533/files#diff-6991da46922c9217820f9c7026814f99700af8a759cbfc8d9c0ebe00a408eaedR154-R161)
Tells a validator what to check before the halt and how to switch after it.
**If wrong:** a validator halts out of step with the network or misses the switch.

### 2. `warm` · The onyx copy, [`misc/deployments/onyx.gno.land/VALIDATOR.md`](https://github.com/gnolang/gno/pull/6533/files#diff-3d0be99ae294dfcedfe60f469908cc0aaf9c653f68eda3fa1cd1f5aee42e2f49R142-R149)
The same text, pointing at the onyx GovDAO realm.
**If wrong:** onyx validators read different guidance from mainnet ones.

</details>
