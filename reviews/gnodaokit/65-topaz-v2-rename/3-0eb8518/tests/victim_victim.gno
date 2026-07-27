// Package victim is an ordinary realm that wants to know whether an address is a
// member of some DAO. It follows the framework's own cross-realm membership
// recipe: hand the DAO handle and its own cur to MustGetMembersViewExtension.
package victim

import (
	"gno.land/p/samcrew/basedao"
	"gno.land/p/samcrew/daokit"
)

const Self = "gno.land/r/samcrew/authlens/victim"

// CheckMembership is the documented cross-realm membership check.
func CheckMembership(cur realm, dao daokit.DAO, who string) bool {
	view := basedao.MustGetMembersViewExtension(dao, cur)
	return view.IsMember(who)
}
