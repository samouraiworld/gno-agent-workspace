// Helper realm for b2-catalog-view-hook-leak_filetest.gno; repro commands are in that file.
// Package issuer6075 is a collection whose extension passes one object as both
// the hook and the view, the shape grc721reg_test's stubView uses.
package issuer6075

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
)

type counter struct {
	tokenID string
	mints   int
	burns   int
}

func (c *counter) ExtensionKind() string                         { return "counter" }
func (c *counter) TokenID() string                               { return c.tokenID }
func (c *counter) OnMint(to address, tid grc721.TokenID)         { c.mints++ }
func (c *counter) OnTransfer(from, to address, tid grc721.TokenID) {}
func (c *counter) OnBurn(tid grc721.TokenID)                     { c.burns++ }

var (
	token  *grc721.Token
	ledger *grc721.PrivateLedger
	ext    *counter
	Key    string
)

func init(cur realm) {
	token, ledger = grc721.NewToken("Leak", "LEAK", 0, cur)
	ext = &counter{tokenID: token.ID()}
	ledger.RegisterExtension(ext, ext)
	if err := ledger.Mint(address("g1wdhk6et0dej47h6lta047h6lta047h6lrnerlk"), "1"); err != nil {
		panic(err)
	}
	Key = grc721reg.Register(cross(cur), token, "")
}

func Mints() int { return ext.mints }
func Burns() int { return ext.burns }
func Supply() int64 { return token.TotalSupply() }
