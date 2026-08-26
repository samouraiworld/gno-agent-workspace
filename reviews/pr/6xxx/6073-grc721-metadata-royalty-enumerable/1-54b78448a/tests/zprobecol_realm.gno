package zprobecol

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/p/demo/tokens/grc721/metadata"
	"gno.land/p/nt/seqid/v0"
)

var (
	meta *metadata.Metadata
	mled *metadata.Ledger
)

// Setup builds a collection owned by this realm and writes one metadata record.
func Setup(cur realm, holder address) {
	_, led := grc721.NewToken("Foo", "FOO", seqid.ID(0), cur)
	meta, mled = metadata.NewMetadata(led)
	if err := led.Mint(holder, "1"); err != nil {
		panic(err)
	}
	if err := mled.SetTokenMetadata("1", metadata.Data{
		Name:       "Sword",
		Attributes: []metadata.Trait{{TraitType: "rarity", Value: "common"}},
	}); err != nil {
		panic(err)
	}
}

// View hands the public read half to any caller, as a realm exposing a collection would.
func View() *metadata.Metadata { return meta }
