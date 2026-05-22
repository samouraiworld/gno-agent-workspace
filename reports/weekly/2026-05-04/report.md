- [ ] Verified

**Quick Intro Context:**

---

From 28/04 to 04/05 **: Samourai crew**

> ⚠️ High priority · 🆕 New this week · ✅ Approved by core team · 📥 Waiting for first review · 🚫 Don't merge · 💥 Merge conflict

## Gno Core (/gnolang/gno)

**⭐ Highlight**

- 🚫 fix(consensus): handle conflicting votes instead of panicking - https://github.com/gnolang/gno/pull/5216 - davd-gzl (Changes requested)

---

**🛡️ PR Waiting for review (HackenProof / Security)**

- ⚠️ ✅ fix(gnovm): add nil checks for unsafe .V type assertions - https://github.com/gnolang/gno/pull/5196 - davd-gzl
- ⚠️ ✅ fix(tm2): add duplicate peer protection - https://github.com/gnolang/gno/pull/5319 - mvallenet
- ✅ fix(gnovm): Add panic on Deepfill execution on constant type - https://github.com/gnolang/gno/pull/4891 - davd-gzl
- fix: consume gas on ComputeMapKey - https://github.com/gnolang/gno/pull/5127 - Villaquiranm
- fix(gnovm): add per-element gas metering for array/struct/string equality comparisons - https://github.com/gnolang/gno/pull/5154 - davd-gzl
- fix(gnovm/debugger): add bounds checks to prevent index panics - https://github.com/gnolang/gno/pull/5202 - davd-gzl
- fix(gnovm): meter gas correctly for switch case - https://github.com/gnolang/gno/pull/5217 - davd-gzl
- fix: prevent path traversal in `pkgdownload.Download` and `MemPackage.WriteTo` - https://github.com/gnolang/gno/pull/5219 - davd-gzl
- fix(consensus): add panic recovery to gossip goroutines - https://github.com/gnolang/gno/pull/5379 - mvallenet (Changes requested)
- fix(gnovm): recover from preprocessing panics on node restart - https://github.com/gnolang/gno/pull/5384 - davd-gzl
- fix(tm2): use separate mutex on ABCI queries client - https://github.com/gnolang/gno/pull/5431 - Villaquiranm
- 💥 feat(gnovm): skip print/println in production discard-output mode - https://github.com/gnolang/gno/pull/5206 - omarsy
- ⚠️ 💥 fix(gnokey): inject block height when not provided in ABCI requests - https://github.com/gnolang/gno/pull/5049 - davd-gzl
- 💥 fix(gnovm): include missing field in shallow size calculation + add overflow protection - https://github.com/gnolang/gno/pull/4892 - davd-gzl
- 📥 💥 fix(tm2/rpc): validate WebSocket origin using `CORSAllowedOrigins` config - https://github.com/gnolang/gno/pull/5258 - davd-gzl

---

**⚙️ PR Waiting for review (GnoVM / TM2)**

- ⚠️ fix(gnovm): add truncation protection to ProtectedString for slices, arrays, and maps - https://github.com/gnolang/gno/pull/5155 - davd-gzl
- 🆕 ✅ fix(gnovm): use proportional refund for storage deposit to prevent fund lock on storage price change - https://github.com/gnolang/gno/pull/5198 - mvallenet
- fix(gnovm): allow []byte -> string cast on realm owned fields - https://github.com/gnolang/gno/pull/4831 - Villaquiranm (Changes requested)
- fix(gnovm): Add missing checks - https://github.com/gnolang/gno/pull/4886 - davd-gzl
- feat(gnovm): consume gas when we preprocess - https://github.com/gnolang/gno/pull/4571 - omarsy
- feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules - https://github.com/gnolang/gno/pull/5068 - mvallenet (Changes requested)
- feat(gnovm): display storage usage after running file tests - https://github.com/gnolang/gno/pull/5350 - davd-gzl
- feat(gnovm): add `errors.Unwrap`, `errors.Is`, and `errors.Join` to stdlib - https://github.com/gnolang/gno/pull/5385 - davd-gzl
- 🚫 fix(consensus): implement `RemovePeer` cleanup - https://github.com/gnolang/gno/pull/5231 - davd-gzl (Changes requested)
- 📥 fix(autofile): halt writes on disk space exhaustion with auto-recovery - https://github.com/gnolang/gno/pull/5313 - davd-gzl
- 📥 feat(bank): `TotalCoin` - track total supply of a denom - https://github.com/gnolang/gno/pull/5230 - davd-gzl
- ✅ 💥 feat(gnovm/lint): enforce last elem of pkg path to match pkg name - https://github.com/gnolang/gno/pull/5048 - mvallenet
- 💥 feat(vm): control namespace enforcement via sysnames_pkgpath VM param - https://github.com/gnolang/gno/pull/5080 - davd-gzl (Changes requested)
- 💥 feat(tm2): add transfer event for bank ops - https://github.com/gnolang/gno/pull/5361 - mvallenet
- 📥 💥 feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg - https://github.com/gnolang/gno/pull/5380 - davd-gzl

---

**📖 PR Waiting for review (Documentation)**

