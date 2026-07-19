/* Run: from a gno checkout:
gh pr checkout 5956 -R gnolang/gno && git checkout 1ad3009b9
go get github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/service/secretsmanager
curl -fsSL -o tm2/pkg/bft/privval/signer/awssecretsmanager/config_arn_create_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5956-aws-secrets-manager-signer/1-1ad3009b9/tests/config_arn_create_test.go
go test -v -run 'TestConfigValidateBasic_ARNWithCreateIfMissing' ./tm2/pkg/bft/privval/signer/awssecretsmanager/
rm tm2/pkg/bft/privval/signer/awssecretsmanager/config_arn_create_test.go && git checkout go.mod go.sum
*/

// secret_id is documented as an ARN or a name, but createAndStoreKey passes it
// straight to CreateSecret as Name, and a Secrets Manager name cannot contain
// the colons an ARN carries. At 1ad3009b9 ValidateBasic accepts the combination
// and the failure surfaces only as an AWS InvalidParameterException at startup.

package awssecretsmanager

import "testing"

func TestConfigValidateBasic_ARNWithCreateIfMissing(t *testing.T) {
	t.Parallel()

	const arn = "arn:aws:secretsmanager:eu-west-1:123456789012:secret:validator-key-a1b2c3"

	// An ARN with create_if_missing is unsatisfiable: CreateSecret only accepts
	// a name, so the combination must be refused during config validation.
	cfg := &Config{SecretID: arn, CreateIfMissing: true}
	if err := cfg.ValidateBasic(); err == nil {
		t.Errorf("ValidateBasic accepted an ARN secret_id with create_if_missing")
	}

	// An ARN on its own is fine: GetSecretValue takes either form.
	cfg = &Config{SecretID: arn}
	if err := cfg.ValidateBasic(); err != nil {
		t.Errorf("ValidateBasic rejected a plain ARN secret_id: %v", err)
	}

	// A plain name with create_if_missing is the supported creation path.
	cfg = &Config{SecretID: "validator-key", CreateIfMissing: true}
	if err := cfg.ValidateBasic(); err != nil {
		t.Errorf("ValidateBasic rejected a name secret_id with create_if_missing: %v", err)
	}
}
