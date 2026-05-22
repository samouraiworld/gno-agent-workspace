5169 highlight: Waiting on core team decision, RPC vs WebSocket. See also #4950 - `feat: Blocks backup restore WebSocket`
5198 highlight: - `fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change`
5206 highlight: - `feat(gnovm): skip print/println in production discard-output mode`
5216 highlight: Don't merge - `fix(consensus): handle conflicting votes instead of panicking`
5356 highlight: Approved - `fix(gnoland): prevent duplicate validator removals in EndBlocker`

5049 high: - `fix(gnokey): inject block height when not provided in ABCI requests`
5314 high: Waiting for first review - `fix(example/avl): simplify Get to return nil as "no value"`
5325 high: - `feat(gnokey): handle CLA error client-side only`
5331 high: Approved - `feat: improve rendering of r/sys/cla realm`

4884: Approved - `feat(daokit): update daokit framework with latest version`
4891: Approved - `fix(gnovm): Add panic on Deepfill execution on constant type`
4908: Approved - `fix(avl): add missing checks in avl package`
5032: Approved - `feat(gnoweb): Add Source and Action button for realm explorer`
5048: Approved - `feat(gnovm/lint): enforce last elem of pkg path to match pkg name`
5081: Approved - `fix(tm2/client): return error message when ID is missing`
5127: Approved - `fix: consume gas on ComputeMapKey`
5155: Approved - `fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps`
5319: Approved - `fix(tm2): add duplicate peer protection`
5385: Approved - `feat(gnovm): add errors.Unwrap, errors.Is, and errors.Join to stdlib`
4834: Approved - `feat(validators): limit valset changes`

4577: Changes requested - `docs: add introduction to Blockchain Indexing`
4731: Changes requested - `feat(GovDAO): add activity page to highlight inactive GovDAO's members`
5080: Changes requested - `feat(vm): control namespace enforcement via sysnames_pkgpath VM param`

4817: In progress - `docs(example): add missing README on every nt packages`
5324: In progress - `feat: add CLA context error within gnovm & improve CLA error on gnokey`
5382: In progress - `feat: realm transaction sponsorship (PayGas + PayStorage)`
5437: In progress - `feat(gnovm): add per-type GC allocation tracking in debug builds`
5440: In progress - `fix(gnovm): fix debug mode panics during uverse initialization`

5231: Don't merge - `fix(consensus): implement RemovePeer cleanup`

5431: Related to NEWTENDG-170, Waiting for first review - `fix(tm2): use separate mutex on ABCI queries client`

4831: Waiting for first review - `fix(gnovm): allow []byte -> string cast on realm owned fields`
5230: Waiting for first review - `feat(bank): TotalCoin - track total supply of a denom`
5240: Waiting for first review - `fix(gnovm): add preprocessor checks for unexported fields in struct literals`
5256: Waiting for first review - `feat(gnovm): add gas metering for go native fn`
5258: Waiting for first review - `fix(tm2/rpc): validate WebSocket origin using CORSAllowedOrigins config`
5290: Waiting for first review - `chore(tm2): remove resolved TODO comments in state/store.go`
5313: Waiting for first review - `fix(autofile): halt writes on disk space exhaustion with auto-recovery`
5350: Waiting for first review - `feat(gnovm): display storage usage after running file tests`
5354: Waiting for first review - `feat(example): add r/sys/security dashboard realm`
5379: Waiting for first review - `fix(consensus): add panic recovery to gossip goroutines`
5380: Waiting for first review - `feat(gnovm): add vm/qlatestversion query and soft version warnings for gnokey addpkg`
5384: Waiting for first review - `fix(gnovm): recover from preprocessing panics on node restart`
5439: Waiting for first review - `feat(gnovm): implement iterative exception recovery to prevent stack overflow`
5447: Waiting for first review - `fix(tm2/rpc): handle malformed elements in batch requests`

4506: - `feat: bech32 address from public key`
4571: - `feat(gnovm): consume gas when we preprocess`
4886: - `fix(gnovm): Add missing checks`
4892: - `fix(gnovm): include missing field in shallow size calculation + add overflow protection`
4931: - `feat(examples): add subscriptions package`
4944: - `feat(govdao): add proposal fee-based for non-member`
5016: - `docs: add new r/docs/... examples`
5030: - `docs: improve clarity in interact-with-gnokey.md`
5051: - `feat(govdao): upgrade UI/UX`
5068: - `feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules`
5069: - `feat(grc20reg): implement pagination`
5154: - `fix(gnovm): add per-element gas metering for array/struct/string equality comparisons`
5196: - `fix(gnovm): add nil checks for unsafe .V type assertions`
5202: - `fix(gnovm/debugger): add bounds checks to prevent index panics`
5217: - `fix(gnovm): meter gas correctly for switch case`
5219: - `fix: prevent path traversal in pkgdownload.Download and MemPackage.WriteTo`
5267: - `chore: clean usages of fail.Fail() function`
5360: - `feat(gnokms): add insecure flag`
5361: - `feat(tm2): add transfer event for bank ops`
5366: - `feat(validators): add attributes to validator event emissions`
5370: - `feat(gno): load bank param from genesis_param.toml`

## HackenProof Triage
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-170

## HackenProof Close
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-181
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-183
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-186
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-187
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-191
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-192
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-195
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-196
https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-197
