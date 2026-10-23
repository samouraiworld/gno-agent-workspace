# Claims: gnolang/gno#6401 https://github.com/gnolang/gno/pull/6401 round 1, 5454f8165, claude-opus-5-5, solo review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Missing test | contribs/gpao/oracle.go:570 | Delete oracle.go:570-572 at 5454f8165 and run go test -count=1 . in contribs/gpao; same mutation of the lone isSettled check at the merge base (oracle.go:553). | head, lines 570-572 deleted: ok github.com/gnolang/gno/contribs/gpao 25.354s; base, `if false && o.isSettled(ctx, path) {` at oracle.go:553: ok 18.336s |  | hot |  |
| 2 | CONFIRMED | Nit | contribs/gpao/oracle.go:345 | Judge test TestJudgeTipLagKeepsGenesisImports, tips {6, 7, 6, 8} from start height 1, expecting the seeded cache kept. | go test -run TestJudgeTipLag -v: one-block_lag_after_reading_7: "map[]" should have 1 item(s), but has 0 | tests/judge_2_tip_lag_test.go | hot |  |
| 3 | PLAUSIBLE | Suggestion | contribs/gpao/oracle.go:345 | Run two in-memory chains with different genesis behind one remote, the second passing the first's tip between polls, and assert genesisImports() is emptied; not run here. | oracle.go:345 `if latest < height-1 && !reset {` is the only writer of importsStale |  | hot |  |
| 4 | CONFIRMED | Suggestion | contribs/gpao/verifyone.go:120 | Read verifyone.go:116-125 and :258-262, tm2/pkg/commands/command.go:136, verifyone.go:324-335 and handleCandidate's rejection branch. | verifyone.go:123 `return fmt.Errorf("decode import seed: %w", err)`; verifyone.go:335 `return errors.New(msg)`; exitNoVerdict used at :147, :154, :165 for the child's other non-verdict failures |  | hot |  |
| 1001 | CONFIRMED | Suggestion | contribs/gpao/oracle.go:513 | Read oracle.go:513, getter.go:104, tm2/pkg/bft/rpc/client/client.go:54 and :371, main.go:99; base oracle.go:498 against :553. | client.go:19 `const defaultTimeout = 60 * time.Second`; client.go:371 `ctx, cancel := context.WithTimeout(ctx, timeout)`; oracle.go:513 `if o.recordIfSettled(ctx, path, key) {` precedes oracle.go:518 `err := o.verify(ctx, mpkg)` |  | hot |  |
| 1002 | CONFIRMED | Nit | contribs/gpao/oracle.go:345 | Judge test TestJudgeTipLagKeepsGenesisImports, start height 1000 and tips {500}, expecting the seeded cache kept. | go test -run TestJudgeTipLag -v: start_height_above_the_tip: "map[]" should have 1 item(s), but has 0 | tests/judge_2_tip_lag_test.go | hot |  |
| 1003 | PLAUSIBLE | Suggestion | contribs/gpao/oracle.go:345 | resetRPC tips {6, 9} with a different genesis behind the second tip, asserting the cache is emptied; needs two chains, not run here. | oracle.go:487 `case o.candidates <- c:` blocks inside the height loop; oracle.go:345 keys on tip direction alone |  | hot |  |
| 1004 | CONFIRMED | Suggestion | contribs/gpao/importcache.go:55 | Read gno.land/pkg/gnoland/app.go:903-905, gno.land/pkg/sdk/vm/keeper.go:1129 and stampGnomod's `gm.AddPkg.Height = int(height)`. | app.go:903 `if metadata.BlockHeight > 0 { header.Height = metadata.BlockHeight }`; keeper.go:638 `gm.AddPkg.Height = int(height)`; importcache.go:55 `gm.AddPkg.Height == 0 && !gm.Private` |  | hot |  |
| 1005 | CONFIRMED | Nit | contribs/gpao/verifyone.go:120 | Same read as candidate 4: verifyone.go:120-123 against :147-165 and :324-335. | verifyone.go:120, :123 plain returns; :147 `exitNoVerdict(cio.Err(), err)` |  | hot |  |

