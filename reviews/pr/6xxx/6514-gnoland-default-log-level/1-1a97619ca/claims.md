# Claims: gnolang/gno#6514 round 1, 1a97619ca, claude-opus-5-5, standard review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`README.md:64`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/README.md#L64) | 3 | Nit | CONFIRMED | differs | posts |
| [`start.go:155`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/start.go#L155) | 1 | Warning | REFUTED |  |  |
| [`start.go:155`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/start.go#L155) | 2 | Warning | REFUTED |  |  |
| [`start_test.go:70`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/start_test.go#L70) | 4 | Missing test | REFUTED |  |  |

### 3 · Nit · CONFIRMED · `gno.land/cmd/gnoland/README.md:64`

The callout says older binaries default to debug without naming a release or a way to check one

- Check: read README.md:62-66 for a release, version or command that shows the binary's default
- Observed: sed -n 64,66p README.md: '> releases before that change defaulted to `debug`, so pass `-log-level info`' / '> explicitly on older binaries.'

### 1 · Warning · REFUTED · `gno.land/cmd/gnoland/start.go:155`

Every other log-level default in the tree is already info

- Check: grep every "log-level" flag and every zapcore.DebugLevel assignment over the whole tree and read each default
- Refuted by: contribs/gnokms/internal/common/flags.go:58 LogLevel: zapcore.InfoLevel.String()

### 2 · Warning · REFUTED · `gno.land/cmd/gnoland/start.go:155`

Genesis replay failures log at Error and Warn, so info still shows them

- Check: read the Logger() level of every genesis-replay failure path in gno.land/pkg/gnoland/app.go
- Refuted by: app.go:1024 ctx.Logger().Error("Unable to deliver genesis tx",

### 4 · Missing test · REFUTED · `gno.land/cmd/gnoland/start_test.go:70`

The new test goes red when the default is put back to debug

- Check: go test -run 'TestStart_DefaultLogLevel$' ./gno.land/cmd/gnoland/ at head, then with start.go:155 set back to DebugLevel
- Refuted by: start_test.go:78 assert.Equal(t, "info", cfg.logLevel) fails with actual "debug" under the mutation
- Test: [`tests/judge-0-default-log-level-mutation.sh`](tests/judge-0-default-log-level-mutation.sh)

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`Makefile:35`](https://github.com/gnolang/gno/blob/1a97619ca/misc/deployments/betanet/Makefile#L35) | removed | betanet `make test` greps node output for a log line that might only print at debug, so the boot check would fail at the new info default | git grep -n 'ABCI Handshake App Info' -> tm2/pkg/bft/consensus/replay.go:251: h.logger.Info("ABCI Handshake App Info", |
| [`UPGRADES.md:150`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/UPGRADES.md#L150) | removed | UPGRADES.md and UPGRADES-TESTING.md tell operators to watch for halt log lines that might only print at debug | git grep -n -E 'GovDAO halt height reached\|Halt height configured\|CONSENSUS FAILURE' -> gno.land/pkg/gnoland/app.go:1177 app.Logger().Info(, gno.land/cmd/gnoland/start.go:291 logger.Info("Halt height configured", tm2/pkg/bft/consensus/state.go:655 cs.Logger.Error("CONSENSUS FAILURE!!!" |
| [`switch.go:477`](https://github.com/gnolang/gno/blob/1a97619ca/tm2/pkg/p2p/switch.go#L477) | reach | peer dial and handshake failures might log only at debug, so a misconfigured node (wrong chain id, unreachable peer) would go silent at the info default | sed -n 477,478p tm2/pkg/p2p/switch.go -> sw.Logger.Error( "unable to dial peer"; transport.go:203 mt.logger.Warn("accept p2p connection error"; transport.go:215 mt.logger.Error( |
| [`start_test.go:78`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/start_test.go#L78) | claims | PR body claims TestStart_DefaultLogLevel fails on master; the test might stay green with the old default | scratch worktree at 1a97619ca, $SANDBOX <tree> -- go test ./gno.land/cmd/gnoland -run 'TestStart_DefaultLogLevel$' -count=1: head prints `ok github.com/gnolang/gno/gno.land/cmd/gnoland 0.043s`; with start.go reverted to zapcore.DebugLevel.String() it prints `--- FAIL: TestStart_DefaultLogLevel` expected: "info" actual: "debug" |
| [`testscript_gnoland.go:505`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/pkg/integration/testscript_gnoland.go#L505) | claims | PR body claims integration testscripts do not reach the start default; they might parse gnoland start flags through startCfg | sed -n 505p -> fs := flag.NewFlagSet("start", flag.ContinueOnError) with only non-validator, lock-transfer, no-parallel, max-gas; git grep 'RegisterFlags\|startCfg{' in gno.land/pkg/integration returns nothing |
| [`flags.go:58`](https://github.com/gnolang/gno/blob/1a97619ca/contribs/gnokms/internal/common/flags.go#L58) | removed | sibling long-running daemons (gnokms, gnofaucet) might still default -log-level to debug, a class member the diff missed | git grep -E 'zapcore\.(Debug\|Info)Level\.String\(\)\|"log-level"' -> contribs/gnokms/internal/common/flags.go:58 LogLevel: zapcore.InfoLevel.String(), contribs/gnofaucet/serve.go:154 "info"; no other debug default in the tree |
| [`README.md:62`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/README.md#L62) | removed | another doc might still state the old debug default after the README rewrite | git grep -n -i -e log-level -e log_level -e LOG_LEVEL over the tree, plus git grep -i 'default.*debug\|debug.*default' over *.md: only gno.land/cmd/gnoland/README.md:62,65,76,209, all describing the new default |
| [`start_test.go:70`](https://github.com/gnolang/gno/blob/1a97619ca/gno.land/cmd/gnoland/start_test.go#L70) | catalog | the new t.Parallel test calls gnoenv.RootDir through RegisterFlags, which might race on package state | git grep in gnovm/pkg/gnoenv -> gnoroot.go:30 var muGnoRoot sync.Mutex guarding _GNOROOT |
| [`ci-genesis-verify.yml:59`](https://github.com/gnolang/gno/blob/1a97619ca/.github/workflows/ci-genesis-verify.yml#L59) | reach | CI genesis verify runs gnoland start without -log-level and might depend on debug output | sed -n 59,61p -> `timeout 60s gnoland start -lazy --genesis "$file" \|\| exit_code=$?` then `if [ "$exit_code" -eq 124 ]`: exit code only, no output parsed |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 3 | CONFIRMED | Nit | gno.land/cmd/gnoland/README.md:64 | read README.md:62-66 for a release, version or command that shows the binary's default | cold |  |
| 1 | REFUTED | Warning | gno.land/cmd/gnoland/start.go:155 | grep every "log-level" flag and every zapcore.DebugLevel assignment over the whole tree and read each default | warm |  |
| 2 | REFUTED | Warning | gno.land/cmd/gnoland/start.go:155 | read the Logger() level of every genesis-replay failure path in gno.land/pkg/gnoland/app.go | warm |  |
| 4 | REFUTED | Missing test | gno.land/cmd/gnoland/start_test.go:70 | go test -run 'TestStart_DefaultLogLevel$' ./gno.land/cmd/gnoland/ at head, then with start.go:155 set back to DebugLevel | cold |  |

</details>

Hit rate per tier, from the record: hot 0/0 confirmed over 0 files, warm 0/2 confirmed over 1 files, cold 1/2 confirmed over 2 files.

## Completeness

- Thin angle: tests. The one added test, `TestStart_DefaultLogLevel`, was mutated: with `start.go:155` set back to `zapcore.DebugLevel.String()` it fails (`expected: "info"`, `actual: "debug"`), PASS at head. Artifact `tests/judge-0-default-log-level-mutation.sh`, run in the sandbox.
- Sibling sweep, by shape: every `"log-level"` flag and every `zapcore.DebugLevel` assignment over the whole tree. gnokms (`flags.go:58`), gnofaucet (`serve.go:154`), gnoweb (`main.go:320`, debug only under `-verbose`) and `misc/val-scenarios/lib/scenario.sh:21` already default to info. No sibling missed.
- Reach: genesis replay failures log at Error (`app.go:1024`, `app.go:482`) or Warn (`app.go:1064`); the InitChainer Debug lines are progress markers. The finder's drops cover p2p dial failures and halt-height lines.
- Catalog: not walked. The diff touches a CLI flag default, a Go test and a README, no GnoVM, stdlib or `.gno` code, so the project's catalog rule skips it.
- Doc claims: the README callout names no release for "that change" (row kept, Nit). The PR body claims checked: the test fails on the old default (run above); deployment and telemetry files pass `info` (`misc/telemetry/docker-compose.yml:79`, `:108`); the integration testscripts write through their own `tsLogWriter` (`testscript_gnoland.go:1073`).

## Retro

- What failed: nothing died; 12 min against the 21 projected; $3.53 against $5.
- What worked: the judge's completeness pass refuted two Warning-shaped doubts, log lines operators grep for, by reading their levels in the code.
- Hit rate per tier: warm 1 file and cold 2, one README Nit confirmed in the cold tier.
- Upgrade: none beyond the batch-wide `--notes` the 6533 retro proposes.
