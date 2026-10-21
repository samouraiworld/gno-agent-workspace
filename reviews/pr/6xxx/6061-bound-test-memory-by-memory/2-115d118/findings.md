# Findings in posting order, from round assemble: 8 to post, 6 SKIP, 0 refuted kept out

## gno.land/pkg/integration/testscript_gnoland.go:216 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L216) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: In a cgroup at its memory.max the node budget never shrinks and re-reads memory on every poll, so test nodes keep being admitted into a full cgroup.
Check: cp tests/b1-lines-removed-reach-catalog-budget-at-cgroup-limit_test.go gno.land/pkg/integration/zz_budget_at_cgroup_limit_test.go && go test ./gno.land/pkg/integration -run TestReproBudgetAtCgroupLimit -v -count=1: the 0-byte row keeps limit 6 and reads 20 times in 20 polls, the 1-byte row cuts to 4 with 0 reads
Details: clampToCgroup (tm2/pkg/testutils/parallel_linux.go:71-79) sets free to 0 when used >= limit, so Available becomes 0, and ReadMemInfo then returns `mi, mi.Total > 0 && mi.Available > 0` (line 62), which is ok=false. resize (testscript_gnoland.go:215-217) does `if !ok { return }` before the shrink case and before `b.lastCheck = now`. A suite running in a container or systemd slice whose budget ramped while memory was free therefore keeps its ramped limit once the cgroup fills, handing every released slot to a waiter, and every blocked script's 200ms poll re-reads /proc/meminfo and the cgroup files. The finder's mock copies line 62's expression verbatim; the read of clampToCgroup confirms the real reader produces exactly that 0. Fix: treat a cgroup at its limit as a valid reading of Available 0 (return ok on Total > 0 alone, or report the clamp separately), so resize shrinks, and advance lastCheck before the read so a failed read is paced too.
Evidence: Head 115d118 in a scratch worktree, go1.25.9, go test ./gno.land/pkg/integration -run TestReproBudgetAtCgroupLimit -v -count=1: 'available=1 bytes: limit=4 (started 6, min 2), reads during 20 polls inside the interval=0' / 'available=0 bytes: limit=6 (started 6, min 2), reads during 20 polls inside the interval=20', PASS. Merge base 8580ae38: grep for nodeBudget and ReadMemInfo under gno.land/pkg/integration and tm2/pkg/testutils returns nothing, because the base uses a static parallel cap with no memory reading.
Artifact: reviews/pr/6xxx/6061-bound-test-memory-by-memory/2-115d118/tests/b1-lines-removed-reach-catalog-budget-at-cgroup-limit_test.go

