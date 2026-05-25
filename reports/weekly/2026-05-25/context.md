5016 highlight: - `docs: add new r/docs/... examples`
5217 highlight: - `fix(gnovm): meter gas correctly for switch case`
5431 highlight: Have some differences between simulate and tx on gas measuring, investigating but team input would be appreciated - `fix(tm2): use separate mutex on ABCI queries client`
5641 highlight: In progress - `refactor(gnovm): stream Protected*String through allocWriter for per-byte gas accounting`
5644 highlight: - `feat(example/bptree): simplify Get to return nil as "no value"`
5648 highlight: In progress - `fix(gnolang): O(N^2) in Go2Gno Span for BinaryExpr chains`
5663 highlight: Approved - `test(misc/e2e): add gnovm audit and e2e regression scripts`

5049 high: - `fix(gnokey): inject block height when not provided in ABCI requests`
5155 high: - `fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps`
5314 high: - `fix(example/avl): simplify Get to return nil as "no value"`
5553 high: Approved - `docs: add editor setup guide`

4884: Approved - `feat(daokit): update daokit framework with latest version`
4891: Approved - `fix(gnovm): Add panic on Deepfill execution on constant type`
4908: Approved - `fix(avl): add missing checks in avl package`
5048: Approved - `feat(gnovm/lint): enforce last elem of pkg path to match pkg name`
5198: Approved - `fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change`

4731: Changes requested - `feat(GovDAO): add activity page to highlight inactive GovDAO's members`
5068: Changes requested - `feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules`
5080: Changes requested - `feat(vm): control namespace enforcement via sysnames_pkgpath VM param`
5216: Changes requested - `fix(consensus): handle conflicting votes instead of panicking`
5231: Changes requested - `fix(consensus): implement RemovePeer cleanup`
5585: Changes requested - `feat(gnoweb): make heading text clickable to set URL hash`

5382: In progress - `feat: realm transaction sponsorship (PayGas + PayStorage)`
5437: In progress - `feat(gnovm): add per-type GC allocation tracking in debug builds`
5593: In progress - `feat(gnoweb): add :::details collapsible block`
5619: In progress - `WIP: feat(gnovm): add gas metering for go native fn`
5678: In progress - `WIP feat(gnovm): add math/big stdlib (Int subset)`
5680: In progress - `feat(gnodev): auto-import the dev key into the local keybase`
5712: In progress - `feat(tm2/std,gnovm): drop _filetest.gno suffix requirement`

5051: Waiting for first review - `feat(govdao): upgrade UI/UX`
5230: Waiting for first review - `feat(bank): TotalCoin - track total supply of a denom`
5258: Waiting for first review - `fix(tm2/rpc): validate WebSocket origin using CORSAllowedOrigins config`
5313: Waiting for first review - `fix(autofile): halt writes on disk space exhaustion with auto-recovery`
5354: Waiting for first review - `feat(example): add r/sys/security dashboard realm`
5380: Waiting for first review - `feat(gnovm): add vm/qlatestversion query and soft version warnings for gnokey addpkg`
5478: Waiting for first review - `fix(validators): handle duplicate validator entries in same block`
5608: Waiting for first review - `feat(gnokey): print pkgpath after maketx addpkg`
5646: Waiting for first review - `fix(gnovm): meter BigInt and BigDec comparison operators`
5656: Waiting for first review - `docs(builders): consolidate and clean up builder documentation`
5676: Waiting for first review - `feat(stdlibs/bytes): port Cut, Clone, ContainsFunc, Buffer helpers`
5677: Waiting for first review - `docs: list per-function stdlib gaps in compatibility doc`
5679: Waiting for first review - `feat(stdlibs): port encoding/ascii85 and encoding/pem`

4506: - `feat: bech32 address from public key`
4571: - `feat(gnovm): consume gas when we preprocess`
4831: - `fix(gnovm): allow []byte -> string cast on realm owned fields`
4886: - `fix(gnovm): Add missing checks`
4892: - `fix(gnovm): include missing field in shallow size calculation + add overflow protection`
4931: - `feat(examples): add subscriptions package`
4944: - `feat(govdao): add proposal fee-based for non-member`
5069: - `feat(grc20reg): implement pagination`
5169: - `feat: Blocks backup restore WebSocket`
5202: - `fix(gnovm/debugger): add bounds checks to prevent index panics`
5206: - `feat(gnovm): skip print/println in production discard-output mode`
5219: - `fix: prevent path traversal in pkgdownload.Download and MemPackage.WriteTo`
5350: - `feat(gnovm): display storage usage after running file tests`
5360: - `feat(gnokms): add insecure flag`
5361: - `feat(tm2): add transfer event for bank ops`
5366: - `feat(validators): add attributes to validator event emissions`
5370: - `feat(gno): load bank param from genesis_param.toml`
5384: - `fix(gnovm): recover from preprocessing panics on node restart`
5385: - `feat(gnovm): add errors.Unwrap, errors.Is, and errors.Join to stdlib`
5469: - `fix(gnoland): recover validator changes after node restart`
5551: - `docs: add cheat sheet page`
5563: - `feat(gnodev): add gnodev version command`
5618: - `feat(gnoweb): expose render link on realm directory views`
5622: - `feat(gnoweb): differenciate render and dir view with $dir`
5672: - `fix(examples/urequire): delegate NotAborts to uassert.NotAborts`
5673: - `feat(examples/urequire): add missing uassert wrappers`
5682: - `fix(gnovm): allow fallthrough from non-last default clause`
5689: - `fix(gnolang): allow indirect cur-call through a local func variable`

## HackenProof Triage

## HackenProof Close