Hit rate per tier, from the rows above: hot 7/9 confirmed over 6 files, warm 0/0 confirmed over 1 files, cold 0/0 confirmed over 8 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | contribs/gpao/oracle.go:513 | The pre-verification isSettled runs outside the prepare clock, so a stalled node holds each candidate before verification starts | oracle.go:447-455 runVerifier passes the daemon ctx; tm2/pkg/bft/rpc/client/client.go:19 `const defaultTimeout = 60 * time.Second` bounds each query. A candidate handled during a stall ends pending (errVerifyUnavailable) either way and is never re-offered, so a slower stall consumes no more candidates. The ctx makes shutdown prompt, which is what TestSettledCheckEndsWithItsContext pins. |
| catalog | contribs/gpao/importcache.go:38 | The parent now parses untrusted import blocks, and packages.FileImports panics when strconv.Unquote rejects a literal the parser accepted | A go run probe of parser.ParseFile(..., parser.ImportsOnly) gave `"\q"`, an unterminated `"abc`, `"\xff"` and `"a" "b"`: all 'parse err', so FileImports returns before the panic. A raw literal holding \r gave 'lit "`ab`" unquote err <nil>'. |
| claims | contribs/gpao/importcache.go:56 | A submitter writes `[addpkg] height = 0` so that a post-genesis package passes as genesis and grows the cache | gno.land/pkg/sdk/vm/keeper.go:637-639 stampGnomod assigns `gm.AddPkg.Height = int(height)` unconditionally, and both AddPackage paths call it with ctx.BlockHeight() (keeper.go:1014, :1129). |
| catalog | contribs/gpao/verifyone.go:124 | A seeded MemPackage loses its Type in the amino JSON round trip, so the child cannot build it | TestOnlyGenesisImportsOutliveAVerification spawns the child with a seeded genesis import and requires NoError, and .github/workflows/ci-dir-contribs.yml:40 builds the matrix from every contribs/*/ module, gpao included, so CI catches it. |
| claims | contribs/gpao/README.md:46 | Packages from a hard-fork genesis that replays history carry their historical [addpkg] height, so they are never cached | gno.land/pkg/gnoland/app.go:903-905 `if metadata.BlockHeight > 0 { header.Height = metadata.BlockHeight }`: those packages are refetched every verification. That costs queries only, and the fix is the README/ADR wording, so it is not a candidate. |
| removed | contribs/gpao/oracle.go:761 | isSettled lost its `res == nil` guard when it moved to queryPkgMeta, so a nil result with a nil error would panic the daemon. | sed -n 360,400p tm2/pkg/bft/rpc/client/client.go: unmarshalResponseBytes ends `return &result, nil`, so ABCIQuery never returns a nil result without an error. |
| reach | contribs/gpao/importcache.go:55 | A submitter could ship a gno.mod without [addpkg] beside the stamped gnomod.toml so the parse reads height 0 and the cache keeps a post-genesis package without bound. | gnovm/pkg/gnomod/parse.go:129 `for _, fname := range []string{"gnomod.toml", "gno.mod"}` reads gnomod.toml first, and keeper.go:1129 stampGnomod always writes gnomod.toml. |
| reach | contribs/gpao/importcache.go:43 | A body query failing mid-package could leave a partial genesis package in the cache for the life of the run. | sed -n 186,227p contribs/gpao/getter.go: fetch returns `nil, err` on any body query error, so closure sees nil and records an absence, which importCache never keeps. |
| removed | contribs/gpao/verifier.go:347 | The walkImports extraction could change which paths the child builds, descends into or records as unserved. | diff.md lines 2074-2101 against removed lines 2131-2162: stdlib builds and does not descend, a nil userlib dep is recorded unserved with cur.Path, a buildOne error ends the walk, a fetched dep is queued; seen is set before visit in both. |
| reach | contribs/gpao/verifyone.go:259 | Seeded dependencies carry Type gno.MPUserAll, which the candidate never did, and could fail the amino round trip. | gnovm/pkg/gnolang/package.go:135 registers `MemPackageType("")` with amino, and `gh pr checks 6401` shows `gpao / test pass`, which includes TestOnlyGenesisImportsOutliveAVerification verifying through a seeded child. |

## Completeness

- Angles: the five finders' angles returned candidates on oracle.go, verifyone.go and importcache.go. `getter.go` and `verifier.go`, both hot, returned none; the judge read `rpcGetter.seed` (getter.go:178-185) and its call at verifyone.go:149 and found the seeded `Absent` paths still go through the child's miss classification, so no candidate was added.
- Catalog: `contribs/gpao` is Go tooling, not GnoVM, stdlib or `.gno` code, so the invariant catalog does not apply per the project's review rule.
- Changed tests: the sibling of the edited settled tests that would pin the re-check after verification does not exist, which is candidate 1; the same mutation at the merge base (oracle.go:553) also stays green, so the gap predates the diff.
- Reflector file: `candidates/reflector.json`, no drops and no new candidates.

## Retro

- What failed: nothing died, and no agent left the session's model, per `./scripts/review-retro.py --model claude-opus-5-5` exiting 0. The draft carried `Event: APPROVE` under `post as an AI` and a round note counting two PLAUSIBLE Suggestions where one finding was raised twice; the parent set the event to COMMENT and corrected the note.
- What worked: both finders reached the reset detector independently, rows 2 and 1002 as one Nit and rows 3 and 1003 as one held Suggestion; the judge ran the Missing test's mutation at both head and base.
- Hit rate per tier: hot 7 of 9 confirmed over 6 files, the 2 PLAUSIBLE rows hot too; no warm or cold file carried a candidate.
- Upgrade: the same `--brief` placement as round 1 of 6400, named to the user rather than applied.
- Cost: 3 agents, 96 turns, 186k output, 13.9M cache read, about $10.43.