## SKIP gno.land/pkg/integration/testscript_gnoland.go:246 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L246) · Missing test
State: PLAUSIBLE, band: Missing test, angle: lines, on the finder's read only
TL;DR: No test pins the running > limit guard on recording a cut: replacing it with `if true` leaves every NodeBudget test green
Check: sed -i 's/^\t\tif b.running > b.limit {$/\t\tif true {/' gno.land/pkg/integration/testscript_gnoland.go && go test ./gno.land/pkg/integration -run NodeBudget -count=1: expect a FAIL, observe ok
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gno.land/pkg/integration/testscript_gnoland.go:313 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L313) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: acquireSlot's held flag and its deferred release have no test; gnoland.txtar starts twice in one script and only the flag keeps that to one slot
Check: grep -rn acquireSlot gno.land/pkg/integration/*_test.go returns nothing; mutate the held check and run the NodeBudget tests plus gnoland.txtar under GNO_TEST_MAX_PARALLEL=1
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gnovm/cmd/gno/test.go:261 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/cmd/gno/test.go#L261) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: the new -p clamp at GOMAXPROCS has no test: no Go test or txtar passes -p, so deleting the clamp, or only its `> GOMAXPROCS` arm, stays green
Check: grep -rnE '(^|\s)-p[ =]?[0-9]' gnovm/cmd/gno/testdata gnovm/cmd/gno/*_test.go (expect no hit); then mutate test.go:261 to `if cmd.parallel <= 0 {` and run go test ./gnovm/cmd/gno -run TestTest: expect green
Evidence: not run: no verifier reached it; the check is still to run

## SKIP tm2/pkg/testutils/parallel_linux.go:54 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_linux.go#L54) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: ReadMemInfo's own composition has no test: the v1 bail at :54 and the clamp applied at every ancestor at :57 can be deleted with every test green, since /proc/self/cgroup and cgroupMount are hardcoded and the only caller-level test is the live, skip-on-false TestReadMemInfo
Check: in a scratch tree delete parallel_linux.go:54-56, then separately replace the loop at :57 with cgroupDirs(...)[:1]; run `go test -count=1 -run . ./tm2/pkg/testutils` and `go test -count=1 -run TestNodeBudget ./gno.land/pkg/integration`: expect red, observe green; fix by taking the proc and mount roots as parameters of an unexported readMemInfo and testing it on a temp tree
Evidence: not run: no verifier reached it; the check is still to run

## SKIP tm2/pkg/testutils/parallel_test.go:54 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_test.go#L54) · Missing test
State: PLAUSIBLE, band: Missing test, angle: catalog, on the finder's read only
TL;DR: TestReadMemInfo skips whenever ReadMemInfo returns false, on linux and darwin too, so a reader that never reads passes CI as a skip; no CI job runs on macOS and the darwin reader has never executed
Check: in a scratch tree put `return MemInfo{}, false` at the top of ReadMemInfo in parallel_linux.go, run `go test -count=1 -run 'TestReadMemInfo' -v ./tm2/pkg/testutils`: expect a failure, observe SKIP and ok; `grep -rln -E 'macos|darwin' .github/workflows` lists release-chain-tag.yml alone
Evidence: not run: no verifier reached it; the check is still to run

## examples/Makefile:31 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/examples/Makefile#L31) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: the Makefiles and MaxParallelOverride accept different GNO_TEST_MAX_PARALLEL values, and an out-of-range one makes make test pass with no test run
Check: for v in +8 99999999999999999999; do GNO_TEST_MAX_PARALLEL=$v make -n -C examples test | grep -o 'gno test -v -p [^ ]*'; done, compared with strconv.Atoi on the same strings; then GNO_TEST_MAX_PARALLEL=99999999999999999999 make -C examples test; echo $? (expect 0 with no test run)
Details: The Makefile grep rejects '+8', which strconv.Atoi accepts, so the Go suites pin 8 while gno test runs -p 4. A 20-digit value passes the grep, so gno test gets an out-of-range -p, the flag parse fails, and the recipe ignores the pipeline's status: /tmp/gno-test-failures matches neither '^(FAIL|--- FAIL)' nor the '(type: *...Error' pattern, so make test exits 0. Only odd inputs reach it. Fix: match Atoi in the regex ('[+]?0*[1-9][0-9]{0,8}') or pass the variable through unvalidated and let gno test reject it, and fail the recipe when gno test exits non-zero.
Evidence: echo '+8' | grep -xE '0*[1-9][0-9]*' prints nothing; echo 99999999999999999999 | grep -xE ... prints it; GOROOT/src/strconv/atoi.go:251 if s[0] == '-' || s[0] == '+'; examples/Makefile:45-53: the pipeline ends in '> /tmp/gno-test-failures;' and only 'if [ -s /tmp/gno-test-failures ]' exits 1; test.go:181 fs.IntVar(&c.parallel, "p", ...). make itself not run.

## gno.land/pkg/integration/testscript_gnoland.go:163 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L163) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: on a host with four cores or fewer the node budget reads memory every second though its limit can never move
Check: read newNodeBudget and resize: with min == max neither case of the switch can fire; GOMAXPROCS=4 go test -run TestNewNodeBudgetDefaults shows reserve non-zero
Details: There min = MaxParallel() = GOMAXPROCS = max, yet reserve is set, so resize passes its reserve == 0 guard and reads memory once per nodeReadInterval while neither switch case can fire (limit > min and limit < max are both false). The cost is a read of /proc/meminfo and the cgroup files each second under b.mu, nothing a user sees. Fix: take the static branch when n >= runtime.GOMAXPROCS(0), so reserve stays 0.
Evidence: testscript_gnoland.go:154 n := testutils.MaxParallel(); :161-167 min: n, max: runtime.GOMAXPROCS(0), reserve: mi.Total / nodeMemReserveDiv; resize: if b.reserve == 0 || time.Since(b.lastCheck) < nodeReadInterval { return }; mi, ok := b.readMem(); cases need b.limit > b.min or b.limit < b.max; :92 nodeReadInterval = time.Second; parallel.go:45 min(runtime.GOMAXPROCS(0), defaultMaxParallel).

## SKIP gno.land/pkg/integration/testscript_gnoland.go:167 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gno.land/pkg/integration/testscript_gnoland.go#L167) · Nit
State: UNVERIFIED, band: Nit, angle: removed, on the finder's read only
TL;DR: The PR body still says the budget shrinks 'back towards two'; 115d118c7 removed nodeMinParallel = 2 and the floor is now MaxParallel(), min(GOMAXPROCS, 4)
Check: gh pr view 6061 -R gnolang/gno --json body --jq .body | grep -n 'towards two'; sed -n 167p gno.land/pkg/integration/testscript_gnoland.go shows `min: n,`
Evidence: not run: on the finder's read

## tm2/pkg/testutils/parallel.go:24 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel.go#L24) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: GNO_TEST_MAX_PARALLEL=+8 pins 8 workers in the Go suites and leaves gno test at 4
Check: echo '+8' | grep -xE '0*[1-9][0-9]*' prints nothing; GOROOT/src/strconv/atoi.go:251 strips a leading '+' in ParseInt, which Atoi falls back to
Details: MaxParallelOverride parses with strconv.Atoi, which takes a leading '+', while both Makefiles' grep -xE '0*[1-9][0-9]*' rejects it and falls back to 4, against examples/Makefile's comment that the override 'likewise ignores anything but a positive integer'. Same mismatch as candidate 1 seen from the Go side; one comment covers both. Fix: reject a sign in MaxParallelOverride, or accept one in the Makefiles' regex.
Evidence: parallel.go:24 n, err := strconv.Atoi(os.Getenv(MaxParallelEnv)); GOROOT/src/strconv/atoi.go:251 if s[0] == '-' || s[0] == '+'; echo '+8' | grep -xE '0*[1-9][0-9]*' printed nothing; examples/Makefile:31 and gnovm/Makefile:28 carry that regex.

## tm2/pkg/testutils/parallel_linux.go:62 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_linux.go#L62) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: a cgroup reading of zero free memory still comes back as a failed read, so the node budget skips the shrink it most needs, for one poll
Check: go test -c -o /tmp/tu.test ./tm2/pkg/testutils with the probe copied in, then `ZZ_MODE=anon systemd-run --user --scope -q -p MemoryMax=256M -p MemorySwapMax=2G /tmp/tu.test -test.run TestZZPressure -test.v`: count ok=false readings; fix is `return mi, mi.Total > 0` so ok means the figures were read
Details: clampToCgroup sets Available to 0 when used >= limit, and ReadMemInfo then returns ok false, so resize returns before lastCheck is set and cuts nothing. The next tryAcquire reads again within one poll, so the cost is a delayed cut, not a missed one. Round 1 raised the same line and it is unchanged. Fix: return mi, mi.Total > 0, so ok means the figures were read.
Evidence: parallel_linux.go:62 return mi, mi.Total > 0 && mi.Available > 0; :74-77 var free uint64; if limit > used { free = limit - used }; mi.Available = min(mi.Available, free); testscript_gnoland.go resize: mi, ok := b.readMem(); if !ok { return } before b.lastCheck = now. Finder's run: readings=2000 ok=false=13; not re-run here.
Artifact: reviews/pr/6xxx/6061-bound-test-memory-by-memory/2-115d118/tests/b2-lines-reach-catalog-zero-available.go

## tm2/pkg/testutils/parallel_test.go:60 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/tm2/pkg/testutils/parallel_test.go#L60) · Nit
State: CONFIRMED, band: Nit, angle: reach
TL;DR: TestReadMemInfo fails on a correct reading in any cgroup limited under 256 MiB
Check: go test -c -o /tmp/tu.test ./tm2/pkg/testutils; systemd-run --user --scope -q -p MemoryMax=200M /tmp/tu.test -test.run 'TestReadMemInfo$' -test.v
Details: Total is now clamped to the cgroup limit (clampToCgroup: mi.Total = min(mi.Total, limit)), but the test still reads anything at or under 256 MiB as a kB/byte mix-up. No CI runner is that small, so it only bites a constrained local or container run. Fix: check the units against /proc/meminfo's MemTotal (or assert Total is a multiple of 1024 and above a few MiB) instead of a fixed 256 MiB floor.
Evidence: parallel_test.go:60 assert.Greater(t, mi.Total, uint64(256<<20), "total looks like it is not in bytes"); parallel_linux.go:72 mi.Total = min(mi.Total, limit). Finder's run in a 200M scope: '"209715200" is not greater than "268435456"' FAIL; not re-run here.
Artifact: reviews/pr/6xxx/6061-bound-test-memory-by-memory/2-115d118/tests/b2-lines-reach-catalog-readmeminfo-small-cgroup.sh

## gnovm/pkg/gnolang/files_test.go:66 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/pkg/gnolang/files_test.go#L66) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: GNO_TEST_MAX_PARALLEL above the core count builds a TestFiles pool larger than can run, while gno test lowers the same setting to GOMAXPROCS
Check: read files_test.go:66-73 (poolSize := testutils.MaxParallel(), then that many newOpts) against test.go:261-263; run GNO_TEST_MAX_PARALLEL=8 GOMAXPROCS=4 go test ./gnovm/pkg/gnolang -run TestFiles -short with a print of distinct stores drawn from optsPool: expect 8
Details: poolSize takes the override unclamped and builds that many stores, but go test's default -test.parallel is GOMAXPROCS, so with 8 on a 4-core host only 4 subtests run while the FIFO channel cycles all 8 stores, each growing towards the package set. gno test, fed the same variable through the Makefiles, lowers -p to GOMAXPROCS at test.go:261, so the knob caps at the core count there and not here. The description's 'pins the worker count outright, the examples Makefile's included' holds for neither reading on such a host. Fix: clamp the pool at runtime.GOMAXPROCS(0) as gno test does, or say in MaxParallelEnv's doc that the pin is taken literally by the Go suites and capped by gno test.
Evidence: files_test.go:66 poolSize := testutils.MaxParallel(); :69-71 for range poolSize { optsPool <- newOpts() }; parallel.go:42-43 returns the override unclamped; gnovm/cmd/gno/test.go:261-262 if cmd.parallel <= 0 || cmd.parallel > runtime.GOMAXPROCS(0) { cmd.parallel = runtime.GOMAXPROCS(0) }; PR body line 46. Distinct-store count not run.

## gnovm/pkg/gnolang/files_test.go:186 [gh](https://github.com/gnolang/gno/blob/115d118c79c72256f69dcd5e419104f794316b8f/gnovm/pkg/gnolang/files_test.go#L186) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: TestFiles and TestStdlibs together can hold three times the static cap in stores, where the description says the total cannot double
Check: count live stores across both tests: an atomic in-flight counter incremented around newOpts in TestFiles (pool and fresh) and TestStdlibs, run go test ./gnovm/pkg/gnolang -run 'TestFiles|TestStdlibs' -short, and print the max: expect above 4, up to 12
Details: TestFiles builds poolSize stores up front (lines 66-71) and bounds its fresh stores by a second channel of the same size (line 76), whose own comment says the total stays under twice the pool. TestStdlibs, also a top-level t.Parallel test (line 165), adds a third channel of MaxParallel() (line 186), so on a host with -test.parallel of 12 or more the package can hold 4 pooled plus 4 + 4 fresh stores. The description says opting-out tests 'draw against the same budget, so the total cannot quietly double'. Fix: draw the fresh stores from the pool's own budget (or one shared semaphore across both tests), or make the description say the bound is the pool plus one fresh store per slot.
Evidence: files_test.go:37 and :165 t.Parallel(); :66 poolSize := testutils.MaxParallel(); :76 freshStore := make(chan struct{}, poolSize); :74 comment 'can't push the total past twice the pool size'; :186 freshStore := make(chan struct{}, testutils.MaxParallel()); PR body line 24 'draw against the same budget, so the total cannot quietly double'. Store count not measured.
