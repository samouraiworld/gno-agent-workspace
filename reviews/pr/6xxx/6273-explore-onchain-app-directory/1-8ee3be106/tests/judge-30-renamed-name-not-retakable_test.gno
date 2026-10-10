package users

// judge-30: r/gnoland/store/v0 Claim authorises through r/sys/names, which
// grants a namespace to the CURRENT holder of that name. This asserts that a
// renamed (or deleted) user's old name can never be registered by another
// address, so no second address ever gains Claim rights over r/alice/*.
// Passes at 8ee3be106: the takeover path is unreachable.
//
// Repro from a plain clone of gnolang/gno:
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp judge-30-renamed-name-not-retakable_test.gno examples/gno.land/r/sys/users/judge30_test.gno
//   cd examples/gno.land/r/sys/users && gno test -v -run TestJudge30 .
//   rm judge30_test.gno

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

func TestJudge30RenamedNameNotRetakable(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm(initControllerPath))
	cleanStore(t)

	urequire.NoError(t, RegisterUser(cross(cur), "alice", aliceAddr))
	data := ResolveAddress(aliceAddr)
	testing.SetOriginCaller(whitelistedCallerAddr)
	urequire.NoError(t, data.UpdateName(0, cur, "alice2"))
	testing.SetRealm(testing.NewCodeRealm(initControllerPath))

	// old name no longer current: r/sys/names authorises nobody on r/alice/*
	_, isCurrent := ResolveName("alice")
	uassert.False(t, isCurrent)

	// a second address cannot take the old name
	err := RegisterUser(cross(cur), "alice", bobAddr)
	uassert.ErrorContains(t, err, ErrNameTaken.Error())
}

func TestJudge30DeletedNameNotRetakable(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm(initControllerPath))
	cleanStore(t)

	urequire.NoError(t, RegisterUser(cross(cur), "alice", aliceAddr))
	data := ResolveAddress(aliceAddr)
	testing.SetOriginCaller(whitelistedCallerAddr)
	urequire.NoError(t, data.Delete(0, cur))
	testing.SetRealm(testing.NewCodeRealm(initControllerPath))

	err := RegisterUser(cross(cur), "alice", bobAddr)
	uassert.ErrorContains(t, err, ErrNameTaken.Error())
}
