- [ ] Verified

**Quick Intro Context:**

---

From 14/04 to 20/04  **: Samourai crews**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- #5169 feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm — Waiting on core team decision, RPC vs WebSocket. See also #4950.
- ✅ #5198 fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change - https://github.com/gnolang/gno/pull/5198 - MikaelVallenet
- 💥 #5206 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy
- 🚫 #5216 fix(consensus): handle conflicting votes instead of panicking - https://github.com/gnolang/gno/pull/5216 - davd-gzl

---

**🛡️ PR Waiting for review (HackenProof / Security)**

- ⚠️ ✅ fix(tm2): add duplicate peer protection - https://github.com/gnolang/gno/pull/5319 - MikaelVallenet
- ⚠️ ✅ fix(gnovm): add nil checks for unsafe .V type assertions - https://github.com/gnolang/gno/pull/5196 - davd-gzl
- ⚠️ fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- ✅ fix(gnovm): Add panic on Deepfill execution on constant type - https://github.com/gnolang/gno/pull/4891 - davd-gzl
- ✅ fix(gnovm): add per-element gas metering for array/struct/string equality comparisons - https://github.com/gnolang/gno/pull/5154 - davd-gzl
- fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl
- fix: prevent path traversal in pkgdownload.Download and MemPackage.WriteTo - https://github.com/gnolang/gno/pull/5219 - davd-gzl
- fix(tm2): use separate mutex on ABCI queries client - https://github.com/gnolang/gno/pull/5431 - Villaquiranm (Related to NEWTENDG-170)
- 🚫 fix(consensus): implement RemovePeer cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl
- 📥 fix(consensus): add panic recovery to gossip goroutines - https://github.com/gnolang/gno/pull/5379 - MikaelVallenet
- 📥 fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl
- 💥 fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl
- 💥 fix: consume gas on ComputeMapKey - https://github.com/gnolang/gno/pull/5127 - Villaquiranm
- 💥 fix(gnovm): meter gas correctly for switch case - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- 💥 📥 fix(tm2/rpc): validate WebSocket origin using CORSAllowedOrigins config - https://github.com/gnolang/gno/pull/5258 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ⚠️ fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps - https://github.com/gnolang/gno/pull/5155 - davd-gzl
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- fix(gnovm): add preprocessor checks for unexported fields in struct literals - https://github.com/gnolang/gno/pull/5240 - davd-gzl
- feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - MikaelVallenet (changes requested)
- feat(gnovm): display storage usage after running file tests - https://github.com/gnolang/gno/pull/5350 - davd-gzl
- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - MikaelVallenet
- feat(gnovm): add errors.Unwrap, errors.Is, and errors.Join to stdlib - https://github.com/gnolang/gno/pull/5385 - davd-gzl
- 📥 fix(gnovm): allow []byte -> string cast on realm owned fields - https://github.com/gnolang/gno/pull/4831 - Villaquiranm
- 📥 feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5256 - MikaelVallenet
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 feat(bank): TotalCoin - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl
- ✅ 💥 feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - MikaelVallenet
- 💥 feat(gnovm): consume gas when we preprocess - https://github.com/gnolang/gno/pull/4571 - omarsy
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl (changes requested)
- 💥 feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - MikaelVallenet
- 💥 📥 feat(gnovm): add vm/qlatestversion query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- docs: add introduction to Blockchain Indexing - https://github.com/gnolang/gno/pull/4577 - davd-gzl
- docs: add new r/docs/... examples - https://github.com/gnolang/gno/pull/5016 - davd-gzl
- 💥 docs: improve clarity in interact-with-gnokey.md - https://github.com/gnolang/gno/pull/5030 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl
- ✅ feat(daokit): update daokit framework with latest version - https://github.com/gnolang/gno/pull/4884 - davd-gzl
- feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl (changes requested)
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - MikaelVallenet
- feat(govdao): upgrade UI/UX - https://github.com/gnolang/gno/pull/5051 - davd-gzl
- feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl
- ⚠️ 📥 fix(example/avl): simplify Get to return nil as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl
- 📥 feat(example): add r/sys/security dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl

---

