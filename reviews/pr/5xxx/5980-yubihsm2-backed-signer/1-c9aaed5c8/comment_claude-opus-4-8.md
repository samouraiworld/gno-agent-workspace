# Review: PR [#5980](https://github.com/gnolang/gno/pull/5980)
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Reproduced on c9aaed5c8: booting the node with the `connector_url` value the field doc gives sends the session to `http://http//127.0.0.1:12345/connector/api`, and `gnoland config get consensus.priv_validator` prints the HSM password in cleartext.

- Issue [#3236](https://github.com/gnolang/gno/issues/3236) asks for the YubiHSM2 as a remote signer under [#3230](https://github.com/gnolang/gno/issues/3230), and the [gnokms README](https://github.com/gnolang/gno/blob/c9aaed5c8/contribs/gnokms/README.md?plain=1#L5) already reserves an HSM backend slot. Wiring the device into the node's own privval instead keeps the connector address and the device password on the validator machine, so say which placement was intended.
- `github.com/certusone/yubihsm-go` last released v0.3.0 in January 2023 against Go 1.14, and its SCP03 channel MACs every command with [`github.com/enceve/crypto/cmac`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/securechannel/channel.go#L12) pinned to a 2016 commit. That puts unmaintained crypto on the path every validator vote crosses; the issue flags the dormancy but leaves the answer open.
- No operator guide ships with this mode, so the device-side setup of the auth key, the `sign-eddsa` capability and the connector service is written down nowhere. The tmkms mode ships [`docs/validators/tmkms.md`](https://github.com/gnolang/gno/blob/c9aaed5c8/docs/validators/tmkms.md?plain=1#L1).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5980-yubihsm2-backed-signer/1-c9aaed5c8/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/privval/signer/yubihsm/client.go:6-8 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/client.go#L6-L8)
Critical: `github.com/certusone/yubihsm-go` is imported but missing from `go.mod` and `go.sum`, so the module does not build. The PR's checks are green because the build and test workflows are gated behind maintainer approval and never ran.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5980 -R gnolang/gno
grep -c certusone go.mod go.sum
go build ./tm2/pkg/bft/privval/...
```

```
go.mod:0
go.sum:0
tm2/pkg/bft/privval/signer/yubihsm/client.go:6:2: no required module provides package github.com/certusone/yubihsm-go; to add it:
	go get github.com/certusone/yubihsm-go
tm2/pkg/bft/privval/signer/yubihsm/client.go:7:2: no required module provides package github.com/certusone/yubihsm-go/commands; to add it:
	go get github.com/certusone/yubihsm-go/commands
tm2/pkg/bft/privval/signer/yubihsm/client.go:8:2: no required module provides package github.com/certusone/yubihsm-go/connector; to add it:
	go get github.com/certusone/yubihsm-go/connector
```
</details>

## tm2/pkg/bft/privval/signer/yubihsm/config.go:9-12 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L9-L12)
Critical: following the `http://127.0.0.1:12345` example given here and in the `comment:` tag that lands in every generated `config.toml` makes the node fail to start. [`HTTPConnector.Request`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/connector/http.go#L39) prepends `http://` itself, so `connector_url` has to be a bare `host:port`, and the hardcoded scheme also puts a TLS connector on another host out of reach. `ValidateBasic` accepts the broken form, so it only surfaces at node start.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5980 -R gnolang/gno
go get github.com/certusone/yubihsm-go@v0.3.0
go build -o /tmp/gnoland ./gno.land/cmd/gnoland
rm -rf /tmp/yhnode && mkdir -p /tmp/yhnode
(cd /tmp/yhnode && timeout 60 /tmp/gnoland start -lazy -data-dir /tmp/yhnode/data \
  -gnoroot-dir "$PWD" >/dev/null 2>&1)
CFG=/tmp/yhnode/data/config/config.toml
for kv in auth_key_id=1 key_id=100 password=s3cret; do
  /tmp/gnoland config set -config-path $CFG "consensus.priv_validator.yubihsm.${kv%%=*}" "${kv#*=}" >/dev/null
done
for url in 'http://127.0.0.1:12399' '127.0.0.1:12399'; do
  /tmp/gnoland config set -config-path $CFG consensus.priv_validator.yubihsm.connector_url "$url" >/dev/null
  echo "connector_url = $url"
  (cd /tmp/yhnode && timeout 60 /tmp/gnoland start -data-dir /tmp/yhnode/data -gnoroot-dir "$PWD" 2>&1 | tail -1)
done
rm -rf /tmp/yhnode /tmp/gnoland
git checkout go.mod go.sum
```

```
connector_url = http://127.0.0.1:12399
unable to create the Gnoland node, unable to open YubiHSM2 session: Post "http://http//127.0.0.1:12399/connector/api": dial tcp: lookup http: no such host
connector_url = 127.0.0.1:12399
unable to create the Gnoland node, unable to open YubiHSM2 session: Post "http://127.0.0.1:12399/connector/api": dial tcp 127.0.0.1:12399: connect: connection refused
```
</details>

## tm2/pkg/bft/privval/signer/yubihsm/config.go:18-19 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L18-L19)
The password lands in `config.toml`, which [`WriteConfigFile`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/config/toml.go#L61) writes at 0644 and `gnoland config get` prints in full, while the key file this replaces is written at [0600](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/local/key.go#L65). Anyone who can read the config and reach the connector can sign. Take the password from a file path or the environment instead.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5980 -R gnolang/gno
go get github.com/certusone/yubihsm-go@v0.3.0
go build -o /tmp/gnoland ./gno.land/cmd/gnoland
rm -rf /tmp/yhcfg && mkdir -p /tmp/yhcfg
/tmp/gnoland config init -config-path /tmp/yhcfg/config.toml
for kv in auth_key_id=1 key_id=100 password=sup3rs3cret connector_url=127.0.0.1:12345; do
  /tmp/gnoland config set -config-path /tmp/yhcfg/config.toml \
    "consensus.priv_validator.yubihsm.${kv%%=*}" "${kv#*=}" >/dev/null
done
stat -c '%a %n' /tmp/yhcfg/config.toml
/tmp/gnoland config get -config-path /tmp/yhcfg/config.toml consensus.priv_validator | tail -7
rm -rf /tmp/yhcfg /tmp/gnoland
git checkout go.mod go.sum
```

```
644 /tmp/yhcfg/config.toml
    "yubihsm": {
        "connector_url": "127.0.0.1:12345",
        "auth_key_id": 1,
        "password": "sup3rs3cret",
        "key_id": 100
    }
}
```
</details>

## tm2/pkg/bft/privval/signer/yubihsm/config.go:41-44 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L41-L44)
A section with `auth_key_id`, `key_id` and `password` filled in but an empty `connector_url` reads as disabled, so the node falls through to [`LoadOrMakeLocalSigner`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L157) and signs with the file key without saying anything. `gnoland config set` walks an operator straight into that state: setting `connector_url` first is rejected while the other three succeed. Either reject a partly-filled disabled section or warn at startup.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5980 -R gnolang/gno
go get github.com/certusone/yubihsm-go@v0.3.0
go build -o /tmp/gnoland ./gno.land/cmd/gnoland
rm -rf /tmp/yhord && mkdir -p /tmp/yhord
/tmp/gnoland config init -config-path /tmp/yhord/config.toml >/dev/null
for kv in connector_url=127.0.0.1:12345 auth_key_id=1 key_id=100 password=s3cret; do
  printf '%-14s -> ' "${kv%%=*}"
  /tmp/gnoland config set -config-path /tmp/yhord/config.toml \
    "consensus.priv_validator.yubihsm.${kv%%=*}" "${kv#*=}" 2>&1 | sed 's|/tmp/.*||'
done
/tmp/gnoland config get -config-path /tmp/yhord/config.toml consensus.priv_validator.yubihsm
rm -rf /tmp/yhord /tmp/gnoland
git checkout go.mod go.sum
```

```
connector_url  -> unable to validate config, yubihsm signer: auth_key_id cannot be zero when enabled
auth_key_id    -> Updated configuration saved at
key_id         -> Updated configuration saved at
password       -> Updated configuration saved at
{
    "connector_url": "",
    "auth_key_id": 1,
    "password": "s3cret",
    "key_id": 100
}
```
</details>

## tm2/pkg/bft/privval/signer/yubihsm/signer.go:82-87 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L82-L87)
None of `newSigner`'s four failure exits destroys the session opened on the line above, so a failed start leaves an authenticated session and its keepalive goroutine holding one of the device's limited slots until the process exits. A wrong `key_id` reaches this, and each retry burns another slot.

## tm2/pkg/bft/privval/config.go:110-126 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L110-L126)
Missing test: nothing exercises `errNilYubiHSMConfig` or either `errMultipleSignerSourcesSet` branch, though the matching [`errNilRemoteSignerConfig`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config_test.go#L62) and [`errBothExternalSignersEnabled`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config_test.go#L272) rules are covered. The mutual-exclusion check is the only thing between an operator and two live signer sources.

<details><summary>test cases</summary>

```go
func enabledYubiHSMConfig() *yubihsm.Config {
	return &yubihsm.Config{
		ConnectorURL: "127.0.0.1:12345",
		AuthKeyID:    1,
		KeyID:        2,
	}
}

func TestPrivValidatorConfig_YubiHSMValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("nil yubihsm config", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = nil

		assert.ErrorIs(t, cfg.ValidateBasic(), errNilYubiHSMConfig)
	})

	t.Run("yubihsm with remote signer", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.RemoteSigner.ServerAddress = "tcp://127.0.0.1:26659"

		assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)
	})

	t.Run("yubihsm with tmkms listener", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.TmkmsListener.ListenAddr = "unix:///tmp/tmkms-yubihsm-test.sock"
		cfg.TmkmsListener.ChainID = "test-chain"

		assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)
	})

	t.Run("yubihsm alone", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()

		require.NoError(t, cfg.ValidateBasic())
	})
}

