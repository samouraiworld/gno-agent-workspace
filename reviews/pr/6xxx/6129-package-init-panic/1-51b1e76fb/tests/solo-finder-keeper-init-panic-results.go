package vm

// Package-level initializer panics through the keeper: error text, the
// DeliverTx Error that ABCIResults hashes, and gas, per case.
//
// Repro from a plain clone of gnolang/gno:
//   git fetch origin pull/6129/head && git checkout 51b1e76fb3c69b6ff52e27d950fdd7392d20da2b
//   cp <this file> gno.land/pkg/sdk/vm/solo_finder_initpanic_test.go
//   go test ./gno.land/pkg/sdk/vm -run TestSoloFinderInitPanic -count=1 -v | grep CASE
//   git checkout 1fc4c140ec4064e8d66cb9828ddea280a723cca6 -- gnovm/pkg/gnolang/machine.go   # merge-base VM
//   go test ./gno.land/pkg/sdk/vm -run TestSoloFinderInitPanic -count=1 -v | grep CASE

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestSoloFinderInitPanic(t *testing.T) {
	cases := []struct {
		name string
		body string
		run  bool
	}{
		{"addpkg_index_direct", "package test\n\nvar a = \"\"\nvar A = a[0]\n", false},
		{"addpkg_index_viafunc", "package test\n\nvar a = \"\"\nvar A = f()\n\nfunc f() byte { return a[0] }\n", false},
		{"addpkg_divzero_direct", "package test\n\nvar z int\nvar A = 1 / z\n", false},
		{"addpkg_nilderef_direct", "package test\n\nvar p *int\nvar A = *p\n", false},
		{"run_index_direct", "package main\n\nvar a = \"\"\nvar A = a[0]\n\nfunc main() {}\n", true},
		{"run_index_viafunc", "package main\n\nvar a = \"\"\nvar A = f()\n\nfunc f() byte { return a[0] }\n\nfunc main() {}\n", true},
	}
	for _, c := range cases {
		env := setupTestEnv()
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		addr := crypto.AddressFromPreimage([]byte("addr1"))
		acc := env.acck.NewAccountWithAddress(ctx, addr)
		env.acck.SetAccount(ctx, acc)
		env.bankk.SetCoins(ctx, addr, initialBalance)
		const pkgPath = "gno.land/r/test"
		files := []*std.MemFile{
			{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
			{Name: "test.gno", Body: c.body},
		}
		g0 := ctx.GasMeter().GasConsumed()
		var err error
		escaped := ""
		func() {
			defer func() {
				if r := recover(); r != nil {
					escaped = fmt.Sprintf("%T %v", r, r)
				}
			}()
			if c.run {
				_, err = env.vmk.Run(ctx, NewMsgRun(addr, std.MustParseCoins(""), files))
			} else {
				err = env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files))
			}
		}()
		gas := ctx.GasMeter().GasConsumed() - g0
		aerr := abci.ABCIErrorOrStringError(err)
		h := sha256.Sum256(bft.ABCIResults{bft.ABCIResult{Error: aerr}}.Hash())
		first := ""
		if err != nil {
			first = strings.SplitN(err.Error(), "\n", 2)[0]
		}
		fmt.Printf("CASE %-24s escaped=%q gas=%d resultsHash=%x abciErr=%q errLine1=%q\n",
			c.name, escaped, gas, h[:6], fmt.Sprint(aerr), first)
	}
}
