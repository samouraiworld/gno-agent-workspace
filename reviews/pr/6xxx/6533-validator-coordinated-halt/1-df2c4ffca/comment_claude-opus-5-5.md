# Review: [#6533](https://github.com/gnolang/gno/pull/6533)
Posted: https://github.com/gnolang/gno/pull/6533#pullrequestreview-6080535000
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES, the pre-halt `config get` check passes on a node that will still halt at an old `halt_height`, and section 6 ships that check to every validator.
Model: claude-opus-5-5, quick review
Commit: df2c4ffca (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6533 df2c4ffca`
Round: 1. One finder, one reflector, 5 candidates, the Warning run from scratch by a judge that was not its finder, the rest read.

## Body

The pre-halt check can pass on a node that still halts at the wrong height, so this can't go in yet. The `config get` comment is the blocker; the block-sync and data-directory ones are optional.

## misc/deployments/mainnet.gno.land/VALIDATOR.md:159 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) · [↗](../../../../../.worktrees/gno-review-6533/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) · Warning [posted](https://github.com/gnolang/gno/pull/6533#discussion_r4713412849)

This check passes on a node that will still halt at an old height, since a changed value reaches the running node only at its next start ([`start.go:288-291`](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/gno.land/cmd/gnoland/start.go#L288-L291)).

**Proposed fix:** say that a changed `halt_height` takes effect only after a restart, so a node found with a future value is restarted before that height, in both `VALIDATOR.md` files.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6533 -R gnolang/gno
go build -o "$PWD/.halt-bin/gnoland" ./gno.land/cmd/gnoland
export PATH="$PWD/.halt-bin:$PATH" GNOROOT="$PWD"
D=$(mktemp -d) && cd "$D"
gnoland config init >/dev/null && gnoland config set halt_height 6 >/dev/null
gnoland start --lazy --skip-genesis-sig-verification >node.log 2>&1 & PID=$!
until grep -q 'Halt height configured' node.log; do sleep 1; done
gnoland config set halt_height 0 >/dev/null
echo "config get after clearing: $(gnoland config get halt_height)"
until grep -q 'halt height 6 reached' node.log; do sleep 1; done
grep -m1 -o 'halt height 6 reached[^"]*' node.log
kill "$PID"; wait "$PID" 2>/dev/null; cd - >/dev/null; rm -rf "$D" .halt-bin
```

The check prints `0`, and the node halts at 6 anyway:

```
config get after clearing: 0
halt height 6 reached, node shutting down
```

</details>

## misc/deployments/mainnet.gno.land/VALIDATOR.md:161 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) · [↗](../../../../../.worktrees/gno-review-6533/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) · Nit [posted](https://github.com/gnolang/gno/pull/6533#discussion_r4713412855)

Nit: under block sync, nothing recovers the `BeginBlock` panic below [`ApplyBlock`](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/tm2/pkg/bft/blockchain/reactor.go#L536), so a validator still catching up at the halt exits with code 2 and `Restart=on-failure` does fire.

## misc/deployments/mainnet.gno.land/VALIDATOR.md:161 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) · [↗](../../../../../.worktrees/gno-review-6533/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) · Suggestion [posted](https://github.com/gnolang/gno/pull/6533#discussion_r4713412861)

Suggestion: a release that changes the state format restarts from a fresh genesis rather than the same data directory, per [`UPGRADES.md:13`](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/gno.land/cmd/gnoland/UPGRADES.md?plain=1#L13).

## SKIP misc/deployments/mainnet.gno.land/VALIDATOR.md:159 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) · [↗](../../../../../.worktrees/gno-review-6533/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) · Nit

Nit: `gnoland config get` reads `./gnoland-data/config/config.toml` unless `--config-path` names the node's file, so the check run from another directory reads a different file.

**Fix:** for instance, show the command with `--config-path <your node directory>/gnoland-data/config/config.toml`.

Not posted: whether a missing file fails loudly or a stale one prints `0` was not run this round.

## SKIP misc/deployments/onyx.gno.land/VALIDATOR.md:147 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/onyx.gno.land/VALIDATOR.md#L147) · [↗](../../../../../.worktrees/gno-review-6533/misc/deployments/onyx.gno.land/VALIDATOR.md#L147) · Warning

This line repeats the `config get` check of the mainnet line 159.

**Fix:** the same restart sentence as in the mainnet file.

Not posted: the mainnet section's fix names both files.

## SKIP misc/deployments/onyx.gno.land/VALIDATOR.md:149 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/onyx.gno.land/VALIDATOR.md#L149) · [↗](../../../../../.worktrees/gno-review-6533/misc/deployments/onyx.gno.land/VALIDATOR.md#L149) · Nit

This paragraph repeats the halt text of the mainnet line 161, block-sync exit and data directory included.

**Fix:** the same two edits as in the mainnet file.

Not posted: the mainnet sections' fixes name both files.
