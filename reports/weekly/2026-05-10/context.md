5206: - `feat(gnovm): skip print/println in production discard-output mode`
5216 highlight: Changes requested - `fix(consensus): handle conflicting votes instead of panicking`

5049 high: - `fix(gnokey): inject block height when not provided in ABCI requests`
5155 high: Related to NEWTENDG-59 - `fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps`
5196 high: Approved - `fix(gnovm): add nil checks for unsafe .V type assertions`
5314 high: - `fix(example/avl): simplify Get to return nil as "no value"`

4884: Approved - `feat(daokit): update daokit framework with latest version`
4891: Approved - `fix(gnovm): Add panic on Deepfill execution on constant type`
4908: Approved - `fix(avl): add missing checks in avl package`
5048: Approved - `feat(gnovm/lint): enforce last elem of pkg path to match pkg name`
5198: Approved - `fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change`
5552: Approved - `docs: add dedicated installation page`
5553: Approved - `docs: add editor setup guide`
5612: Approved - `feat(gnoweb): accept gno.land URLs in search bar`

4731: Changes requested - `feat(GovDAO): add activity page to highlight inactive GovDAO's members`
4831: Changes requested - `fix(gnovm): allow []byte -> string cast on realm owned fields`
5068: Changes requested - `feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules`
5080: Changes requested - `feat(vm): control namespace enforcement via sysnames_pkgpath VM param`
5231: Changes requested - `fix(consensus): implement RemovePeer cleanup`
5585: Changes requested - `feat(gnoweb): make heading text clickable to set URL hash`

5030: In progress - `docs: improve clarity in interact-with-gnokey.md`
5382: In progress - `feat: realm transaction sponsorship (PayGas + PayStorage)`
5437: In progress - `feat(gnovm): add per-type GC allocation tracking in debug builds`
5593: In progress - `feat(gnoweb): add :::details collapsible block`
5619: In progress - `WIP: feat(gnovm): add gas metering for go native fn`
5641: In progress - `fix(gnovm): meter gas in ProtectedSprint to prevent DoS`
5646: In progress - `fix(gnovm): meter BigInt and BigDec comparison operators`

5127: Related to GHSA-m7rp-96x5-hvpx - `fix: consume gas on ComputeMapKey`
5217 highlight: Related to NEWTENDG-81 - `fix(gnovm): meter gas correctly for switch case`
5219: Related to NEWTENDG-143 - `fix: prevent path traversal in pkgdownload.Download and MemPackage.WriteTo`
5360: Related to NEWTENDG-155 - `feat(gnokms): add insecure flag`
5370: Related to NEWTENDG-172 - `feat(gno): load bank param from genesis_param.toml`

5230: Waiting for first review - `feat(bank): TotalCoin - track total supply of a denom`
5258: Waiting for first review - `fix(tm2/rpc): validate WebSocket origin using CORSAllowedOrigins config`
5313: Waiting for first review - `fix(autofile): halt writes on disk space exhaustion with auto-recovery`
5354: Waiting for first review - `feat(example): add r/sys/security dashboard realm`
5380: Waiting for first review - `feat(gnovm): add vm/qlatestversion query and soft version warnings for gnokey addpkg`
5478: Waiting for first review - `fix(validators): handle duplicate validator entries in same block`
5592: Waiting for first review - `docs: add getting started (alternative to #5519)`
5608: Waiting for first review - `feat(gnokey): print pkgpath after maketx addpkg`
5618: Waiting for first review - `feat(gnoweb): expose render link on realm directory views`
5622: Waiting for first review - `feat(gnoweb): differenciate render and dir view with $dir`

4506: - `feat: bech32 address from public key`
4571 highlight: - `feat(gnovm): consume gas when we preprocess`
4577: - `docs: add introduction to Blockchain Indexing`
4834: - `feat(validators): limit valset changes`
4886: - `fix(gnovm): Add missing checks`
4892: - `fix(gnovm): include missing field in shallow size calculation + add overflow protection`
4931: - `feat(examples): add subscriptions package`
4944: - `feat(govdao): add proposal fee-based for non-member`
5016: - `docs: add new r/docs/... examples`
5051: - `feat(govdao): upgrade UI/UX`
5069: - `feat(grc20reg): implement pagination`
5169: - `feat: Blocks backup restore WebSocket`
5202: - `fix(gnovm/debugger): add bounds checks to prevent index panics`
5350: - `feat(gnovm): display storage usage after running file tests`
5361: - `feat(tm2): add transfer event for bank ops`
5366: - `feat(validators): add attributes to validator event emissions`
5384: - `fix(gnovm): recover from preprocessing panics on node restart`
5385: - `feat(gnovm): add errors.Unwrap, errors.Is, and errors.Join to stdlib`
5431: - `fix(tm2): use separate mutex on ABCI queries client`
5469: - `fix(gnoland): recover validator changes after node restart`
5551: - `docs: add cheat sheet page`
5563: - `feat(gnodev): add gnodev version command`
5644: - `feat(example/bptree): simplify Get to return nil as "no value"`

## HackenProof Triage
(none)

## HackenProof Close
(none)
