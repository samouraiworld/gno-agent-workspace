package main

/* Run: from a gno checkout:
gh pr checkout 6112 -R gnolang/gno && git checkout 6dcb5272a
curl -fsSL -o contribs/gpao/casespace_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6112-parked-redeploy-not-active/1-6dcb5272a/tests/casespace_test.go
go test -C contribs/gpao -count=1 -v -run 'TestPreflightCaseSpace' .
rm contribs/gpao/casespace_test.go
*/

// TestPreflightCaseSpace walks every state the chain can hold at a package path
// and prints, per cell, what each pre-flight answers and where handleCandidate
// leaves the status board.
//
// oldIsActive below is the deleted vm/qfile pre-flight, copied verbatim, so the
// before and after columns come from one run against one chain rather than from
// two checkouts.
//
// Measured on 6dcb5272a. The table is the output; what it asserts is the
// premise each cell needs, never the cell's outcome.

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoclient"
	"github.com/gnolang/gno/gno.land/pkg/gnoland"
	"github.com/gnolang/gno/gno.land/pkg/integration"
	vm "github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
	rpcclient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	"github.com/gnolang/gno/tm2/pkg/commands"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

// oldIsActive is the pre-flight this PR replaces, copied from oracle.go at the
// merge base 2ed70a202.
func oldIsActive(o *oracle, ctx context.Context, pkgPath string) bool {
	res, err := o.client.Query(gnoclient.QueryCfg{
		Path: "vm/qfile",
		Data: []byte(pkgPath),
	})
	if err != nil || res == nil {
		return false
	}
	return res.Response.Error == nil
}

func csMod(path string, private bool) string {
	m := gno.GenGnoModLatest(path)
	if private {
		m += "\nprivate = true"
	}
	return m + "\n"
}

// csPkg builds a one-file realm. The file name sorts after gnomod.toml, which
// ValidateMemPackageAny requires.
func csPkg(path, name, version string, private bool) *std.MemPackage {
	return &std.MemPackage{
		Name: name,
		Path: path,
		Files: []*std.MemFile{
			{Name: "gnomod.toml", Body: csMod(path, private)},
			{Name: name + ".gno", Body: "package " + name +
				"\n\nfunc V(cur realm) string { return \"" + version + "\" }\n"},
		},
	}
}