**🔧 PR Waiting for review (Tools)**

- feat(gnokms): add insecure flag - https://github.com/gnolang/gno/pull/5360 - MikaelVallenet

---

**📂 PR Waiting for review (Other)**

- feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - MikaelVallenet
- fix(gnoland): recover validator changes after node restart - https://github.com/gnolang/gno/pull/5469 - omarsy
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 💥 feat: bech32 address from public key - https://github.com/gnolang/gno/pull/4506 - Villaquiranm
- 💥 feat(validators): limit valset changes - https://github.com/gnolang/gno/pull/4834 - omarsy

---

**🚧 PR In Progress:**

- 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- fix(gnovm): fix debug mode panics during uverse initialization - https://github.com/gnolang/gno/pull/5440 - omarsy
- 🆕 feat(gnokey): show gnoweb URL after successful addpkg deploy - https://github.com/gnolang/gno/pull/5543 - davd-gzl
- 🆕 docs: add Quick Start page - https://github.com/gnolang/gno/pull/5551 - davd-gzl
- 🆕 docs: add dedicated installation page - https://github.com/gnolang/gno/pull/5552 - davd-gzl
- 🆕 docs: add editor setup guide - https://github.com/gnolang/gno/pull/5553 - davd-gzl

---

**🐛 Issues Opened:**

- gno.land: validator set changes should be validated at tx level - https://github.com/gnolang/gno/issues/5488 - omarsy
- 🆕 gnodev: add `version` subcommand - https://github.com/gnolang/gno/issues/5550 - davd-gzl

---

**🔍 Security HackenProof - Issue to Triage @nemanja**

(none)

---

**🔒 Security HackenProof - Issue to Close @nemanja**

- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-202
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-203
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-207
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-208
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-209
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-210
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-211
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-213
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-214
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-215
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-217
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-218
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-226

---

**🎉 PR Merged**

- fix(tm2/rpc): handle malformed elements in batch requests - https://github.com/gnolang/gno/pull/5447 - davd-gzl
- feat(gnovm): implement iterative exception recovery to prevent stack overflow - https://github.com/gnolang/gno/pull/5439 - davd-gzl
- fix(tm2/client): return error message when ID is missing - https://github.com/gnolang/gno/pull/5081 - davd-gzl
- feat(gnoweb): Add Source and Action button for realm explorer - https://github.com/gnolang/gno/pull/5032 - davd-gzl
- feat(gnokey): improve CLA error display - https://github.com/gnolang/gno/pull/5325 - MikaelVallenet
- feat: improve rendering of r/sys/cla realm - https://github.com/gnolang/gno/pull/5331 - MikaelVallenet
- chore: clean usages of fail.Fail() function - https://github.com/gnolang/gno/pull/5267 - Villaquiranm
- chore(tm2): remove resolved TODO comments in state/store.go - https://github.com/gnolang/gno/pull/5290 - davd-gzl
- docs: fix missing cross and add gnomod.toml to gnokey addpkg section - https://github.com/gnolang/gno/pull/5516 - davd-gzl
- docs: fix remaining stale std package references - https://github.com/gnolang/gno/pull/5541 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

*gno-validator-tools ([samouraiworld/gno-validator-tools](https://github.com/samouraiworld/gno-validator-tools)):*
- 📥 feat/simplify_deployment_ansible_gnovalidator and update vagrantfile - https://github.com/samouraiworld/gno-validator-tools/pull/1 - louis14448

*gno-watchtower ([aeddi/gno-watchtower](https://github.com/aeddi/gno-watchtower) / [louis14448/gno-watchtower](https://github.com/louis14448/gno-watchtower)):*
- 🆕 Fix sentinel/auto reconnect docker log collector on container restart - https://github.com/aeddi/gno-watchtower/pull/2 - louis14448
- 🆕 🎉 fix: feedback + first batch of fixes - https://github.com/aeddi/gno-watchtower/pull/1 - louis14448 (merged Apr 18)

*gnomonitoring ([samouraiworld/gnomonitoring](https://github.com/samouraiworld/gnomonitoring)):*
No activity this week.

---

**📝 NOTE:**
