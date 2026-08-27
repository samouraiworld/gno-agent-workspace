# Review: PR [#5917](https://github.com/gnolang/gno/pull/5917)
Event: APPROVE
Status: not posted. Round re-anchored to 134db89ba. On `post as an AI` the Body leads with `[AI review, opus 4.8]`, then `Status: APPROVE`.

## Body
Looks good. Verified on 134db89ba: the valoper instructions block `set-valoper-instructions.sh` embeds is byte-identical to init.gno, except the deliberate relative-to-absolute txlink swap on the Register link, which resolves to the same URL.

- `add-validator-v3.sh` and `rm-validator-v3.sh` next to these still emit the pre-[#5669](https://github.com/gnolang/gno/pull/5669) three-argument `NewValidatorProposalRequest([]ValoperChange{...}, title, desc)` call, which no longer compiles against `r/sys/validators/v3` now that the builder takes `cur realm` first. Update them in this PR, or leave them for a separate cleanup?

## misc/govdao-scripts/set-valoper-instructions.sh:11-12 [gh](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/set-valoper-instructions.sh#L11-L12) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/set-valoper-instructions.sh#L11)
The comment says this branch's `init.gno` still carries the pre-PR text, but [#5842](https://github.com/gnolang/gno/pull/5842) is merged and the branch's [init.gno](https://github.com/gnolang/gno/blob/134db89ba/examples/gno.land/r/gnops/valopers/init.gno#L21) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/gnops/valopers/init.gno#L21) already holds the post-#5842 text. The pre-PR text survives only on the deployed test13 realm, which is what this script updates.

## misc/govdao-scripts/unlock-transfer.sh:9 [gh](https://github.com/gnolang/gno/blob/134db89ba/misc/govdao-scripts/unlock-transfer.sh#L9) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/unlock-transfer.sh#L9)
This points at a `lock-transfer` command to re-lock, but no `lock-transfer` script exists in this directory. An operator who needs to re-lock finds nothing under that name.
