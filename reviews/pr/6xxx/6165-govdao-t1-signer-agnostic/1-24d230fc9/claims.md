# Claim gate: PR 6165 round 1 (24d230fc9), run by claude-opus-5

Every falsifiable claim in `overview.md`, `review_claude-opus-5_davd-gzl.md` and
`comment_claude-opus-5.md`, with the check that would have shown it false. Runs
are from the reviewed worktree at `24d230fc98ace3cfc5186d0165eb869263811260`,
Go `go1.25.9`, tree clean before and after (`git status --short` empty).

Shorthand in the Check column:

- `W` = the reviewed worktree, `blob <path>#La-Lb` = read those lines in `W`.
- `run <name>` = copy `tests/<name>.txtar` into `gno.land/pkg/integration/testdata/`,
  `go test ./gno.land/pkg/integration/ -run 'TestTestdata/<name>' -v`, remove it.
- `probe <name>` = same harness, fixture built for this gate (recipes below).

## Probes built for this gate

Each is one of the two shipped fixtures with one section swapped. `base prog` is
the embedded gno program of `git show bc35e978a:misc/govdao-scripts/extend-govdao-t1.sh`;
`head prog` is the same extraction at `24d230fc9`.

| Probe | Program | Loader seats | `AllowedDAOs` | Asserted |
| --- | --- | --- | --- | --- |
| `zzclaim_postfix` | head prog, literal replaced by `memberstore.NewMember(3)` | Manfred | empty | 6 `seat` lines, `skip Manfred`, `OK!` |
| `zzclaim_base_open` | base prog | Manfred | empty | `cannot allocate` |
| `zzclaim_base_locked` | base prog | test1 | `[impl]` | `this Realm is not allowed` |
| `zzclaim_base_aeddi` | base prog | Aeddi | empty | `cannot allocate`, and NOT `member already exists` |
| `zzclaim_test13` | head prog | test1 | `[impl, r/test13/rotate]` | `this Realm is not allowed` |
| `zzclaim_revert` | seat Jae with `NewMember`, `println`, then `panic("boom")`; second run reads the tier back | none | empty | `jae-tier=[]` after the panic |

All six pass. `gno lint` probes used `go build -o gnobin ./gnovm/cmd/gno`, with the
extracted program in a throwaway package under `W/examples/`, removed after.

## Claims

### `overview.md`

