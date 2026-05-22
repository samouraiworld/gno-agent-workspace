- [ ] Verified

**Quick Intro Context:**

---

From 07/04 to 13/04  **: Samourai crews**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- ✅ **Iterative exception recovery** — feat(gnovm): implement iterative exception recovery to prevent stack overflow - https://github.com/gnolang/gno/pull/5439 - davd-gzl (2 ✅) (Related to NEWTENDG-182)
- **Blocks backup restore WebSocket** — feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm (1 🔄) (Waiting on core team decision, RPC vs WebSocket. See also #4950)
- **Proportional storage deposit refund** — fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change - https://github.com/gnolang/gno/pull/5198 - MikaelVallenet (1 ✅, 2 💬) (Related to NEWTENDG-128)
- 💥 **Skip print/println in production** — feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy (2 ✅, 3 💬)
- 🚫 **Handle conflicting votes** — fix(consensus): handle conflicting votes instead of panicking - https://github.com/gnolang/gno/pull/5216 - davd-gzl (4 ✅, 1 💬, 1 🔄) (Related to NEWTENDG-138)

---

**🛡️ PR Waiting for review (HackenProof / Security)**

- ⚠️ fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps - https://github.com/gnolang/gno/pull/5155 - davd-gzl (2 ✅, 2 💬) (Related to NEWTENDG-59)
- ⚠️ 💥 fix(gnovm): add nil checks for unsafe .V type assertions - https://github.com/gnolang/gno/pull/5196 - davd-gzl (1 ✅, 1 💬)
- ✅ fix(tm2): add duplicate peer protection - https://github.com/gnolang/gno/pull/5319 - MikaelVallenet (4 ✅, 1 💬) (Related to NEWTENDG-169)
- fix: consume gas on ComputeMapKey - https://github.com/gnolang/gno/pull/5127 - Villaquiranm (1 ✅, 3 💬) (Related to GHSA-m7rp-96x5-hvpx)
- ✅ fix(gnovm): add per-element gas metering for array/struct/string equality comparisons - https://github.com/gnolang/gno/pull/5154 - davd-gzl (2 ✅, 2 💬) (Related to NEWTENDG-82)
- ✅ fix(tm2): use separate mutex on ABCI queries client - https://github.com/gnolang/gno/pull/5431 - Villaquiranm (1 ✅, 2 💬) (Related to NEWTENDG-170)
- fix(gnovm): meter gas correctly for switch case - https://github.com/gnolang/gno/pull/5217 - davd-gzl (1 ✅, 1 💬) (Related to NEWTENDG-81, NEWTENDG-184)
- fix: prevent path traversal in `pkgdownload.Download` and `MemPackage.WriteTo` - https://github.com/gnolang/gno/pull/5219 - davd-gzl (1 ✅, 1 💬) (Related to NEWTENDG-143)
- fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl (1 ✅, 1 💬)
- 💥 fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl (3 ✅, 2 💬)
- 📥 feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5256 - MikaelVallenet (2 💬) (Related to NEWTENDG-129)
- 📥 fix(consensus): add panic recovery to gossip goroutines - https://github.com/gnolang/gno/pull/5379 - MikaelVallenet (1 ✅) (Related to NEWTENDG-179)

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ✅ fix(gnovm): Add panic on `Deepfill` execution on constant type - https://github.com/gnolang/gno/pull/4891 - davd-gzl (3 ✅, 1 💬)
- fix(tm2/client): return error message when ID is missing - https://github.com/gnolang/gno/pull/5081 - davd-gzl (3 ✅, 2 💬, 1 🔄)
- ✅ feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - MikaelVallenet (4 ✅, 2 💬)
- feat(gnovm): add `errors.Unwrap`, `errors.Is`, and `errors.Join` to stdlib - https://github.com/gnolang/gno/pull/5385 - davd-gzl (1 ✅)
- 💥 feat(gnovm): consume gas when we preprocess - https://github.com/gnolang/gno/pull/4571 - omarsy (2 💬)
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl (2 ✅, 1 💬)
- feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - MikaelVallenet (1 ✅, 1 💬)
- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - MikaelVallenet (1 ✅, 1 💬) (Related to NEWTENDG-172)
- feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - MikaelVallenet (2 ✅, 1 💬)
- chore: clean usages of fail.Fail() function - https://github.com/gnolang/gno/pull/5267 - Villaquiranm (1 ✅)
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl (1 🔄)
- 📥 fix(gnovm): add preprocessor checks for unexported fields in struct literals - https://github.com/gnolang/gno/pull/5240 - davd-gzl (1 ✅)
- 📥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl (2 💬)
- 📥 chore(tm2): remove resolved TODO comments in `state/store.go` - https://github.com/gnolang/gno/pull/5290 - davd-gzl (1 ✅)
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 feat(gnovm): display storage usage after running file tests - https://github.com/gnolang/gno/pull/5350 - davd-gzl (1 ✅)
- 📥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl
- 📥 fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl (1 🔄)
- 📥 fix(tm2/rpc): handle malformed elements in batch requests - https://github.com/gnolang/gno/pull/5447 - davd-gzl
- 🆕 📥 fix(gnoland): recover validator changes after node restart - https://github.com/gnolang/gno/pull/5469 - omarsy
- 🆕 📥 fix(validators): reject duplicate addresses in validator proposals - https://github.com/gnolang/gno/pull/5478 - omarsy (1 💬)
- 📥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl (2 💬)

---

**📖 PR Waiting for review (Documentation)**

- docs: add introduction to Blockchain Indexing - https://github.com/gnolang/gno/pull/4577 - davd-gzl (1 ✅, 3 💬, 1 🔄)
- docs: add new `r/docs/...` examples - https://github.com/gnolang/gno/pull/5016 - davd-gzl (2 ✅, 1 💬)
- 💥 docs: improve clarity in interact-with-gnokey.md - https://github.com/gnolang/gno/pull/5030 - davd-gzl (2 ✅)

---

**📦 PR Waiting for review (Packages)**

- ⚠️ ✅ feat: improve rendering of r/sys/cla realm - https://github.com/gnolang/gno/pull/5331 - MikaelVallenet (2 ✅, 1 💬)
- ✅ 💥 feat(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl (4 ✅, 2 💬)
- ✅ 💥 feat(daokit): update daokit framework with latest version - https://github.com/gnolang/gno/pull/4884 - davd-gzl (2 ✅)
- 💥 feat(validators): limit valset changes - https://github.com/gnolang/gno/pull/4834 - omarsy (5 💬, 1 🔄)
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - MikaelVallenet (2 ✅, 1 💬)
- 💥 feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl (3 ✅, 1 💬)
- 💥 feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl (3 ✅)
- 💥 feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl (4 ✅, 1 💬, 1 🔄)
- feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl (2 ✅, 2 💬, 3 🔄)
- feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - MikaelVallenet (1 ✅, 1 💬)
- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- ⚠️ ✅ 💥 feat(gnoweb): Add Source and Action button for realm explorer - https://github.com/gnolang/gno/pull/5032 - davd-gzl (6 ✅, 1 💬)

---

**🔧 PR Waiting for review (Tools)**

- ⚠️ fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl (2 ✅, 1 💬)
- ⚠️ feat(gnokey): handle CLA error client-side only - https://github.com/gnolang/gno/pull/5325 - MikaelVallenet (1 ✅, 1 💬)
- feat(gnokms): add insecure flag - https://github.com/gnolang/gno/pull/5360 - MikaelVallenet (2 ✅, 2 💬) (Related to NEWTENDG-155)

---

**📂 PR Waiting for review (Other)**

- ⚠️ 📥 💥 fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl (1 ✅, 1 💬)
- feat: bech32 address from public key - https://github.com/gnolang/gno/pull/4506 - Villaquiranm
- 💥 feat(govdao): upgrade UI/UX - https://github.com/gnolang/gno/pull/5051 - davd-gzl (1 ✅)
- 📥 fix(gnovm): allow []byte -> string cast on realm owned fields - https://github.com/gnolang/gno/pull/4831 - Villaquiranm (2 💬)

---

**🚧 PR In Progress:**

- 💥 docs(example): add missing README on every nt packages - https://github.com/gnolang/gno/pull/4817 - davd-gzl
- feat: add CLA context error within gnovm & improve CLA error on gnokey - https://github.com/gnolang/gno/pull/5324 - MikaelVallenet
- 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- fix(gnovm): fix debug mode panics during uverse initialization - https://github.com/gnolang/gno/pull/5440 - omarsy (1 💬)

---

**🐛 Issues Opened:**

None this week.

---

**🔍 Security HackenProof - Issue to Triage @nemanja**

- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-170

---

**🔒 Security HackenProof - Issue to Close @nemanja**

- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-181
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-183
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-186
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-187
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-191
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-192
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-195
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-196
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-197
- https://dashboard.hackenproof.com/manager/companies/newtendermint/gno-dot-land/reports/NEWTENDG-200
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

---

**🎉 PR Merged**

- fix(gnoland): prevent duplicate validator removals in EndBlocker - https://github.com/gnolang/gno/pull/5356 - omarsy
- fix(gnovm): track block item allocations in PrepareNewValues - https://github.com/gnolang/gno/pull/5436 - omarsy
- fix(gnovm): inconsistency in the single-linked list implementation (cont.) - https://github.com/gnolang/gno/pull/4960 - davd-gzl
- fix(tm2/rpc): prevent index out of bounds panic - https://github.com/gnolang/gno/pull/5136 - davd-gzl
- fix(consensus): error when block header parts are too big - https://github.com/gnolang/gno/pull/5246 - Villaquiranm

---

**🖥️ Validators / Infrastructure Tools:**

- fix/suppress-daily-reports-when-chain-is-stuck-or-disabled - https://github.com/samouraiworld/gnomonitoring/pull/82 - louis14448
- feat/add-fallback-URLs-for-RPC-GraphQL-and-GnoWeb-endpoints - https://github.com/samouraiworld/gnomonitoring/pull/83 - louis14448
- Feat improve daily summary - https://github.com/samouraiworld/gnomonitoring/pull/84 - louis14448
- fix: replace start_height dedup with time-based dedup and add backfill - https://github.com/samouraiworld/gnomonitoring/pull/85 - louis14448
- Feat/interactive command menu telegram - https://github.com/samouraiworld/gnomonitoring/pull/86 - louis14448
- fix: stop RESOLVED spam — use alert_logs as source of truth - https://github.com/samouraiworld/gnomonitoring/pull/87 - louis14448
- fix/resolve-stop-mute - https://github.com/samouraiworld/gnomonitoring/pull/88 - louis14448
- feat-improve-report-format-v2 - https://github.com/samouraiworld/gnomonitoring/pull/89 - louis14448
- fix/add-log-for-telegram-msg - https://github.com/samouraiworld/gnomonitoring/pull/90 - louis14448
- update dockerfile - https://github.com/samouraiworld/gnomonitoring/pull/91 - louis14448
- feat/fix-sync-gate-resolved-alerts - https://github.com/samouraiworld/gnomonitoring/pull/92 - louis14448
- fix/alert-pipeline - https://github.com/samouraiworld/gnomonitoring/pull/93 - louis14448

---

**📝 NOTE:**
