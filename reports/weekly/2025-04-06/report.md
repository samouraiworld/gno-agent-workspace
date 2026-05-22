- [x] Verified

**Quick Intro Context: 👋**



---

🗒️️️️️️ From 30/03 to 06/04  **: Samouraï crews  🥷**

## Gno Core (/gnolang/gno)

**⚠️ Highlight**

- fix(gnoland): prevent duplicate validator removals in EndBlocker - https://github.com/gnolang/gno/pull/5356 - Omar
- feat(gnovm): skip print/println in production discard-output mode #5206 - https://github.com/gnolang/gno/pull/5206 - Omar
- fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change https://github.com/gnolang/gno/pull/5198 - Mikael
- **feat: blocks backup / restore** (RPC) & (WebSocket) - Two similar implementation of the feature for backup and restore blocks on a chain https://github.com/gnolang/gno/pull/4950 -  https://github.com/gnolang/gno/pull/5169. The only change between these two is the way we serve the information (RPC) or (WebSocket).Waiting for core team decision on which communication layer to be used. The pull request with fresher changes is [5169](https://github.com/gnolang/gno/pull/5169).
- fix(consensus): handle conflicting votes instead of panicking#5216 - https://github.com/gnolang/gno/pull/5216 - David

---

🐛 **PR Waiting for review (Hackenproof / Security)**

- fix(gnovm): inconsistency in the single-linked list implementation (cont.) - https://github.com/gnolang/gno/pull/4960 - David (Approved by core team)
- fix(tm2): use separate mutex on ABCI queries client - https://github.com/gnolang/gno/pull/5431 - Miguel. Related to https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-170
- feat(gnovm): implement iterative exception recovery to prevent stack overflow#5439 - https://github.com/gnolang/gno/pull/5439 - David
- fix(tm2): add duplicate peer protection - https://github.com/gnolang/gno/pull/5319 - Mikael
- feat(gnovm): add gas metering for go native fn#5256 - https://github.com/gnolang/gno/pull/5256 - Mikael
- fix(autofile): halt writes on disk space exhaustion with auto-recovery#5313 - https://github.com/gnolang/gno/pull/5313 - David
- fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config#5258 - https://github.com/gnolang/gno/pull/5258 - David
- fix: prevent path traversal in `pkgdownload.Download` and `MemPackage.WriteTo`#5219 - https://github.com/gnolang/gno/pull/5219 - David
- fix(gnovm): add nil checks for unsafe .V type assertions#5196 - https://github.com/gnolang/gno/pull/5196 - David
- fix(gnovm/debugger): add bounds checks to prevent index panics#5202 - https://github.com/gnolang/gno/pull/5202 - David
- fix(gnovm): meter gas correctly for switch case#5217 - https://github.com/gnolang/gno/pull/5217 - David
- fix(gnovm): add per-element gas metering for array/struct equality comparisons#5154 - https://github.com/gnolang/gno/pull/5154 - David
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - David
- fix(tm2/rpc): prevent index out of bounds panic#5136 - https://github.com/gnolang/gno/pull/5136 - David
- fix: consume gas on ComputeMapKey - https://github.com/gnolang/gno/pull/5127 - Miguel Related to security issue https://github.com/gnolang/gno/security/advisories/GHSA-m7rp-96x5-hvpx
- chore: clean usages of fail.Fail() function - https://github.com/gnolang/gno/pull/5267 - Miguel
- fix(gnovm): Add debug panic on `Deepfill` execution on constant type - https://github.com/gnolang/gno/pull/4891 - David

---

💻 **PR Waiting for review (Gnovm / TM2)**

