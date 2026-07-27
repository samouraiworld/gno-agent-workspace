// Package evil implements daokit.DAO. Every method it declares receives a realm
// value chosen by its caller, so it pockets that realm and crosses under it.
package evil

import (
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
	"gno.land/r/samcrew/authlens/probe"
)

type evilDAO struct{}

func (e *evilDAO) Propose(req daokit.ProposalRequest, rlm realm) uint64 {
	probe.Record(cross(rlm))
	return 0
}

func (e *evilDAO) Vote(id uint64, vote daocond.Vote, rlm realm) {
	probe.Record(cross(rlm))
}

func (e *evilDAO) Execute(id uint64, rlm realm) {
	probe.Record(cross(rlm))
}

func (e *evilDAO) Render(path string) string { return "" }

func (e *evilDAO) Extension(path string, rlm realm) daokit.Extension {
	probe.Record(cross(rlm))
	return &fakeMembersView{}
}

func (e *evilDAO) ExtensionsList() daokit.ExtensionsList { return nil }

type fakeMembersView struct{}

func (f *fakeMembersView) Info() daokit.ExtensionInfo {
	return daokit.ExtensionInfo{Path: "gno.land/p/samcrew/basedao.MembersView", Version: "1"}
}

func (f *fakeMembersView) IsMember(id string) bool { return true }

// Handle hands the hostile DAO value to anyone who asks.
func Handle() daokit.DAO { return &evilDAO{} }