func TestPrivValidatorConfig_YubiHSMNewPrivValidator(t *testing.T) {
	t.Parallel()

	privKey := ed25519.GenPrivKey()
	logger := slog.New(slog.DiscardHandler)

	t.Run("yubihsm with remote signer", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.RemoteSigner.ServerAddress = "tcp://127.0.0.1:26659"

		_, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
		assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
	})

	t.Run("yubihsm with tmkms listener", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.TmkmsListener.ListenAddr = "unix:///tmp/tmkms-yubihsm-test2.sock"
		cfg.TmkmsListener.ChainID = "test-chain"

		_, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
		assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
	})
}
```
</details>

## tm2/pkg/bft/privval/config.go:38-42 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L38-L42)
Nit: the struct's own doc comment above still says "At most one of RemoteSigner or TmkmsListener may be enabled" now that there are three mutually exclusive modes.

## tm2/pkg/bft/privval/signer/local/key.go:83 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/local/key.go#L83)
Nit: this wraps everything `ParseFileKey` returns as "unable to unmarshal", so a malformed file now reads `unable to unmarshal FileKey from <path>: unable to unmarshal FileKey: EOF` and a `validate` failure gets labelled an unmarshal failure.

## tm2/pkg/bft/privval/signer/local/key.go:89-92 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/local/key.go#L89-L92)
Suggestion: the only caller is [`LoadFileKey`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/local/key.go#L81) in the same file, and the yubihsm signer never reads a `FileKey`. Splitting the function is fine; exporting it for a consumer that does not exist yet widens the package API for nothing.

## tm2/pkg/bft/privval/signer/yubihsm/signer.go:15 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L15)
Nit: there are no AWS, GCP or Vault backed signers in the tree. The existing modes are the [local file](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L157), the [gnokms remote client](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L143) and the [tmkms listener](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L201).

## tm2/pkg/bft/privval/signer/yubihsm/signer.go:50 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L50)
Suggestion: [`parseSignDataEddsaResponse`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/commands/response.go#L279-L283) copies the device payload into `Signature` with no length check of its own, and this passes it straight through, so a malformed response surfaces later as a rejected vote rather than a signer error. The public key gets a length check at construction; the signature deserves the same.

## tm2/pkg/bft/privval/signer/yubihsm/signer.go:104-111 [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L104-L111)
Suggestion: [`GetPubKeyResponse`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/commands/response.go#L83-L87) carries the key's `Algorithm` and only `KeyData` is read here, so pointing `key_id` at a non-Ed25519 object reports an unexpected public key length instead of naming the real problem.