- fix(gnovm): recover from preprocessing panics on node restart#5384 - https://github.com/gnolang/gno/pull/5384 - David
- feat(vm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg#5380 - https://github.com/gnolang/gno/pull/5380 - David
- feat(bank): `TotalCoin` - track total supply of a denom#5230 - https://github.com/gnolang/gno/pull/5230 - David
- feat(gnovm): display storage usage after running file tests#5350 - https://github.com/gnolang/gno/pull/5350 - David
- feat(gnovm): add `errors.Unwrap`, `errors.Is`, and `errors.Join` to stdlib#5385 - https://github.com/gnolang/gno/pull/5385 - David
- chore(tm2): remove resolved TODO comments in `state/store.go`#5290 - https://github.com/gnolang/gno/pull/5290 - David
- fix(tm2/client): return error message when ID is missing (e.g.: truncate) #5081 - https://github.com/gnolang/gno/pull/5081 - David
- fix(consensus): implement `RemovePeer` cleanup#5231 - https://github.com/gnolang/gno/pull/5231 - David (Don't merge)

📕 **PR Waiting for review (Documentation)**

- docs: add new `r/docs/...` examples - https://github.com/gnolang/gno/pull/5016 - David
- feat(govdao): upgrade UI/UX #5051 - https://github.com/gnolang/gno/pull/5051 - David
- docs: add introduction to Blockchain Indexing#4577 - https://github.com/gnolang/gno/pull/5051 - David
- docs: improve clarity in interact-with-gnokey.md#5030 - https://github.com/gnolang/gno/pull/5030 - David

**🔍 PR Waiting for review (Packages)**

- ⚠️ feat: improve rendering of r/sys/cla realm - https://github.com/gnolang/gno/pull/5331 - Mikael
- ⚠️ fix(example/avl): simplify `Get` to return `nil` as "no value"#5314 - https://github.com/gnolang/gno/pull/5314 - David
- feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - David
- fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - David
- feat(example): add `r/sys/security` dashboard realm#5354 - https://github.com/gnolang/gno/pull/5354 - David
- feat(GovDAO): add activity page to highlight inactive GovDAO's members #4731- https://github.com/gnolang/gno/pull/4731 - David
- feat(daokit): update daokit framework with latest version - https://github.com/gnolang/gno/pull/4884 - David
- feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - David
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - Mikael

---

**🔍 PR Waiting for review (gnoweb, gnokey, CI, CLA, etc.)**

- feat(gnoweb): Add Source and Action button for realm explorer #5032 - https://github.com/gnolang/gno/pull/5032 - David (Approved by core team)
- ⚠️ feat(gnokey): handle CLA error client-side only - https://github.com/gnolang/gno/pull/5325 - Mikael
- ⚠️ fix(gnokey): inject block height when not provided in ABCI requests #5049 - https://github.com/gnolang/gno/pull/5049 - David
- fix(ci): standardize test failure output format#5386 - https://github.com/gnolang/gno/pull/5386 - David
- feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - Mikael
- feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - Mikael

---

**🚧 PR In Progress:**

---

✅ **Security HackenProof - Issue to Triage @nemanja**

- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-170

❌ **Security HackenProof - Issue to Close @nemanja**

- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-181
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-183
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-192
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-186
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-187
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-195
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-197
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-191
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-196

---

**✅ PR Merged**

- fix(consensus): error when block header parts are too big - https://github.com/gnolang/gno/pull/5246 - Miguel
- fix(gnovm): reject `make(chan T)` at preprocess time#5238 - https://github.com/gnolang/gno/pull/5238 - David
- fix(tm2/bft): add nil checks for block and block meta retrievals#5137 - https://github.com/gnolang/gno/pull/5137 - David

---

⚙️ **Validators / Infrastructure Tools:**

- Feat/Admin Panel Feature (backend) https://github.com/samouraiworld/gnomonitoring/pull/78
- Feat/add-fallback-URLs-for-RPC-GraphQL-and-GnoWeb-endpoints https://github.com/samouraiworld/gnomonitoring/pull/83
- Feat improve daily summary https://github.com/samouraiworld/gnomonitoring/pull/84
- Feat/interactive command menu telegram https://github.com/samouraiworld/gnomonitoring/pull/86 (In progress)
- fix/replace-start_height-dedup-with-time-based-dedup-and-add-backfill-sync-gate https://github.com/samouraiworld/gnomonitoring/pull/85 (In progress)

---

**🗒️ NOTE:**