func TestPreflightCaseSpace(t *testing.T) {
	gnoroot := gnoenv.RootDir()
	cfg := integration.TestingMinimalNodeConfig(gnoroot)
	cfg.SkipGenesisSigVerification = true

	signer, err := gnoclient.SignerFromBip39(
		integration.DefaultAccount_Seed, cfg.Genesis.ChainID, "", 0, 0)
	require.NoError(t, err)
	info, err := signer.Info()
	require.NoError(t, err)
	who := info.GetAddress()

	ggs := cfg.Genesis.AppState.(gnoland.GnoGenesisState)
	ggs.Balances = []gnoland.Balance{{
		Address: who,
		Amount:  std.NewCoins(std.NewCoin("ugnot", 100_000_000_000)),
	}}
	ggs.VM.Params.CodeSubmissionPolicy = "inert"
	ggs.VM.Params.PkgApprovers = []crypto.Address{who}
	cfg.Genesis.AppState = ggs

	node, remote := integration.TestingInMemoryNode(t, log.NewNoopLogger(), cfg)
	defer node.Stop()

	rpc, err := rpcclient.NewHTTPClient(remote)
	require.NoError(t, err)
	client := gnoclient.Client{Signer: signer, RPCClient: rpc}

	// park submits MsgAddPackage; under "inert" that stores without activating.
	// DeliverTx is checked explicitly: BroadcastTxCommit's error does not cover
	// a delivery that the keeper refused.
	park := func(mpkg *std.MemPackage) error {
		signed, err := client.SignTx(std.Tx{
			Msgs: []std.Msg{vm.MsgAddPackage{Creator: who, Package: mpkg}},
			Fee:  std.NewFee(20_000_000, std.MustParseCoin("1000000ugnot")),
		}, 0, 0)
		if err != nil {
			return err
		}
		res, err := client.BroadcastTxCommit(signed)
		if err != nil {
			return err
		}
		if !res.DeliverTx.IsOK() {
			return res.DeliverTx.Error
		}
		return nil
	}
	enable := func(mpkg *std.MemPackage) error {
		signed, err := client.SignTx(std.Tx{
			Msgs: []std.Msg{vm.MsgEnablePackage{
				Approver: who, PkgPath: mpkg.Path, PkgHash: vm.PackageContentHash(mpkg),
			}},
			Fee: std.NewFee(20_000_000, std.MustParseCoin("1000000ugnot")),
		}, 0, 0)
		if err != nil {
			return err
		}
		res, err := client.BroadcastTxCommit(signed)
		if err != nil {
			return err
		}
		if !res.DeliverTx.IsOK() {
			return res.DeliverTx.Error
		}
		return nil
	}
	// meta is the raw vm/qpkgmeta_json answer: the chain's own word on the cell.
	meta := func(path string) string {
		res, err := client.Query(gnoclient.QueryCfg{Path: "vm/qpkgmeta_json", Data: []byte(path)})
		if err != nil {
			return "query error: " + err.Error()
		}
		if res.Response.Error != nil {
			return "response error: " + res.Response.Error.Error()
		}
		return string(res.Response.Data)
	}

	tio := commands.NewTestIO()
	tio.SetOut(commands.WriteNopCloser(io.Discard))
	tio.SetErr(commands.WriteNopCloser(io.Discard))
	o, err := newOracle(config{
		remote: remote, chainID: cfg.Genesis.ChainID,
		mnemonic: integration.DefaultAccount_Seed, gnoRoot: gnoroot,
		gasFee: defaultGasFee, gasWanted: defaultGasWanted,
		verifyBudget: time.Minute,
	}, tio)
	require.NoError(t, err)
	o.blockMaxGas = o.queryBlockMaxGas(t.Context())

	type row struct {
		cell, meta, status, reason string
		old, new                   bool
		spent                      int64
	}
	var rows []row

	// run reads both pre-flights against the chain as it stands, then drives the
	// candidate through handleCandidate and records where the board lands.
	run := func(cell string, cand *std.MemPackage) {
		before := o.spent
		m := meta(cand.Path)
		old := oldIsActive(o, t.Context(), cand.Path)
		nw := o.isSettled(t.Context(), cand.Path)
		o.handleCandidate(t.Context(), cand)
		st := o.status.get(cand.Path)
		rows = append(rows, row{cell, m, st.Status, st.Reason, old, nw, o.spent - before})
	}

	// 1. absent: nothing was ever submitted at this path.
	run("absent", csPkg("gno.land/r/test/nothing", "nothing", "v1", false))

	// 2. inert: parked, never enabled.
	inert := csPkg("gno.land/r/test/inert", "inert", "v1", false)
	require.NoError(t, park(inert))
	run("inert, candidate is the parked blob", inert)

	// 3. live public, nothing pending.
	pub := csPkg("gno.land/r/test/pub", "pub", "v1", false)
	require.NoError(t, park(pub))
	require.NoError(t, enable(pub))
	run("live public, nothing pending", pub)

	// The live-public-and-pending cell is unreachable by the ordinary route:
	// AddPackage refuses to park over a live public package.
	require.Error(t, park(csPkg("gno.land/r/test/pub", "pub", "v2", false)),
		"premise: AddPackage must refuse to park over a live PUBLIC package")

	// 4. live private, nothing pending.
	priv := csPkg("gno.land/r/test/priv", "priv", "v1", true)
	require.NoError(t, park(priv))
	require.NoError(t, enable(priv))
	run("live private, nothing pending", priv)

	// 5. live private with a redeploy parked, candidate IS the parked blob.
	//    The cell this PR fixes.
	r1 := csPkg("gno.land/r/test/redeploy", "redeploy", "v1", true)
	r2 := csPkg("gno.land/r/test/redeploy", "redeploy", "v2", true)
	require.NoError(t, park(r1))
	require.NoError(t, enable(r1))
	require.NoError(t, park(r2))
	run("live private + parked, candidate is the parked v2", r2)

	// 6. live private with a redeploy parked, candidate is the LIVE blob: a
	//    replay of the older AddPackage block during -start-height catch-up.
	s1 := csPkg("gno.land/r/test/stale", "stale", "v1", true)
	s2 := csPkg("gno.land/r/test/stale", "stale", "v2", true)
	require.NoError(t, park(s1))
	require.NoError(t, enable(s1))
	require.NoError(t, park(s2))
	run("live private + parked, candidate is the live v1", s1)

	// What the chain actually said about that cell, versus what reached the
	// status board. sim.Error is the typed error the board's reason carries;
	// sim.Log is the sentence EnablePackage wrote.
	probe, err := client.SignTx(std.Tx{
		Msgs: []std.Msg{vm.MsgEnablePackage{
			Approver: who, PkgPath: s1.Path, PkgHash: vm.PackageContentHash(s1),
		}},
		Fee: std.NewFee(o.blockMaxGas, std.MustParseCoin("1000000ugnot")),
	}, 0, 0)
	require.NoError(t, err)
	sim, simErr := client.SimulateResult(probe)
	require.NoError(t, simErr)
	fmt.Printf("\nstale-replay simulate:\n  sim.Error = %v\n  sim.Log   = %s\n", sim.Error, sim.Log)

	// The next candidate at the same path, which is what a catch-up run reaches
	// one block later. The board is keyed by path, so this overwrites the row
	// above.
	run("...then the parked v2 at the same path", s2)

	// 7. live private, nothing pending, candidate is a SUPERSEDED blob: the
	//    parked bytes were replaced before any enable landed.
	u1 := csPkg("gno.land/r/test/superseded", "superseded", "v1", true)
	u2 := csPkg("gno.land/r/test/superseded", "superseded", "v2", true)
	require.NoError(t, park(u1))
	require.NoError(t, park(u2)) // replaces the parked blob
	require.NoError(t, enable(u2))
	run("live private, nothing pending, candidate is the dead v1", u1)

	fmt.Println("\n| cell | vm/qpkgmeta_json before the candidate | old isActive | new isSettled | board status | board reason | spend charged |")
	fmt.Println("| --- | --- | --- | --- | --- | --- | --- |")
	for _, r := range rows {
		reason := r.reason
		if len(reason) > 110 {
			reason = reason[:110] + "..."
		}
		fmt.Printf("| %s | `%s` | %v | %v | %s | %s | %d |\n",
			r.cell, r.meta, r.old, r.new, r.status, reason, r.spent)
	}
}
