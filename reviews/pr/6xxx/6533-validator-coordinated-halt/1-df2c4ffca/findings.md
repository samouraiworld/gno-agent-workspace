# Findings in posting order, from round assemble: 2 to post, 0 SKIP, 1 refuted kept out

## misc/deployments/mainnet.gno.land/VALIDATOR.md:159 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/mainnet.gno.land/VALIDATOR.md#L159) · Warning
State: CONFIRMED, band: Warning, angle: claims
TL;DR: the pre-halt check reads config.toml, so a future halt_height cleared without a restart passes it and the running node still halts at the old height
Check: PATH=<bin with gnoland built at df2c4ffca>:$PATH GNOROOT=<tree> bash tests/solo-finder-config-cleared-no-restart.sh 6
Details: start.go:288-291 applies cfg.BaseConfig.HaltHeight through baseApp.SetHaltHeight once at startup, and gnoland UPGRADES.md:87-88 says `gnoland config set` only edits the file. An operator who finds a future value C below the voted height H, runs `gnoland config set halt_height 0` and re-checks sees 0, as section 6 asks, and the node still stops at C, out of step with the network, until someone restarts it; validators holding a third of the power doing so stall the chain short of H. Recovery is a restart with 0 before C. Same text at onyx VALIDATOR.md:147. The diff touches only the two docs, so the node behaves the same at the merge base; the misleading check is the branch's own text.
Evidence: Judge's rerun of the finder's script in the sandbox, gnoland built from df2c4ffca: 'config get before start: 6' / 'Halt height configured {"height": 6}' / 'config get after clearing: 0' / 'halt height 6 reached, node shutting down' / 'process alive 5s after the halt' / 'exited 2s after SIGTERM, rc=0'.
Artifact: tests/solo-finder-config-cleared-no-restart.sh
State: PLAUSIBLE, band: Nit, angle: claims
TL;DR: the check reads ./gnoland-data/config/config.toml unless --config-path is passed, so run elsewhere it reads another file
Check: Run gnoland config get halt_height in an empty directory and in one holding a stale gnoland-data/config/config.toml
Details: config.go:46 declares the config-path flag; whether a missing file errors loudly was not run this round. Still to run: the two-directory check named above.

## misc/deployments/mainnet.gno.land/VALIDATOR.md:161 [gh](https://github.com/gnolang/gno/blob/df2c4ffcaa229fbc574fdc6bb33de2a52641484f/misc/deployments/mainnet.gno.land/VALIDATOR.md#L161) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: 'The process does not exit' holds only for a node at the tip; a validator still block-syncing at the halt exits with code 2 and a restart-on-failure supervisor fires
Check: Read gno.land/cmd/gnoland/UPGRADES.md:175 and grep -n 'recover()' tm2/pkg/bft/blockchain/reactor.go
Details: BaseApp.BeginBlock panics at baseapp.go:663-664 once the height passes haltHeight. Under block sync that call runs from poolRoutine's bcR.blockExec.ApplyBlock at reactor.go:536 with no recover: the reactor's only recover is respondRoutine's at :269, and none sits in tm2/pkg/bft/state/execution.go, proxy or the abci client; the consensus recover at consensus/state.go:652 covers only a node at the tip. The supervisor restarts the old binary, which the after-halt gate refuses below halt_min_version, so the operator meets a restart loop the section says cannot happen. The remedy is the same stop and switch. Same text at onyx VALIDATOR.md:149.
Evidence: reactor.go:269 is the only recover() in the reactor; grep over execution.go, proxy and abci/client returns none; gnoland UPGRADES.md:175 states the exit code 2 case.
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: 'start the new version on the same data directory' is unconditional while a coordinated upgrade may instead replay history into a fresh genesis
Check: Read gno.land/cmd/gnoland/UPGRADES.md:13 and :129
Details: gnoland UPGRADES.md:13 defines the coordinated path as a restart on the new binary 'replaying history into a fresh genesis if the state format changed', and :129 says a genesis replay starts the new chain at halt_height + 1 from InitialHeight while the same data directory replays the last block from the block store. Latent: it bites only on a halt whose release ships a new genesis; whether any past mainnet halt did was not checked this round. Same text at onyx VALIDATOR.md:149.
Evidence: UPGRADES.md:13 and :129 read at df2c4ffca.