| # | Claim | Check | Observed | Result |
| --- | --- | --- | --- | --- |
| 1 | Script writes a gno program to a temp file and hands it to `gnokey maketx run` | `blob misc/govdao-scripts/extend-govdao-t1.sh#L20-L23,#L73-L80` | `cat >"$TMPDIR/extend_govdao.gno" <<'GOEOF'` … `gnokey maketx run … "$TMPDIR/extend_govdao.gno"` | PASS |
| 2 | `memberstore.Get` is at `memberstore.gno:182` | `blob examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L182` | `func Get(_ int, rlm realm) MembersByTier {` | PASS |
| 3 | The quoted `Get` body (IsCurrent, PkgPath, InAllowedDAOs, panic text) matches source | `sed -n '182,192p'` on that file | `if !rlm.IsCurrent()` / `currealm := rlm.PkgPath()` / `if !dao.InAllowedDAOs(currealm)` / `panic("this Realm is not allowed to get the Members data: " + currealm)` / `return members` | PASS |
| 4 | `InAllowedDAOs` is at `proxy.gno:231-240` | `blob examples/gno.land/r/gov/dao/proxy.gno#L231-L240` | `func InAllowedDAOs(pkg string) bool {` … `return false` | PASS |
| 5 | The allowlist starts empty | `grep -n "var allowedDAOs" examples/gno.land/r/gov/dao/proxy.gno` | `20:var allowedDAOs []string` | PASS |
| 6 | An empty list means yes to everyone | `sed -n '232,234p' proxy.gno` | `if len(allowedDAOs) == 0 { return true // corner case for initialization }` | PASS |
| 7 | The run path is built at `keeper.go:1398` from the signer's address | `blob gno.land/pkg/sdk/vm/keeper.go#L1398` | `memPkg.Path = chainDomain + "/e/" + msg.Caller.String() + "/run"` | PASS |
| 8 | A run signed by `g1jg8m…` reaches `Get` as `gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run` | `run govdao_t1_roster_locked` | `panic: this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run` | PASS |
| 9 | gnoland1 closes the window at `govdao_prop1.gno:118-120` with `[impl]` | `blob misc/deployments/gnoland1/govdao_prop1.gno#L118-L120` | `dao.UpdateImpl(cross(cur), dao.UpdateRequest{ AllowedDAOs: []string{"gno.land/r/gov/dao/v3/impl"}, })` | PASS |
| 10 | test13 closes at `:109` with `[impl, r/test13/rotate]` | `blob …/govdao_prop1_test13.gno#L109` | `dao.NewUpdateRequest(impl.NewGovDAO(), []string{"gno.land/r/gov/dao/v3/impl", "gno.land/r/test13/rotate"})` | PASS |
| 11 | pearl, sapphire and topaz each close at `:49` with `[impl]` | `blob …/govdao_prop1_{pearl,sapphire,topaz}.gno#L49` | identical line in all three: `dao.NewUpdateRequest(impl.NewGovDAO(), []string{"gno.land/r/gov/dao/v3/impl"})` | PASS |
| 12 | Every one of the five is the last statement of its bootstrap | `grep -n UpdateImpl misc/deployments/*/govdao_prop1*.gno misc/deployments/*/transactions/base/bootstrap/govdao_prop1_*.gno`, then read the next line | five hits, each followed by `}` | PASS |
| 13 | A MsgRun is refused on the `[impl]` shape | `run govdao_t1_roster_locked` | `this Realm is not allowed to get the Members data: …/run`, `Get at memberstore.gno:188` | PASS |
| 14 | A MsgRun is refused on the test13 two-entry shape | `probe zzclaim_test13` | `Data: this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu…/run` | PASS |
| 15 | `NewAddMemberRequest` is at `prop_requests.gno:80` | `blob examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L80` | `func NewAddMemberRequest(cur realm, addr address, tier string, portfolio string) dao.ProposalRequest {` | PASS |
| 16 | The composite literal aborts with `cannot allocate gno.land/r/gov/dao/v3/memberstore.Member in realm gno.land/e/<signer>/run` | `run govdao_t1_roster_open` | `Data: cannot allocate gno.land/r/gov/dao/v3/memberstore.Member in realm gno.land/e/g1jg8mtutu…/run` | PASS |
| 17 | pearl:44 and test13:97 already call `NewMember` | `blob …govdao_prop1_pearl.gno#L44`, `…_test13.gno#L97` | `must(ms.SetMember(memberstore.T1, govdaoT1Addr, memberstore.NewMember(3)))` / `…address("g1manfred47…"), memberstore.NewMember(3)))` | PASS |
| 18 | Neither rule is visible to a static check: `gno lint` type-checks the file and says nothing | `gnobin lint .` on the extracted head program | no output, rc 0 | PASS |
| 19 | `SetMember` at `types.gno:75-78` refuses an address in any tier with `ErrMemberAlreadyExists` rather than moving it | `blob …/memberstore/types.gno#L75-L78` | `_, t := mbt.GetMember(addr)` / `if t != "" { return &ErrMemberAlreadyExists{Tier: t} }` | PASS |
| 20 | `GetMember` is at `types.gno:94` | `blob …/types.gno#L94` | `func (mbt MembersByTier) GetMember(addr address) (m *Member, t string)` | PASS |
| 21 | "a signer authorised to make this call is by definition already seated" | `probe zzclaim_postfix`: signer `test1` is not seated by the loader (only Manfred is) | run succeeds, prints six `seat` lines and `OK!` | **FAIL** |

### `review_claude-opus-5_davd-gzl.md`

