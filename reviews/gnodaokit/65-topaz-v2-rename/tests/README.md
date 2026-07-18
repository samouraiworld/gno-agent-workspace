# Adversarial tests for gnodaokit PR 65

Four realm files reproducing two defects in the interrealm-v2 port commit `15dbc83`. They are not drop-in package tests: they need a gno checkout at the Topaz launch tip with the branch's packages vendored at their on-chain paths.

## Run

```bash
# from a local clone of gnolang/gno:
git fetch origin fc40526511474e40b8a66419f5ba28255085bc08 && git checkout fc4052651
git clone https://github.com/samouraiworld/gnodaokit /tmp/gdk
git -C /tmp/gdk fetch origin pull/65/head && git -C /tmp/gdk checkout FETCH_HEAD
git clone --depth 1 https://github.com/samouraiworld/samcrew-deployer /tmp/scd

EX=examples/gno.land
cp -r /tmp/scd/deps/avl      $EX/p/samcrew/avl
cp -r /tmp/gdk/gno/p/realmid $EX/p/samcrew/realmid
cp -r /tmp/gdk/gno/p/daocond $EX/p/samcrew/daocond
cp -r /tmp/gdk/gno/p/daokit  $EX/p/samcrew/daokit
cp -r /tmp/gdk/gno/p/basedao $EX/p/samcrew/basedao
find $EX/p/samcrew/daocond $EX/p/samcrew/daokit $EX/p/samcrew/basedao \
  -type f \( -name '*.gno' -o -name '*.toml' \) \
  -exec sed -i 's#/samcrew/daocond/v2#/samcrew/daocond#g; s#/samcrew/daokit/v2#/samcrew/daokit#g; s#/samcrew/basedao/v2#/samcrew/basedao#g' {} +

# then place each file from this directory, un-flattening the name:
#   recorder_recorder.gno  -> $EX/r/test/recorder/recorder.gno
#   daorlm_daorlm.gno      -> $EX/r/test/daorlm/daorlm.gno
#   caller_caller_test.gno -> $EX/r/test/caller/caller_test.gno
#   probe_probe_test.gno   -> $EX/r/test/probe/probe_test.gno
# each directory also needs a gnomod.toml declaring its module path.

cd examples && export GNOROOT=$(cd .. && pwd)
go run ../gnovm/cmd/gno test -v ./gno.land/r/test/probe
go run ../gnovm/cmd/gno test -v ./gno.land/r/test/caller
```

## What each file shows

`probe_probe_test.gno` calls `daorlm.Priv.CallerID()` from a different realm than the one owning the DAO. Observed at `0612859`:

```
CallerID seen by the DAO when r/test/probe calls in: []
```

Caller identity for a cross-realm caller is the empty string. `realmid.Previous()` reads `unsafe.PreviousRealm()`, which does not resolve a calling realm across the non-crossing `daokit.DAO` interface method.

`caller_caller_test.gno` proposes, votes and executes an EditProfile action on the DAO owned by `r/test/daorlm`, passing its own `cur`. `daorlm_daorlm.gno` registers a member whose address is the empty string, which `MembersStore.AddMember` accepts without validation. Observed at `0612859`:

```
setter observed: gno.land/r/test/daorlm|DisplayName=victim-dao
setter observed: gno.land/r/test/daorlm|Bio=d
setter observed: gno.land/r/test/daorlm|Avatar=
setter observed: gno.land/r/test/caller|Bio=written-by-caller
```

The first three lines are the DAO's own `basedao.New` init, correctly attributed to the DAO realm. The fourth is the attack: a foreign realm passed the member gate on the empty identity, and the profile write was attributed to `gno.land/r/test/caller` rather than to the DAO realm, because `Execute(id, rlm)` crosses under the realm the caller supplied.

Remove the empty-address member from `daorlm_daorlm.gno` and the same run panics `caller is not a member`, so the gate is the only thing standing between a foreign realm and the DAO.
