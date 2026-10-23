# Findings in posting order, from round assemble: 5 to post, 0 SKIP, 0 refuted kept out

## contribs/gpao/oracle.go:570 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L570) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: no test reaches the settled re-check after verification: removing it leaves the gpao package green
Check: Delete oracle.go:570-572 at 5454f8165 and run go test -count=1 . in contribs/gpao; same mutation of the lone isSettled check at the merge base (oracle.go:553).
Details: With `if o.recordIfSettled(ctx, path, key) { return }` deleted after verify, every test passes; the settled cases are all answered by the pre-check at oracle.go:513. The same mutation of the only settled check at the merge base also stays green, so the gap predates the diff, which now states the re-check as a decision in its ADR.
Evidence: head, lines 570-572 deleted: ok github.com/gnolang/gno/contribs/gpao 25.354s; base, `if false && o.isSettled(ctx, path) {` at oracle.go:553: ok 18.336s

## contribs/gpao/oracle.go:345 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L345) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: a tip one block below the last read height counts as a chain restart: the genesis cache is emptied and the error log reports a reset
Check: Judge test TestJudgeTipLagKeepsGenesisImports, tips {6, 7, 6, 8} from start height 1, expecting the seeded cache kept.
Details: After reading through 7, height is 8; a Status answering 6 gives `latest < height-1`, 6 < 7, so importsStale is set and errf reports the chain's tip below a height already read. The cost is a cold closure on the next verification and a false error line, nothing terminal.
Evidence: go test -run TestJudgeTipLag -v: one-block_lag_after_reading_7: "map[]" should have 1 item(s), but has 0
Artifact: tests/judge_2_tip_lag_test.go
State: CONFIRMED, band: Nit, angle: reach
TL;DR: a -start-height above the tip logs that unread heights were already read and empties the genesis cache
Check: Judge test TestJudgeTipLagKeepsGenesisImports, start height 1000 and tips {500}, expecting the seeded cache kept.
Details: With startHeight 1000 the first poll answering 500 gives 500 < 999: errf says height 999 was already read though no block was, and the cache is dropped. Same line and same edit as candidate 2.
Evidence: go test -run TestJudgeTipLag -v: start_height_above_the_tip: "map[]" should have 1 item(s), but has 0
Artifact: tests/judge_2_tip_lag_test.go
State: PLAUSIBLE, band: Suggestion, angle: lines
TL;DR: a relaunched chain that is already past the read height when polled keeps the old genesis cached
Check: Run two in-memory chains with different genesis behind one remote, the second passing the first's tip between polls, and assert genesisImports() is emptied; not run here.
Details: run() compares Status().LatestBlockHeight against height-1 and nothing else (oracle.go:345-350), so a tip sequence {6, 9} across a relaunch reads as a growing chain. Whether that serves a stale genesis package needs a relaunch with a changed genesis under the same remote, which no harness here builds.
Evidence: oracle.go:345 `if latest < height-1 && !reset {` is the only writer of importsStale
State: PLAUSIBLE, band: Suggestion, angle: reach
TL;DR: a restart is detected only by a poll that lands below the old tip, which enqueue's blocking send can skip
Check: resetRPC tips {6, 9} with a different genesis behind the second tip, asserting the cache is emptied; needs two chains, not run here.
Details: enqueue (oracle.go:478-492) blocks the block reader while the queue is full, so no Status poll runs; a new chain that passes the old tip in that window is never seen as a reset. Mechanism read off the lines; the stale verdict needs a relaunched chain to show.
Evidence: oracle.go:487 `case o.candidates <- c:` blocks inside the height loop; oracle.go:345 keys on tip direction alone

## contribs/gpao/verifyone.go:120 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/verifyone.go#L120) · Nit
State: CONFIRMED, band: Nit, angle: removed
TL;DR: the seed decode failure takes exit 1 where the function's other oracle-side failures take exitNoVerdict
Check: Same read as candidate 4: verifyone.go:120-123 against :147-165 and :324-335.
Details: Same mechanism as candidate 4, one edit closes both; newVerifier and prepare failures in execVerifyOne call exitNoVerdict (exit 3, unavailable), the seed failures do not.
Evidence: verifyone.go:120, :123 plain returns; :147 `exitNoVerdict(cio.Err(), err)`
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: a seed the child cannot read or decode exits 1, which verify files as a rejection of the candidate
Check: Read verifyone.go:116-125 and :258-262, tm2/pkg/commands/command.go:136, verifyone.go:324-335 and handleCandidate's rejection branch.
Details: execVerifyOne returns `read import seed` and `decode import seed` as plain errors (verifyone.go:120, :123); the command exits 1 (command.go:136); verify maps any non-negative exit other than 3 and 4 to errors.New(msg) (verifyone.go:335), and handleCandidate records statusRejected and marks the bytes seen. The parent writes a decodable seed today, so this is latent, reached by a binary replaced on disk between parent and child. The parent-side `encode import seed` (:261) lands in the same branch.
Evidence: verifyone.go:123 `return fmt.Errorf("decode import seed: %w", err)`; verifyone.go:335 `return errors.New(msg)`; exitNoVerdict used at :147, :154, :165 for the child's other non-verdict failures

## contribs/gpao/importcache.go:55 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/importcache.go#L55) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: a package a hard-forked genesis replays with BlockHeight metadata is stamped with that height, so the cache never keeps it
Check: Read gno.land/pkg/gnoland/app.go:903-905, gno.land/pkg/sdk/vm/keeper.go:1129 and stampGnomod's `gm.AddPkg.Height = int(height)`.
Details: app.go sets header.Height = metadata.BlockHeight for a genesis tx carrying it, keeper.go:1129 stamps ctx.BlockHeight() into [addpkg], and closure keeps only Height == 0. Such packages are as fixed as the height-0 set yet are refetched on every verification: no worse than the merge base, which cached nothing, but the saving does not reach them. Not run against a replayed genesis.
Evidence: app.go:903 `if metadata.BlockHeight > 0 { header.Height = metadata.BlockHeight }`; keeper.go:638 `gm.AddPkg.Height = int(height)`; importcache.go:55 `gm.AddPkg.Height == 0 && !gm.Private`

## contribs/gpao/oracle.go:513 [gh](https://github.com/gnolang/gno/blob/5454f8165cee97056ae64ef955c541e2c37d9af2/contribs/gpao/oracle.go#L513) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: the settled pre-check runs under the daemon's deadline-free context, so a stalled node holds each candidate up to the RPC client's 60 s before verification starts its own prepare budget
Check: Read oracle.go:513, getter.go:104, tm2/pkg/bft/rpc/client/client.go:54 and :371, main.go:99; base oracle.go:498 against :553.
Details: handleCandidate calls recordIfSettled(ctx, ...) at oracle.go:513 before o.verify; ctx is main's signal.NotifyContext with no deadline (main.go:99), and queryPkgMeta's ABCIQuery is bounded only by the client's requestTimeout, defaultTimeout = 60 s (client.go:54, applied at :371). At the merge base verify ran first (oracle.go:498) and the settled query ran only after a verdict (:553), so a stalled node cost each candidate the prepare budget alone. Magnitude read off the constants, not timed.
Evidence: client.go:19 `const defaultTimeout = 60 * time.Second`; client.go:371 `ctx, cancel := context.WithTimeout(ctx, timeout)`; oracle.go:513 `if o.recordIfSettled(ctx, path, key) {` precedes oracle.go:518 `err := o.verify(ctx, mpkg)`
