# Review: PR [#5917](https://github.com/gnolang/gno/pull/5917)
Event: APPROVE
Status: not posted. Round re-anchored to 134db89ba. On `post as an AI` the Body leads with `[AI review, opus 4.8]`, then `Status: APPROVE`.

## Body
Looks good.

- The [`add-validators-v3.sh`](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/add-validators-v3.sh#L77-L78) this PR adds passes `cross(cur)` first, while the two older siblings beside it, [`add-validator-v3.sh`](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/add-validator-v3.sh#L57-L58) and [`rm-validator-v3.sh`](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/rm-validator-v3.sh#L46-L47), still open on `[]valv3.ValoperChange{`. That form no longer type-checks against [the builder](https://github.com/gnolang/gno/blob/134db89ba/examples/gno.land/r/sys/validators/v3/proposal.gno#L69), so both scripts fail before they reach the chain. They belong in this PR, which is where the govdao scripts are being brought current.

<details><summary>the type-check error</summary>

Both older scripts emit the same call shape into a `package main` body. Type-checking it against the current `r/sys/validators/v3`:

```
add_validator.gno:15:2: not enough arguments in call to valv3.NewValidatorProposalRequest
	have ([]"gno.land/r/sys/validators/v3".ValoperChange, string, string)
	want ("gno.land/r/sys/validators/v3".realm, []"gno.land/r/sys/validators/v3".ValoperChange, string, string) (code=gnoTypeCheckError)
```

Adding `cross(cur)` as the first argument, the way `add-validators-v3.sh` already does, lints clean.
</details>

<details><summary>what was checked</summary>

The valoper instructions block `set-valoper-instructions.sh` embeds is byte-identical to `init.gno`, apart from the relative-to-absolute txlink swap on the Register link, which resolves to the same URL.
</details>

## SKIP misc/govdao-scripts/set-valoper-instructions.sh:11-12 [gh](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/set-valoper-instructions.sh#L11-L12) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/set-valoper-instructions.sh#L11)
Already raised: https://github.com/gnolang/gno/pull/5917#discussion_r3560467440
The comment says this branch's `init.gno` still carries the pre-PR text, but [#5842](https://github.com/gnolang/gno/pull/5842) is merged and the branch's [init.gno](https://github.com/gnolang/gno/blob/134db89ba/examples/gno.land/r/gnops/valopers/init.gno#L21) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/gnops/valopers/init.gno#L21) already holds the post-#5842 text. The pre-PR text survives only on the deployed test13 realm, which is what this script updates.

## misc/govdao-scripts/unlock-transfer.sh:9 [gh](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/unlock-transfer.sh#L9) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/unlock-transfer.sh#L9)
This points at a `lock-transfer` command to re-lock, but no `lock-transfer` script exists in this directory. An operator who needs to re-lock finds nothing under that name.