- ✅ docs: add dedicated installation page - https://github.com/gnolang/gno/pull/5552 - davd-gzl
- ✅ docs: add editor setup guide - https://github.com/gnolang/gno/pull/5553 - davd-gzl
- docs: add introduction to Blockchain Indexing - https://github.com/gnolang/gno/pull/4577 - davd-gzl
- docs: add new `r/docs/...` examples - https://github.com/gnolang/gno/pull/5016 - davd-gzl

---

**📦 PR Waiting for review (Packages)**

- ✅ fix(avl): add missing checks in avl package - https://github.com/gnolang/gno/pull/4908 - davd-gzl
- ✅ feat(daokit): update daokit framework with latest version - https://github.com/gnolang/gno/pull/4884 - davd-gzl
- feat(GovDAO): add activity page to highlight inactive GovDAO's members - https://github.com/gnolang/gno/pull/4731 - davd-gzl (Changes requested)
- feat(examples): add subscriptions package - https://github.com/gnolang/gno/pull/4931 - mvallenet
- feat(govdao): add proposal fee-based for non-member - https://github.com/gnolang/gno/pull/4944 - davd-gzl
- feat(govdao): upgrade UI/UX - https://github.com/gnolang/gno/pull/5051 - davd-gzl
- feat(grc20reg): implement pagination - https://github.com/gnolang/gno/pull/5069 - davd-gzl
- 📥 feat(example): add `r/sys/security` dashboard realm - https://github.com/gnolang/gno/pull/5354 - davd-gzl
- ⚠️ 📥 💥 fix(example/avl): simplify `Get` to return `nil` as "no value" - https://github.com/gnolang/gno/pull/5314 - davd-gzl

---

**🌐 PR Waiting for review (Gnoweb)**

- ✅ feat(gnoweb): accept `gno.land` URLs in search bar - https://github.com/gnolang/gno/pull/5612 - davd-gzl
- feat(gnoweb): make heading text clickable to set URL hash - https://github.com/gnolang/gno/pull/5585 - davd-gzl (Changes requested)
- 🆕 📥 feat(gnoweb): expose render link on realm directory views - https://github.com/gnolang/gno/pull/5618 - AmozPay
- 🆕 📥 feat(gnoweb): differenciate render and dir view with $dir - https://github.com/gnolang/gno/pull/5622 - AmozPay

---

**🔧 PR Waiting for review (Tools)**

- feat(gnokms): add insecure flag - https://github.com/gnolang/gno/pull/5360 - mvallenet
- 📥 feat(gnokey): print pkgpath after `maketx addpkg` - https://github.com/gnolang/gno/pull/5608 - davd-gzl

---

**📂 PR Waiting for review (Other)**

- 🆕 feat: Blocks backup restore WebSocket - https://github.com/gnolang/gno/pull/5169 - Villaquiranm
- feat(gno): load bank param from genesis_param.toml - https://github.com/gnolang/gno/pull/5370 - mvallenet
- 📥 fix(validators): handle duplicate validator entries in same block - https://github.com/gnolang/gno/pull/5478 - omarsy
- 🆕 📥 feat(gnodev): add gnodev version command - https://github.com/gnolang/gno/pull/5563 - AmozPay
- 🆕 💥 fix(gnoland): recover validator changes after node restart - https://github.com/gnolang/gno/pull/5469 - omarsy
- 💥 feat: bech32 address from public key - https://github.com/gnolang/gno/pull/4506 - Villaquiranm
- 💥 feat(validators): limit valset changes - https://github.com/gnolang/gno/pull/4834 - omarsy
- 💥 feat(validators): add attributes to validator event emissions - https://github.com/gnolang/gno/pull/5366 - mvallenet

---

**🚧 PR In Progress:**

- fix(gnovm): fix debug mode panics during uverse initialization - https://github.com/gnolang/gno/pull/5440 - omarsy
- 🆕 WIP: feat(gnovm): add gas metering for go native fn - https://github.com/gnolang/gno/pull/5619 - davd-gzl
- feat(gnoweb): add `:::details` collapsible block - https://github.com/gnolang/gno/pull/5593 - davd-gzl
- docs: add getting started (alternative to #5519) - https://github.com/gnolang/gno/pull/5592 - davd-gzl
- 🚫 docs: add cheat sheet page - https://github.com/gnolang/gno/pull/5551 - davd-gzl (Changes requested)
- 💥 feat: realm transaction sponsorship (PayGas + PayStorage) - https://github.com/gnolang/gno/pull/5382 - omarsy
- 💥 feat(gnovm): add per-type GC allocation tracking in debug builds - https://github.com/gnolang/gno/pull/5437 - omarsy
- 💥 docs: improve clarity in interact-with-gnokey.md - https://github.com/gnolang/gno/pull/5030 - davd-gzl

---

**🐛 Issues Opened:**

- fix(gnoweb): Link button from Action page does not copy whole url - https://github.com/gnolang/gno/issues/5611 - AmozPay
- ci: ci-codegen-verify fails because of timeout on checkUrl - https://github.com/gnolang/gno/issues/5623 - AmozPay

---

**🔍 Security HackenProof - Issue to Triage @nemanja**

(none)

---

**🔒 Security HackenProof - Issue to Close @nemanja**

(none)

---

**🎉 PR Merged**

- fix(gnovm): add preprocessor checks for unexported fields in struct literals - https://github.com/gnolang/gno/pull/5240 - davd-gzl

---

**🖥️ Validators / Infrastructure Tools:**

(none)

---

**📝 NOTE:**
