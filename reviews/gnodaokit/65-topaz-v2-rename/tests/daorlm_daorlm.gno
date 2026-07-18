package daorlm

import (
	"gno.land/p/samcrew/basedao"
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
	"gno.land/r/test/recorder"
)

// DAO is exported on purpose: this is the shape the README Quick Start
// still teaches, and the shape the sub-DAO InstantExecute pattern needs.
var DAO daokit.DAO
var Priv *basedao.DAOPrivate

func init(cur realm) {
	store := basedao.NewMembersStore(nil, []basedao.Member{{Address: "", Roles: []string{}}})
	cond := daocond.MembersThreshold(0.1, store.IsMember, store.MembersCount)
	DAO, Priv = basedao.New(&basedao.Config{
		Name:             "victim-dao",
		Description:      "d",
		Members:          store,
		InitialCondition: cond,
		GetProfileString: recorder.GetString,
		SetProfileString: recorder.SetString,
		NoDefaultHandlers: false,
	}, cur)
}