| # | Claim | Check | Observed | Result |
| --- | --- | --- | --- | --- |
| 22 | Author moul, base master, 1 file, +36 -13 | `gh pr view 6165 -R gnolang/gno --json author,baseRefName,changedFiles,additions,deletions` | `"additions":36,"deletions":13,"changedFiles":1,"author":{"login":"moul"},"baseRefName":"master"` | PASS |
| 23 | Reviewer is `davd-gzl` | `gh api user --jq '.login'` | `davd-gzl` | PASS |
| 24 | Head `24d230fc9`, merge base `bc35e978a` | `git -C W log --oneline -2` | `24d230fc9 chore: make the govDAO T1 extend script signer-agnostic` / `bc35e978a chore: replace ray with aeddi…` | PASS |
| 25 | "The list was written as 'the six members who are not moul', so only moul could sign it" | `git show bc35e978a:…extend-govdao-t1.sh`; then `probe zzclaim_postfix` for the signer half | base roster is 6 entries, `g1manfred47…` absent; but no code compares the signer to the roster, and non-member `test1` signs a working run | **FAIL** (second half) |
| 26 | The change turns six calls into a seven-entry table plus a loop | `git diff bc35e978a 24d230fc9 -- misc/govdao-scripts/extend-govdao-t1.sh` | base: 6 `must(ms.SetMember(…))`; head: `var t1Roster = []rosterEntry{…}` with 7 entries and `for _, r := range t1Roster` | PASS |
| 27 | It reads each address's tier with `GetMember` and skips the seated ones | `blob misc/govdao-scripts/extend-govdao-t1.sh#L55-L58` | `if _, tier := ms.GetMember(r.addr); tier != "" { println("skip " + …); continue }` | PASS |
| 28 | "The loop also makes a rerun idempotent" | `run govdao_t1_roster_open` (a first run at the head) | the first run already aborts: `cannot allocate …` — no run at the head is idempotent, only the post-fix form | **FAIL** |
| 29 | "the old `must()` turned the first already-seated address into a panic that aborted the whole transaction" | `probe zzclaim_base_aeddi`: base program, Aeddi seated as sole T1, open window | `cannot allocate … Member in realm …/run`, `main at …/extend_govdao.gno:15` (the `// Jae` line); the `! stderr 'member already exists'` assertion passes | **FAIL** |
| 30 | Verdict count: 2 warnings, 1 nit | `awk '/^## Warnings/,/^## Nits/' … \| grep -c '^- \*\*\['` and the same for Nits | `2` and `1` | PASS |
| 31 | Verify-first 1: every network's bootstrap passes a non-empty `AllowedDAOs` | the `grep -n UpdateImpl …` the bullet names | 5 hits, every one with a non-empty `[]string{…}` | PASS |
| 32 | `Get` gates on the calling realm's pkgpath, `memberstore.gno:182-189` | `blob …/memberstore.gno#L182-L189` | range spans `func Get` through the `panic(…)` and its closing brace | PASS |
| 33 | "Both fire before a single member is seated" | `probe zzclaim_base_aeddi` (allocation check beats the already-exists check) + `probe zzclaim_revert` (whole tx reverts) | abort at the first roster line; `jae-tier=[]` read back after a mid-run panic | PASS |
| 34 | "both predate this branch: the same two aborts reproduce on the merge base bc35e978a" | `probe zzclaim_base_open` and `probe zzclaim_base_locked`, both with the base revision's own script body | open: `cannot allocate …`, `main at …:15`; locked: `this Realm is not allowed…`, `Get at memberstore.gno:188`, `main at …:14` | PASS |
| 35 | `gno lint` is clean on the extracted program and its type check is live, the `ms.GetMemberXXX` typo giving `ms.GetMemberXXX undefined (type memberstore.MembersByTier has no field or method GetMemberXXX) (code=gnoTypeCheckError)` | `gnobin lint .` clean, then `sed -i 's/ms\.GetMember(/ms.GetMemberXXX(/'` and re-run | clean run: no output. Typo run: `extend_govdao.gno:32:20: ms.GetMemberXXX undefined (type memberstore.MembersByTier has no field or method GetMemberXXX) (code=gnoTypeCheckError)` | PASS |
| 36 | With `NewMember(3)`, a run in the bootstrap window seats six entries and prints `skip Manfred -- already T1` for the seventh | `probe zzclaim_postfix` | `seat Jae as T1` … `seat Milos as T1` (6), `skip Manfred -- already T1`, `OK!` | PASS |
| 37 | Six `must(ms.SetMember(…))` become the table and loop; `SetMember` now panics inline through a deleted `must` | `git show bc35e978a:…` vs the head file | base has `func must(err error)` at L27-31 and 6 call sites; head has no `must` and `if err := …; err != nil { panic(err.Error()) }` at L59-61 | PASS |
| 38 | Warning 1: `Get` refuses a `maketx run` on all five networks the header names | script header + `probe zzclaim_test13` + `run govdao_t1_roster_locked` (the `[impl]` shape all four others use) | header names moul on gnoland1/test-13 and aeddi on pearl/sapphire/topaz; both allowlist shapes refuse | PASS |
| 39 | Warning 1: "the signer's own T1 membership is never read" | `sed -n '182,192p' memberstore.gno`; `probe zzclaim_postfix` with a non-member signer | `Get` reads only `rlm.IsCurrent()` and `rlm.PkgPath()`; non-member `test1` completes a successful run | PASS |
| 40 | Warning 1: the locked fixture yields `this Realm is not allowed to get the Members data: gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run` | `run govdao_t1_roster_locked` | that string verbatim, from `Get at gno.land/r/gov/dao/v3/memberstore/memberstore.gno:188` | PASS |
| 41 | Warning 1: the description's `ErrMemberAlreadyExists` on the `// Aeddi` line names a line the transaction never reaches | `gh pr view 6165 --json body` for the claim; `probe zzclaim_base_aeddi` for the behaviour | description does say it; the run aborts on the `// Jae` line with `cannot allocate` and never returns `member already exists` | PASS |
| 42 | Warning 1 Fix: `NewAddMemberRequest` is a path `AllowedDAOs` still admits | `blob prop_requests.gno#L80-L120` | it builds a `dao.ProposalRequest` whose executor runs inside `r/gov/dao/v3/impl`, the sole allowed realm | PASS |
| 43 | Warning 2: the abort happens before the first member is seated | `probe zzclaim_base_aeddi` + `alloc.go:448-478` (`checkConstructionTime` fires at composite literals, evaluated as the `SetMember` argument) + `probe zzclaim_revert` | abort at roster entry 1 even with entry 3 pre-seated; a mid-run panic leaves `jae-tier=[]` | PASS |
| 44 | Warning 2: the VM raises `cannot allocate gno.land/r/gov/dao/v3/memberstore.Member in realm gno.land/e/<signer>/run` and the whole MsgRun reverts | `run govdao_t1_roster_open`, `probe zzclaim_revert` | that message verbatim; `jae-tier=[]` after the revert probe | PASS |
| 45 | Warning 2: `NewMember` at `types.gno:21`; both forms type-check | `blob …/types.gno#L21`; `gnobin lint .` on both variants | `func NewMember(invitationPoints int) *Member {`; both lint clean | PASS |
| 46 | Warning 2: the open fixture's commented assertions are the measured post-fix output | `probe zzclaim_postfix` asserting each commented line | all pass: `seat Jae as T1`, `skip Manfred -- already T1`, `OK!` | PASS |
| 47 | Nit: `README.md:17` still reads `add 6 T1 members to govDAO (one-time bootstrap)` | `blob misc/govdao-scripts/README.md?plain=1#L17` | `./govdao extend-govdao-t1                    # add 6 T1 members to govDAO (one-time bootstrap)` | PASS |
| 48 | Verified: the locked fixture locks `AllowedDAOs` to `gno.land/r/gov/dao/v3/impl` exactly as the pearl bootstrap does | diff the fixture's `dao.UpdateImpl` line against `govdao_prop1_pearl.gno:49` | allowlist value identical; the DAO argument differs, `impl.GetInstance(0, cur)` vs `impl.NewGovDAO()`, which does not change the allowlist | PASS |
| 49 | Verified: the base run stops at `cannot allocate` on its first `SetMember` line | `probe zzclaim_base_open` and `probe zzclaim_base_aeddi` | `main at gno.land/e/g1jg8mtutu…/run/extend_govdao.gno:15`, which is the first `must(ms.SetMember(… // Jae))` of the base program | PASS |
| 50 | Verified: six seats, one skip, `STORAGE DELTA: 7085 bytes` | `probe zzclaim_postfix` | `STORAGE DELTA:  7085 bytes` on the receipt, beside the six `seat` lines and one `skip` | PASS |
| 51 | Verified: the untyped string constants assign to the `address` field with no conversion | `gnobin lint .` on the head program, whose roster literal uses bare strings | clean, so `{"Jae", "g1ecsuj0…"}` assigns to `addr address` untouched | PASS |
| 52 | Verified: no CI job compiles or executes the script or the gno source it embeds | `grep -rn "extend-govdao-t1" .` and `grep -rn shellcheck .github/workflows/` | only two hits, the script's own usage line and `README.md:17`; no shellcheck job; no workflow references `misc/govdao-scripts` | PASS |
| 53 | Verified: `ci-dir-misc.yml` runs `go test` over a fixed list of Go programs under `misc/` | read the file and `_ci-go.yml`; cross-check the head's check runs | matrix is 6 programs, `uses: ./.github/workflows/_ci-go.yml` with `modulepath: misc/${{ matrix.program }}`, and `_ci-go.yml:131` runs `go test`; the head shows exactly `audit-pattern-harness, autocounterd, genproto, genstd, goscan, loop` × build/lint/test | PASS |
| 54 | The `#L26-L40` anchor proves that clause | `blob .github/workflows/ci-dir-misc.yml?plain=1#L26-L40` | range holds the "Fixed list" comment and only 3 of the 6 entries; the list ends at L43 and `go test` is reached via L45-L48, both outside the range | **FAIL** |
| 55 | Verified: the head is green on every check | `gh api --paginate repos/gnolang/gno/commits/24d230fc9…/check-runs` and `/status` | 22 `success`, 1 `skipped` (`save-pr-number`), no failure; combined status `success` | PASS |
| 56 | Verified: a mid-loop panic reverts the whole transaction, so no partial seating survives | `probe zzclaim_revert` | run 1 seats Jae then panics `boom-after-first-seat`; run 2 reads `jae-tier=[]` | PASS |
| 57 | Verified: `t1Roster` is a package-level var written once and read-only afterwards | `blob misc/govdao-scripts/extend-govdao-t1.sh#L39-L47,#L51` | declared once at package level, only ranged over in `main` | PASS |
| 58 | Open question: the script validates no address, and only `RemoveMember` clears a bad entry | `sed -n '74,92p' types.gno`; `grep -n "func (mbt MembersByTier)" types.gno` | no `IsValid` in `SetMember` (unlike `NewAddMemberRequest`, which panics on `!addr.IsValid()`); `RemoveMember` at L121 is the only remover | PASS |
| 59 | Open question: the same risk sits in every sibling script under `misc/govdao-scripts/` | `ls misc/govdao-scripts/*.sh` | 13 scripts, all taking raw addresses from the caller or a literal | PASS |

