package vm

// judge-7: VMKeeper.Initialize on a store that lacks one stdlib of
// stdlibs.InitOrder(), as a store written by an earlier binary lacks a stdlib
// the next binary adds. Run at the PR head and at its merge base.
// Head: missing crypto/bls12381 panics (nil mempackage into TypeCheckMemPackage).
// Base: the same store boots, since crypto/bls12381 is not in InitOrder; a
// stdlib the base does list (crypto/bech32) panics the same way at both.
//
// Repro, from a plain clone of gnolang/gno:
//   git fetch origin pull/6375/head:pr6375 && git checkout 0ee53fefc
//   cp judge-7-upgrade-boot-base.go gno.land/pkg/sdk/vm/judge7_upgrade_boot_test.go
//   go test ./gno.land/pkg/sdk/vm -run 'TestJudge7UpgradeBoot' -count=1 -v -timeout 2m
//   rm gno.land/pkg/sdk/vm/judge7_upgrade_boot_test.go
//   # then the same three lines after: git checkout c32ca7b60187102d184cb6c1cc80b56f75d2dcd6

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/stdlibs"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestJudge7UpgradeBoot(t *testing.T) {
	for _, pkg := range []string{"crypto/bls12381", "crypto/bech32"} {
		t.Run(strings.ReplaceAll(pkg, "/", "_"), func(t *testing.T) {
			env := setupTestEnv()
			ms := env.ctx.MultiStore()
			ms.GetStore(env.vmk.iavlKey).Delete(nil, []byte("pkg:_/"+pkg))
			listed := false
			for _, p := range stdlibs.InitOrder() {
				listed = listed || p == pkg
			}
			vmk2 := NewVMKeeper(env.vmk.baseKey, env.vmk.iavlKey, env.acck, env.bankk, env.prmk)
			var rec any
			var where string
			func() {
				defer func() {
					if rec = recover(); rec != nil {
						for _, l := range strings.Split(string(debug.Stack()), "\n") {
							if strings.Contains(l, "gotypecheck.go") || strings.Contains(l, "sdk/vm/keeper.go") {
								where += strings.TrimSpace(l) + " "
							}
						}
					}
				}()
				vmk2.Initialize(log.NewNoopLogger(), ms.MultiCacheWrap())
			}()
			t.Logf("store lacks %s; in InitOrder: %v; boot panic: %v %s", pkg, listed, rec, where)
			// IS: a store lacking a listed stdlib panics the boot.
			// SHOULD: the boot loads or skips the missing stdlib and returns.
			if rec != nil {
				t.Errorf("boot panics on a store lacking %s", pkg)
			}
		})
	}
}
