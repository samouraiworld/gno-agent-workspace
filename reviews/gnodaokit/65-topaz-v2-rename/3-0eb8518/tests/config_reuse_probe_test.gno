/* Run: from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08
cp <this file> gno/p/basedao/config_reuse_probe_test.gno
/tmp/gno-topaz test -v -run 'TestConfigReuse' ./gno/p/basedao
rm gno/p/basedao/config_reuse_probe_test.gno
*/

// New fills its defaults into the caller's Config rather than into a copy, so a
// second DAO built from the same Config starts with InitialCondition already set
// by the first call.
// At 0eb8518 the second DAO gets a 60% migration bar instead of 80%, and its
// default governance counts the first DAO's members.
// Both assertions pass once New stops writing back into conf.

package basedao

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
)

func TestConfigReuseKeepsTheMigrationBar(cur realm, t *testing.T) {
	conf := &Config{
		Name:             "d",
		Description:      "d",
		GetProfileString: func(addr address, field, def string) string { return "" },
		SetImplemFn:      func(daokit.DAO) {},
	}

	testing.SetOriginCaller(alice)
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/testing/dao1"))
	conf.Members = NewMembersStore(nil, []Member{
		{alice.String(), nil}, {bob.String(), nil}, {carol.String(), nil},
	})
	_, first := New(conf, cur)

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/testing/dao2"))
	conf.Members = NewMembersStore(nil, []Member{
		{alice.String(), nil}, {bob.String(), nil}, {carol.String(), nil},
	})
	_, second := New(conf, cur)

	firstMig := first.Core.Resources.Get(ActionChangeDAOImplementationKind).Condition
	secondMig := second.Core.Resources.Get(ActionChangeDAOImplementationKind).Condition

	println("first  migration condition: " + firstMig.Render())
	println("second migration condition: " + secondMig.Render())

	urequire.Equal(t, firstMig.Render(), secondMig.Render(),
		"two DAOs built from one default Config must get the same migration bar")

	// Two of three: enough for a default 60% action, short of the 80% a
	// migration is supposed to require.
	ballot := daocond.NewBallot()
	ballot.Vote(alice.String(), daocond.VoteYes)
	ballot.Vote(bob.String(), daocond.VoteYes)
	urequire.False(t, secondMig.Eval(ballot),
		"two of three must not carry a migration on the second DAO either")
}

func TestConfigReuseDoesNotBindTheFirstDAOsMembers(cur realm, t *testing.T) {
	conf := &Config{
		Name:             "d",
		Description:      "d",
		GetProfileString: func(addr address, field, def string) string { return "" },
	}

	testing.SetOriginCaller(alice)
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/testing/dao3"))
	conf.Members = NewMembersStore(nil, []Member{{alice.String(), nil}})
	_, _ = New(conf, cur)

	testing.SetRealm(testing.NewCodeRealm("gno.land/r/testing/dao4"))
	conf.Members = NewMembersStore(nil, []Member{
		{alice.String(), nil}, {bob.String(), nil}, {carol.String(), nil},
		{dave.String(), nil},
	})
	_, second := New(conf, cur)

	addMember := second.Core.Resources.Get(ActionAddMemberKind).Condition

	// alice alone is 1/4 of the second DAO, far below 60%.
	ballot := daocond.NewBallot()
	ballot.Vote(alice.String(), daocond.VoteYes)
	println("second DAO add-member with 1 of 4 yes: ", addMember.Eval(ballot))
	urequire.False(t, addMember.Eval(ballot),
		"one of four must not carry an action in the second DAO")
}
