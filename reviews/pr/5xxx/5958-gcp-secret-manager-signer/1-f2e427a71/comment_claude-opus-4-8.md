# Review: PR [#5958](https://github.com/gnolang/gno/pull/5958)
Posted: https://github.com/gnolang/gno/pull/5958#pullrequestreview-4733410778
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

[`docs/validators/tmkms.md:13-14`](https://github.com/gnolang/gno/blob/f2e427a71/docs/validators/tmkms.md?plain=1#L13-L14) says gnoland supports three mutually exclusive privval setups and tables them, so this mode needs a row. Its production-readiness cell matches the [local-file row](https://github.com/gnolang/gno/blob/f2e427a71/docs/validators/tmkms.md?plain=1#L18): the key sits in the gnoland process next to a network listener and there is no signer-side double-sign protection.

Verified on f2e427a71: `go build ./...` fails on the missing `go.sum` entries, and CI ran only the semantic-title check. `gnoland config set consensus.priv_validator.gcp_secret_manager.project_id my-proj` is accepted with `secret_id` left empty, and the node then boots on a locally minted key.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5958-gcp-secret-manager-signer/1-f2e427a71/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go:7-8 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go#L7-L8) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028370)
Critical: these imports have no `go.mod` or `go.sum` entry, so `go build ./...` fails at this head, and lint fails the same way under [`modules-download-mode: readonly`](https://github.com/gnolang/gno/blob/f2e427a71/.github/golangci.yml#L6). Resolving the closure adds 16 modules, 2 direct and 14 indirect.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5958 -R gnolang/gno
go build ./... 2>&1 | head -2
```

```
tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go:7:2: missing go.sum entry for module providing package cloud.google.com/go/secretmanager/apiv1 (imported by github.com/gnolang/gno/tm2/pkg/bft/privval/signer/gcpsecretmanager); to add:
	go get github.com/gnolang/gno/tm2/pkg/bft/privval/signer/gcpsecretmanager
```
</details>

## tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go:54-58 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028389)
Critical: a block with only `project_id` or only `secret_id` passes here, reads as disabled through [`IsEnabled`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L48-L50), and falls through to [`LoadOrMakeLocalSigner`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L156-L157), which mints and persists a fresh validator key. An operator moving a live validator to Secret Manager ends up signing with an identity they never chose.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5958 -R gnolang/gno
GOFLAGS=-mod=mod go get ./tm2/pkg/bft/privval/signer/gcpsecretmanager
cat > tm2/pkg/bft/privval/zz_repro_test.go <<'EOF'
package privval

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/privval/signer/gcpsecretmanager"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHalfConfiguredGCPIsRejected(t *testing.T) {
	root := t.TempDir()

	cfg := DefaultPrivValidatorConfig()
	cfg.RootDir = root
	cfg.GCPSecretManager = &gcpsecretmanager.Config{ProjectID: "my-project", Version: "latest"}

	assert.Error(t, cfg.ValidateBasic(), "a partially filled gcp block should be rejected")

	signer, err := NewSignerFromConfig(
		context.Background(), cfg, ed25519.GenPrivKey(), slog.New(slog.DiscardHandler),
	)
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(root, "priv_validator_key.json"))
	assert.True(t, os.IsNotExist(statErr),
		"no fresh validator key should be minted on disk, got signer %s", signer)
}
EOF
go test -run TestHalfConfiguredGCPIsRejected ./tm2/pkg/bft/privval/ 2>&1 | grep -E "Messages:|FAIL"
rm tm2/pkg/bft/privval/zz_repro_test.go
git checkout HEAD -- go.mod go.sum
```

```
        	Messages:   	a partially filled gcp block should be rejected
        	Messages:   	no fresh validator key should be minted on disk, got signer {Type: LocalSigner, Addr: g19e4ml9tuldw3j4fjjd43c32mdagp3luwhgnydm}
--- FAIL: TestHalfConfiguredGCPIsRejected (0.00s)
FAIL	github.com/gnolang/gno/tm2/pkg/bft/privval	0.007s
```
</details>

## tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go:123-136 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L123-L136) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028398)
A secret that exists with no versions yet, the second case [`isNotFound`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L103-L105) documents, aborts with `AlreadyExists` because `CreateSecret` runs unconditionally, and cannot recover without deleting the secret. The same shape blocks a retry after a transient `AddSecretVersion` failure, which leaves a created-but-empty secret behind. Treating `AlreadyExists` as success and continuing to `AddSecretVersion` covers both.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5958 -R gnolang/gno
GOFLAGS=-mod=mod go get ./tm2/pkg/bft/privval/signer/gcpsecretmanager
cat > tm2/pkg/bft/privval/signer/gcpsecretmanager/zz_repro_test.go <<'EOF'
package gcpsecretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSecretExistsWithoutVersion(t *testing.T) {
	client := newMockClient()
	client.createErr = status.Error(codes.AlreadyExists, "Secret already exists")

	cfg := &Config{ProjectID: "p", SecretID: "s", Version: "latest", CreateIfMissing: true}

	signer, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err, "an existing but empty secret should receive a first version")
	require.NotNil(t, signer)

	_, ok := client.secrets[cfg.versionName()]
	assert.True(t, ok, "the generated key should read back at the configured version")
}
EOF
go test -run TestSecretExistsWithoutVersion ./tm2/pkg/bft/privval/signer/gcpsecretmanager/ 2>&1 | grep -E "unable to|Messages:|FAIL"
rm tm2/pkg/bft/privval/signer/gcpsecretmanager/zz_repro_test.go
git checkout HEAD -- go.mod go.sum
```

```
        	            	unable to create secret "s" in GCP Secret Manager: rpc error: code = AlreadyExists desc = Secret already exists
        	Messages:   	an existing but empty secret should receive a first version
--- FAIL: TestSecretExistsWithoutVersion (0.00s)
FAIL	github.com/gnolang/gno/tm2/pkg/bft/privval/signer/gcpsecretmanager	0.007s
```
</details>

## tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go:138-145 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L138-L145) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028405)
With `create_if_missing = true` and `version = "5"` the first boot signs blocks with a key that every restart then fails to load: `AddSecretVersion` can only create version 1, while [`versionName`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L62-L69) reads whatever `version` pins, and `ValidateBasic` accepts the combination. Reject `create_if_missing` together with a version other than `latest`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5958 -R gnolang/gno
GOFLAGS=-mod=mod go get ./tm2/pkg/bft/privval/signer/gcpsecretmanager
cat > tm2/pkg/bft/privval/signer/gcpsecretmanager/zz_repro_test.go <<'EOF'
package gcpsecretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateIfMissingWithPinnedVersion(t *testing.T) {
	client := newMockClient()
	cfg := &Config{ProjectID: "p", SecretID: "s", Version: "5", CreateIfMissing: true}

	require.Error(t, cfg.ValidateBasic(), "a pinned version cannot be satisfied by a fresh secret")

	first, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err)

	// Real GCP answers AlreadyExists once the secret has been created.
	client.createErr = status.Error(codes.AlreadyExists, "Secret already exists")

	second, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err, "a restart must find the key the first boot minted")
	assert.True(t, second.PubKey().Equals(first.PubKey()), "a restart must keep the same identity")
}
EOF
go test -run TestCreateIfMissingWithPinnedVersion ./tm2/pkg/bft/privval/signer/gcpsecretmanager/ 2>&1 | grep -E "unable to|Messages:|FAIL"
rm tm2/pkg/bft/privval/signer/gcpsecretmanager/zz_repro_test.go
git checkout HEAD -- go.mod go.sum
```

```
        	Messages:   	a pinned version cannot be satisfied by a fresh secret
        	            	unable to create secret "s" in GCP Secret Manager: rpc error: code = AlreadyExists desc = Secret already exists
        	Messages:   	a restart must find the key the first boot minted
--- FAIL: TestCreateIfMissingWithPinnedVersion (0.00s)
FAIL	github.com/gnolang/gno/tm2/pkg/bft/privval/signer/gcpsecretmanager	0.008s
```
</details>

## tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go:69-74 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L69-L74) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028408)
The client built by [`newClient`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go#L68-L75) is never closed, so its gRPC connection and background goroutines outlive the one fetch that needs them. The only `Close` in the package is [`Signer.Close`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L42-L44), which returns nil and never sees the client.

## tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_test.go:49-58 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_test.go#L49-L58) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028413)
Missing test: no test can express a secret that already exists, because `CreateSecret` returns a fresh secret on every call and never consults `m.secrets`, so `TestNewSigner_MissingSecret_CreateIfMissing` passes on a path that cannot fail.

<details><summary>test cases</summary>

Track created names and answer `AlreadyExists` on a repeat, then both `create_if_missing` cases above become reachable:

```go
func (m *mockClient) CreateSecret(
	_ context.Context,
	req *secretmanagerpb.CreateSecretRequest,
) (*secretmanagerpb.Secret, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	name := req.Parent + "/secrets/" + req.SecretId
	if _, ok := m.created[name]; ok {
		return nil, status.Error(codes.AlreadyExists, "Secret already exists")
	}
	m.created[name] = struct{}{}

	return &secretmanagerpb.Secret{Name: name}, nil
}
```
</details>

## tm2/pkg/bft/privval/config.go:110-118 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L110-L118) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028417)
Missing test: `tm2/pkg/bft/privval` covers none of the new `PrivValidatorConfig` behavior, not the nil guard here, not `errMultipleSignerSourcesSet`, not the fall-through to the local signer. The mutual-exclusion pair is what a fourth signer PR will silently break.

## tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go:17 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L17) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028421)
Nit: "If set (together with project_id), the local signer is disabled" buries in a parenthetical that both fields are required and one alone is an error, and this text renders into every generated `config.toml`.

## tm2/pkg/bft/privval/config.go:151-154 [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L151-L154) [posted](https://github.com/gnolang/gno/pull/5958#discussion_r3613028427)
Suggestion: [#5956](https://github.com/gnolang/gno/pull/5956), [#5959](https://github.com/gnolang/gno/pull/5959), and [#5980](https://github.com/gnolang/gno/pull/5980) each add a branch to this chain, a clause to the [pairwise exclusion check](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L124-L126), and the same `ParseFileKey` extraction, so all four conflict and the `only one of remote_signer, tmkms_listener, or gcp_secret_manager` text goes stale in each. Collecting enabled mode names into a list with a single "at most one" check would let them land independently. Was the per-field shape deliberate?