### `comment_claude-opus-5.md`

| # | Claim | Check | Observed | Result |
| --- | --- | --- | --- | --- |
| 60 | Body anchor `README.md?plain=1#L17` carries the quoted text | `blob misc/govdao-scripts/README.md?plain=1#L17` | the quoted line verbatim | PASS |
| 61 | Anchor `extend-govdao-t1.sh:50` is the `memberstore.Get` call | `sed -n '50p'` on the head file | `	ms := memberstore.Get(0, cur)` | PASS |
| 62 | Anchor `extend-govdao-t1.sh:59` is the composite-literal seat line | `sed -n '59p'` | `		if err := ms.SetMember(memberstore.T1, r.addr, &memberstore.Member{InvitationPoints: 3}); err != nil {` | PASS |
| 63 | `memberstore.gno#L186-L188` is the pkgpath read and the refusal | `blob …/memberstore.gno#L186-L188` | `currealm := rlm.PkgPath()` / `if !dao.InAllowedDAOs(currealm) {` / `panic("this Realm is not allowed…")` | PASS |
| 64 | `types.gno#L17-L19` is the type `memberstore` owns | `blob …/types.gno#L17-L19` | `type Member struct { InvitationPoints int }` | PASS |
| 65 | The `suggestion` block is line 59 with only the literal replaced | `sed -n '59p' script \| cat -A` vs `sed -n '124p' comment \| cat -A` | both `^I^Iif err := ms.SetMember(memberstore.T1, r.addr, …); err != nil {$`; only the third argument differs | PASS |
| 66 | The suggested form compiles and runs | `probe zzclaim_postfix`, which is the head program with exactly that substitution | seats six, skips Manfred, `OK!` | PASS |
| 67 | Both repro txtars are the shipped fixtures | `diff` of each extracted heredoc against `tests/*.txtar` with the header stripped | identical modulo blank lines, both files | PASS |
| 68 | The pasted locked output matches a verbatim run | `run govdao_t1_roster_locked` | every pasted line reproduces: `Data: this Realm is not allowed…/run`, `panic: …`, `Get at …/memberstore.gno:188`, `main at …/extend_govdao.gno:27`. Only the elapsed time differs (1.75s here vs the pasted 2.40s) | PASS |
| 69 | The pasted open output matches a verbatim run | `run govdao_t1_roster_open` | `Data: cannot allocate …Member in realm …/run` and `main at …/extend_govdao.gno:0` reproduce; elapsed 1.73s vs the pasted 2.41s | PASS |
| 70 | `main at …/extend_govdao.gno:27` is the `Get` call in the fixture's own program | `sed -n '27p'` on the fixture's `run/extend_govdao.gno` section | `	ms := memberstore.Get(0, cur)` | PASS |
| 71 | The five-row network table's line numbers | same reads as claims 9-11 | all five land on their `UpdateImpl` line | PASS |

