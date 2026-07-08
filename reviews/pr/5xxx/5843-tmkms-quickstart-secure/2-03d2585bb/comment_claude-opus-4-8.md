# Review: PR [#5843](https://github.com/gnolang/gno/pull/5843)
Event: REQUEST_CHANGES

## Body
Verified on 03d2585bb: driving Part C's exact three-field `unix://` listener config plus its `gnoland start -lazy` aborts with `-lazy cannot derive a genesis in tmkms_listener mode`; the committed guard test drives the `tcp://` shape only, so this UDS case is not covered by CI. Re-checked the delta against the code: the UDS allowlist and `secret_key` removal, the per-validator note, and the Part E softsign+TCP scoping all match master.

The red docs check is an unrelated live-link lint flagging `https://docs.gno.land/` in `docs/MANIFESTO.md`, a file this PR does not touch.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5843-tmkms-quickstart-secure/2-03d2585bb/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/validators/tmkms-quickstart.md:721-775 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L721)
Part C starts with `-lazy` and no pre-existing genesis while the tmkms listener is enabled, which master refuses: `gnoland start` aborts with `-lazy cannot derive a genesis in tmkms_listener mode`. The guard keys on whether the listener is enabled, not on where the key lives, so the softsign key on disk still hits it. Parts B and D build genesis with gnogenesis and omit `-lazy`; the troubleshooting row at line 1048 and the note at line 774 carry the same assumption.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5843 -R gnolang/gno
TMPD=$(mktemp -d)
go run ./gno.land/cmd/gnoland config init -config-path "$TMPD/config/config.toml"
# enable the listener exactly as C.3 does (three fields, empty allowlist, listen_addr last):
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.chain_id gno-tmkms-test
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.protocol_version v0.34
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.listen_addr "unix://$TMPD/privval.sock"
# Part C's final command:
GNOROOT=$(pwd) go run ./gno.land/cmd/gnoland start -data-dir "$TMPD" -genesis "$TMPD/genesis.json" -chainid gno-tmkms-test -lazy -skip-genesis-sig-verification 2>&1 | grep -i lazy
rm -rf "$TMPD"
```

```
-lazy cannot derive a genesis in tmkms_listener mode: tmkms holds the validator key and signs only votes/proposals, so the validator pubkey is not locally available. Provide an explicit genesis.json (see the gnogenesis tool) and start without -lazy
```
</details>

## docs/validators/tmkms-quickstart.md:245-248 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L245)
A.3 calls the `0600` socket the whole auth boundary, but gnoland's chmod on the socket is best-effort: on a filesystem that ignores chmod on a socket it logs a warning and leaves default perms. The real guard is the `700 gnoland` parent dir you already create; say so, or a reader who relocates the socket loses the protection.

## docs/validators/tmkms-quickstart.md:160 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L160)
The optional cleanup line uses brace expansion (`{nodeid-hex,tmkms-identity-keygen}`), which is bash/zsh-only. A reader pasting it into `sh`/dash deletes nothing.

## docs/validators/tmkms-quickstart.md:456-458 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L456)
The production path ends at "submit the gpub/address to the chain's validator-onboarding path" with no concrete mechanism, while the lab paths are fully runnable. A pointer to where validator onboarding is documented, or a note that it isn't yet, would close the loop.

## docs/validators/tmkms-quickstart.md:493-511 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L493)
The `tmkms.toml` heredoc is unquoted, so the shell bakes in `${CHAIN_ID}` and `${TMKMS_ADDR}` at write time. A re-runner who only re-exports `$CHAIN_ID` keeps the stale `chain_id` and stalls on mismatch; one line saying the file must be regenerated when it changes would prevent that.
</content>
</invoke>
