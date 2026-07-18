package caller

import (
	"testing"

	"gno.land/p/samcrew/basedao"
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
	"gno.land/r/test/daorlm"
	"gno.land/r/test/recorder"
)

// Executes an EditProfile proposal on the DAO owned by r/test/daorlm,
// passing THIS realm cur. Records which realm the profile setter observes.
func TestExecuteAttribution(cur realm, t *testing.T) {
	req := daokit.ProposalRequest{
		Title:       "edit profile",
		Description: "d",
		Action:      basedao.NewEditProfileAction([2]string{"Bio", "written-by-caller"}),
	}
	id := daorlm.DAO.Propose(req)
	daorlm.DAO.Vote(id, daocond.VoteYes)
	daorlm.DAO.Execute(id, cur)
	for _, s := range recorder.Get() {
		t.Log("setter observed:", s)
	}
	if len(recorder.Get()) == 0 {
		t.Fatal("no profile write recorded")
	}
}
