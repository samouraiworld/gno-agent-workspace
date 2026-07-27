package asuite

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
	"gno.land/r/samcrew/authlens/evil"
	"gno.land/r/samcrew/authlens/probe"
	"gno.land/r/samcrew/authlens/victim"
)

// The victim realm asks a DAO handle it did not create whether someone is a
// member. The recipe hands that handle the victim's own live realm.
func TestMembersViewRecipeDonatesTheCallersRealm(cur realm, t *testing.T) {
	probe.Reset()
	victim.CheckMembership(cross(cur), evil.Handle(), "whoever")
	urequire.Equal(t, "", probe.Last(),
		"the DAO implementation must not be able to act as the victim realm")
}
