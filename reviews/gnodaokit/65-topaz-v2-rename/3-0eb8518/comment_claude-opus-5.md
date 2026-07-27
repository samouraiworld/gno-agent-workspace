# Review: PR [#65](https://github.com/samouraiworld/gnodaokit/pull/65)
Event: REQUEST_CHANGES

## Body
The caller-identity work holds. Reproduced on 0eb8518: deleting the `IsCurrent()` clause reddens exactly the two dead-frame tests and deleting the pkgpath comparison reddens six, so both halves of the gate carry weight, and a migration authored in a different realm still produces a DAO bound to the host realm across three real realms.

Two things the gate cannot reach are inline. Both freeze on merge, which is what makes them blocking rather than follow-ups.

- The description describes 7 files and +10/−9. The head is 225 files and +26323/−447, it re-signatures `daokit.DAO`, `daokit.ActionHandler`, `basedao.Config`, `CallerIDFn`, `MigrateFn` and `SetImplemRaw`, and the CI retool it lists as a follow-up has landed. A reviewer reading the description approves a different change.
- [#67](https://github.com/samouraiworld/gnodaokit/pull/67) through [#72](https://github.com/samouraiworld/gnodaokit/pull/72) were merged into this branch rather than into `main` and carry no reviews, so merging this is the only gate any of them passes. Was that deliberate, or should they land on `main` separately first?

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnodaokit/65-topaz-v2-rename/3-0eb8518/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gno/p/daokit/daokit.gno:26-43 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/daokit.gno#L26-L43)
Critical: every `DAO` method takes the caller's realm, so a realm that queries a DAO it does not own hands the implementation a capability to act as itself. `assertRealmIsOwn` guards the callee, and only when the callee is a basedao DAO; the interface is satisfiable by anyone, and [`MustGetMembersViewExtension`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/members_extension.gno#L18-L19) forwards the caller's `rlm` straight into it. Taking the caller's `address`, or keeping the private lookup off the interface, removes the transfer; a doc line cannot, because the interface is what third-party implementations conform to.

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
</details>

## .github/workflows/vendored-provenance.yml:81 [↗](../../../../.worktrees/gnodaokit-review-65/.github/workflows/vendored-provenance.yml#L81)
Critical: the exemption is a directory prefix, but the compiler resolves a package by the `module` line in its `gnomod.toml`, so a directory placed under the exempt prefix can claim any package path and become what the build compiles while the job reports no drift. The step's own comment states the guarantee this defeats: byte-for-byte in both directions, so `vendored/` cannot drift unnoticed. Deciding the exemption on the declared module path instead of the directory closes it.

<details><summary>repro</summary>

```bash
# from a local clone of samouraiworld/gnodaokit:
gh pr checkout 65 -R samouraiworld/gnodaokit
go build -o /tmp/gno-topaz github.com/gnolang/gno/gnovm/cmd/gno@fc40526511474e40b8a66419f5ba28255085bc08

git clone --filter=blob:none --no-checkout https://github.com/gnolang/gno /tmp/gnoup
git -C /tmp/gnoup checkout --quiet fc40526511474e40b8a66419f5ba28255085bc08 -- examples

# A shadow package, parked under the exempt prefix, claiming another package's path.
# Its Render drops the title parameter, so whichever copy compiles is observable.
mkdir -p vendored/gno.land/p/samcrew/avl/shadow
printf 'module = "gno.land/p/samcrew/piechart"\ngno = "0.9"\n' > vendored/gno.land/p/samcrew/avl/shadow/gnomod.toml
sed 's|^func Render(slices \[\]PieSlice, title string) string {|func Render(slices []PieSlice) string {\n\ttitle := "SHADOWED"|' \
  vendored/gno.land/p/samcrew/piechart/piechart.gno > vendored/gno.land/p/samcrew/avl/shadow/piechart.gno

# The workflow's own byte-comparison loop, verbatim.
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

Provenance passes. The compile error is against the shadow's signature, so the shadow is what the build resolved for `gno.land/p/samcrew/piechart`. Keep the signature identical and it is a silent substitution.
</details>

## gno/p/basedao/basedao.gno:206 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/basedao.gno#L206)
`New` writes its defaults back into the caller's `Config`, so a second DAO built from the same `Config` reads the first call's leftovers: `conf.InitialCondition` is already set at line 208, `governanceIsCallerChosen` is therefore true, and the migration bar falls to the first DAO's 0.6 threshold bound to the first DAO's member store. The comment at [line 230](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/basedao.gno#L230-L233) states the opposite invariant and names this exact hazard. `conf.CallerID` at [line 170](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/basedao.gno#L170) and `conf.MigrationParamsFn` at [line 227](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/basedao.gno#L227) are written back too.

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

## .github/workflows/vendored-provenance.yml:164-177 [↗](../../../../.worktrees/gnodaokit-review-65/.github/workflows/vendored-provenance.yml#L164-L177)
The exempt avl fork is verified only by grepping for its `Get` signature, so a change to the read path every member, role and proposal lookup goes through passes every gate. Inserting `if key == "backdoor" { return nil, false }` into `Tree.Get` passes all four steps. [`vendored/README.md`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/vendored/README.md?plain=1#L36-L38) points at `samouraiworld/samcrew-deployer` as the compensating control, but nothing here executes it and that repository is private, so a reader of this one cannot verify the claim either.

## vendored/README.md:36-38 [↗](../../../../.worktrees/gnodaokit-review-65/vendored/README.md#L36-L38)
The fork is described everywhere as upstream avl plus a two-value `Get`, but it is upstream `p/nt/avl/v0` at `f3d5a5d13` with none of the input checks `d2737d84e fix(avl): add missing checks` added before the pinned ref: the `GetByIndex` negative-index panic, the `TraverseByOffset` negative-offset clamp and the `GetPageWithSize` non-positive page-size panic are all absent. Latent today, since `Pager.ParseQuery` clamps both parameters and both call sites pass a constant page size. The next person to reason about the fork will reason about the wrong delta.

## vendored/NOTICE:9 [↗](../../../../.worktrees/gnodaokit-review-65/vendored/NOTICE#L9)
The attribution records `2c7f1abe` while [`Makefile:1`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/Makefile#L1) pins `fc40526`, and 15 vendored files differ between the two. [Lines 19-22](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/vendored/NOTICE?plain=1#L19-L22) claim CI re-derives the sha every run, and no job reads the file.

## gno/p/daokit/daokit.gno:137-140 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/daokit.gno#L137-L140)
Marking the proposal before the handler runs makes a failed execution indistinguishable from a successful one: with a host realm that survives the abort, the proposal stores `Executed`, renders `Status - Executed`, leaves the active list and appears in history with its action never applied. The burn predates this branch, but at the merge-base it left the proposal visibly stuck as `Passed` in the active list. A flag set before and cleared after the handler blocks the same re-entrancy without moving the status write.

## gno/p/daokit/daokit.gno:132-136 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/daokit.gno#L132-L136)
Nit: the comment names `recover()` as the hazard, but `recover()` cannot catch a panic unwinding across a realm boundary on this toolchain. `revive()` is what catches it, and [`custom_resource_test.gno:104`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/r/daodemo/custom_resource/custom_resource_test.gno#L104) already wraps `Execute` in it.

## gno/p/basedao/view_proposal_detail_page.gno:32-34 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/view_proposal_detail_page.gno#L32-L34)
`GetProposal` returns nil for an absent id and `proposal.Title` is read two lines later, so `proposal/0` and any unused id abort on the cross-realm handle gnoweb drives. `proposal/<non-numeric>` aborts in [`MuxProposalDetailPage`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/render.gno#L75-L81) and `role/<absent name>` in [`RoleInfo`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/members.gno#L65). All four predate the branch, and `7a18b35` here is a sweep of this class.

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

## gno/p/basedao/README.md:186-192 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/README.md#L186-L192)
The list offers `proposals/1` and `roles`, which return 404; the router registers `proposal/{id}` and `role/{name}` at [`render.gno:31-39`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/render.gno#L31-L39). Three registered paths are missing from the list.

## gno/p/basedao/README.md:281-310 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/README.md#L281-L310)
The `Config` listing omits `MigrationCondition`, and so does the one at [`README.md:206-235`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/README.md?plain=1#L206-L235). Its own comment tells a caller who sets `InitialCondition` to set it too, which is what the documented upgrade path at [§5.1](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/basedao/README.md?plain=1#L317) does.

## gno/p/daokit/action_spoofing_test.gno:26 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/action_spoofing_test.gno#L26)
Missing test: nothing drives a foreign payload through a registered built-in handler, because all three spoofing tests build their handler inside the test. Switching `NewAddMemberHandler` to assert `interface{ MemberAddress() address; MemberRoles() []string }` instead of `*ActionAddMember`, which is exactly the mistake the [SECURITY block](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/daokit/actions.gno#L76-L93) warns against, leaves basedao, daokit, the identity suite and all three demos green.

<details><summary>test cases</summary>

```go
// gno/p/basedao/handler_contract_test.gno
package basedao

import (
	"testing"

	"gno.land/p/nt/urequire/v0"
)

type lookalikeAddMember struct {
	Address address
	Roles   []string
}

func (l *lookalikeAddMember) String() string { return "lookalike" }

// A registered handler must key on its own concrete payload type, so a foreign
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

## gno/r/daoidentity/initdao/initdao.gno:11-12 [↗](../../../../.worktrees/gnodaokit-review-65/gno/r/daoidentity/initdao/initdao.gno#L11-L12)
Missing test: the comment claims seeding a DAO with the deployer as a member and calling `InstantExecute` from `init` works, and the fixture does neither, so it pins `cur.IsCurrent()` as a VM fact rather than proving construction survives the gate. Writing the realm the comment describes gives `member id must not be empty` at package load, because `cur.Previous().Address()` is empty at init under the test harness. On chain `OriginCaller` is the deployer, which is exactly why the claim needs a fixture: it is unverifiable in the only environment CI has.

## gno/r/daoidentity/suite/suite_test.gno:120-122 [↗](../../../../.worktrees/gnodaokit-review-65/gno/r/daoidentity/suite/suite_test.gno#L120-L122)
Nit: the comment says swapping the gate and `CallerID` leaves every other test green. Swapping them in `Propose` alone also reddens `TestInstantExecuteIsSameRealmOnly`, which wants `realm mismatch: ...` and gets `caller id is empty`.

## gno/p/basedao/view_proposals_page.gno:29-30 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/view_proposals_page.gno#L29-L30)
Nit: this says only `Execute` writes `Status`. `1eabd78` on this branch re-exported `UpdateStatus`, which also does.

## gno/p/daocond/cond_role_treshold.gno:1 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daocond/cond_role_treshold.gno#L1)
Nit: `treshold` is misspelled in this filename and in the new `cond_role_treshold_test.gno`; every identifier inside is spelled correctly. File names are what `vm/qfile` and gnoweb list, so this freezes with the rest.

## Makefile:22 [↗](../../../../.worktrees/gnodaokit-review-65/Makefile#L22)
Nit: this globs `gno.mod`, of which the repo has zero; all 16 manifests are `gnomod.toml`. The `gno mod tidy` CI step and the no-diff check after it therefore assert nothing.

## gno/p/daokit/proposals.gno:130 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/daokit/proposals.gno#L130)
Suggestion: this freezes as a method whose own doc says nothing in the package may call it, that must not be called while rendering, and that is safe only while `Vote` and `Execute` both key on `Executed` alone. Its only caller in the tree is the probe at [`dao.gno:199`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/r/daoidentity/dao/dao.gno#L199), added to show calling it is harmless. Publishing makes that constraint on [`daokit.gno:98`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/daokit/daokit.gno#L98) and [`:118`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/daokit/daokit.gno#L118) permanent and enforced only by a comment.

## gno/p/realmid/realmid.gno:9 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/realmid/realmid.gno#L9)
Suggestion: nothing in the tree imports `realmid` since basedao stopped, and [`README.md:21-32`](https://github.com/samouraiworld/gnodaokit/blob/0eb8518/gno/p/realmid/README.md?plain=1#L21-L32) heads a block "Not for caller authentication". `Previous` and `Current` still freeze at a published path as bare stack walks; publishing only `IsPackage` and `IsUser` would leave nothing to reach for by mistake.

## gno/p/basedao/basedao.gno:308-310 [↗](../../../../.worktrees/gnodaokit-review-65/gno/p/basedao/basedao.gno#L308-L310)
Suggestion: only the length is checked, so a member who sends `YES` has it stored and counted in the ballot total while every tally reads zero: `Yes: 0/3`, `No: 0/3`, `Abstain: 0/3`. A realm's `Vote` entry point takes `daocond.Vote` straight off a `MsgCall` argument. Pre-existing, raised because `8c8ef04` typed the three constants so a misuse is a compile error, and a runtime value cannot be typed after publication.