### `tests/`

| # | Claim | Check | Observed | Result |
| --- | --- | --- | --- | --- |
| 72 | Both fixtures embed the head script's program verbatim | extract each `-- run/extend_govdao.gno --` section and diff against the heredoc of `24d230fc9:misc/govdao-scripts/extend-govdao-t1.sh` | identical, both fixtures | PASS |
| 73 | `govdao_t1_roster_locked` header: "Passes at 24d230fc9" | `run govdao_t1_roster_locked` | `--- PASS: TestTestdata/govdao_t1_roster_locked (1.75s)` | PASS |
| 74 | `govdao_t1_roster_open` header: "Passes at 24d230fc9 and at the merge base bc35e978a: both abort" | `run govdao_t1_roster_open` and `probe zzclaim_base_open` | `--- PASS … (1.73s)`; the base program aborts with the same `cannot allocate` message | PASS |
| 75 | The open fixture's commented post-fix assertions hold | `probe zzclaim_postfix` asserting all four | all pass | PASS |

## Failures

**21, `overview.md`, last line.** "since a signer authorised to make this call
is by definition already seated" is not true of the code: `Get` reads only
`rlm.IsCurrent()` and `rlm.PkgPath()`, never the signer's membership, and
`zzclaim_postfix` completes a successful seven-entry run signed by `test1`, whom
the loader never seats. It also contradicts this file's own earlier line, "the
signer's own membership does not enter into it". Say instead that the skip drops
the signer's entry whenever the signer happens to be seated, which is how the
script is meant to be run.

