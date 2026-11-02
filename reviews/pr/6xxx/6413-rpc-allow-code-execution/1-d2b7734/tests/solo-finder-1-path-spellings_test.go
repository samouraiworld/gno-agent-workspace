// Probe: does any abci_query path spelling reach withQueryEvalMachine or the
// .app/simulate branch on a real in-memory gno.land node without runsCode
// refusing it once rpc.allow_code_execution is false?
//
// Repro from a plain clone of gnolang/gno at d2b7734c00d2cb9773ba51e8fa2e0be7b5292cfd:
//   cp solo-finder-1-path-spellings_test.go gno.land/pkg/gnoland/
//   go test ./gno.land/pkg/gnoland -run TestProbePathSpellings -timeout 2m -v
//
// Phase 1 runs every spelling with allow=true and records which ones executed
// (vm: no error and data; simulate: a marshalled Result in Value). Phase 2 flips
// the same Environment to allow=false and asserts each executed spelling is refused.
package gnoland_test

import (
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/integration"
	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	rpccore "github.com/gnolang/gno/tm2/pkg/bft/rpc/core"
	rpctypes "github.com/gnolang/gno/tm2/pkg/bft/rpc/lib/types"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestProbePathSpellings(t *testing.T) {
	cfg := integration.TestingMinimalNodeConfig(gnoenv.RootDir())
	n, _ := integration.TestingInMemoryNode(t, log.NewTestingLogger(t), cfg)
	t.Cleanup(func() { _ = n.Stop() })
	env := n.RPCEnvironment()

	vmData := []byte("strconv.Itoa(42)")
	vm := []string{
		"vm/qeval", "/vm/qeval", "vm/qeval?", "vm/qeval/", "vm/qeval?a/b", "vm/qeval/x/y",
		"vm//qeval", "vm/./qeval", "./vm/qeval", "vm/qeval#x", "Vm/qeval", "vm/QEVAL",
		"vm/qeval%3F", "vm/qeval ", "vm/qeval\x00", "vm/qeval_json", "vm/qeval_json?x",
		"vm/qrender", "vm/qrender?", "vm/qrender/", "vm?/qeval", "vm/qeval;x",
	}
	sim := []string{
		".app/simulate", "/.app/simulate", ".app/simulate/", ".app/simulate/x", ".app/./simulate",
		"//.app/simulate", "./.app/simulate", ".app/simulate?", ".app//simulate", ".APP/simulate",
		".app/Simulate", "/.app/simulate/", ".app/simulate#",
	}
	executed := []string{}
	for _, p := range vm {
		res, err := func() (r *struct{ ok bool; info string }, e error) {
			defer func() {
				if rec := recover(); rec != nil {
					r = &struct{ ok bool; info string }{false, "panic"}
				}
			}()
			out, err := env.ABCIQuery(&rpctypes.Context{}, p, vmData, 0, false)
			if err != nil {
				return nil, err
			}
			ok := out.Response.Error == nil && len(out.Response.Data) > 0
			return &struct{ ok bool; info string }{ok, string(out.Response.Data) + " " + out.Response.Log}, nil
		}()
		if err != nil {
			t.Logf("vm  %-22q rpc error: %v", p, err)
			continue
		}
		t.Logf("vm  %-22q executed=%v %s", p, res.ok, res.info)
		if res.ok {
			executed = append(executed, p)
		}
	}
	for _, p := range sim {
		out, err := env.ABCIQuery(&rpctypes.Context{}, p, nil, 0, false)
		if err != nil {
			t.Logf("sim %-22q rpc error: %v", p, err)
			continue
		}
		ok := out.Response.Value != nil
		t.Logf("sim %-22q executed=%v log=%s", p, ok, out.Response.Log)
		if ok {
			executed = append(executed, p)
		}
	}
	t.Logf("executed with allow=true: %q", executed)

	env.Config.AllowCodeExecution = false
	for _, p := range executed {
		_, err := env.ABCIQuery(&rpctypes.Context{}, p, vmData, 0, false)
		if err == nil || err.Error() != rpccore.NewCodeExecutionRefusedError(p).Error() {
			t.Errorf("BYPASS: %q executed with allow=true and is not refused with allow=false (err=%v)", p, err)
		}
	}
}
