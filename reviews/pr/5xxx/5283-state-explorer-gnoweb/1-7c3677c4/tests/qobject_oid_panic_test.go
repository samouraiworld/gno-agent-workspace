// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5283 -R gnolang/gno && git checkout 7c3677c4
curl -fsSL -o gno.land/pkg/sdk/vm/qobject_oid_panic_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5283-state-explorer-gnoweb/1-7c3677c4/tests/qobject_oid_panic_test.go
go test -v -count=1 -timeout 60s -run 'TestVmHandlerQuery_ObjectJSON_LongHexPanic|TestVmHandlerQuery_ObjectBinary_LongHexPanic' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/qobject_oid_panic_test.go
*/

// Mechanism: gnovm/pkg/gnolang/ownership.go's ObjectID.UnmarshalAmino calls
// hex.Decode(oid.PkgID.Hashlet[:], []byte(parts[0])). Hashlet is [20]byte, so the
// destination has fixed capacity 20. If parts[0] is >= 42 hex chars, hex.Decode
// writes past index 20 and panics with "index out of range [20] with length 20".
// QueryObjectJSON/QueryObjectBinary forward the raw user oidStr into UnmarshalAmino
// with no length pre-validation. The panic is not recovered in vmHandler.Query nor
// in baseapp.handleQueryCustom — only the outer RPC server recovers, so the node
// stays alive but every malformed long-OID query returns a generic 500 + stack-trace
// log entry. This is a low-cost DoS / log-spam vector exposed to the public web by
// gnoweb's `$state&oid=...` route and the new `vm/qobject_json` ABCI endpoint.
//
// Observed: panic propagates; test asserts a clean error response instead of panic.
// To flip the assertion post-fix: comment the require.Fail line and leave the
// res.IsOK()==false / regex check active.

package vm

import (
	"strings"
	"testing"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
)

func TestVmHandlerQuery_ObjectJSON_LongHexPanic(t *testing.T) {
	env := setupTestEnv()
	vmHandler := env.vmh

	// 42 hex chars (PkgID Hashlet is 20 bytes => max 40 hex chars).
	bad := []byte(strings.Repeat("a", 42) + ":1")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("qobject_json panicked on oversized hex prefix: %v", r)
		}
	}()

	req := abci.RequestQuery{
		Path: "vm/qobject_json",
		Data: bad,
	}
	res := vmHandler.Query(env.ctx, req)
	if res.IsOK() {
		t.Fatalf("expected error response, got OK: %s", res.Data)
	}
	// After a fix that pre-validates length, expect a clean "invalid object id"
	// error instead of an internal panic/500.
}

func TestVmHandlerQuery_ObjectBinary_LongHexPanic(t *testing.T) {
	env := setupTestEnv()
	vmHandler := env.vmh

	bad := []byte(strings.Repeat("b", 60) + ":7")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("qobject_binary panicked on oversized hex prefix: %v", r)
		}
	}()

	req := abci.RequestQuery{
		Path: "vm/qobject_binary",
		Data: bad,
	}
	res := vmHandler.Query(env.ctx, req)
	if res.IsOK() {
		t.Fatalf("expected error response, got OK")
	}
}
