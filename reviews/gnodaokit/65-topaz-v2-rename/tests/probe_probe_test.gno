package probe

import (
	"testing"

	"gno.land/r/test/daorlm"
)

// Calls the DAO CallerID through the same non-crossing path Propose uses,
// from a DIFFERENT realm than the one that owns the DAO.
func TestCallerIdentity(cur realm, t *testing.T) {
	got := daorlm.Priv.CallerID()
	t.Log("CallerID seen by the DAO when r/test/probe calls in: [" + got + "]")
	t.Log("this realm is gno.land/r/test/probe; the DAO lives in gno.land/r/test/daorlm")
}