**25, review Overview, "so only moul could sign it".** The first half is right:
the base roster is the six T1 addresses other than `g1manfred47…`. The second
half is a behavioural claim and nothing enforces it: no code compares the signer
to the roster, and at the base every signer aborts at `cannot allocate` before
any address is compared. Say the list assumed moul as the signer.

**28, review Overview, "The loop also makes a rerun idempotent".** At the
reviewed head no run is idempotent, because the first one already aborts at
`cannot allocate`. The idempotence is a property of the post-fix form only. Say
it would make reruns idempotent once the allocation is fixed.

**29, review Overview, "the old `must()` turned the first already-seated
address into a panic that aborted the whole transaction".** `zzclaim_base_aeddi`
puts the base program in exactly the pearl/sapphire/topaz shape, aeddi seated as
sole T1 and the window open, and the run aborts with `cannot allocate` at
`extend_govdao.gno:15`, the `// Jae` line: `SetMember` never executes, so `must`
never sees an `ErrMemberAlreadyExists`. This is the same mistake Warning 1
attributes to the PR description, restated in the review's own Overview. Say the
base aborted for both signers at the first roster line, on the allocation, and
that the already-exists path was never reached.

**54, review Verified, the `ci-dir-misc.yml` anchor.** The clause is true, but
`#L26-L40` proves only the "fixed list" comment and three of the six entries.
The list runs to L43 and `go test` is reached through `uses:` and `modulepath:`
at L45-L48. Widen the anchor to `#L26-L48`.

## Rows written after the gate, run by the parent

These are the sentences the five failures above were repaired with, plus the
Nit and the fixture the repair rests on.

| # | Claim | Check | Observed | Result |
| --- | --- | --- | --- | --- |
| 76 | Overview: the base list held the six T1 members other than moul | `git show bc35e978a:…extend-govdao-t1.sh` | six `SetMember` lines, `g1manfred47…` absent | PASS |
| 77 | Overview: no line in the base program read the signer | grep its heredoc for `OriginCaller`, `PreviousRealm`, `Address()`, `runtime`, `IsUser` | no match; the head program's only hit is the word `runtime` inside the roster comment | PASS |
| 78 | Overview: the entry a seated signer occupies drops out | the post-fix form, run with Manfred seated by the loader | `skip Manfred -- already T1`, six seats | PASS |
| 79 | Verified: the base aborts at `extend_govdao.gno:15`, the `// Jae` line | `run govdao_t1_roster_base_aeddi`, the base program with aeddi sole T1 and the window open | `main at …/extend_govdao.gno:15`; the `! stderr 'member already exists'` assertion passes | PASS |
| 80 | Nit: nothing reads the signer's tier | read `Get` at `memberstore.gno:182-189` and the roster loop | `Get` reads `rlm.IsCurrent()` and `rlm.PkgPath()` only; the loop reads roster addresses | PASS |
| 81 | Verified: the widened `ci-dir-misc.yml#L26-L48` anchor carries the clause | `sed -n '26,48p'` on the workflow | the whole six-entry `program:` list, then `uses: ./.github/workflows/_ci-go.yml` with `modulepath: misc/${{ matrix.program }}` | PASS |
