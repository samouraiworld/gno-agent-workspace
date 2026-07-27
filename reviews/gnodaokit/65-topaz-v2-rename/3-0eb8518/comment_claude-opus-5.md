# Review: PR [#65](https://github.com/samouraiworld/gnodaokit/pull/65)
Posted: https://github.com/samouraiworld/gnodaokit/pull/65#pullrequestreview-4786429417
Event: COMMENT

## Body
[AI bot]

Automated pass over the diff. Technical checks only: no design judgement and no merge verdict.

Verified on 0eb8518: dropping the `IsCurrent()` clause reddens exactly the two dead-frame tests and dropping the pkgpath comparison reddens six, so both halves of the gate carry weight. A migration authored in a different realm still produces a DAO bound to the host realm, checked across three real realms.

Two process notes:

- The description says 7 files, +10/−9. The head is 225 files, +26323/−447, and it re-signatures `daokit.DAO`, `daokit.ActionHandler`, `basedao.Config`, `CallerIDFn`, `MigrateFn` and `SetImplemRaw`. The CI retool it lists as a follow-up has landed.
- [#67](https://github.com/samouraiworld/gnodaokit/pull/67) through [#72](https://github.com/samouraiworld/gnodaokit/pull/72) were merged into this branch rather than into `main` and carry no reviews, so merging this is the only gate any of them passes. Deliberate?

Five more sit on lines this diff does not touch, so they have no inline anchor:

- [`view_proposal_detail_page.gno:32-34`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/view_proposal_detail_page.gno#L32-L34) reads `proposal.Title` straight off a nil `GetProposal`, so `proposal/0` and any unused id abort on the cross-realm handle gnoweb drives. `proposal/<non-numeric>` aborts in [`MuxProposalDetailPage`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/render.gno#L75-L81) and `role/<absent name>` in [`RoleInfo`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/members.gno#L65). All four predate this branch, and `7a18b35` here sweeps that class.
- [`gno/p/basedao/README.md:186-192`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/README.md?plain=1#L186-L192) offers `proposals/1` and `roles`, which both 404; the router registers `proposal/{id}` and `role/{name}` at [`render.gno:31-39`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/render.gno#L31-L39). Three registered paths are missing from the list.
- [`gno/p/basedao/README.md:281-310`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/README.md?plain=1#L281-L310) and [`README.md:206-235`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/README.md?plain=1#L206-L235) both omit `MigrationCondition`, whose own comment tells a caller who sets `InitialCondition` to set it too.
- [`cond_role_treshold.gno`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/daocond/cond_role_treshold.gno#L1) misspells `treshold` in its filename and in the new `cond_role_treshold_test.gno`; the identifiers inside are correct. File names are what `vm/qfile` and gnoweb list, so this freezes with the rest.
- [`Makefile:22`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/Makefile#L22) globs `gno.mod`, of which the repo has zero; all 16 manifests are `gnomod.toml`, so the `gno mod tidy` step and the no-diff check after it assert nothing.

<details><summary>repro</summary>

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08

cat > gno/p/basedao/render_probe_test.gno <<'EOF'
package basedao

import "testing"

func TestRenderProbe(cur realm, t *testing.T) {
	tdao := newTestingDAO(cur, t, 0.6, []Member{
		{alice.String(), []string{"admin"}}, {bob.String(), []string{}},
	})
	tdao.dao.Propose(tdao.mockProposalRequest, cur)

	for _, p := range []string{"proposal/1", "proposal/0", "proposal/999", "proposal/abc", "role/admin", "role/nope"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					println("PANIC  " + p)
				}
			}()
			tdao.privdao.Render(p)
			println("ok     " + p)
		}()
	}
}
EOF

/tmp/gno-topaz test -v -run 'TestRenderProbe' ./gno/p/basedao
rm gno/p/basedao/render_probe_test.gno
```

```
ok     proposal/1
PANIC  proposal/0
PANIC  proposal/999
PANIC  proposal/abc
ok     role/admin
PANIC  role/nope
```
</details>

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnodaokit/65-topaz-v2-rename/3-0eb8518/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gno/p/daokit/daokit.gno:26-43 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/daokit.gno#L26-L43) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664180)
Critical: every `DAO` method takes the caller's realm, so a realm querying a DAO it does not own hands the implementation a capability to act as itself. `assertRealmIsOwn` guards the callee only, and the interface is satisfiable by anyone: [`MustGetMembersViewExtension`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/members_extension.gno#L18-L19) forwards the caller's `rlm` straight into it. Taking the caller's `address` instead removes the transfer; a doc line cannot, because the interface is what third-party implementations conform to.

<details><summary>repro</summary>

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08

mkdir -p gno/r/authlens/{probe,victim,evil,asuite}
for d in probe victim evil asuite; do
  printf 'module = "gno.land/r/samcrew/authlens/%s"\ngno = "0.9"\n' "$d" > "gno/r/authlens/$d/gnomod.toml"
done

cat > gno/r/authlens/probe/probe.gno <<'EOF'
package probe

var last string

func Record(cur realm) {
	p := cur.Previous()
	if p.IsUser() {
		last = "user:" + p.Address().String()
	} else {
		last = p.PkgPath()
	}
}

func Last() string { return last }
func Reset()       { last = "" }
EOF

cat > gno/r/authlens/evil/evil.gno <<'EOF'
package evil

import (
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
	"gno.land/r/samcrew/authlens/probe"
)

type evilDAO struct{}

func (e *evilDAO) Propose(req daokit.ProposalRequest, rlm realm) uint64 { probe.Record(cross(rlm)); return 0 }
func (e *evilDAO) Vote(id uint64, vote daocond.Vote, rlm realm)         { probe.Record(cross(rlm)) }
func (e *evilDAO) Execute(id uint64, rlm realm)                         { probe.Record(cross(rlm)) }
func (e *evilDAO) Render(path string) string                            { return "" }
func (e *evilDAO) ExtensionsList() daokit.ExtensionsList                 { return nil }

func (e *evilDAO) Extension(path string, rlm realm) daokit.Extension {
	probe.Record(cross(rlm))
	return &fakeMembersView{}
}

type fakeMembersView struct{}

func (f *fakeMembersView) Info() daokit.ExtensionInfo {
	return daokit.ExtensionInfo{Path: "gno.land/p/samcrew/basedao.MembersView", Version: "1"}
}
func (f *fakeMembersView) IsMember(id string) bool { return true }

func Handle() daokit.DAO { return &evilDAO{} }
EOF

cat > gno/r/authlens/victim/victim.gno <<'EOF'
package victim

import (
	"gno.land/p/samcrew/basedao"
	"gno.land/p/samcrew/daokit"
)

// The documented cross-realm membership check.
func CheckMembership(cur realm, dao daokit.DAO, who string) bool {
	view := basedao.MustGetMembersViewExtension(dao, cur)
	return view.IsMember(who)
}
EOF

cat > gno/r/authlens/asuite/asuite_test.gno <<'EOF'
package asuite

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
	"gno.land/r/samcrew/authlens/evil"
	"gno.land/r/samcrew/authlens/probe"
	"gno.land/r/samcrew/authlens/victim"
)

func TestMembersViewRecipeDonatesTheCallersRealm(cur realm, t *testing.T) {
	probe.Reset()
	victim.CheckMembership(cross(cur), evil.Handle(), "whoever")
	urequire.Equal(t, "", probe.Last(),
		"the DAO implementation must not be able to act as the victim realm")
}
EOF

/tmp/gno-topaz test -v ./gno/r/authlens/asuite
rm -rf gno/r/authlens
```

```
=== RUN   TestMembersViewRecipeDonatesTheCallersRealm
uassert.Equal: strings are different
	Diff: [+gno.land/r/samcrew/authlens/victim] - the DAO implementation must not be able to act as the victim realm
--- FAIL: TestMembersViewRecipeDonatesTheCallersRealm (0.00s)
```

A third realm saw the victim as its caller.
</details>

## .github/workflows/vendored-provenance.yml:81 [↗](../../../../.worktrees/gnodaokit-review-65/.github/workflows/vendored-provenance.yml#L81) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664187)
Critical: the exemption is a directory prefix, but the compiler resolves a package by the `module` line in its `gnomod.toml`. A directory under the exempt prefix can therefore claim any package path and become what the build compiles, while the job reports no drift and lint stays clean.

<details><summary>repro</summary>

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08

git clone --filter=blob:none --no-checkout https://github.com/gnolang/gno /tmp/gnoup
git -C /tmp/gnoup checkout --quiet fc40526511474e40b8a66419f5ba28255085bc08 -- examples

# A shadow package under the exempt prefix, claiming piechart's path. Its Render
# drops the title parameter, so whichever copy compiles is observable.
mkdir -p vendored/gno.land/p/samcrew/avl/shadow
printf 'module = "gno.land/p/samcrew/piechart"\ngno = "0.9"\n' > vendored/gno.land/p/samcrew/avl/shadow/gnomod.toml
sed 's|^func Render(slices \[\]PieSlice, title string) string {|func Render(slices []PieSlice) string {\n\ttitle := "SHADOWED"|' \
  vendored/gno.land/p/samcrew/piechart/piechart.gno > vendored/gno.land/p/samcrew/avl/shadow/piechart.gno

# The workflow's own byte-comparison loop.
drift=0; checked=0
while IFS= read -r f; do
  rel="${f#vendored/}"
  case "$rel" in
    gno.land/p/samcrew/avl/*) continue ;;
    README.md|NOTICE|LICENSE.md) continue ;;
  esac
  checked=$((checked + 1))
  if [ ! -f "/tmp/gnoup/examples/$rel" ]; then echo "MISSING UPSTREAM: $rel"; drift=$((drift+1));
  elif ! cmp -s "$f" "/tmp/gnoup/examples/$rel"; then echo "DRIFT: $rel"; drift=$((drift+1)); fi
done < <(find vendored ! -type d)
echo "provenance: checked $checked, drift $drift"

/tmp/gno-topaz lint ./gno/...

rm -rf vendored/gno.land/p/samcrew/avl/shadow /tmp/gnoup
```

```
provenance: checked 118, drift 0
gno/p/basedao/view_members_page.gno:40:35: too many arguments in call to piechart.Render
	have ([]piechart.PieSlice, string)
	want ([]piechart.PieSlice) (code=gnoTypeCheckError)
```

Provenance passes, and the compile error is against the shadow's signature, so the shadow is what the build resolved. Keep the signature identical and the substitution is silent: with a byte-copy of piechart parked there, provenance reports the same `drift 0`, `lint ./gno/...` exits 0 and `test ./gno/p/basedao` is ok.
</details>

## gno/p/basedao/basedao.gno:206 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/basedao.gno#L206) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664190)
`New` writes its defaults back into the caller's `Config`, so a second DAO built from the same `Config` finds `conf.InitialCondition` already set at line 208 and its migration bar falls from 80% to the first DAO's 0.6 default, bound to the first DAO's member store. The comment at [line 230](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/basedao.gno#L230-L233) states the opposite invariant and names this exact hazard. `conf.CallerID` at [line 170](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/basedao.gno#L170) and `conf.MigrationParamsFn` at [line 227](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/basedao.gno#L227) are written back too.

<details><summary>repro</summary>

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08

cat > gno/p/basedao/config_reuse_probe_test.gno <<'EOF'
package basedao

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
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
	urequire.Equal(t, firstMig.Render(), secondMig.Render(),
		"two DAOs built from one default Config must get the same migration bar")
}
EOF

/tmp/gno-topaz test -v -run 'TestConfigReuseKeepsTheMigrationBar' ./gno/p/basedao
rm gno/p/basedao/config_reuse_probe_test.gno
```

```
=== RUN   TestConfigReuseKeepsTheMigrationBar
uassert.Equal: strings are different
	Diff: [-8][+6]0% of members - two DAOs built from one default Config must get the same migration bar
--- FAIL: TestConfigReuseKeepsTheMigrationBar (0.01s)
```
</details>

## .github/workflows/vendored-provenance.yml:164-177 [↗](../../../../.worktrees/gnodaokit-review-65/.github/workflows/vendored-provenance.yml#L164-L177) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664195)
The fork is checked only by grepping its `Get` signature, so a change to the read path every member, role and proposal lookup goes through passes every gate. [`vendored/README.md:36`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/vendored/README.md?plain=1#L35-L36) names `samouraiworld/samcrew-deployer` as the guard that pins the fork; that repository is private, so a reader of this one cannot verify the claim either.

## gno/p/daokit/daokit.gno:137-140 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/daokit.gno#L137-L140) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664201)
Marking the proposal before the handler runs makes a failed execution indistinguishable from a successful one. A host realm that recovers, the case the comment above warns about, leaves the proposal stored as `Executed`, rendering `Status - Executed`, out of the active list and in history, with its action never applied. At the merge-base the same failure left it visibly stuck as `Passed` in the active list.

<details><summary>repro</summary>

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08

mkdir -p gno/r/burnprobe/{host,suite}
printf 'module = "gno.land/r/burnprobe/host"\ngno = "0.9"\n' > gno/r/burnprobe/host/gnomod.toml
printf 'module = "gno.land/r/burnprobe/suite"\ngno = "0.9"\n' > gno/r/burnprobe/suite/gnomod.toml

cat > gno/r/burnprobe/host/host.gno <<'EOF'
package host

import (
	"gno.land/p/samcrew/basedao"
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
)

const SuiteID = "gno.land/r/burnprobe/suite"

var (
	localDAO   daokit.DAO
	daoPrivate *basedao.DAOPrivate
)

func init(cur realm) {
	members := basedao.NewMembersStore(nil, []basedao.Member{{Address: SuiteID}})
	cond := daocond.MembersThreshold(0.1, members.IsMember, members.MembersCount)
	localDAO, daoPrivate = basedao.New(&basedao.Config{
		Name: "Burn DAO", Description: "burn probe",
		Members: members, InitialCondition: cond,
		GetProfileString: func(addr address, field, def string) string { return "" },
		PrivateVarName:   "daoPrivate",
		NoCreationEvent:  true,
	}, cur)
}

func Propose(cur realm, req daokit.ProposalRequest) uint64 { return localDAO.Propose(req, cur) }
func Vote(cur realm, id uint64, v daocond.Vote)            { localDAO.Vote(id, v, cur) }

// The recover the daokit.gno comment warns about.
func ExecuteRecovering(cur realm, id uint64) (caught string) {
	defer func() {
		if r := recover(); r != nil {
			caught = "recovered"
		}
	}()
	localDAO.Execute(id, cur)
	return "no panic"
}

func StatusOf(id uint64) string { return daoPrivate.Core.Proposals.GetProposal(id).Status.String() }
func Detail() string            { return daoPrivate.Render("proposal/1") }
func JSON() string              { return daoPrivate.Core.Proposals.GetProposalsJSON() }

// Make the pending AddMember fail.
func SeedDuplicate(cur realm, who string) { daoPrivate.Members.AddMember(who, nil) }
EOF

cat > gno/r/burnprobe/suite/suite_test.gno <<'EOF'
package suite

import (
	"strings"
	"testing"

	"gno.land/p/samcrew/basedao"
	"gno.land/p/samcrew/daocond"
	"gno.land/p/samcrew/daokit"
	"gno.land/r/burnprobe/host"
)

const target = "g1v9kxjcm9ta047h6lta047h6lta047h6lzd40gh"

func TestBurn(cur realm, t *testing.T) {
	id := host.Propose(cross(cur), daokit.ProposalRequest{
		Title:  "add a member",
		Action: basedao.NewAddMemberAction(&basedao.ActionAddMember{Address: target}),
	})
	host.Vote(cross(cur), id, daocond.VoteYes)
	host.SeedDuplicate(cross(cur), target)

	println("recover() around Execute -> " + host.ExecuteRecovering(cross(cur), id))
	println("stored status            -> " + host.StatusOf(id))
	println("detail shows Executed    -> ", strings.Contains(host.Detail(), "Status - Executed"))
	println("json says Executed       -> ", strings.Contains(host.JSON(), "\"status\":\"Executed\""))
}
EOF

/tmp/gno-topaz test -v ./gno/r/burnprobe/suite
rm -rf gno/r/burnprobe
```

```
recover() around Execute -> recovered
stored status            -> Executed
detail shows Executed    ->  true
json says Executed       ->  true
--- PASS: TestBurn (0.02s)
```
</details>


## vendored/NOTICE:9 [↗](../../../../.worktrees/gnodaokit-review-65/vendored/NOTICE#L9) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664208)
The attribution records `2c7f1abe`; [`Makefile:1`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/Makefile#L1) pins `fc40526`, and 15 vendored files differ between the two. [Lines 19-22](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/vendored/NOTICE?plain=1#L19-L22) claim CI re-derives the sha every run, and no job reads the file.



## gno/p/daokit/action_spoofing_test.gno:26 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/action_spoofing_test.gno#L26) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664213)
Missing test: all three tests build their handler inline, so nothing drives a foreign payload through a registered built-in handler. Breaking `NewAddMemberHandler` into the interface assertion this file warns against leaves basedao, daokit, the identity suite and all three demos green.

<details><summary>test cases</summary>

```go
// gno/p/basedao/handler_contract_test.gno
package basedao

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
	"gno.land/p/samcrew/daokit"
)

type lookalikeAddMember struct {
	Address address
	Roles   []string
}

func (l *lookalikeAddMember) String() string { return "lookalike" }

// A registered handler keys on its own concrete payload type, so a foreign
// payload carrying the same fields is refused rather than executed.
func TestRegisteredHandlerRefusesALookalikePayload(cur realm, t *testing.T) {
	tdao := newTestingDAO(cur, t, 0.6, []Member{{alice.String(), nil}})
	handler := tdao.privdao.Core.Resources.Get(ActionAddMemberKind).Handler

	urequire.PanicsWithMessage(t, cur, "invalid payload type", func() {
		handler.Execute(daokit.NewAction(ActionAddMemberKind, &lookalikeAddMember{
			Address: dave,
		}), cur)
	})
}
```
</details>

## vendored/README.md:35-36 [↗](../../../../.worktrees/gnodaokit-review-65/vendored/README.md#L35-L36) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664221)
Nit: the fork is pinned at `f3d5a5d13` and the delta is described as the two-value `Get`. It also trails `d2737d84e`, so the `GetByIndex` negative-index panic, the `TraverseByOffset` negative-offset clamp and the `GetPageWithSize` non-positive page-size panic are all absent from a package published permanently. Latent, since `Pager.ParseQuery` clamps both parameters and both call sites pass a constant page size.

## gno/r/daoidentity/suite/suite_test.gno:120-122 [↗](../../../../.worktrees/gnodaokit-review-65/gno/r/daoidentity/suite/suite_test.gno#L120-L122) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664228)
Nit: swapping the gate and `CallerID` in `Propose` alone also reddens `TestInstantExecuteIsSameRealmOnly`, which wants `realm mismatch: ...` and gets `caller id is empty`.

## gno/p/basedao/view_proposals_page.gno:29-30 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/view_proposals_page.gno#L29-L30) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664232)
Nit: this says only `Execute` writes `Status`. `1eabd78` re-exported `UpdateStatus`, which also does.



## gno/p/daokit/proposals.gno:130 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/proposals.gno#L130) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664240)
Suggestion: this freezes as a method its own doc says nothing in the package may call, that must not run while rendering, and that is safe only while `Vote` and `Execute` both key on `Executed` alone. Its only caller in the tree is the probe at [`dao.gno:199`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/r/daoidentity/dao/dao.gno#L199), added to show calling it is harmless. Publishing makes that constraint permanent and enforced only by a comment.

## gno/p/realmid/realmid.gno:9 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/realmid/realmid.gno#L9) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664244)
Suggestion: nothing in the tree imports `realmid` since basedao stopped, and [`README.md:21-32`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/realmid/README.md?plain=1#L21-L32) heads a block "Not for caller authentication". `Previous` and `Current` still freeze at a published path as bare stack walks; publishing only `IsPackage` and `IsUser` would leave nothing to reach for by mistake.

## gno/p/basedao/basedao.gno:308-310 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/basedao.gno#L308-L310) [posted](https://github.com/samouraiworld/gnodaokit/pull/65#discussion_r3656664248)
Suggestion: only the length is checked, so a member who sends `YES` has it stored and counted in the ballot total while every tally reads `0/3`. Pre-existing, raised because `8c8ef04` typed the three constants so a misuse is a compile error, and a runtime value cannot be typed after publication.
