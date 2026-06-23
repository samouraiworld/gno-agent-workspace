# Review: PR #5843
Event: REQUEST_CHANGES

## Body
Two paths don't match master; both come from #5718 changes that landed after the superseded #5797 review. Verified on fbeeb60fa: the Part C softsign start aborts with `errLazyTmkmsListener`, and a `unix://` listener logs `allowed_kms_pubkeys is ignored` rather than requiring it. The rest of the guide checks out against the code, including the `pkconv.go` example pubkey resolving to its `g1qmptf8…` address.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5843-tmkms-quickstart-secure/1-fbeeb60fa/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/validators/tmkms-quickstart.md:703-758 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L703)
blocking. Part C starts with `-lazy` and no pre-existing genesis while the listener is enabled, which master refuses: `gnoland start` aborts with `-lazy cannot derive a genesis in tmkms_listener mode`. The guard keys on whether the listener is enabled, not on where the key lives, so softsign hits it even though the key is on disk. Parts B and D build genesis with `gnogenesis` and omit `-lazy`; only Part C diverges, and the troubleshooting row at line 1035 carries the same wrong assumption.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5843 -R gnolang/gno
TMPD=$(mktemp -d)
go run ./gno.land/cmd/gnoland config init -config-path "$TMPD/config/config.toml"
# enable the listener exactly as C.3 does (listen_addr last):
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.chain_id gno-tmkms-test
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.protocol_version v0.34
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.allowed_kms_pubkeys 0000000000000000000000000000000000000000000000000000000000000000
go run ./gno.land/cmd/gnoland config set -config-path "$TMPD/config/config.toml" consensus.priv_validator.tmkms_listener.listen_addr "unix://$TMPD/privval.sock"
# Part C's final command:
GNOROOT=$(pwd) go run ./gno.land/cmd/gnoland start -data-dir "$TMPD" -genesis "$TMPD/genesis.json" -chainid gno-tmkms-test -lazy -skip-genesis-sig-verification 2>&1 | grep -i lazy
rm -rf "$TMPD"
```

```
-lazy cannot derive a genesis in tmkms_listener mode: tmkms holds the validator key and signs only votes/proposals, so the validator pubkey is not locally available. Provide an explicit genesis.json (see the gnogenesis tool) and start without -lazy
```
</details>

## docs/validators/tmkms-quickstart.md:230 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L230)
blocking. The UDS sections say gnoland "requires" a non-empty `allowed_kms_pubkeys` (also lines 665 and 709), but on a `unix://` listener gnoland ignores it and logs `allowed_kms_pubkeys is ignored on a unix:// listener`. A reader who follows C.1/C.3/D.5 verbatim sets it on a socket and gets a warning the guide never mentions, justified by the opposite of the truth. The "socket perms are the real boundary" framing is right; only the "required" wording and its reason are wrong.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5843 -R gnolang/gno
# drive the config validator: empty allowlist is rejected on tcp:// but accepted
# on unix:// (i.e. not required on a socket).
cat > tm2/pkg/bft/privval/zz_doc_uds_test.go <<'EOF'
package privval

import "testing"

func TestDoc_UDSAllowlistNotRequired(t *testing.T) {
	mk := func(addr string) *TmkmsListenerConfig {
		c := DefaultTmkmsListenerConfig()
		c.ListenAddr, c.ChainID, c.AllowedKMSPubKeys = addr, "c", []string{}
		return c
	}
	if err := mk("tcp://127.0.0.1:26659").ValidateBasic(); err == nil {
		t.Fatal("tcp:// empty allowlist: expected rejection, got nil")
	} else {
		t.Logf("tcp:// empty allowlist -> %v", err)
	}
	if err := mk("unix:///tmp/p.sock").ValidateBasic(); err != nil {
		t.Fatalf("unix:// empty allowlist: expected accepted, got %v", err)
	}
	t.Log("unix:// empty allowlist -> accepted (allowlist NOT required on UDS)")
}
EOF
go test -run TestDoc_UDSAllowlistNotRequired -v ./tm2/pkg/bft/privval/
rm tm2/pkg/bft/privval/zz_doc_uds_test.go
```

```
tcp:// empty allowlist -> tmkms_listener.allowed_kms_pubkeys must not be empty on a tcp:// listener (...)
unix:// empty allowlist -> accepted (allowlist NOT required on UDS)
--- PASS: TestDoc_UDSAllowlistNotRequired
```
</details>

## docs/validators/tmkms-quickstart.md:229-231 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L229)
Nit. A.3 calls the `0600` socket "the whole auth boundary", but gnoland's chmod on the socket is best-effort: on a filesystem that ignores chmod on a socket it logs a warning and leaves default perms. The real guard is the `700 gnoland` parent dir you already create, so say so in half a sentence to stop a reader who relocates the socket from trusting the perm alone.

## docs/validators/tmkms-quickstart.md:144 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L144)
Nit. The optional cleanup line uses brace expansion (`{nodeid-hex,tmkms-identity-keygen}`), which is bash/zsh-only. A reader pasting it into `sh`/dash deletes nothing.

## docs/validators/tmkms-quickstart.md:438-440 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L438)
Optional. The production path ends at "submit the gpub/address to the chain's validator-onboarding path" with no concrete mechanism. A one-line pointer to where validator onboarding is documented, or a note that it isn't yet, would close the loop.

## docs/validators/tmkms-quickstart.md:475-493 [↗](../../../../../.worktrees/gno-review-5843/docs/validators/tmkms-quickstart.md#L475)
Optional. The `tmkms.toml` heredoc is unquoted, so the shell bakes in `${CHAIN_ID}`/`${TMKMS_ADDR}` at write time. A re-runner who only re-exports `$CHAIN_ID` keeps the stale `chain_id` and stalls on mismatch; one line saying the file must be regenerated when it changes would prevent that.
