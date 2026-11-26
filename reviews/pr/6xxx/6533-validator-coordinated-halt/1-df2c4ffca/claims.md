# Claims: gnolang/gno#6533 round 1, df2c4ffca, claude-opus-5-5, quick review

Round shape: solo round, one finder and one judge and writer, every Critical and Warning run, the rest read

## Findings

| Line | # | Band | State | Merge base | Draft |
| --- | --- | --- | --- | --- | --- |
| [`VALIDATOR.md:159`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) | 1 | Warning | CONFIRMED | differs | posts |
| [`VALIDATOR.md:159`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) | 5 | Nit | PLAUSIBLE | same | with row 1 |
| [`VALIDATOR.md:161`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) | 2 | Nit | CONFIRMED | differs | posts |
| [`VALIDATOR.md:161`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) | 3 | Suggestion | CONFIRMED | differs | with row 2 |
| [`VALIDATOR.md:101`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/topaz.gno.land/VALIDATOR.md#L101) | 4 | Suggestion | REFUTED |  |  |

### 1 · Warning · CONFIRMED · `misc/deployments/mainnet.gno.land/VALIDATOR.md:159`

The pre-halt check reads config.toml, so a future halt_height cleared without a restart passes it and the running node still halts at the old height

- Check: PATH=<bin with gnoland built at df2c4ffca>:$PATH GNOROOT=<tree> bash tests/solo-finder-config-cleared-no-restart.sh 6
- Observed: Judge's rerun of the finder's script in the sandbox, gnoland built from df2c4ffca: 'config get before start: 6' / 'Halt height configured {"height": 6}' / 'config get after clearing: 0' / 'halt height 6 reached, node shutting down' / 'process alive 5s after the halt' / 'exited 2s after SIGTERM, rc=0'.
- Test: [`tests/solo-finder-config-cleared-no-restart.sh`](tests/solo-finder-config-cleared-no-restart.sh)

### 5 · Nit · PLAUSIBLE · `misc/deployments/mainnet.gno.land/VALIDATOR.md:159`

The check reads ./gnoland-data/config/config.toml unless --config-path is passed, so run elsewhere it reads another file

- Check: Run gnoland config get halt_height in an empty directory and in one holding a stale gnoland-data/config/config.toml
- Observed: 

### 2 · Nit · CONFIRMED · `misc/deployments/mainnet.gno.land/VALIDATOR.md:161`

'The process does not exit' holds only for a node at the tip; a validator still block-syncing at the halt exits with code 2 and a restart-on-failure supervisor fires

- Check: Read gno.land/cmd/gnoland/UPGRADES.md:175 and grep -n 'recover()' tm2/pkg/bft/blockchain/reactor.go
- Observed: reactor.go:269 is the only recover() in the reactor; grep over execution.go, proxy and abci/client returns none; gnoland UPGRADES.md:175 states the exit code 2 case.

### 3 · Suggestion · CONFIRMED · `misc/deployments/mainnet.gno.land/VALIDATOR.md:161`

'start the new version on the same data directory' is unconditional while a coordinated upgrade may instead replay history into a fresh genesis

- Check: Read gno.land/cmd/gnoland/UPGRADES.md:13 and :129
- Observed: UPGRADES.md:13 and :129 read at df2c4ffca.

### 4 · Suggestion · REFUTED · `misc/deployments/topaz.gno.land/VALIDATOR.md:101`

Topaz and test13 carry no UPGRADES.md and no halt text, so section 6 does not fit them

- Check: ls misc/deployments/{topaz,test13}.gno.land/ and grep -n -i halt in their VALIDATOR.md
- Refuted by: ls misc/deployments/topaz.gno.land/ and test13.gno.land/: no UPGRADES.md for section 6 to link

## Cleared by the finder

No verifier ran these: the finder's own read or run settled each.

| Line | Angle | Suspected | Cleared by |
| --- | --- | --- | --- |
| [`VALIDATOR.md:159`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) | claims | `gnoland config get halt_height` may not print a bare `0` on a default config | sandbox: gnoland config init; gnoland config get halt_height |
| [`VALIDATOR.md:161`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) | claims | the halted node may not idle, or may not exit on SIGTERM, so 'wait until it has exited' could hang | sandbox run of tests/solo-finder-config-cleared-no-restart.sh 6 |
| [`VALIDATOR.md:161`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) | claims | the quoted halt log line may not match the code or be filtered by the default log level | gno.land/pkg/gnoland/app.go:1178 Info "GovDAO halt height reached, will halt after this block"; gnoland start --help |
| [`VALIDATOR.md:161`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) | claims | the Details link or its anchor may not resolve | grep -n halt gno.land/cmd/gnoland/UPGRADES.md |
| [`VALIDATOR.md:159`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) | reach | the check run from another directory could read the wrong config and print 0 | sandbox: cd / && gnoland config get halt_height |
| [`VALIDATOR.md:146`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/onyx.gno.land/VALIDATOR.md#L146) | reach | onyx section 6 drops section 1's VERSION= caveat for a from-source candidate build, which could report the final version | gno.land/cmd/gnoland/UPGRADES.md Restart gates: 'a pre-release sorts below the release it leads to', so a build reporting the final still meets an rc floor after the halt, and section 6 starts it only after the halt |
| [`VALIDATOR.md:144`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/onyx.gno.land/VALIDATOR.md#L144) | claims | PR body: the two sections differ only in the GovDAO link | diff.md lines 32 and 66 differ only in the r/gov/dao URL; lines 34-37 equal 68-71 |
| [`VALIDATOR.md:154`](https://github.com/gnolang/gno/blob/df2c4ffca/misc/deployments/mainnet.gno.land/VALIDATOR.md#L154) | catalog | invariant catalog classes against the diff | docs-only PR; the gno delta's Review the diff says skip the catalog for docs- or tooling-only PRs |

## Record

<details><summary>One row per candidate, the table the round tools parse</summary>

| # | State | Band | file:line | Check | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Warning | misc/deployments/mainnet.gno.land/VALIDATOR.md:159 | PATH=<bin with gnoland built at df2c4ffca>:$PATH GNOROOT=<tree> bash tests/solo-finder-config-cleared-no-restart.sh 6 | warm | config-check-without-restart |
| 2 | CONFIRMED | Nit | misc/deployments/mainnet.gno.land/VALIDATOR.md:161 | Read gno.land/cmd/gnoland/UPGRADES.md:175 and grep -n 'recover()' tm2/pkg/bft/blockchain/reactor.go | warm |  |
| 3 | CONFIRMED | Suggestion | misc/deployments/mainnet.gno.land/VALIDATOR.md:161 | Read gno.land/cmd/gnoland/UPGRADES.md:13 and :129 | warm |  |
| 5 | PLAUSIBLE | Nit | misc/deployments/mainnet.gno.land/VALIDATOR.md:159 | Run gnoland config get halt_height in an empty directory and in one holding a stale gnoland-data/config/config.toml | warm |  |
| 4 | REFUTED | Suggestion | misc/deployments/topaz.gno.land/VALIDATOR.md:101 | ls misc/deployments/{topaz,test13}.gno.land/ and grep -n -i halt in their VALIDATOR.md |  |  |

</details>

Hit rate per tier, from the record: hot 0/0 confirmed over 0 files, warm 3/4 confirmed over 2 files, cold 0/0 confirmed over 0 files.

## Completeness

- Thin angles: the diff is two Markdown files, so the invariant catalog does not apply (docs-only per the delta). Each claim of section 6 was read against the code: the log line `GovDAO halt height reached, will halt after this block` is `gno.land/pkg/gnoland/app.go:1178`; `CHECKSUMS.txt` is what `.github/workflows/release-chain-tag.yml:220` writes; "lists the release only after the halt" matches line 10 of both `UPGRADES.md` files; the `#what-a-halt-actually-looks-like` anchor matches the heading at `gno.land/cmd/gnoland/UPGRADES.md:127`; the onyx GovDAO link uses the same host as the file's other realm links.
- Siblings swept by shape: every `misc/deployments/*/VALIDATOR.md` with a "Register as a validator candidate" section. topaz and test13 repeat sections 1-5 without section 6 and carry no `UPGRADES.md` for it to link (row 4, refuted).
- Added by the reflector: row 5, the `--config-path` reading of the check, still to run: `gnoland config get halt_height` in an empty directory and beside a stale `gnoland-data/config/config.toml`.
- Repro run: the draft's block was extracted and run in the sandbox with `gnoland` built from df2c4ffca, its `gh pr checkout` and `go build` lines replaced by that binary since the sandbox has no network. It printed `config get after clearing: 0` and `halt height 6 reached, node shutting down`.
- Departure: the onyx sites, `VALIDATOR.md:147` and `:149`, ship as SKIP sections naming the mainnet ones, whose Fix lines name both files, rather than one posted section per site.
- Merge base: the diff changes no code, so the node behaves the same at 702fe2880; "differs" in the table means the base carries no section 6 text.

## Retro

- What failed: the draft's Body missed the requested header line, since this round was set up without the `--notes` the other three carried; the parent added it. 18 min against the 21 projected; $4.58 against $5.
- What worked: the finder's live sandbox boot settled the one Warning, and the judge reran it clean.
- Hit rate per tier: warm 2 files, 1 Warning, 2 Nits, 1 Suggestion confirmed; no hot or cold files.
- Upgrade, proposed and not applied: `review-setup.sh` could take a batch-wide `--notes` once for every target of a `review all`, so no target misses a line the user asked for on all of them.
